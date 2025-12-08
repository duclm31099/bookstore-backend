package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	addressModel "bookstore-backend/internal/domains/address/model"
	address "bookstore-backend/internal/domains/address/repository"
	book "bookstore-backend/internal/domains/book/service"
	cartModel "bookstore-backend/internal/domains/cart/model"
	cart "bookstore-backend/internal/domains/cart/repository"
	invenRepo "bookstore-backend/internal/domains/inventory/repository"
	invenSer "bookstore-backend/internal/domains/inventory/service"
	"bookstore-backend/internal/domains/order/model"
	"bookstore-backend/internal/domains/order/repository"
	modelPromo "bookstore-backend/internal/domains/promotion/model"
	promo "bookstore-backend/internal/domains/promotion/repository"
	whModel "bookstore-backend/internal/domains/warehouse/model"
	warehouse "bookstore-backend/internal/domains/warehouse/service"
	"bookstore-backend/internal/shared"
	"bookstore-backend/internal/shared/utils"
	"bookstore-backend/pkg/logger"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/shopspring/decimal"
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
	bookService      book.ServiceInterface
}

// NewOrderService creates a new order service
func NewOrderService(
	orderRepo repository.OrderRepository,
	warehouseService warehouse.Service,
	inventoryRepo invenRepo.RepositoryInterface,
	addressRepo address.RepositoryInterface,
	cartRepo cart.RepositoryInterface,
	promoRepo promo.PromotionRepository,
	bookService book.ServiceInterface,
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
		bookService:      bookService,
	}
}

// =====================================================
// CREATE ORDER - MAIN BUSINESS LOGIC
// =====================================================
// =====================================================
// CREATE ORDER - V2 (DÙNG CHO CHECKOUT TỪ CART)
// =====================================================

