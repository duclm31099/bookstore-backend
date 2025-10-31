<img src="https://r2cdn.perplexity.ai/pplx-full-logo-primary-dark%402x.png" style="height:64px;margin-right:32px"/>

# TODO LIST CHI TIẾT CHO BACKEND DEVELOPER - SPRINT 11-12: PROMOTIONS \& REVIEWS

Dựa trên URD, dưới đây là danh sách công việc chi tiết và đầy đủ cho backend developer trong Sprint 11-12 (Phase 2, 2 tuần - 10 ngày làm việc).[^1]

## 1. Promotions Table + Seed Data (P2-T009)

### Mô tả

Tạo database schema cho hệ thống promotions và seed data mẫu để test.[^1]

### Database Schema

#### 1.1 Promotions Table Migration

Tạo file `migrations/000021_create_promotions_table.up.sql`:[^1]

```sql
CREATE TABLE promotions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT UNIQUE NOT NULL,
    description TEXT,
    
    -- Discount type
    type TEXT NOT NULL CHECK (type IN ('percentage', 'fixed')),
    value NUMERIC(10,2) NOT NULL CHECK (value > 0),
    
    -- Limits
    max_discount_amount NUMERIC(10,2), -- For percentage type
    min_order_amount NUMERIC(10,2) DEFAULT 0,
    
    -- Usage constraints
    max_usage INT, -- NULL = unlimited
    max_usage_per_user INT DEFAULT 1,
    used_count INT DEFAULT 0,
    
    -- Category restriction (NULL = all categories)
    category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    
    -- Time validity
    start_at TIMESTAMPTZ NOT NULL,
    end_at TIMESTAMPTZ NOT NULL,
    
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT valid_dates CHECK (end_at > start_at),
    CONSTRAINT valid_usage CHECK (max_usage IS NULL OR max_usage > 0)
);

-- Indexes
CREATE UNIQUE INDEX idx_promotions_code ON promotions(code) WHERE is_active = true;
CREATE INDEX idx_promotions_dates ON promotions(start_at, end_at);
CREATE INDEX idx_promotions_category ON promotions(category_id) WHERE category_id IS NOT NULL;
CREATE INDEX idx_promotions_active ON promotions(is_active, start_at, end_at) 
    WHERE is_active = true;

-- Trigger auto update updated_at
CREATE TRIGGER update_promotions_updated_at
    BEFORE UPDATE ON promotions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```


#### 1.2 Promotion Usage Tracking Table

```sql
CREATE TABLE promotion_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    promotion_id UUID NOT NULL REFERENCES promotions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    discount_amount NUMERIC(10,2) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE (promotion_id, order_id)
);

-- Indexes
CREATE INDEX idx_promotion_usage_promo ON promotion_usage(promotion_id);
CREATE INDEX idx_promotion_usage_user ON promotion_usage(user_id, promotion_id);
CREATE INDEX idx_promotion_usage_order ON promotion_usage(order_id);
```


#### 1.3 Update Orders Table

```sql
-- Add promotion_id to orders table (if not exists)
ALTER TABLE orders ADD COLUMN IF NOT EXISTS promotion_id UUID REFERENCES promotions(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_orders_promotion ON orders(promotion_id) WHERE promotion_id IS NOT NULL;
```


### Công việc cụ thể

#### 1.4 Seed Data Script

Tạo file `seeds/002_promotions_seed.sql`:[^1]

```sql
-- Percentage discount promotions
INSERT INTO promotions (code, description, type, value, max_discount_amount, min_order_amount, max_usage, max_usage_per_user, start_at, end_at, is_active)
VALUES
-- 20% off, max 100k discount, min order 200k
('SUMMER20', 'Giảm 20% cho đơn hàng từ 200k', 'percentage', 20.00, 100000, 200000, 1000, 1, 
 '2025-11-01 00:00:00+07', '2025-12-31 23:59:59+07', true),

-- 10% off for new users
('NEWUSER10', 'Giảm 10% cho khách hàng mới', 'percentage', 10.00, 50000, 100000, NULL, 1,
 '2025-11-01 00:00:00+07', '2026-12-31 23:59:59+07', true),

-- Flash sale 50% limited to 100 uses
('FLASH50', 'Flash Sale 50% - Giới hạn 100 suất', 'percentage', 50.00, 200000, 300000, 100, 1,
 '2025-11-15 10:00:00+07', '2025-11-15 12:00:00+07', true);

-- Fixed amount discount promotions
INSERT INTO promotions (code, description, type, value, min_order_amount, max_usage, max_usage_per_user, start_at, end_at, is_active)
VALUES
-- Free shipping
('FREESHIP', 'Miễn phí vận chuyển cho đơn từ 150k', 'fixed', 30000, 150000, NULL, 3,
 '2025-11-01 00:00:00+07', '2025-12-31 23:59:59+07', true),

-- 50k off
('SAVE50K', 'Giảm ngay 50,000đ cho đơn từ 500k', 'fixed', 50000, 500000, 500, 1,
 '2025-11-01 00:00:00+07', '2025-11-30 23:59:59+07', true),

-- VIP customer 100k off
('VIP100K', 'Giảm 100,000đ cho khách hàng VIP', 'fixed', 100000, 1000000, 200, 2,
 '2025-11-01 00:00:00+07', '2025-12-31 23:59:59+07', true);

-- Category-specific promotion (Literature category)
INSERT INTO promotions (code, description, type, value, max_discount_amount, min_order_amount, category_id, max_usage, max_usage_per_user, start_at, end_at, is_active)
SELECT 
    'BOOK15',
    'Giảm 15% cho sách Văn học',
    'percentage',
    15.00,
    80000,
    150000,
    id, -- category_id from categories table
    NULL,
    2,
    '2025-11-01 00:00:00+07',
    '2025-12-31 23:59:59+07',
    true
FROM categories 
WHERE slug = 'van-hoc'
LIMIT 1;
```


#### 1.5 Domain Model

Tạo file `internal/domains/promotion/model/promotion.go`:[^1]

```go
package model

import "time"

type PromotionType string

const (
    PromotionTypePercentage PromotionType = "percentage"
    PromotionTypeFixed      PromotionType = "fixed"
)

type Promotion struct {
    ID                 string         `json:"id"`
    Code               string         `json:"code"`
    Description        string         `json:"description"`
    Type               PromotionType  `json:"type"`
    Value              float64        `json:"value"`
    MaxDiscountAmount  *float64       `json:"max_discount_amount,omitempty"`
    MinOrderAmount     float64        `json:"min_order_amount"`
    MaxUsage           *int           `json:"max_usage,omitempty"`
    MaxUsagePerUser    int            `json:"max_usage_per_user"`
    UsedCount          int            `json:"used_count"`
    CategoryID         *string        `json:"category_id,omitempty"`
    StartAt            time.Time      `json:"start_at"`
    EndAt              time.Time      `json:"end_at"`
    IsActive           bool           `json:"is_active"`
    CreatedAt          time.Time      `json:"created_at"`
    UpdatedAt          time.Time      `json:"updated_at"`
}

type PromotionUsage struct {
    ID             string    `json:"id"`
    PromotionID    string    `json:"promotion_id"`
    UserID         string    `json:"user_id"`
    OrderID        string    `json:"order_id"`
    DiscountAmount float64   `json:"discount_amount"`
    CreatedAt      time.Time `json:"created_at"`
}
```


### Acceptance Criteria

