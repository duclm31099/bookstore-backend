package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	cart "bookstore-backend/internal/domains/cart/model"
	cartService "bookstore-backend/internal/domains/cart/service"
	"bookstore-backend/internal/domains/promotion/model"
	"bookstore-backend/internal/domains/promotion/repository"
	"bookstore-backend/pkg/logger"
)

// PromotionService xử lý business logic cho promotion
type promotionService struct {
	repo       repository.PromotionRepository
	calculator *DiscountCalculator
	pool       *pgxpool.Pool // Để tạo transaction
	cart       cartService.ServiceInterface
}

// OrderRepository interface để tránh circular dependency
type OrderRepository interface {
	GetCompletedOrderCount(ctx context.Context, userID uuid.UUID) (int, error)
}

// NewPromotionService tạo service instance mới
func NewPromotionService(
	repo repository.PromotionRepository,
	pool *pgxpool.Pool,
	cart cartService.ServiceInterface,
) ServiceInterface {
	return &promotionService{
		repo:       repo,
		calculator: NewDiscountCalculator(),
		pool:       pool,
		cart:       cart,
	}
}

// -------------------------------------------------------------------
// REMOVE PROMOTION FROM CART
// -------------------------------------------------------------------

// RemovePromotionFromCart xóa promotion khỏi cart
//
// Simple flow:
// 1. Call CartService.RemovePromotion
// 2. Return updated cart
func (s *promotionService) RemovePromotionFromCart(
	ctx context.Context,
	userID uuid.UUID,
) (*cart.CartResponse, error) {
	// Remove promotion
	err := s.cart.RemovePromoCode(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("remove promotion from cart: %w", err)
	}

	// Get updated cart
	updatedCart, err := s.cart.GetOrCreateCart(ctx, &userID, nil)
	if err != nil {
		return nil, fmt.Errorf("get updated cart: %w", err)
	}

	return updatedCart, nil
}

// -------------------------------------------------------------------
// ADDITIONAL HELPER METHODS
// -------------------------------------------------------------------

// GetPromotionByCode lấy promotion theo code (dùng cho internal services)
func (s *promotionService) GetPromotionByCode(ctx context.Context, code string) (*model.Promotion, error) {
	return s.repo.FindByCode(ctx, code)
}

// ValidatePromotionForOrder validates promotion khi checkout
//
// Giống ValidatePromotion nhưng có thêm validation cho order context
func (s *promotionService) ValidatePromotionForOrder(
	ctx context.Context,
	code string,
	userID uuid.UUID,
	orderItems []cart.CartItem,
	subtotal decimal.Decimal,
) (*model.ValidationResult, error) {
	req := &model.ValidatePromotionRequest{
		Code:      code,
		CartItems: nil,
		Subtotal:  subtotal,
		UserID:    &userID,
	}

	return s.ValidatePromotion(ctx, req)
}

// RecordUsageWithTx record promotion usage trong transaction (called by Order service)
//
// Note: Đây là version nhận transaction từ bên ngoài
func (s *promotionService) RecordUsageWithTx(
	ctx context.Context,
	tx *pgxpool.Tx,
	orderID, promoID, userID uuid.UUID,
	discountAmount decimal.Decimal,
) error {
	usage := &model.PromotionUsage{
		PromotionID:    promoID,
		UserID:         userID,
		OrderID:        orderID,
		DiscountAmount: discountAmount,
	}

	err := s.repo.CreateUsage(ctx, tx, usage)
	if err != nil {
		return err
	}

	return nil
}

// GetPromotionStats lấy stats overview của một promotion
func (s *promotionService) GetPromotionStats(ctx context.Context, promoID uuid.UUID) (*model.UsageStats, error) {
	return s.repo.GetUsageStats(ctx, promoID, nil, nil)
}

// CheckPromotionAvailability kiểm tra promotion có available không (quick check)
func (s *promotionService) CheckPromotionAvailability(ctx context.Context, code string) (bool, error) {
	promo, err := s.repo.FindByCodeActive(ctx, code)
	if err != nil {
		return false, err
	}

	// Check usage limit
	if promo.IsUsageLimitReached() {
		return false, nil
	}

	return true, nil
}