func (s *orderService) CreateOrder(ctx context.Context, userID uuid.UUID, req model.CreateOrderRequest) (*model.CreateOrderResponse, error) {
	// Step 1: Validate request cơ bản (format)
	if err := req.Validate(); err != nil {
		return nil, model.NewOrderError(model.ErrCodeOrderNotFound, "Invalid request", err)
	}
	// ==================== STEP 2: LẤY CART + ITEMS TỪ DB ====================
	cart, err := s.cartRepo.GetByUserID(ctx, userID)
	if err != nil || cart == nil {
		logger.Error("GetByUserID error:", err)
		return nil, model.NewOrderError(model.ErrCodeOrderNotFound, "Cart not found for user", err)
	}

	cartItems, err := s.cartRepo.GetItemsByCartID(ctx, cart.ID)
	if err != nil || len(cartItems) == 0 {
		return nil, model.NewOrderError(model.ErrCodeOrderNotFound, "Cart is empty", err)
	}

	// ==================== STEP 3: ADDRESS HANDLING ====================
	var address *addressModel.Address
	if req.AddressID != uuid.Nil {
		// Nếu client gửi address_id thì dùng
		address, err = s.addressRepo.GetByID(ctx, req.AddressID)
		if err != nil {
			return nil, model.NewOrderError(model.ErrCodeInvalidAddress, "Invalid shipping address", err)
		}
		if address.UserID != userID {
			return nil, model.NewOrderError(model.ErrCodeInvalidAddress, "Address does not belong to user", nil)
		}
	} else {
		// Nếu không gửi thì fallback default address
		address, err = s.addressRepo.GetDefaultByUserID(ctx, userID)
		if err != nil {
			return nil, model.NewOrderError(model.ErrCodeInvalidAddress, "Missing default address", err)
		}
		req.AddressID = address.ID
	}
	var oi []model.CreateOrderItem
	for _, item := range cartItems {
		oi = append(oi, model.CreateOrderItem{
			BookID:   item.BookID,
			Quantity: item.Quantity,
		})
	}
	// ==================== STEP 4: LẤY BOOK DATA & TÍNH SUBTOTAL ====================
	bookItems, err := s.validateAndFetchBookItems(ctx, oi)
	if err != nil {
		return nil, model.NewOrderError(model.ErrCodeOrderNotFound, "Invalid cart items", err)
	}

	subtotal := cart.Subtotal

	// ==================== STEP 5: PROMO TỪ CART (KHÔNG TIN CLIENT) ====================
	var promotion *modelPromo.Promotion
	var discountAmount decimal.Decimal

	if cart.PromoCode != nil && *cart.PromoCode != "" && cart.PromoMetadata != nil {
		// Lấy promo_id từ promo_metadata trong cart
		if promoIDStr, ok := cart.PromoMetadata["promotion_id"].(string); ok {
			if promoID, err := uuid.Parse(promoIDStr); err == nil {
				promotion, err = s.promoRepo.FindByID(ctx, promoID)
				if err != nil {
					return nil, model.NewOrderError(model.ErrCodePromoInvalid, "Invalid promotion attached to cart", err)
				}
				// Validate lại với subtotal hiện tại
				if err := s.validatePromotion(promotion, subtotal, userID); err != nil {
					return nil, err
				}
				discountAmount = s.calculateDiscount(promotion, subtotal)
			}
		}
	} else {
		discountAmount = decimal.Zero
	}

	// ==================== STEP 6: TÍNH TỔNG TIỀN ====================
	isCOD := req.PaymentMethod == model.PaymentMethodCOD
	_, finalDiscount, shippingFee, codFee, taxAmount, total := model.CalculateOrderAmounts(
		subtotal,
		discountAmount,
		isCOD,
	)

	// ==================== STEP 7: CHỌN WAREHOUSE (V1: 1 KHO) ====================
	selectedWH, err := s.selectSingleWarehouseForOrder(ctx, address, bookItems)
	if err != nil {
		return nil, err
	}
	selectedWarehouseID := selectedWH.ID

	// ==================== STEP 8: TRANSACTION BẮT ĐẦU ====================
	tx, err := s.orderRepo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer s.orderRepo.RollbackTx(ctx, tx)

	// Step 9: Reserve inventory cho TẤT CẢ items tại 1 kho
	for _, item := range bookItems {
		if err := s.inventoryRepo.ReserveStockWithTx(ctx, tx, selectedWarehouseID, item.BookID, item.Quantity, &userID); err != nil {
			return nil, model.NewOrderError(
				model.ErrCodeInsufficientStock,
				fmt.Sprintf("Failed to reserve stock for book: %s", item.BookID),
				err,
			)
		}
	}

	// Step 10: Build order entity
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
	logger.Info("Go to serrvice:", map[string]interface{}{
		"order request": order,
	})
	// Step 11: Tạo order
	if err := s.orderRepo.CreateOrderWithTx(ctx, tx, order); err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// Step 12: Tạo order items
	orderItems := s.buildOrderItems(orderID, bookItems)
	logger.Info("Go to save order items :", map[string]interface{}{
		"order items": orderItems,
	})
	if err := s.orderRepo.CreateOrderItemsWithTx(ctx, tx, orderItems); err != nil {
		return nil, fmt.Errorf("failed to create order items: %w", err)
	}

	// Step 13: Status history
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

	// Step 14: Promotion usage (nếu có)
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

	// Step 15: Clear cart TRONG TX
	if err := s.cartRepo.DeleteCartWithTx(ctx, tx, cart.ID); err != nil {
		return nil, fmt.Errorf("failed to clear cart in transaction: %w", err)
	}

	// Step 16: Commit
	if err := s.orderRepo.CommitTx(ctx, tx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// ==================== STEP 17: JOBS SAU COMMIT ====================
	for _, item := range orderItems {
		payload := shared.InventorySyncPayload{
			BookID: item.BookID.String(),
			Source: "SALE",
		}
		if b, err := json.Marshal(payload); err == nil {
			task := asynq.NewTask(shared.TypeInventorySyncBookStock, b)
			if _, err := s.asynq.Enqueue(task, asynq.Queue(shared.QueueInventory)); err != nil {
				logger.Error("Failed to enqueue InventorySyncJob after order", err)
			}
		}
	}
	// (Optional) enqueue payment-timeout job ở Phase 1.3

	// Step 18: Response
	resp := &model.CreateOrderResponse{
		OrderID:     order.ID,
		OrderNumber: order.OrderNumber,
		Total:       order.Total,
		Status:      order.Status,
	}

	return resp, nil
}

// selectSingleWarehouseForOrder chọn 1 kho duy nhất có thể fulfill toàn bộ items.
// Hiện tại strategy đơn giản:
// 1. Dùng item đầu tiên để tìm kho gần nhất có đủ stock.
// 2. Validate kho đó có đủ stock cho tất cả items còn lại.
// Sau này Phase 2 có thể nâng cấp để hỗ trợ multi-warehouse splitting.
func (s *orderService) selectSingleWarehouseForOrder(
	ctx context.Context,
	address *addressModel.Address,
	bookItems []bookItemData,
) (*whModel.WarehouseWithInventory, error) {
	// Safety check
	if len(bookItems) == 0 {
		return nil, model.NewOrderError(
			model.ErrCodeOrderNotFound,
			"No items in order",
			nil,
		)
	}

	// Nếu address không có toạ độ, có thể fallback theo province / default warehouse
	if address.Latitude == 0 || address.Longitude == 0 {
		// Fallback: dùng DefaultWarehouseCode hoặc map ProvinceWarehouseMap
		// Ở Phase 1, có thể đơn giản: lấy warehouse theo province hoặc default
		wh, err := s.warehouseService.GetWarehouseByCode(ctx, model.DefaultWarehouseCode)
		if err != nil {
			return nil, model.NewOrderError(
				model.ErrCodeInsufficientStock,
				"No default warehouse available",
				err,
			)
		}

		// Ở mức tối thiểu, chỉ trả về warehouse có code default, không check stock chi tiết
		// Nếu muốn chặt hơn: lặp qua bookItems và gọi CheckAvailableStock
		for _, item := range bookItems {
			ok, err := s.inventorySerivce.CheckAvailableStock(ctx, wh.ID, item.BookID, item.Quantity)
			if err != nil || !ok {
				return nil, model.NewOrderError(
					model.ErrCodeInsufficientStock,
					fmt.Sprintf("Default warehouse does not have sufficient stock for book: %s", item.BookID),
					err,
				)
			}
		}

		// Cần wrap lại thành WarehouseWithInventory hoặc trả về struct tương đương
		return &whModel.WarehouseWithInventory{
			Warehouse: whModel.Warehouse{
				ID:   wh.ID,
				Name: wh.Name,
				Code: wh.Code,
			},
			// các field khác tuỳ struct
		}, nil
	}

	// ==================== STRATEGY: DÙNG ITEM ĐẦU TIÊN ĐỂ TÌM KHO GẦN NHẤT ====================
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
			fmt.Sprintf("No warehouse with stock found for book: %s", firstItem.BookID),
			err,
		)
	}

	// Validate: kho này phải đủ stock cho tất cả items
	for _, item := range bookItems {
		hasStock, err := s.warehouseService.ValidateWarehouseHasStock(
			ctx,
			nearestWH.ID,
			item.BookID,
			item.Quantity,
		)
		if err != nil || !hasStock {
			return nil, model.NewOrderError(
				model.ErrCodeInsufficientStock,
				fmt.Sprintf("Warehouse %s does not have sufficient stock for book: %s", nearestWH.Name, item.BookID),
				nil,
			)
		}
	}

	return nearestWH, nil
}