- Migration chạy thành công tạo bảng promotions và promotion_usage[^1]
- Seed data insert được 7-10 promotions mẫu[^1]
- Indexes được tạo đúng để optimize queries[^1]
- Constraints validate data integrity (valid dates, positive values)[^1]


### Dependencies

- P1-T002: Database setup[^1]
- P1-T003: Core tables (categories, orders)[^1]


### Effort

1 ngày[^1]

***

## 2. Apply Promo Code to Cart API (P2-T010)

### Mô tả

API để user áp dụng mã giảm giá vào giỏ hàng và tính toán discount.[^1]

### API Endpoint

`POST /v1/cart/promo`[^1]

### Request Body

```json
{
  "promo_code": "SUMMER20"
}
```


### Response Format

```json
{
  "success": true,
  "data": {
    "promo_code": "SUMMER20",
    "discount_type": "percentage",
    "discount_value": 20.00,
    "discount_amount": 80000,
    "min_order_amount": 200000,
    "cart_summary": {
      "subtotal": 400000,
      "discount": 80000,
      "shipping_fee": 30000,
      "total": 350000
    }
  }
}
```


### Công việc cụ thể

#### 2.1 Cart Service - Apply Promotion

Tạo file `internal/domains/cart/service/promotion_service.go`:[^1]

```go
package service

import (
    "context"
    "fmt"
    "time"
)

type ApplyPromoResult struct {
    PromoCode        string  `json:"promo_code"`
    DiscountType     string  `json:"discount_type"`
    DiscountValue    float64 `json:"discount_value"`
    DiscountAmount   float64 `json:"discount_amount"`
    MinOrderAmount   float64 `json:"min_order_amount"`
    CartSummary      CartSummary `json:"cart_summary"`
}

type CartSummary struct {
    Subtotal    float64 `json:"subtotal"`
    Discount    float64 `json:"discount"`
    ShippingFee float64 `json:"shipping_fee"`
    Total       float64 `json:"total"`
}

func (s *CartService) ApplyPromotion(ctx context.Context, userID string, promoCode string) (*ApplyPromoResult, error) {
    // 1. Get cart
    cart, err := s.cartRepo.FindByUserID(ctx, userID)
    if err != nil {
        return nil, fmt.Errorf("cart not found")
    }
    
    if len(cart.Items) == 0 {
        return nil, fmt.Errorf("cart is empty")
    }
    
    // 2. Validate promotion code
    promo, err := s.promoRepo.FindByCode(ctx, promoCode)
    if err != nil {
        return nil, fmt.Errorf("invalid promo code")
    }
    
    // 3. Validate promotion rules
    validationErr := s.validatePromotion(ctx, promo, cart, userID)
    if validationErr != nil {
        return nil, validationErr
    }
    
    // 4. Calculate cart subtotal
    subtotal := s.calculateSubtotal(cart)
    
    // 5. Calculate discount amount
    discountAmount := s.calculateDiscount(promo, subtotal)
    
    // 6. Calculate final total
    shippingFee := 30000.0 // Default shipping fee
    total := subtotal - discountAmount + shippingFee
    
    if total < 0 {
        total = 0
    }
    
    // 7. Update cart with promotion
    cart.PromotionID = &promo.ID
    cart.DiscountAmount = discountAmount
    err = s.cartRepo.Update(ctx, cart)
    if err != nil {
        return nil, err
    }
    
    return &ApplyPromoResult{
        PromoCode:      promo.Code,
        DiscountType:   string(promo.Type),
        DiscountValue:  promo.Value,
        DiscountAmount: discountAmount,
        MinOrderAmount: promo.MinOrderAmount,
        CartSummary: CartSummary{
            Subtotal:    subtotal,
            Discount:    discountAmount,
            ShippingFee: shippingFee,
            Total:       total,
        },
    }, nil
}

func (s *CartService) calculateSubtotal(cart *Cart) float64 {
    var subtotal float64
    for _, item := range cart.Items {
        subtotal += item.Price * float64(item.Quantity)
    }
    return subtotal
}

func (s *CartService) calculateDiscount(promo *Promotion, subtotal float64) float64 {
    var discount float64
    
    if promo.Type == PromotionTypePercentage {
        discount = subtotal * (promo.Value / 100.0)
        
        // Apply max discount limit
        if promo.MaxDiscountAmount != nil && discount > *promo.MaxDiscountAmount {
            discount = *promo.MaxDiscountAmount
        }
    } else if promo.Type == PromotionTypeFixed {
        discount = promo.Value
    }
    
    // Discount cannot exceed subtotal
    if discount > subtotal {
        discount = subtotal
    }
    
    return discount
}
```


#### 2.2 Validation Logic

```go
func (s *CartService) validatePromotion(ctx context.Context, promo *Promotion, cart *Cart, userID string) error {
    // 1. Check if promotion is active
    if !promo.IsActive {
        return fmt.Errorf("promotion is not active")
    }
    
    // 2. Check date validity
    now := time.Now()
    if now.Before(promo.StartAt) {
        return fmt.Errorf("promotion has not started yet")
    }
    if now.After(promo.EndAt) {
        return fmt.Errorf("promotion has expired")
    }
    
    // 3. Check minimum order amount
    subtotal := s.calculateSubtotal(cart)
    if subtotal < promo.MinOrderAmount {
        return fmt.Errorf("minimum order amount is %.0f VND", promo.MinOrderAmount)
    }
    
    // 4. Check max usage limit
    if promo.MaxUsage != nil && promo.UsedCount >= *promo.MaxUsage {
        return fmt.Errorf("promotion has reached maximum usage limit")
    }
    
    // 5. Check per-user usage limit (will implement in P2-T011)
    // This validation is in validatePromoUsageLimit service
    
    // 6. Check category restriction
    if promo.CategoryID != nil {
        hasValidCategory := false
        for _, item := range cart.Items {
            if item.Book.CategoryID == *promo.CategoryID {
                hasValidCategory = true
                break
            }
        }
        if !hasValidCategory {
            return fmt.Errorf("promotion only applies to specific category")
        }
    }
    
    return nil
}
```


#### 2.3 Remove Promotion API

`DELETE /v1/cart/promo`[^1]

```go
func (s *CartService) RemovePromotion(ctx context.Context, userID string) error {
    cart, err := s.cartRepo.FindByUserID(ctx, userID)
    if err != nil {
        return err
    }
    
    cart.PromotionID = nil
    cart.DiscountAmount = 0
    
    return s.cartRepo.Update(ctx, cart)
}
```


#### 2.4 Update Cart Schema

```sql
-- Add promotion fields to carts table
ALTER TABLE carts 
ADD COLUMN IF NOT EXISTS promotion_id UUID REFERENCES promotions(id) ON DELETE SET NULL,
ADD COLUMN IF NOT EXISTS discount_amount NUMERIC(10,2) DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_carts_promotion ON carts(promotion_id) WHERE promotion_id IS NOT NULL;
```


#### 2.5 Handler Implementation

```go
func (h *CartHandler) ApplyPromo(c *gin.Context) {
    var req struct {
        PromoCode string `json:"promo_code" binding:"required"`
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"success": false, "error": "Invalid request"})
        return
    }
    
    userID := c.GetString("user_id")
    
    result, err := h.cartService.ApplyPromotion(c.Request.Context(), userID, req.PromoCode)
    if err != nil {
        c.JSON(400, gin.H{"success": false, "error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{"success": true, "data": result})
}

func (h *CartHandler) RemovePromo(c *gin.Context) {
    userID := c.GetString("user_id")
    
    err := h.cartService.RemovePromotion(c.Request.Context(), userID)
    if err != nil {
        c.JSON(500, gin.H{"success": false, "error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{"success": true, "message": "Promotion removed"})
}
```