// CALCULATE DISCOUNT (WRAPPER AROUND CALCULATOR)
// -------------------------------------------------------------------

// CalculateDiscount tính toán discount amount
//
// Đây là wrapper method để dễ gọi từ bên ngoài
// Logic thực sự nằm trong DiscountCalculator
func (s *promotionService) CalculateDiscount(promo *model.Promotion, subtotal decimal.Decimal) decimal.Decimal {
	return s.calculator.Calculate(promo, subtotal)
}

// -------------------------------------------------------------------
// APPLY PROMOTION TO CART
// -------------------------------------------------------------------

// ApplyPromotionToCart áp dụng promotion vào cart của user
//
// Business Flow:
// 1. Get user's cart
// 2. Build cart items for validation
// 3. Validate promotion với cart items
// 4. If valid: Store promo code + discount amount in cart
// 5. Return updated cart
//
// Note: Không record vào promotion_usage ở đây
//
//	Chỉ record khi order payment success
func (s *promotionService) ApplyPromotionToCart(
	ctx context.Context,
	userID uuid.UUID,
	code string,
) (*cart.CartResponse, error) {
	// Step 1: Get user's cart
	cart, err := s.cart.GetOrCreateCart(ctx, &userID, nil)
	if err != nil {
		return nil, fmt.Errorf("get cart: %w", err)
	}

	// Validate cart không rỗng
	if len(cart.Items) == 0 {
		return nil, errors.New("Giỏ hàng trống, không thể áp dụng mã giảm giá")
	}

	// Step 2: Build cart items for validation
	cartItems := make([]model.CartItem, len(cart.Items))
	subtotal := decimal.Zero

	for i, item := range cart.Items {
		cartItems[i] = model.CartItem{
			BookID:   item.BookID,
			Price:    item.Price,
			Quantity: item.Quantity,
		}

		itemSubtotal := item.Price.Mul(decimal.NewFromInt(int64(item.Quantity)))
		subtotal = subtotal.Add(itemSubtotal)
	}

	// Step 3: Validate promotion
	validateReq := &model.ValidatePromotionRequest{
		Code:      code,
		CartItems: cartItems,
		Subtotal:  subtotal,
		UserID:    &userID,
	}

	validationResult, err := s.ValidatePromotion(ctx, validateReq)
	if err != nil {
		return nil, err
	}
	fmt.Printf("validationResult %w", validationResult)

	// Step 4: Store promo in cart
	c, err := s.cart.ApplyPromoCode(ctx, cart.ID, code, userID)
	if err != nil {
		return nil, fmt.Errorf("apply promotion to cart: %w", err)
	}

	// Step 5: Get updated cart
	updatedCart, err := s.cart.GetOrCreateCart(ctx, &userID, nil)
	if err != nil {
		return nil, fmt.Errorf("get updated cart: %w", c)
	}
	return updatedCart, nil
}

// -------------------------------------------------------------------
// UPDATE PROMOTION (IMPLEMENTATION CHI TIẾT)
// -------------------------------------------------------------------

