package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	authorModel "bookstore-backend/internal/domains/author/model"
	authorRepo "bookstore-backend/internal/domains/author/repository"
	"bookstore-backend/internal/domains/book/model"
	"bookstore-backend/internal/domains/book/repository"
	"bookstore-backend/internal/domains/category"
	publisherModel "bookstore-backend/internal/domains/publisher/model"
	publisherRepo "bookstore-backend/internal/domains/publisher/repository"
	"bookstore-backend/internal/infrastructure/storage"
	"bookstore-backend/internal/shared"
	"bookstore-backend/internal/shared/utils"
	"bookstore-backend/pkg/database"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
)

// BulkImportServiceInterface defines bulk import operations
type BulkImportServiceInterface interface {
	// ImportBooks processes CSV file and creates books (sync mode)
	ImportBooks(ctx context.Context, file *multipart.FileHeader, userID string) (*model.BulkImportResult, error)
}

type bulkImportService struct {
	// Repositories
	bookRepo       repository.RepositoryInterface
	bookImageRepo  repository.BookImageRepository
	authorRepo     authorRepo.RepositoryInterface
	categoryRepo   category.CategoryRepository
	publisherRepo  publisherRepo.RepositoryInterface
	bulkImportRepo repository.BookImageRepository

	// Infrastructure
	pool           *pgxpool.Pool // pgxpool
	minioStorage   *storage.MinIOStorage
	imageProcessor *storage.ImageProcessor
	asynqClient    *asynq.Client
}

// NewBulkImportService creates a new bulk import service
func NewBulkImportService(
	bookRepo repository.RepositoryInterface,
	bookImageRepo repository.BookImageRepository,
	authorRepo authorRepo.RepositoryInterface,
	categoryRepo category.CategoryRepository,
	publisherRepo publisherRepo.RepositoryInterface,
	bulkImportRepo repository.BookImageRepository,
	pool *pgxpool.Pool,
	minioStorage *storage.MinIOStorage,
	imageProcessor *storage.ImageProcessor,
	asynqClient *asynq.Client,
) BulkImportServiceInterface {
	return &bulkImportService{
		bookRepo:       bookRepo,
		bookImageRepo:  bookImageRepo,
		authorRepo:     authorRepo,
		categoryRepo:   categoryRepo,
		publisherRepo:  publisherRepo,
		bulkImportRepo: bulkImportRepo,
		pool:           pool,
		minioStorage:   minioStorage,
		imageProcessor: imageProcessor,
		asynqClient:    asynqClient,
	}
}

// ImportBooks là main entry point cho bulk import (sync mode)
func (s *bulkImportService) ImportBooks(ctx context.Context, file *multipart.FileHeader, userID string) (*model.BulkImportResult, error) {
	log.Info().
		Str("user_id", userID).
		Str("file_name", file.Filename).
		Int64("file_size", file.Size).
		Msg("Starting bulk import books")

	// PHASE 1: Parse CSV file
	csvRows, err := s.parseCSVFile(file)
	if err != nil {
		return &model.BulkImportResult{
			Success:   false,
			TotalRows: 0,
			Errors: []model.ImportValidationError{
				{Row: 0, Field: "file", Error: err.Error()},
			},
		}, nil
	}

	totalRows := len(csvRows)
	log.Info().Int("total_rows", totalRows).Msg("CSV parsed successfully")

	// Check row limit (1000 rows max)
	if totalRows > 1000 {
		return &model.BulkImportResult{
			Success:   false,
			TotalRows: totalRows,
			Errors: []model.ImportValidationError{
				{Row: 0, Field: "file", Error: "file exceeds 1000 rows limit"},
			},
		}, nil
	}

	// PHASE 2: Validate ALL rows (không insert gì)
	validationErrors := s.validateAllRows(ctx, csvRows)
	if len(validationErrors) > 0 {
		log.Warn().
			Int("error_count", len(validationErrors)).
			Msg("Validation failed")

		return &model.BulkImportResult{
			Success:    false,
			TotalRows:  totalRows,
			FailedRows: len(validationErrors),
			Errors:     validationErrors,
		}, nil
	}

	log.Info().Msg("All rows validated successfully")

	// PHASE 3: Download & Upload images (NGOÀI transaction)
	imageResults, err := s.uploadImagesForAllRows(ctx, csvRows)
	if err != nil {
		// Cleanup uploaded images
		s.cleanupUploadedImages(ctx, imageResults)

		return &model.BulkImportResult{
			Success:   false,
			TotalRows: totalRows,
			Errors: []model.ImportValidationError{
				{Row: 0, Field: "images", Error: err.Error()},
			},
		}, nil
	}

	log.Info().Msg("All images uploaded successfully")

	// PHASE 4: Create entities & books (TRONG transaction)
	createdBooks, err := s.createBooksInTransaction(ctx, csvRows, imageResults)
	if err != nil {
		// Cleanup uploaded images
		s.cleanupUploadedImages(ctx, imageResults)

		return &model.BulkImportResult{
			Success:   false,
			TotalRows: totalRows,
			Errors: []model.ImportValidationError{
				{Row: 0, Field: "transaction", Error: err.Error()},
			},
		}, nil
	}

	log.Info().
		Int("success_count", len(createdBooks)).
		Msg("Bulk import completed successfully")

	// Return success result
	return &model.BulkImportResult{
		Success:      true,
		TotalRows:    totalRows,
		SuccessRows:  len(createdBooks),
		CreatedBooks: createdBooks,
	}, nil
}

