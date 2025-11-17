package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/shopspring/decimal"

	addressModel "bookstore-backend/internal/domains/address/model"
	address "bookstore-backend/internal/domains/address/repository"
	cart "bookstore-backend/internal/domains/cart/repository"
	invenRepo "bookstore-backend/internal/domains/inventory/repository"
	invenSer "bookstore-backend/internal/domains/inventory/service"
	"bookstore-backend/internal/domains/order/model"
	"bookstore-backend/internal/domains/order/repository"
	modelPromo "bookstore-backend/internal/domains/promotion/model"
	promo "bookstore-backend/internal/domains/promotion/repository"
	warehouse "bookstore-backend/internal/domains/warehouse/service"
	"bookstore-backend/internal/shared"
	"bookstore-backend/pkg/logger"
)

// =====================================================
// ORDER SERVICE IMPLEMENTATION
// =====================================================
type orderService struct {
	orderRepo        repository.OrderRepository
	warehouseService warehouse.Service
	inventoryRepo    invenRepo.RepositoryInterface
	addressRepo      address.RepositoryInterface
	cartRepo         cart.RepositoryInterface
	promoRepo        promo.PromotionRepository
	inventorySerivce invenSer.ServiceInterface
	asynq            *asynq.Client // DI từ container, queue riêng inventory

}

// NewOrderService creates a new order service
func NewOrderService(
	orderRepo repository.OrderRepository,
	warehouseService warehouse.Service,
	inventoryRepo invenRepo.RepositoryInterface,
	addressRepo address.RepositoryInterface,
	cartRepo cart.RepositoryInterface,
	promoRepo promo.PromotionRepository,
	inventorySerivce invenSer.ServiceInterface,
	asynq *asynq.Client,

) OrderService {
	return &orderService{
		orderRepo:        orderRepo,
		warehouseService: warehouseService,
		inventoryRepo:    inventoryRepo,
		addressRepo:      addressRepo,
		cartRepo:         cartRepo,
		promoRepo:        promoRepo,
		inventorySerivce: inventorySerivce,
		asynq:            asynq,
	}
}

