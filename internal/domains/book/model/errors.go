package model

import (
	"bookstore-backend/internal/shared/response"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

var (
	ErrInvalidPageLimit         = errors.New("page and limit must be positive")
	ErrInvalidPriceRange        = errors.New("price_min must be <= price_max")
	ErrInvalidSort              = errors.New("invalid sort parameter")
	ErrBookNotFound             = errors.New("book not found")
	ErrDatabaseQuery            = errors.New("database query error")
	ErrCacheWrite               = errors.New("cache write error")
	ErrInvalidISBN              = errors.New("invalid ISBN format")
	ErrInvalidPublishedYear     = errors.New("invalid published year")
	ErrAuthorNotFound           = errors.New("author not found")
	ErrCategoryNotFound         = errors.New("category not found")
	ErrSlugAlreadyExists        = errors.New("slug already exists")
	ErrVersionConflict          = errors.New("version conflict: book was modified by another user")
	ErrISBNAlreadyExists        = errors.New("ISBN already exists")
	ErrPublisherNotFound        = errors.New("publisher not found")
	ErrInvalidImageCount        = errors.New("book must have 3-7 images")
	ErrImageValidationFail      = errors.New("one or more images are invalid")
	ErrImageTooLarge            = errors.New("image exceeds maximum size (5MB)")
	ErrInvalidImageFormat       = errors.New("image must be JPEG or PNG format")
	ErrBookHasActiveOrders      = errors.New("book has active orders and cannot be deleted")
	ErrBookHasReservedInventory = errors.New("book has reserved inventory and cannot be deleted")
)
var bookErrorMap = map[error]struct {
	Status  int
	Title   string
	Message string
}{
	ErrISBNAlreadyExists: {
		Status:  http.StatusConflict,
		Title:   "ISBN already exists",
		Message: "This ISBN is already registered in the system",
	},
	ErrAuthorNotFound: {
		Status:  http.StatusBadRequest,
		Title:   "Author not found",
		Message: "The specified author does not exist",
	},
	ErrCategoryNotFound: {
		Status:  http.StatusBadRequest,
		Title:   "Category not found",
		Message: "The specified category does not exist",
	},
	ErrPublisherNotFound: {
		Status:  http.StatusBadRequest,
		Title:   "Publisher not found",
		Message: "The specified publisher does not exist",
	},
	ErrSlugAlreadyExists: {
		Status:  http.StatusConflict,
		Title:   "Book title already exists",
		Message: "A book with similar title already exists",
	},
	ErrBookNotFound: {
		Status:  http.StatusNotFound,
		Title:   "Book not found",
		Message: "The specified book does not exist",
	},
	ErrVersionConflict:   {Status: http.StatusConflict, Title: "Version conflict", Message: "The book has been modified by another user. Please refresh and try again"},
	ErrISBNAlreadyExists: {Status: http.StatusConflict, Title: "ISBN already exists", Message: "This ISBN is already used by another book"},
	ErrAuthorNotFound:    {Status: http.StatusBadRequest, Title: "Author not found", Message: "The specified author does not exist"},
	ErrCategoryNotFound:  {Status: http.StatusBadRequest, Title: "Category not found", Message: "The specified category does not exist"},
	ErrPublisherNotFound: {Status: http.StatusBadRequest, Title: "Publisher not found", Message: "The specified publisher does not exist"},
}

func HandleBookError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}

	if config, ok := bookErrorMap[err]; ok {
		response.Error(c, config.Status, config.Title, config.Message)
		return true
	}

	// Lỗi không xác định
	log.Printf("[Handler] Error updating book: %v", err)
	response.Error(c, http.StatusInternalServerError, "Failed to update book", "Internal server error")
	return true
}

// ImageValidationError chứa chi tiết lỗi từng ảnh
type ImageValidationError struct {
	Index   int    `json:"index"`
	URL     string `json:"url"`
	Message string `json:"message"`
}

type ImageValidationErrors struct {
	Message string                 `json:"message"`
	Errors  []ImageValidationError `json:"errors"`
}

func (e *ImageValidationErrors) Error() string {
	return e.Message
}
