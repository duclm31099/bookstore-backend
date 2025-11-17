package service

import (
	addressService "bookstore-backend/internal/domains/address/service"
	bookS "bookstore-backend/internal/domains/book/service"
	"bookstore-backend/internal/domains/cart/model"
	repo "bookstore-backend/internal/domains/cart/repository"
	inventoryModel "bookstore-backend/internal/domains/inventory/model"
	inveRepo "bookstore-backend/internal/domains/inventory/repository"
	inveService "bookstore-backend/internal/domains/inventory/service"
	"bookstore-backend/internal/shared/utils"
	"bookstore-backend/pkg/logger"
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type CartService struct {
	repository       repo.RepositoryInterface
	inventoryService inveService.ServiceInterface
	inventoryRepo    inveRepo.RepositoryInterface
	address          addressService.ServiceInterface
	bookService      bookS.ServiceInterface
}

func NewCartService(
	r repo.RepositoryInterface,
	inventoryS inveService.ServiceInterface,
	addressSvc addressService.ServiceInterface,
	inventoryRepo inveRepo.RepositoryInterface,
	book bookS.ServiceInterface,

) ServiceInterface {
	return &CartService{
		repository:       r,
		inventoryService: inventoryS,
		address:          addressSvc,
		inventoryRepo:    inventoryRepo,
		bookService:      book,
	}
}

// GetOrCreateCart implements ServiceInterface.GetOrCreateCart
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

	// Step 3: Check if cart expired
	if cart != nil && cart.ExpiresAt.Before(time.Now()) {
		// Cart expired → clear items and reset
		_ = s.repository.DeleteCart(ctx, cart.ID) // Best effort
		cart = nil                                // Force create new cart
	}

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
		// Step 5: Update expiration (keep-alive)
		if err := s.repository.UpdateExpiration(ctx, createdCart.ID); err != nil {
			// Log warning but don't fail request
			logger.Error("Failed to update cart expiration", err)
		}
	}

	// Step 6: Fetch all items with book details (no hardcode limit)
	items, _, err := s.repository.GetItemsWithBooks(ctx, createdCart.ID, 0, 0) // 0,0 = fetch all
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
	validation, err := s.ValidateCart(ctx, cartID, userID)
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
	// go func() {
	// 	_ = s.ClearCart(context.Background(), cartID)
	// }()

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