### Acceptance Criteria

- User apply được promo code hợp lệ vào cart[^1]
- Discount amount tính đúng theo type (percentage/fixed)[^1]
- Validate min order amount[^1]
- Validate promotion active status và date range[^1]
- User xóa được promo code khỏi cart[^1]
- Response hiển thị discount breakdown rõ ràng[^1]


### Dependencies

- P2-T009: Promotions table[^1]
- P1-T020: Cart APIs[^1]


### Effort

2 ngày[^1]

***

## 3. Validate Promo Usage Limits (P2-T011)

### Mô tả

Implement logic validate usage limits của promotion (per-user limit, total usage).[^1]

### Validation Rules

1. **Max usage per user**: User chỉ được dùng promo code X lần[^1]
2. **Total max usage**: Promo code chỉ có Y lượt sử dụng trong toàn hệ thống[^1]
3. **Concurrent usage**: Race condition prevention khi nhiều users dùng promo cùng lúc[^1]

### Công việc cụ thể

#### 3.1 Promotion Usage Validation Service

Tạo file `internal/domains/promotion/service/validation_service.go`:[^1]

```go
package service

import (
    "context"
    "fmt"
)

type ValidationService struct {
    promoRepo      *repository.PromotionRepository
    promoUsageRepo *repository.PromotionUsageRepository
}

func (s *ValidationService) ValidatePromoUsageLimit(ctx context.Context, promoID string, userID string) error {
    // 1. Get promotion
    promo, err := s.promoRepo.FindByID(ctx, promoID)
    if err != nil {
        return err
    }
    
    // 2. Check per-user usage limit
    userUsageCount, err := s.promoUsageRepo.CountByUserAndPromo(ctx, userID, promoID)
    if err != nil {
        return err
    }
    
    if userUsageCount >= promo.MaxUsagePerUser {
        return fmt.Errorf("you have already used this promo code %d time(s). Maximum usage per user is %d", 
            userUsageCount, promo.MaxUsagePerUser)
    }
    
    // 3. Check total usage limit (with atomic check)
    if promo.MaxUsage != nil {
        if promo.UsedCount >= *promo.MaxUsage {
            return fmt.Errorf("promo code has reached maximum usage limit")
        }
    }
    
    return nil
}
```


#### 3.2 Repository - Count User Usage

```go
func (r *PromotionUsageRepository) CountByUserAndPromo(ctx context.Context, userID string, promoID string) (int, error) {
    var count int
    query := `
        SELECT COUNT(*) 
        FROM promotion_usage 
        WHERE user_id = $1 AND promotion_id = $2
    `
    err := r.db.QueryRowContext(ctx, query, userID, promoID).Scan(&count)
    return count, err
}
```


#### 3.3 Atomic Usage Increment (Race Condition Prevention)

Sử dụng database transaction và pessimistic locking:[^1]

```go
func (s *OrderService) CreateOrderWithPromotion(ctx context.Context, params CreateOrderParams) (*Order, error) {
    // Start transaction
    tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
    if err != nil {
        return nil, err
    }
    defer tx.Rollback()
    
    if params.PromotionID != nil {
        // 1. Lock promotion row for update (pessimistic lock)
        promo, err := s.promoRepo.FindByIDForUpdateTx(ctx, tx, *params.PromotionID)
        if err != nil {
            return nil, err
        }
        
        // 2. Re-validate usage limits within transaction
        err = s.validationService.ValidatePromoUsageLimitTx(ctx, tx, promo.ID, params.UserID)
        if err != nil {
            return nil, err
        }
        
        // 3. Check total usage with locked row
        if promo.MaxUsage != nil && promo.UsedCount >= *promo.MaxUsage {
            return nil, fmt.Errorf("promotion usage limit exceeded")
        }
        
        // 4. Create order with promotion
        order := &Order{
            // ... order fields
            PromotionID:    params.PromotionID,
            DiscountAmount: params.DiscountAmount,
        }
        
        err = s.orderRepo.CreateTx(ctx, tx, order)
        if err != nil {
            return nil, err
        }
        
        // 5. Record promotion usage
        usage := &PromotionUsage{
            PromotionID:    promo.ID,
            UserID:         params.UserID,
            OrderID:        order.ID,
            DiscountAmount: params.DiscountAmount,
        }
        
        err = s.promoUsageRepo.CreateTx(ctx, tx, usage)
        if err != nil {
            return nil, err
        }
        
        // 6. Increment used_count atomically
        err = s.promoRepo.IncrementUsedCountTx(ctx, tx, promo.ID)
        if err != nil {
            return nil, err
        }
    }
    
    // Commit transaction
    if err := tx.Commit(); err != nil {
        return nil, err
    }
    
    return order, nil
}
```


#### 3.4 Repository - Lock Promotion Row

```go
func (r *PromotionRepository) FindByIDForUpdateTx(ctx context.Context, tx *sql.Tx, id string) (*Promotion, error) {
    var promo Promotion
    query := `
        SELECT * FROM promotions 
        WHERE id = $1 
        FOR UPDATE  -- Pessimistic lock
    `
    err := tx.QueryRowContext(ctx, query, id).Scan(&promo)
    return &promo, err
}

func (r *PromotionRepository) IncrementUsedCountTx(ctx context.Context, tx *sql.Tx, id string) error {
    query := `
        UPDATE promotions 
        SET used_count = used_count + 1 
        WHERE id = $1
    `
    _, err := tx.ExecContext(ctx, query, id)
    return err
}
```


#### 3.5 Integration với Cart Apply Promo

Update `ApplyPromotion` service từ P2-T010:

```go
func (s *CartService) ApplyPromotion(ctx context.Context, userID string, promoCode string) (*ApplyPromoResult, error) {
    // ... existing validation
    
    // ADD: Validate usage limits
    err = s.validationService.ValidatePromoUsageLimit(ctx, promo.ID, userID)
    if err != nil {
        return nil, err
    }
    
    // ... rest of the code
}
```


### Acceptance Criteria

- User không thể dùng promo vượt quá max_usage_per_user[^1]
- Promo code stop working khi đạt max_usage total[^1]
- Không có race condition khi nhiều users dùng promo cùng lúc (stress test)[^1]
- Promotion usage được ghi log đầy đủ trong promotion_usage table[^1]
- Transaction rollback đúng khi có lỗi[^1]


### Dependencies

- P2-T009: Promotions table[^1]
- P2-T010: Apply promo API[^1]
- P1-T025: Order creation API[^1]


### Effort

1 ngày[^1]

***

## 4. Admin: CRUD Promotions (P2-T012)

### Mô tả

Admin panel APIs để quản lý promotions (Create, Read, Update, Delete).[^1]

### API Endpoints

- `GET /v1/admin/promotions` - List promotions[^1]
- `GET /v1/admin/promotions/:id` - Get promotion detail[^1]
- `POST /v1/admin/promotions` - Create promotion[^1]
- `PUT /v1/admin/promotions/:id` - Update promotion[^1]
- `DELETE /v1/admin/promotions/:id` - Soft delete promotion[^1]
- `GET /v1/admin/promotions/:id/usage` - View usage statistics[^1]


