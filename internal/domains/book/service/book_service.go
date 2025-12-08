package service

import (
	model "bookstore-backend/internal/domains/book/model"
	"bookstore-backend/internal/domains/book/repository"
	"bookstore-backend/internal/infrastructure/storage"
	types "bookstore-backend/internal/shared"
	"bookstore-backend/internal/shared/utils"
	"bookstore-backend/pkg/cache"
	"bookstore-backend/pkg/logger"
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg" // Registers JPEG decoder
	_ "image/png"  // Registers PNG decoder
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"github.com/xuri/excelize/v2"
)

// Service - Implements ServiceInterface
type BookService struct {
	repo           repository.RepositoryInterface
	imageRepo      repository.BookImageRepository
	cache          cache.Cache
	imageProcessor *storage.ImageProcessor
	minio          *storage.MinIOStorage
	asynqClient    *asynq.Client
}

// NewService - Constructor with DI
func NewService(
	repo repository.RepositoryInterface,
	cache cache.Cache,
	imageProcessor *storage.ImageProcessor,
	minio *storage.MinIOStorage,
	imageRepo repository.BookImageRepository,
	asynqClient *asynq.Client,
) ServiceInterface {
	return &BookService{
		repo:           repo,
		cache:          cache,
		imageProcessor: imageProcessor,
		minio:          minio,
		imageRepo:      imageRepo,
		asynqClient:    asynqClient,
	}
}

// ListBooks - Business logic for listing books
func (s *BookService) ListBooks(ctx context.Context, req model.ListBooksRequest) ([]model.ListBooksResponse, *model.PaginationMeta, error) {
	// Validate input
	if err := model.ValidateListRequest(req); err != nil {
		return nil, nil, err
	}
	// 1) Định nghĩa type rõ ràng (dễ debug hơn)
	type BooksCache struct {
		Data       []model.ListBooksResponse `json:"data"`
		Pagination model.PaginationMeta      `json:"pagination"`
	}
	var result BooksCache

	// Generate cache key from request parameters
	cacheKey := model.GenerateCacheKey("books:list", req)
	found, err := s.cache.Get(ctx, cacheKey, &result)
	// Try to get from cache first
	if err != nil {
		return nil, nil, err
	}
	if found {
		return result.Data, &result.Pagination, nil
	}

	// Cache MISS - query database
	log.Printf("Cache MISS for key: %s", cacheKey)

	// Build filter for repository
	filter := &model.BookFilter{
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
		if totalCount == 0 {
			return []model.ListBooksResponse{}, &model.PaginationMeta{}, nil
		}
		return nil, nil, fmt.Errorf("list books error: %w", err)
	}

	// Map entities to DTOs
	responses := make([]model.ListBooksResponse, len(books))
	for i, book := range books {
		responses[i] = model.BookToListDTO(book)
	}

	// Calculate pagination metadata
	totalPages := (totalCount + req.Limit - 1) / req.Limit
	meta := &model.PaginationMeta{
		Page:      req.Page,
		PageSize:  req.Limit,
		Total:     totalCount,
		TotalPage: totalPages,
	}

	// Cache the result
	cacheData := BooksCache{
		Data:       responses,
		Pagination: *meta,
	}

	if err := s.cache.Set(ctx, cacheKey, cacheData, 24*time.Hour); err != nil {
		log.Printf("Cache SET error for key %s: %v", cacheKey, err)
	}

	return responses, meta, nil
}
func (s *BookService) GetBookDetail(ctx context.Context, id string) (*model.BookDetailResponse, error) {
	// Lấy dữ liệu chi tiết sách
	b, inventories, err := s.repo.GetBookByID(ctx, id)

	if err != nil {
		return nil, err
	}
	// Lấy review nổi bật
	reviews, _ := s.repo.GetReviewsHighlight(ctx, id)
	// Build DTO cha
	detail := model.ToBookDetailResponse(*b, inventories, reviews)
	// Tăng view_count async
	go s.repo.IncrementViewCount(context.Background(), id)
	return detail, nil
}