// parseCSVFile parses uploaded CSV file thành CSVBookRow structs
func (s *bulkImportService) parseCSVFile(file *multipart.FileHeader) ([]model.CSVBookRow, error) {
	// Open file
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	// Create CSV reader
	reader := csv.NewReader(src)
	reader.TrimLeadingSpace = true

	// Read all rows
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV file is empty (no data rows)")
	}

	// Parse header
	header := records[0]
	colIndexMap := s.buildColumnIndexMap(header)

	// Parse data rows
	var csvRows []model.CSVBookRow
	for i, record := range records[1:] { // Skip header
		row, err := s.parseCSVRow(record, colIndexMap, i+2) // Row number starts from 2
		if err != nil {
			return nil, fmt.Errorf("failed to parse row %d: %w", i+2, err)
		}
		csvRows = append(csvRows, row)
	}

	return csvRows, nil
}

// buildColumnIndexMap tạo map từ column name → index
func (s *bulkImportService) buildColumnIndexMap(header []string) map[string]int {
	colMap := make(map[string]int)
	for i, colName := range header {
		colMap[strings.TrimSpace(strings.ToLower(colName))] = i
	}
	return colMap
}

// parseCSVRow parse một row CSV thành CSVBookRow struct
func (s *bulkImportService) parseCSVRow(record []string, colMap map[string]int, rowNum int) (model.CSVBookRow, error) {
	row := model.CSVBookRow{
		Row: rowNum,
	}

	// Helper to get column value safely
	getCol := func(name string) string {
		if idx, ok := colMap[name]; ok && idx < len(record) {
			return strings.TrimSpace(record[idx])
		}
		return ""
	}

	// Parse required fields
	row.Title = getCol("title")
	row.AuthorName = getCol("author_name")
	row.CategoryName = getCol("category_name")
	row.PublisherName = getCol("publisher_name")

	// Parse price (required)
	priceStr := getCol("price")
	if priceStr != "" {
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			return row, fmt.Errorf("invalid price: %s", priceStr)
		}
		row.Price = price
	}

	// Parse optional numeric fields
	if val := getCol("compare_at_price"); val != "" {
		if price, err := strconv.ParseFloat(val, 64); err == nil {
			row.CompareAtPrice = &price
		}
	}

	if val := getCol("cost_price"); val != "" {
		if price, err := strconv.ParseFloat(val, 64); err == nil {
			row.CostPrice = &price
		}
	}

	if val := getCol("pages"); val != "" {
		if pages, err := strconv.Atoi(val); err == nil {
			row.Pages = &pages
		}
	}

	if val := getCol("published_year"); val != "" {
		if year, err := strconv.Atoi(val); err == nil {
			row.PublishedYear = &year
		}
	}

	if val := getCol("weight_grams"); val != "" {
		if weight, err := strconv.Atoi(val); err == nil {
			row.WeightGrams = &weight
		}
	}

	// Parse optional string fields
	if val := getCol("isbn"); val != "" {
		row.ISBN = &val
	}
	if val := getCol("description"); val != "" {
		row.Description = &val
	}
	if val := getCol("language"); val != "" {
		row.Language = &val
	}
	if val := getCol("format"); val != "" {
		row.Format = &val
	}
	if val := getCol("dimensions"); val != "" {
		row.Dimensions = &val
	}
	if val := getCol("meta_title"); val != "" {
		row.MetaTitle = &val
	}
	if val := getCol("meta_description"); val != "" {
		row.MetaDesc = &val
	}

	// Parse image URLs (image_url_1 to image_url_7)
	for i := 1; i <= 7; i++ {
		colName := fmt.Sprintf("image_url_%d", i)
		if url := getCol(colName); url != "" {
			row.ImageURLs = append(row.ImageURLs, url)
		}
	}

	// Parse meta_keywords (pipe-delimited: "keyword1|keyword2|keyword3")
	if val := getCol("meta_keywords"); val != "" {
		keywords := strings.Split(val, "|")
		for _, kw := range keywords {
			kw = strings.TrimSpace(kw)
			if kw != "" {
				row.MetaKeywords = append(row.MetaKeywords, kw)
			}
		}
	}

	return row, nil
}