// UpdatePromotion cập nhật promotion với business rules
//
// Business Rules:
// 1. Nếu current_uses > 0 (đã có người dùng):
//   - KHÔNG được thay đổi: code, discount_type, discount_value, applicable_category_ids
//   - Lý do: Làm ảnh hưởng đến tính toán của các order đã áp dụng
//
// 2. max_uses chỉ có thể tăng hoặc giữ nguyên, không được giảm xuống < current_uses
//
// 3. max_uses_per_user chỉ có thể tăng, không được giảm nếu đã có user usage
//
// 4. expires_at có thể extend nhưng không được rút ngắn nếu đã có usage
//
// 5. Sử dụng optimistic locking (version field) để tránh race condition
func (s *promotionService) UpdatePromotion(
	ctx context.Context,
	id uuid.UUID,
	req *model.UpdatePromotionRequest,
) (*model.Promotion, error) {
	// Step 1: Get existing promotion với FOR UPDATE lock (trong transaction)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get current promotion
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Step 3: Apply updates (chỉ update các field được gửi lên)
	updated := *existing // Copy
	hasChanges := false

	if req.Name != nil {
		updated.Name = *req.Name
		hasChanges = true
	}

	if req.Description != nil {
		updated.Description = req.Description
		hasChanges = true
	}

	if req.MaxDiscountAmount != nil {
		updated.MaxDiscountAmount = req.MaxDiscountAmount
		hasChanges = true
	}

	if req.MinOrderAmount != nil {
		// Validate: Có thể tăng hoặc giảm
		updated.MinOrderAmount = *req.MinOrderAmount
		hasChanges = true
	}

	if req.MaxUses != nil {
		// Validate: Không được giảm xuống dưới current_uses
		if *req.MaxUses < existing.CurrentUses {
			return nil, errors.New("max_uses không được nhỏ hơn current_uses")
		}
		updated.MaxUses = req.MaxUses
		hasChanges = true
	}

	if req.MaxUsesPerUser != nil {
		// Validate: Nên check max user usage hiện tại
		// Để đơn giản: Chỉ cho phép tăng
		if *req.MaxUsesPerUser < existing.MaxUsesPerUser {
			return nil, errors.New("max_uses_per_user chỉ được phép tăng")
		}
		updated.MaxUsesPerUser = *req.MaxUsesPerUser
		hasChanges = true
	}

	if req.StartsAt != nil {
		startsAt, err := time.Parse(time.RFC3339, *req.StartsAt)
		if err != nil {
			return nil, errors.New("Định dạng starts_at không hợp lệ")
		}

		// Validate: starts_at < expires_at
		if !startsAt.Before(updated.ExpiresAt) {
			return nil, errors.New("starts_at phải trước expires_at")
		}

		updated.StartsAt = startsAt
		hasChanges = true
	}

	if req.ExpiresAt != nil {
		expiresAt, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			return nil, fmt.Errorf("Định dạng expires_at không hợp lệ")
		}

		// Validate: expires_at > starts_at
		if !expiresAt.After(updated.StartsAt) {
			return nil, fmt.Errorf("invalid promotion code")
		}

		// Nếu đã có usage, không được rút ngắn thời gian
		if existing.CurrentUses > 0 && expiresAt.Before(existing.ExpiresAt) {
			return nil, fmt.Errorf("invalid promotion code")
		}

		updated.ExpiresAt = expiresAt
		hasChanges = true
	}

	if req.IsActive != nil {
		updated.IsActive = *req.IsActive
		hasChanges = true
	}

	// Nếu không có gì thay đổi
	if !hasChanges {
		return existing, nil
	}

	// Step 4: Save với optimistic locking
	err = s.repo.Update(ctx, &updated)
	if err != nil {
		return nil, err
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &updated, nil
}

// -------------------------------------------------------------------
// PUBLIC METHODS (User-facing)
// -------------------------------------------------------------------

