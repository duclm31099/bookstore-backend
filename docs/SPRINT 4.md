<img src="https://r2cdn.perplexity.ai/pplx-full-logo-primary-dark%402x.png" style="height:64px;margin-right:32px"/>

# **SPRINT 4-5: CART \& CHECKOUT (Tuần 7-10)**


***

## **SPRINT 4: Shopping Cart (Tuần 7-8)**

### **Ngày 1-2: Cart Database \& Models**

**☐ Task 4.1: Create Cart Tables Migration**

File: `migrations/000006_create_carts.up.sql`

```sql
-- Carts table
CREATE TABLE carts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    session_id TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT user_or_session CHECK (
        (user_id IS NOT NULL AND session_id IS NULL) OR
        (user_id IS NULL AND session_id IS NOT NULL)
    )
);

CREATE INDEX idx_carts_user ON carts(user_id);
CREATE INDEX idx_carts_session ON carts(session_id);
CREATE INDEX idx_carts_expires ON carts(expires_at);

-- Cart items table
CREATE TABLE cart_items (
    cart_id UUID NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
    book_id UUID NOT NULL REFERENCES books(id),
    quantity INT NOT NULL CHECK (quantity > 0),
    added_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (cart_id, book_id)
);

CREATE INDEX idx_cart_items_book ON cart_items(book_id);

-- Trigger to update cart updated_at
CREATE TRIGGER carts_updated_at BEFORE UPDATE ON carts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

File: `migrations/000006_create_carts.down.sql`

```sql
DROP TRIGGER IF EXISTS carts_updated_at ON carts;
DROP TABLE IF EXISTS cart_items;
DROP TABLE IF EXISTS carts;
```

**☐ Task 4.2: Run Migration**

```bash
./scripts/migrate.sh up
```

**☐ Task 4.3: Create Cart Models**

File: `internal/domains/cart/model/cart.go`

```go
package model

import (
    "time"
    "github.com/google/uuid"
)

type Cart struct {
    ID        uuid.UUID  `json:"id"`
    UserID    *uuid.UUID `json:"user_id"`
    SessionID *string    `json:"session_id"`
    ExpiresAt time.Time  `json:"expires_at"`
    CreatedAt time.Time  `json:"created_at"`
    UpdatedAt time.Time  `json:"updated_at"`
    
    // Relationships
    Items []CartItem `json:"items"`
}

type CartItem struct {
    CartID   uuid.UUID `json:"cart_id"`
    BookID   uuid.UUID `json:"book_id"`
    Quantity int       `json:"quantity"`
    AddedAt  time.Time `json:"added_at"`
    
    // Populated via join
    Book *CartBook `json:"book,omitempty"`
}

type CartBook struct {
    ID             uuid.UUID `json:"id"`
    Title          string    `json:"title"`
    Slug           string    `json:"slug"`
    Price          float64   `json:"price"`
    CompareAtPrice *float64  `json:"compare_at_price"`
    CoverURL       *string   `json:"cover_url"`
    Stock          int       `json:"stock"` // Available stock
    AuthorName     string    `json:"author_name"`
    Format         string    `json:"format"`
}
```

**☐ Task 4.4: Create Cart DTOs**

File: `internal/domains/cart/dto/cart.go`

```go
package dto

import (
    validation "github.com/go-ozzo/ozzo-validation/v4"
)

type AddToCartRequest struct {
    BookID   string `json:"book_id"`
    Quantity int    `json:"quantity"`
}

func (r AddToCartRequest) Validate() error {
    return validation.ValidateStruct(&r,
        validation.Field(&r.BookID, validation.Required),
        validation.Field(&r.Quantity, validation.Required, validation.Min(1), validation.Max(99)),
    )
}

type UpdateCartItemRequest struct {
    Quantity int `json:"quantity"`
}

func (r UpdateCartItemRequest) Validate() error {
    return validation.ValidateStruct(&r,
        validation.Field(&r.Quantity, validation.Required, validation.Min(0), validation.Max(99)),
    )
}

type CartResponse struct {
    ID        string         `json:"id"`
    Items     []CartItemResponse `json:"items"`
    Summary   CartSummary    `json:"summary"`
    ExpiresAt string         `json:"expires_at"`
}

type CartItemResponse struct {
    BookID         string  `json:"book_id"`
    Title          string  `json:"title"`
    Slug           string  `json:"slug"`
    AuthorName     string  `json:"author_name"`
    Price          float64 `json:"price"`
    CompareAtPrice float64 `json:"compare_at_price"`
    CoverURL       string  `json:"cover_url"`
    Quantity       int     `json:"quantity"`
    Subtotal       float64 `json:"subtotal"`
    Stock          int     `json:"stock"`
    Format         string  `json:"format"`
}

type CartSummary struct {
    ItemCount      int     `json:"item_count"`
    TotalQuantity  int     `json:"total_quantity"`
    Subtotal       float64 `json:"subtotal"`
    Discount       float64 `json:"discount"`
    ShippingFee    float64 `json:"shipping_fee"`
    Total          float64 `json:"total"`
}
```


### **Ngày 3-4: Cart Repository**

**☐ Task 4.5: Create Cart Repository Interface**

File: `internal/domains/cart/repository/repository.go`

```go
package repository

import (
    "context"
    "github.com/google/uuid"
    "bookstore-backend/internal/domains/cart/model"
)

type CartRepository interface {
    // Cart operations
    Create(ctx context.Context, cart *model.Cart) error
    FindByUserID(ctx context.Context, userID uuid.UUID) (*model.Cart, error)
    FindBySessionID(ctx context.Context, sessionID string) (*model.Cart, error)
    FindByID(ctx context.Context, cartID uuid.UUID) (*model.Cart, error)
    UpdateExpiresAt(ctx context.Context, cartID uuid.UUID, expiresAt time.Time) error
    Delete(ctx context.Context, cartID uuid.UUID) error
    DeleteExpired(ctx context.Context) error
    
    // Cart items operations
    AddItem(ctx context.Context, item *model.CartItem) error
    UpdateItemQuantity(ctx context.Context, cartID, bookID uuid.UUID, quantity int) error
    RemoveItem(ctx context.Context, cartID, bookID uuid.UUID) error
    ClearCart(ctx context.Context, cartID uuid.UUID) error
    GetItemsWithBooks(ctx context.Context, cartID uuid.UUID) ([]model.CartItem, error)
    
    // Stock validation
    CheckStock(ctx context.Context, bookID uuid.UUID, quantity int) (bool, int, error)
}
```

**☐ Task 4.6: Implement PostgreSQL Repository**

File: `internal/domains/cart/repository/postgres.go`

```go
package repository

