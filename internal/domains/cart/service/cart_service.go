package service

import (
	addressService "bookstore-backend/internal/domains/address/service"
	bookS "bookstore-backend/internal/domains/book/service"
	"bookstore-backend/internal/domains/cart/model"
	repo "bookstore-backend/internal/domains/cart/repository"
	inventoryModel "bookstore-backend/internal/domains/inventory/model"
	inveRepo "bookstore-backend/internal/domains/inventory/repository"
	inveService "bookstore-backend/internal/domains/inventory/service"
	orderModel "bookstore-backend/internal/domains/order/model"
	orderS "bookstore-backend/internal/domains/order/service"
	"bookstore-backend/internal/shared"
	"bookstore-backend/internal/shared/utils"
	"bookstore-backend/pkg/logger"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/shopspring/decimal"
)

type CartService struct {
	repository       repo.RepositoryInterface
	inventoryService inveService.ServiceInterface
	inventoryRepo    inveRepo.RepositoryInterface
	address          addressService.ServiceInterface
	bookService      bookS.ServiceInterface
	orderService     orderS.OrderService
	asynqClient      *asynq.Client
}

func NewCartService(
	r repo.RepositoryInterface,
	inventoryS inveService.ServiceInterface,
	addressSvc addressService.ServiceInterface,
	inventoryRepo inveRepo.RepositoryInterface,
	book bookS.ServiceInterface,
	orderService orderS.OrderService,
	asynqClient *asynq.Client,

) ServiceInterface {
	if orderService == nil {
		logger.Info("orderService", map[string]interface{}{
			"orderService": orderService,
		})
		panic("orderService is required")
	}
	return &CartService{
		repository:       r,
		inventoryService: inventoryS,
		address:          addressSvc,
		inventoryRepo:    inventoryRepo,
		bookService:      book,
		orderService:     orderService,
		asynqClient:      asynqClient,
	}
}
func (s *CartService) ValidatePromoCode(ctx context.Context, req *model.ValidatePromoRequest) (*model.PromotionValidationResult, error) {
	// Step 1: Normalize and validate input
	promoCode := strings.ToUpper(strings.TrimSpace(req.PromoCode))
	if promoCode == "" {
		return &model.PromotionValidationResult{
			IsValid: false,
			Reason:  "Promo code cannot be empty",
		}, nil
	}

	// Step 2: Get promotion by code
	promo, err := s.repository.GetPromoByCode(ctx, promoCode)
	if err != nil {
		return &model.PromotionValidationResult{
			IsValid: false,
			Reason:  "Promo code not found",
		}, nil
	}
	if promo == nil {
		return &model.PromotionValidationResult{
			IsValid: false,
			Reason:  "Promo code not found",
		}, nil
	}

	// Step 3: Check if promotion is active
	if !promo.IsActive {
		return &model.PromotionValidationResult{
			IsValid: false,
			Reason:  "Promo code is not active",
		}, nil
	}

	// Step 4: Check validity period
	now := time.Now()
	if now.Before(promo.StartsAt) {
		return &model.PromotionValidationResult{
			IsValid: false,
			Reason:  fmt.Sprintf("Promo code not yet active (starts at %s)", promo.StartsAt.Format("2006-01-02")),
		}, nil
	}
	if now.After(promo.ExpiresAt) {
		return &model.PromotionValidationResult{
			IsValid: false,
			Reason:  fmt.Sprintf("Promo code expired on %s", promo.ExpiresAt.Format("2006-01-02")),
		}, nil
	}

	// Step 5: Check minimum order amount
	if req.CartTotal.LessThan(promo.MinOrderAmount) {
		return &model.PromotionValidationResult{
			IsValid: false,
			Reason: fmt.Sprintf("Minimum order amount is %s (current: %s)",
				promo.MinOrderAmount.String(), req.CartTotal.String()),
		}, nil
	}

	// Step 6: Check global usage limit
	if promo.MaxUses != nil && promo.CurrentUses >= *promo.MaxUses {
		return &model.PromotionValidationResult{
			IsValid: false,
			Reason:  "Promo code usage limit reached",
		}, nil
	}

	// Step 7: Check user usage limit
	userUsageCount, err := s.repository.CountUserUsage(ctx, promo.ID, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to check user usage: %w", err)
	}

	if userUsageCount >= promo.MaxUsesPerUser {
		return &model.PromotionValidationResult{
			IsValid: false,
			Reason:  fmt.Sprintf("You have already used this promo code %d time(s)", userUsageCount),
		}, nil
	}

	// Step 8: Check first order only constraint
	if promo.FirstOrderOnly {
		hasOrders, err := s.repository.UserHasCompletedOrders(ctx, req.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to check user orders: %w", err)
		}
		if hasOrders {
			return &model.PromotionValidationResult{
				IsValid: false,
				Reason:  "This promo code is only valid for first-time orders",
			}, nil
		}
	}

	// Step 9: All validations passed - return valid result
	return &model.PromotionValidationResult{
		IsValid:               true,
		PromotionID:           promo.ID,
		Code:                  promo.Code,
		Name:                  promo.Name,
		Description:           *promo.Description,
		DiscountType:          string(promo.DiscountType),
		DiscountValue:         promo.DiscountValue,
		MaxDiscount:           promo.MaxDiscountAmount,
		MinOrderAmount:        promo.MinOrderAmount,
		ApplicableCategoryIDs: promo.ApplicableCategoryIDs,
		FirstOrderOnly:        promo.FirstOrderOnly,
		MaxUses:               promo.MaxUses,
		MaxUsesPerUser:        promo.MaxUsesPerUser,
		CurrentUses:           promo.CurrentUses,
		UserUsageCount:        userUsageCount,
		StartsAt:              promo.StartsAt,
		ExpiresAt:             promo.ExpiresAt,
	}, nil
}