// ValidatePromotion validates promotion code với cart hiện tại
//
// Business Logic Flow:
// 1. Tìm promotion active theo code
// 2. Validate time window (starts_at <= now <= expires_at)
// 3. Check global usage limit
// 4. Check user usage limit (nếu authenticated)
// 5. Check minimum order amount
// 6. Check first order only (nếu enabled)
// 7. Check category applicability
// 8. Calculate discount amount
// 9. Return validation result
//
// Error Cases:
// - PROMO_NOT_FOUND: Code không tồn tại hoặc inactive
// - PROMO_NOT_STARTED: Chưa đến thời gian bắt đầu
// - PROMO_EXPIRED: Đã hết hạn
// - PROMO_USAGE_LIMIT_EXCEEDED: Hết lượt sử dụng (global)
// - PROMO_USER_LIMIT_EXCEEDED: User đã dùng hết lượt
// - PROMO_MIN_ORDER_NOT_MET: Giá trị đơn hàng chưa đủ
// - PROMO_FIRST_ORDER_ONLY: Không phải đơn đầu tiên
// - PROMO_CATEGORY_NOT_APPLICABLE: Không có sản phẩm phù hợp
func (s *promotionService) ValidatePromotion(ctx context.Context, req *model.ValidatePromotionRequest) (*model.ValidationResult, error) {
	// Normalize code
	req.NormalizeCode()

	// Step 1: Find active promotion
	promo, err := s.repo.FindByCodeActive(ctx, req.Code)
	logger.Info("find promo by code", map[string]interface{}{
		"promo": promo,
	})
	if err != nil {
		return nil, err
	}

	// Step 2: Validate time window (double-check vì query đã filter)
	now := time.Now()
	if now.Before(promo.StartsAt) {
		return nil, &model.AppError{
			Code:       model.ErrCodePromoNotStarted,
			Message:    "Mã giảm giá chưa bắt đầu",
			HTTPStatus: 400,
			Details: map[string]interface{}{
				"starts_at": promo.StartsAt,
			},
		}
	}

	if now.After(promo.ExpiresAt) {
		return nil, &model.AppError{
			Code:       model.ErrCodePromoExpired,
			Message:    "Mã giảm giá đã hết hạn",
			HTTPStatus: 400,
			Details: map[string]interface{}{
				"expired_at": promo.ExpiresAt,
			},
		}
	}

	// Step 3: Check global usage limit
	if promo.MaxUses != nil && promo.CurrentUses >= *promo.MaxUses {
		return nil, &model.AppError{
			Code:       model.ErrCodePromoUsageLimitExceeded,
			Message:    "Mã giảm giá đã hết lượt sử dụng",
			HTTPStatus: 400,
			Details: map[string]interface{}{
				"max_uses":     *promo.MaxUses,
				"current_uses": promo.CurrentUses,
			},
		}
	}

	// Step 4: Check user usage limit (chỉ khi authenticated)
	var userUsageCount int
	var userRemainingUses int

	if req.UserID != nil && *req.UserID != uuid.Nil {
		userUsageCount, err = s.repo.GetUserUsageCount(ctx, promo.ID, *req.UserID)
		if err != nil {
			return nil, fmt.Errorf("get user usage count: %w", err)
		}

		if userUsageCount >= promo.MaxUsesPerUser {
			return nil, &model.AppError{
				Code:       model.ErrCodePromoUserLimitExceeded,
				Message:    fmt.Sprintf("Bạn đã sử dụng hết %d lượt cho mã này", promo.MaxUsesPerUser),
				HTTPStatus: 400,
				Details: map[string]interface{}{
					"max_uses_per_user": promo.MaxUsesPerUser,
					"user_usage_count":  userUsageCount,
				},
			}
		}

		userRemainingUses = promo.MaxUsesPerUser - userUsageCount
	} else {
		// Guest user
		userRemainingUses = promo.MaxUsesPerUser
	}

	// Step 5: Check minimum order amount
	if req.Subtotal.LessThan(promo.MinOrderAmount) {
		return nil, &model.AppError{
			Code:       model.ErrCodePromoMinOrderNotMet,
			Message:    fmt.Sprintf("Đơn hàng chưa đạt giá trị tối thiểu %s VND", promo.MinOrderAmount.String()),
			HTTPStatus: 400,
			Details: map[string]interface{}{
				"min_order_amount": promo.MinOrderAmount,
				"current_subtotal": req.Subtotal,
				"needed_amount":    promo.MinOrderAmount.Sub(req.Subtotal),
			},
		}
	}

	// Step 6: Check first order only
	// if promo.FirstOrderOnly && req.UserID != nil && *req.UserID != uuid.Nil {
	// 	completedCount, err := s.orderRepo.GetCompletedOrderCount(ctx, *req.UserID)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("get completed order count: %w", err)
	// 	}

	// 	if completedCount > 0 {
	// 		return nil, &model.AppError{
	// 			Code:       model.ErrCodePromoFirstOrderOnly,
	// 			Message:    "Mã giảm giá chỉ dành cho đơn hàng đầu tiên",
	// 			HTTPStatus: 400,
	// 			Details: map[string]interface{}{
	// 				"completed_orders": completedCount,
	// 			},
	// 		}
	// 	}
	// }

	// Step 7: Check category applicability
	// Logic: Nếu applicable_category_ids không rỗng, ít nhất 1 item phải match
	if len(promo.ApplicableCategoryIDs) > 0 {
		hasApplicable := false

		for _, item := range req.CartItems {
			if containsUUID(promo.ApplicableCategoryIDs, item.CategoryID) {
				hasApplicable = true
				break
			}
		}

		if !hasApplicable {
			return nil, &model.AppError{
				Code:       model.ErrCodePromoCategoryNotApplicable,
				Message:    "Mã giảm giá không áp dụng cho sản phẩm trong giỏ hàng",
				HTTPStatus: 400,
				Details: map[string]interface{}{
					"applicable_categories": promo.ApplicableCategoryIDs,
				},
			}
		}
	}

	// Step 8: Calculate discount
	discountAmount := s.calculator.Calculate(promo, req.Subtotal)
	finalAmount := req.Subtotal.Sub(discountAmount)

	// Step 9: Build validation result
	result := &model.ValidationResult{
		IsValid: true,
		Promotion: &model.PromotionInfo{
			ID:                promo.ID,
			Code:              promo.Code,
			Name:              promo.Name,
			Description:       promo.Description,
			DiscountType:      string(promo.DiscountType),
			DiscountValue:     promo.DiscountValue,
			MaxDiscountAmount: promo.MaxDiscountAmount,
			MinOrderAmount:    promo.MinOrderAmount,
			ExpiresAt:         promo.ExpiresAt,
		},
		DiscountAmount:      discountAmount,
		FinalAmount:         finalAmount,
		Message:             "Mã giảm giá áp dụng thành công",
		RemainingGlobalUses: promo.RemainingUses(),
		UserRemainingUses:   userRemainingUses,
	}

	return result, nil
}

