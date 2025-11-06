package book

import (
	"fmt"
	"strconv"
	"strings"
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
		ID:    book.ID,
		Title: book.Title,
		Slug:  book.Slug,
		// AuthorName:     book.AuthorName,
		// PublisherName:  book.PublisherName,
		Price:          book.Price,
		CompareAtPrice: book.CompareAtPrice,
		CoverURL:       book.CoverURL,
		Language:       book.Language,
		Format:         book.Format,
		RatingAverage:  book.RatingAverage,
		RatingCount:    book.RatingCount,
		ViewCount:      book.ViewCount,
		SoldCount:      book.SoldCount,
		IsFeatured:     book.IsFeatured,
		// TotalStock:     book.TotalStock,
		CreatedAt: book.CreatedAt,
	}
}

// Helper: Hash string to integer
func hashString(s string) uint32 {
	h := uint32(5381)
	for i := 0; i < len(s); i++ {
		h = ((h << 5) + h) + uint32(s[i])
	}
	return h
}