// =====================================================
// CREATE ORDER - MAIN BUSINESS LOGIC
// =====================================================
func (s *orderService) CreateOrder(ctx context.Context, userID uuid.UUID, req model.CreateOrderRequest) (*model.CreateOrderResponse, error) {
	// Step 1: Validate request
	if err := req.Validate(); err != nil {
		return nil, model.NewOrderError(model.ErrCodeOrderNotFound, "Invalid request", err)
	}

	// Step 2: Get address and coordinates (for warehouse selection)
	address, err := s.addressRepo.GetDefaultByUserID(ctx, userID)
	if err != nil {
		return nil, model.NewOrderError(model.ErrCodeInvalidAddress, "Invalid or missing address", err)
	}

	// Validate address có tọa độ (nếu cần thiết cho lookup kho gần nhất)
	// Nếu address chưa có lat/lon, có thể fallback về province mapping hoặc geocode
	// if address.Latitude == 0 || address.Longitude == 0 {
	// 	return nil, model.NewOrderError(model.ErrCodeInvalidAddress, "Address must have valid coordinates", nil)
	// }

	// Step 3: Validate và lấy book items
	bookItems, err := s.validateAndFetchBookItems(ctx, req.Items)
	if err != nil {
		return nil, err
	}

	// Step 4: Calculate order subtotal
	subtotal := s.calculateItemsSubtotal(bookItems)

	// Step 5: Validate và apply promotion (nếu có)
	var promotion *modelPromo.Promotion
	var discountAmount decimal.Decimal
	if req.PromoCode != nil && *req.PromoCode != "" {
		promotion, err = s.promoRepo.FindByCodeActive(ctx, *req.PromoCode)
		if err != nil {
			return nil, model.NewOrderError(model.ErrCodePromoInvalid, "Invalid promotion code", err)
		}
		if err := s.validatePromotion(promotion, subtotal, userID); err != nil {
			return nil, err
		}
		discountAmount = s.calculateDiscount(promotion, subtotal)
	} else {
		discountAmount = decimal.Zero
	}

	// Step 6: Calculate final amounts
	isCOD := req.PaymentMethod == model.PaymentMethodCOD
	_, finalDiscount, shippingFee, codFee, taxAmount, total := model.CalculateOrderAmounts(
		subtotal,
		decimal.Zero,
		decimal.Zero,
		discountAmount,
		"",
		isCOD,
	)

	// ==================== WAREHOUSE SELECTION (SỬ DỤNG WAREHOUSE SERVICE MỚI) ====================
	// Step 7: Tìm kho gần nhất có đủ hàng cho TẤT CẢ books trong đơn
	// Logic: Với mỗi book, tìm kho gần nhất còn đủ stock, sau đó chọn kho xuất hiện nhiều nhất
	// hoặc đơn giản: chọn kho đầu tiên có thể fulfill toàn bộ đơn (tuỳ business bạn)

	// Đơn giản nhất (v1): Giả định order chỉ ship từ 1 kho duy nhất
	// → Chọn kho gần nhất mà có thể đáp ứng TẤT CẢ item (loop qua từng item check)

	var selectedWarehouseID uuid.UUID
	var selectedWarehouseName string

	// Tìm kho gần nhất cho item đầu tiên (làm ví dụ)
	// Trong thực tế, nên loop qua tất cả item và chọn kho phù hợp nhất
	firstItem := bookItems[0]
	nearestWH, err := s.warehouseService.FindNearestWarehouseWithStock(
		ctx,
		firstItem.BookID,
		address.Latitude,
		address.Longitude,
		firstItem.Quantity,
	)
	if err != nil || nearestWH == nil {
		return nil, model.NewOrderError(
			model.ErrCodeInsufficientStock,
			fmt.Sprintf("No warehouse with stock found for book: %s", firstItem.BookTitle),
			err,
		)
	}

	selectedWarehouseID = nearestWH.ID
	selectedWarehouseName = nearestWH.Name

	// Validate kho này có đủ stock cho TẤT CẢ items không (vòng lặp check)
	for _, item := range bookItems {
		hasStock, err := s.warehouseService.ValidateWarehouseHasStock(
			ctx,
			selectedWarehouseID,
			item.BookID,
			item.Quantity,
		)
		if err != nil || !hasStock {
			// Nếu kho này không đủ stock cho 1 item bất kỳ
			// Logic nâng cao: Tìm kho gần tiếp theo hoặc split shipment
			// Logic đơn giản: Trả lỗi hết hàng
			return nil, model.NewOrderError(
				model.ErrCodeInsufficientStock,
				fmt.Sprintf("Warehouse %s does not have sufficient stock for book: %s", selectedWarehouseName, item.BookTitle),
				nil,
			)
		}
	}

	// ==================== TRANSACTION BEGINS ====================
	// Step 8: Start transaction
	tx, err := s.orderRepo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer s.orderRepo.RollbackTx(ctx, tx)

	// Step 9: Reserve inventory cho tất cả items TẠI KHO ĐÃ CHỌN
	for _, item := range bookItems {
		err = s.inventoryRepo.ReserveStockWithTx(ctx, tx, selectedWarehouseID, item.BookID, item.Quantity, &userID)
		if err != nil {
			// Nếu reserve fail (race condition, hết hàng đột ngột), rollback transaction
			return nil, model.NewOrderError(
				model.ErrCodeInsufficientStock,
				fmt.Sprintf("Failed to reserve stock for book: %s", item.BookTitle),
				err,
			)
		}
	}

	// Step 10: Tạo order entity
	orderID := uuid.New()
	var promotionID *uuid.UUID
	if promotion != nil {
		promotionID = &promotion.ID
	}

	order := &model.Order{
		ID:             orderID,
		UserID:         userID,
		AddressID:      req.AddressID,
		PromotionID:    promotionID,
		WarehouseID:    &selectedWarehouseID,
		Subtotal:       subtotal,
		ShippingFee:    shippingFee,
		CODFee:         codFee,
		DiscountAmount: finalDiscount,
		TaxAmount:      taxAmount,
		Total:          total,
		PaymentMethod:  req.PaymentMethod,
		PaymentStatus:  model.PaymentStatusPending,
		CustomerNote:   req.CustomerNote,
		Version:        0,
	}

	if isCOD {
		order.Status = model.OrderStatusConfirmed
	} else {
		order.Status = model.OrderStatusPending
	}

	// Step 11: Tạo order trong DB
	if err := s.orderRepo.CreateOrderWithTx(ctx, tx, order); err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// Step 12: Tạo order items
	orderItems := s.buildOrderItems(orderID, bookItems)
	if err := s.orderRepo.CreateOrderItemsWithTx(ctx, tx, orderItems); err != nil {
		return nil, fmt.Errorf("failed to create order items: %w", err)
	}

	// Step 13: Tạo order status history
	statusHistory := &model.OrderStatusHistory{
		OrderID:    orderID,
		FromStatus: nil,
		ToStatus:   order.Status,
		ChangedBy:  &userID,
		Notes:      nil,
	}
	if err := s.orderRepo.CreateOrderStatusHistoryWithTx(ctx, tx, statusHistory); err != nil {
		return nil, fmt.Errorf("failed to create order status history: %w", err)
	}

	// Step 14: Tạo promotion usage (nếu có)
	if promotion != nil {
		usage := &modelPromo.PromotionUsage{
			PromotionID:    promotion.ID,
			UserID:         userID,
			OrderID:        orderID,
			DiscountAmount: discountAmount,
		}
		if err := s.promoRepo.CreateUsage(ctx, tx, usage); err != nil {
			return nil, fmt.Errorf("failed to create promotion usage: %w", err)
		}
	}

	// Step 15: Clear cart
	cart, err := s.cartRepo.GetByUserID(ctx, userID)
	if err == nil && cart != nil {
		if _, err := s.cartRepo.ClearCartItems(ctx, cart.ID); err != nil {
			logger.Error("Warning: failed to clear cart after order", err)
		}
	}

	// Step 16: Commit transaction
	if err := s.orderRepo.CommitTx(ctx, tx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// ==================== POST-COMMIT: ENQUEUE INVENTORY SYNC JOB ====================
	// Step 17: Enqueue InventorySyncJob cho từng book trong đơn
	for _, item := range orderItems {
		payload := shared.InventorySyncPayload{
			BookID: item.BookID.String(),
			Source: "SALE", // Source chính xác là SALE khi order thành công
		}
		b, err := json.Marshal(payload)
		if err == nil {
			task := asynq.NewTask(shared.TypeInventorySyncBookStock, b)
			if _, err := s.asynq.Enqueue(task, asynq.Queue("inventory")); err != nil {
				logger.Error("Failed to enqueue InventorySyncJob after order", err)
			}
		} else {
			logger.Error("InventorySyncJob payload marshal error after order", err)
		}
	}

	// Step 18: Trả response
	response := &model.CreateOrderResponse{
		OrderID:     orderID,
		OrderNumber: order.OrderNumber,
		Total:       total,
		Status:      order.Status,
		PaymentURL:  nil, // Sẽ được set bởi payment service
	}

	return response, nil
}

// =====================================================
// GET ORDER DETAIL
// =====================================================

func (s *orderService) GetOrderDetail(ctx context.Context, orderID uuid.UUID, userID uuid.UUID) (*model.OrderDetailResponse, error) {
	// Get order and verify ownership
	order, err := s.orderRepo.GetOrderByIDAndUserID(ctx, orderID, userID)
	if err != nil {
		return nil, err
	}

	// Get order items
	items, err := s.orderRepo.GetOrderItemsByOrderID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order items: %w", err)
	}

	// Get address details
	address, err := s.addressRepo.GetByID(ctx, order.AddressID)
	if err != nil {
		// Address not found is not critical, continue without it
		address = nil
	}

	// Build response
	response := s.buildOrderDetailResponse(order, items, *address)
	return response, nil
}