// ListActivePromotions lấy danh sách promotion active (Public API)
func (s *promotionService) ListActivePromotions(ctx context.Context, categoryID *uuid.UUID, page, limit int) ([]*model.Promotion, int, error) {
	return s.repo.ListActive(ctx, categoryID, page, limit)
}

// GetAvailablePromotionsForCart lấy danh sách promotions có thể áp dụng cho cart
//
// Business Logic Flow:
// 1. Get cart info by cartID and verify ownership
// 2. Get all active promotions
// 3. Filter promotions based on:
//   - is_active = true
//   - Valid time window (starts_at <= now <= expires_at)
//   - min_order_amount <= cart.subtotal
//   - Usage limit not reached
//
// 4. Convert to AvailablePromotionResponse and return
func (s *promotionService) GetAvailablePromotionsForCart(
	ctx context.Context,
	cartID uuid.UUID,
	userID uuid.UUID,
) ([]*model.AvailablePromotionResponse, error) {
	// Step 1: Get cart info and verify ownership
	cartInfo, err := s.cart.GetOrCreateCart(ctx, &userID, nil)
	if err != nil {
		return nil, fmt.Errorf("get cart: %w", err)
	}

	// Verify cart ID matches
	if cartInfo.ID != cartID {
		return nil, fmt.Errorf("cart not found or access denied")
	}

	// Step 2: Get all active promotions (limit 100 for now)
	allPromotions, _, err := s.repo.ListActive(ctx, nil, 1, 100)
	if err != nil {
		return nil, fmt.Errorf("list active promotions: %w", err)
	}

	// Step 3: Filter promotions
	var availablePromotions []*model.AvailablePromotionResponse
	now := time.Now()

	for _, promo := range allPromotions {
		// Check is_active
		if !promo.IsActive {
			continue
		}

		// Check valid time window
		if !promo.IsValidTimeWindow() {
			continue
		}

		// Check if not expired or not started
		if now.Before(promo.StartsAt) || now.After(promo.ExpiresAt) {
			continue
		}

		// Check usage limit not reached
		if promo.IsUsageLimitReached() {
			continue
		}

		// Check minimum order amount
		if cartInfo.Subtotal.LessThan(promo.MinOrderAmount) {
			continue
		}

		// Promotion is available, add to result
		availablePromotions = append(availablePromotions, &model.AvailablePromotionResponse{
			Code:              promo.Code,
			Name:              promo.Name,
			Description:       promo.Description,
			DiscountType:      string(promo.DiscountType),
			DiscountValue:     promo.DiscountValue,
			MaxDiscountAmount: promo.MaxDiscountAmount,
			MinOrderAmount:    promo.MinOrderAmount,
			ExpiresAt:         promo.ExpiresAt,
		})
	}

	return availablePromotions, nil
}

