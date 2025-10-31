<img src="https://r2cdn.perplexity.ai/pplx-full-logo-primary-dark%402x.png" style="height:64px;margin-right:32px"/>

# **TIẾP TỤC SPRINT 5: CHECKOUT \& ORDER MANAGEMENT**


***

### **Ngày 5-7: Order Repository \& Service (Tiếp theo)**

**☐ Task 5.10: Create Order Repository**

File: `internal/domains/order/repository/order_repository.go`

```go
package repository

import (
    "context"
    "errors"
    "fmt"
    "time"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/google/uuid"
    "bookstore-backend/internal/domains/order/model"
)

type OrderRepository interface {
    // Order CRUD
    Create(ctx context.Context, order *model.Order) error
    CreateOrderItems(ctx context.Context, items []model.OrderItem) error
    FindByID(ctx context.Context, id uuid.UUID) (*model.Order, error)
    FindByOrderNumber(ctx context.Context, orderNumber string) (*model.Order, error)
    FindByUserID(ctx context.Context, userID uuid.UUID, page, limit int) ([]model.Order, int64, error)
    UpdateStatus(ctx context.Context, orderID uuid.UUID, newStatus string) error
    UpdatePaymentStatus(ctx context.Context, orderID uuid.UUID, status, paymentID string) error
    
    // Status history
    AddStatusHistory(ctx context.Context, history *model.OrderStatusHistory) error
    GetStatusHistory(ctx context.Context, orderID uuid.UUID) ([]model.OrderStatusHistory, error)
    
    // Inventory management
    ReserveInventory(ctx context.Context, items []model.OrderItem, warehouseID uuid.UUID) error
    ReleaseInventory(ctx context.Context, orderID uuid.UUID) error
    
    // Warehouse selection
    FindNearestWarehouse(ctx context.Context, province string) (*model.Warehouse, error)
}

type orderRepository struct {
    db *pgxpool.Pool
}

func NewOrderRepository(db *pgxpool.Pool) OrderRepository {
    return &orderRepository{db: db}
}

func (r *orderRepository) Create(ctx context.Context, order *model.Order) error {
    tx, err := r.db.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx)
    
    // Generate order number
    var orderNumber string
    err = tx.QueryRow(ctx, "SELECT generate_order_number()").Scan(&orderNumber)
    if err != nil {
        return err
    }
    
    order.OrderNumber = orderNumber
    
    // Insert order
    query := `
        INSERT INTO orders (
            order_number, user_id, address_id, warehouse_id,
            subtotal, discount_amount, shipping_fee, tax_amount, total,
            status, payment_method, payment_status,
            promotion_id, promotion_code, notes
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
        RETURNING id, created_at, updated_at
    `
    
    err = tx.QueryRow(ctx, query,
        order.OrderNumber,
        order.UserID,
        order.AddressID,
        order.WarehouseID,
        order.Subtotal,
        order.DiscountAmount,
        order.ShippingFee,
        order.TaxAmount,
        order.Total,
        order.Status,
        order.PaymentMethod,
        order.PaymentStatus,
        order.PromotionID,
        order.PromotionCode,
        order.Notes,
    ).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)
    
    if err != nil {
        return err
    }
    
    return tx.Commit(ctx)
}

func (r *orderRepository) CreateOrderItems(ctx context.Context, items []model.OrderItem) error {
    if len(items) == 0 {
        return errors.New("no items to insert")
    }
    
    tx, err := r.db.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx)
    
    query := `
        INSERT INTO order_items (
            order_id, book_id, book_title, book_author, book_isbn, book_format,
            quantity, price_at_purchase, subtotal
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    `
    
    for _, item := range items {
        _, err = tx.Exec(ctx, query,
            item.OrderID,
            item.BookID,
            item.BookTitle,
            item.BookAuthor,
            item.BookISBN,
            item.BookFormat,
            item.Quantity,
            item.PriceAtPurchase,
            item.Subtotal,
        )
        if err != nil {
            return err
        }
    }
    
    return tx.Commit(ctx)
}

func (r *orderRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Order, error) {
    query := `
        SELECT 
            o.id, o.order_number, o.user_id, o.address_id, o.warehouse_id,
            o.subtotal, o.discount_amount, o.shipping_fee, o.tax_amount, o.total,
            o.status, o.payment_method, o.payment_status, o.payment_id,
            o.promotion_id, o.promotion_code,
            o.tracking_number, o.shipping_provider, o.notes, o.cancelled_reason,
            o.cancelled_at, o.paid_at, o.shipped_at, o.delivered_at,
            o.created_at, o.updated_at
        FROM orders o
        WHERE o.id = $1
    `
    
    order := &model.Order{}
    err := r.db.QueryRow(ctx, query, id).Scan(
        &order.ID, &order.OrderNumber, &order.UserID, &order.AddressID, &order.WarehouseID,
        &order.Subtotal, &order.DiscountAmount, &order.ShippingFee, &order.TaxAmount, &order.Total,
        &order.Status, &order.PaymentMethod, &order.PaymentStatus, &order.PaymentID,
        &order.PromotionID, &order.PromotionCode,
        &order.TrackingNumber, &order.ShippingProvider, &order.Notes, &order.CancelledReason,
        &order.CancelledAt, &order.PaidAt, &order.ShippedAt, &order.DeliveredAt,
        &order.CreatedAt, &order.UpdatedAt,
    )
    
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, errors.New("order not found")
        }
        return nil, err
    }
    
    // Get order items
    itemsQuery := `
        SELECT id, order_id, book_id, book_title, book_author, book_isbn, book_format,
               quantity, price_at_purchase, subtotal, created_at
        FROM order_items
        WHERE order_id = $1
    `
    
    rows, err := r.db.Query(ctx, itemsQuery, id)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var items []model.OrderItem
    for rows.Next() {
        var item model.OrderItem
        err := rows.Scan(
            &item.ID, &item.OrderID, &item.BookID, &item.BookTitle, &item.BookAuthor,
            &item.BookISBN, &item.BookFormat, &item.Quantity, &item.PriceAtPurchase,
            &item.Subtotal, &item.CreatedAt,
        )
        if err != nil {
            return nil, err
        }
        items = append(items, item)
    }
    
    order.Items = items
    
    // Get address
    addressQuery := `
        SELECT id, user_id, recipient_name, phone, province, district, ward, street, address_type, is_default
        FROM addresses
        WHERE id = $1
    `
    
    address := &model.Address{}
    err = r.db.QueryRow(ctx, addressQuery, order.AddressID).Scan(
        &address.ID, &address.UserID, &address.RecipientName, &address.Phone,
        &address.Province, &address.District, &address.Ward, &address.Street,
        &address.AddressType, &address.IsDefault,
    )
    if err == nil {
        order.Address = address
    }
    
    return order, nil
}

func (r *orderRepository) UpdateStatus(ctx context.Context, orderID uuid.UUID, newStatus string) error {
    query := `
        UPDATE orders
        SET status = $1,
            paid_at = CASE WHEN $1 = 'confirmed' THEN NOW() ELSE paid_at END,
            shipped_at = CASE WHEN $1 = 'shipped' THEN NOW() ELSE shipped_at END,
            delivered_at = CASE WHEN $1 = 'delivered' THEN NOW() ELSE delivered_at END,
            cancelled_at = CASE WHEN $1 IN ('cancelled', 'refunded') THEN NOW() ELSE cancelled_at END
        WHERE id = $2
    `
    
    result, err := r.db.Exec(ctx, query, newStatus, orderID)
    if err != nil {
        return err
    }
    
    if result.RowsAffected() == 0 {
        return errors.New("order not found")
    }
    
    return nil
}

func (r *orderRepository) UpdatePaymentStatus(ctx context.Context, orderID uuid.UUID, status, paymentID string) error {
    query := `
        UPDATE orders
        SET payment_status = $1, payment_id = $2
        WHERE id = $3
    `
    
    _, err := r.db.Exec(ctx, query, status, paymentID, orderID)
    return err
}

func (r *orderRepository) ReserveInventory(ctx context.Context, items []model.OrderItem, warehouseID uuid.UUID) error {
    tx, err := r.db.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx)
    
    for _, item := range items {
        query := `
            UPDATE warehouse_inventory
            SET reserved = reserved + $1
            WHERE warehouse_id = $2 AND book_id = $3 AND (quantity - reserved) >= $1
        `
        
        result, err := tx.Exec(ctx, query, item.Quantity, warehouseID, item.BookID)
        if err != nil {
            return err
        }
        
        if result.RowsAffected() == 0 {
            return fmt.Errorf("insufficient stock for book %s", item.BookTitle)
        }
    }
    
    return tx.Commit(ctx)
}

func (r *orderRepository) ReleaseInventory(ctx context.Context, orderID uuid.UUID) error {
    query := `
        UPDATE warehouse_inventory wi
        SET reserved = reserved - oi.quantity
        FROM order_items oi
        JOIN orders o ON oi.order_id = o.id
        WHERE wi.warehouse_id = o.warehouse_id
          AND wi.book_id = oi.book_id
          AND o.id = $1
    `
    
    _, err := r.db.Exec(ctx, query, orderID)
    return err
}

func (r *orderRepository) FindNearestWarehouse(ctx context.Context, province string) (*model.Warehouse, error) {
    query := `
        SELECT id, name, code, address, province
        FROM warehouses
        WHERE is_active = true AND province = $1
        LIMIT 1
    `
    
    warehouse := &model.Warehouse{}
    err := r.db.QueryRow(ctx, query, province).Scan(
        &warehouse.ID,
        &warehouse.Name,
        &warehouse.Code,
        &warehouse.Address,
        &warehouse.Province,
    )
    
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            // Fallback to default warehouse
            err = r.db.QueryRow(ctx, `
                SELECT id, name, code, address, province
                FROM warehouses
                WHERE is_active = true
                ORDER BY created_at
                LIMIT 1
            `).Scan(
                &warehouse.ID,
                &warehouse.Name,
                &warehouse.Code,
                &warehouse.Address,
                &warehouse.Province,
            )
        }
    }
    
    return warehouse, err
}

func (r *orderRepository) AddStatusHistory(ctx context.Context, history *model.OrderStatusHistory) error {
    query := `
        INSERT INTO order_status_history (order_id, old_status, new_status, note, changed_by)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id, created_at
    `
    
    return r.db.QueryRow(ctx, query,
        history.OrderID,
        history.OldStatus,
        history.NewStatus,
        history.Note,
        history.ChangedBy,
    ).Scan(&history.ID, &history.CreatedAt)
}
```

**☐ Task 5.11: Create Warehouse Model**

File: `internal/domains/order/model/warehouse.go`

```go
package model

import (
    "github.com/google/uuid"
)

type Warehouse struct {
    ID       uuid.UUID `json:"id"`
    Name     string    `json:"name"`
    Code     string    `json:"code"`
    Address  string    `json:"address"`
    Province string    `json:"province"`
}
```

**☐ Task 5.12: Create Order Service với Business Logic**

File: `internal/domains/order/service/order_service.go`

```go
package service

import (
    "context"
    "errors"
    "time"
    "github.com/google/uuid"
    "bookstore-backend/internal/domains/order/model"
    "bookstore-backend/internal/domains/order/repository"
    "bookstore-backend/internal/domains/order/dto"
    "bookstore-backend/internal/domains/cart/service"
)

type OrderService struct {
    orderRepo   repository.OrderRepository
    addressRepo repository.AddressRepository
    cartService *service.CartService
}

func NewOrderService(
    orderRepo repository.OrderRepository,
    addressRepo repository.AddressRepository,
    cartService *service.CartService,
) *OrderService {
    return &OrderService{
        orderRepo:   orderRepo,
        addressRepo: addressRepo,
        cartService: cartService,
    }
}

// CreateOrder - Create order from cart
func (s *OrderService) CreateOrder(ctx context.Context, userID uuid.UUID, req dto.CreateOrderRequest) (*dto.OrderResponse, error) {
    // Validate request
    if err := req.Validate(); err != nil {
        return nil, err
    }
    
    // Get cart
    cart, err := s.cartService.GetCart(ctx, &userID, nil)
    if err != nil {
        return nil, err
    }
    
    if len(cart.Items) == 0 {
        return nil, errors.New("cart is empty")
    }
    
    // Validate address
    addressID, err := uuid.Parse(req.AddressID)
    if err != nil {
        return nil, errors.New("invalid address ID")
    }
    
    address, err := s.addressRepo.FindByID(ctx, addressID)
    if err != nil {
        return nil, err
    }
    
    if address.UserID != userID {
        return nil, errors.New("address does not belong to user")
    }
    
    // Find nearest warehouse
    warehouse, err := s.orderRepo.FindNearestWarehouse(ctx, address.Province)
    if err != nil {
        return nil, errors.New("no available warehouse")
    }
    
    // Calculate totals
    subtotal := cart.Summary.Subtotal
    discountAmount := cart.Summary.Discount
    shippingFee := s.calculateShippingFee(req.PaymentMethod, subtotal)
    taxAmount := 0.0 // TODO: Calculate tax if needed
    total := subtotal - discountAmount + shippingFee + taxAmount
    
    // Create order
    order := &model.Order{
        UserID:         userID,
        AddressID:      addressID,
        WarehouseID:    &warehouse.ID,
        Subtotal:       subtotal,
        DiscountAmount: discountAmount,
        ShippingFee:    shippingFee,
        TaxAmount:      taxAmount,
        Total:          total,
        Status:         "pending",
        PaymentMethod:  req.PaymentMethod,
        PaymentStatus:  "pending",
        Notes:          &req.Notes,
    }
    
    // If promo code used, save it
    if req.PromotionCode != "" {
        order.PromotionCode = &req.PromotionCode
        // TODO: Validate and apply promotion
    }
    
    // Create order in DB
    if err := s.orderRepo.Create(ctx, order); err != nil {
        return nil, err
    }
    
    // Create order items from cart items
    orderItems := make([]model.OrderItem, len(cart.Items))
    for i, cartItem := range cart.Items {
        orderItems[i] = model.OrderItem{
            OrderID:         order.ID,
            BookID:          uuid.MustParse(cartItem.BookID),
            BookTitle:       cartItem.Title,
            BookAuthor:      cartItem.AuthorName,
            BookFormat:      cartItem.Format,
            Quantity:        cartItem.Quantity,
            PriceAtPurchase: cartItem.Price,
            Subtotal:        cartItem.Subtotal,
        }
    }
    
    if err := s.orderRepo.CreateOrderItems(ctx, orderItems); err != nil {
        return nil, err
    }
    
    // Reserve inventory
    if err := s.orderRepo.ReserveInventory(ctx, orderItems, *warehouse.ID); err != nil {
        return nil, err
    }
    
    // Add status history
    history := &model.OrderStatusHistory{
        OrderID:   order.ID,
        NewStatus: "pending",
        Note:      func() *string { s := "Order created"; return &s }(),
        ChangedBy: &userID,
    }
    s.orderRepo.AddStatusHistory(ctx, history)
    
    // Clear cart after successful order
    s.cartService.ClearCart(ctx, &userID, nil)
    
    // Return order response
    return s.GetOrderByID(ctx, userID, order.ID.String())
}

// GetOrderByID - Get order details
func (s *OrderService) GetOrderByID(ctx context.Context, userID uuid.UUID, orderID string) (*dto.OrderResponse, error) {
    id, err := uuid.Parse(orderID)
    if err != nil {
        return nil, errors.New("invalid order ID")
    }
    
    order, err := s.orderRepo.FindByID(ctx, id)
    if err != nil {
        return nil, err
    }
    
    // Check if order belongs to user
    if order.UserID != userID {
        return nil, errors.New("order not found")
    }
    
    return s.toOrderResponse(order), nil
}

// GetUserOrders - Get all orders for user
func (s *OrderService) GetUserOrders(ctx context.Context, userID uuid.UUID, page, limit int) (*dto.OrderListResponse, error) {
    if page < 1 {
        page = 1
    }
    if limit < 1 || limit > 100 {
        limit = 20
    }
    
    orders, total, err := s.orderRepo.FindByUserID(ctx, userID, page, limit)
    if err != nil {
        return nil, err
    }
    
    orderResponses := make([]dto.OrderResponse, len(orders))
    for i, order := range orders {
        orderResponses[i] = *s.toOrderResponse(&order)
    }
    
    totalPages := int((total + int64(limit) - 1) / int64(limit))
    
    return &dto.OrderListResponse{
        Orders: orderResponses,
        Meta: dto.PaginationMeta{
            Page:       page,
            Limit:      limit,
            Total:      total,
            TotalPages: totalPages,
        },
    }, nil
}

// CancelOrder - Cancel order before payment/processing
func (s *OrderService) CancelOrder(ctx context.Context, userID uuid.UUID, orderID string, reason string) error {
    id, err := uuid.Parse(orderID)
    if err != nil {
        return errors.New("invalid order ID")
    }
    
    order, err := s.orderRepo.FindByID(ctx, id)
    if err != nil {
        return err
    }
    
    if order.UserID != userID {
        return errors.New("order not found")
    }
    
    // Check if order can be cancelled
    if order.Status != "pending" && order.Status != "confirmed" {
        return errors.New("order cannot be cancelled at this stage")
    }
    
    // Update status
    if err := s.orderRepo.UpdateStatus(ctx, id, "cancelled"); err != nil {
        return err
    }
    
    // Release inventory
    if err := s.orderRepo.ReleaseInventory(ctx, id); err != nil {
        return err
    }
    
    // Add status history
    history := &model.OrderStatusHistory{
        OrderID:   id,
        OldStatus: &order.Status,
        NewStatus: "cancelled",
        Note:      &reason,
        ChangedBy: &userID,
    }
    s.orderRepo.AddStatusHistory(ctx, history)
    
    // TODO: If already paid, initiate refund
    
    return nil
}

// calculateShippingFee - Calculate shipping fee based on payment method and total
func (s *OrderService) calculateShippingFee(paymentMethod string, subtotal float64) float64 {
    // Free shipping for orders over 500,000 VND
    if subtotal >= 500000 {
        return 0
    }
    
    // COD has extra fee
    if paymentMethod == "cod" {
        return 30000 // Base shipping + COD fee
    }
    
    return 15000 // Standard shipping
}

// toOrderResponse - Convert order model to response DTO
func (s *OrderService) toOrderResponse(order *model.Order) *dto.OrderResponse {
    items := make([]dto.OrderItemResponse, len(order.Items))
    for i, item := range order.Items {
        items[i] = dto.OrderItemResponse{
            BookID:          item.BookID.String(),
            BookTitle:       item.BookTitle,
            BookAuthor:      item.BookAuthor,
            BookFormat:      item.BookFormat,
            Quantity:        item.Quantity,
            PriceAtPurchase: item.PriceAtPurchase,
            Subtotal:        item.Subtotal,
        }
    }
    
    var addressResp dto.AddressResponse
    if order.Address != nil {
        addressResp = dto.AddressResponse{
            RecipientName: order.Address.RecipientName,
            Phone:         order.Address.Phone,
            Province:      order.Address.Province,
            District:      order.Address.District,
            Ward:          order.Address.Ward,
            Street:        order.Address.Street,
            FullAddress:   order.Address.Street + ", " + order.Address.Ward + ", " + order.Address.District + ", " + order.Address.Province,
        }
    }
    
    resp := &dto.OrderResponse{
        ID:             order.ID.String(),
        OrderNumber:    order.OrderNumber,
        Status:         order.Status,
        PaymentMethod:  order.PaymentMethod,
        PaymentStatus:  order.PaymentStatus,
        Subtotal:       order.Subtotal,
        DiscountAmount: order.DiscountAmount,
        ShippingFee:    order.ShippingFee,
        Total:          order.Total,
        Items:          items,
        Address:        addressResp,
        CreatedAt:      order.CreatedAt.Format(time.RFC3339),
    }
    
    if order.TrackingNumber != nil {
        resp.TrackingNumber = *order.TrackingNumber
    }
    
    if order.PaidAt != nil {
        resp.PaidAt = order.PaidAt.Format(time.RFC3339)
    }
    
    if order.ShippedAt != nil {
        resp.ShippedAt = order.ShippedAt.Format(time.RFC3339)
    }
    
    if order.DeliveredAt != nil {
        resp.DeliveredAt = order.DeliveredAt.Format(time.RFC3339)
    }
    
    return resp
}
```


### **Ngày 8-9: Order Handler \& Routes**

**☐ Task 5.13: Create Order Handler**

File: `internal/domains/order/handler/order_handler.go`

```go
package handler

import (
    "net/http"
    "strconv"
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "bookstore-backend/internal/domains/order/service"
    "bookstore-backend/internal/domains/order/dto"
)

type OrderHandler struct {
    orderService *service.OrderService
}

func NewOrderHandler(orderService *service.OrderService) *OrderHandler {
    return &OrderHandler{orderService: orderService}
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
    userIDStr := c.GetString("user_id")
    userID, _ := uuid.Parse(userIDStr)
    
    var req dto.CreateOrderRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "success": false,
            "error": gin.H{
                "code":    "VAL_001",
                "message": "Invalid request body",
            },
        })
        return
    }
    
    order, err := h.orderService.CreateOrder(c.Request.Context(), userID, req)
    if err != nil {
        statusCode := http.StatusBadRequest
        errorCode := "BIZ_001"
        
        if err.Error() == "cart is empty" {
            errorCode = "BIZ_002"
        } else if err.Error() == "no available warehouse" {
            statusCode = http.StatusServiceUnavailable
            errorCode = "SYS_002"
        }
        
        c.JSON(statusCode, gin.H{
            "success": false,
            "error": gin.H{
                "code":    errorCode,
                "message": err.Error(),
            },
        })
        return
    }
    
    c.JSON(http.StatusCreated, gin.H{
        "success": true,
        "data":    order,
        "message": "Order created successfully",
    })
}

func (h *OrderHandler) GetOrder(c *gin.Context) {
    userIDStr := c.GetString("user_id")
    userID, _ := uuid.Parse(userIDStr)
    
    orderID := c.Param("id")
    
    order, err := h.orderService.GetOrderByID(c.Request.Context(), userID, orderID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{
            "success": false,
            "error": gin.H{
                "code":    "RES_001",
                "message": "Order not found",
            },
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data":    order,
    })
}

func (h *OrderHandler) GetUserOrders(c *gin.Context) {
    userIDStr := c.GetString("user_id")
    userID, _ := uuid.Parse(userIDStr)
    
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
    
    orders, err := h.orderService.GetUserOrders(c.Request.Context(), userID, page, limit)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "success": false,
            "error": gin.H{
                "code":    "SYS_001",
                "message": "Failed to fetch orders",
            },
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data":    orders.Orders,
        "meta":    orders.Meta,
    })
}

func (h *OrderHandler) CancelOrder(c *gin.Context) {
    userIDStr := c.GetString("user_id")
    userID, _ := uuid.Parse(userIDStr)
    
    orderID := c.Param("id")
    
    var req struct {
        Reason string `json:"reason"`
    }
    c.ShouldBindJSON(&req)
    
    if err := h.orderService.CancelOrder(c.Request.Context(), userID, orderID, req.Reason); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "success": false,
            "error": gin.H{
                "code":    "BIZ_003",
                "message": err.Error(),
            },
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "message": "Order cancelled successfully",
    })
}
```

**☐ Task 5.14: Create Address Handler**

File: `internal/domains/order/handler/address_handler.go`

```go
package handler

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "bookstore-backend/internal/domains/order/service"
    "bookstore-backend/internal/domains/order/dto"
)

type AddressHandler struct {
    addressService *service.AddressService
}

func NewAddressHandler(addressService *service.AddressService) *AddressHandler {
    return &AddressHandler{addressService: addressService}
}

func (h *AddressHandler) CreateAddress(c *gin.Context) {
    userIDStr := c.GetString("user_id")
    userID, _ := uuid.Parse(userIDStr)
    
    var req dto.CreateAddressRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "success": false,
            "error": gin.H{
                "code":    "VAL_001",
                "message": "Invalid request body",
            },
        })
        return
    }
    
    address, err := h.addressService.CreateAddress(c.Request.Context(), userID, req)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "success": false,
            "error": gin.H{
                "code":    "BIZ_001",
                "message": err.Error(),
            },
        })
        return
    }
    
    c.JSON(http.StatusCreated, gin.H{
        "success": true,
        "data":    address,
    })
}

func (h *AddressHandler) GetUserAddresses(c *gin.Context) {
    userIDStr := c.GetString("user_id")
    userID, _ := uuid.Parse(userIDStr)
    
    addresses, err := h.addressService.GetUserAddresses(c.Request.Context(), userID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "success": false,
            "error": gin.H{
                "code":    "SYS_001",
                "message": "Failed to fetch addresses",
            },
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data":    addresses,
    })
}
```

**☐ Task 5.15: Setup Order \& Address Routes**

File: `cmd/api/main.go` (cập nhật)

```go
// ... trong main() ...

// Initialize order repositories & services
addressRepo := orderRepository.NewAddressRepository(pgPool)
orderRepo := orderRepository.NewOrderRepository(pgPool)

addressService := orderService.NewAddressService(addressRepo)
orderService := orderService.NewOrderService(orderRepo, addressRepo, cartService)

addressHandler := orderHandler.NewAddressHandler(addressService)
orderHandler := orderHandler.NewOrderHandler(orderService)

// Routes
v1 := r.Group("/v1")
{
    // ... existing routes ...
    
    // Address routes (authenticated)
    addresses := v1.Group("/user/addresses")
    addresses.Use(middleware.AuthRequired(cfg.JWT.Secret))
    {
        addresses.GET("", addressHandler.GetUserAddresses)
        addresses.POST("", addressHandler.CreateAddress)
        addresses.PATCH("/:id", addressHandler.UpdateAddress)
        addresses.DELETE("/:id", addressHandler.DeleteAddress)
    }
    
    // Order routes (authenticated)
    orders := v1.Group("/orders")
    orders.Use(middleware.AuthRequired(cfg.JWT.Secret))
    {
        orders.POST("", orderHandler.CreateOrder)
        orders.GET("", orderHandler.GetUserOrders)
        orders.GET("/:id", orderHandler.GetOrder)
        orders.PATCH("/:id/cancel", orderHandler.CancelOrder)
    }
}
```


### **Ngày 10: Testing \& Documentation**

**☐ Task 5.16: Create Order Service Tests**

File: `internal/domains/order/service/order_service_test.go`

```go
package service_test

import (
    "context"
    "testing"
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

func TestOrderService_CreateOrder_Success(t *testing.T) {
    // Mock repositories
    // Test create order flow
    // Assert order created, inventory reserved, cart cleared
}

func TestOrderService_CancelOrder_Success(t *testing.T) {
    // Test cancel order
    // Assert inventory released, status updated
}
```

**☐ Task 5.17: Test Complete Checkout Flow**

```bash
# 1. Login
TOKEN=$(curl -s -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}' \
  | jq -r '.data.access_token')

# 2. Add items to cart
curl -X POST http://localhost:8080/v1/cart/items \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"book_id":"uuid-here","quantity":2}'

# 3. Create address
ADDRESS_ID=$(curl -s -X POST http://localhost:8080/v1/user/addresses \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "recipient_name": "Nguyen Van A",
    "phone": "0123456789",
    "province": "Hà Nội",
    "district": "Hoàng Mai",
    "ward": "Giáp Bát",
    "street": "123 Giải Phóng",
    "is_default": true
  }' | jq -r '.data.id')

# 4. Create order
ORDER=$(curl -s -X POST http://localhost:8080/v1/orders \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"address_id\": \"$ADDRESS_ID\",
    \"payment_method\": \"cod\",
    \"notes\": \"Giao giờ hành chính\"
  }")

echo $ORDER | jq .

# 5. Get order details
ORDER_ID=$(echo $ORDER | jq -r '.data.id')
curl -X GET http://localhost:8080/v1/orders/$ORDER_ID \
  -H "Authorization: Bearer $TOKEN" | jq .

# 6. Cancel order
curl -X PATCH http://localhost:8080/v1/orders/$ORDER_ID/cancel \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"reason":"Đặt nhầm"}' | jq .
```

**☐ Task 5.18: Create Postman Collection**

- [ ] Export all endpoints to Postman collection
- [ ] Add environment variables (base_url, token)
- [ ] Add test scripts cho từng request
- [ ] Document expected responses

**☐ Task 5.19: Performance Testing**

```bash
# Load test checkout flow với k6
k6 run --vus 50 --duration 30s checkout_test.js
```

**☐ Task 5.20: Update Documentation**

- [ ] Update API documentation với order endpoints
- [ ] Document order status flow
- [ ] Add sequence diagrams cho checkout process
- [ ] Update ERD diagram

***

# **FILE DOCS: SPRINT 4 \& SPRINT 5 SUMMARY**

Dưới đây là tài liệu tổng hợp chi tiết cho Sprint 4 và Sprint 5:

***

# **BACKEND DEVELOPER TO-DO LIST**

## **SPRINT 4-5: CART \& CHECKOUT**

**Hệ Thống E-commerce Bán Sách Online**

***

## **TỔNG QUAN**

### **Sprint 4: Shopping Cart (Tuần 7-8)**

**Mục tiêu:** Xây dựng hệ thống giỏ hàng hoàn chỉnh hỗ trợ cả guest và authenticated users

**Deliverables:**

- ✅ Cart database schema
- ✅ Cart service với stock validation
- ✅ Session management cho guest users
- ✅ Redis caching cho cart data
- ✅ API endpoints: Get, Add, Update, Delete cart items


### **Sprint 5: Checkout \& Order Management (Tuần 9-10)**

**Mục tiêu:** Triển khai checkout flow và order management system

**Deliverables:**

- ✅ Order database schema với partitioning
- ✅ Multi-warehouse inventory system
- ✅ Address management
- ✅ Order creation với inventory reservation
- ✅ Order status tracking
- ✅ Cancel order functionality

***

## **SPRINT 4: SHOPPING CART - CHI TIẾT TASKS**

### **Tuần 7 - Ngày 1-2: Database Setup**

#### **Task 4.1: Create Cart Tables**

**File:** `migrations/000006_create_carts.up.sql`

**Công việc:**

1. Tạo bảng `carts` với các cột:
    - `id` (UUID primary key)
    - `user_id` (reference users, nullable)
    - `session_id` (TEXT, nullable)
    - `expires_at` (timestamp - cart hết hạn sau 30 phút)
    - Constraint: phải có user_id HOẶC session_id
2. Tạo bảng `cart_items`:
    - `cart_id` + `book_id` (composite primary key)
    - `quantity` (integer, check > 0)
    - `added_at` (timestamp)
3. Tạo indexes:
    - `idx_carts_user` trên `user_id`
    - `idx_carts_session` trên `session_id`
    - `idx_carts_expires` trên `expires_at`
4. Run migration: `./scripts/migrate.sh up`

**Testing:**

```sql
-- Verify tables created
\d carts
\d cart_items

-- Test constraint
INSERT INTO carts (expires_at) VALUES (NOW() + INTERVAL '30 minutes');
-- Should fail: violates user_or_session constraint
```


***

#### **Task 4.2-4.4: Models \& DTOs**

**File:** `internal/domains/cart/model/cart.go`

**Công việc:**

1. Define Cart struct với fields theo database schema
2. Define CartItem struct
3. Define CartBook struct (cho JOIN data)

**File:** `internal/domains/cart/dto/cart.go`

**Công việc:**

1. `AddToCartRequest` với validation:
    - BookID required
    - Quantity 1-99
2. `UpdateCartItemRequest`:
    - Quantity 0-99 (0 = delete)
3. `CartResponse`:
    - Items array
    - Summary (subtotal, discount, total)
    - ExpiresAt

**Testing:**

```go
func TestAddToCartRequest_Validate(t *testing.T) {
    req := dto.AddToCartRequest{
        BookID: "",
        Quantity: 0,
    }
    err := req.Validate()
    assert.Error(t, err)
}
```


***

### **Tuần 7 - Ngày 3-4: Repository Layer**

#### **Task 4.5-4.6: Cart Repository**

**File:** `internal/domains/cart/repository/postgres.go`

**Công việc chính:**

1. **Create cart:**
```go
func (r *postgresRepository) Create(ctx context.Context, cart *model.Cart) error
```

- Insert vào bảng carts
- Return ID và timestamps

2. **Find cart:**
```go
func (r *postgresRepository) FindByUserID(ctx context.Context, userID uuid.UUID) (*model.Cart, error)
func (r *postgresRepository) FindBySessionID(ctx context.Context, sessionID string) (*model.Cart, error)
```

- WHERE user_id/session_id AND expires_at > NOW()
- ORDER BY created_at DESC LIMIT 1

3. **Add/Update item:**
```go
func (r *postgresRepository) AddItem(ctx context.Context, item *model.CartItem) error
```

- Use ON CONFLICT (cart_id, book_id) DO UPDATE
- Tăng quantity nếu item đã tồn tại

4. **Get items with books:**
```go
func (r *postgresRepository) GetItemsWithBooks(ctx context.Context, cartID uuid.UUID) ([]model.CartItem, error)
```

- JOIN với books, authors, warehouse_inventory
- Calculate available stock: `SUM(quantity - reserved)`

5. **Check stock:**
```go
func (r *postgresRepository) CheckStock(ctx context.Context, bookID uuid.UUID, quantity int) (bool, int, error)
```

- Query warehouse_inventory
- Return hasStock bool và availableStock int

**Testing:**

```go
func TestCartRepository_AddItem_DuplicateBookID(t *testing.T) {
    // Setup testcontainers
    // Add same book twice
    // Assert quantity incremented
}
```


***

### **Tuần 7 - Ngày 5-6: Service Layer**

#### **Task 4.7: Cart Service**

**File:** `internal/domains/cart/service/cart_service.go`

**Business Logic quan trọng:**

1. **GetOrCreateCart:**

- Tìm cart hiện tại (chưa expired)
- Nếu không có → tạo mới với expires_at = NOW() + 30 phút

2. **AddToCart:**
```go
// Pseudo code
- Validate request
- Parse book ID
- CHECK STOCK trước khi add ⚠️
- Get/Create cart
- Add item
- Update expires_at (reset 30 phút)
- Return updated cart
```

3. **UpdateCartItem:**

- Nếu quantity = 0 → remove item
- Nếu quantity > 0 → check stock
- Update quantity
- Reset expiration

4. **buildCartResponse:**

- Convert cart items → DTO
- Calculate summary:
    - subtotal = SUM(price * quantity)
    - discount (nếu có promo)
    - total = subtotal - discount

**Constants:**

```go
const (
    CartExpirationDuration = 30 * time.Minute
    CartExpirationWarning  = 25 * time.Minute
)
```

**Lưu ý:**

- ⚠️ Luôn validate stock trước khi add/update
- ⚠️ Reset expiration sau mỗi interaction
- ⚠️ Handle race condition cho concurrent requests

***

### **Tuần 7-8 - Ngày 7-8: Handler \& Routes**

#### **Task 4.8: Session Management**

**File:** `pkg/session/session.go`

```go
func GetOrCreateSessionID(c *gin.Context) string {
    sessionID, err := c.Cookie(SessionCookieName)
    if err != nil || sessionID == "" {
        sessionID = uuid.New().String()
        c.SetCookie(
            SessionCookieName,
            sessionID,
            3600*24*30, // 30 days
            "/",
            "",
            false, // set true in production
            true,  // httpOnly
        )
    }
    return sessionID
}
```

**Mục đích:** Tạo và quản lý session cho guest users

***

#### **Task 4.9: Cart Handler**

**File:** `internal/domains/cart/handler/cart_handler.go`

**Endpoints cần implement:**

1. **GET /v1/cart**
```go
func (h *CartHandler) GetCart(c *gin.Context)
```

- Extract userID (nếu authenticated) hoặc sessionID
- Call service.GetCart()
- Return 200 với cart data

2. **POST /v1/cart/items**
```go
func (h *CartHandler) AddToCart(c *gin.Context)
```

- Bind JSON request
- Validate
- Call service.AddToCart()
- Return 200 hoặc 400 (out of stock)

3. **PATCH /v1/cart/items/:book_id**
```go
func (h *CartHandler) UpdateCartItem(c *gin.Context)
```

- Get book_id from params
- Bind quantity từ body
- Update item
- Return updated cart

4. **DELETE /v1/cart/items/:book_id**

- Remove item
- Return updated cart

5. **DELETE /v1/cart**

- Clear all items
- Return success message

**Helper function:**

```go
func (h *CartHandler) getUserOrSession(c *gin.Context) (*uuid.UUID, *string) {
    // Check if authenticated
    if userIDStr, exists := c.Get("user_id"); exists {
        userID := uuid.Parse(userIDStr.(string))
        return &userID, nil
    }
    // Guest user
    sessionID := session.GetOrCreateSessionID(c)
    return nil, &sessionID
}
```


***

### **Tuần 8 - Ngày 9-10: Testing \& Optimization**

#### **Task 4.11: Unit Tests**

**Test cases quan trọng:**

1. **Service Tests:**
```go
TestCartService_AddToCart_Success
TestCartService_AddToCart_OutOfStock
TestCartService_AddToCart_InvalidBookID
TestCartService_UpdateCartItem_SetToZero
TestCartService_GetCart_Expired
```

2. **Repository Tests:**
```go
TestCartRepository_AddItem_DuplicateBookID
TestCartRepository_CheckStock_Insufficient
```

**Run tests:**

```bash
go test ./internal/domains/cart/... -v -cover
```


***

#### **Task 4.13: Redis Caching**

**File:** `internal/infrastructure/cache/cart_cache.go`

**Caching strategy:**

- Key format: `cart:{cart_id}`
- TTL: 30 minutes (match cart expiration)
- Cache hit → return immediately
- Cache miss → query DB → cache result

**Update flow:**

```
AddToCart/UpdateItem → Clear cache → Get from DB → Set cache
```

**Testing:**

```bash
# Monitor Redis
redis-cli MONITOR

# Watch cache hits
redis-cli INFO stats | grep keyspace_hits
```


***

## **SPRINT 5: CHECKOUT \& ORDERS - CHI TIẾT TASKS**

### **Tuần 9 - Ngày 1-2: Database Schema**

#### **Task 5.1: Addresses Table**

**Migration:** `000007_create_addresses.up.sql`

**Schema highlights:**

- Phone validation: CHECK (phone ~ '^0[0-9]{9}\$')
- Unique default address per user: partial index WHERE is_default = true
- Trigger auto-update updated_at

**Seed data:**

```sql
INSERT INTO addresses (user_id, recipient_name, phone, province, district, ward, street, is_default)
VALUES 
  ('user-uuid-1', 'Nguyen Van A', '0123456789', 'Hà Nội', 'Hoàng Mai', 'Giáp Bát', '123 Giải Phóng', true);
```


***

#### **Task 5.2-5.3: Warehouses \& Inventory**

**Migration:** `000008_create_warehouses.up.sql`, `000009_create_warehouse_inventory.up.sql`

**Warehouse table:**

- `code` UNIQUE (e.g., 'HN001', 'HCM001')
- `latitude`, `longitude` (cho tính khoảng cách sau này)
- `is_active` boolean

**Inventory table:**

- Composite PK: (warehouse_id, book_id)
- `quantity` - total stock
- `reserved` - stock đang được reserve cho orders pending
- Constraint: `quantity >= reserved`
- View: `books_total_stock` aggregate từ multiple warehouses

**Seed inventory:**

```sql
-- Random stock 20-120 for each book in Hanoi warehouse
INSERT INTO warehouse_inventory (warehouse_id, book_id, quantity)
SELECT 
    (SELECT id FROM warehouses WHERE code = 'HN001'),
    b.id,
    floor(random() * 100 + 20)::int
FROM books b WHERE b.format != 'ebook';
```


***

#### **Task 5.4: Orders Tables**

**Migration:** `000010_create_orders.up.sql`

**Orders table (PARTITIONED):**

```sql
CREATE TABLE orders (...) PARTITION BY RANGE (created_at);

CREATE TABLE orders_2025_10 PARTITION OF orders
    FOR VALUES FROM ('2025-10-01') TO ('2025-11-01');
```

**Tại sao partition:**

- Performance: Query orders theo tháng nhanh hơn
- Maintenance: Drop old partitions dễ dàng
- Scalability: Mỗi partition có indexes riêng

**Order items table:**

- Snapshot book info (title, author, price) at purchase time
- Tránh data inconsistency khi book bị update/delete

**Order status flow:**

```
pending → confirmed → processing → shipped → delivered → completed
       ↓
   payment_failed
   cancelled
   refunded
```

**Status history table:**

- Audit trail cho mọi thay đổi status
- Lưu who changed, when, why

***

#### **Task 5.5: Order Number Generator**

**Function:** `generate_order_number()`

**Format:** `ORD-YYYYMMDD-NNNN`

Example: `ORD-20251031-0001`

**Logic:**

1. Count orders created today
2. Increment counter
3. Format với LPAD(4, '0')

**Testing:**

```sql
SELECT generate_order_number();
-- ORD-20251031-0001

SELECT generate_order_number();
-- ORD-20251031-0002
```


***

### **Tuần 9 - Ngày 3-4: Models \& DTOs**

#### **Task 5.7-5.8: Order Models**

**File:** `internal/domains/order/model/order.go`

**Key models:**

1. **Order:**

- All fields từ database
- Relationships: Items, Address
- Timestamps cho từng status change

2. **OrderItem:**

- Snapshot data (book_title, book_author, price_at_purchase)
- Không reference trực tiếp book để tránh stale data

3. **Address:**

- Đầy đủ thông tin giao hàng
- `FullAddress` computed field

**DTOs:**

**CreateOrderRequest:**

```json
{
  "address_id": "uuid",
  "payment_method": "cod|vnpay|momo",
  "promotion_code": "SUMMER20",
  "notes": "Giao giờ hành chính"
}
```

**OrderResponse:**

```json
{
  "id": "uuid",
  "order_number": "ORD-20251031-0001",
  "status": "pending",
  "items": [...],
  "address": {...},
  "summary": {
    "subtotal": 200000,
    "shipping_fee": 30000,
    "total": 230000
  }
}
```


***

### **Tuần 9 - Ngày 5-7: Repository \& Service**

#### **Task 5.9: Address Repository**

**Chức năng chính:**

1. **Create với transaction:**
```go
func (r *addressRepository) Create(ctx context.Context, address *model.Address) error {
    tx := r.db.Begin()
    // If is_default = true, unset others
    if address.IsDefault {
        tx.Exec("UPDATE addresses SET is_default = false WHERE user_id = $1")
    }
    tx.Exec("INSERT INTO addresses ...")
    tx.Commit()
}
```

2. **FindByUserID:**

- ORDER BY is_default DESC, created_at DESC
- Default address luôn đầu tiên

***

#### **Task 5.10: Order Repository**

**Critical methods:**

1. **Create order:**
```go
func Create(ctx context.Context, order *model.Order) error
```

- Generate order number
- Insert order
- Return với ID và timestamps

2. **ReserveInventory (TRANSACTION):**
```go
func ReserveInventory(ctx context.Context, items []OrderItem, warehouseID uuid.UUID) error {
    for each item {
        UPDATE warehouse_inventory
        SET reserved = reserved + item.quantity
        WHERE warehouse_id = ? AND book_id = ?
          AND (quantity - reserved) >= item.quantity
        
        IF affected_rows = 0 THEN
            ROLLBACK
            RETURN error "insufficient stock"
    }
    COMMIT
}
```

**⚠️ Race condition handling:**

- Use row-level locking: `FOR UPDATE`
- Hoặc optimistic locking với version field

3. **ReleaseInventory:**
```go
func ReleaseInventory(ctx context.Context, orderID uuid.UUID) error {
    UPDATE warehouse_inventory wi
    SET reserved = reserved - oi.quantity
    FROM order_items oi
    WHERE wi.book_id = oi.book_id AND oi.order_id = ?
}
```

4. **FindNearestWarehouse:**
```go
// V1: Simple - match province
SELECT * FROM warehouses WHERE province = ? AND is_active = true LIMIT 1

// V2: Advanced - calculate distance
SELECT *, 
  ST_Distance(
    ST_MakePoint(latitude, longitude),
    ST_MakePoint(?, ?)
  ) as distance
FROM warehouses
ORDER BY distance
LIMIT 1
```


***

#### **Task 5.12: Order Service**

**Core business logic:**

1. **CreateOrder flow:**
```
1. Validate request
2. Get cart (must not empty)
3. Validate address (belongs to user)
4. Find nearest warehouse
5. Calculate totals:
   - Subtotal từ cart
   - Shipping fee (free if > 500k, COD +15k)
   - Discount (nếu có promo)
   - Tax (future)
6. BEGIN TRANSACTION
   a. Create order
   b. Create order items (snapshot book info)
   c. Reserve inventory
   d. Add status history
7. COMMIT
8. Clear cart
9. Return order
```

2. **Shipping fee calculation:**
```go
func calculateShippingFee(method string, subtotal float64) float64 {
    if subtotal >= 500000 {
        return 0 // Free shipping
    }
    if method == "cod" {
        return 30000 // Base + COD fee
    }
    return 15000
}
```

3. **CancelOrder flow:**
```
1. Validate order belongs to user
2. Check status (only pending/confirmed can cancel)
3. Update status to cancelled
4. Release inventory
5. Add status history
6. If paid → initiate refund (future)
```


***

### **Tuần 10 - Ngày 8-9: Handler \& Routes**

#### **Task 5.13-5.14: Handlers**

**OrderHandler endpoints:**

1. **POST /v1/orders** (Create order)

- Require authentication
- Bind CreateOrderRequest
- Validate
- Call service.CreateOrder()
- Return 201 Created

2. **GET /v1/orders** (List orders)

- Pagination (page, limit)
- Filter by status (future)
- Return orders array + meta

3. **GET /v1/orders/:id** (Order detail)

- Verify order belongs to user
- Return full order with items

4. **PATCH /v1/orders/:id/cancel**

- Bind cancel reason
- Call service.CancelOrder()
- Return 200 OK

**AddressHandler endpoints:**

1. **GET /v1/user/addresses**

- Return all addresses for user
- Default address first

2. **POST /v1/user/addresses**

- Create new address
- Validate phone format
- Return created address

3. **PATCH /v1/user/addresses/:id**

- Update address
- Handle set default

4. **DELETE /v1/user/addresses/:id**

- Soft delete or hard delete
- Cannot delete if used in active order

***

### **Tuần 10 - Ngày 10: Testing**

#### **Task 5.16-5.20: Complete Testing**

**Unit tests:**

```bash
go test ./internal/domains/order/service/... -v -cover
# Target: >80% coverage
```

**Integration tests:**

```go
func TestCheckoutFlow_E2E(t *testing.T) {
    // Setup testcontainers
    // 1. Create user
    // 2. Add items to cart
    // 3. Create address
    // 4. Create order
    // 5. Verify:
    //    - Order created
    //    - Inventory reserved
    //    - Cart cleared
    //    - Order items match cart
}
```

**Manual testing checklist:**

- [ ] Guest add to cart → login → cart merged
- [ ] Add to cart → insufficient stock → error
- [ ] Create order → COD → shipping fee correct
- [ ] Create order → online payment → shipping fee correct
- [ ] Create order → free shipping (>500k)
- [ ] Cancel order → inventory released
- [ ] Create order → check warehouse_inventory.reserved increased
- [ ] Multiple concurrent orders same book → no overselling

**Performance testing:**

```javascript
// k6 script
export default function() {
  // Login
  // Add to cart
  // Checkout
  // Check response time < 500ms
}
```


***

## **KEY TAKEAWAYS**

### **Sprint 4 - Cart Management**

✅ **Learned:**

- Session management cho guest users
- Race condition handling với inventory
- Redis caching strategy
- Soft expiration với background cleanup

⚠️ **Common Pitfalls:**

- Không validate stock trước khi add → overselling
- Không reset cart expiration → cart bị xóa sớm
- Cache inconsistency giữa cart và inventory


### **Sprint 5 - Orders \& Checkout**

✅ **Learned:**

- Database partitioning cho scalability
- Transaction management phức tạp
- Inventory reservation pattern
- Order status state machine
- Snapshot pattern cho historical data

⚠️ **Common Pitfalls:**

- Race condition khi reserve inventory → use locking
- Không release inventory khi cancel → inventory leak
- Snapshot thiếu data → không reproduce order cũ
- Partition management → cần cronjob tạo partition mới

***

## **NEXT STEPS**

Sau khi hoàn thành Sprint 4-5, tiếp theo cần:

1. **Sprint 6: Payment Integration** (VNPay, Momo)
2. **Sprint 7: Background Jobs** (Email, inventory sync, cleanup)
3. **Sprint 8: Admin Panel** (Manage orders, update status)
4. **Sprint 9-10: Production Ready** (Monitoring, security, performance)

***

## **APPENDIX**

### **Database Diagram**

```
users ──┬── carts ── cart_items ── books
        └── addresses
        └── orders ── order_items ── books
                  └── warehouses ── warehouse_inventory ── books
```


### **API Endpoints Summary**

| Method | Endpoint | Auth | Description |
| :-- | :-- | :-- | :-- |
| GET | /v1/cart | Optional | Get cart |
| POST | /v1/cart/items | Optional | Add to cart |
| PATCH | /v1/cart/items/:id | Optional | Update quantity |
| DELETE | /v1/cart/items/:id | Optional | Remove item |
| GET | /v1/user/addresses | Required | List addresses |
| POST | /v1/user/addresses | Required | Create address |
| POST | /v1/orders | Required | Create order |
| GET | /v1/orders | Required | List orders |
| GET | /v1/orders/:id | Required | Order detail |
| PATCH | /v1/orders/:id/cancel | Required | Cancel order |

### **Error Codes Reference**

| Code | HTTP | Description |
| :-- | :-- | :-- |
| BIZ_001 | 409 | Out of stock |
| BIZ_002 | 400 | Cart is empty |
| BIZ_003 | 400 | Cannot cancel order |
| RES_001 | 404 | Resource not found |


***

**Tài liệu này cung cấp hướng dẫn chi tiết cho backend developer để triển khai Sprint 4 và Sprint 5. Mỗi task đều có code examples, testing guidelines, và common pitfalls để tránh.**
<span style="display:none">[^1][^2][^3][^4][^5][^6][^7][^8]</span>

<div align="center">⁂</div>

[^1]: https://github.com/pwagnerde/ecommerce-backend

[^2]: https://www.scribd.com/document/904988382/Documentation-Template

[^3]: https://www.studocu.vn/vn/document/hoc-vien-hang-khong-viet-nam/quan-tri-du-an/personal-project-overview-and-scenario/92882546

[^4]: https://www.slideshare.net/slideshow/ecommerce-documentation/71682860

[^5]: https://www.postman.com/templates/collections/ecommerce-store-api/

[^6]: https://ekransamara.ru/files/7cabc7bb-87fc-494f-ad0d-bf0fd370f1c7.pdf

[^7]: https://bitbag.io/blog/how-to-prepare-project-ecommerce-documentation

[^8]: https://www.linkedin.com/pulse/example-project-e-commerce-website-supun-ruwanthi-perera-phkcc