// validateAllRows validates tất cả rows, returns list of errors
func (s *bulkImportService) validateAllRows(ctx context.Context, rows []model.CSVBookRow) []model.ImportValidationError {
	var errors []model.ImportValidationError

	// Track duplicates within file
	isbnMap := make(map[string]int) // ISBN -> first row
	slugMap := make(map[string]int) // slug -> first row

	for _, row := range rows {
		// Validate single row
		rowErrors := s.validateRow(row)
		errors = append(errors, rowErrors...)

		// Check duplicate ISBN within file
		if row.ISBN != nil && *row.ISBN != "" {
			if firstRow, exists := isbnMap[*row.ISBN]; exists {
				errors = append(errors, model.ImportValidationError{
					Row:   row.Row,
					Field: "isbn",
					Value: *row.ISBN,
					Error: fmt.Sprintf("duplicate ISBN (also at row %d)", firstRow),
				})
			} else {
				isbnMap[*row.ISBN] = row.Row
			}
		}

		// Check duplicate slug (generated từ title)
		slug := utils.GenerateSlug(row.Title)
		if firstRow, exists := slugMap[slug]; exists {
			errors = append(errors, model.ImportValidationError{
				Row:   row.Row,
				Field: "title",
				Value: row.Title,
				Error: fmt.Sprintf("duplicate title/slug (also at row %d)", firstRow),
			})
		} else {
			slugMap[slug] = row.Row
		}
	}

	return errors
}