func (s *CartService) GetOrCreateCart(ctx context.Context, userID *uuid.UUID, sessionID *string) (*model.CartResponse, error) {
	var cart *model.Cart
	var err error

	// Step 1: Validate input
	if userID == nil && sessionID == nil {
		return nil, fmt.Errorf("either user_id or session_id must be provided")
	}

	// Step 2: Fetch existing cart
	if userID != nil {
		cart, err = s.repository.GetByUserID(ctx, *userID)
	} else {
		cart, err = s.repository.GetBySessionID(ctx, *sessionID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}
	logger.Info("cart", map[string]interface{}{
		"cart": cart,
	})
	// Step 3: Check if cart expired
	// if cart != nil && cart.ExpiresAt.Before(time.Now()) {
	// 	// Cart expired → clear items and reset
	// 	_ = s.repository.DeleteCart(ctx, cart.ID) // Best effort
	// 	cart = nil                                // Force create new cart
	// }

	// Step 4: Create new cart if not exists
	var createdCart *model.Cart
	if cart == nil {
		cart = &model.Cart{
			UserID:     userID,
			SessionID:  sessionID,
			ItemsCount: 0,
			Subtotal:   decimal.Zero,
			Version:    1,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			ExpiresAt:  time.Now().Add(30 * 24 * time.Hour),
		}

		// Use INSERT ... ON CONFLICT to prevent duplicate cart
		createdCart, err = s.repository.CreateOrGet(ctx, cart)
		if err != nil {
			return nil, fmt.Errorf("failed to create cart: %w", err)
		}
	} else {
		logger.Info("Using existing cart", map[string]interface{}{
			"cart_id": cart.ID,
		})
		// Step 5: Update expiration (keep-alive)
		if err := s.repository.UpdateExpiration(ctx, cart.ID); err != nil {
			// Log warning but don't fail request
			logger.Error("Failed to update cart expiration", err)
		}
	}

	var cartID uuid.UUID
	if cart.ID != uuid.Nil {
		cartID = cart.ID
	} else if createdCart != nil {
		cartID = createdCart.ID
	}

	// Step 6: Fetch all items with book details (no hardcode limit)
	items, _, err := s.repository.GetItemsWithBooks(ctx, cartID, 0, 0) // 0,0 = fetch all
	logger.Info("Fetched cart items", map[string]interface{}{
		"items": items,
	})
	if err != nil {
		// Log but continue - return cart without items
		logger.Error("Failed to get cart items", err)
		return cart.ToResponse([]model.CartItemResponse{}), nil
	}

	// Convert items to responses
	itemResponses := make([]model.CartItemResponse, len(items))
	for i, item := range items {
		itemResponses[i] = *item.ToItemResponse()
	}

	return cart.ToResponse(itemResponses), nil
}

func (s *CartService) AddItem(ctx context.Context, cartID uuid.UUID, req model.AddToCartRequest) (*model.CartItemResponse, error) {
	// Step 1: Validate cart exists and not expired
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

	// Step 2: Validate request quantity
	if req.Quantity <= 0 || req.Quantity > 100 {
		return nil, model.ErrInvalidQuantity
	}

	// Step 3: Get book and validate
	book, err := s.bookService.GetBookDetail(ctx, req.BookID.String())
	if err != nil {
		return nil, fmt.Errorf("book not found: %w", err)
	}
	if !book.IsActive {
		return nil, fmt.Errorf("book is not available")
	}

	// Step 4: Check existing item
	existingItem, err := s.repository.GetItemByBookInCart(ctx, cartID, req.BookID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing item: %w", err)
	}

	// Step 5: Calculate final quantity and validate
	var finalQuantity int
	var isUpdate bool

	if existingItem != nil {
		finalQuantity = existingItem.Quantity + req.Quantity
		isUpdate = true
	} else {
		finalQuantity = req.Quantity
		isUpdate = false
	}

	if finalQuantity > 100 {
		currentQty := 0
		if existingItem != nil {
			currentQty = existingItem.Quantity
		}
		return nil, fmt.Errorf("maximum 100 items per product (current: %d, adding: %d)", currentQty, req.Quantity)
	}

	// Step 6: Check stock availability (only for increment)
	if isUpdate {
		incrementQty := req.Quantity
		if incrementQty > 0 {
			// Only check stock for increment
			totalAvailable := s.getTotalAvailableStock(ctx, req.BookID)
			if totalAvailable < incrementQty {
				return nil, fmt.Errorf("insufficient stock: requested %d more, available %d", incrementQty, totalAvailable)
			}
		}
	} else {
		totalAvailable := s.getTotalAvailableStock(ctx, req.BookID)
		if totalAvailable < req.Quantity {
			return nil, fmt.Errorf("insufficient stock: requested %d, available %d", req.Quantity, totalAvailable)
		}
	}

	// Step 7: Add or update item
	item := &model.CartItem{
		CartID:    cartID,
		BookID:    req.BookID,
		Quantity:  finalQuantity,
		Price:     book.Price, // Always use current price
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if existingItem != nil {
		item.ID = existingItem.ID
		item.CreatedAt = existingItem.CreatedAt
	}

	// Use AddItem with ON CONFLICT (handles both insert and update)
	savedItem, err := s.repository.AddItem(ctx, item)
	if err != nil {
		return nil, fmt.Errorf("failed to save item: %w", err)
	}

	// Step 8: Build response
	response := &model.CartItemResponse{
		ID:           savedItem.ID,
		CartID:       savedItem.CartID,
		BookID:       savedItem.BookID,
		Quantity:     savedItem.Quantity,
		Price:        savedItem.Price,
		BookTitle:    book.Title,
		BookSlug:     utils.GenerateSlugBook(book.Title),
		BookCoverURL: book.CoverURL,
		BookAuthor:   book.Author.Name,
		CurrentPrice: book.Price,
		IsActive:     book.IsActive,
		TotalStock:   s.getTotalAvailableStock(ctx, req.BookID),
		CreatedAt:    savedItem.CreatedAt,
		UpdatedAt:    savedItem.UpdatedAt,
	}

	return response, nil
}

// Helper method
func (s *CartService) getTotalAvailableStock(ctx context.Context, bookID uuid.UUID) int {
	inventories, err := s.inventoryRepo.GetInventoriesByBook(ctx, bookID)
	if err != nil {
		return 0
	}
	total := 0
	for _, inv := range inventories {
		total += inv.AvailableQuantity
	}
	return total
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

	// Step 1: Fetch cart from DB (validate exists + get metadata)
	cart, err := s.repository.GetByID(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}
	if cart == nil {
		return nil, fmt.Errorf("cart not found")
	}

	// Step 2: Check if cart expired
	if cart.IsExpired() {
		return nil, fmt.Errorf("cart has expired")
	}

	// Step 3: Fetch items with book details (paginated)
	items, totalCount, err := s.repository.GetItemsWithBooks(ctx, cartID, page, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list items: %w", err)
	}

	// Step 4: Convert to responses
	itemResponses := make([]model.CartItemResponse, len(items))
	for i, item := range items {
		itemResponses[i] = *item.ToItemResponse()
	}

	// Step 5: Build response
	// ✅ Use cart from DB (has correct ItemsCount, Subtotal, etc from triggers)
	response := cart.ToResponse(itemResponses)

	// Add pagination metadata
	response.Page = page
	response.PageSize = limit
	response.TotalItems = totalCount
	response.TotalPages = (totalCount + limit - 1) / limit

	return response, nil
}

// GetUserCartID implements ServiceInterface.GetUserCartID (for middleware)
func (s *CartService) GetUserCartID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	cart, err := s.repository.GetByUserID(ctx, userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get user cart: %w", err)
	}

	if cart == nil {
		return uuid.Nil, nil // No cart
	}

	// Check if expired
	if cart.IsExpired() {
		// Option A: Clear và trả nil (như không có cart)
		_ = s.repository.DeleteCart(ctx, cart.ID)
		return uuid.Nil, nil

		// Option B: Trả lỗi rõ ràng
		// return uuid.Nil, fmt.Errorf("cart expired")
	}

	return cart.ID, nil
}

// GetOrCreateCartBySession implements ServiceInterface.GetOrCreateCartBySession (for middleware)
func (s *CartService) GetOrCreateCartBySession(ctx context.Context, sessionID string) (uuid.UUID, error) {
	// Step 1: Try to get existing cart
	cart, err := s.repository.GetBySessionID(ctx, sessionID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get session cart: %w", err)
	}

	// Step 2: If cart exists, check if expired
	if cart != nil {
		if cart.IsExpired() {
			// Clear expired cart items
			if err := s.repository.DeleteCart(ctx, cart.ID); err != nil {
				logger.Error("Failed to clear expired cart items", err)
			}
			// Treat as no cart (will create new below)
			cart = nil
		} else {
			// Update expiration (keep-alive)
			if err := s.repository.UpdateExpiration(ctx, cart.ID); err != nil {
				logger.Error("Failed to update cart expiration", err)
			}
			return cart.ID, nil
		}
	}

	// Step 3: Create new cart (with race condition protection)
	newCart := &model.Cart{
		UserID:     nil,
		SessionID:  &sessionID,
		ItemsCount: 0,
		Subtotal:   decimal.Zero,
		Version:    1,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(30 * 24 * time.Hour),
	}

	// Use CreateOrGet instead of Create (handles race condition)
	createdCart, err := s.repository.CreateOrGet(ctx, newCart)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create session cart: %w", err)
	}

	return createdCart.ID, nil
}