import (
    "context"
    "errors"
    "time"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/google/uuid"
    "bookstore-backend/internal/domains/cart/model"
)

type postgresRepository struct {
    db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) CartRepository {
    return &postgresRepository{db: db}
}

func (r *postgresRepository) Create(ctx context.Context, cart *model.Cart) error {
    query := `
        INSERT INTO carts (user_id, session_id, expires_at)
        VALUES ($1, $2, $3)
        RETURNING id, created_at, updated_at
    `
    
    return r.db.QueryRow(ctx, query, cart.UserID, cart.SessionID, cart.ExpiresAt).
        Scan(&cart.ID, &cart.CreatedAt, &cart.UpdatedAt)
}

func (r *postgresRepository) FindByUserID(ctx context.Context, userID uuid.UUID) (*model.Cart, error) {
    query := `
        SELECT id, user_id, session_id, expires_at, created_at, updated_at
        FROM carts
        WHERE user_id = $1 AND expires_at > NOW()
        ORDER BY created_at DESC
        LIMIT 1
    `
    
    cart := &model.Cart{}
    err := r.db.QueryRow(ctx, query, userID).Scan(
        &cart.ID,
        &cart.UserID,
        &cart.SessionID,
        &cart.ExpiresAt,
        &cart.CreatedAt,
        &cart.UpdatedAt,
    )
    
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, nil
        }
        return nil, err
    }
    
    return cart, nil
}

func (r *postgresRepository) FindBySessionID(ctx context.Context, sessionID string) (*model.Cart, error) {
    query := `
        SELECT id, user_id, session_id, expires_at, created_at, updated_at
        FROM carts
        WHERE session_id = $1 AND expires_at > NOW()
        ORDER BY created_at DESC
        LIMIT 1
    `
    
    cart := &model.Cart{}
    err := r.db.QueryRow(ctx, query, sessionID).Scan(
        &cart.ID,
        &cart.UserID,
        &cart.SessionID,
        &cart.ExpiresAt,
        &cart.CreatedAt,
        &cart.UpdatedAt,
    )
    
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, nil
        }
        return nil, err
    }
    
    return cart, nil
}

func (r *postgresRepository) AddItem(ctx context.Context, item *model.CartItem) error {
    query := `
        INSERT INTO cart_items (cart_id, book_id, quantity)
        VALUES ($1, $2, $3)
        ON CONFLICT (cart_id, book_id) 
        DO UPDATE SET quantity = cart_items.quantity + EXCLUDED.quantity
        RETURNING added_at
    `
    
    return r.db.QueryRow(ctx, query, item.CartID, item.BookID, item.Quantity).
        Scan(&item.AddedAt)
}

func (r *postgresRepository) UpdateItemQuantity(ctx context.Context, cartID, bookID uuid.UUID, quantity int) error {
    if quantity == 0 {
        return r.RemoveItem(ctx, cartID, bookID)
    }
    
    query := `
        UPDATE cart_items
        SET quantity = $1
        WHERE cart_id = $2 AND book_id = $3
    `
    
    result, err := r.db.Exec(ctx, query, quantity, cartID, bookID)
    if err != nil {
        return err
    }
    
    if result.RowsAffected() == 0 {
        return errors.New("cart item not found")
    }
    
    return nil
}

func (r *postgresRepository) RemoveItem(ctx context.Context, cartID, bookID uuid.UUID) error {
    query := `DELETE FROM cart_items WHERE cart_id = $1 AND book_id = $2`
    
    result, err := r.db.Exec(ctx, query, cartID, bookID)
    if err != nil {
        return err
    }
    
    if result.RowsAffected() == 0 {
        return errors.New("cart item not found")
    }
    
    return nil
}

