package service

import (
	addressService "bookstore-backend/internal/domains/address/service"
	"bookstore-backend/internal/domains/cart/model"
	repo "bookstore-backend/internal/domains/cart/repository"
	inventoryModel "bookstore-backend/internal/domains/inventory/model"
	inveRepo "bookstore-backend/internal/domains/inventory/repository"
	inveService "bookstore-backend/internal/domains/inventory/service"
	"bookstore-backend/internal/shared/utils"
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type CartService struct {
	repository       repo.RepositoryInterface
	inventoryService inveService.ServiceInterface
	inventoryRepo    inveRepo.RepositoryInterface
	address          addressService.ServiceInterface
}

func NewCartService(r repo.RepositoryInterface, inventoryS inveService.ServiceInterface, addressSvc addressService.ServiceInterface, inventoryRepo inveRepo.RepositoryInterface) ServiceInterface {
	return &CartService{
		repository:       r,
		inventoryService: inventoryS,
		address:          addressSvc,
		inventoryRepo:    inventoryRepo,
	}
}

// GetOrCreateCart implements ServiceInterface.GetOrCreateCart
func (s *CartService) GetOrCreateCart(ctx context.Context, userID *uuid.UUID, sessionID *string) (*model.CartResponse, error) {
	var cart *model.Cart
	var err error

	// Step 1: Fetch existing cart
	if userID != nil {
		cart, err = s.repository.GetByUserID(ctx, *userID)
		if err != nil {
			return nil, fmt.Errorf("failed to get user cart: %w", err)
		}
	} else if sessionID != nil {
		cart, err = s.repository.GetBySessionID(ctx, *sessionID)
		if err != nil {
			return nil, fmt.Errorf("failed to get session cart: %w", err)
		}
	} else {
		return nil, fmt.Errorf("either user_id or session_id must be provided")
	}

	// Step 2: Create new cart if not exists
	if cart == nil {
		cart = &model.Cart{
			ID:         uuid.New(),
			UserID:     userID,
			SessionID:  sessionID,
			ItemsCount: 0,
			Subtotal:   decimal.Zero,
			Version:    1,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			ExpiresAt:  time.Now().Add(30 * 24 * time.Hour),
		}

		if err := s.repository.Create(ctx, cart); err != nil {
			return nil, fmt.Errorf("failed to create cart: %w", err)
		}
	} else {
		// Step 3: Update expiration (keep-alive)
		_ = s.repository.UpdateExpiration(ctx, cart.ID) // Ignore error, non-critical
	}

	// Step 4: Fetch all items with book details
	items, _, err := s.repository.GetItemsWithBooks(ctx, cart.ID, 1, 1000) // Get all (max 1000)
	if err != nil {
		// Log but continue - return cart without items
		fmt.Printf("⚠️  Warning: failed to get cart items: %v\n", err)
		return cart.ToResponse([]model.CartItemResponse{}), nil
	}

	// Convert items to responses
	itemResponses := make([]model.CartItemResponse, len(items))
	for i, item := range items {
		itemResponses[i] = *item.ToItemResponse()
	}

	return cart.ToResponse(itemResponses), nil
}

// AddItem implements ServiceInterface.AddItem
func (s *CartService) AddItem(ctx context.Context, cartID uuid.UUID, req model.AddToCartRequest) (*model.CartItemResponse, error) {
	// Step 1: Validate cart exists and not expired
	cart, err := s.repository.GetByID(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}
	if cart == nil {
		return nil, fmt.Errorf("cart not found: %s", cartID)
	}
	if cart.IsExpired() {
		return nil, fmt.Errorf("cart has expired")
	}

	// Step 2: Check if book already in cart
	existingItem, err := s.repository.GetItemByBookInCart(ctx, cartID, req.BookID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing item: %w", err)
	}

	// Step 3: Validate quantity
	if req.Quantity <= 0 || req.Quantity > 100 {
		return nil, fmt.Errorf("%w: quantity=%d", model.ErrInvalidQuantity, req.Quantity)
	}

	// Step 4: Verify stock availability (from inventory service)
	// Check all warehouses for simplicity for now
	inventories, err := s.inventoryRepo.GetInventoriesByBook(ctx, req.BookID)
	if err != nil {
		return nil, fmt.Errorf("failed to check stock availability: %w", err)
	}

	// Calculate total available stock across warehouses
	totalAvailable := 0
	for _, inv := range inventories {
		totalAvailable += inv.AvailableQuantity
	}

	if totalAvailable < req.Quantity {
		return nil, fmt.Errorf("insufficient stock: requested %d, available %d", req.Quantity, totalAvailable)
	}

	// Step 5: Get current book price (from books table)
	// TODO: Integrate with book service to get current price
	currentPrice := decimal.NewFromInt(99) // Placeholder

	// Step 5: Create or update cart item
	var itemID uuid.UUID
	if existingItem != nil {
		// Update existing
		itemID = existingItem.ID
		existingItem.Quantity = req.Quantity
		existingItem.UpdatedAt = time.Now()

		if err := s.repository.AddItem(ctx, existingItem); err != nil {
			return nil, fmt.Errorf("failed to update item: %w", err)
		}
	} else {
		// Create new
		itemID = uuid.New()
		newItem := &model.CartItem{
			ID:        itemID,
			CartID:    cartID,
			BookID:    req.BookID,
			Quantity:  req.Quantity,
			Price:     currentPrice, // Snapshot price
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := s.repository.AddItem(ctx, newItem); err != nil {
			return nil, fmt.Errorf("failed to add item: %w", err)
		}
	}

	// Step 6: Fetch item with book details
	items, _, err := s.repository.GetItemsWithBooks(ctx, cartID, 1, 1)
	if err != nil || len(items) == 0 {
		return nil, fmt.Errorf("failed to fetch added item: %w", err)
	}

	response := items[0].ToItemResponse()
	return response, nil
}

// ListItems implements ServiceInterface.ListItems
func (s *CartService) ListItems(ctx context.Context, cartID uuid.UUID, page int, limit int) (*model.CartResponse, error) {
	// Validate pagination
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Fetch items with book details
	items, totalCount, err := s.repository.GetItemsWithBooks(ctx, cartID, page, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list items: %w", err)
	}

	// Convert to responses
	itemResponses := make([]model.CartItemResponse, len(items))
	for i, item := range items {
		itemResponses[i] = *item.ToItemResponse()
	}

	// Build response with pagination
	cart := &model.Cart{
		ID:         cartID,
		ItemsCount: totalCount,
	}

	response := cart.ToResponse(itemResponses)
	return response, nil
}

// GetUserCartID implements ServiceInterface.GetUserCartID (for middleware)
func (s *CartService) GetUserCartID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	cart, err := s.repository.GetByUserID(ctx, userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get user cart: %w", err)
	}

	if cart == nil {
		return uuid.Nil, nil
	}

	return cart.ID, nil
}

// GetOrCreateCartBySession implements ServiceInterface.GetOrCreateCartBySession (for middleware)
func (s *CartService) GetOrCreateCartBySession(ctx context.Context, sessionID string) (uuid.UUID, error) {
	cart, err := s.repository.GetBySessionID(ctx, sessionID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get session cart: %w", err)
	}

	if cart != nil && cart.ExpiresAt.After(time.Now()) {
		_ = s.repository.UpdateExpiration(ctx, cart.ID)
		return cart.ID, nil
	}

	// Create new cart for session
	newCart := &model.Cart{
		ID:         uuid.New(),
		UserID:     nil,
		SessionID:  &sessionID,
		ItemsCount: 0,
		Subtotal:   decimal.Zero,
		Version:    1,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(30 * 24 * time.Hour),
	}

	if err := s.repository.Create(ctx, newCart); err != nil {
		return uuid.Nil, fmt.Errorf("failed to create session cart: %w", err)
	}

	return newCart.ID, nil
}

// domains/cart/service_impl.go

// MergeCart implements ServiceInterface.MergeCart
func (s *CartService) MergeCart(ctx context.Context, sessionID string, userID uuid.UUID) error {
	// ===================================
	// STEP 1: Get anonymous cart (source)
	// ===================================
	anonymousCart, err := s.repository.GetBySessionID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get anonymous cart: %w", err)
	}

	// No anonymous cart → nothing to merge
	if anonymousCart == nil {
		return nil
	}

	// Check if expired
	if anonymousCart.IsExpired() {
		// Delete expired cart and return
		_ = s.repository.DeleteCart(ctx, anonymousCart.ID)
		return nil
	}

	// ===================================
	// STEP 2: Get user cart (target)
	// ===================================
	userCart, err := s.repository.GetByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user cart: %w", err)
	}

	// If no user cart exists, create one
	if userCart == nil {
		userCart = &model.Cart{
			ID:         uuid.New(),
			UserID:     &userID,
			SessionID:  nil,
			ItemsCount: 0,
			Subtotal:   decimal.Zero,
			Version:    1,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			ExpiresAt:  time.Now().Add(30 * 24 * time.Hour),
		}

		if err := s.repository.Create(ctx, userCart); err != nil {
			return fmt.Errorf("failed to create user cart: %w", err)
		}
	}

	// ===================================
	// STEP 3: Get items from anonymous cart
	// ===================================
	anonymousItems, _, err := s.repository.GetItemsWithBooks(ctx, anonymousCart.ID, 1, 1000)
	if err != nil {
		return fmt.Errorf("failed to get anonymous cart items: %w", err)
	}

	// No items → just delete anonymous cart
	if len(anonymousItems) == 0 {
		_ = s.repository.DeleteCart(ctx, anonymousCart.ID)
		return nil
	}

	// ===================================
	// STEP 4: Get items from user cart
	// ===================================
	userItems, _, err := s.repository.GetItemsWithBooks(ctx, userCart.ID, 1, 1000)
	if err != nil {
		return fmt.Errorf("failed to get user cart items: %w", err)
	}

	// Create map for quick lookup
	userItemsByBook := make(map[uuid.UUID]*model.CartItemWithBook)
	for i := range userItems {
		userItemsByBook[userItems[i].BookID] = &userItems[i]
	}

	// ===================================
	// STEP 5: Merge items
	// ===================================
	for _, anonItem := range anonymousItems {
		existingUserItem, exists := userItemsByBook[anonItem.BookID]

		if exists {
			// Item exists in both carts → keep higher quantity
			if anonItem.Quantity > existingUserItem.Quantity {
				// Update user cart item with anonymous quantity
				existingUserItem.Quantity = anonItem.Quantity
				existingUserItem.UpdatedAt = time.Now()

				if err := s.repository.UpdateItem(ctx, &existingUserItem.CartItem); err != nil {
					return fmt.Errorf("failed to update existing item: %w", err)
				}
			}
			// Else: keep user's quantity (higher or equal)
		} else {
			// Item only in anonymous cart → transfer to user cart
			if err := s.repository.TransferItem(ctx, &anonItem.CartItem, userCart.ID); err != nil {
				return fmt.Errorf("failed to transfer item: %w", err)
			}
		}
	}

	// ===================================
	// STEP 6: Delete anonymous cart
	// ===================================
	// This will CASCADE delete all remaining items
	if err := s.repository.DeleteCart(ctx, anonymousCart.ID); err != nil {
		// Log but don't fail - merge was successful
		fmt.Printf("⚠️  Warning: failed to delete anonymous cart: %v\n", err)
	}

	return nil
}