// =====================================================
// GET ORDER DETAIL
// =====================================================

func (s *orderService) GetOrderDetail(
	ctx context.Context,
	orderID uuid.UUID,
	userID uuid.UUID,
) (*model.OrderDetailResponse, error) {
	// 1. Get order and verify ownership
	order, err := s.orderRepo.GetOrderByIDAndUserID(ctx, orderID, userID)
	if err != nil {
		// Đã map sẵn ErrNoRows -> model.ErrOrderNotFound trong repo
		return nil, err
	}

	// 2. Get order items
	items, err := s.orderRepo.GetOrderItemsByOrderID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order items: %w", err)
	}

	// 3. Get address details (optional)
	var addr *addressModel.Address
	addr, err = s.addressRepo.GetByID(ctx, order.AddressID)
	if err != nil {
		// Nếu là not found -> coi như order không có địa chỉ (ví dụ address bị xóa)
		// Nếu cần phân biệt, có thể inspect error type ở đây.
		logger.Info("Order address not found or error, continue without address", map[string]interface{}{
			"order_id":   order.ID,
			"address_id": order.AddressID,
			"error":      err.Error(),
		})
		addr = nil
	}

	// 4. Build response (buildOrderDetailResponse chấp nhận address = nil)
	response := model.BuildOrderDetailResponse(order, items, *addr)
	return response, nil
}

// =====================================================
// LIST ORDERS
// =====================================================