func (r *postgresRepository) GetItemsWithBooks(ctx context.Context, cartID uuid.UUID) ([]model.CartItem, error) {
    query := `
        SELECT 
            ci.cart_id, ci.book_id, ci.quantity, ci.added_at,
            b.id, b.title, b.slug, b.price, b.compare_at_price, b.cover_url, b.format,
            a.name as author_name,
            COALESCE(SUM(wi.quantity - wi.reserved), 0) as available_stock
        FROM cart_items ci
        JOIN books b ON ci.book_id = b.id
        JOIN authors a ON b.author_id = a.id
        LEFT JOIN warehouse_inventory wi ON b.id = wi.book_id
        WHERE ci.cart_id = $1 AND b.deleted_at IS NULL AND b.is_active = true
        GROUP BY ci.cart_id, ci.book_id, ci.quantity, ci.added_at,
                 b.id, b.title, b.slug, b.price, b.compare_at_price, b.cover_url, b.format,
                 a.name
        ORDER BY ci.added_at DESC
    `
    
    rows, err := r.db.Query(ctx, query, cartID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var items []model.CartItem
    for rows.Next() {
        var item model.CartItem
        var book model.CartBook
        
        err := rows.Scan(
            &item.CartID,
            &item.BookID,
            &item.Quantity,
            &item.AddedAt,
            &book.ID,
            &book.Title,
            &book.Slug,
            &book.Price,
            &book.CompareAtPrice,
            &book.CoverURL,
            &book.Format,
            &book.AuthorName,
            &book.Stock,
        )
        if err != nil {
            return nil, err
        }
        
        item.Book = &book
        items = append(items, item)
    }
    
    return items, nil
}

func (r *postgresRepository) CheckStock(ctx context.Context, bookID uuid.UUID, quantity int) (bool, int, error) {
    query := `
        SELECT COALESCE(SUM(quantity - reserved), 0)
        FROM warehouse_inventory
        WHERE book_id = $1
    `
    
    var availableStock int
    err := r.db.QueryRow(ctx, query, bookID).Scan(&availableStock)
    if err != nil {
        return false, 0, err
    }
    
    return availableStock >= quantity, availableStock, nil
}

func (r *postgresRepository) UpdateExpiresAt(ctx context.Context, cartID uuid.UUID, expiresAt time.Time) error {
    query := `UPDATE carts SET expires_at = $1 WHERE id = $2`
    _, err := r.db.Exec(ctx, query, expiresAt, cartID)
    return err
}

func (r *postgresRepository) ClearCart(ctx context.Context, cartID uuid.UUID) error {
    query := `DELETE FROM cart_items WHERE cart_id = $1`
    _, err := r.db.Exec(ctx, query, cartID)
    return err
}

func (r *postgresRepository) DeleteExpired(ctx context.Context) error {
    query := `DELETE FROM carts WHERE expires_at < NOW()`
    _, err := r.db.Exec(ctx, query)
    return err
}
```


### **Ngày 5-6: Cart Service với Business Logic**

**☐ Task 4.7: Create Cart Service**

File: `internal/domains/cart/service/cart_service.go`

```go
package service

import (
    "context"
    "errors"
    "time"
    "github.com/google/uuid"
    "bookstore-backend/internal/domains/cart/model"
    "bookstore-backend/internal/domains/cart/repository"
    "bookstore-backend/internal/domains/cart/dto"
)

const (
    CartExpirationDuration = 30 * time.Minute
    CartExpirationWarning  = 25 * time.Minute
)

type CartService struct {
    repo repository.CartRepository
}

func NewCartService(repo repository.CartRepository) *CartService {
    return &CartService{repo: repo}
}

// GetOrCreateCart - Get existing cart or create new one
func (s *CartService) GetOrCreateCart(ctx context.Context, userID *uuid.UUID, sessionID *string) (*model.Cart, error) {
    var cart *model.Cart
    var err error
    
    // Find existing cart
    if userID != nil {
        cart, err = s.repo.FindByUserID(ctx, *userID)
    } else if sessionID != nil {
        cart, err = s.repo.FindBySessionID(ctx, *sessionID)
    } else {
        return nil, errors.New("either userID or sessionID is required")
    }
    
    if err != nil {
        return nil, err
    }
    
    // Create new cart if not exists
    if cart == nil {
        cart = &model.Cart{
            UserID:    userID,
            SessionID: sessionID,
            ExpiresAt: time.Now().Add(CartExpirationDuration),
        }
        
        if err := s.repo.Create(ctx, cart); err != nil {
            return nil, err
        }
    }
    
    return cart, nil
}

// AddToCart - Add item to cart with stock validation
func (s *CartService) AddToCart(ctx context.Context, userID *uuid.UUID, sessionID *string, req dto.AddToCartRequest) (*dto.CartResponse, error) {
    // Validate request
    if err := req.Validate(); err != nil {
        return nil, err
    }
    
    bookID, err := uuid.Parse(req.BookID)
    if err != nil {
        return nil, errors.New("invalid book ID")
    }
    
    // Check stock availability
    hasStock, availableStock, err := s.repo.CheckStock(ctx, bookID, req.Quantity)
    if err != nil {
        return nil, err
    }
    
    if !hasStock {
        return nil, errors.New("insufficient stock")
    }
    
    // Get or create cart
    cart, err := s.GetOrCreateCart(ctx, userID, sessionID)
    if err != nil {
        return nil, err
    }
    
    // Add item to cart
    item := &model.CartItem{
        CartID:   cart.ID,
        BookID:   bookID,
        Quantity: req.Quantity,
    }
    
    if err := s.repo.AddItem(ctx, item); err != nil {
        return nil, err
    }
    
    // Update cart expiration
    newExpiration := time.Now().Add(CartExpirationDuration)
    if err := s.repo.UpdateExpiresAt(ctx, cart.ID, newExpiration); err != nil {
        return nil, err
    }
    
    // Return updated cart
    return s.GetCart(ctx, userID, sessionID)
}

// GetCart - Get cart with items and summary
func (s *CartService) GetCart(ctx context.Context, userID *uuid.UUID, sessionID *string) (*dto.CartResponse, error) {
    cart, err := s.GetOrCreateCart(ctx, userID, sessionID)
    if err != nil {
        return nil, err
    }
    
    // Get items with book details
    items, err := s.repo.GetItemsWithBooks(ctx, cart.ID)
    if err != nil {
        return nil, err
    }
    
    // Convert to response
    return s.buildCartResponse(cart, items), nil
}

// UpdateCartItem - Update item quantity
func (s *CartService) UpdateCartItem(ctx context.Context, userID *uuid.UUID, sessionID *string, bookID string, req dto.UpdateCartItemRequest) (*dto.CartResponse, error) {
    if err := req.Validate(); err != nil {
        return nil, err
    }
    
    bookUUID, err := uuid.Parse(bookID)
    if err != nil {
        return nil, errors.New("invalid book ID")
    }
    
    cart, err := s.GetOrCreateCart(ctx, userID, sessionID)
    if err != nil {
        return nil, err
    }
    
    // If quantity > 0, check stock
    if req.Quantity > 0 {
        hasStock, _, err := s.repo.CheckStock(ctx, bookUUID, req.Quantity)
        if err != nil {
            return nil, err
        }
        if !hasStock {
            return nil, errors.New("insufficient stock")
        }
    }
    
    // Update quantity
    if err := s.repo.UpdateItemQuantity(ctx, cart.ID, bookUUID, req.Quantity); err != nil {
        return nil, err
    }
    
    // Update expiration
    newExpiration := time.Now().Add(CartExpirationDuration)
    s.repo.UpdateExpiresAt(ctx, cart.ID, newExpiration)
    
    return s.GetCart(ctx, userID, sessionID)
}

// RemoveCartItem - Remove item from cart
func (s *CartService) RemoveCartItem(ctx context.Context, userID *uuid.UUID, sessionID *string, bookID string) (*dto.CartResponse, error) {
    bookUUID, err := uuid.Parse(bookID)
    if err != nil {
        return nil, errors.New("invalid book ID")
    }
    
    cart, err := s.GetOrCreateCart(ctx, userID, sessionID)
    if err != nil {
        return nil, err
    }
    
    if err := s.repo.RemoveItem(ctx, cart.ID, bookUUID); err != nil {
        return nil, err
    }
    
    return s.GetCart(ctx, userID, sessionID)
}

// ClearCart - Remove all items
func (s *CartService) ClearCart(ctx context.Context, userID *uuid.UUID, sessionID *string) error {
    cart, err := s.GetOrCreateCart(ctx, userID, sessionID)
    if err != nil {
        return err
    }
    
    return s.repo.ClearCart(ctx, cart.ID)
}

// buildCartResponse - Convert cart to response DTO
func (s *CartService) buildCartResponse(cart *model.Cart, items []model.CartItem) *dto.CartResponse {
    itemResponses := make([]dto.CartItemResponse, len(items))
    
    var totalQuantity int
    var subtotal float64
    
    for i, item := range items {
        itemSubtotal := item.Book.Price * float64(item.Quantity)
        
        itemResponses[i] = dto.CartItemResponse{
            BookID:     item.BookID.String(),
            Title:      item.Book.Title,
            Slug:       item.Book.Slug,
            AuthorName: item.Book.AuthorName,
            Price:      item.Book.Price,
            CoverURL:   *item.Book.CoverURL,
            Quantity:   item.Quantity,
            Subtotal:   itemSubtotal,
            Stock:      item.Book.Stock,
            Format:     item.Book.Format,
        }
        
        if item.Book.CompareAtPrice != nil {
            itemResponses[i].CompareAtPrice = *item.Book.CompareAtPrice
        }
        
        totalQuantity += item.Quantity
        subtotal += itemSubtotal
    }
    
    summary := dto.CartSummary{
        ItemCount:     len(items),
        TotalQuantity: totalQuantity,
        Subtotal:      subtotal,
        Discount:      0, // Will be calculated when promo applied
        ShippingFee:   0, // Will be calculated at checkout
        Total:         subtotal,
    }
    
    return &dto.CartResponse{
        ID:        cart.ID.String(),
        Items:     itemResponses,
        Summary:   summary,
        ExpiresAt: cart.ExpiresAt.Format(time.RFC3339),
    }
}
```


### **Ngày 7-8: Cart Handler \& Routes**

**☐ Task 4.8: Create Session Helper**

File: `pkg/session/session.go`

```go
package session

import (
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
)

const SessionCookieName = "cart_session"

// GetOrCreateSessionID - Get session ID from cookie or create new
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
            false, // secure (set true in production with HTTPS)
            true,  // httpOnly
        )
    }
    return sessionID
}
```

**☐ Task 4.9: Create Cart Handler**

File: `internal/domains/cart/handler/cart_handler.go`

```go
package handler

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "bookstore-backend/internal/domains/cart/service"
    "bookstore-backend/internal/domains/cart/dto"
    "bookstore-backend/pkg/session"
)

