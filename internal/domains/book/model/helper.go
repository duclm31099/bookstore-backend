package model

import (
	"bookstore-backend/internal/shared/utils"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// Helper: Generate cache key from request
func GenerateCacheKey(prefix string, req ListBooksRequest) string {
	parts := []string{
		prefix,
		req.Search,
		req.CategoryID,
		fmt.Sprintf("%.0f", req.PriceMin),
		fmt.Sprintf("%.0f", req.PriceMax),
		req.Language,
		req.Sort,
		strconv.Itoa(req.Page),
		strconv.Itoa(req.Limit),
	}
	// Hash this to create a short cache key
	keyStr := strings.Join(parts, ":")
	return fmt.Sprintf("%s:%x", prefix, hashString(keyStr)) // Use CRC32 or MD5 hash
}

// Helper: Convert Book entity to DTO
func BookToListDTO(book Book) ListBooksResponse {
	return ListBooksResponse{
		ID:              book.ID,
		Title:           book.Title,
		Slug:            book.Slug,
		AuthorName:      book.AuthorName,
		PublisherName:   book.PublisherName,
		Price:           book.Price,
		CompareAtPrice:  book.CompareAtPrice,
		CoverURL:        book.CoverURL,
		Language:        book.Language,
		Format:          book.Format,
		RatingAverage:   book.RatingAverage,
		RatingCount:     book.RatingCount,
		ViewCount:       book.ViewCount,
		SoldCount:       book.SoldCount,
		IsFeatured:      book.IsFeatured,
		TotalStock:      book.TotalStock,
		CreatedAt:       book.CreatedAt,
		Images:          book.Images,
		MetaTitle:       book.MetaTitle,
		MetaDescription: book.MetaDescription,
		MetaKeywords:    book.MetaKeywords,
	}
}
func ToBookResponse(books []Book) []BookResponse {

	var result []BookResponse
	for _, b := range books {
		var temp BookResponse
		temp = BookResponse{
			ID:             b.ID,
			Title:          b.Title,
			Author:         &AuthorResponse{ID: b.AuthorID, Name: b.AuthorName},
			Publisher:      &PublisherResponse{ID: b.PublisherID, Name: b.PublisherName},
			Slug:           b.Slug,
			Description:    b.Description,
			Price:          b.Price,
			CompareAtPrice: b.CompareAtPrice,
			CoverURL:       b.CoverURL,
			Language:       b.Language,
			Format:         b.Format,
			AverageRating:  &b.RatingAverage,
			ReviewCount:    &b.RatingCount,
			ViewCount:      b.ViewCount,
			SoldCount:      b.SoldCount,
			IsFeatured:     b.IsFeatured,
			TotalStock:     &b.TotalStock,
			Images:         b.Images,
		}
		result = append(result, temp)
	}
	return result
}

// Helper: Hash string to integer
func hashString(s string) uint32 {
	h := uint32(5381)
	for i := 0; i < len(s); i++ {
		h = ((h << 5) + h) + uint32(s[i])
	}
	return h
}
func ToBookDetailResponse(b BookDetailRes, inventories []InventoryDetailDTO, reviews []ReviewDTO) *BookDetailResponse {
	return &BookDetailResponse{
		ID:              b.ID,
		Title:           b.Title,
		Author:          &AuthorDTO{ID: b.AuthorID, Name: *b.AuthorName},
		Category:        &CategoryDTO{ID: b.CategoryID, Name: *b.CategoryName},
		Publisher:       &PublisherDTO{ID: b.PublisherID, Name: *b.PublisherName},
		Description:     b.Description,
		Price:           b.Price,
		Language:        b.Language,
		Format:          b.Format,
		CoverURL:        b.CoverURL,
		PublishedYear:   b.PublishedYear,
		ViewCount:       b.ViewCount,
		SoldCount:       b.SoldCount,
		TotalStock:      b.TotalStock,
		Inventories:     inventories,
		Reviews:         reviews,
		Dimensions:      b.Dimensions,
		WeightGrams:     b.WeightGrams,
		EbookFileURL:    b.EbookFileURL,
		EbookFileSizeMB: b.EbookFileSizeMB,
		ISBN:            b.ISBN,
		EbookFormat:     b.EbookFormat,
		IsActive:        b.IsActive,
		MetaTitle:       b.MetaTitle,
		MetaDescription: b.MetaDescription,
		MetaKeywords:    b.MetaKeywords,
		Images:          b.Images,
	}
}

func ToBookEntity(req CreateBookRequest, finalSlug string) *Book {
	return &Book{
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
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

func BookEntityToDetailResponse(b Book) *BookDetailResponse {
	return &BookDetailResponse{
		ID:              b.ID,
		Title:           b.Title,
		Author:          &AuthorDTO{ID: b.AuthorID, Name: b.AuthorName},
		Category:        &CategoryDTO{ID: b.CategoryID, Name: b.CategoryName},
		Publisher:       &PublisherDTO{ID: b.PublisherID, Name: b.PublisherName},
		Description:     b.Description,
		Price:           b.Price,
		Language:        b.Language,
		Format:          b.Format,
		CoverURL:        b.CoverURL,
		PublishedYear:   b.PublishedYear,
		ViewCount:       b.ViewCount,
		SoldCount:       b.SoldCount,
		TotalStock:      b.TotalStock,
		Inventories:     []InventoryDetailDTO{}, // Batch fetch doesn't include detailed inventory
		Reviews:         []ReviewDTO{},          // Batch fetch doesn't include reviews
		Dimensions:      b.Dimensions,
		WeightGrams:     b.WeightGrams,
		EbookFileURL:    b.EbookFileURL,
		EbookFileSizeMB: b.EbookFileSizeMB,
		ISBN:            b.ISBN,
		EbookFormat:     b.EbookFormat,
		IsActive:        b.IsActive,
		MetaTitle:       b.MetaTitle,
		MetaDescription: b.MetaDescription,
		MetaKeywords:    b.MetaKeywords,
		Images:          b.Images,
	}
}