func (s *orderService) ListOrders(
	ctx context.Context,
	userID uuid.UUID,
	req model.ListOrdersRequest,
) (*model.ListOrdersResponse, error) {
	// 1. Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// 2. Query orders page
	orders, total, err := s.orderRepo.ListOrdersByUserID(ctx, userID, req.Status, req.Page, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list orders: %w", err)
	}

	// Nếu không có orders
	if len(orders) == 0 {
		return &model.ListOrdersResponse{
			Orders: []model.OrderSummaryResponse{},
			Pagination: model.PaginationMeta{
				Page:       req.Page,
				Limit:      req.Limit,
				Total:      total,
				TotalPages: 0,
			},
		}, nil
	}

	// 3. Batch count items cho tất cả orders trong page
	orderIDs := make([]uuid.UUID, len(orders))
	for i, o := range orders {
		orderIDs[i] = o.ID
	}

	itemsCountMap, err := s.orderRepo.CountOrderItemsByOrderIDs(ctx, orderIDs)
	if err != nil {
		// Tùy mức critical, có thể trả lỗi luôn hoặc log và để ItemsCount = 0
		return nil, fmt.Errorf("failed to count order items for orders: %w", err)
	}

	// 4. Build response list
	orderSummaries := make([]model.OrderSummaryResponse, 0, len(orders))
	for _, order := range orders {
		itemsCount := itemsCountMap[order.ID] // default 0 nếu không có key

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

	// 5. Pagination meta
	totalPages := 0
	if req.Limit > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(req.Limit)))
	}

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

func (s *orderService) CancelOrder(
	ctx context.Context,
	orderID uuid.UUID,
	userID uuid.UUID,
	req model.CancelOrderRequest,
) error {
	// 1. Validate request
	if err := req.Validate(); err != nil {
		// Dùng code riêng cho invalid request, không phải OrderNotFound
		return model.NewOrderError(model.ErrCodeInvalidOrder, "Invalid cancel request", err)
	}

	// 2. Get order and verify ownership
	order, err := s.orderRepo.GetOrderByIDAndUserID(ctx, orderID, userID)
	if err != nil {
		return err // đã map ErrNoRows -> ErrOrderNotFound trong repo
	}

	// 3. Business rule: chỉ cho cancel trong 1 số trạng thái
	if !order.CanBeCancelled() {
		return model.NewOrderError(
			model.ErrCodeOrderCannotCancel,
			fmt.Sprintf("Order with status '%s' cannot be cancelled", order.Status),
			model.ErrOrderCannotCancel,
		)
	}

	// (Optional) Nếu muốn chặn luôn cancel khi đã paid nhưng chưa refund:
	if order.IsPaymentCompleted() && order.Status != model.OrderStatusPending {
		return model.NewOrderError(
			model.ErrCodeOrderCannotCancel,
			"Paid order cannot be cancelled directly. Please contact support.",
			model.ErrOrderCannotCancel,
		)
	}

	// 4. Begin transaction
	tx, err := s.orderRepo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer s.orderRepo.RollbackTx(ctx, tx)

	// 5. Get order items (trong cùng ctx, nhưng không nhất thiết phải qua tx vì chỉ SELECT)
	items, err := s.orderRepo.GetOrderItemsByOrderID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order items: %w", err)
	}

	// 6. Release reserved inventory (trong TX)
	if order.WarehouseID != nil {
		for _, item := range items {
			if err := s.inventoryRepo.ReleaseStockWithTx(ctx, tx, *order.WarehouseID, item.BookID, item.Quantity, &userID); err != nil {
				// Nếu lỗi là business (ví dụ BIZ02 – không đủ reserved) có thể log và tiếp tục
				// Nếu là lỗi hệ thống (DB, connection) nên rollback toàn bộ
				logger.Info("Failed to release stock when cancelling order", map[string]interface{}{
					"order_id":     order.ID,
					"warehouse_id": *order.WarehouseID,
					"book_id":      item.BookID,
					"quantity":     item.Quantity,
					"error":        err.Error(),
				})
				return fmt.Errorf("failed to release stock for book %s: %w", item.BookID.String(), err)
			}
		}
	}

	// 7. Update order status với optimistic locking
	result, err := tx.Exec(ctx, `
        UPDATE orders
        SET status = $1,
            cancellation_reason = $2,
            cancelled_at = NOW(),
            version = version + 1,
            updated_at = NOW()
        WHERE id = $3 AND version = $4
    `, model.OrderStatusCancelled, req.CancellationReason, orderID, req.Version)
	if err != nil {
		return fmt.Errorf("failed to cancel order: %w", err)
	}
	if result.RowsAffected() == 0 {
		// Version mismatch hoặc order không tồn tại nữa
		return model.ErrVersionMismatch
	}

	// 8. Create status history
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

	// 9. Commit transaction
	if err := s.orderRepo.CommitTx(ctx, tx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 10. Enqueue InventorySyncJob sau commit
	for _, item := range items {
		payload := shared.InventorySyncPayload{
			BookID: item.BookID.String(),
			Source: "ORDER_CANCELLED",
		}
		if b, err := json.Marshal(payload); err == nil {
			task := asynq.NewTask(shared.TypeInventorySyncBookStock, b)
			if _, err := s.asynq.Enqueue(task, asynq.Queue(shared.QueueInventory)); err != nil {
				logger.Error("Failed to enqueue InventorySyncJob after cancel order", err)
			}
		}
	}

	// Refund xử lý riêng (admin / payment service)

	return nil
}