### Công việc cụ thể

#### 4.1 Create Promotion API

**Request Body**:

```json
{
  "code": "BLACKFRIDAY50",
  "description": "Black Friday 50% off",
  "type": "percentage",
  "value": 50.00,
  "max_discount_amount": 500000,
  "min_order_amount": 300000,
  "max_usage": 500,
  "max_usage_per_user": 1,
  "category_id": "uuid-optional",
  "start_at": "2025-11-24T00:00:00+07:00",
  "end_at": "2025-11-25T23:59:59+07:00"
}
```

**Handler**:

```go
func (h *AdminPromotionHandler) CreatePromotion(c *gin.Context) {
    var req CreatePromotionRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, errors.NewValidationError(err.Error()))
        return
    }
    
    // Validate
    if err := req.Validate(); err != nil {
        c.JSON(400, errors.NewValidationError(err.Error()))
        return
    }
    
    // Create
    promo, err := h.promoService.CreatePromotion(c.Request.Context(), req)
    if err != nil {
        c.JSON(500, errors.NewInternalError(err.Error()))
        return
    }
    
    c.JSON(201, gin.H{"success": true, "data": promo})
}
```

**Validation**:

```go
func (r CreatePromotionRequest) Validate() error {
    return validation.ValidateStruct(&r,
        validation.Field(&r.Code, 
            validation.Required, 
            validation.Length(3, 50),
            validation.Match(regexp.MustCompile(`^[A-Z0-9]+$`))),
        validation.Field(&r.Type, 
            validation.Required,
            validation.In("percentage", "fixed")),
        validation.Field(&r.Value, 
            validation.Required, 
            validation.Min(0.01)),
        validation.Field(&r.MinOrderAmount, 
            validation.Min(0.0)),
        validation.Field(&r.MaxUsagePerUser, 
            validation.Required,
            validation.Min(1)),
        validation.Field(&r.StartAt, 
            validation.Required),
        validation.Field(&r.EndAt, 
            validation.Required,
            validation.By(func(value interface{}) error {
                if r.EndAt.Before(r.StartAt) {
                    return fmt.Errorf("end_at must be after start_at")
                }
                return nil
            })),
    )
}
```


#### 4.2 Update Promotion API

```go
func (h *AdminPromotionHandler) UpdatePromotion(c *gin.Context) {
    promoID := c.Param("id")
    
    var req UpdatePromotionRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, errors.NewValidationError(err.Error()))
        return
    }
    
    // Check if promotion exists
    existingPromo, err := h.promoService.GetByID(c.Request.Context(), promoID)
    if err != nil {
        c.JSON(404, errors.NewNotFoundError("Promotion not found"))
        return
    }
    
    // Validate: Cannot change code if already used
    if existingPromo.UsedCount > 0 && req.Code != existingPromo.Code {
        c.JSON(400, gin.H{
            "success": false,
            "error": "Cannot change code for promotion that has been used"
        })
        return
    }
    
    // Update
    promo, err := h.promoService.UpdatePromotion(c.Request.Context(), promoID, req)
    if err != nil {
        c.JSON(500, errors.NewInternalError(err.Error()))
        return
    }
    
    c.JSON(200, gin.H{"success": true, "data": promo})
}
```


#### 4.3 List Promotions API

**Query Parameters**:

- `?page=1&limit=20` - Pagination[^1]
- `?type=percentage` - Filter by type[^1]
- `?is_active=true` - Filter by active status[^1]
- `?status=active|upcoming|expired` - Filter by time status[^1]
- `?sort=created_at:desc` - Sorting[^1]

```go
func (h *AdminPromotionHandler) ListPromotions(c *gin.Context) {
    // Parse query params
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
    typeFilter := c.Query("type")
    isActive := c.Query("is_active")
    status := c.Query("status") // active, upcoming, expired
    
    filters := PromotionFilters{
        Page:     page,
        Limit:    limit,
        Type:     typeFilter,
        IsActive: isActive,
        Status:   status,
    }
    
    result, err := h.promoService.ListPromotions(c.Request.Context(), filters)
    if err != nil {
        c.JSON(500, errors.NewInternalError(err.Error()))
        return
    }
    
    c.JSON(200, gin.H{
        "success": true,
        "data": result.Promotions,
        "meta": gin.H{
            "page":  page,
            "limit": limit,
            "total": result.Total,
        },
    })
}
```

**Service Implementation**:

```go
func (s *PromotionService) ListPromotions(ctx context.Context, filters PromotionFilters) (*PromotionListResult, error) {
    // Build query with filters
    query := `
        SELECT p.*, c.name as category_name
        FROM promotions p
        LEFT JOIN categories c ON p.category_id = c.id
        WHERE 1=1
    `
    args := []interface{}{}
    argPos := 1
    
    // Apply filters
    if filters.Type != "" {
        query += fmt.Sprintf(" AND p.type = $%d", argPos)
        args = append(args, filters.Type)
        argPos++
    }
    
    if filters.IsActive != "" {
        query += fmt.Sprintf(" AND p.is_active = $%d", argPos)
        args = append(args, filters.IsActive == "true")
        argPos++
    }
    
    if filters.Status == "active" {
        query += " AND p.is_active = true AND NOW() BETWEEN p.start_at AND p.end_at"
    } else if filters.Status == "upcoming" {
        query += " AND p.is_active = true AND p.start_at > NOW()"
    } else if filters.Status == "expired" {
        query += " AND p.end_at < NOW()"
    }
    
    // Count total
    countQuery := "SELECT COUNT(*) FROM (" + query + ") as count_query"
    var total int
    s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
    
    // Add sorting and pagination
    query += " ORDER BY p.created_at DESC"
    query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argPos, argPos+1)
    args = append(args, filters.Limit, (filters.Page-1)*filters.Limit)
    
    // Execute query
    rows, err := s.db.QueryContext(ctx, query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    promotions := []Promotion{}
    for rows.Next() {
        var p Promotion
        // Scan rows...
        promotions = append(promotions, p)
    }
    
    return &PromotionListResult{
        Promotions: promotions,
        Total:      total,
    }, nil
}
```


#### 4.4 Delete Promotion API

Soft delete - không xóa thật để preserve data integrity:[^1]

```go
func (h *AdminPromotionHandler) DeletePromotion(c *gin.Context) {
    promoID := c.Param("id")
    
    // Check if promotion has been used
    promo, err := h.promoService.GetByID(c.Request.Context(), promoID)
    if err != nil {
        c.JSON(404, errors.NewNotFoundError("Promotion not found"))
        return
    }
    
    if promo.UsedCount > 0 {
        // Soft delete - just deactivate
        err = h.promoService.DeactivatePromotion(c.Request.Context(), promoID)
    } else {
        // Hard delete if never used
        err = h.promoService.DeletePromotion(c.Request.Context(), promoID)
    }
    
    if err != nil {
        c.JSON(500, errors.NewInternalError(err.Error()))
        return
    }
    
    c.JSON(200, gin.H{"success": true, "message": "Promotion deleted"})
}
```


#### 4.5 View Promotion Usage Statistics

`GET /v1/admin/promotions/:id/usage`[^1]