// -------------------------------------------------------------------
// ADMIN METHODS
// -------------------------------------------------------------------

// CreatePromotion tạo promotion mới
//
// Validation:
// - Code unique (case-insensitive)
// - Expires_at > starts_at
// - Discount value hợp lệ theo type
// - Category IDs tồn tại (TODO: validate với category service)
func (s *promotionService) CreatePromotion(ctx context.Context, req *model.CreatePromotionRequest) (*model.Promotion, error) {
	// Normalize code
	req.NormalizeCode()

	// Check code exists
	exists, err := s.repo.CheckCodeExists(ctx, req.Code, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, &model.AppError{
			Code:       model.ErrCodePromoDuplicateCode,
			Message:    fmt.Sprintf("Mã khuyến mãi '%s' đã tồn tại", req.Code),
			HTTPStatus: 400,
		}
	}

	// Parse timestamps
	startsAt, err := time.Parse(time.RFC3339, req.StartsAt)
	if err != nil {
		return nil, &model.AppError{
			Code:       model.ErrCodeValidationFailed,
			Message:    "Định dạng thời gian bắt đầu không hợp lệ",
			HTTPStatus: 400,
		}
	}

	expiresAt, err := time.Parse(time.RFC3339, req.ExpiresAt)
	if err != nil {
		return nil, &model.AppError{
			Code:       model.ErrCodeValidationFailed,
			Message:    "Định dạng thời gian kết thúc không hợp lệ",
			HTTPStatus: 400,
		}
	}

	// Validate expires_at > starts_at
	if !expiresAt.After(startsAt) {
		return nil, &model.AppError{
			Code:       model.ErrCodeValidationFailed,
			Message:    "Thời gian kết thúc phải sau thời gian bắt đầu",
			HTTPStatus: 400,
		}
	}
	max_discount_amount := decimal.NewFromFloat(*req.MaxDiscountAmount)
	// Build promotion model
	promo := &model.Promotion{
		Code:                  req.Code,
		Name:                  req.Name,
		Description:           req.Description,
		DiscountType:          model.DiscountType(req.DiscountType),
		DiscountValue:         decimal.NewFromFloat(req.DiscountValue),
		MaxDiscountAmount:     &max_discount_amount,
		MinOrderAmount:        decimal.NewFromFloat(req.MinOrderAmount),
		ApplicableCategoryIDs: req.ApplicableCategoryIDs,
		FirstOrderOnly:        req.FirstOrderOnly,
		MaxUses:               req.MaxUses,
		MaxUsesPerUser:        req.MaxUsesPerUser,
		StartsAt:              startsAt,
		ExpiresAt:             expiresAt,
		IsActive:              req.IsActive,
	}

	// Create in DB
	err = s.repo.Create(ctx, promo)
	if err != nil {
		return nil, err
	}

	return promo, nil
}

// GetPromotionByID lấy chi tiết promotion với stats
func (s *promotionService) GetPromotionByID(ctx context.Context, id uuid.UUID) (*model.PromotionDetailResponse, error) {
	promo, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Get usage stats
	stats, err := s.repo.GetUsageStats(ctx, id, nil, nil)
	if err != nil {
		// Continue without stats
	}

	// Calculate usage rate
	var usageRate *float64
	if promo.MaxUses != nil && *promo.MaxUses > 0 {
		rate := float64(promo.CurrentUses) / float64(*promo.MaxUses) * 100
		usageRate = &rate
	}

	response := &model.PromotionDetailResponse{
		ID:                    promo.ID,
		Code:                  promo.Code,
		Name:                  promo.Name,
		Description:           promo.Description,
		DiscountType:          string(promo.DiscountType),
		DiscountValue:         promo.DiscountValue,
		MaxDiscountAmount:     promo.MaxDiscountAmount,
		MinOrderAmount:        promo.MinOrderAmount,
		ApplicableCategoryIDs: promo.ApplicableCategoryIDs,
		FirstOrderOnly:        promo.FirstOrderOnly,
		MaxUses:               promo.MaxUses,
		MaxUsesPerUser:        promo.MaxUsesPerUser,
		CurrentUses:           promo.CurrentUses,
		UsageRate:             usageRate,
		StartsAt:              promo.StartsAt,
		ExpiresAt:             promo.ExpiresAt,
		IsActive:              promo.IsActive,
		Version:               promo.Version,
		CreatedAt:             promo.CreatedAt,
		UpdatedAt:             promo.UpdatedAt,
		Stats:                 stats,
	}

	return response, nil
}