type CartHandler struct {
    cartService *service.CartService
}

func NewCartHandler(cartService *service.CartService) *CartHandler {
    return &CartHandler{cartService: cartService}
}

// getUserOrSession - Extract user ID or session ID from context
func (h *CartHandler) getUserOrSession(c *gin.Context) (*uuid.UUID, *string) {
    // Check if authenticated
    userIDStr, exists := c.Get("user_id")
    if exists {
        userID, _ := uuid.Parse(userIDStr.(string))
        return &userID, nil
    }
    
    // Guest user - use session
    sessionID := session.GetOrCreateSessionID(c)
    return nil, &sessionID
}

func (h *CartHandler) GetCart(c *gin.Context) {
    userID, sessionID := h.getUserOrSession(c)
    
    cart, err := h.cartService.GetCart(c.Request.Context(), userID, sessionID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "success": false,
            "error": gin.H{
                "code":    "SYS_001",
                "message": "Failed to fetch cart",
            },
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data":    cart,
    })
}

func (h *CartHandler) AddToCart(c *gin.Context) {
    var req dto.AddToCartRequest
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
    
    userID, sessionID := h.getUserOrSession(c)
    
    cart, err := h.cartService.AddToCart(c.Request.Context(), userID, sessionID, req)
    if err != nil {
        statusCode := http.StatusBadRequest
        errorCode := "BIZ_001"
        
        if err.Error() == "insufficient stock" {
            errorCode = "BIZ_001"
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
    
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data":    cart,
        "message": "Item added to cart",
    })
}

func (h *CartHandler) UpdateCartItem(c *gin.Context) {
    bookID := c.Param("book_id")
    
    var req dto.UpdateCartItemRequest
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
    
    userID, sessionID := h.getUserOrSession(c)
    
    cart, err := h.cartService.UpdateCartItem(c.Request.Context(), userID, sessionID, bookID, req)
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
    
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data":    cart,
        "message": "Cart updated",
    })
}

func (h *CartHandler) RemoveCartItem(c *gin.Context) {
    bookID := c.Param("book_id")
    
    userID, sessionID := h.getUserOrSession(c)
    
    cart, err := h.cartService.RemoveCartItem(c.Request.Context(), userID, sessionID, bookID)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "success": false,
            "error": gin.H{
                "code":    "RES_001",
                "message": err.Error(),
            },
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data":    cart,
        "message": "Item removed from cart",
    })
}

func (h *CartHandler) ClearCart(c *gin.Context) {
    userID, sessionID := h.getUserOrSession(c)
    
    if err := h.cartService.ClearCart(c.Request.Context(), userID, sessionID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "success": false,
            "error": gin.H{
                "code":    "SYS_001",
                "message": "Failed to clear cart",
            },
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "message": "Cart cleared",
    })
}
```

**☐ Task 4.10: Setup Cart Routes**

File: `cmd/api/main.go` (cập nhật)

```go
// ... trong main() ...

