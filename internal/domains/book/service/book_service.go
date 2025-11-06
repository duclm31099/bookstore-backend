package book

import (
	"bookstore-backend/internal/domains/book"
	dto "bookstore-backend/internal/domains/book"
	"bookstore-backend/internal/shared/utils"
	"bookstore-backend/pkg/cache"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/shopspring/decimal"
)

// Service - Implements ServiceInterface
type BookService struct {
	repo  book.RepositoryInterface
	cache cache.Cache
}

// NewService - Constructor with DI
func NewService(repo book.RepositoryInterface, cache cache.Cache) book.ServiceInterface {
	return &BookService{
		repo:  repo,
		cache: cache,
	}
}

// ListBooks - Business logic for listing books
func (s *BookService) ListBooks(ctx context.Context, req dto.ListBooksRequest) ([]dto.ListBooksResponse, *dto.PaginationMeta, error) {
	// Validate input
	if err := dto.ValidateListRequest(req); err != nil {
		return nil, nil, err
	}

	// Generate cache key from request parameters
	cacheKey := dto.GenerateCacheKey("books:list", req)
	var result struct {
		Data       []dto.ListBooksResponse
		Pagination dto.PaginationMeta
	}
	_, err := s.cache.Get(ctx, cacheKey, &result)
	// Try to get from cache first
	if err != nil {
		return nil, nil, err
	}

	// Cache MISS - query database
	log.Printf("Cache MISS for key: %s", cacheKey)

	// Build filter for repository
	filter := &dto.BookFilter{
		Search:     req.Search,
		CategoryID: req.CategoryID,
		PriceMin:   req.PriceMin,
		PriceMax:   req.PriceMax,
		Language:   req.Language,
		Sort:       req.Sort,
		Offset:     (req.Page - 1) * req.Limit,
		Limit:      req.Limit,
	}

	// Query database
	books, totalCount, err := s.repo.ListBooks(ctx, filter)
	if err != nil {
		return nil, nil, fmt.Errorf("list books error: %w", err)
	}

	// Map entities to DTOs
	responses := make([]dto.ListBooksResponse, len(books))
	for i, book := range books {
		responses[i] = dto.BookToListDTO(book)
	}

	// Calculate pagination metadata
	totalPages := (totalCount + req.Limit - 1) / req.Limit
	meta := &dto.PaginationMeta{
		Page:      req.Page,
		PageSize:  req.Limit,
		Total:     totalCount,
		TotalPage: totalPages,
	}

	// Cache the result
	cacheData := struct {
		Data []dto.ListBooksResponse
		Meta dto.PaginationMeta
	}{
		Data: responses,
		Meta: *meta,
	}

	jsonData, _ := json.Marshal(cacheData)
	if err := s.cache.Set(ctx, cacheKey, string(jsonData), 3600); err != nil { // TTL 1 hour
		log.Printf("Cache SET error for key %s: %v", cacheKey, err)
		// Don't fail request if cache write fails
	}

	return responses, meta, nil
}
func (s *BookService) GetBookDetail(ctx context.Context, id string) (*dto.BookDetailResponse, error) {
	// Lấy dữ liệu chi tiết sách
	b, inventories, err := s.repo.GetBookByID(ctx, id)
	if err != nil {
		return nil, err
	}
	// Lấy review nổi bật
	reviews, _ := s.repo.GetReviewsHighlight(ctx, id)
	// Build DTO cha
	detail := &dto.BookDetailResponse{
		ID:            b.ID,
		Title:         b.Title,
		Author:        &dto.AuthorDTO{ID: b.AuthorID, Name: b.AuthorName},
		Category:      &dto.CategoryDTO{ID: b.CategoryID, Name: b.CategoryName},
		Publisher:     &dto.PublisherDTO{ID: b.PublisherID, Name: b.PublisherName},
		Description:   b.Description,
		Price:         b.Price.InexactFloat64(),
		Language:      b.Language,
		Format:        b.Format,
		CoverURL:      b.CoverURL,
		PublishedYear: b.PublishedYear,
		RatingAverage: b.RatingAverage,
		RatingCount:   b.RatingCount,
		TotalStock:    b.TotalStock,
		Inventories:   inventories,
		Reviews:       reviews,
	}
	// Tăng view_count async
	go s.repo.IncrementViewCount(context.Background(), id)
	return detail, nil
}

