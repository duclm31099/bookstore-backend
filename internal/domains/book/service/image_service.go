package service

import (
	"bookstore-backend/internal/domains/book/model"
	"bookstore-backend/internal/domains/book/repository"
	"bookstore-backend/internal/infrastructure/storage"
	"bookstore-backend/pkg/logger"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type BookImageService interface {
	CreateImagesFromURLs(ctx context.Context, bookID string, imageURLs []string) ([]*model.BookImage, error)
	ProcessImage(ctx context.Context, imageID string) error
	GetBookImages(ctx context.Context, bookID string) ([]*model.BookImage, error)
	DeleteBookImages(ctx context.Context, bookID string) error
}

type bookImageService struct {
	repo           repository.BookImageRepository
	storage        *storage.MinIOStorage
	imageProcessor *storage.ImageProcessor
}

func NewBookImageService(
	repo repository.BookImageRepository,
	storage *storage.MinIOStorage,
	imageProcessor *storage.ImageProcessor,
) BookImageService {
	return &bookImageService{
		repo:           repo,
		storage:        storage,
		imageProcessor: imageProcessor,
	}
}

// CreateImagesFromURLs tạo các book_images record từ mảng URL
// Được gọi từ API khi tạo/update book
func (s *bookImageService) CreateImagesFromURLs(ctx context.Context, bookID string, imageURLs []string) ([]*model.BookImage, error) {
	var images []*model.BookImage

	for i, imgURL := range imageURLs {
		// Download ảnh từ URL (có thể là URL external hoặc đã upload lên MinIO)
		imageData, format, err := s.downloadAndValidateImage(imgURL)
		if err != nil {
			logger.Info("Failed to download image", map[string]interface{}{
				"url":   imgURL,
				"error": err.Error(),
			})
			continue // Skip ảnh lỗi, tiếp tục với ảnh khác
		}

		// Upload ảnh original lên MinIO
		key := fmt.Sprintf("books/%s/%d_original.%s", bookID, i, format)
		originalURL, err := s.storage.Upload(ctx, key, imageData, fmt.Sprintf("image/%s", format))
		if err != nil {
			logger.Info("Failed to upload image to MinIO", map[string]interface{}{
				"key":   key,
				"error": err.Error(),
			})
			continue
		}

		// Tạo book_image record trong DB
		image := &model.BookImage{
			BookID:        bookID,
			OriginalURL:   originalURL,
			SortOrder:     i,
			IsCover:       i == 0, // Ảnh đầu tiên là cover
			Status:        model.ImageStatusProcessing,
			Format:        &format,
			FileSizeBytes: intPtr(int64(len(imageData))),
		}

		err = s.repo.Create(ctx, image)
		if err != nil {
			logger.Info("Failed to create image record", map[string]interface{}{
				"error": err.Error(),
			})
			continue
		}

		images = append(images, image)
	}

	return images, nil
}

// ProcessImage xử lý resize ảnh và upload variants (được gọi từ Worker)
func (s *bookImageService) ProcessImage(ctx context.Context, imageID string) error {
	// Lấy thông tin ảnh từ DB
	image, err := s.repo.GetByID(ctx, imageID)
	if err != nil {
		return fmt.Errorf("failed to get image: %w", err)
	}
	// ✅ LOG 1: Image info từ DB
	logger.Info("Image record from DB", map[string]interface{}{
		"image_id":     imageID,
		"book_id":      image.BookID,
		"original_url": image.OriginalURL,
		"sort_order":   image.SortOrder,
	})

	// Download ảnh original từ MinIO
	key := s.extractKeyFromURL(image.OriginalURL)
	originalData, err := s.storage.Download(ctx, key)
	if err != nil {
		// Update status failed
		s.repo.UpdateStatus(ctx, imageID, model.ImageStatusFailed, err.Error())
		return fmt.Errorf("failed to download original: %w", err)
	}
	// ✅ LOG 2: Key extracted
	logger.Info("Extracted key for download", map[string]interface{}{
		"image_id":      imageID,
		"original_url":  image.OriginalURL,
		"extracted_key": key,
	})

	// Validate ảnh
	err = s.imageProcessor.ValidateImage(originalData)
	if err != nil {
		s.repo.UpdateStatus(ctx, imageID, model.ImageStatusFailed, err.Error())
		return fmt.Errorf("invalid image: %w", err)
	}

	// Resize ảnh thành các variants
	variants, err := s.imageProcessor.ProcessImage(originalData)
	if err != nil {
		s.repo.UpdateStatus(ctx, imageID, model.ImageStatusFailed, err.Error())
		return fmt.Errorf("failed to process image: %w", err)
	}

	// Upload từng variant lên MinIO
	variantURLs := make(map[string]string)
	baseKey := fmt.Sprintf("books/%s/%d", image.BookID, image.SortOrder)

	for variantName, variantData := range variants {
		key := fmt.Sprintf("%s_%s.jpg", baseKey, variantName)
		url, err := s.storage.Upload(ctx, key, variantData, "image/jpeg")
		if err != nil {
			logger.Info("Failed to upload variant", map[string]interface{}{
				"variant": variantName,
				"error":   err.Error(),
			})
			continue
		}
		variantURLs[variantName] = url
	}

	// Update DB với variant URLs
	err = s.repo.UpdateVariants(
		ctx,
		imageID,
		variantURLs["large"],
		variantURLs["medium"],
		variantURLs["thumbnail"],
	)

	if err != nil {
		return fmt.Errorf("failed to update variants: %w", err)
	}

	logger.Info("Image processed successfully", map[string]interface{}{
		"image_id": imageID,
		"book_id":  image.BookID,
	})

	return nil
}

// GetBookImages lấy tất cả ảnh của một book
func (s *bookImageService) GetBookImages(ctx context.Context, bookID string) ([]*model.BookImage, error) {
	return s.repo.GetByBookID(ctx, bookID)
}

// DeleteBookImages xóa tất cả ảnh của book (cả DB và MinIO)
func (s *bookImageService) DeleteBookImages(ctx context.Context, bookID string) error {
	// Xóa files trên MinIO
	prefix := fmt.Sprintf("books/%s/", bookID)
	err := s.storage.DeleteByPrefix(ctx, prefix)
	if err != nil {
		logger.Info("Failed to delete images from MinIO", map[string]interface{}{
			"book_id": bookID,
			"error":   err.Error(),
		})
	}

	// Xóa records trong DB
	return s.repo.DeleteByBookID(ctx, bookID)
}

// Helper functions

func (s *bookImageService) downloadAndValidateImage(url string) ([]byte, string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read: %w", err)
	}

	// Validate
	err = s.imageProcessor.ValidateImage(data)
	if err != nil {
		return nil, "", err
	}

	// Detect format từ content-type
	format := "jpg"
	if contentType := resp.Header.Get("Content-Type"); contentType == "image/png" {
		format = "png"
	}

	return data, format, nil
}

func (s *bookImageService) extractKeyFromURL(fullURL string) string {
	// Parse URL
	u, err := url.Parse(fullURL)
	if err != nil {
		logger.Info("Failed to parse URL", map[string]interface{}{
			"url":   fullURL,
			"error": err.Error(),
		})
		return ""
	}

	// Path: /bookstore/books/uuid/1_original.jpeg
	path := strings.TrimPrefix(u.Path, "/")

	// ✅ Split và bỏ bucket name
	// path = "bookstore/books/uuid/1_original.jpeg"
	// parts = ["bookstore", "books/uuid/1_original.jpeg"]
	parts := strings.SplitN(path, "/", 2)

	if len(parts) < 2 {
		logger.Info("Invalid URL path", map[string]interface{}{
			"url":  fullURL,
			"path": path,
		})
		return path
	}

	key := parts[1] // "books/uuid/1_original.jpeg" ✅

	logger.Info("Extracted key from URL", map[string]interface{}{
		"url": fullURL,
		"key": key,
	})

	return key
}

func intPtr(i int64) *int64 {
	return &i
}