// Initialize cart repository & service
cartRepo := cartRepository.NewPostgresRepository(pgPool)
cartService := cartService.NewCartService(cartRepo)
cartHandler := cartHandler.NewCartHandler(cartService)

// Routes
v1 := r.Group("/v1")
{
    // ... existing routes ...
    
    // Cart routes (guest + authenticated)
    cart := v1.Group("/cart")
    {
        cart.GET("", cartHandler.GetCart)
        cart.POST("/items", cartHandler.AddToCart)
        cart.PATCH("/items/:book_id", cartHandler.UpdateCartItem)
        cart.DELETE("/items/:book_id", cartHandler.RemoveCartItem)
        cart.DELETE("", cartHandler.ClearCart)
    }
}
```


### **Ngày 9-10: Cart Testing \& Optimization**

**☐ Task 4.11: Create Unit Tests**

File: `internal/domains/cart/service/cart_service_test.go`

```go
package service_test

import (
    "context"
    "testing"
    "time"
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "bookstore-backend/internal/domains/cart/dto"
    "bookstore-backend/internal/domains/cart/model"
    "bookstore-backend/internal/domains/cart/service"
)

type MockCartRepository struct {
    mock.Mock
}

func (m *MockCartRepository) FindByUserID(ctx context.Context, userID uuid.UUID) (*model.Cart, error) {
    args := m.Called(ctx, userID)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*model.Cart), args.Error(1)
}

func (m *MockCartRepository) Create(ctx context.Context, cart *model.Cart) error {
    args := m.Called(ctx, cart)
    return args.Error(0)
}

func (m *MockCartRepository) CheckStock(ctx context.Context, bookID uuid.UUID, quantity int) (bool, int, error) {
    args := m.Called(ctx, bookID, quantity)
    return args.Bool(0), args.Int(1), args.Error(2)
}

func (m *MockCartRepository) AddItem(ctx context.Context, item *model.CartItem) error {
    args := m.Called(ctx, item)
    return args.Error(0)
}

// Implement other methods...

func TestCartService_AddToCart_Success(t *testing.T) {
    mockRepo := new(MockCartRepository)
    cartService := service.NewCartService(mockRepo)
    
    userID := uuid.New()
    bookID := uuid.New()
    
    req := dto.AddToCartRequest{
        BookID:   bookID.String(),
        Quantity: 2,
    }
    
    // Mock cart doesn't exist, will create new
    mockRepo.On("FindByUserID", mock.Anything, userID).Return(nil, nil)
    mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.Cart")).Return(nil)
    
    // Mock stock check - has stock
    mockRepo.On("CheckStock", mock.Anything, bookID, 2).Return(true, 10, nil)
    
    // Mock add item
    mockRepo.On("AddItem", mock.Anything, mock.AnythingOfType("*model.CartItem")).Return(nil)
    
    // Mock update expiration
    mockRepo.On("UpdateExpiresAt", mock.Anything, mock.Anything, mock.Anything).Return(nil)
    
    // Mock get items
    mockRepo.On("GetItemsWithBooks", mock.Anything, mock.Anything).Return([]model.CartItem{}, nil)
    
    cart, err := cartService.AddToCart(context.Background(), &userID, nil, req)
    
    assert.NoError(t, err)
    assert.NotNil(t, cart)
    mockRepo.AssertExpectations(t)
}

func TestCartService_AddToCart_OutOfStock(t *testing.T) {
    mockRepo := new(MockCartRepository)
    cartService := service.NewCartService(mockRepo)
    
    userID := uuid.New()
    bookID := uuid.New()
    
    req := dto.AddToCartRequest{
        BookID:   bookID.String(),
        Quantity: 10,
    }
    
    mockRepo.On("FindByUserID", mock.Anything, userID).Return(nil, nil)
    mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.Cart")).Return(nil)
    
    // Mock stock check - insufficient stock
    mockRepo.On("CheckStock", mock.Anything, bookID, 10).Return(false, 5, nil)
    
    _, err := cartService.AddToCart(context.Background(), &userID, nil, req)
    
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "insufficient stock")
}
```

**☐ Task 4.12: Test APIs với cURL**

```bash
# Guest - Add to cart
curl -X POST http://localhost:8080/v1/cart/items \
  -H "Content-Type: application/json" \
  -d '{
    "book_id": "uuid-here",
    "quantity": 2
  }' \
  -c cookies.txt

# Get cart (with cookies)
curl -X GET http://localhost:8080/v1/cart \
  -b cookies.txt

# Update quantity
curl -X PATCH http://localhost:8080/v1/cart/items/uuid-here \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"quantity": 3}'

# Remove item
curl -X DELETE http://localhost:8080/v1/cart/items/uuid-here \
  -b cookies.txt

# Authenticated user - Add to cart
curl -X POST http://localhost:8080/v1/cart/items \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "book_id": "uuid-here",
    "quantity": 1
  }'
```

**☐ Task 4.13: Add Redis Cache cho Cart**

File: `internal/infrastructure/cache/cart_cache.go`

```go
package cache

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
    "github.com/redis/go-redis/v9"
    "bookstore-backend/internal/domains/cart/dto"
)

type CartCache struct {
    redis *redis.Client
}

func NewCartCache(redis *redis.Client) *CartCache {
    return &CartCache{redis: redis}
}

func (c *CartCache) Get(ctx context.Context, cartID string) (*dto.CartResponse, error) {
    key := fmt.Sprintf("cart:%s", cartID)
    data, err := c.redis.Get(ctx, key).Result()
    if err != nil {
        return nil, err
    }
    
    var cart dto.CartResponse
    if err := json.Unmarshal([]byte(data), &cart); err != nil {
        return nil, err
    }
    
    return &cart, nil
}

func (c *CartCache) Set(ctx context.Context, cartID string, cart *dto.CartResponse) error {
    key := fmt.Sprintf("cart:%s", cartID)
    data, err := json.Marshal(cart)
    if err != nil {
        return err
    }
    
    return c.redis.Set(ctx, key, data, 30*time.Minute).Err()
}