// =====================================================
// UPDATE ORDER STATUS (ADMIN)
// =====================================================

func (s *orderService) UpdateOrderStatus(
	ctx context.Context,
	orderID uuid.UUID,
	userID uuid.UUID,
	req model.UpdateOrderStatusRequest,
) error {
	// 1. Validate request
	if err := req.Validate(); err != nil {
		return model.NewOrderError(model.ErrCodeInvalidStatus, "Invalid request", err)
	}

	// 2. Get current order
	order, err := s.orderRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		return err
	}

	// 3. Validate status transition
	if err := s.validateStatusTransition(order.Status, req.Status); err != nil {
		return err
	}

	// 4. Begin transaction
	tx, err := s.orderRepo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer s.orderRepo.RollbackTx(ctx, tx)

	// 5. Determine delivered_at (nếu cần)
	var deliveredAt *time.Time
	if req.Status == model.OrderStatusDelivered {
		now := time.Now()
		deliveredAt = &now
	}

	// 6. Update status + optional fields trong 1 câu lệnh với optimistic locking
	if err := s.orderRepo.UpdateOrderStatusWithTx(
		ctx,
		tx,
		orderID,
		req.Status,
		req.Version,
		req.TrackingNumber,
		req.AdminNote,
		deliveredAt,
	); err != nil {
		return err
	}

	// 7. Create status history
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

	// 8. Commit
	if err := s.orderRepo.CommitTx(ctx, tx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// (Optional) Enqueue event để gửi notification/ email cho user, v.v.

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
	if len(originalItems) == 0 {
		return nil, model.NewOrderError(
			model.ErrCodeOrderNotFound,
			"Original order has no items",
			nil,
		)
	}

	// Build create order request from original order
	items := make([]model.CreateOrderItem, len(originalItems))
	for i, item := range originalItems {
		items[i] = model.CreateOrderItem{
			BookID:   item.BookID,
			Quantity: item.Quantity,
		}
	}
	addressID := req.AddressID
	if addressID == uuid.Nil {
		// fallback: dùng address của order gốc
		addressID = originalOrder.AddressID
	}
	paymentMethod := originalOrder.PaymentMethod
	createReq := model.CreateOrderFromItemsRequest{
		AddressID:     addressID,
		PaymentMethod: paymentMethod,
		PromoCode:     nil, // Không reuse promo cũ
		CustomerNote:  nil, // có thể lấy từ req nếu mở rộng DTO
		Items:         items,
	}

	// Create new order
	return s.createOrderFromItems(ctx, userID, createReq)
}