// CreateBook - Business logic for creating book
func (s *BookService) CreateBook(ctx context.Context, req book.CreateBookRequest) error {
	// 1. Validate foreign keys exist
	if exists, err := s.repo.ValidateAuthor(ctx, req.AuthorID); err != nil || !exists {
		return book.ErrAuthorNotFound
	}

	if req.CategoryID != "" {
		if exists, err := s.repo.ValidateCategory(ctx, req.CategoryID); err != nil || !exists {
			return book.ErrCategoryNotFound
		}
	}

	if req.PublisherID != "" {
		if exists, err := s.repo.ValidatePublisher(ctx, req.PublisherID); err != nil || !exists {
			return book.ErrPublisherNotFound
		}
	}

	// 2. Check ISBN uniqueness (nếu có ISBN)
	if req.ISBN != "" {
		if exists, err := s.repo.CheckISBNExists(ctx, req.ISBN); err != nil || exists {
			return book.ErrISBNAlreadyExists
		}
	}

	// 3. Generate slug from title
	slug := utils.GenerateSlugBook(req.Title)

	// 4. Check slug uniqueness (nếu trùng, append số: book-title-2)
	finalSlug, err := s.repo.GenerateUniqueSlug(ctx, slug)
	if err != nil {
		return err
	}

	// 5. Build Book entity
	now := time.Now()

	book := &book.Book{
		Title:           req.Title,
		Slug:            finalSlug,
		ISBN:            req.ISBN,
		AuthorID:        utils.ParseStringToUUID(req.AuthorID),
		PublisherID:     utils.ParseStringToUUID(req.PublisherID),
		CategoryID:      utils.ParseStringToUUID(req.CategoryID),
		Price:           decimal.NewFromFloat(req.Price),
		CompareAtPrice:  utils.ParseFloatToDecimal(req.CompareAtPrice),
		CostPrice:       utils.ParseFloatToDecimal(req.CostPrice),
		CoverURL:        req.CoverURL,
		Description:     req.Description,
		Pages:           req.Pages,
		Language:        req.Language,
		PublishedYear:   req.PublishedYear,
		Format:          req.Format,
		Dimensions:      req.Dimensions,
		WeightGrams:     req.WeightGrams,
		EbookFileURL:    req.EbookFileURL,
		EbookFileSizeMB: utils.ParseFloatToDecimal(req.EbookFileSizeMb),
		EbookFormat:     req.EbookFormat,
		IsActive:        req.IsActive,
		IsFeatured:      req.IsFeatured,
		MetaTitle:       req.MetaTitle,
		MetaDescription: req.MetaDescription,
		MetaKeywords:    req.MetaKeywords,
		Images:          req.Images,
		ViewCount:       0,
		SoldCount:       0,
		RatingAverage:   0.0,
		RatingCount:     0,
		Version:         0,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// 6. Save to database
	if err := s.repo.CreateBook(ctx, book); err != nil {
		return fmt.Errorf("failed to create book: %w", err)
	}

	// 7. Invalidate list cache (xóa cache danh sách sách)
	if err := s.cache.Delete(ctx, "books:list:*"); err != nil {
		log.Printf("[Service] Failed to invalidate list cache: %v", err)
	}

	// 8. Get full detail (with joins) to return
	return nil
}

// UPDATE BOOK
// UpdateBook - Business logic for updating book
func (s *BookService) UpdateBook(ctx context.Context, id string, req book.UpdateBookRequest) (*book.BookDetailResponse, error) {
	// 1. Get existing book
	existing, err := s.repo.GetBookByIDForUpdate(ctx, id)
	if err != nil {
		return nil, err
	}

	// 2. Check version (Optimistic Locking)
	if existing.Version != req.Version {
		return nil, book.ErrVersionConflict
	}

	// 3. Validate foreign keys if changed
	if req.AuthorID != nil && *req.AuthorID != existing.AuthorID.String() {
		if exists, err := s.repo.ValidateAuthor(ctx, *req.AuthorID); err != nil || !exists {
			return nil, book.ErrAuthorNotFound
		}
	}

	if req.CategoryID != nil {
		if exists, err := s.repo.ValidateCategory(ctx, *req.CategoryID); err != nil || !exists {
			return nil, book.ErrCategoryNotFound
		}
	}

	if req.PublisherID != nil {
		if exists, err := s.repo.ValidatePublisher(ctx, *req.PublisherID); err != nil || !exists {
			return nil, book.ErrPublisherNotFound
		}
	}

	// 4. Check ISBN uniqueness (nếu thay đổi ISBN)
	if req.ISBN != nil && (existing.ISBN == "" || *req.ISBN != existing.ISBN) {
		if exists, err := s.repo.CheckISBNExistsExcept(ctx, *req.ISBN, id); err != nil || exists {
			return nil, book.ErrISBNAlreadyExists
		}
	}

	// 5. Update slug nếu title thay đổi
	var newSlug string
	if req.Title != nil && *req.Title != existing.Title {
		newSlug, err = s.repo.GenerateUniqueSlug(ctx, utils.GenerateSlugBook(*req.Title))
		if err != nil {
			return nil, err
		}
	} else {
		newSlug = existing.Slug
	}

	// 6. Apply updates to existing book
	book.ApplyUpdates(*existing, req, newSlug)

	// 7. Save changes
	if err := s.repo.UpdateBook(ctx, existing); err != nil {
		return nil, fmt.Errorf("failed to update book: %w", err)
	}

	// 8. Invalidate cache
	cacheKey := book.GenerateBookDetailCacheKey(id)
	if err := s.cache.Delete(ctx, cacheKey); err != nil {
		log.Printf("[Service] Failed to delete cache: %v", err)
	}
	if err := s.cache.Delete(ctx, "books:list:*"); err != nil {
		log.Printf("[Service] Failed to invalidate list cache: %v", err)
	}

	// 9. Return updated detail
	return s.GetBookDetail(ctx, id)
}

// DELETE
func (s *BookService) DeleteBook(c context.Context, bookID string) (*book.DeleteBookResponse, error) {
	book, err := s.repo.GetBaseBookByID(c, bookID)
	if err != nil {
		return nil, err
	}
	hasActiveOrders, err := s.repo.CheckBookHasActiveOrders(c, bookID)
	if err != nil {
		return nil, fmt.Errorf("failed to check active orders: %w", err)
	}
	if hasActiveOrders {
		return nil, dto.ErrBookHasActiveOrders
	}
	// 3. Check if book has reserved inventory
	hasReservedInventory, err := s.repo.CheckBookHasReservedInventory(c, bookID)
	if err != nil {
		return nil, fmt.Errorf("failed to check reserved inventory: %w", err)
	}
	if hasReservedInventory {
		return nil, dto.ErrBookHasReservedInventory
	}

	// 4. Perform soft delete
	deletedAt := time.Now()
	if err := s.repo.SoftDeleteBook(c, bookID, deletedAt); err != nil {
		return nil, fmt.Errorf("failed to delete book: %w", err)
	}

	// 6. Invalidate cache
	cacheKey := dto.GenerateBookDetailCacheKey(bookID)
	if err := s.cache.Delete(c, cacheKey); err != nil {
		log.Printf("[Service] Failed to delete cache: %v", err)
	}

	// Invalidate list cache
	if err := s.cache.Delete(c, "books:list:*"); err != nil {
		log.Printf("[Service] Failed to invalidate list cache: %v", err)
	}

	// 7. Return deleted book info
	return &dto.DeleteBookResponse{
		ID:        bookID,
		Title:     book.Title,
		DeletedAt: deletedAt,
	}, nil
}

// ====================== SEARCH BOOK SERVICE ==============================
func (s *BookService) SearchBooks(ctx context.Context, req dto.SearchBooksRequest) ([]dto.BookSearchResponse, error) {
	// 1. Generate cache key
	cacheKey := generateSearchCacheKey(req)

	// 2. Try to get from cache
	var cachedResults []dto.BookSearchResponse
	found, err := s.cache.Get(ctx, cacheKey, &cachedResults)
	if found {
		log.Printf("[Service] Search cache HIT: %s", cacheKey)
		return cachedResults, nil
	}
	if err != nil {
		log.Printf("[Service] Search cache error: %v", err)
	}

	// 3. Cache MISS - query database
	log.Printf("[Service] Search cache MISS: %s", cacheKey)
	results, err := s.repo.SearchBooks(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to search books: %w", err)
	}

	// 4. Cache the results (TTL 1 hour = 3600 seconds)
	if err := s.cache.Set(ctx, cacheKey, results, 3600); err != nil {
		log.Printf("[Service] Failed to cache search results: %v", err)
		// Don't fail request if cache write fails
	}

	return results, nil
}

// generateSearchCacheKey - Create consistent cache key for search params
func generateSearchCacheKey(req dto.SearchBooksRequest) string {
	// Create hash from query params
	data := fmt.Sprintf("q=%s|lang=%s|limit=%d", req.Query, req.Language, req.Limit)
	hash := md5.Sum([]byte(data))
	return fmt.Sprintf("books:search:%x", hash)
}