func (c *CartCache) Delete(ctx context.Context, cartID string) error {
    key := fmt.Sprintf("cart:%s", cartID)
    return c.redis.Del(ctx, key).Err()
}
```

**☐ Task 4.14: Update Service để sử dụng Cache**

Thêm cache vào CartService:

```go
type CartService struct {
    repo  repository.CartRepository
    cache *cache.CartCache // Add this
}

func (s *CartService) GetCart(ctx context.Context, userID *uuid.UUID, sessionID *string) (*dto.CartResponse, error) {
    cart, err := s.GetOrCreateCart(ctx, userID, sessionID)
    if err != nil {
        return nil, err
    }
    
    // Try cache first
    if s.cache != nil {
        cached, err := s.cache.Get(ctx, cart.ID.String())
        if err == nil && cached != nil {
            return cached, nil
        }
    }
    
    // Get from DB
    items, err := s.repo.GetItemsWithBooks(ctx, cart.ID)
    if err != nil {
        return nil, err
    }
    
    response := s.buildCartResponse(cart, items)
    
    // Cache it
    if s.cache != nil {
        s.cache.Set(ctx, cart.ID.String(), response)
    }
    
    return response, nil
}
```


***

## **SPRINT 5: CHECKOUT \& ORDER MANAGEMENT (Tuần 9-10)**

### **Ngày 1-2: Order Database Schema**

**☐ Task 5.1: Create Addresses Table**

File: `migrations/000007_create_addresses.up.sql`

```sql
CREATE TABLE addresses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recipient_name TEXT NOT NULL,
    phone TEXT NOT NULL CHECK (phone ~ '^0[0-9]{9}$'),
    province TEXT NOT NULL,
    district TEXT NOT NULL,
    ward TEXT NOT NULL,
    street TEXT NOT NULL,
    address_type TEXT CHECK (address_type IN ('home', 'office')),
    is_default BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_addresses_user ON addresses(user_id);
CREATE UNIQUE INDEX idx_addresses_default ON addresses(user_id) 
    WHERE is_default = true;

CREATE TRIGGER addresses_updated_at BEFORE UPDATE ON addresses
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

**☐ Task 5.2: Create Warehouses Table**

File: `migrations/000008_create_warehouses.up.sql`

```sql
CREATE TABLE warehouses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    code TEXT UNIQUE NOT NULL,
    address TEXT NOT NULL,
    province TEXT NOT NULL,
    district TEXT,
    latitude DECIMAL(9,6),
    longitude DECIMAL(9,6),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_warehouses_active ON warehouses(is_active);
CREATE INDEX idx_warehouses_province ON warehouses(province);

-- Seed default warehouse
INSERT INTO warehouses (name, code, address, province, is_active) VALUES
('Kho Trung Tâm Hà Nội', 'HN001', 'Hoàng Mai, Hà Nội', 'Hà Nội', true),
('Kho Trung Tâm TP.HCM', 'HCM001', 'Quận 7, TP.HCM', 'TP.HCM', true);
```

**☐ Task 5.3: Create Warehouse Inventory Table**

File: `migrations/000009_create_warehouse_inventory.up.sql`

```sql
CREATE TABLE warehouse_inventory (
    warehouse_id UUID NOT NULL REFERENCES warehouses(id),
    book_id UUID NOT NULL REFERENCES books(id),
    quantity INT NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    reserved INT NOT NULL DEFAULT 0 CHECK (reserved >= 0),
    alert_threshold INT DEFAULT 10,
    last_restocked_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (warehouse_id, book_id),
    CONSTRAINT available_stock CHECK (quantity >= reserved)
);

CREATE INDEX idx_inventory_book ON warehouse_inventory(book_id);
CREATE INDEX idx_inventory_low_stock ON warehouse_inventory(warehouse_id, book_id) 
    WHERE quantity <= alert_threshold;

-- View for total stock
CREATE VIEW books_total_stock AS
SELECT 
    book_id,
    SUM(quantity) as total_quantity,
    SUM(reserved) as total_reserved,
    SUM(quantity - reserved) as available
FROM warehouse_inventory
GROUP BY book_id;

-- Seed inventory for sample books
INSERT INTO warehouse_inventory (warehouse_id, book_id, quantity)
SELECT 
    (SELECT id FROM warehouses WHERE code = 'HN001'),
    b.id,
    floor(random() * 100 + 20)::int
FROM books b
WHERE b.format != 'ebook';
```

**☐ Task 5.4: Create Orders Tables**

File: `migrations/000010_create_orders.up.sql`

```sql
-- Main orders table (partitioned by month)
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_number TEXT UNIQUE NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id),
    address_id UUID NOT NULL REFERENCES addresses(id),
    warehouse_id UUID REFERENCES warehouses(id),
    
    -- Pricing
    subtotal NUMERIC(10,2) NOT NULL CHECK (subtotal >= 0),
    discount_amount NUMERIC(10,2) DEFAULT 0 CHECK (discount_amount >= 0),
    shipping_fee NUMERIC(10,2) DEFAULT 0 CHECK (shipping_fee >= 0),
    tax_amount NUMERIC(10,2) DEFAULT 0 CHECK (tax_amount >= 0),
    total NUMERIC(10,2) NOT NULL CHECK (total >= 0),
    
    -- Status
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN (
        'pending', 'payment_failed', 'confirmed', 'processing', 
        'shipped', 'delivered', 'completed', 'cancelled', 'refunded'
    )),
    
    -- Payment
    payment_method TEXT NOT NULL CHECK (payment_method IN ('cod', 'vnpay', 'momo')),
    payment_status TEXT DEFAULT 'pending' CHECK (payment_status IN ('pending', 'paid', 'failed', 'refunded')),
    payment_id TEXT,
    
    -- Promotion
    promotion_id UUID,
    promotion_code TEXT,
    
    -- Shipping
    tracking_number TEXT,
    shipping_provider TEXT,
    
    -- Notes
    notes TEXT,
    cancelled_reason TEXT,
    
    -- Timestamps
    cancelled_at TIMESTAMPTZ,
    paid_at TIMESTAMPTZ,
    shipped_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
) PARTITION BY RANGE (created_at);

-- Create partitions for current and next months
CREATE TABLE orders_2025_10 PARTITION OF orders
    FOR VALUES FROM ('2025-10-01') TO ('2025-11-01');
    
CREATE TABLE orders_2025_11 PARTITION OF orders
    FOR VALUES FROM ('2025-11-01') TO ('2025-12-01');

CREATE INDEX idx_orders_user ON orders(user_id, created_at DESC);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_number ON orders(order_number);
CREATE INDEX idx_orders_payment_status ON orders(payment_status);

CREATE TRIGGER orders_updated_at BEFORE UPDATE ON orders
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Order items table
CREATE TABLE order_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    book_id UUID NOT NULL REFERENCES books(id),
    
    -- Snapshot data (preserve book info at time of purchase)
    book_title TEXT NOT NULL,
    book_author TEXT NOT NULL,
    book_isbn TEXT,
    book_format TEXT NOT NULL,
    
    quantity INT NOT NULL CHECK (quantity > 0),
    price_at_purchase NUMERIC(10,2) NOT NULL CHECK (price_at_purchase >= 0),
    subtotal NUMERIC(10,2) NOT NULL CHECK (subtotal >= 0),
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_order_items_order ON order_items(order_id);
CREATE INDEX idx_order_items_book ON order_items(book_id);

-- Order status history table
CREATE TABLE order_status_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    old_status TEXT,
    new_status TEXT NOT NULL,
    note TEXT,
    changed_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_status_history_order ON order_status_history(order_id, created_at);
```