```go
func (h *AdminPromotionHandler) GetPromotionUsage(c *gin.Context) {
    promoID := c.Param("id")
    
    stats, err := h.promoService.GetPromotionUsageStats(c.Request.Context(), promoID)
    if err != nil {
        c.JSON(500, errors.NewInternalError(err.Error()))
        return
    }
    
    c.JSON(200, gin.H{"success": true, "data": stats})
}
```

**Response**:

```json
{
  "success": true,
  "data": {
    "promotion_id": "uuid",
    "code": "SUMMER20",
    "total_used": 150,
    "max_usage": 1000,
    "usage_percentage": 15.0,
    "unique_users": 142,
    "total_discount_given": 12000000,
    "recent_usage": [
      {
        "user_email": "user@example.com",
        "order_number": "ORD-20251031-001",
        "discount_amount": 80000,
        "used_at": "2025-10-31T10:30:00Z"
      }
    ]
  }
}
```


### Acceptance Criteria

- Admin tạo được promotion với validation đầy đủ[^1]
- Update promotion với constraints (không đổi code nếu đã dùng)[^1]
- List promotions với filter và pagination[^1]
- Soft delete promotions đã dùng, hard delete chưa dùng[^1]
- View usage statistics chi tiết[^1]
- Audit log ghi lại mọi thay đổi[^1]


### Dependencies

- P2-T009: Promotions table[^1]
- P1-T029: RBAC middleware[^1]


### Effort

2 ngày[^1]

***

## 5. Reviews Table (P2-T013)

### Mô tả

Tạo database schema cho hệ thống reviews và ratings.[^1]

### Database Schema

#### 5.1 Reviews Table Migration

Tạo file `migrations/000022_create_reviews_table.up.sql`:[^1]

```sql
CREATE TABLE reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    book_id UUID NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    
    -- Rating and content
    rating INT NOT NULL CHECK (rating >= 1 AND rating <= 5),
    title TEXT,
    comment TEXT,
    images JSONB, -- ["url1", "url2", "url3"]
    
    -- Engagement
    helpful_count INT DEFAULT 0,
    
    -- Metadata
    is_verified_purchase BOOLEAN DEFAULT true,
    is_hidden BOOLEAN DEFAULT false,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- One review per user per order per book
    UNIQUE (book_id, user_id, order_id)
);

-- Indexes
CREATE INDEX idx_reviews_book ON reviews(book_id) WHERE is_hidden = false;
CREATE INDEX idx_reviews_user ON reviews(user_id);
CREATE INDEX idx_reviews_order ON reviews(order_id);
CREATE INDEX idx_reviews_rating ON reviews(book_id, rating);
CREATE INDEX idx_reviews_created ON reviews(created_at DESC);
CREATE INDEX idx_reviews_helpful ON reviews(book_id, helpful_count DESC) WHERE is_hidden = false;

-- Trigger auto update updated_at
CREATE TRIGGER update_reviews_updated_at
    BEFORE UPDATE ON reviews
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```


#### 5.2 Review Helpful Votes Table

```sql
CREATE TABLE review_votes (
    review_id UUID NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    PRIMARY KEY (review_id, user_id)
);

CREATE INDEX idx_review_votes_user ON review_votes(user_id);
```


#### 5.3 Update Books Table - Add Review Stats

```sql
ALTER TABLE books 
ADD COLUMN IF NOT EXISTS review_count INT DEFAULT 0,
ADD COLUMN IF NOT EXISTS average_rating NUMERIC(2,1) DEFAULT 0.0 CHECK (average_rating >= 0 AND average_rating <= 5);

CREATE INDEX IF NOT EXISTS idx_books_rating ON books(average_rating DESC) WHERE is_active = true;
```


#### 5.4 Trigger Auto Update Book Review Stats

```sql
CREATE OR REPLACE FUNCTION update_book_review_stats()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE books
    SET 
        review_count = (
            SELECT COUNT(*) 
            FROM reviews 
            WHERE book_id = NEW.book_id AND is_hidden = false
        ),
        average_rating = (
            SELECT COALESCE(AVG(rating), 0) 
            FROM reviews 
            WHERE book_id = NEW.book_id AND is_hidden = false
        )
    WHERE id = NEW.book_id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_book_review_stats
AFTER INSERT OR UPDATE OR DELETE ON reviews
FOR EACH ROW
EXECUTE FUNCTION update_book_review_stats();
```


### Công việc cụ thể

#### 5.5 Domain Model

Tạo file `internal/domains/review/model/review.go`:[^1]

```go
package model

import "time"

type Review struct {
    ID                 string    `json:"id"`
    BookID             string    `json:"book_id"`
    UserID             string    `json:"user_id"`
    OrderID            string    `json:"order_id"`
    Rating             int       `json:"rating"`
    Title              string    `json:"title"`
    Comment            string    `json:"comment"`
    Images             []string  `json:"images,omitempty"`
    HelpfulCount       int       `json:"helpful_count"`
    IsVerifiedPurchase bool      `json:"is_verified_purchase"`
    IsHidden           bool      `json:"is_hidden"`
    CreatedAt          time.Time `json:"created_at"`
    UpdatedAt          time.Time `json:"updated_at"`
    
    // Joined fields
    UserName           string    `json:"user_name,omitempty"`
    UserAvatar         string    `json:"user_avatar,omitempty"`
    BookTitle          string    `json:"book_title,omitempty"`
}

type ReviewVote struct {
    ReviewID  string    `json:"review_id"`
    UserID    string    `json:"user_id"`
    CreatedAt time.Time `json:"created_at"`
}
```


### Acceptance Criteria

- Migration tạo bảng reviews và review_votes thành công[^1]
- Constraint đảm bảo 1 user chỉ review 1 lần cho mỗi book trong mỗi order[^1]
- Trigger auto update book review stats hoạt động[^1]
- Indexes optimize queries[^1]


### Dependencies

- P1-T002: Database setup[^1]
- P1-T003: Core tables (books, users, orders)[^1]


### Effort

1 ngày[^1]

***

## 6. Create Review API (After Delivery) (P2-T014)

### Mô tả

API cho user viết review sau khi đơn hàng delivered.[^1]

### API Endpoint

`POST /v1/books/:book_id/reviews`[^1]

### Request Body

```json
{
  "order_id": "uuid",
  "rating": 5,
  "title": "Sách rất hay!",
  "comment": "Nội dung sâu sắc, đáng đọc. Giao hàng nhanh.",
  "images": ["url1", "url2"]
}
```


### Validation Rules

1. User phải đã mua sách (tồn tại order_item với book_id)[^1]
2. Order status phải là "delivered" hoặc "completed"[^1]
3. User chưa review book này trong order này[^1]
4. Rating 1-5 stars (required)[^1]
5. Comment max 1000 chars (optional)[^1]
6. Max 5 images[^1]

### Công việc cụ thể

#### 6.1 Review Service

Tạo file `internal/domains/review/service/review_service.go`:[^1]