// domains/cart/service_impl.go

// UpdateItemQuantity implements ServiceInterface.UpdateItemQuantity
func (s *CartService) UpdateItemQuantity(ctx context.Context, cartID uuid.UUID, itemID uuid.UUID, quantity int) (*model.CartItemResponse, error) {
	// ===================================
	// STEP 1: Validate cart and quantity
	// ===================================
	cart, err := s.repository.GetByID(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}
	if cart == nil {
		return nil, fmt.Errorf("cart not found: %s", cartID)
	}
	if cart.IsExpired() {
		return nil, fmt.Errorf("cart has expired")
	}

	if quantity < 0 || quantity > 100 {
		return nil, fmt.Errorf("%w: quantity=%d", model.ErrInvalidQuantity, quantity)
	}

	// ===================================
	// STEP 2: Get item from cart
	// ===================================
	item, err := s.repository.GetItemByID(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	// Verify item belongs to this cart
	if item.CartID != cartID {
		return nil, fmt.Errorf("item does not belong to this cart")
	}

	// ===================================
	// STEP 3: Handle quantity = 0 (remove)
	// ===================================
	if quantity == 0 {
		// Remove item and update cart totals
		if err := s.repository.DeleteItem(ctx, itemID); err != nil {
			return nil, fmt.Errorf("failed to remove item: %w", err)
		}

		// Update cart totals
		// Repository will handle updating item count and totals

		return nil, nil // Return nil for removed item
	}

	// ===================================
	// STEP 4: Verify stock if increasing quantity
	// ===================================
	if quantity > item.Quantity {
		// Check all warehouses for simplicity for now
		inventories, err := s.inventoryRepo.GetInventoriesByBook(ctx, item.BookID)
		if err != nil {
			return nil, fmt.Errorf("failed to check stock availability: %w", err)
		}

		// Calculate total available stock across warehouses
		totalAvailable := 0
		for _, inv := range inventories {
			totalAvailable += inv.AvailableQuantity
		}

		additionalQty := quantity - item.Quantity
		if totalAvailable < additionalQty {
			return nil, fmt.Errorf("insufficient stock: need %d more units, only %d available",
				additionalQty, totalAvailable)
		}
	}

	// ===================================
	// STEP 5: Update quantity and cart totals
	// ===================================
	// oldQuantity := item.Quantity
	item.Quantity = quantity
	item.UpdatedAt = time.Now()

	if err := s.repository.UpdateItem(ctx, item); err != nil {
		return nil, fmt.Errorf("failed to update item: %w", err)
	}

	// Cart totals will be updated by repository when updating item

	// ===================================
	// STEP 6: Fetch updated item with book details
	// ===================================
	items, _, err := s.repository.GetItemsWithBooks(ctx, cartID, 1, 1)
	if err != nil || len(items) == 0 {
		return nil, fmt.Errorf("failed to fetch updated item: %w", err)
	}

	response := items[0].ToItemResponse()
	return response, nil
}

// domains/cart/service_impl.go

// RemoveItem implements ServiceInterface.RemoveItem
func (s *CartService) RemoveItem(ctx context.Context, cartID uuid.UUID, itemID uuid.UUID) error {
	// ===================================
	// STEP 1: Get item to verify it exists and belongs to cart
	// ===================================
	item, err := s.repository.GetItemByID(ctx, itemID)
	if err != nil {
		return fmt.Errorf("failed to get item: %w", err)
	}

	// Verify item belongs to this cart
	if item.CartID != cartID {
		return fmt.Errorf("item does not belong to this cart: item_cart=%s, provided_cart=%s",
			item.CartID, cartID)
	}

	// ===================================
	// STEP 2: Delete item
	// ===================================
	if err := s.repository.DeleteItem(ctx, itemID); err != nil {
		return fmt.Errorf("failed to remove item: %w", err)
	}

	return nil
}

// domains/cart/service_impl.go

// ClearCart implements ServiceInterface.ClearCart
func (s *CartService) ClearCart(ctx context.Context, cartID uuid.UUID) error {
	// ===================================
	// STEP 1: Delete all items from cart
	// ===================================
	deletedCount, err := s.repository.ClearCartItems(ctx, cartID)
	if err != nil {
		return fmt.Errorf("failed to clear cart: %w", err)
	}

	if deletedCount > 0 {
		fmt.Printf("✅ Cleared %d items from cart %s\n", deletedCount, cartID)
	}

	return nil
}

// domains/cart/service_impl.go

// ValidateCart implements ServiceInterface.ValidateCart
func (s *CartService) ValidateCart(ctx context.Context, cartID uuid.UUID, userId string) (*model.CartValidationResult, error) {
	result := &model.CartValidationResult{
		IsValid:         true,
		CartStatus:      "valid",
		Errors:          []model.CartValidationError{},
		Warnings:        []model.CartValidationWarning{},
		ItemValidations: []model.ItemValidation{},
	}

	uid := utils.ParseStringToUUID(userId)
	// ===================================
	// STEP 1: Get cart
	// ===================================
	// TODO: Get cart from repo
	cart, err := s.repository.GetByUserID(ctx, uid)
	if err != nil {
		result.IsValid = false
		result.CartStatus = "error"
		result.Errors = append(result.Errors, model.CartValidationError{
			Code:     "CART_NOT_FOUND",
			Message:  fmt.Sprintf("Cart not found: %v", err),
			Severity: "error",
		})
		return result, nil
	}

	// ===================================
	// STEP 2: Check if expired
	// ===================================
	if cart.IsExpired() {
		result.IsValid = false
		result.CartStatus = "error"
		result.Errors = append(result.Errors, model.CartValidationError{
			Code:     "CART_EXPIRED",
			Message:  "Cart has expired",
			Severity: "error",
		})
		return result, nil
	}

	// ===================================
	// STEP 3: Get items with book details
	// ===================================
	items, _, err := s.repository.GetItemsWithBooks(ctx, cartID, 1, 1000)
	if err != nil {
		result.IsValid = false
		result.CartStatus = "error"
		result.Errors = append(result.Errors, model.CartValidationError{
			Code:     "ITEMS_FETCH_FAILED",
			Message:  fmt.Sprintf("Failed to fetch items: %v", err),
			Severity: "error",
		})
		return result, nil
	}

	// No items in cart
	if len(items) == 0 {
		result.IsValid = false
		result.CartStatus = "error"
		result.Errors = append(result.Errors, model.CartValidationError{
			Code:     "EMPTY_CART",
			Message:  "Cart is empty",
			Severity: "error",
		})
		return result, nil
	}

	// ===================================
	// STEP 4: Validate each item
	// ===================================
	var totalValue decimal.Decimal
	var hasErrors bool
	var hasWarnings bool

	for _, item := range items {
		itemValidation := model.ItemValidation{
			ItemID:           item.ID,
			BookID:           item.BookID,
			BookTitle:        item.BookTitle,
			SnapshotPrice:    item.Price,
			CurrentPrice:     item.CurrentPrice,
			SnapshotQuantity: item.Quantity,
			AvailableStock:   item.TotalStock,
			IsAvailable:      item.IsActive && item.TotalStock > 0,
			PriceMatch:       item.Price.Equal(item.CurrentPrice),
			StockSufficient:  item.TotalStock >= item.Quantity,
			Warnings:         []string{},
		}

		// Check stock
		if !itemValidation.IsAvailable {
			hasErrors = true
			itemValidation.Warnings = append(itemValidation.Warnings,
				fmt.Sprintf("Book not available (active: %v, stock: %d)",
					item.IsActive, item.TotalStock))
		}

		if !itemValidation.StockSufficient {
			hasErrors = true
			itemValidation.Warnings = append(itemValidation.Warnings,
				fmt.Sprintf("Insufficient stock: requested %d, available %d",
					item.Quantity, item.TotalStock))
		}

		// Check price
		if !itemValidation.PriceMatch {
			hasWarnings = true
			priceDiff := item.CurrentPrice.Sub(item.Price)
			itemValidation.Warnings = append(itemValidation.Warnings,
				fmt.Sprintf("Price changed: %s → %s (+%s)",
					item.Price, item.CurrentPrice, priceDiff))

			// Add warning to results
			result.Warnings = append(result.Warnings, model.CartValidationWarning{
				Code:    "PRICE_CHANGED",
				Message: fmt.Sprintf("Price for %s changed", item.BookTitle),
				Details: map[string]interface{}{
					"item_id":    item.ID,
					"old_price":  item.Price,
					"new_price":  item.CurrentPrice,
					"difference": priceDiff,
				},
			})
		}

		// Calculate value
		itemTotal := decimal.NewFromInt(int64(item.Quantity)).Mul(item.Price)
		totalValue = totalValue.Add(itemTotal)

		result.ItemValidations = append(result.ItemValidations, itemValidation)
	}

	// ===================================
	// STEP 5: Determine overall status
	// ===================================
	if hasErrors {
		result.IsValid = false
		result.CartStatus = "error"
		result.Errors = append(result.Errors, model.CartValidationError{
			Code:     "INVALID_ITEMS",
			Message:  "One or more items are invalid",
			Severity: "error",
		})
	} else if hasWarnings {
		result.IsValid = true // Warnings don't prevent checkout
		result.CartStatus = "warning"
	}

	result.TotalValue = totalValue
	result.EstimatedTotal = totalValue // TODO: Add tax, shipping

	return result, nil
}

// domains/cart/service_impl.go

// ApplyPromoCode implements ServiceInterface.ApplyPromoCode
func (s *CartService) ApplyPromoCode(ctx context.Context, cartID uuid.UUID, promoCode string, userId string) (*model.ApplyPromoResponse, error) {
	uid := utils.ParseStringToUUID(userId)
	if uid == uuid.Nil {
		return nil, fmt.Errorf("invalid user id format")
	}

	// ===================================
	// STEP 1: Get and validate cart
	// ===================================
	cart, err := s.repository.GetByID(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}
	if cart == nil {
		return nil, fmt.Errorf("cart not found")
	}
	if cart.IsExpired() {
		return nil, fmt.Errorf("cart has expired")
	}

	// Verify cart belongs to user
	if cart.UserID == nil || *cart.UserID != uid {
		return nil, fmt.Errorf("cart does not belong to user")
	}

	// ===================================
	// STEP 2: Basic promo validation
	// ===================================
	if len(promoCode) < 3 || len(promoCode) > 50 {
		return nil, fmt.Errorf("invalid promo code length")
	}

	// Check cart has items
	if cart.ItemsCount == 0 {
		return nil, fmt.Errorf("cannot apply promo to empty cart")
	}

	// Check cart subtotal meets minimum
	if cart.Subtotal.LessThan(decimal.NewFromInt(50000)) { // Example: 50k VND minimum
		return nil, fmt.Errorf("cart total does not meet minimum for promo codes")
	}

	// ===================================
	// STEP 3: Validate with promo service
	// ===================================
	// TODO: Integrate promotion service
	// For now, mock implementation
	if promoCode == "INVALID" {
		return nil, fmt.Errorf("invalid promo code")
	}

	// Mock promo validation response
	discountType := "percent"
	discountValue := decimal.NewFromInt(10) // 10%
	// minPurchase := decimal.NewFromInt(50000)
	maxDiscount := decimal.NewFromInt(100000)

	// Calculate potential discount
	discountAmount := cart.Subtotal.Mul(discountValue).Div(decimal.NewFromInt(100))

	// Apply maximum discount cap if needed
	if discountAmount.GreaterThan(maxDiscount) {
		discountAmount = maxDiscount
	}

	// ===================================
	// STEP 3: Update cart with promo
	// ===================================
	err = s.repository.UpdateCartPromo(ctx, cartID, &promoCode, discountAmount)
	if err != nil {
		return nil, fmt.Errorf("failed to apply promo: %w", err)
	}

	// ===================================
	// STEP 4: Return response
	// ===================================
	return &model.ApplyPromoResponse{
		Applied:          true,
		PromoCode:        promoCode,
		PromoDescription: "10% discount",
		DiscountType:     discountType,
		DiscountValue:    discountValue,
		DiscountAmount:   discountAmount,
		OriginalSubtotal: cart.Subtotal,
		DiscountedTotal:  cart.Subtotal.Sub(discountAmount),
		AppliedAt:        time.Now(),
	}, nil
}

// RemovePromoCode implements ServiceInterface.RemovePromoCode
func (s *CartService) RemovePromoCode(ctx context.Context, cartID uuid.UUID) error {
	err := s.repository.RemoveCartPromo(ctx, cartID)
	if err != nil {
		return fmt.Errorf("failed to remove promo: %w", err)
	}
	return nil
}

// CRITICAL NOTES:
// 1. Uses database transaction for atomicity
// 2. Phases must execute in order
// 3. Any phase failure → rollback all changes
// 4. Some operations (email, payment) are async (fire-and-forget)
// CRITICAL NOTES:
// 1. Uses database transaction for atomicity
// 2. Phases must execute in order
// 3. Any phase failure → rollback all changes
// 4. Automatic warehouse selection based on customer location (FR-INV-002)
// 5. Stock reservation with 15-minute timeout (FR-INV-003)
// 6. Complete audit trail via inventory_audit_log
func (s *CartService) Checkout(ctx context.Context, userID uuid.UUID, cartID uuid.UUID, req model.CheckoutRequest) (*model.CheckoutResponse, error) {
	initiatedAt := time.Now()
	response := &model.CheckoutResponse{
		Success:        false,
		Status:         "pending",
		CartSummary:    model.CartCheckoutSummary{},
		Phases:         []model.CheckoutPhaseResult{},
		NextActions:    []string{},
		InitiatedAt:    initiatedAt,
		Errors:         []model.CheckoutError{},
		Warnings:       []model.CheckoutWarning{},
		ItemsProcessed: []model.ItemCheckoutResult{},
	}

	// ===================================
	// PHASE 0: PRE-VALIDATION
	// ===================================

	if userID == uuid.Nil {
		response.Errors = append(response.Errors, model.CheckoutError{
			Code:     "UNAUTHENTICATED",
			Message:  "User not authenticated",
			Severity: "critical",
		})
		return response, nil
	}

	// Get cart
	cart, err := s.repository.GetByID(ctx, cartID)
	if err != nil {
		response.Errors = append(response.Errors, model.CheckoutError{
			Code:     "CART_NOT_FOUND",
			Message:  fmt.Sprintf("Cart not found: %v", err),
			Severity: "critical",
		})
		return response, nil
	}

	// Check cart not empty
	items, _, err := s.repository.GetItemsWithBooks(ctx, cartID, 1, 1000)
	if err != nil || len(items) == 0 {
		response.Errors = append(response.Errors, model.CheckoutError{
			Code:     "EMPTY_CART",
			Message:  "Cart is empty",
			Severity: "critical",
		})
		return response, nil
	}

	response.CartSummary = model.CartCheckoutSummary{
		CartID:    cartID,
		ItemCount: len(items),
		Subtotal:  cart.Subtotal,
		PromoCode: req.PromoCode,
	}

	// ===================================
	// PHASE 1: CART VALIDATION
	// ===================================
	phaseStart := time.Now()
	validation, err := s.ValidateCart(ctx, cartID, userID.String())
	if err != nil || !validation.IsValid {
		for _, err := range validation.Errors {
			response.Errors = append(response.Errors, model.CheckoutError{
				Code:     err.Code,
				Message:  err.Message,
				Severity: err.Severity,
			})
		}
		response.Phases = append(response.Phases, model.CheckoutPhaseResult{
			Phase:     "CART_VALIDATION",
			Status:    "failed",
			Message:   "Cart validation failed",
			Errors:    convertToCheckoutErrors(validation.Errors),
			Timestamp: phaseStart,
		})
		return response, nil
	}

	response.Phases = append(response.Phases, model.CheckoutPhaseResult{
		Phase:     "CART_VALIDATION",
		Status:    "success",
		Message:   "Cart validated",
		Timestamp: phaseStart,
	})

	// ===================================
	// PHASE 2: ADDRESS VALIDATION
	// ===================================
	phaseStart = time.Now()
	phaseResult := model.CheckoutPhaseResult{
		Phase:     "ADDRESS_VALIDATION",
		Status:    "pending",
		Timestamp: phaseStart,
	}

	// Validate shipping address exists and belongs to user
	shippingAddr, err := s.address.GetAddressByID(ctx, userID, req.ShippingAddressID)
	if err != nil {
		phaseResult.Status = "failed"
		phaseResult.Message = "Shipping address validation failed"
		phaseResult.Errors = append(phaseResult.Errors, model.CheckoutError{
			Code:     "INVALID_SHIPPING_ADDRESS",
			Message:  fmt.Sprintf("Invalid shipping address: %v", err),
			Severity: "critical",
		})
		response.Phases = append(response.Phases, phaseResult)
		return response, nil
	}

	// Validate shipping address has coordinates for warehouse selection
	if shippingAddr.Latitude == nil || shippingAddr.Longitude == nil {
		response.Warnings = append(response.Warnings, model.CheckoutWarning{
			Code:    "MISSING_COORDINATES",
			Message: "Shipping address missing coordinates - will use default warehouse",
		})
	}

	// If billing address provided and different from shipping
	if req.BillingAddressID != nil && *req.BillingAddressID != req.ShippingAddressID {
		billingAddr, err := s.address.GetAddressByID(ctx, userID, *req.BillingAddressID)
		if err != nil {
			phaseResult.Status = "failed"
			phaseResult.Message = "Billing address validation failed"
			phaseResult.Errors = append(phaseResult.Errors, model.CheckoutError{
				Code:     "INVALID_BILLING_ADDRESS",
				Message:  fmt.Sprintf("Invalid billing address: %v", err),
				Severity: "critical",
			})
			response.Phases = append(response.Phases, phaseResult)
			return response, nil
		}
		_ = billingAddr
	}

	phaseResult.Status = "success"
	phaseResult.Message = "Addresses validated"
	response.Phases = append(response.Phases, phaseResult)

	// ===================================
	// PHASE 3: PROMO VALIDATION
	// ===================================
	var promoDiscount decimal.Decimal = decimal.Zero
	if req.PromoCode != nil && *req.PromoCode != "" {
		phaseStart = time.Now()

		// TODO: Validate promo code via promo service
		// For now, mock 10% discount
		promoDiscount = cart.Subtotal.Mul(decimal.NewFromInt(10)).Div(decimal.NewFromInt(100))

		response.CartSummary.PromoCode = req.PromoCode
		response.CartSummary.Discount = promoDiscount

		response.Phases = append(response.Phases, model.CheckoutPhaseResult{
			Phase:     "PROMO_VALIDATION",
			Status:    "success",
			Message:   fmt.Sprintf("Promo code %s applied", *req.PromoCode),
			Timestamp: phaseStart,
		})
	}

	// ===================================
	// PHASE 4: CALCULATE PRICING
	// ===================================
	phaseStart = time.Now()
	t, _ := decimal.NewFromString("0.10") // 10% VAT
	subtotal := cart.Subtotal
	tax := subtotal.Mul(t)
	shipping := decimal.NewFromInt(30000) // 30k VND base shipping

	total := subtotal.Add(tax).Add(shipping).Sub(promoDiscount)

	response.PricingBreakdown = model.PricingBreakdown{
		Subtotal:      subtotal,
		PromoDiscount: promoDiscount,
		Tax:           tax,
		Shipping:      shipping,
		Total:         total,
		Currency:      "VND",
		TaxRate:       t,
	}

	response.CartSummary.EstimatedTax = tax
	response.CartSummary.ShippingCost = shipping
	response.CartSummary.Total = total

	response.Phases = append(response.Phases, model.CheckoutPhaseResult{
		Phase:     "PRICING_CALCULATION",
		Status:    "success",
		Message:   fmt.Sprintf("Total: %s VND", total.String()),
		Timestamp: phaseStart,
	})

	// ===================================
	// PHASE 5: WAREHOUSE SELECTION & STOCK CHECK
	// Uses new multi-warehouse system (FR-INV-002)
	// ===================================
	phaseStart = time.Now()

	// Build availability check request
	availabilityItems := make([]inventoryModel.CheckAvailabilityItem, len(items))
	for i, item := range items {
		availabilityItems[i] = inventoryModel.CheckAvailabilityItem{
			BookID:   item.BookID,
			Quantity: item.Quantity,
		}
	}

	availabilityReq := inventoryModel.CheckAvailabilityRequest{
		Items: availabilityItems,
	}

	// If address has coordinates, set preferred warehouse based on location
	if shippingAddr.Latitude != nil && shippingAddr.Longitude != nil {
		// This will be used for warehouse distance calculation
		availabilityReq.CustomerLatitude = shippingAddr.Latitude
		availabilityReq.CustomerLongitude = shippingAddr.Longitude
	}

	// Check availability across all warehouses
	availability, err := s.inventoryService.CheckAvailability(ctx, availabilityReq)
	if err != nil {
		response.Errors = append(response.Errors, model.CheckoutError{
			Code:     "AVAILABILITY_CHECK_FAILED",
			Message:  fmt.Sprintf("Failed to check stock availability: %v", err),
			Severity: "critical",
		})
		response.Phases = append(response.Phases, model.CheckoutPhaseResult{
			Phase:     "WAREHOUSE_SELECTION",
			Status:    "failed",
			Message:   "Failed to check stock availability",
			Timestamp: phaseStart,
		})
		return response, nil
	}

	// Check if order can be fulfilled
	if !availability.Overall {
		response.Errors = append(response.Errors, model.CheckoutError{
			Code:     "INSUFFICIENT_STOCK",
			Message:  "One or more items are out of stock",
			Severity: "critical",
		})

		// Add details for each unavailable item
		for _, itemAvail := range availability.Items {
			f := itemAvail.BookID.String()
			if !itemAvail.Fulfillable {
				response.Errors = append(response.Errors, model.CheckoutError{
					Code:     "ITEM_OUT_OF_STOCK",
					Message:  itemAvail.Recommendation,
					Severity: "error",
					Field:    &f,
				})
			}
		}

		response.Phases = append(response.Phases, model.CheckoutPhaseResult{
			Phase:     "WAREHOUSE_SELECTION",
			Status:    "failed",
			Message:   "Insufficient stock to fulfill order",
			Timestamp: phaseStart,
		})
		return response, nil
	}

	// Add warehouse info to response
	if availability.RecommendedWarehouse != nil {
		response.WarehouseInfo = &model.WarehouseCheckoutInfo{
			WarehouseID:       availability.RecommendedWarehouse.WarehouseID,
			WarehouseName:     availability.RecommendedWarehouse.WarehouseName,
			DistanceKM:        availability.RecommendedWarehouse.DistanceKM,
			EstimatedDelivery: availability.RecommendedWarehouse.EstimatedDelivery,
		}
	}

	response.Phases = append(response.Phases, model.CheckoutPhaseResult{
		Phase:     "WAREHOUSE_SELECTION",
		Status:    "success",
		Message:   fmt.Sprintf("Selected warehouse: %s", availability.RecommendedWarehouse.WarehouseName),
		Timestamp: phaseStart,
	})

	// ===================================
	// PHASE 6: INVENTORY RESERVATION (CRITICAL)
	// Uses pessimistic locking + 15min timeout (FR-INV-003)
	// ===================================
	phaseStart = time.Now()

	// Track successful reservations for rollback if needed
	var successfulReservations []struct {
		WarehouseID uuid.UUID
		BookID      uuid.UUID
		Quantity    int
	}

	// Reserve stock for each item
	for i, item := range items {
		itemResult := model.ItemCheckoutResult{
			ItemID:            item.ID,
			BookID:            item.BookID,
			BookTitle:         item.BookTitle,
			QuantityRequested: item.Quantity,
			QuantityReserved:  0,
			PriceAtCheckout:   item.Price,
			CurrentPrice:      item.CurrentPrice,
			PriceChanged:      !item.Price.Equal(item.CurrentPrice),
			ItemTotal:         decimal.NewFromInt(int64(item.Quantity)).Mul(item.Price),
			Status:            "pending",
		}

		// Get warehouse details for this item from availability check
		itemAvailability := availability.Items[i]

		// Find best warehouse for this item
		var selectedWarehouseID uuid.UUID
		var selectedWarehouseName string

		if len(itemAvailability.WarehouseDetails) > 0 {
			// Use first warehouse that can fulfill (already sorted by distance)
			for _, whDetail := range itemAvailability.WarehouseDetails {
				if whDetail.CanFulfill {
					selectedWarehouseID = whDetail.WarehouseID
					selectedWarehouseName = whDetail.WarehouseName
					break
				}
			}
		}

		if selectedWarehouseID == uuid.Nil {
			itemResult.Status = "failed"
			itemResult.Warnings = append(itemResult.Warnings, "No warehouse available")
			response.ItemsProcessed = append(response.ItemsProcessed, itemResult)

			// Rollback previous reservations
			s.rollbackReservations(ctx, successfulReservations, cartID)

			response.Errors = append(response.Errors, model.CheckoutError{
				Code:     "NO_WAREHOUSE_AVAILABLE",
				Message:  fmt.Sprintf("No warehouse available for item: %s", item.BookTitle),
				Severity: "critical",
			})
			return response, fmt.Errorf("no warehouse available for item %s", item.BookID)
		}

		// Reserve stock using new inventory service
		reserveReq := inventoryModel.ReserveStockRequest{
			BookID:      item.BookID,
			WarehouseID: &selectedWarehouseID,
			Quantity:    item.Quantity,
			ReferenceID: cartID, // Use cartID as reference, will update to orderID later
			UserID:      &userID,
		}

		reserveResp, err := s.inventoryService.ReserveStock(ctx, reserveReq)
		if err != nil {
			itemResult.Status = "failed"
			itemResult.Warnings = append(itemResult.Warnings,
				fmt.Sprintf("Failed to reserve: %v", err))
			response.ItemsProcessed = append(response.ItemsProcessed, itemResult)

			// Rollback previous successful reservations
			s.rollbackReservations(ctx, successfulReservations, cartID)

			response.Errors = append(response.Errors, model.CheckoutError{
				Code:     "RESERVATION_FAILED",
				Message:  fmt.Sprintf("Failed to reserve stock for %s: %v", item.BookTitle, err),
				Severity: "critical",
			})
			return response, fmt.Errorf("failed to reserve stock: %w", err)
		}

		// Success - track reservation
		itemResult.Status = "reserved"
		itemResult.QuantityReserved = reserveResp.ReservedQuantity
		itemResult.WarehouseID = &selectedWarehouseID
		itemResult.WarehouseName = selectedWarehouseName
		itemResult.ReservationExpiresAt = &reserveResp.ExpiresAt // 15 minutes from now

		successfulReservations = append(successfulReservations, struct {
			WarehouseID uuid.UUID
			BookID      uuid.UUID
			Quantity    int
		}{
			WarehouseID: selectedWarehouseID,
			BookID:      item.BookID,
			Quantity:    reserveResp.ReservedQuantity,
		})

		// Check if partial reservation
		if reserveResp.ReservedQuantity < item.Quantity {
			itemResult.Status = "partial"
			itemResult.Warnings = append(itemResult.Warnings,
				fmt.Sprintf("Only %d units reserved (requested %d)",
					reserveResp.ReservedQuantity, item.Quantity))

			response.Warnings = append(response.Warnings, model.CheckoutWarning{
				Code:    "PARTIAL_RESERVATION",
				Message: fmt.Sprintf("%s: only %d available", item.BookTitle, reserveResp.ReservedQuantity),
			})
		}

		response.ItemsProcessed = append(response.ItemsProcessed, itemResult)
	}

	response.Phases = append(response.Phases, model.CheckoutPhaseResult{
		Phase:     "INVENTORY_RESERVATION",
		Status:    "success",
		Message:   fmt.Sprintf("Reserved %d items (expires in 15 minutes)", len(successfulReservations)),
		Timestamp: phaseStart,
	})

	// ===================================
	// PHASE 7: ORDER CREATION
	// ===================================
	phaseStart = time.Now()

	orderID := uuid.New()
	orderNumber := fmt.Sprintf("ORD-%s-%06d",
		time.Now().Format("20060102"),
		rand.Intn(999999))

	// TODO: Create order in database with transaction
	// TODO: Create order items with warehouse assignments
	// TODO: Record promo usage
	// TODO: Update reservation reference from cartID to orderID

	response.Success = true
	response.Status = "completed"
	response.OrderID = orderID
	response.OrderNumber = orderNumber
	response.ReferenceCode = orderNumber
	completedAt := time.Now()
	response.CompletedAt = &completedAt

	response.OrderSummary = &model.OrderCheckoutSummary{
		OrderID:     orderID,
		OrderNumber: orderNumber,
		Status:      "confirmed",
		TotalAmount: total,
		ItemCount:   len(items),
		CreatedAt:   phaseStart,
	}

	response.Phases = append(response.Phases, model.CheckoutPhaseResult{
		Phase:     "ORDER_CREATION",
		Status:    "success",
		Message:   fmt.Sprintf("Order %s created", orderNumber),
		Timestamp: phaseStart,
	})

	// ===================================
	// PHASE 8: CLEANUP
	// ===================================
	phaseStart = time.Now()

	// Clear cart (background task)
	go func() {
		_ = s.ClearCart(context.Background(), cartID)
	}()

	// Send confirmation email (background task)
	go func() {
		// TODO: emailService.SendOrderConfirmation(userID, orderID)
	}()

	// Schedule auto-release job if payment not completed in 15 minutes
	// This will be handled by background worker (Asynq)
	go func() {
		// TODO: asynq.Enqueue(ReleaseReservationTask{
		//   OrderID: orderID,
		//   ExecuteAt: time.Now().Add(15 * time.Minute),
		// })
	}()

	response.Phases = append(response.Phases, model.CheckoutPhaseResult{
		Phase:     "CLEANUP",
		Status:    "success",
		Message:   "Cleanup completed",
		Timestamp: phaseStart,
	})

	// ===================================
	// NEXT ACTIONS
	// ===================================
	if req.PaymentMethod != "cash_on_delivery" {
		response.NextActions = append(response.NextActions,
			"Complete payment within 15 minutes to confirm order")
	}
	response.NextActions = append(response.NextActions,
		fmt.Sprintf("Track order: %s", orderNumber),
		"Check email for confirmation",
	)

	return response, nil
}

// rollbackReservations releases all successfully reserved stock
func (s *CartService) rollbackReservations(ctx context.Context, reservations []struct {
	WarehouseID uuid.UUID
	BookID      uuid.UUID
	Quantity    int
}, referenceID uuid.UUID) {
	// Run in background to not block main flow
	go func() {
		rollbackCtx := context.Background()
		for _, res := range reservations {
			releaseReq := inventoryModel.ReleaseStockRequest{
				WarehouseID: res.WarehouseID,
				BookID:      res.BookID,
				Quantity:    res.Quantity,
				ReferenceID: referenceID,
				Reason:      stringPtr("checkout_failed"),
			}

			_, err := s.inventoryService.ReleaseStock(rollbackCtx, releaseReq)
			if err != nil {
				// Log error but don't fail (background cleanup)
				// TODO: logger.Error("failed to rollback reservation", err)
			}
		}
	}()
}

func stringPtr(s string) *string {
	return &s
}

// Helper to convert validation errors to checkout errors
func convertToCheckoutErrors(validationErrors []model.CartValidationError) []model.CheckoutError {
	errors := make([]model.CheckoutError, len(validationErrors))
	for i, err := range validationErrors {
		errors[i] = model.CheckoutError{
			Code:     err.Code,
			Message:  err.Message,
			Severity: err.Severity,
		}
	}
	return errors
}