// validateRow validates một row, returns list of errors cho row đó
func (s *bulkImportService) validateRow(row model.CSVBookRow) []model.ImportValidationError {
	var errors []model.ImportValidationError

	// 1. Validate required fields
	if row.Title == "" {
		errors = append(errors, model.ImportValidationError{
			Row:   row.Row,
			Field: "title",
			Error: "required field",
		})
	}

	if row.AuthorName == "" {
		errors = append(errors, model.ImportValidationError{
			Row:   row.Row,
			Field: "author_name",
			Error: "required field",
		})
	}

	if row.Price <= 0 {
		errors = append(errors, model.ImportValidationError{
			Row:   row.Row,
			Field: "price",
			Value: fmt.Sprintf("%.2f", row.Price),
			Error: "price must be greater than 0",
		})
	}

	// 2. Validate compare_at_price >= price
	if row.CompareAtPrice != nil && *row.CompareAtPrice < row.Price {
		errors = append(errors, model.ImportValidationError{
			Row:   row.Row,
			Field: "compare_at_price",
			Value: fmt.Sprintf("%.2f", *row.CompareAtPrice),
			Error: "compare_at_price must be >= price",
		})
	}

	// 3. Validate cost_price >= 0
	if row.CostPrice != nil && *row.CostPrice < 0 {
		errors = append(errors, model.ImportValidationError{
			Row:   row.Row,
			Field: "cost_price",
			Value: fmt.Sprintf("%.2f", *row.CostPrice),
			Error: "cost_price must be >= 0",
		})
	}

	// 4. Validate pages (1 - 10000)
	if row.Pages != nil {
		if *row.Pages < 1 || *row.Pages > 10000 {
			errors = append(errors, model.ImportValidationError{
				Row:   row.Row,
				Field: "pages",
				Value: fmt.Sprintf("%d", *row.Pages),
				Error: "pages must be between 1 and 10000",
			})
		}
	}

	// 5. Validate published_year (1000 - current year)
	if row.PublishedYear != nil {
		currentYear := time.Now().Year()
		if *row.PublishedYear < 1000 || *row.PublishedYear > currentYear {
			errors = append(errors, model.ImportValidationError{
				Row:   row.Row,
				Field: "published_year",
				Value: fmt.Sprintf("%d", *row.PublishedYear),
				Error: fmt.Sprintf("published_year must be between 1000 and %d", currentYear),
			})
		}
	}

	// 6. Validate format enum
	if row.Format != nil {
		validFormats := map[string]bool{
			"paperback": true,
			"hardcover": true,
			"ebook":     true,
		}
		if !validFormats[*row.Format] {
			errors = append(errors, model.ImportValidationError{
				Row:   row.Row,
				Field: "format",
				Value: *row.Format,
				Error: "format must be: paperback, hardcover, or ebook",
			})
		}
	}

	// 7. Validate weight_grams > 0
	if row.WeightGrams != nil && *row.WeightGrams <= 0 {
		errors = append(errors, model.ImportValidationError{
			Row:   row.Row,
			Field: "weight_grams",
			Value: fmt.Sprintf("%d", *row.WeightGrams),
			Error: "weight_grams must be greater than 0",
		})
	}

	// 8. Validate images (nếu có)
	if len(row.ImageURLs) > 0 {
		if len(row.ImageURLs) < 3 {
			errors = append(errors, model.ImportValidationError{
				Row:   row.Row,
				Field: "images",
				Error: "minimum 3 images required",
			})
		}
		if len(row.ImageURLs) > 7 {
			errors = append(errors, model.ImportValidationError{
				Row:   row.Row,
				Field: "images",
				Error: "maximum 7 images allowed",
			})
		}
	}

	return errors
}

// uploadImagesForAllRows downloads và uploads ảnh cho tất cả rows
// Returns ImageUploadResult cho từng row
func (s *bulkImportService) uploadImagesForAllRows(ctx context.Context, rows []model.CSVBookRow) ([]model.ImageUploadResult, error) {
	results := make([]model.ImageUploadResult, len(rows))

	// Generate unique job ID for temp folder
	jobID := uuid.New().String()

	for i, row := range rows {
		if len(row.ImageURLs) == 0 {
			// No images for this row
			results[i] = model.ImageUploadResult{
				RowNumber: row.Row,
			}
			continue
		}

		// Upload images for this row
		result, err := s.uploadImagesForRow(ctx, row, jobID)
		if err != nil {
			// Return error immediately (all-or-nothing)
			return results, fmt.Errorf("row %d: %w", row.Row, err)
		}

		results[i] = result
	}

	return results, nil
}