// createOrderFromItems - core flow để tạo order từ danh sách items (Reorder, Buy Now)
// Không dùng cart, không clear cart.
//
// Không export ra interface vội (tuỳ bạn), hiện dùng nội bộ cho Reorder.
func (s *orderService) createOrderFromItems(
	ctx context.Context,
	userID uuid.UUID,
	req model.CreateOrderFromItemsRequest,
) (*model.CreateOrderResponse, error) {
	// 1. Validate request
	if err := req.Validate(); err != nil {
		return nil, model.NewOrderError(model.ErrCodeInvalidOrder, "Invalid request", err)
	}

	// 2. Address
	address, err := s.addressRepo.GetByID(ctx, req.AddressID)
	if err != nil {
		return nil, model.NewOrderError(model.ErrCodeInvalidAddress, "Invalid shipping address", err)
	}

	// 3. Lấy book data & subtotal
	bookItems, err := s.validateAndFetchBookItems(ctx, req.Items)
	if err != nil {
		return nil, err
	}
	subtotal := s.calculateItemsSubtotal(bookItems)

	var discountAmount decimal.Decimal = decimal.Zero

	// 5. Tính tổng tiền
	isCOD := req.PaymentMethod == model.PaymentMethodCOD
	_, finalDiscount, shippingFee, codFee, taxAmount, total := model.CalculateOrderAmounts(
		subtotal,
		discountAmount,
		isCOD,
	)

	// 6. Chọn warehouse (V1: single warehouse)
	selectedWH, err := s.selectSingleWarehouseForOrder(ctx, address, bookItems)
	if err != nil {
		return nil, err
	}
	selectedWarehouseID := selectedWH.ID

	// 7. Bắt đầu transaction
	tx, err := s.orderRepo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer s.orderRepo.RollbackTx(ctx, tx)

	// 8. Reserve inventory
	for _, item := range bookItems {
		if err := s.inventoryRepo.ReserveStockWithTx(ctx, tx, selectedWarehouseID, item.BookID, item.Quantity, &userID); err != nil {
			return nil, model.NewOrderError(
				model.ErrCodeInsufficientStock,
				fmt.Sprintf("Failed to reserve stock for book: %s", item.BookID),
				err,
			)
		}
	}

	// 9. Build order entity
	orderID := uuid.New()

	order := &model.Order{
		ID:             orderID,
		UserID:         userID,
		AddressID:      req.AddressID,
		PromotionID:    nil,
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
	// 10. Insert order
	if err := s.orderRepo.CreateOrderWithTx(ctx, tx, order); err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// 11. Insert order items
	orderItems := s.buildOrderItems(orderID, bookItems)

	if err := s.orderRepo.CreateOrderItemsWithTx(ctx, tx, orderItems); err != nil {
		return nil, fmt.Errorf("failed to create order items: %w", err)
	}

	// 12. Status history
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

	// 14. Commit
	if err := s.orderRepo.CommitTx(ctx, tx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 15. Jobs sau commit
	for _, item := range orderItems {
		payload := shared.InventorySyncPayload{
			BookID: item.BookID.String(),
			Source: "SALE",
		}
		if b, err := json.Marshal(payload); err == nil {
			task := asynq.NewTask(shared.TypeInventorySyncBookStock, b)
			if _, err := s.asynq.Enqueue(task, asynq.Queue(shared.QueueInventory)); err != nil {
				logger.Error("Failed to enqueue InventorySyncJob after reorder", err)
			}
		}
	}
	go s.enqueueAutoReleaseReservation(order.ID, order.OrderNumber, userID)
	// 16. Response
	resp := &model.CreateOrderResponse{
		OrderID:     order.ID,
		OrderNumber: order.OrderNumber,
		Total:       order.Total,
		Status:      order.Status,
		PaymentURL:  nil,
	}

	return resp, nil
}
func (s *orderService) enqueueAutoReleaseReservation(orderID uuid.UUID, orderNumber string, userID uuid.UUID) {
	payload := cartModel.AutoReleaseReservationPayload{
		OrderID:     orderID,
		OrderNumber: orderNumber,
		UserID:      userID,
	}

	task, err := utils.MarshalTask(shared.TypeAutoReleaseReservation, payload)
	if err != nil {
		logger.Info("Failed to marshal auto-release task", map[string]interface{}{
			"order_id": orderID,
			"error":    err.Error(),
		})
		return
	}

	_, err = s.asynq.Enqueue(task,
		asynq.Queue(shared.QueueInventory), // High priority
		asynq.MaxRetry(3),                  // Critical task
		asynq.ProcessIn(15*time.Minute),    // Execute after 15 minutes
	)

	if err != nil {
		logger.Info("Failed to enqueue auto-release task", map[string]interface{}{
			"order_id": orderID,
			"error":    err.Error(),
		})
	} else {
		logger.Info("Enqueued auto-release reservation task", map[string]interface{}{
			"order_id":   orderID,
			"execute_at": time.Now().Add(15 * time.Minute).Format(time.RFC3339),
		})
	}
}

// =====================================================
// ADMIN: LIST ALL ORDERS
// =====================================================

func (s *orderService) ListAllOrders(
	ctx context.Context,
	req model.ListOrdersRequest,
) (*model.ListOrdersResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	orders, total, err := s.orderRepo.ListAllOrders(ctx, req.Status, req.Page, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list all orders: %w", err)
	}

	if len(orders) == 0 {
		return &model.ListOrdersResponse{
			Orders: []model.OrderSummaryResponse{},
			Pagination: model.PaginationMeta{
				Page:       req.Page,
				Limit:      req.Limit,
				Total:      total,
				TotalPages: 0,
			},
		}, nil
	}

	// Batch count items
	orderIDs := make([]uuid.UUID, len(orders))
	for i, o := range orders {
		orderIDs[i] = o.ID
	}

	itemsCountMap, err := s.orderRepo.CountOrderItemsByOrderIDs(ctx, orderIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to count order items for orders: %w", err)
	}

	orderSummaries := make([]model.OrderSummaryResponse, 0, len(orders))
	for _, order := range orders {
		itemsCount := itemsCountMap[order.ID] // default 0 nếu không có key

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

	totalPages := 0
	if req.Limit > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(req.Limit)))
	}

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

func (s *orderService) GetOrderByNumber(
	ctx context.Context,
	orderNumber string,
	userID uuid.UUID,
) (*model.OrderDetailResponse, error) {
	// 1. Get order by number
	order, err := s.orderRepo.GetOrderByNumber(ctx, orderNumber)
	if err != nil {
		return nil, err
	}

	// 2. Verify ownership
	if order.UserID != userID {
		return nil, model.NewOrderError(
			model.ErrCodeUnauthorized,
			"Unauthorized access",
			model.ErrUnauthorized,
		)
	}

	// 3. Get order items
	items, err := s.orderRepo.GetOrderItemsByOrderID(ctx, order.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order items: %w", err)
	}

	// 4. Get address (optional)
	addr, err := s.addressRepo.GetByID(ctx, order.AddressID)
	if err != nil {
		addr = nil
	}

	// 5. Build response
	response := s.buildOrderDetailResponse(order, items, *addr)
	return response, nil
}

// =====================================================
// PRIVATE HELPER METHODS
// =====================================================

// validateAndFetchBookItems validates items and fetches book details
// In production, this should call Book service/repository
func (s *orderService) validateAndFetchBookItems(ctx context.Context, items []model.CreateOrderItem) ([]bookItemData, error) {
	bookIDs := make([]string, len(items))
	for i, book := range items {
		bookIDs[i] = book.BookID.String()
	}
	books, err := s.bookService.GetBooksCheckout(ctx, bookIDs)
	if err != nil {
		return nil, err
	}

	result := make([]bookItemData, len(items))
	for i, book := range books {
		// Mock data - replace with actual book service call
		result[i] = bookItemData{
			BookID:     book.ID,
			Quantity:   items[i].Quantity,
			Price:      book.Price,
			Title:      book.Title,
			AuthorName: book.AuthorName,
			CoverURL:   *book.CoverURL,
		}
	}

	return result, nil
}

// bookItemData holds book details for order creation
type bookItemData struct {
	BookID     uuid.UUID
	Quantity   int
	Price      decimal.Decimal
	Title      string
	AuthorName string
	CoverURL   string
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
		items[i] = model.OrderItem{
			ID:         uuid.New(),
			OrderID:    orderID,
			BookID:     book.BookID,
			Quantity:   book.Quantity,
			Price:      book.Price,
			AuthorName: &book.AuthorName,
			BookTitle:  book.Title,
			Subtotal:   book.Price.Mul(decimal.NewFromInt(int64(book.Quantity))),
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
		if _, err := s.asynq.Enqueue(task, asynq.Queue(shared.QueueInventory)); err != nil {
			logger.Error("Failed to enqueue InventorySyncJob after cancel order", err)
		}
	}

	// TODO: Send notification to user
	fmt.Printf("System cancelled order %s: source=%s, reason=%s\n", orderID, source, reason)

	return nil
}