// CreateBook - Business logic for creating book
func (s *BookService) CreateBook(ctx context.Context, req model.CreateBookRequest) error {
	// 1. Validate foreign keys exist
	logger.Info("Validate Author", map[string]interface{}{
		"AuthorID": req.AuthorID,
	})
	if exists, err := s.repo.ValidateAuthor(ctx, req.AuthorID); err != nil || !exists {
		return model.ErrAuthorNotFound
	}

	if req.CategoryID != "" {
		if exists, err := s.repo.ValidateCategory(ctx, req.CategoryID); err != nil || !exists {
			return model.ErrCategoryNotFound
		}
	}

	if req.PublisherID != "" {
		if exists, err := s.repo.ValidatePublisher(ctx, req.PublisherID); err != nil || !exists {
			return model.ErrPublisherNotFound
		}
	}

	// 2. Check ISBN uniqueness (nếu có ISBN)
	if req.ISBN != "" {
		if exists, err := s.repo.CheckISBNExists(ctx, req.ISBN); err != nil || exists {
			return model.ErrISBNAlreadyExists
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

	book := model.ToBookEntity(req, finalSlug)

	// 6. Save to database
	bookID, err := s.repo.CreateBook(ctx, book)
	if err != nil {
		return fmt.Errorf("failed to create book: %w", err)
	}
	// Đầu vào: req.Images []string là mảng URL ảnh (đã upload lên minio hoặc external)
	for i, imgURL := range req.Images {
		// 1. Fetch ảnh về memory
		imgBytes, format, err := s.downloadAndValidateImage(imgURL)
		if err != nil {
			log.Printf("[WARN] Image %d invalid/fetch failed: %v", i, err)
			continue // Bỏ qua ảnh lỗi
		}
		// 2. Upload lên MinIO (original)
		key := fmt.Sprintf("books/%s/%d_original.%s", bookID.String(), i, format)
		origURL, err := s.minio.Upload(ctx, key, imgBytes, "image/"+format)
		if err != nil {
			log.Printf("[WARN] Cannot upload original %d: %v", i, err)
			continue
		}
		// 3. Lưu record book_images vào db
		imgRec := &model.BookImage{
			BookID:        bookID.String(),
			OriginalURL:   origURL,
			SortOrder:     i,
			IsCover:       i == 0,
			Status:        model.ImageStatusProcessing,
			Format:        &format,
			FileSizeBytes: int64Ptr(int64(len(imgBytes))),
		}
		if err := s.imageRepo.Create(ctx, imgRec); err != nil {
			log.Printf("[WARN] Cannot create book_image record: %v", err)
			continue
		}
		// 4. Enqueue job xử lý (resize/upload các variant)
		p := map[string]string{"image_id": imgRec.ID}
		payload, _ := json.Marshal(p)
		job := asynq.NewTask(types.TypeProcessBookImage, payload)
		s.asynqClient.Enqueue(job, asynq.Queue(types.QueueBook), asynq.MaxRetry(2))
	}

	// 7. Invalidate list cache (xóa cache danh sách sách)
	if err := s.cache.Delete(ctx, "books:list:*"); err != nil {
		log.Printf("[Service] Failed to invalidate list cache: %v", err)
	}

	// 8. Get full detail (with joins) to return
	return nil
}
func int64Ptr(i int64) *int64 {
	return &i
}
func (s *BookService) downloadAndValidateImage(url string) ([]byte, string, error) {
	// ✅ 1. HTTP client với timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("create request failed: %w", err)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	// ✅ 2. Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("bad http status: %d %s", resp.StatusCode, resp.Status)
	}

	// ✅ 3. Read body với limit (tránh OOM nếu file quá lớn)
	const maxSize = 10 * 1024 * 1024 // 10MB
	limitedReader := io.LimitReader(resp.Body, maxSize)

	bts, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, "", fmt.Errorf("read body failed: %w", err)
	}

	// ✅ 4. Validate format và detect extension (chỉ 1 lần)
	_, format, err := image.DecodeConfig(bytes.NewReader(bts))
	if err != nil {
		return nil, "", fmt.Errorf("not a valid image: %w", err)
	}

	// ✅ 5. Map format sang extension
	formatMap := map[string]string{
		"jpeg": "jpeg",
		"png":  "png",
		// "gif":  "gif",  // Uncomment nếu support
		// "webp": "webp", // Uncomment nếu support
	}

	ext, ok := formatMap[format]
	if !ok {
		return nil, "", fmt.Errorf("unsupported format: %s (allowed: jpeg, png)", format)
	}

	// ✅ 6. Validate size (nếu ImageProcessor chưa check)
	if err := s.imageProcessor.ValidateImage(bts); err != nil {
		return nil, "", fmt.Errorf("image validation failed: %w", err)
	}

	logger.Info("Image downloaded and validated", map[string]interface{}{
		"url":    url,
		"format": format,
		"size":   len(bts),
	})

	return bts, ext, nil
}