// =====================================================
// LIST ORDERS
// =====================================================

func (s *orderService) ListOrders(ctx context.Context, userID uuid.UUID, req model.ListOrdersRequest) (*model.ListOrdersResponse, error) {
	// Validate and set defaults
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Get orders from repository
	orders, total, err := s.orderRepo.ListOrdersByUserID(ctx, userID, req.Status, req.Page, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list orders: %w", err)
	}

	// Build response with pagination
	orderSummaries := make([]model.OrderSummaryResponse, 0, len(orders))
	for _, order := range orders {
		// Get items count for each order
		itemsCount, err := s.orderRepo.CountOrderItemsByOrderID(ctx, order.ID)
		if err != nil {
			itemsCount = 0 // Default to 0 if error
		}

		orderSummaries = append(orderSummaries, model.OrderSummaryResponse{
			ID:            order.ID,
			OrderNumber:   order.OrderNumber,
			Status:        order.Status,
			PaymentMethod: order.PaymentMethod,
			PaymentStatus: order.PaymentStatus,
			Total:         order.Total,
			ItemsCount:    itemsCount,
			CreatedAt:     order.CreatedAt,
		})
	}

	totalPages := int(math.Ceil(float64(total) / float64(req.Limit)))

	response := &model.ListOrdersResponse{
		Orders: orderSummaries,
		Pagination: model.PaginationMeta{
			Page:       req.Page,
			Limit:      req.Limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}

	return response, nil
}

// =====================================================
// CANCEL ORDER
// =====================================================

func (s *orderService) CancelOrder(ctx context.Context, orderID uuid.UUID, userID uuid.UUID, req model.CancelOrderRequest) error {
	// Validate request
	if err := req.Validate(); err != nil {
		return model.NewOrderError(model.ErrCodeOrderNotFound, "Invalid request", err)
	}

	// Get order and verify ownership
	order, err := s.orderRepo.GetOrderByIDAndUserID(ctx, orderID, userID)
	if err != nil {
		return err
	}

	// Check if order can be cancelled (business rule)
	if !order.CanBeCancelled() {
		return model.NewOrderError(
			model.ErrCodeOrderCannotCancel,
			fmt.Sprintf("Order with status '%s' cannot be cancelled", order.Status),
			model.ErrOrderCannotCancel,
		)
	}

	// Start transaction for atomic operations
	tx, err := s.orderRepo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer s.orderRepo.RollbackTx(ctx, tx)

	// Release reserved inventory
	items, err := s.orderRepo.GetOrderItemsByOrderID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order items: %w", err)
	}

	if order.WarehouseID != nil {
		for _, item := range items {
			err = s.inventoryRepo.ReleaseStockWithTx(ctx, tx, *order.WarehouseID, item.BookID, item.Quantity, &userID)
			if err != nil {
				// Log error but continue - inventory might already be released
				fmt.Printf("Warning: failed to release stock for book %s: %v\n", item.BookID, err)
			}
		}
	}

	// Update order status to cancelled
	result, err := tx.Exec(ctx, `
		UPDATE orders
		SET status = $1, 
			cancellation_reason = $2,
			cancelled_at = NOW(),
			version = version + 1,
			updated_at = NOW()
		WHERE id = $3 AND version = $4
	`, model.OrderStatusCancelled, req.CancellationReason, orderID, req.Version)

	if err != nil || result.RowsAffected() == 0 {
		if err.Error() == "no rows affected" {
			return model.ErrVersionMismatch
		}
		return fmt.Errorf("failed to cancel order: %w", err)
	}

	// Create status history
	statusHistory := &model.OrderStatusHistory{
		ID:         uuid.New(),
		OrderID:    orderID,
		FromStatus: &order.Status,
		ToStatus:   model.OrderStatusCancelled,
		ChangedBy:  &userID,
		Notes:      &req.CancellationReason,
	}
	if err := s.orderRepo.CreateOrderStatusHistoryWithTx(ctx, tx, statusHistory); err != nil {
		return fmt.Errorf("failed to create order status history: %w", err)
	}

	// Commit transaction
	if err := s.orderRepo.CommitTx(ctx, tx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	// ===== Enqueue InventorySyncJob cho từng book trong đơn =====
	for _, item := range items {
		payload := shared.InventorySyncPayload{
			BookID: item.BookID.String(),
			Source: "RELEASE", // hoặc "ORDER_CANCELLED"
		}
		b, _ := json.Marshal(payload)
		task := asynq.NewTask(shared.TypeInventorySyncBookStock, b)
		if _, err := s.asynq.Enqueue(task, asynq.Queue("inventory")); err != nil {
			logger.Error("Failed to enqueue InventorySyncJob after cancel order", err)
		}
	}
	// Note: For paid orders, refund will be handled manually by admin
	// as per business requirement

	return nil
}

// =====================================================
// UPDATE ORDER STATUS (ADMIN)
// =====================================================

func (s *orderService) UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, userID uuid.UUID, req model.UpdateOrderStatusRequest) error {
	// Validate request
	if err := req.Validate(); err != nil {
		return model.NewOrderError(model.ErrCodeInvalidStatus, "Invalid request", err)
	}

	// Get current order
	order, err := s.orderRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		return err
	}

	// Validate status transition
	if err := s.validateStatusTransition(order.Status, req.Status); err != nil {
		return err
	}

	// Start transaction
	tx, err := s.orderRepo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer s.orderRepo.RollbackTx(ctx, tx)

	// Update order status
	if err := s.orderRepo.UpdateOrderStatusWithTx(ctx, tx, orderID, req.Status, req.Version); err != nil {
		return err
	}

	// Update tracking number if provided (for shipping status)
	if req.TrackingNumber != nil && *req.TrackingNumber != "" {
		result, err := tx.Exec(ctx, `
			UPDATE orders
			SET tracking_number = $1, updated_at = NOW()
			WHERE id = $2
		`, *req.TrackingNumber, orderID)
		if err != nil || result.RowsAffected() == 0 {
			return fmt.Errorf("failed to update tracking number: %w", err)
		}
	}

	// Update admin note if provided
	if req.AdminNote != nil && *req.AdminNote != "" {
		r, err := tx.Exec(ctx, `
			UPDATE orders
			SET admin_note = $1, updated_at = NOW()
			WHERE id = $2
		`, *req.AdminNote, orderID)
		if err != nil || r.RowsAffected() == 0 {
			return fmt.Errorf("failed to update admin note: %w", err)
		}
	}

	// Set delivered_at timestamp if status is delivered
	if req.Status == model.OrderStatusDelivered {
		now := time.Now()
		r, err := tx.Exec(ctx, `
			UPDATE orders
			SET delivered_at = $1, updated_at = NOW()
			WHERE id = $2
		`, now, orderID)
		if err != nil || r.RowsAffected() == 0 {
			return fmt.Errorf("failed to update delivered_at: %w", err)
		}
	}

	// Create status history
	statusHistory := &model.OrderStatusHistory{
		ID:         uuid.New(),
		OrderID:    orderID,
		FromStatus: &order.Status,
		ToStatus:   req.Status,
		ChangedBy:  &userID,
		Notes:      req.AdminNote,
	}
	if err := s.orderRepo.CreateOrderStatusHistoryWithTx(ctx, tx, statusHistory); err != nil {
		return fmt.Errorf("failed to create order status history: %w", err)
	}

	// Commit transaction
	if err := s.orderRepo.CommitTx(ctx, tx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// =====================================================
// REORDER FROM EXISTING ORDER
// =====================================================

func (s *orderService) ReorderFromExisting(ctx context.Context, userID uuid.UUID, req model.ReorderRequest) (*model.CreateOrderResponse, error) {
	// Get original order
	originalOrder, err := s.orderRepo.GetOrderByIDAndUserID(ctx, req.OrderID, userID)
	if err != nil {
		return nil, err
	}

	// Get original order items
	originalItems, err := s.orderRepo.GetOrderItemsByOrderID(ctx, req.OrderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get original order items: %w", err)
	}

	// Build create order request from original order
	items := make([]model.CreateOrderItem, len(originalItems))
	for i, item := range originalItems {
		items[i] = model.CreateOrderItem{
			BookID:   item.BookID,
			Quantity: item.Quantity,
		}
	}

	createReq := model.CreateOrderRequest{
		AddressID:     req.AddressID,
		PaymentMethod: originalOrder.PaymentMethod,
		PromoCode:     nil, // Don't reuse promo code
		CustomerNote:  nil,
		Items:         items,
	}

	// Create new order
	return s.CreateOrder(ctx, userID, createReq)
}

// =====================================================
// ADMIN: LIST ALL ORDERS
// =====================================================

func (s *orderService) ListAllOrders(ctx context.Context, req model.ListOrdersRequest) (*model.ListOrdersResponse, error) {
	// Validate and set defaults
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Get orders from repository
	orders, total, err := s.orderRepo.ListAllOrders(ctx, req.Status, req.Page, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list all orders: %w", err)
	}

	// Build response
	orderSummaries := make([]model.OrderSummaryResponse, 0, len(orders))
	for _, order := range orders {
		itemsCount, err := s.orderRepo.CountOrderItemsByOrderID(ctx, order.ID)
		if err != nil {
			itemsCount = 0
		}

		orderSummaries = append(orderSummaries, model.OrderSummaryResponse{
			ID:            order.ID,
			OrderNumber:   order.OrderNumber,
			Status:        order.Status,
			PaymentMethod: order.PaymentMethod,
			PaymentStatus: order.PaymentStatus,
			Total:         order.Total,
			ItemsCount:    itemsCount,
			CreatedAt:     order.CreatedAt,
		})
	}

	totalPages := int(math.Ceil(float64(total) / float64(req.Limit)))

	response := &model.ListOrdersResponse{
		Orders: orderSummaries,
		Pagination: model.PaginationMeta{
			Page:       req.Page,
			Limit:      req.Limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}

	return response, nil
}

// =====================================================
// GET ORDER BY NUMBER
// =====================================================

func (s *orderService) GetOrderByNumber(ctx context.Context, orderNumber string, userID uuid.UUID) (*model.OrderDetailResponse, error) {
	// Get order by number
	order, err := s.orderRepo.GetOrderByNumber(ctx, orderNumber)
	if err != nil {
		return nil, err
	}

	// Verify ownership (users can only view their own orders)
	if order.UserID != userID {
		return nil, model.NewOrderError(model.ErrCodeUnauthorized, "Unauthorized access", model.ErrUnauthorized)
	}

	// Get order items
	items, err := s.orderRepo.GetOrderItemsByOrderID(ctx, order.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order items: %w", err)
	}

	// Get address details
	address, err := s.addressRepo.GetByID(ctx, order.AddressID)
	if err != nil {
		address = nil
	}

	// Build response
	response := s.buildOrderDetailResponse(order, items, *address)
	return response, nil
}

// =====================================================
// PRIVATE HELPER METHODS
// =====================================================

// validateAndFetchBookItems validates items and fetches book details
// In production, this should call Book service/repository
func (s *orderService) validateAndFetchBookItems(ctx context.Context, items []model.CreateOrderItem) ([]bookItemData, error) {
	// TODO: Implement book service call to fetch book details
	// For now, return placeholder
	// This should fetch: book_id, title, slug, cover_url, author, price

	result := make([]bookItemData, len(items))
	for i, item := range items {
		// Mock data - replace with actual book service call
		result[i] = bookItemData{
			BookID:       item.BookID,
			BookTitle:    "Book Title", // Fetch from book service
			BookSlug:     "book-slug",
			BookCoverURL: nil,
			AuthorName:   nil,
			Quantity:     item.Quantity,
			Price:        decimal.NewFromInt(100000), // Fetch from book service
		}
	}

	return result, nil
}

// bookItemData holds book details for order creation
type bookItemData struct {
	BookID       uuid.UUID
	BookTitle    string
	BookSlug     string
	BookCoverURL *string
	AuthorName   *string
	Quantity     int
	Price        decimal.Decimal
}

// calculateItemsSubtotal calculates total subtotal from all items
func (s *orderService) calculateItemsSubtotal(items []bookItemData) decimal.Decimal {
	subtotal := decimal.Zero
	for _, item := range items {
		itemSubtotal := item.Price.Mul(decimal.NewFromInt(int64(item.Quantity)))
		subtotal = subtotal.Add(itemSubtotal)
	}
	return subtotal
}

// validatePromotion validates promotion code
func (s *orderService) validatePromotion(promo *modelPromo.Promotion, subtotal decimal.Decimal, userID uuid.UUID) error {
	// Check if promotion is active
	if !promo.IsActive {
		return model.NewOrderError(model.ErrCodePromoInvalid, "Promotion is not active", model.ErrPromoInvalid)
	}
	min := decimal.NewFromInt(0)
	// Check minimum order amount
	if promo.MinOrderAmount != min {
		if subtotal.LessThan(promo.MinOrderAmount) {
			return model.NewOrderError(
				model.ErrCodePromoMinAmount,
				fmt.Sprintf("Order amount must be at least %s", promo.MinOrderAmount.String()),
				model.ErrPromoMinAmount,
			)
		}
	}

	// Check usage limit
	if promo.MaxUses != nil && promo.CurrentUses >= *promo.MaxUses {
		return model.NewOrderError(model.ErrCodePromoUsageLimitReached, "Promotion usage limit reached", model.ErrPromoUsageLimitReached)
	}

	// Check date validity
	now := time.Now()
	if now.Before(promo.StartsAt) {
		return model.NewOrderError(model.ErrCodePromoExpired, "Promotion not yet started", model.ErrPromoExpired)
	}

	if now.After(promo.ExpiresAt) {
		return model.NewOrderError(model.ErrCodePromoExpired, "Promotion has expired", model.ErrPromoExpired)
	}

	return nil
}

// calculateDiscount calculates discount amount based on promotion type
func (s *orderService) calculateDiscount(promo *modelPromo.Promotion, subtotal decimal.Decimal) decimal.Decimal {

	if promo.DiscountValue == decimal.NewFromInt32(0) {
		return decimal.Zero
	}

	if promo.DiscountType == "percentage" {
		discount := subtotal.Mul(promo.DiscountValue).Div(decimal.NewFromInt(100))

		// Apply max discount limit
		if promo.MaxDiscountAmount != nil {

			if discount.GreaterThan(*promo.MaxDiscountAmount) {
				return *promo.MaxDiscountAmount
			}
		}

		return discount
	} else if promo.DiscountType == "fixed" {
		return promo.DiscountValue
	}

	return decimal.Zero
}

// buildOrderItems builds order items from book items
func (s *orderService) buildOrderItems(orderID uuid.UUID, bookItems []bookItemData) []model.OrderItem {
	items := make([]model.OrderItem, len(bookItems))
	for i, book := range bookItems {
		subtotal := book.Price.Mul(decimal.NewFromInt(int64(book.Quantity)))
		items[i] = model.OrderItem{
			ID:           uuid.New(),
			OrderID:      orderID,
			BookID:       book.BookID,
			BookTitle:    book.BookTitle,
			BookSlug:     book.BookSlug,
			BookCoverURL: book.BookCoverURL,
			AuthorName:   book.AuthorName,
			Quantity:     book.Quantity,
			Price:        book.Price,
			Subtotal:     subtotal,
		}
	}
	return items
}

// buildOrderDetailResponse builds order detail response
func (s *orderService) buildOrderDetailResponse(
	order *model.Order,
	items []model.OrderItem,
	address addressModel.Address,
) *model.OrderDetailResponse {
	// Build items response
	itemsResponse := make([]model.OrderItemResponse, len(items))
	for i, item := range items {
		itemsResponse[i] = model.OrderItemResponse{
			ID:           item.ID,
			BookID:       item.BookID,
			BookTitle:    item.BookTitle,
			BookSlug:     item.BookSlug,
			BookCoverURL: item.BookCoverURL,
			AuthorName:   item.AuthorName,
			Quantity:     item.Quantity,
			Price:        item.Price,
			Subtotal:     item.Subtotal,
		}
	}

	// Build address response
	var addressResponse *model.OrderAddressResponse
	addressResponse = &model.OrderAddressResponse{
		ID:           address.ID,
		ReceiverName: address.RecipientName,
		Phone:        address.Phone,
		Province:     address.Province,
		District:     address.District,
		Ward:         address.Ward,
		FullAddress:  fmt.Sprintf("%s - %s - %s", address.Ward, address.District, address.Province),
	}

	return &model.OrderDetailResponse{
		ID:                  order.ID,
		OrderNumber:         order.OrderNumber,
		Status:              order.Status,
		PaymentMethod:       order.PaymentMethod,
		PaymentStatus:       order.PaymentStatus,
		Subtotal:            order.Subtotal,
		ShippingFee:         order.ShippingFee,
		CODFee:              order.CODFee,
		DiscountAmount:      order.DiscountAmount,
		TaxAmount:           order.TaxAmount,
		Total:               order.Total,
		Items:               itemsResponse,
		Address:             addressResponse,
		TrackingNumber:      order.TrackingNumber,
		EstimatedDeliveryAt: order.EstimatedDeliveryAt,
		DeliveredAt:         order.DeliveredAt,
		CustomerNote:        order.CustomerNote,
		AdminNote:           order.AdminNote,
		CancellationReason:  order.CancellationReason,
		PaidAt:              order.PaidAt,
		CreatedAt:           order.CreatedAt,
		UpdatedAt:           order.UpdatedAt,
		CancelledAt:         order.CancelledAt,
		Version:             order.Version,
	}
}

// validateStatusTransition validates if status transition is allowed
func (s *orderService) validateStatusTransition(currentStatus, newStatus string) error {
	// Define allowed transitions
	allowedTransitions := map[string][]string{
		model.OrderStatusPending:    {model.OrderStatusConfirmed, model.OrderStatusCancelled},
		model.OrderStatusConfirmed:  {model.OrderStatusProcessing, model.OrderStatusCancelled},
		model.OrderStatusProcessing: {model.OrderStatusShipping, model.OrderStatusCancelled},
		model.OrderStatusShipping:   {model.OrderStatusDelivered, model.OrderStatusReturned},
		model.OrderStatusDelivered:  {model.OrderStatusReturned},
	}

	allowed, exists := allowedTransitions[currentStatus]
	if !exists {
		return model.NewOrderError(
			model.ErrCodeInvalidStatus,
			fmt.Sprintf("Cannot transition from status '%s'", currentStatus),
			model.ErrInvalidStatus,
		)
	}

	for _, allowedStatus := range allowed {
		if allowedStatus == newStatus {
			return nil
		}
	}

	return model.NewOrderError(
		model.ErrCodeInvalidStatus,
		fmt.Sprintf("Cannot transition from '%s' to '%s'", currentStatus, newStatus),
		model.ErrInvalidStatus,
	)
}

// internal/domains/order/service/implement.go

// GetOrderByIDWithoutUser gets order without user verification
// Used by payment service to check order status
func (s *orderService) GetOrderByIDWithoutUser(
	ctx context.Context,
	orderID uuid.UUID,
) (*model.OrderDetailResponse, error) {
	// Get order without user verification
	order, err := s.orderRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	// Get order items
	items, err := s.orderRepo.GetOrderItemsByOrderID(ctx, order.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order items: %w", err)
	}

	// Get address details
	address, err := s.addressRepo.GetByID(ctx, order.AddressID)
	if err != nil {
		address = nil
	}

	// Build response
	response := s.buildOrderDetailResponse(order, items, *address)

	return response, nil
}

// CancelOrderBySystem cancels order via system action
// Skip version check, check only current status
func (s *orderService) CancelOrderBySystem(
	ctx context.Context,
	orderID uuid.UUID,
	reason string,
	source string, // "payment_timeout", "fraud_detection", "admin_action"
) error {
	// Step 1: Get order (NO user verification)
	order, err := s.orderRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("order not found: %w", err)
	}

	// Step 2: Check if order can be cancelled
	if !order.CanBeCancelled() {
		return fmt.Errorf("order cannot be cancelled: status=%s", order.Status)
	}

	// Step 3: Start transaction
	tx, err := s.orderRepo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer s.orderRepo.RollbackTx(ctx, tx)

	// Step 4: Release inventory
	items, err := s.orderRepo.GetOrderItemsByOrderID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order items: %w", err)
	}

	if order.WarehouseID != nil {
		for _, item := range items {
			// Release stock with system user (nil)
			err = s.inventoryRepo.ReleaseStockWithTx(
				ctx,
				tx,
				*order.WarehouseID,
				item.BookID,
				item.Quantity,
				nil,
			)
			if err != nil {
				fmt.Printf("Warning: failed to release stock for book %s: %v\n", item.BookID, err)
			}
		}
	}

	// Step 5: Update order status (NO version check for system actions)
	_, err = tx.Exec(ctx, `
		UPDATE orders
		SET status = 'cancelled', 
			cancellation_reason = $2,
			cancelled_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
	`, orderID, reason)

	if err != nil {
		return fmt.Errorf("failed to cancel order: %w", err)
	}

	// Step 6: Create status history (system action)
	statusHistory := &model.OrderStatusHistory{
		ID:         uuid.New(),
		OrderID:    orderID,
		FromStatus: &order.Status,
		ToStatus:   "cancelled",
		ChangedBy:  nil, // System action
		Notes:      &reason,
	}
	if err := s.orderRepo.CreateOrderStatusHistoryWithTx(ctx, tx, statusHistory); err != nil {
		return fmt.Errorf("failed to create status history: %w", err)
	}

	// Step 7: Commit transaction
	if err := s.orderRepo.CommitTx(ctx, tx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	// ===== Enqueue InventorySyncJob cho từng book trong đơn =====
	for _, item := range items {
		payload := shared.InventorySyncPayload{
			BookID: item.BookID.String(),
			Source: "RELEASE", // hoặc "ORDER_CANCELLED"
		}
		b, _ := json.Marshal(payload)
		task := asynq.NewTask(shared.TypeInventorySyncBookStock, b)
		if _, err := s.asynq.Enqueue(task, asynq.Queue("inventory")); err != nil {
			logger.Error("Failed to enqueue InventorySyncJob after cancel order", err)
		}
	}

	// TODO: Send notification to user
	fmt.Printf("System cancelled order %s: source=%s, reason=%s\n", orderID, source, reason)

	return nil
}
