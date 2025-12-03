package service

import (
	"bookstore-backend/internal/domains/book/model"
	"context"

	"github.com/xuri/excelize/v2"
)

// ServiceInterface - Định nghĩa business logic methods
type ServiceInterface interface {
	ListBooks(ctx context.Context, req model.ListBooksRequest) ([]model.ListBooksResponse, *model.PaginationMeta, error)
	GetBookDetail(ctx context.Context, id string) (*model.BookDetailResponse, error)
	CreateBook(ctx context.Context, req model.CreateBookRequest) error
	UpdateBook(ctx context.Context, id string, req model.UpdateBookRequest) (*model.BookDetailResponse, error)
	DeleteBook(ctx context.Context, id string) (*model.DeleteBookResponse, error)
	ExportBooksToExcel(ctx context.Context, req model.ListBooksRequest) (*excelize.File, *[]model.ListBooksResponse, error)
	SearchBooks(ctx context.Context, req model.SearchBooksRequest) ([]model.BookSearchResponse, error)
	GetBooksByIDs(ctx context.Context, ids []string) ([]model.BookDetailResponse, error)
	GetBooksCheckout(ctx context.Context, ids []string) ([]model.BookCheckoutResponse, error)
}