**☐ Task 5.5: Create Order Number Generator Function**

File: `migrations/000011_order_number_function.up.sql`

```sql
CREATE OR REPLACE FUNCTION generate_order_number()
RETURNS TEXT AS $$
DECLARE
    new_number TEXT;
    counter INT;
BEGIN
    -- Get today's order count
    SELECT COUNT(*) + 1 INTO counter
    FROM orders
    WHERE DATE(created_at) = CURRENT_DATE;
    
    -- Format: ORD-YYYYMMDD-NNN
    new_number := 'ORD-' || TO_CHAR(CURRENT_DATE, 'YYYYMMDD') || '-' || LPAD(counter::TEXT, 4, '0');
    
    RETURN new_number;
END;
$$ LANGUAGE plpgsql;
```

**☐ Task 5.6: Run Migrations**

```bash
./scripts/migrate.sh up
```


### **Ngày 3-4: Order Models \& DTOs**

**☐ Task 5.7: Create Order Models**

File: `internal/domains/order/model/order.go`

```go
package model

import (
    "time"
    "github.com/google/uuid"
)

type Order struct {
    ID              uuid.UUID  `json:"id"`
    OrderNumber     string     `json:"order_number"`
    UserID          uuid.UUID  `json:"user_id"`
    AddressID       uuid.UUID  `json:"address_id"`
    WarehouseID     *uuid.UUID `json:"warehouse_id"`
    
    Subtotal        float64 `json:"subtotal"`
    DiscountAmount  float64 `json:"discount_amount"`
    ShippingFee     float64 `json:"shipping_fee"`
    TaxAmount       float64 `json:"tax_amount"`
    Total           float64 `json:"total"`
    
    Status          string `json:"status"`
    PaymentMethod   string `json:"payment_method"`
    PaymentStatus   string `json:"payment_status"`
    PaymentID       *string `json:"payment_id"`
    
    PromotionID     *uuid.UUID `json:"promotion_id"`
    PromotionCode   *string    `json:"promotion_code"`
    
    TrackingNumber  *string `json:"tracking_number"`
    ShippingProvider *string `json:"shipping_provider"`
    
    Notes           *string `json:"notes"`
    CancelledReason *string `json:"cancelled_reason"`
    
    CancelledAt *time.Time `json:"cancelled_at"`
    PaidAt      *time.Time `json:"paid_at"`
    ShippedAt   *time.Time `json:"shipped_at"`
    DeliveredAt *time.Time `json:"delivered_at"`
    CreatedAt   time.Time  `json:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at"`
    
    // Relationships
    Items   []OrderItem `json:"items,omitempty"`
    Address *Address    `json:"address,omitempty"`
}

type OrderItem struct {
    ID              uuid.UUID `json:"id"`
    OrderID         uuid.UUID `json:"order_id"`
    BookID          uuid.UUID `json:"book_id"`
    BookTitle       string    `json:"book_title"`
    BookAuthor      string    `json:"book_author"`
    BookISBN        *string   `json:"book_isbn"`
    BookFormat      string    `json:"book_format"`
    Quantity        int       `json:"quantity"`
    PriceAtPurchase float64   `json:"price_at_purchase"`
    Subtotal        float64   `json:"subtotal"`
    CreatedAt       time.Time `json:"created_at"`
}

type Address struct {
    ID            uuid.UUID `json:"id"`
    UserID        uuid.UUID `json:"user_id"`
    RecipientName string    `json:"recipient_name"`
    Phone         string    `json:"phone"`
    Province      string    `json:"province"`
    District      string    `json:"district"`
    Ward          string    `json:"ward"`
    Street        string    `json:"street"`
    AddressType   *string   `json:"address_type"`
    IsDefault     bool      `json:"is_default"`
}

type OrderStatusHistory struct {
    ID         uuid.UUID  `json:"id"`
    OrderID    uuid.UUID  `json:"order_id"`
    OldStatus  *string    `json:"old_status"`
    NewStatus  string     `json:"new_status"`
    Note       *string    `json:"note"`
    ChangedBy  *uuid.UUID `json:"changed_by"`
    CreatedAt  time.Time  `json:"created_at"`
}
```

**☐ Task 5.8: Create Order DTOs**

File: `internal/domains/order/dto/order.go`

```go
package dto

import (
    validation "github.com/go-ozzo/ozzo-validation/v4"
)

type CreateOrderRequest struct {
    AddressID     string  `json:"address_id"`
    PaymentMethod string  `json:"payment_method"`
    PromotionCode string  `json:"promotion_code,omitempty"`
    Notes         string  `json:"notes,omitempty"`
}