```go
package service

import (
    "context"
    "fmt"
)

type ReviewService struct {
    reviewRepo *repository.ReviewRepository
    orderRepo  *repository.OrderRepository
}

type CreateReviewParams struct {
    BookID  string
    UserID  string
    OrderID string
    Rating  int
    Title   string
    Comment string
    Images  []string
}

func (s *ReviewService) CreateReview(ctx context.Context, params CreateReviewParams) (*Review, error) {
    // 1. Validate user has purchased the book
    hasPurchased, err := s.validatePurchase(ctx, params.UserID, params.BookID, params.OrderID)
    if err != nil {
        return nil, err
    }
    if !hasPurchased {
        return nil, fmt.Errorf("you have not purchased this book")
    }
    
    // 2. Validate order is delivered
    order, err := s.orderRepo.FindByID(ctx, params.OrderID)
    if err != nil {
        return nil, fmt.Errorf("order not found")
    }
    
    if order.Status != "delivered" && order.Status != "completed" {
        return nil, fmt.Errorf("you can only review after receiving the order")
    }
    
    // 3. Check if user already reviewed
    exists, err := s.reviewRepo.ExistsByUserBookOrder(ctx, params.UserID, params.BookID, params.OrderID)
    if err != nil {
        return nil, err
    }
    if exists {
        return nil, fmt.Errorf("you have already reviewed this book for this order")
    }
    
    // 4. Validate images count
    if len(params.Images) > 5 {
        return nil, fmt.Errorf("maximum 5 images allowed")
    }
    
    // 5. Create review
    review := &Review{
        BookID:             params.BookID,
        UserID:             params.UserID,
        OrderID:            params.OrderID,
        Rating:             params.Rating,
        Title:              params.Title,
        Comment:            params.Comment,
        Images:             params.Images,
        IsVerifiedPurchase: true,
        IsHidden:           false,
    }
    
    err = s.reviewRepo.Create(ctx, review)
    if err != nil {
        return nil, err
    }
    
    // 6. Trigger update book stats (handled by database trigger)
    
    return review, nil
}

func (s *ReviewService) validatePurchase(ctx context.Context, userID string, bookID string, orderID string) (bool, error) {
    query := `
        SELECT EXISTS(
            SELECT 1 
            FROM orders o
            JOIN order_items oi ON o.id = oi.order_id
            WHERE o.id = $1 
            AND o.user_id = $2
            AND oi.book_id = $3
        )
    `
    
    var exists bool
    err := s.db.QueryRowContext(ctx, query, orderID, userID, bookID).Scan(&exists)
    return exists, err
}
```


#### 6.2 Validation with ozzo-validation

```go
type CreateReviewRequest struct {
    OrderID string   `json:"order_id"`
    Rating  int      `json:"rating"`
    Title   string   `json:"title"`
    Comment string   `json:"comment"`
    Images  []string `json:"images"`
}

func (r CreateReviewRequest) Validate() error {
    return validation.ValidateStruct(&r,
        validation.Field(&r.OrderID, 
            validation.Required,
            is.UUIDv4),
        validation.Field(&r.Rating, 
            validation.Required,
            validation.Min(1),
            validation.Max(5)),
        validation.Field(&r.Title,
            validation.Length(0, 200)),
        validation.Field(&r.Comment,
            validation.Length(0, 1000)),
        validation.Field(&r.Images,
            validation.Length(0, 5)),
    )
}
```


#### 6.3 Handler Implementation

```go
func (h *ReviewHandler) CreateReview(c *gin.Context) {
    bookID := c.Param("book_id")
    userID := c.GetString("user_id") // From JWT
    
    var req CreateReviewRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"success": false, "error": "Invalid request"})
        return
    }
    
    if err := req.Validate(); err != nil {
        c.JSON(400, gin.H{"success": false, "error": err.Error()})
        return
    }
    
    review, err := h.reviewService.CreateReview(c.Request.Context(), CreateReviewParams{
        BookID:  bookID,
        UserID:  userID,
        OrderID: req.OrderID,
        Rating:  req.Rating,
        Title:   req.Title,
        Comment: req.Comment,
        Images:  req.Images,
    })
    
    if err != nil {
        c.JSON(400, gin.H{"success": false, "error": err.Error()})
        return
    }
    
    c.JSON(201, gin.H{"success": true, "data": review})
}
```


#### 6.4 Response Format

```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "book_id": "uuid",
    "user_id": "uuid",
    "order_id": "uuid",
    "rating": 5,
    "title": "Sách rất hay!",
    "comment": "Nội dung sâu sắc...",
    "images": ["url1", "url2"],
    "helpful_count": 0,
    "is_verified_purchase": true,
    "created_at": "2025-10-31T10:30:00Z"
  }
}
```


### Acceptance Criteria

- User chỉ review được sau khi order delivered[^1]
- User phải đã mua sách mới review được[^1]
- Không review trùng lặp cho cùng book + order[^1]
- Validation đầy đủ (rating, comment length, images count)[^1]
- Book review stats tự động update (trigger)[^1]


### Dependencies

- P2-T013: Reviews table[^1]
- P1-T025: Order creation API[^1]
- P1-T012: JWT middleware[^1]


### Effort

2 ngày[^1]

***

## 7. List Reviews for Book (P2-T015)

### Mô tả

Public API để hiển thị reviews của sách với filter và sort.[^1]

### API Endpoint

`GET /v1/books/:book_id/reviews`[^1]

### Query Parameters

- `?page=1&limit=10` - Pagination (default: 10 items/page)[^1]
- `?rating=5` - Filter by rating (1-5)[^1]
- `?sort=helpful` - Sort by: `helpful`, `recent`, `rating_high`, `rating_low`[^1]
- `?verified_only=true` - Show only verified purchases[^1]


### Công việc cụ thể

#### 7.1 Review Service - List Reviews

```go
type ReviewFilters struct {
    BookID       string
    Rating       *int
    SortBy       string // helpful, recent, rating_high, rating_low
    VerifiedOnly bool
    Page         int
    Limit        int
}

type ReviewListResult struct {
    Reviews      []Review `json:"reviews"`
    Total        int      `json:"total"`
    AverageRating float64 `json:"average_rating"`
    RatingBreakdown map[int]int `json:"rating_breakdown"`
}

func (s *ReviewService) ListReviews(ctx context.Context, filters ReviewFilters) (*ReviewListResult, error) {
    // 1. Build base query
    query := `
        SELECT 
            r.*,
            u.fullname as user_name,
            u.avatar_url as user_avatar
        FROM reviews r
        JOIN users u ON r.user_id = u.id
        WHERE r.book_id = $1
        AND r.is_hidden = false
    `
    
    args := []interface{}{filters.BookID}
    argPos := 2
    
    // 2. Apply filters
    if filters.Rating != nil {
        query += fmt.Sprintf(" AND r.rating = $%d", argPos)
        args = append(args, *filters.Rating)
        argPos++
    }
    
    if filters.VerifiedOnly {
        query += " AND r.is_verified_purchase = true"
    }
    
    // 3. Apply sorting
    switch filters.SortBy {
    case "helpful":
        query += " ORDER BY r.helpful_count DESC, r.created_at DESC"
    case "rating_high":
        query += " ORDER BY r.rating DESC, r.created_at DESC"
    case "rating_low":
        query += " ORDER BY r.rating ASC, r.created_at DESC"
    case "recent":
        fallthrough
    default:
        query += " ORDER BY r.created_at DESC"
    }
    
    // 4. Count total
    countQuery := `
        SELECT COUNT(*) 
        FROM reviews 
        WHERE book_id = $1 AND is_hidden = false
    `
    var total int
    s.db.QueryRowContext(ctx, countQuery, filters.BookID).Scan(&total)
    
    // 5. Apply pagination
    query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argPos, argPos+1)
    args = append(args, filters.Limit, (filters.Page-1)*filters.Limit)
    
    // 6. Execute query
    rows, err := s.db.QueryContext(ctx, query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    reviews := []Review{}
    for rows.Next() {
        var r Review
        err := rows.Scan(
            &r.ID, &r.BookID, &r.UserID, &r.OrderID,
            &r.Rating, &r.Title, &r.Comment, &r.Images,
            &r.HelpfulCount, &r.IsVerifiedPurchase, &r.IsHidden,
            &r.CreatedAt, &r.UpdatedAt,
            &r.UserName, &r.UserAvatar,
        )
        if err != nil {
            return nil, err
        }
        reviews = append(reviews, r)
    }
    
    // 7. Get rating breakdown
    ratingBreakdown := s.getRatingBreakdown(ctx, filters.BookID)
    
    // 8. Get average rating from book
    var avgRating float64
    s.db.QueryRowContext(ctx, 
        "SELECT average_rating FROM books WHERE id = $1", 
        filters.BookID,
    ).Scan(&avgRating)
    
    return &ReviewListResult{
        Reviews:         reviews,
        Total:           total,
        AverageRating:   avgRating,
        RatingBreakdown: ratingBreakdown,
    }, nil
}

func (s *ReviewService) getRatingBreakdown(ctx context.Context, bookID string) map[int]int {
    query := `
        SELECT rating, COUNT(*) 
        FROM reviews 
        WHERE book_id = $1 AND is_hidden = false
        GROUP BY rating
        ORDER BY rating DESC
    `
    
    breakdown := make(map[int]int)
    // Initialize all ratings to 0
    for i := 1; i <= 5; i++ {
        breakdown[i] = 0
    }
    
    rows, err := s.db.QueryContext(ctx, query, bookID)
    if err != nil {
        return breakdown
    }
    defer rows.Close()
    
    for rows.Next() {
        var rating, count int
        rows.Scan(&rating, &count)
        breakdown[rating] = count
    }
    
    return breakdown
}
```