// ListPromotions lấy danh sách promotion với filter (Admin)
func (s *promotionService) ListPromotions(ctx context.Context, filter *model.ListPromotionsFilter) ([]*model.PromotionListItem, int, error) {
	return s.repo.ListAdmin(ctx, filter)
}

// UpdatePromotionStatus cập nhật is_active
func (s *promotionService) UpdatePromotionStatus(ctx context.Context, id uuid.UUID, isActive bool) error {
	err := s.repo.UpdateStatus(ctx, id, isActive)
	if err != nil {
		return err
	}

	return nil
}

// DeletePromotion xóa promotion (soft delete)
func (s *promotionService) DeletePromotion(ctx context.Context, id uuid.UUID) error {
	return s.repo.SoftDelete(ctx, id)
}

// GetUsageHistory lấy lịch sử sử dụng promotion
func (s *promotionService) GetUsageHistory(
	ctx context.Context,
	promoID uuid.UUID,
	startDate, endDate *time.Time,
	userID *uuid.UUID,
	page, limit int,
) (*model.UsageHistoryResponse, error) {
	// Get promotion info
	promo, err := s.repo.FindByID(ctx, promoID)
	if err != nil {
		return nil, err
	}

	// Get stats
	stats, err := s.repo.GetUsageStats(ctx, promoID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Get usage history
	usages, _, err := s.repo.GetUsageHistory(ctx, promoID, startDate, endDate, userID, page, limit)
	if err != nil {
		return nil, err
	}

	// Convert to DTO
	usageItems := make([]model.PromotionUsageDetailItem, len(usages))
	for i, u := range usages {
		usageItems[i] = model.PromotionUsageDetailItem{
			ID: u.ID,
			User: model.UserInfo{
				ID:       u.UserID,
				Email:    u.UserEmail,
				FullName: u.UserFullName,
			},
			Order: model.OrderInfo{
				ID:          u.OrderID,
				OrderNumber: u.OrderNumber,
				Subtotal:    u.OrderTotal,
				Status:      u.OrderStatus,
			},
			DiscountAmount: u.DiscountAmount,
			UsedAt:         u.UsedAt,
		}
	}

	response := &model.UsageHistoryResponse{
		Promotion: model.PromotionInfo{
			ID:   promo.ID,
			Code: promo.Code,
			Name: promo.Name,
		},
		Statistics:   *stats,
		UsageHistory: usageItems,
	}

	return response, nil
}

// -------------------------------------------------------------------
// INTERNAL METHODS (Called by Order Service)
// -------------------------------------------------------------------

// RecordUsage ghi lại việc sử dụng promotion (called khi payment success)
//
// Important:
// - Phải gọi trong transaction của Order Service
// - Trigger DB sẽ tự động increment current_uses
// - Không rollback nếu order bị cancel (theo spec)
func (s *promotionService) RecordUsage(
	ctx context.Context,
	orderID, promoID, userID uuid.UUID,
	discountAmount interface{},
) error {
	// ĐÚNG: BeginTx → trả về *pgxpool.Tx
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	// KHÔNG dùng defer tx.Rollback() ở đây nếu muốn commit
	// → defer chỉ rollback khi error

	usage := &model.PromotionUsage{
		PromotionID:    promoID,
		UserID:         userID,
		OrderID:        orderID,
		DiscountAmount: discountAmount.(decimal.Decimal),
	}

	if err := s.repo.CreateUsage(ctx, tx, usage); err != nil {
		tx.Rollback(ctx) // rollback ngay khi lỗi
		return fmt.Errorf("create usage failed: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	return nil
}

// -------------------------------------------------------------------
// HELPER FUNCTIONS
// -------------------------------------------------------------------

// containsUUID kiểm tra UUID có trong slice không
func containsUUID(slice []uuid.UUID, item uuid.UUID) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}