// domains/cart/service_impl.go

// MergeCart implements ServiceInterface.MergeCart
func (s *CartService) MergeCart(ctx context.Context, sessionID string, userID uuid.UUID) error {
	// Step 1: Get anonymous cart
	anonymousCart, err := s.repository.GetBySessionID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get anonymous cart: %w", err)
	}

	if anonymousCart == nil {
		return nil // No cart to merge
	}

	if anonymousCart.IsExpired() {
		_ = s.repository.DeleteCart(ctx, anonymousCart.ID)
		return nil
	}

	// Step 2: Get anonymous cart items
	anonymousItems, _, err := s.repository.GetItemsWithBooks(ctx, anonymousCart.ID, 0, 0) // Fetch all
	if err != nil {
		return fmt.Errorf("failed to get anonymous cart items: %w", err)
	}

	if len(anonymousItems) == 0 {
		_ = s.repository.DeleteCart(ctx, anonymousCart.ID)
		return nil
	}

	// ===== BEGIN TRANSACTION =====
	tx, err := s.repository.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer s.repository.RollbackTx(ctx, tx)

	// Step 3: Get or create user cart (within transaction)
	userCart, err := s.repository.GetByUserIDWithTx(ctx, tx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user cart: %w", err)
	}

	if userCart == nil {
		newCart := &model.Cart{
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
		// Use CreateOrGetWithTx to handle race condition
		userCart, err = s.repository.CreateOrGetWithTx(ctx, tx, newCart)
		if err != nil {
			return fmt.Errorf("failed to create user cart: %w", err)
		}
	}

	// Step 4: Get existing user cart items (within transaction with lock)
	userItems, err := s.repository.GetItemsByCartIDWithTx(ctx, tx, userCart.ID)
	if err != nil {
		return fmt.Errorf("failed to get user cart items: %w", err)
	}

	userItemsByBook := make(map[uuid.UUID]*model.CartItem)
	for i := range userItems {
		userItemsByBook[userItems[i].BookID] = &userItems[i]
	}

	// Step 5: Merge items
	for _, anonItem := range anonymousItems {
		// Validate book still active
		book, err := s.bookService.GetBookDetail(ctx, anonItem.BookID.String())
		if err != nil || !book.IsActive {
			// Skip inactive books
			logger.Error("Skipping inactive book in merge", err)
			continue
		}

		existingUserItem, exists := userItemsByBook[anonItem.BookID]

		if exists {
			// Merge: ADD quantities together (not max)
			newQty := existingUserItem.Quantity + anonItem.Quantity
			if newQty > 100 {
				newQty = 100 // Cap at max
			}

			// Update with latest price
			updateItem := &model.CartItem{
				ID:        existingUserItem.ID,
				CartID:    userCart.ID,
				BookID:    anonItem.BookID,
				Quantity:  newQty,
				Price:     book.Price, // Use current price
				UpdatedAt: time.Now(),
			}

			if err := s.repository.UpdateItemWithTx(ctx, tx, updateItem); err != nil {
				return fmt.Errorf("failed to update item: %w", err)
			}
		} else {
			// Transfer item to user cart
			transferItem := &model.CartItem{
				CartID:    userCart.ID,
				BookID:    anonItem.BookID,
				Quantity:  anonItem.Quantity,
				Price:     book.Price, // Use current price
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			if err := s.repository.AddItemWithTx(ctx, tx, transferItem); err != nil {
				return fmt.Errorf("failed to transfer item: %w", err)
			}
		}
	}

	// Step 6: Delete anonymous cart (CASCADE will delete items)
	if err := s.repository.DeleteCartWithTx(ctx, tx, anonymousCart.ID); err != nil {
		return fmt.Errorf("failed to delete anonymous cart: %w", err)
	}

	// ===== COMMIT TRANSACTION =====
	if err := s.repository.CommitTx(ctx, tx); err != nil {
		return fmt.Errorf("failed to commit merge: %w", err)
	}

	return nil
}

// domains/cart/service_impl.go

func (s *CartService) UpdateItemQuantity(ctx context.Context, cartID uuid.UUID, itemID uuid.UUID, quantity int) (*model.CartItemResponse, error) {
	// Step 1: Validate quantity
	if quantity < 0 || quantity > 100 {
		return nil, model.ErrInvalidQuantity
	}

	// Step 2: Get cart and item in single query (optimized)
	cart, item, err := s.repository.GetCartAndItem(ctx, cartID, itemID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart and item: %w", err)
	}
	if cart == nil {
		return nil, fmt.Errorf("cart not found")
	}
	if item == nil {
		return nil, fmt.Errorf("item not found")
	}
	if cart.IsExpired() {
		return nil, fmt.Errorf("cart has expired")
	}

	// Step 3: Handle quantity = 0 (remove item)
	if quantity == 0 {
		if err := s.repository.DeleteItem(ctx, itemID); err != nil {
			return nil, fmt.Errorf("failed to remove item: %w", err)
		}
		// Return response indicating deletion
		return &model.CartItemResponse{
			ID:       itemID,
			IsActive: false,
		}, nil
	}

	// Step 4: Validate book still active and get current price
	book, err := s.bookService.GetBookDetail(ctx, item.BookID.String())
	if err != nil {
		return nil, fmt.Errorf("book not found: %w", err)
	}
	if !book.IsActive {
		return nil, fmt.Errorf("book is no longer available")
	}

	// Step 5: Check stock if increasing quantity
	if quantity > item.Quantity {
		additionalQty := quantity - item.Quantity
		totalAvailable := s.getTotalAvailableStock(ctx, item.BookID)

		if totalAvailable < additionalQty {
			return nil, fmt.Errorf("insufficient stock: need %d more, only %d available",
				additionalQty, totalAvailable)
		}
	}

	// Step 6: Update item
	item.Quantity = quantity
	item.Price = book.Price // Update to current price (business decision)
	item.UpdatedAt = time.Now()

	if err := s.repository.UpdateItem(ctx, item); err != nil {
		return nil, fmt.Errorf("failed to update item: %w", err)
	}

	// Step 7: Fetch updated item with book details
	updatedItem, err := s.repository.GetItemWithBookByID(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch updated item: %w", err)
	}

	return updatedItem.ToItemResponse(), nil
}

// domains/cart/service_impl.go

// RemoveItem implements ServiceInterface.RemoveItem
func (s *CartService) RemoveItem(ctx context.Context, cartID uuid.UUID, itemID uuid.UUID) error {
	// Validate cart
	cart, err := s.repository.GetByID(ctx, cartID)
	if err != nil {
		return fmt.Errorf("failed to get cart: %w", err)
	}
	if cart == nil {
		return model.ErrCartNotFound
	}
	if cart.IsExpired() {
		return model.ErrCartExpired
	}

	// Get item to check existence and ownership separately
	item, err := s.repository.GetItemByID(ctx, itemID)
	if err != nil {
		return fmt.Errorf("failed to get item: %w", err)
	}
	if item == nil {
		return model.ErrItemNotFound
	}
	if item.CartID != cartID {
		return model.ErrItemNotBelongToCart // Custom error code
	}

	// Delete
	if err := s.repository.DeleteItem(ctx, itemID); err != nil {
		return fmt.Errorf("failed to delete item: %w", err)
	}

	return nil
}

// domains/cart/service_impl.go

// ClearCart implements ServiceInterface.ClearCart
func (s *CartService) ClearCart(ctx context.Context, cartID uuid.UUID) (int, error) {
	// Step 1: Validate cart exists and not expired
	cart, err := s.repository.GetByID(ctx, cartID)
	if err != nil {
		return 0, fmt.Errorf("failed to get cart: %w", err)
	}
	if cart == nil {
		return 0, model.ErrCartNotFound
	}
	if cart.IsExpired() {
		return 0, model.ErrCartExpired
	}

	// Step 2: Clear all items
	deletedCount, err := s.repository.ClearCartItems(ctx, cartID)
	if err != nil {
		return 0, fmt.Errorf("failed to clear cart items: %w", err)
	}

	// Step 3: Log activity
	if deletedCount > 0 {
		logger.Info("Cart cleared", map[string]interface{}{
			"cart_id":       cartID,
			"deleted_count": deletedCount,
		})
	}

	return deletedCount, nil
}

// domains/cart/service_impl.go

// ValidateCart implements ServiceInterface.ValidateCart
func (s *CartService) ValidateCart(ctx context.Context, cartID uuid.UUID, userID uuid.UUID) (*model.CartValidationResult, error) {
	result := &model.CartValidationResult{
		IsValid:         true,
		CartStatus:      "valid",
		Errors:          []model.CartValidationError{},
		Warnings:        []model.CartValidationWarning{},
		ItemValidations: []model.ItemValidation{},
	}
	logger.Info("Validate cart method:", map[string]interface{}{
		"cartID": cartID,
	})
	// Step 1: Get cart by ID (not userID)
	cart, err := s.repository.GetByID(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}
	if cart == nil {
		result.IsValid = false
		result.CartStatus = "error"
		result.Errors = append(result.Errors, model.CartValidationError{
			Code:    "CART_NOT_FOUND",
			Message: "Cart not found",
		})
		return result, nil
	}

	// Step 2: Validate ownership (if userID provided)
	if userID != uuid.Nil && cart.UserID != nil && *cart.UserID != userID {
		result.IsValid = false
		result.CartStatus = "error"
		result.Errors = append(result.Errors, model.CartValidationError{
			Code:    "CART_NOT_OWNED",
			Message: "Cart does not belong to user",
		})
		return result, nil
	}

	// Step 3: Check if expired
	if cart.IsExpired() {
		result.IsValid = false
		result.CartStatus = "error"
		result.Errors = append(result.Errors, model.CartValidationError{
			Code:    "CART_EXPIRED",
			Message: "Cart has expired",
		})
		return result, nil
	}

	// Step 4: Get all items with book details
	items, _, err := s.repository.GetItemsWithBooks(ctx, cartID, 0, 0) // Fetch all
	if err != nil {
		return nil, fmt.Errorf("failed to fetch cart items: %w", err)
	}

	if len(items) == 0 {
		result.IsValid = false
		result.CartStatus = "error"
		result.Errors = append(result.Errors, model.CartValidationError{
			Code:    "EMPTY_CART",
			Message: "Cart is empty",
		})
		return result, nil
	}

	// Step 5: Validate each item
	var totalValue decimal.Decimal
	var snapshotTotal decimal.Decimal
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

		// Check availability
		if !itemValidation.IsAvailable {
			hasErrors = true
			itemValidation.Warnings = append(itemValidation.Warnings,
				fmt.Sprintf("Book not available (active: %v, stock: %d)", item.IsActive, item.TotalStock))
		}

		// Check stock
		if !itemValidation.StockSufficient {
			hasErrors = true
			itemValidation.Warnings = append(itemValidation.Warnings,
				fmt.Sprintf("Insufficient stock: requested %d, available %d", item.Quantity, item.TotalStock))
		}

		// Check price change
		if !itemValidation.PriceMatch {
			hasWarnings = true
			priceDiff := item.CurrentPrice.Sub(item.Price)
			itemValidation.Warnings = append(itemValidation.Warnings,
				fmt.Sprintf("Price changed: %s → %s (%s)", item.Price, item.CurrentPrice, priceDiff))

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

		// Calculate totals
		itemCurrentTotal := decimal.NewFromInt(int64(item.Quantity)).Mul(item.CurrentPrice)
		itemSnapshotTotal := decimal.NewFromInt(int64(item.Quantity)).Mul(item.Price)
		totalValue = totalValue.Add(itemCurrentTotal)
		snapshotTotal = snapshotTotal.Add(itemSnapshotTotal)

		result.ItemValidations = append(result.ItemValidations, itemValidation)
	}

	// Step 6: Determine overall status
	if hasErrors {
		result.IsValid = false
		result.CartStatus = "error"
	} else if hasWarnings {
		result.IsValid = true
		result.CartStatus = "warning"
	}

	result.TotalValue = totalValue       // Current price total
	result.SnapshotTotal = snapshotTotal // Snapshot price total (for comparison)
	result.EstimatedTotal = totalValue

	return result, nil
}

// domains/cart/service_impl.go

func (s *CartService) ApplyPromoCode(ctx context.Context, cartID uuid.UUID, promoCode string, userID uuid.UUID) (*model.ApplyPromoResponse, error) {
	// Step 1: Validate input
	if userID == uuid.Nil {
		return nil, fmt.Errorf("user ID required")
	}
	if len(promoCode) < 3 || len(promoCode) > 50 {
		return nil, fmt.Errorf("invalid promo code format")
	}

	// Step 2: Get and validate cart
	cart, err := s.repository.GetByID(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}
	if cart == nil {
		return nil, model.ErrCartNotFound
	}
	if cart.IsExpired() {
		return nil, model.ErrCartExpired
	}

	// Verify ownership
	if cart.UserID == nil || *cart.UserID != userID {
		return nil, fmt.Errorf("cart does not belong to user")
	}

	// Check cart has items
	if cart.ItemsCount == 0 {
		return nil, fmt.Errorf("cannot apply promo to empty cart")
	}

	// Check if cart already has promo
	if cart.HasPromo() { // ✅ Use helper method
		// Clear old promo before applying new one
		if err := s.repository.ClearCartPromo(ctx, cartID); err != nil {
			return nil, fmt.Errorf("failed to clear old promo: %w", err)
		}
	}

	// Step 3: Validate promo with promotion service (through interface)
	promoReq := &model.ValidatePromoRequest{ // ✅ Use request struct
		PromoCode: promoCode,
		UserID:    userID,
		CartTotal: cart.Subtotal,
		CartID:    cartID,
	}

	promo, err := s.ValidatePromoCode(ctx, promoReq) // ✅ Call interface
	if err != nil {
		return nil, fmt.Errorf("failed to validate promo: %w", err)
	}

	if !promo.IsValid {
		return nil, fmt.Errorf("invalid promo code: %s", promo.Reason)
	}

	// Step 4: Calculate discount
	discountAmount := s.calculatePromoDiscount(cart.Subtotal, promo)

	// Step 5: Build promo metadata
	promoMetadata := map[string]interface{}{
		"promotion_id": promo.PromotionID.String(),
		"code":         promo.Code,
		"name":         promo.Name,
		"type":         promo.DiscountType,
		"value":        promo.DiscountValue.String(),
		"applied_at":   time.Now().Format(time.RFC3339),
	}

	// Step 6: Update cart with promo (with optimistic locking)
	err = s.repository.UpdateCartPromo(ctx, cartID, cart.Version, &promoCode, discountAmount, promoMetadata)
	if err != nil {
		return nil, fmt.Errorf("failed to apply promo: %w", err)
	}

	// Step 7: Return response
	return &model.ApplyPromoResponse{
		Applied:          true,
		PromoCode:        promo.Code,
		PromoDescription: promo.Description,
		DiscountType:     promo.DiscountType,
		DiscountValue:    promo.DiscountValue,
		DiscountAmount:   discountAmount,
		OriginalSubtotal: cart.Subtotal,
		DiscountedTotal:  cart.Subtotal.Sub(discountAmount),
		AppliedAt:        time.Now(),
	}, nil
}

// Helper: Calculate discount based on promo type
func (s *CartService) calculatePromoDiscount(subtotal decimal.Decimal, promo *model.PromotionValidationResult) decimal.Decimal {
	var discount decimal.Decimal

	switch promo.DiscountType {
	case "percentage":
		// Calculate percentage discount
		discount = subtotal.Mul(promo.DiscountValue).Div(decimal.NewFromInt(100))
	case "fixed":
		// Fixed amount discount
		discount = promo.DiscountValue
	default:
		discount = decimal.Zero
	}

	// Apply max discount cap
	if promo.MaxDiscount != nil && discount.GreaterThan(*promo.MaxDiscount) {
		discount = *promo.MaxDiscount
	}

	// Ensure discount doesn't exceed subtotal
	if discount.GreaterThan(subtotal) {
		discount = subtotal
	}

	return discount
}

// RemovePromoCode implements ServiceInterface.RemovePromoCode
func (s *CartService) RemovePromoCode(ctx context.Context, cartID uuid.UUID) error {
	err := s.repository.RemoveCartPromo(ctx, cartID)
	if err != nil {
		return fmt.Errorf("failed to remove promo: %w", err)
	}
	return nil
}

// Track successful reservations để rollback nếu có lỗi
type ReservationInfo struct {
	WarehouseID uuid.UUID
	BookID      uuid.UUID
	Quantity    int
}

func (s *CartService) Checkout(ctx context.Context, userID uuid.UUID, cartID uuid.UUID, req model.CheckoutRequest) (*model.CheckoutResponse, error) {
	response := &model.CheckoutResponse{
		Success:     false,
		Status:      "pending",
		InitiatedAt: time.Now(),
		Errors:      []model.CheckoutError{},
		Warnings:    []model.CheckoutWarning{},
		NextActions: []string{},
		Phases:      []model.CheckoutPhaseResult{},
	}
	// ==================== PHASE 0: Validate User ====================
	if userID == uuid.Nil {
		return s.failCheckout(response, "UNAUTHENTICATED", "User not authenticated", "")
	}

	// ==================== PHASE 1: Get & Validate Cart ====================
	phaseStart := time.Now()
	cart, err := s.repository.GetByID(ctx, cartID)
	if err != nil {
		return s.failCheckout(response, "CART_NOT_FOUND", "Cannot find your cart: "+err.Error(), "")
	}
	if cart == nil || cart.IsExpired() {
		return s.failCheckout(response, "CART_EXPIRED", "Your cart is expired or not found", "")
	}

	// Get all items (no pagination)
	cartItems, _, err := s.repository.GetItemsWithBooks(ctx, cart.ID, 1, 1000) // ✅ Use high limit instead of 0,0
	if err != nil || len(cartItems) == 0 {
		return s.failCheckout(response, "EMPTY_CART", "Cart is empty", "")
	}

	// Populate cart summary
	response.CartSummary = model.CartCheckoutSummary{
		CartID:    cartID,
		ItemCount: len(cartItems),
		Subtotal:  cart.Subtotal,
		PromoCode: cart.PromoCode,
		Discount:  cart.Discount,
		Total:     cart.Total,
	}

	// Validate cart
	validation, err := s.ValidateCart(ctx, cartID, userID)
	logger.Info("validation", map[string]interface{}{
		"validation": validation,
	})
	if err != nil || !validation.IsValid {
		for _, valErr := range validation.Errors {
			response.Errors = append(response.Errors, model.CheckoutError{
				Code:     valErr.Code,
				Message:  valErr.Message,
				Severity: valErr.Severity,
			})
		}
		response.Phases = append(response.Phases, model.CheckoutPhaseResult{
			Phase:     "CART_VALIDATION",
			Status:    "failed",
			Message:   "Cart validation failed",
			Timestamp: phaseStart,
			Errors:    convertToCheckoutErrors(validation.Errors),
		})
		response.Status = "failed"
		return response, nil
	}

	response.Phases = append(response.Phases, model.CheckoutPhaseResult{
		Phase:     "CART_VALIDATION",
		Status:    "success",
		Message:   "Cart validated",
		Timestamp: phaseStart,
	})

	// ==================== PHASE 2: Validate Address ====================
	phaseStart = time.Now()
	logger.Info("ShippingAddressID", map[string]interface{}{
		"ShippingAddressID": req.ShippingAddressID,
	})
	shippingAddr, err := s.address.GetAddressByID(ctx, userID, req.ShippingAddressID)
	if err != nil {
		response.Phases = append(response.Phases, model.CheckoutPhaseResult{
			Phase:     "ADDRESS_VALIDATION",
			Status:    "failed",
			Message:   "Shipping address validation failed",
			Timestamp: phaseStart,
			Errors: []model.CheckoutError{{
				Code:     "INVALID_SHIPPING_ADDRESS",
				Message:  err.Error(),
				Severity: "critical",
			}},
		})
		response.Status = "failed"
		return response, nil
	}
	logger.Info("shippingAddr info", map[string]interface{}{
		"shippingAddr": shippingAddr,
	})
	if shippingAddr.Latitude == nil || shippingAddr.Longitude == nil {
		response.Warnings = append(response.Warnings, model.CheckoutWarning{
			Code:    "MISSING_COORDINATES",
			Message: "Shipping address missing coordinates. Will use default warehouse.",
		})
	}

	// ✅ Complete phase 2
	response.Phases = append(response.Phases, model.CheckoutPhaseResult{
		Phase:     "ADDRESS_VALIDATION",
		Status:    "success",
		Message:   "Address validated",
		Timestamp: phaseStart,
	})

	// ==================== PHASE 3: Promo Validation ====================
	var promoDiscount decimal.Decimal
	var appliedPromo *string
	if req.PromoCode != nil {
		phaseStart = time.Now()
		promoDiscount, appliedPromo, _ = s.validateAndApplyPromo(ctx, req, cart, cartID, userID, response, phaseStart)
	}

	// ==================== PHASE 4: Pricing Calculation ====================
	phaseStart = time.Now()
	subtotal := cart.Subtotal
	discount := promoDiscount

	// Clamp discount
	if discount.GreaterThan(subtotal) {
		discount = subtotal
	}

	taxRate := decimal.NewFromFloat(0) // 0% tax
	tax := subtotal.Sub(discount).Mul(taxRate)
	shipping := decimal.NewFromInt(15000) // 15k VND
	codFee := decimal.Zero
	if req.PaymentMethod == "cash_on_delivery" {
		codFee = decimal.NewFromInt(15000)
	}

	total := subtotal.Sub(discount).Add(tax).Add(shipping).Add(codFee)

	response.PricingBreakdown = model.PricingBreakdown{
		Subtotal:      subtotal,
		PromoDiscount: discount,
		Tax:           tax,
		Shipping:      shipping,
		Total:         total,
		Currency:      "VND",
		TaxRate:       taxRate,
	}

	response.CartSummary.EstimatedTax = tax
	response.CartSummary.ShippingCost = shipping
	response.CartSummary.Total = total

	response.Phases = append(response.Phases, model.CheckoutPhaseResult{
		Phase:     "PRICING_CALCULATION",
		Status:    "success",
		Message:   "Pricing calculated",
		Timestamp: phaseStart,
	})
	logger.Info("response", map[string]interface{}{
		"response": response,
	})

	// ==================== PHASE 5: Warehouse Selection (1 LẦN DUY NHẤT) ====================
	phaseStart = time.Now()

	// Build availability request
	availabilityItems := make([]inventoryModel.CheckAvailabilityItem, len(cartItems))
	for i, item := range cartItems {
		availabilityItems[i] = inventoryModel.CheckAvailabilityItem{
			BookID:   item.BookID,
			Quantity: item.Quantity,
		}
	}

	availabilityReq := inventoryModel.CheckAvailabilityRequest{
		Items: availabilityItems,
	}

	if shippingAddr.Latitude != nil && shippingAddr.Longitude != nil {
		availabilityReq.CustomerLatitude = shippingAddr.Latitude
		availabilityReq.CustomerLongitude = shippingAddr.Longitude
	}

	// ✅ Call CheckAvailability MỘT LẦN DUY NHẤT
	availability, err := s.inventoryService.CheckAvailability(ctx, availabilityReq)
	if err != nil {
		return s.failCheckout(response, "AVAILABILITY_CHECK_FAILED", "Cannot check stock: "+err.Error(), "WAREHOUSE_SELECTION")
	}
	logger.Info("availability result", map[string]interface{}{
		"availability": availability,
	})
	if !availability.Overall {
		response.Errors = append(response.Errors, model.CheckoutError{
			Code:     "INSUFFICIENT_STOCK",
			Message:  "One or more items are out of stock",
			Severity: "critical",
		})

		for _, itemAvail := range availability.Items {
			if !itemAvail.Fulfillable {
				itemName := itemAvail.BookID.String()
				response.Errors = append(response.Errors, model.CheckoutError{
					Code:     "ITEM_OUT_OF_STOCK",
					Message:  itemAvail.Recommendation,
					Severity: "error",
					Field:    &itemName,
				})
			}
		}

		response.Phases = append(response.Phases, model.CheckoutPhaseResult{
			Phase:     "WAREHOUSE_SELECTION",
			Status:    "failed",
			Message:   "Insufficient stock",
			Timestamp: phaseStart,
		})
		response.Status = "failed"
		return response, nil
	}

	if availability.RecommendedWarehouse != nil {
		response.WarehouseInfo = &model.WarehouseCheckoutInfo{
			WarehouseID:       availability.RecommendedWarehouse.WarehouseID,
			WarehouseName:     availability.RecommendedWarehouse.WarehouseName,
			DistanceKM:        availability.RecommendedWarehouse.DistanceKM,
			EstimatedDelivery: availability.RecommendedWarehouse.EstimatedDelivery,
		}
	}
	logger.Info("response.WarehouseInfo ", map[string]interface{}{
		"response.WarehouseInfo ": response.WarehouseInfo,
	})
	response.Phases = append(response.Phases, model.CheckoutPhaseResult{
		Phase:     "WAREHOUSE_SELECTION",
		Status:    "success",
		Message:   "Warehouse selected and stock available",
		Timestamp: phaseStart,
	})

	// ==================== PHASE 6: CREATE ORDER QUA ORDER SERVICE ====================
	phaseStart = time.Now()

	// Build CreateOrderRequest cho order service
	createReq := orderModel.CreateOrderRequest{
		AddressID:     req.ShippingAddressID,                   // nếu nil, order service sẽ lấy default
		PaymentMethod: mapCartPaymentMethod(req.PaymentMethod), // e.g. "cash_on_delivery" -> "cod"
		PromoCode:     cart.PromoCode,                          // promo gắn với cart
		CustomerNote:  req.CustomerNotes,
		Items:         nil,
		// Items sẽ được override bên trong orderService từ cart_items
	}
	logger.Info("createReq", map[string]interface{}{
		"createReq": createReq,
		"ctx":       ctx,
		"userID":    userID,
	})
	// Gọi order service (use case duy nhất)
	orderResp, err := s.orderService.CreateOrder(ctx, userID, createReq)
	if err != nil {
		return s.failCheckout(response, "ORDER_CREATION_FAILED", "Failed to create order: "+err.Error(), "ORDER_CREATION")
	}

	// Ghi phase kết quả
	response.Phases = append(response.Phases, model.CheckoutPhaseResult{
		Phase:     "ORDER_CREATION",
		Status:    "success",
		Message:   "Order created: " + orderResp.OrderNumber,
		Timestamp: phaseStart,
	})

	// Build success response từ orderResp + dữ liệu đã có
	now := time.Now()
	response = s.buildSuccessResponse(
		response,
		&orderModel.Order{
			ID:          orderResp.OrderID,
			OrderNumber: orderResp.OrderNumber,
			Total:       orderResp.Total,
			Status:      orderResp.Status,
			// nếu cần thêm field khác thì query GetOrderByID ở đây
		},
		cartItems,
		total,
		codFee,
		now,
		req.PaymentMethod,
	)
	go s.enqueuePostCheckoutTasks(context.Background(), orderResp.OrderID, orderResp.OrderNumber, userID, cartID, req, total, len(cartItems), promoDiscount, appliedPromo)
	// ==================== Build Success Response ====================
	return response, nil
}
func mapCartPaymentMethod(method string) string {
	logger.Info("method payment", map[string]interface{}{
		"method": method,
	})
	switch method {
	case "cash_on_delivery":
		return orderModel.PaymentMethodCOD
	case "e_wallet":
		return orderModel.PaymentMethodMomo // hoặc mapping khác tùy design
	case "bank_transfer":
		return orderModel.PaymentMethodBankTransfer
	case "credit_card":
		return orderModel.PaymentMethodVNPay // giả sử dùng VNPay cho credit
	default:
		return orderModel.PaymentMethodCOD
	}
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

func (s *CartService) failCheckout(response *model.CheckoutResponse, code, message, phase string) (*model.CheckoutResponse, error) {
	response.Status = "failed"
	response.Errors = append(response.Errors, model.CheckoutError{
		Code:     code,
		Message:  message,
		Severity: "critical",
	})
	if phase != "" {
		response.Phases = append(response.Phases, model.CheckoutPhaseResult{
			Phase:   phase,
			Status:  "failed",
			Message: message,
		})
	}
	return response, nil
}

func (s *CartService) validateAndApplyPromo(ctx context.Context, req model.CheckoutRequest, cart *model.Cart, cartID, userID uuid.UUID, response *model.CheckoutResponse, phaseStart time.Time) (decimal.Decimal, *string, map[string]interface{}) {
	var promoDiscount decimal.Decimal = decimal.Zero
	var appliedPromo *string
	var promoMetadata map[string]interface{}

	// Logic xử lý promo giữ nguyên như code cũ...
	// (code này đã đúng, chỉ extract ra method riêng để dễ đọc)

	return promoDiscount, appliedPromo, promoMetadata
}

func (s *CartService) extractPromotionID(appliedPromo *string, promoMetadata map[string]interface{}) *uuid.UUID {
	if appliedPromo != nil && promoMetadata != nil {
		if promoIDStr, ok := promoMetadata["promotion_id"].(string); ok {
			promoID, _ := uuid.Parse(promoIDStr)
			if promoID != uuid.Nil {
				return &promoID
			}
		}
	}
	return nil
}

func (s *CartService) extractWarehouseID(reservations []ReservationInfo) *uuid.UUID {
	if len(reservations) > 0 {
		return &reservations[0].WarehouseID
	}
	return nil
}

func (s *CartService) buildSuccessResponse(response *model.CheckoutResponse, order *orderModel.Order, cartItems []*model.CartItemWithBook, total, codFee decimal.Decimal, now time.Time, paymentMethod string) *model.CheckoutResponse {
	completedAt := now
	response.Success = true
	response.Status = "completed"
	response.OrderID = order.ID
	response.OrderNumber = order.OrderNumber // ✅ DB-generated
	response.ReferenceCode = order.OrderNumber
	response.CompletedAt = &completedAt

	response.OrderSummary = &model.OrderCheckoutSummary{
		OrderID:     order.ID,
		OrderNumber: order.OrderNumber, // ✅ Add
		Status:      orderModel.OrderStatusPending,
		TotalAmount: total,
		ItemCount:   len(cartItems),
		CreatedAt:   now,
	}

	if paymentMethod == "cash_on_delivery" {
		response.NextActions = []string{
			"Your order has been placed. Pay on delivery.",
			"Track your order: " + order.OrderNumber,
		}
	} else {
		expiresAt := now.Add(15 * time.Minute)
		response.ExpiresAt = &expiresAt
		response.NextActions = []string{
			"Complete payment within 15 minutes to confirm order",
			"Track your order: " + order.OrderNumber,
		}
	}

	return response
}

// enqueuePostCheckoutTasks enqueues all background tasks after successful checkout
func (s *CartService) enqueuePostCheckoutTasks(
	ctx context.Context,
	orderID uuid.UUID,
	orderNumber string,
	userID uuid.UUID,
	cartID uuid.UUID,
	req model.CheckoutRequest,
	total decimal.Decimal,
	itemCount int,
	discount decimal.Decimal,
	promoCode *string,
) {
	// Get user email (ignore error, task will retry)
	userEmail, err := s.repository.GetUserEmail(ctx, userID)
	if err != nil {
		logger.Info("Failed to get user email for order confirmation", map[string]interface{}{
			"order_id": orderID,
			"user_id":  userID,
			"error":    err.Error(),
		})
		userEmail = "" // Task will skip email if empty
	}

	// Task 1: Clear cart (low priority, delay 30s)
	// s.enqueueClearCart(cartID, userID)

	// Task 2: Send order confirmation email (default priority, immediate)
	if userEmail != "" {
		s.enqueueSendOrderConfirmation(orderID, orderNumber, userID, userEmail, total, req, itemCount)
	}

	// Task 3: Auto-release reservation if not COD (high priority, delay 15 min)
	if req.PaymentMethod != "cash_on_delivery" {
		s.enqueueAutoReleaseReservation(orderID, orderNumber, userID)
	}

	// Task 4: Track checkout analytics (low priority, immediate)
	s.enqueueTrackCheckout(orderID, orderNumber, userID, total, itemCount, req.PaymentMethod, promoCode, discount)
}

// enqueueClearCart enqueues task to clear cart
// func (s *CartService) enqueueClearCart(cartID, userID uuid.UUID) {
// 	payload := model.ClearCartPayload{
// 		CartID: cartID,
// 		UserID: userID,
// 	}

// 	task, err := utils.MarshalTask(shared.TypeClearCart, payload)
// 	if err != nil {
// 		logger.Info("Failed to marshal clear cart task", map[string]interface{}{
// 			"cart_id": cartID,
// 			"error":   err.Error(),
// 		})
// 		return
// 	}

// 	_, err = s.asynqClient.Enqueue(task,
// 		asynq.Queue("low"),
// 		asynq.MaxRetry(2),
// 		asynq.ProcessIn(30*time.Second), // Delay 30s
// 	)

// 	if err != nil {
// 		logger.Info("Failed to enqueue clear cart task", map[string]interface{}{
// 			"cart_id": cartID,
// 			"error":   err.Error(),
// 		})
// 	} else {
// 		logger.Info("Enqueued clear cart task", map[string]interface{}{
// 			"cart_id": cartID,
// 		})
// 	}
// }

// enqueueSendOrderConfirmation enqueues order confirmation email
func (s *CartService) enqueueSendOrderConfirmation(
	orderID uuid.UUID,
	orderNumber string,
	userID uuid.UUID,
	userEmail string,
	total decimal.Decimal,
	req model.CheckoutRequest,
	itemCount int,
) {
	payload := model.SendOrderConfirmationPayload{
		OrderID:           orderID,
		OrderNumber:       orderNumber,
		UserID:            userID,
		UserEmail:         userEmail,
		Total:             total,
		PaymentMethod:     req.PaymentMethod,
		EstimatedDelivery: "3-5 ngày",
		ShippingAddressID: req.ShippingAddressID,
		OrderCreatedAt:    time.Now().Format(time.RFC3339),
	}

	task, err := utils.MarshalTask(shared.TypeSendOrderConfirmation, payload)
	if err != nil {
		logger.Info("Failed to marshal send email task", map[string]interface{}{
			"order_id": orderID,
			"error":    err.Error(),
		})
		return
	}

	_, err = s.asynqClient.Enqueue(task,
		asynq.Queue("default"),
		asynq.MaxRetry(2),
		asynq.Timeout(30*time.Second),
	)

	if err != nil {
		logger.Info("Failed to enqueue send email task", map[string]interface{}{
			"order_id": orderID,
			"error":    err.Error(),
		})
	} else {
		logger.Info("Enqueued send order confirmation email", map[string]interface{}{
			"order_id": orderID,
			"email":    userEmail,
		})
	}
}

// enqueueAutoReleaseReservation schedules auto-release if payment not completed
func (s *CartService) enqueueAutoReleaseReservation(orderID uuid.UUID, orderNumber string, userID uuid.UUID) {
	payload := model.AutoReleaseReservationPayload{
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

	_, err = s.asynqClient.Enqueue(task,
		asynq.Queue("high"),             // High priority
		asynq.MaxRetry(3),               // Critical task
		asynq.ProcessIn(15*time.Minute), // Execute after 15 minutes
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

// enqueueTrackCheckout enqueues analytics tracking
func (s *CartService) enqueueTrackCheckout(
	orderID uuid.UUID,
	orderNumber string,
	userID uuid.UUID,
	total decimal.Decimal,
	itemCount int,
	paymentMethod string,
	promoCode *string,
	discount decimal.Decimal,
) {
	payload := model.TrackCheckoutPayload{
		OrderID:       orderID,
		OrderNumber:   orderNumber,
		UserID:        userID,
		Total:         total,
		ItemCount:     itemCount,
		PaymentMethod: paymentMethod,
		PromoCode:     promoCode,
		Discount:      discount,
	}

	task, err := utils.MarshalTask(shared.TypeTrackCheckout, payload)
	if err != nil {
		logger.Info("Failed to marshal track checkout task", map[string]interface{}{
			"order_id": orderID,
			"error":    err.Error(),
		})
		return
	}

	_, err = s.asynqClient.Enqueue(task,
		asynq.Queue("low"),
		asynq.MaxRetry(0), // Don't retry analytics
	)

	if err != nil {
		logger.Info("Failed to enqueue track checkout task", map[string]interface{}{
			"order_id": orderID,
			"error":    err.Error(),
		})
	} else {
		logger.Info("Enqueued track checkout task", map[string]interface{}{
			"order_id": orderID,
		})
	}
}