// uploadImagesForRow uploads ảnh cho một row
func (s *bulkImportService) uploadImagesForRow(ctx context.Context, row model.CSVBookRow, jobID string) (model.ImageUploadResult, error) {
	result := model.ImageUploadResult{
		RowNumber: row.Row,
		TempKeys:  make([]string, 0),
	}

	for i, imgURL := range row.ImageURLs {
		// Download and validate image
		imgBytes, format, err := s.downloadAndValidateImage(ctx, imgURL)
		if err != nil {
			result.Error = fmt.Errorf("image %d: %w", i+1, err)
			return result, result.Error
		}

		// Generate temp key: bulk-import-temp/{job_id}/row-{row}/image-{i}_original.{ext}
		tempKey := fmt.Sprintf("bulk-import-temp/%s/row-%d/image-%d_original.%s",
			jobID, row.Row, i, format)

		// Upload to MinIO
		originalURL, err := s.minioStorage.Upload(ctx, tempKey, imgBytes, "image/"+format)
		if err != nil {
			result.Error = fmt.Errorf("failed to upload image %d: %w", i+1, err)
			return result, result.Error
		}

		// Track temp key for cleanup
		result.TempKeys = append(result.TempKeys, tempKey)

		// Prepare BookImage record (chưa insert DB)
		bookImage := model.BookImage{
			ID: uuid.New().String(),
			// BookID: sẽ set sau khi create book
			OriginalURL:   originalURL,
			SortOrder:     i,
			IsCover:       i == 0,
			Status:        "processing", // Defer variant processing
			Format:        &format,
			FileSizeBytes: int64Ptr(int64(len(imgBytes))),
		}

		result.ImageRecords = append(result.ImageRecords, bookImage)
	}

	return result, nil
}

// downloadAndValidateImage downloads image from URL và validates
func (s *bulkImportService) downloadAndValidateImage(ctx context.Context, url string) ([]byte, string, error) {
	// HTTP client với timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("invalid URL: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch: %w", err)
	}
	defer resp.Body.Close()

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Read body (limit 10MB)
	const maxSize = 10 * 1024 * 1024
	limitedReader := io.LimitReader(resp.Body, maxSize)

	imgBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read: %w", err)
	}

	// Validate image format
	_, format, err := image.DecodeConfig(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, "", fmt.Errorf("not a valid image: %w", err)
	}

	// Check format support
	supportedFormats := map[string]bool{
		"jpeg": true,
		"png":  true,
	}

	if !supportedFormats[format] {
		return nil, "", fmt.Errorf("unsupported format: %s (only jpeg/png)", format)
	}

	// Validate with ImageProcessor
	if err := s.imageProcessor.ValidateImage(imgBytes); err != nil {
		return nil, "", fmt.Errorf("validation failed: %w", err)
	}

	return imgBytes, format, nil
}

// Helper function