#### 7.2 Handler Implementation

```go
func (h *ReviewHandler) ListReviews(c *gin.Context) {
    bookID := c.Param("book_id")
    
    // Parse query params
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
    sortBy := c.DefaultQuery("sort", "recent")
    verifiedOnly := c.Query("verified_only") == "true"
    
    var rating *int
    if r := c.Query("rating"); r != "" {
        rInt, _ := strconv.Atoi(r)
        rating = &rInt
    }
    
    filters := ReviewFilters{
        BookID:       bookID,
        Rating:       rating,
        SortBy:       sortBy,
        VerifiedOnly: verifiedOnly,
        Page:         page,
        Limit:        limit,
    }
    
    result, err := h.reviewService.ListReviews(c.Request.Context(), filters)
    if err != nil {
        c.JSON(500, gin.H{"success": false, "error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{
        "success": true,
        "data": gin.H{
            "reviews": result.Reviews,
            "statistics": gin.H{
                "average_rating":   result.AverageRating,
                "total_reviews":    result.Total,
                "rating_breakdown": result.RatingBreakdown,
            },
        },
        "meta": gin.H{
            "page":  page,
            "limit": limit,
            "total": result.Total,
        },
    })
}
```


#### 7.3 Response Format

```json
{
  "success": true,
  "data": {
    "reviews": [
      {
        "id": "uuid",
        "rating": 5,
        "title": "Tuyệt vời!",
        "comment": "Nội dung hay...",
        "images": ["url1"],
        "helpful_count": 25,
        "is_verified_purchase": true,
        "created_at": "2025-10-31T10:00:00Z",
        "user_name": "Nguyễn Văn A",
        "user_avatar": "avatar_url"
      }
    ],
    "statistics": {
      "average_rating": 4.5,
      "total_reviews": 150,
      "rating_breakdown": {
        "5": 80,
        "4": 40,
        "3": 20,
        "2": 7,
        "1": 3
      }
    }
  },
  "meta": {
    "page": 1,
    "limit": 10,
    "total": 150
  }
}
```


### Acceptance Criteria

- List reviews với pagination[^1]
- Filter theo rating[^1]
- Sort theo helpful, recent, rating[^1]
- Hiển thị rating breakdown (5★: 80, 4★: 40...)[^1]
- Performance: P95 < 200ms với 1000 reviews[^1]


### Dependencies

- P2-T013: Reviews table[^1]
- P2-T014: Create review API[^1]


### Effort

1 ngày[^1]

***

## 8. Admin: Hide/Show Reviews (P2-T016)

### Mô tả

Admin có thể ẩn/hiện reviews không phù hợp (spam, offensive).[^1]

### API Endpoints

- `PATCH /v1/admin/reviews/:id/hide` - Hide review[^1]
- `PATCH /v1/admin/reviews/:id/show` - Show review[^1]
- `GET /v1/admin/reviews?is_hidden=true` - List all hidden reviews[^1]


### Công việc cụ thể

#### 8.1 Admin Review Service

```go
func (s *AdminReviewService) HideReview(ctx context.Context, reviewID string, reason string) error {
    // 1. Get review
    review, err := s.reviewRepo.FindByID(ctx, reviewID)
    if err != nil {
        return fmt.Errorf("review not found")
    }
    
    // 2. Update is_hidden flag
    review.IsHidden = true
    err = s.reviewRepo.Update(ctx, review)
    if err != nil {
        return err
    }
    
    // 3. Log audit trail
    s.auditService.Log(ctx, AuditLog{
        Action:   "HIDE_REVIEW",
        EntityID: reviewID,
        Reason:   reason,
    })
    
    // 4. Book stats will auto-update via trigger
    
    return nil
}

func (s *AdminReviewService) ShowReview(ctx context.Context, reviewID string) error {
    review, err := s.reviewRepo.FindByID(ctx, reviewID)
    if err != nil {
        return fmt.Errorf("review not found")
    }
    
    review.IsHidden = false
    err = s.reviewRepo.Update(ctx, review)
    if err != nil {
        return err
    }
    
    s.auditService.Log(ctx, AuditLog{
        Action:   "SHOW_REVIEW",
        EntityID: reviewID,
    })
    
    return nil
}
```


#### 8.2 Handler Implementation

```go
func (h *AdminReviewHandler) HideReview(c *gin.Context) {
    reviewID := c.Param("id")
    
    var req struct {
        Reason string `json:"reason" binding:"required"`
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"success": false, "error": "Reason is required"})
        return
    }
    
    err := h.adminReviewService.HideReview(c.Request.Context(), reviewID, req.Reason)
    if err != nil {
        c.JSON(500, gin.H{"success": false, "error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{"success": true, "message": "Review hidden"})
}

func (h *AdminReviewHandler) ShowReview(c *gin.Context) {
    reviewID := c.Param("id")
    
    err := h.adminReviewService.ShowReview(c.Request.Context(), reviewID)
    if err != nil {
        c.JSON(500, gin.H{"success": false, "error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{"success": true, "message": "Review shown"})
}
```


#### 8.3 List Hidden Reviews for Moderation

```go
func (h *AdminReviewHandler) ListHiddenReviews(c *gin.Context) {
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
    
    reviews, total, err := h.adminReviewService.ListHiddenReviews(c.Request.Context(), page, limit)
    if err != nil {
        c.JSON(500, gin.H{"success": false, "error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{
        "success": true,
        "data":    reviews,
        "meta": gin.H{
            "page":  page,
            "limit": limit,
            "total": total,
        },
    })
}
```


### Acceptance Criteria

- Admin hide được review với lý do[^1]
- Review bị hidden không hiển thị trong public API[^1]
- Book stats tự động update khi hide/show[^1]
- Audit log ghi lại action[^1]
- Admin xem được list reviews đã hidden[^1]