// UPDATE BOOK
// UpdateBook - Business logic for updating book
func (s *BookService) UpdateBook(ctx context.Context, id string, req model.UpdateBookRequest) (*model.BookDetailResponse, error) {
	// 1. Get existing book
	existing, err := s.repo.GetBookByIDForUpdate(ctx, id)
	if err != nil {
		return nil, err
	}

	// 2. Check version (Optimistic Locking)
	if existing.Version != req.Version {
		return nil, model.ErrVersionConflict
	}

	// 3. Validate foreign keys if changed
	if req.AuthorID != nil && *req.AuthorID != existing.AuthorID.String() {
		if exists, err := s.repo.ValidateAuthor(ctx, *req.AuthorID); err != nil || !exists {
			return nil, model.ErrAuthorNotFound
		}
	}

	if req.CategoryID != nil {
		if exists, err := s.repo.ValidateCategory(ctx, *req.CategoryID); err != nil || !exists {
			return nil, model.ErrCategoryNotFound
		}
	}

	if req.PublisherID != nil {
		if exists, err := s.repo.ValidatePublisher(ctx, *req.PublisherID); err != nil || !exists {
			return nil, model.ErrPublisherNotFound
		}
	}

	// 4. Check ISBN uniqueness (nếu thay đổi ISBN)
	if req.ISBN != nil && (existing.ISBN == "" || *req.ISBN != existing.ISBN) {
		if exists, err := s.repo.CheckISBNExistsExcept(ctx, *req.ISBN, id); err != nil || exists {
			return nil, model.ErrISBNAlreadyExists
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
	model.ApplyUpdates(*existing, req, newSlug)

	// 7. Save changes
	if err := s.repo.UpdateBook(ctx, existing); err != nil {
		return nil, fmt.Errorf("failed to update book: %w", err)
	}

	// 8. Invalidate cache
	cacheKey := model.GenerateBookDetailCacheKey(id)
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
func (s *BookService) DeleteBook(c context.Context, bookID string) (*model.DeleteBookResponse, error) {
	book, err := s.repo.GetBaseBookByID(c, bookID)
	if err != nil {
		return nil, err
	}
	hasActiveOrders, err := s.repo.CheckBookHasActiveOrders(c, bookID)
	if err != nil {
		return nil, fmt.Errorf("failed to check active orders: %w", err)
	}
	if hasActiveOrders {
		return nil, model.ErrBookHasActiveOrders
	}
	// 3. Check if book has reserved inventory
	hasReservedInventory, err := s.repo.CheckBookHasReservedInventory(c, bookID)
	if err != nil {
		return nil, fmt.Errorf("failed to check reserved inventory: %w", err)
	}
	if hasReservedInventory {
		return nil, model.ErrBookHasReservedInventory
	}

	// 4. Perform soft delete
	deletedAt := time.Now()
	if err := s.repo.SoftDeleteBook(c, bookID, deletedAt); err != nil {
		return nil, fmt.Errorf("failed to delete book: %w", err)
	}

	// 6. Invalidate cache
	cacheKey := model.GenerateBookDetailCacheKey(bookID)
	if err := s.cache.Delete(c, cacheKey); err != nil {
		log.Printf("[Service] Failed to delete cache: %v", err)
	}

	// Invalidate list cache
	if err := s.cache.Delete(c, "books:list:*"); err != nil {
		log.Printf("[Service] Failed to invalidate list cache: %v", err)
	}

	// 7. Return deleted book info
	return &model.DeleteBookResponse{
		ID:        bookID,
		Title:     book.Title,
		DeletedAt: deletedAt,
	}, nil
}

// ====================== SEARCH BOOK SERVICE ==============================
func (s *BookService) SearchBooks(ctx context.Context, req model.SearchBooksRequest) ([]model.BookSearchResponse, error) {
	// 1. Generate cache key
	cacheKey := generateSearchCacheKey(req)

	// 2. Try to get from cache
	var cachedResults []model.BookSearchResponse
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
	if err := s.cache.Set(ctx, cacheKey, results, 60*time.Minute); err != nil {
		log.Printf("[Service] Failed to cache search results: %v", err)
		// Don't fail request if cache write fails
	}

	return results, nil
}

// generateSearchCacheKey - Create consistent cache key for search params
func generateSearchCacheKey(req model.SearchBooksRequest) string {
	// Create hash from query params
	data := fmt.Sprintf("q=%s|lang=%s|limit=%d", req.Query, req.Language, req.Limit)
	hash := md5.Sum([]byte(data))
	return fmt.Sprintf("books:search:%x", hash)
}
func (s *BookService) ExportBooksToExcel(
	ctx context.Context,
	req model.ListBooksRequest,
) (*excelize.File, *[]model.ListBooksResponse, error) {
	// 1. Ép limit tối đa = 100
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	// 2. Chỉ export book active (mặc định list đã filter is_active = true)
	// Nếu ListBooks đã xử lý is_active trong repo, không cần set thêm
	// Nếu cần, có thể set:
	// req.IsActive = utils.BoolPtr(true)

	// 3. Lấy dữ liệu từ ListBooks
	books, _, err := s.ListBooks(ctx, req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list books: %w", err)
	}
	// 4. Tạo file Excel bằng excelize
	f, err := s.buildBooksExcelFile(books)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build excel file: %w", err)
	}

	return f, &books, nil
}
func (s *BookService) buildBooksExcelFile(books []model.ListBooksResponse) (*excelize.File, error) {
	f := excelize.NewFile()

	sheetName := "Book list"
	// Rename default sheet
	f.SetSheetName("Sheet1", sheetName)

	// Row 1: Header
	headers := []string{
		"ID",
		"Title",
		"Slug",
		"Author",
		"Publisher",
		"Category",
		"Price",
		"Compare Price",
		"Language",
		"Format",
		"Rating Average",
		"Rating Count",
		"View Count",
		"Sold Count",
		"Is Featured",
		"Total Stock",
		"Created At",
		"Cover URL",
		"Images",
		"Meta Title",
		"Meta Description",
		"Meta Keywords",
	}

	for colIdx, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 1) // (col, row=1)
		f.SetCellValue(sheetName, cell, header)
	}

	// Optional: style header (bold)
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
		},
	})
	if err == nil {
		f.SetCellStyle(sheetName, "A1", "V1", headerStyle)
	}

	// Data rows, bắt đầu từ row 2
	for i, b := range books {
		rowNum := i + 2

		// Helpers
		rowStr := func(col int) string {
			cell, _ := excelize.CoordinatesToCellName(col, rowNum)
			return cell
		}

		// ID
		f.SetCellValue(sheetName, rowStr(1), b.ID.String())
		// Title
		f.SetCellValue(sheetName, rowStr(2), b.Title)
		// Slug
		f.SetCellValue(sheetName, rowStr(3), b.Slug)
		// Author
		f.SetCellValue(sheetName, rowStr(4), b.AuthorName)
		// Publisher
		f.SetCellValue(sheetName, rowStr(5), b.PublisherName)
		// Category
		f.SetCellValue(sheetName, rowStr(6), b.CategoryName)

		// Price (decimal → float64)
		f.SetCellValue(sheetName, rowStr(7), b.Price.InexactFloat64())

		// Compare Price
		if b.CompareAtPrice != nil {
			f.SetCellValue(sheetName, rowStr(8), b.CompareAtPrice.InexactFloat64())
		} else {
			f.SetCellValue(sheetName, rowStr(8), nil)
		}

		// Language
		f.SetCellValue(sheetName, rowStr(9), b.Language)

		// Format
		if b.Format != nil {
			f.SetCellValue(sheetName, rowStr(10), *b.Format)
		} else {
			f.SetCellValue(sheetName, rowStr(10), nil)
		}

		// Rating Average
		f.SetCellValue(sheetName, rowStr(11), b.RatingAverage)
		// Rating Count
		f.SetCellValue(sheetName, rowStr(12), b.RatingCount)
		// View Count
		f.SetCellValue(sheetName, rowStr(13), b.ViewCount)
		// Sold Count
		f.SetCellValue(sheetName, rowStr(14), b.SoldCount)
		// Is Featured (TRUE/FALSE)
		f.SetCellValue(sheetName, rowStr(15), b.IsFeatured)
		// Total Stock
		f.SetCellValue(sheetName, rowStr(16), b.TotalStock)

		// Created At (YYYY-MM-DD HH:MM:SS)
		f.SetCellValue(sheetName, rowStr(17), b.CreatedAt.Format("2006-01-02 15:04:05"))

		// Cover URL
		if b.CoverURL != nil {
			f.SetCellValue(sheetName, rowStr(18), *b.CoverURL)
		} else {
			f.SetCellValue(sheetName, rowStr(18), nil)
		}

		// Images (join by |)
		if len(b.Images) > 0 {
			f.SetCellValue(sheetName, rowStr(19), strings.Join(b.Images, "|"))
		} else {
			f.SetCellValue(sheetName, rowStr(19), "")
		}

		// Meta Title
		if b.MetaTitle != nil {
			f.SetCellValue(sheetName, rowStr(20), *b.MetaTitle)
		} else {
			f.SetCellValue(sheetName, rowStr(20), nil)
		}

		// Meta Description
		if b.MetaDescription != nil {
			f.SetCellValue(sheetName, rowStr(21), *b.MetaDescription)
		} else {
			f.SetCellValue(sheetName, rowStr(21), nil)
		}

		// Meta Keywords (join by |)
		if len(b.MetaKeywords) > 0 {
			f.SetCellValue(sheetName, rowStr(22), strings.Join(b.MetaKeywords, "|"))
		} else {
			f.SetCellValue(sheetName, rowStr(22), "")
		}
	}

	// Optional: auto width
	if err := f.SetColWidth(sheetName, "A", "V", 18); err != nil {
		// ignore error
	}

	return f, nil
}
func (s *BookService) GetBooksByIDs(ctx context.Context, ids []string) ([]model.BookDetailResponse, error) {
	if len(ids) == 0 {
		return []model.BookDetailResponse{}, nil
	}

	// 1. Get books from repository
	books, err := s.repo.GetBooksByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to get books by IDs: %w", err)
	}

	// 2. Map to DTOs
	responses := make([]model.BookDetailResponse, len(books))
	for i, book := range books {
		// Note: Batch fetch currently doesn't include inventories or reviews for performance
		// If needed, we can add batch fetching for those as well
		responses[i] = *model.BookEntityToDetailResponse(book)
	}

	return responses, nil
}

func (s *BookService) GetBooksCheckout(ctx context.Context, ids []string) ([]model.BookCheckoutResponse, error) {
	books, err := s.repo.GetBooksCheckout(ctx, ids)
	if err != nil {
		return nil, err
	}
	return books, nil
}