// createBooksInTransaction creates entities & books trong 1 transaction
func (s *bulkImportService) createBooksInTransaction(
	ctx context.Context,
	rows []model.CSVBookRow,
	imageResults []model.ImageUploadResult,
) ([]string, error) {
	var createdBookIDs []string

	// Execute in transaction
	err := database.WithTransaction(ctx, s.pool, func(tx pgx.Tx) error {
		// Step 1: Create entity cache
		entityCache := model.NewEntityCache()

		// Step 2: Batch create/find entities
		if err := s.prepareCachedEntities(ctx, tx, rows, entityCache); err != nil {
			return fmt.Errorf("failed to prepare entities: %w", err)
		}

		// Step 3: Insert books & images
		for i, row := range rows {
			bookID, err := s.createBookWithImages(ctx, tx, row, entityCache, imageResults[i])
			if err != nil {
				return fmt.Errorf("failed to create book at row %d: %w", row.Row, err)
			}

			createdBookIDs = append(createdBookIDs, bookID)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return createdBookIDs, nil
}

// prepareCachedEntities batch creates/finds entities và cache IDs
func (s *bulkImportService) prepareCachedEntities(
	ctx context.Context,
	tx pgx.Tx,
	rows []model.CSVBookRow,
	cache *model.EntityCache,
) error {
	// Extract unique entity names
	uniqueAuthors := make(map[string]bool)
	uniqueCategories := make(map[string]bool)
	uniquePublishers := make(map[string]bool)

	for _, row := range rows {
		if row.AuthorName != "" {
			normalized := normalizeEntityName(row.AuthorName)
			uniqueAuthors[normalized] = true
		}

		if row.CategoryName != "" {
			normalized := normalizeEntityName(row.CategoryName)
			uniqueCategories[normalized] = true
		}

		if row.PublisherName != "" {
			normalized := normalizeEntityName(row.PublisherName)
			uniquePublishers[normalized] = true
		}
	}

	// Process authors
	for normalizedName := range uniqueAuthors {
		authorID, err := s.findOrCreateAuthor(ctx, tx, normalizedName)
		if err != nil {
			return fmt.Errorf("failed to process author '%s': %w", normalizedName, err)
		}
		cache.Authors[normalizedName] = authorID
	}

	// Process categories
	for normalizedName := range uniqueCategories {
		categoryID, err := s.findOrCreateCategory(ctx, tx, normalizedName)
		if err != nil {
			return fmt.Errorf("failed to process category '%s': %w", normalizedName, err)
		}
		cache.Categories[normalizedName] = categoryID
	}

	// Process publishers
	for normalizedName := range uniquePublishers {
		publisherID, err := s.findOrCreatePublisher(ctx, tx, normalizedName)
		if err != nil {
			return fmt.Errorf("failed to process publisher '%s': %w", normalizedName, err)
		}
		cache.Publishers[normalizedName] = publisherID
	}

	return nil
}

// normalizeEntityName normalizes entity name (lowercase, trim)
func normalizeEntityName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// toTitleCase converts string to Title Case
func toTitleCase(s string) string {
	words := strings.Fields(strings.ToLower(s))
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

// findOrCreateAuthor tìm hoặc tạo author
func (s *bulkImportService) findOrCreateAuthor(ctx context.Context, tx pgx.Tx, normalizedName string) (string, error) {
	// Try find existing (case-insensitive)
	author, err := s.authorRepo.FindByNameCaseInsensitive(ctx, normalizedName)
	if err == nil && author != nil {
		// Found existing author
		return author.ID.String(), nil
	}

	// Not found → Create new
	titleCaseName := toTitleCase(normalizedName)
	slug := utils.GenerateSlug(titleCaseName)

	// Check slug collision
	slug = s.generateUniqueAuthorSlug(ctx, tx, slug)

	// Create author
	newAuthor := &authorModel.Author{
		ID:   uuid.New(),
		Name: titleCaseName,
		Slug: slug,
		Bio:  nil, // Empty
	}

	err = s.authorRepo.CreateWithTx(ctx, tx, newAuthor)
	if err != nil {
		return "", fmt.Errorf("failed to create author: %w", err)
	}

	log.Info().
		Str("author_id", newAuthor.ID.String()).
		Str("name", titleCaseName).
		Str("slug", slug).
		Msg("Created new author")

	return newAuthor.ID.String(), nil
}

// generateUniqueAuthorSlug generates unique slug (handle collision)
func (s *bulkImportService) generateUniqueAuthorSlug(ctx context.Context, tx pgx.Tx, baseSlug string) string {
	slug := baseSlug
	counter := 2

	for {
		existing, _ := s.authorRepo.FindBySlugWithTx(ctx, tx, slug)
		if existing == nil {
			// Slug available
			return slug
		}

		// Collision → Append counter
		slug = fmt.Sprintf("%s-%d", baseSlug, counter)
		counter++
	}
}

// findOrCreateCategory tìm hoặc tạo category
func (s *bulkImportService) findOrCreateCategory(ctx context.Context, tx pgx.Tx, normalizedName string) (string, error) {
	// Try find existing
	cate, err := s.categoryRepo.FindByNameCaseInsensitive(ctx, normalizedName)
	if err == nil && cate != nil {
		return cate.ID.String(), nil
	}

	// Create new
	titleCaseName := toTitleCase(normalizedName)
	slug := utils.GenerateSlug(titleCaseName)
	slug = s.generateUniqueCategorySlug(ctx, tx, slug)

	newCategory := &category.Category{
		ID:          uuid.New(),
		Name:        titleCaseName,
		Slug:        slug,
		Description: "",
		ParentID:    nil,
	}

	err = s.categoryRepo.CreateWithTx(ctx, tx, newCategory)
	if err != nil {
		return "", fmt.Errorf("failed to create category: %w", err)
	}

	log.Info().
		Str("category_id", newCategory.ID.String()).
		Str("name", titleCaseName).
		Msg("Created new category")

	return newCategory.ID.String(), nil
}

// generateUniqueCategorySlug generates unique slug
func (s *bulkImportService) generateUniqueCategorySlug(ctx context.Context, tx pgx.Tx, baseSlug string) string {
	slug := baseSlug
	counter := 2

	for {
		existing, _ := s.categoryRepo.FindBySlugWithTx(ctx, tx, slug)
		if existing == nil {
			return slug
		}
		slug = fmt.Sprintf("%s-%d", baseSlug, counter)
		counter++
	}
}

// findOrCreatePublisher tìm hoặc tạo publisher
func (s *bulkImportService) findOrCreatePublisher(ctx context.Context, tx pgx.Tx, normalizedName string) (string, error) {
	// Try find existing
	publisher, err := s.publisherRepo.FindByNameCaseInsensitive(ctx, normalizedName)
	if err == nil && publisher != nil {
		return publisher.ID.String(), nil
	}

	// Create new
	titleCaseName := toTitleCase(normalizedName)
	slug := utils.GenerateSlug(titleCaseName)
	slug = s.generateUniquePublisherSlug(ctx, tx, slug)

	newPublisher := &publisherModel.Publisher{
		ID:   uuid.New(),
		Name: titleCaseName,
		Slug: slug,
	}

	err = s.publisherRepo.CreateWithTx(ctx, tx, newPublisher)
	if err != nil {
		return "", fmt.Errorf("failed to create publisher: %w", err)
	}

	log.Info().
		Str("publisher_id", newPublisher.ID.String()).
		Str("name", titleCaseName).
		Msg("Created new publisher")

	return newPublisher.ID.String(), nil
}

// generateUniquePublisherSlug generates unique slug
func (s *bulkImportService) generateUniquePublisherSlug(ctx context.Context, tx pgx.Tx, baseSlug string) string {
	slug := baseSlug
	counter := 2

	for {
		existing, _ := s.publisherRepo.FindBySlugWithTx(ctx, tx, slug)
		if existing == nil {
			return slug
		}
		slug = fmt.Sprintf("%s-%d", baseSlug, counter)
		counter++
	}
}

// createBookWithImages creates một book với images trong transaction
func (s *bulkImportService) createBookWithImages(
	ctx context.Context,
	tx pgx.Tx,
	row model.CSVBookRow,
	cache *model.EntityCache,
	imageResult model.ImageUploadResult,
) (string, error) {
	// Map entity IDs from cache
	authorID := cache.Authors[normalizeEntityName(row.AuthorName)]

	var categoryID *string
	if row.CategoryName != "" {
		catID := cache.Categories[normalizeEntityName(row.CategoryName)]
		categoryID = &catID
	}

	var publisherID *string
	if row.PublisherName != "" {
		pubID := cache.Publishers[normalizeEntityName(row.PublisherName)]
		publisherID = &pubID
	}

	// Generate unique slug for book
	baseSlug := utils.GenerateSlug(row.Title)
	slug := s.generateUniqueBookSlug(ctx, tx, baseSlug)

	// Create book
	book := &model.Book{
		ID:              uuid.New(),
		Title:           row.Title,
		Slug:            slug,
		ISBN:            *row.ISBN,
		AuthorID:        utils.ParseStringToUUID(authorID),
		CategoryID:      utils.ParseStringToUUID(*categoryID),
		PublisherID:     utils.ParseStringToUUID(*publisherID),
		Price:           decimal.NewFromFloat(row.Price),
		CompareAtPrice:  utils.ParseFloatToDecimal(row.CompareAtPrice),
		CostPrice:       utils.ParseFloatToDecimal(row.CostPrice),
		Description:     row.Description,
		Pages:           row.Pages,
		Language:        *row.Language,
		PublishedYear:   row.PublishedYear,
		Format:          row.Format,
		Dimensions:      row.Dimensions,
		WeightGrams:     row.WeightGrams,
		IsActive:        false, // Draft mode
		IsFeatured:      false,
		MetaTitle:       row.MetaTitle,
		MetaDescription: row.MetaDesc,
		MetaKeywords:    row.MetaKeywords,
		Version:         0,
	}

	// Insert book
	err := s.bookRepo.CreateBookWithTx(ctx, tx, book)
	if err != nil {
		return "", fmt.Errorf("failed to insert book: %w", err)
	}

	// Process images (nếu có)
	if len(imageResult.ImageRecords) > 0 {
		err = s.processBookImages(ctx, tx, book.ID.String(), imageResult)
		if err != nil {
			return "", fmt.Errorf("failed to process images: %w", err)
		}
	}

	log.Info().
		Str("book_id", book.ID.String()).
		Str("title", book.Title).
		Int("images", len(imageResult.ImageRecords)).
		Msg("Created book")

	return book.ID.String(), nil
}

// generateUniqueBookSlug generates unique slug for book
func (s *bulkImportService) generateUniqueBookSlug(ctx context.Context, tx pgx.Tx, baseSlug string) string {
	slug := baseSlug
	counter := 2

	for {
		existing, _ := s.bookRepo.FindBySlugWithTx(ctx, tx, slug)
		if existing == nil {
			return slug
		}
		slug = fmt.Sprintf("%s-%d", baseSlug, counter)
		counter++
	}
}

// processBookImages moves images từ temp → final location và insert records
func (s *bulkImportService) processBookImages(
	ctx context.Context,
	tx pgx.Tx,
	bookID string,
	imageResult model.ImageUploadResult,
) error {
	// Prepare images cho batch insert
	var imagesToInsert []*model.BookImage

	for i, imageRecord := range imageResult.ImageRecords {
		// Set book_id
		imageRecord.BookID = bookID

		// Move image từ temp → final location
		tempKey := imageResult.TempKeys[i]
		finalKey := fmt.Sprintf("books/%s/%d_original.%s", bookID, i, *imageRecord.Format)

		err := s.minioStorage.MoveObject(ctx, tempKey, finalKey)
		if err != nil {
			return fmt.Errorf("failed to move image %d: %w", i, err)
		}

		// Update URL to final location
		finalURL := fmt.Sprintf("http://localhost:9000/bookstore/%s", finalKey)
		imageRecord.OriginalURL = finalURL

		imagesToInsert = append(imagesToInsert, &imageRecord)

		// Enqueue job xử lý variant (defer processing)
		payload := map[string]string{"image_id": imageRecord.ID}
		payloadBytes, _ := json.Marshal(payload)
		task := asynq.NewTask(shared.TypeProcessBookImage, payloadBytes)

		_, err = s.asynqClient.Enqueue(task, asynq.Queue("default"), asynq.MaxRetry(2))
		if err != nil {
			log.Warn().
				Err(err).
				Str("image_id", imageRecord.ID).
				Msg("Failed to enqueue image processing job")
		}
	}

	// Batch insert book_images
	err := s.bookImageRepo.CreateBatchWithTx(ctx, tx, imagesToInsert)
	if err != nil {
		return fmt.Errorf("failed to insert book_images: %w", err)
	}

	return nil
}

// cleanupUploadedImages xóa tất cả temp images nếu transaction fail
func (s *bulkImportService) cleanupUploadedImages(ctx context.Context, imageResults []model.ImageUploadResult) {
	var keysToDelete []string

	for _, result := range imageResults {
		keysToDelete = append(keysToDelete, result.TempKeys...)
	}

	if len(keysToDelete) == 0 {
		return
	}

	log.Info().
		Int("count", len(keysToDelete)).
		Msg("Cleaning up uploaded images")

	err := s.minioStorage.RemoveObjects(ctx, keysToDelete)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Failed to cleanup images")
	}
}