### Dependencies

- P2-T013: Reviews table[^1]
- P1-T029: RBAC middleware[^1]


### Effort

1 ngày[^1]

***

## 9. Helpful Votes on Reviews (P2-T017)

### Mô tả

User có thể vote review "helpful" để giúp sắp xếp reviews chất lượng lên đầu.[^1]

### API Endpoints

- `POST /v1/reviews/:id/vote` - Vote review as helpful[^1]
- `DELETE /v1/reviews/:id/vote` - Remove vote[^1]


### Business Logic

- 1 user chỉ vote 1 lần cho mỗi review[^1]
- Tăng `helpful_count` của review khi vote[^1]
- User có thể unvote[^1]


### Công việc cụ thể

#### 9.1 Vote Service

```go
func (s *ReviewService) VoteHelpful(ctx context.Context, reviewID string, userID string) error {
    // 1. Check if already voted
    exists, err := s.reviewVoteRepo.Exists(ctx, reviewID, userID)
    if err != nil {
        return err
    }
    
    if exists {
        return fmt.Errorf("you have already voted this review")
    }
    
    // 2. Start transaction
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // 3. Insert vote
    vote := &ReviewVote{
        ReviewID: reviewID,
        UserID:   userID,
    }
    
    err = s.reviewVoteRepo.CreateTx(ctx, tx, vote)
    if err != nil {
        return err
    }
    
    // 4. Increment helpful_count
    err = s.reviewRepo.IncrementHelpfulCountTx(ctx, tx, reviewID)
    if err != nil {
        return err
    }
    
    // 5. Commit
    return tx.Commit()
}

func (s *ReviewService) UnvoteHelpful(ctx context.Context, reviewID string, userID string) error {
    // 1. Check if voted
    exists, err := s.reviewVoteRepo.Exists(ctx, reviewID, userID)
    if err != nil {
        return err
    }
    
    if !exists {
        return fmt.Errorf("you have not voted this review")
    }
    
    // 2. Start transaction
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // 3. Delete vote
    err = s.reviewVoteRepo.DeleteTx(ctx, tx, reviewID, userID)
    if err != nil {
        return err
    }
    
    // 4. Decrement helpful_count
    err = s.reviewRepo.DecrementHelpfulCountTx(ctx, tx, reviewID)
    if err != nil {
        return err
    }
    
    // 5. Commit
    return tx.Commit()
}
```


#### 9.2 Repository Implementation

```go
func (r *ReviewVoteRepository) Exists(ctx context.Context, reviewID string, userID string) (bool, error) {
    var exists bool
    query := `SELECT EXISTS(SELECT 1 FROM review_votes WHERE review_id = $1 AND user_id = $2)`
    err := r.db.QueryRowContext(ctx, query, reviewID, userID).Scan(&exists)
    return exists, err
}

func (r *ReviewRepository) IncrementHelpfulCountTx(ctx context.Context, tx *sql.Tx, reviewID string) error {
    query := `UPDATE reviews SET helpful_count = helpful_count + 1 WHERE id = $1`
    _, err := tx.ExecContext(ctx, query, reviewID)
    return err
}

func (r *ReviewRepository) DecrementHelpfulCountTx(ctx context.Context, tx *sql.Tx, reviewID string) error {
    query := `UPDATE reviews SET helpful_count = GREATEST(helpful_count - 1, 0) WHERE id = $1`
    _, err := tx.ExecContext(ctx, query, reviewID)
    return err
}
```


#### 9.3 Handler Implementation

```go
func (h *ReviewHandler) VoteHelpful(c *gin.Context) {
    reviewID := c.Param("id")
    userID := c.GetString("user_id") // From JWT
    
    err := h.reviewService.VoteHelpful(c.Request.Context(), reviewID, userID)
    if err != nil {
        c.JSON(400, gin.H{"success": false, "error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{"success": true, "message": "Voted as helpful"})
}

func (h *ReviewHandler) UnvoteHelpful(c *gin.Context) {
    reviewID := c.Param("id")
    userID := c.GetString("user_id")
    
    err := h.reviewService.UnvoteHelpful(c.Request.Context(), reviewID, userID)
    if err != nil {
        c.JSON(400, gin.H{"success": false, "error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{"success": true, "message": "Vote removed"})
}
```


#### 9.4 Include User Vote Status in List Reviews

Update `ListReviews` response để show user đã vote chưa:

```go
// In Review model
type Review struct {
    // ... existing fields
    HasUserVoted bool `json:"has_user_voted,omitempty"`
}

// In ListReviews service
func (s *ReviewService) ListReviews(ctx context.Context, filters ReviewFilters, currentUserID *string) (*ReviewListResult, error) {
    // ... existing query
    
    // If user is logged in, check which reviews they voted
    if currentUserID != nil {
        votedReviewIDs := s.getUserVotedReviews(ctx, *currentUserID, filters.BookID)
        votedMap := make(map[string]bool)
        for _, id := range votedReviewIDs {
            votedMap[id] = true
        }
        
        for i := range reviews {
            reviews[i].HasUserVoted = votedMap[reviews[i].ID]
        }
    }
    
    return result, nil
}
```


### Acceptance Criteria

- User vote được review helpful[^1]
- Không vote trùng lặp[^1]
- Unvote hoạt động đúng[^1]
- Transaction safety (increment/decrement atomic)[^1]
- Sort by helpful work correctly[^1]
- Response hiển thị user đã vote chưa[^1]


### Dependencies

- P2-T013: Reviews table[^1]
- P2-T015: List reviews API[^1]


### Effort

1 ngày[^1]

***

## SUMMARY

### Total Effort Sprint 11-12

| Task ID | Task | Effort (days) |
| :-- | :-- | :-- |
| P2-T009 | Promotions table + seed data | 1 |
| P2-T010 | Apply promo code to cart API | 2 |
| P2-T011 | Validate promo usage limits | 1 |
| P2-T012 | Admin CRUD promotions | 2 |
| P2-T013 | Reviews table | 1 |
| P2-T014 | Create review API (after delivery) | 2 |
| P2-T015 | List reviews for book | 1 |
| P2-T016 | Admin Hide/Show reviews | 1 |
| P2-T017 | Helpful votes on reviews | 1 |
| **TOTAL** |  | **12 days** |

**Sprint duration**: 2 tuần (10 ngày làm việc)[^1]
**Team size**: 2 backend developers (có thể song song hóa tasks)[^1]

### Parallelization Strategy

**Week 1** (5 ngày):

- **Dev 1**: P2-T009 → P2-T010 → P2-T011 (1+2+1 = 4 days) + Review code (1 day)[^1]
- **Dev 2**: P2-T013 → P2-T014 (1+2 = 3 days) + P2-T015 (1 day) + Review code (1 day)[^1]

**Week 2** (5 ngày):

- **Dev 1**: P2-T012 (Admin CRUD promotions) (2 days) + Testing \& bug fixes (3 days)[^1]
- **Dev 2**: P2-T016 → P2-T017 (1+1 = 2 days) + Testing \& bug fixes (3 days)[^1]


### Deliverables Checklist Sprint 11-12

- ✅ Promo code system hoàn chỉnh với

<div align="center">⁂</div>

[^1]: USER-REQUIREMENTS-DOCUMENT-URD-PHIEN-BAN-HOA.docx