func (r CreateOrderRequest) Validate() error {
    return validation.ValidateStruct(&r,
        validation.Field(&r.AddressID, validation.Required),
        validation.Field(&r.PaymentMethod, validation.Required, validation.In("cod", "vnpay", "momo")),
    )
}

type OrderResponse struct {
    ID              string               `json:"id"`
    OrderNumber     string               `json:"order_number"`
    Status          string               `json:"status"`
    PaymentMethod   string               `json:"payment_method"`
    PaymentStatus   string               `json:"payment_status"`
    Subtotal        float64              `json:"subtotal"`
    DiscountAmount  float64              `json:"discount_amount"`
    ShippingFee     float64              `json:"shipping_fee"`
    Total           float64              `json:"total"`
    Items           []OrderItemResponse  `json:"items"`
    Address         AddressResponse      `json:"address"`
    TrackingNumber  string               `json:"tracking_number,omitempty"`
    CreatedAt       string               `json:"created_at"`
    PaidAt          string               `json:"paid_at,omitempty"`
    ShippedAt       string               `json:"shipped_at,omitempty"`
    DeliveredAt     string               `json:"delivered_at,omitempty"`
}

type OrderItemResponse struct {
    BookID          string  `json:"book_id"`
    BookTitle       string  `json:"book_title"`
    BookAuthor      string  `json:"book_author"`
    BookFormat      string  `json:"book_format"`
    CoverURL        string  `json:"cover_url,omitempty"`
    Quantity        int     `json:"quantity"`
    PriceAtPurchase float64 `json:"price_at_purchase"`
    Subtotal        float64 `json:"subtotal"`
}

type AddressResponse struct {
    RecipientName string `json:"recipient_name"`
    Phone         string `json:"phone"`
    Province      string `json:"province"`
    District      string `json:"district"`
    Ward          string `json:"ward"`
    Street        string `json:"street"`
    FullAddress   string `json:"full_address"`
}

type OrderListResponse struct {
    Orders []OrderResponse `json:"orders"`
    Meta   PaginationMeta  `json:"meta"`
}

type PaginationMeta struct {
    Page       int   `json:"page"`
    Limit      int   `json:"limit"`
    Total      int64 `json:"total"`
    TotalPages int   `json:"total_pages"`
}
```


### **Ngày 5-7: Order Repository \& Service**

**☐ Task 5.9: Create Address Repository**

File: `internal/domains/order/repository/address_repository.go`

```go
package repository

import (
    "context"
    "errors"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/google/uuid"
    "bookstore-backend/internal/domains/order/model"
)

type AddressRepository interface {
    Create(ctx context.Context, address *model.Address) error
    FindByID(ctx context.Context, id uuid.UUID) (*model.Address, error)
    FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Address, error)
    Update(ctx context.Context, address *model.Address) error
    Delete(ctx context.Context, id uuid.UUID) error
    SetDefault(ctx context.Context, userID, addressID uuid.UUID) error
}

type addressRepository struct {
    db *pgxpool.Pool
}

func NewAddressRepository(db *pgxpool.Pool) AddressRepository {
    return &addressRepository{db: db}
}

func (r *addressRepository) Create(ctx context.Context, address *model.Address) error {
    tx, err := r.db.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx)
    
    // If this is default, unset other defaults
    if address.IsDefault {
        _, err = tx.Exec(ctx, 
            "UPDATE addresses SET is_default = false WHERE user_id = $1",
            address.UserID,
        )
        if err != nil {
            return err
        }
    }
    
    query := `
        INSERT INTO addresses (user_id, recipient_name, phone, province, district, ward, street, address_type, is_default)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        RETURNING id, created_at, updated_at
    `
    
    err = tx.QueryRow(ctx, query,
        address.UserID,
        address.RecipientName,
        address.Phone,
        address.Province,
        address.District,
        address.Ward,
        address.Street,
        address.AddressType,
        address.IsDefault,
    ).Scan(&address.ID, &address.CreatedAt, &address.UpdatedAt)
    
    if err != nil {
        return err
    }
    
    return tx.Commit(ctx)
}

func (r *addressRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Address, error) {
    query := `
        SELECT id, user_id, recipient_name, phone, province, district, ward, 
               street, address_type, is_default, created_at, updated_at
        FROM addresses
        WHERE id = $1
    `
    
    address := &model.Address{}
    err := r.db.QueryRow(ctx, query, id).Scan(
        &address.ID,
        &address.UserID,
        &address.RecipientName,
        &address.Phone,
        &address.Province,
        &address.District,
        &address.Ward,
        &address.Street,
        &address.AddressType,
        &address.IsDefault,
        &address.CreatedAt,
        &address.UpdatedAt,
    )
    
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, errors.New("address not found")
        }
        return nil, err
    }
    
    return address, nil
}

func (r *addressRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Address, error) {
    query := `
        SELECT id, user_id, recipient_name, phone, province, district, ward, 
               street, address_type, is_default, created_at, updated_at
        FROM addresses
        WHERE user_id = $1
        ORDER BY is_default DESC, created_at DESC
    `
    
    rows, err := r.db.Query(ctx, query, userID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var addresses []model.Address
    for rows.Next() {
        var address model.Address
        err := rows.Scan(
            &address.ID,
            &address.UserID,
            &address.RecipientName,
            &address.Phone,
            &address.Province,
            &address.District,
            &address.Ward,
            &address.Street,
            &address.AddressType,
            &address.IsDefault,
            &address.CreatedAt,
            &address.UpdatedAt,
        )
        if err != nil {
            return nil, err
        }
        addresses = append(addresses, address)
    }
    
    return addresses, nil
}
```

**☐ Task 5.10: Create Order Repository** (tiếp tục trong phần 2)

Tôi đã cung cấp chi tiết cho **Sprint 4 và nửa đầu Sprint 5**. Bạn có muốn tôi:

1. **Tiếp tục Sprint 5** (Order Repository, Service, Handler)
2. **Xuất toàn bộ nội dung hiện tại** thành file để lưu trữ
3. **Nhảy sang các sprint tiếp theo** (Payment, Admin, Background Jobs)

Hãy cho tôi biết bạn muốn tiếp tục theo hướng nào nhé!

