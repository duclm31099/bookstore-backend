<img src="https://r2cdn.perplexity.ai/pplx-full-logo-primary-dark%402x.png" style="height:64px;margin-right:32px"/>

# TODO LIST CHI TIẾT CHO BACKEND DEVELOPER - SPRINT 13-14: WISHLIST \& BANNERS

Dựa trên URD, dưới đây là danh sách công việc chi tiết và đầy đủ cho backend developer trong Sprint 13-14 (Phase 2, 2 tuần - 10 ngày làm việc).[^1]

## 1. Wishlist Table (P2-T018)

### Mô tả

Tạo database schema cho hệ thống wishlist/favorite books của users.[^1]

### Database Schema

#### 1.1 Wishlists Table Migration

Tạo file `migrations/000023_create_wishlists_table.up.sql`:[^1]

```sql
CREATE TABLE wishlists (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    book_id UUID NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    PRIMARY KEY (user_id, book_id)
);

-- Indexes
CREATE INDEX idx_wishlists_user ON wishlists(user_id, created_at DESC);
CREATE INDEX idx_wishlists_book ON wishlists(book_id);

-- Composite index for checking existence
CREATE INDEX idx_wishlists_user_book ON wishlists(user_id, book_id);
```


#### 1.2 Add Wishlist Count to Books

```sql
-- Add wishlist_count column to books table for analytics
ALTER TABLE books ADD COLUMN IF NOT EXISTS wishlist_count INT DEFAULT 0;

-- Trigger to auto-update wishlist count
CREATE OR REPLACE FUNCTION update_book_wishlist_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE books SET wishlist_count = wishlist_count + 1 WHERE id = NEW.book_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE books SET wishlist_count = GREATEST(wishlist_count - 1, 0) WHERE id = OLD.book_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_book_wishlist_count
AFTER INSERT OR DELETE ON wishlists
FOR EACH ROW
EXECUTE FUNCTION update_book_wishlist_count();
```


### Công việc cụ thể

#### 1.3 Domain Model

Tạo file `internal/domains/wishlist/model/wishlist.go`:[^1]

```go
package model

import "time"

type Wishlist struct {
    UserID    string    `json:"user_id"`
    BookID    string    `json:"book_id"`
    CreatedAt time.Time `json:"created_at"`
    
    // Joined fields (when fetching with book details)
    Book      *Book     `json:"book,omitempty"`
}

type Book struct {
    ID             string  `json:"id"`
    Title          string  `json:"title"`
    Slug           string  `json:"slug"`
    AuthorName     string  `json:"author_name"`
    CoverURL       string  `json:"cover_url"`
    Price          float64 `json:"price"`
    CompareAtPrice *float64 `json:"compare_at_price,omitempty"`
    AverageRating  float64 `json:"average_rating"`
    IsActive       bool    `json:"is_active"`
}
```


### Acceptance Criteria

- Migration tạo bảng wishlists thành công[^1]
- Composite primary key (user_id, book_id) đảm bảo không duplicate[^1]
- Trigger auto-update wishlist_count hoạt động[^1]
- Indexes optimize queries[^1]


### Dependencies

- P1-T002: Database setup[^1]
- P1-T003: Core tables (books, users)[^1]


### Effort

0.5 ngày[^1]

***

## 2. Add/Remove Wishlist APIs (P2-T019)

### Mô tả

APIs cho user thêm và xóa sách khỏi wishlist.[^1]

### API Endpoints

- `POST /v1/user/wishlist` - Add book to wishlist[^1]
- `DELETE /v1/user/wishlist/:book_id` - Remove book from wishlist[^1]


### Request/Response

#### Add to Wishlist

**Request Body**:

```json
{
  "book_id": "uuid"
}
```

**Response**:

```json
{
  "success": true,
  "message": "Book added to wishlist"
}
```


#### Remove from Wishlist

**Response**:

```json
{
  "success": true,
  "message": "Book removed from wishlist"
}
```


### Công việc cụ thể

#### 2.1 Wishlist Service

Tạo file `internal/domains/wishlist/service/wishlist_service.go`:[^1]

```go
package service

import (
    "context"
    "fmt"
)

type WishlistService struct {
    wishlistRepo *repository.WishlistRepository
    bookRepo     *repository.BookRepository
}

func NewWishlistService(wr *repository.WishlistRepository, br *repository.BookRepository) *WishlistService {
    return &WishlistService{
        wishlistRepo: wr,
        bookRepo:     br,
    }
}

func (s *WishlistService) AddToWishlist(ctx context.Context, userID string, bookID string) error {
    // 1. Validate book exists and is active
    book, err := s.bookRepo.FindByID(ctx, bookID)
    if err != nil {
        return fmt.Errorf("book not found")
    }
    
    if !book.IsActive {
        return fmt.Errorf("book is not available")
    }
    
    // 2. Check if already in wishlist
    exists, err := s.wishlistRepo.Exists(ctx, userID, bookID)
    if err != nil {
        return err
    }
    
    if exists {
        return fmt.Errorf("book already in wishlist")
    }
    
    // 3. Add to wishlist
    wishlist := &Wishlist{
        UserID: userID,
        BookID: bookID,
    }
    
    err = s.wishlistRepo.Create(ctx, wishlist)
    if err != nil {
        return err
    }
    
    return nil
}

func (s *WishlistService) RemoveFromWishlist(ctx context.Context, userID string, bookID string) error {
    // Check if exists
    exists, err := s.wishlistRepo.Exists(ctx, userID, bookID)
    if err != nil {
        return err
    }
    
    if !exists {
        return fmt.Errorf("book not in wishlist")
    }
    
    // Delete
    err = s.wishlistRepo.Delete(ctx, userID, bookID)
    if err != nil {
        return err
    }
    
    return nil
}
```


#### 2.2 Wishlist Repository

Tạo file `internal/domains/wishlist/repository/wishlist_repository.go`:[^1]

```go
package repository

import (
    "context"
    "database/sql"
)

type WishlistRepository struct {
    db *sql.DB
}

func NewWishlistRepository(db *sql.DB) *WishlistRepository {
    return &WishlistRepository{db: db}
}

func (r *WishlistRepository) Exists(ctx context.Context, userID string, bookID string) (bool, error) {
    var exists bool
    query := `SELECT EXISTS(SELECT 1 FROM wishlists WHERE user_id = $1 AND book_id = $2)`
    err := r.db.QueryRowContext(ctx, query, userID, bookID).Scan(&exists)
    return exists, err
}

func (r *WishlistRepository) Create(ctx context.Context, wishlist *Wishlist) error {
    query := `
        INSERT INTO wishlists (user_id, book_id)
        VALUES ($1, $2)
        ON CONFLICT (user_id, book_id) DO NOTHING
    `
    _, err := r.db.ExecContext(ctx, query, wishlist.UserID, wishlist.BookID)
    return err
}

func (r *WishlistRepository) Delete(ctx context.Context, userID string, bookID string) error {
    query := `DELETE FROM wishlists WHERE user_id = $1 AND book_id = $2`
    _, err := r.db.ExecContext(ctx, query, userID, bookID)
    return err
}
```


#### 2.3 Handler Implementation

Tạo file `internal/domains/wishlist/handler/wishlist_handler.go`:[^1]

```go
package handler

import (
    "github.com/gin-gonic/gin"
    "bookstore/internal/domains/wishlist/service"
    "bookstore/pkg/errors"
)

type WishlistHandler struct {
    wishlistService *service.WishlistService
}

func NewWishlistHandler(ws *service.WishlistService) *WishlistHandler {
    return &WishlistHandler{wishlistService: ws}
}

func (h *WishlistHandler) AddToWishlist(c *gin.Context) {
    userID := c.GetString("user_id") // From JWT middleware
    
    var req struct {
        BookID string `json:"book_id" binding:"required"`
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"success": false, "error": "Invalid request"})
        return
    }
    
    err := h.wishlistService.AddToWishlist(c.Request.Context(), userID, req.BookID)
    if err != nil {
        c.JSON(400, gin.H{"success": false, "error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{"success": true, "message": "Book added to wishlist"})
}

func (h *WishlistHandler) RemoveFromWishlist(c *gin.Context) {
    userID := c.GetString("user_id")
    bookID := c.Param("book_id")
    
    err := h.wishlistService.RemoveFromWishlist(c.Request.Context(), userID, bookID)
    if err != nil {
        c.JSON(400, gin.H{"success": false, "error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{"success": true, "message": "Book removed from wishlist"})
}
```


#### 2.4 Routes Registration

```go
// In cmd/api/main.go or routes setup
authRoutes := r.Group("/v1/user")
authRoutes.Use(authMiddleware.JWT())
{
    authRoutes.POST("/wishlist", wishlistHandler.AddToWishlist)
    authRoutes.DELETE("/wishlist/:book_id", wishlistHandler.RemoveFromWishlist)
    authRoutes.GET("/wishlist", wishlistHandler.GetWishlist) // Next task
}
```


### Acceptance Criteria

- User thêm được sách vào wishlist[^1]
- Không duplicate (composite primary key)[^1]
- Validate book tồn tại và active[^1]
- User xóa được sách khỏi wishlist[^1]
- Book wishlist_count tự động update (trigger)[^1]


### Dependencies

- P2-T018: Wishlist table[^1]
- P1-T012: JWT middleware[^1]


### Effort

1 ngày[^1]

***

## 3. Get Wishlist API (P2-T020)

### Mô tả

API để user xem danh sách wishlist với thông tin chi tiết của books.[^1]

### API Endpoint

`GET /v1/user/wishlist`[^1]

### Query Parameters

- `?page=1&limit=20` - Pagination[^1]
- `?sort=created_at:desc` - Sort by added date[^1]


### Response Format

```json
{
  "success": true,
  "data": {
    "items": [
      {
        "book_id": "uuid",
        "title": "Book Title",
        "slug": "book-title",
        "author_name": "Author Name",
        "cover_url": "https://...",
        "price": 150000,
        "compare_at_price": 200000,
        "average_rating": 4.5,
        "is_active": true,
        "added_at": "2025-10-31T10:00:00Z"
      }
    ]
  },
  "meta": {
    "page": 1,
    "limit": 20,
    "total": 15
  }
}
```


### Công việc cụ thể

#### 3.1 Service Implementation

```go
func (s *WishlistService) GetWishlist(ctx context.Context, userID string, page int, limit int) (*WishlistResult, error) {
    offset := (page - 1) * limit
    
    // Get wishlist items with book details
    items, err := s.wishlistRepo.FindByUserID(ctx, userID, limit, offset)
    if err != nil {
        return nil, err
    }
    
    // Count total
    total, err := s.wishlistRepo.CountByUserID(ctx, userID)
    if err != nil {
        return nil, err
    }
    
    return &WishlistResult{
        Items: items,
        Total: total,
    }, nil
}
```


#### 3.2 Repository - Find with Book Details

```go
func (r *WishlistRepository) FindByUserID(ctx context.Context, userID string, limit int, offset int) ([]WishlistItem, error) {
    query := `
        SELECT 
            w.book_id,
            w.created_at as added_at,
            b.title,
            b.slug,
            b.cover_url,
            b.price,
            b.compare_at_price,
            b.average_rating,
            b.is_active,
            a.name as author_name
        FROM wishlists w
        JOIN books b ON w.book_id = b.id
        JOIN authors a ON b.author_id = a.id
        WHERE w.user_id = $1
        ORDER BY w.created_at DESC
        LIMIT $2 OFFSET $3
    `
    
    rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    items := []WishlistItem{}
    for rows.Next() {
        var item WishlistItem
        err := rows.Scan(
            &item.BookID,
            &item.AddedAt,
            &item.Title,
            &item.Slug,
            &item.CoverURL,
            &item.Price,
            &item.CompareAtPrice,
            &item.AverageRating,
            &item.IsActive,
            &item.AuthorName,
        )
        if err != nil {
            return nil, err
        }
        items = append(items, item)
    }
    
    return items, nil
}

func (r *WishlistRepository) CountByUserID(ctx context.Context, userID string) (int, error) {
    var count int
    query := `SELECT COUNT(*) FROM wishlists WHERE user_id = $1`
    err := r.db.QueryRowContext(ctx, query, userID).Scan(&count)
    return count, err
}
```


#### 3.3 Handler Implementation

```go
func (h *WishlistHandler) GetWishlist(c *gin.Context) {
    userID := c.GetString("user_id")
    
    // Parse pagination
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
    
    result, err := h.wishlistService.GetWishlist(c.Request.Context(), userID, page, limit)
    if err != nil {
        c.JSON(500, gin.H{"success": false, "error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{
        "success": true,
        "data": gin.H{
            "items": result.Items,
        },
        "meta": gin.H{
            "page":  page,
            "limit": limit,
            "total": result.Total,
        },
    })
}
```


#### 3.4 Include Wishlist Status in Book Listing

Update book listing API để hiển thị user đã wishlist book chưa:

```go
// In BookService
func (s *BookService) ListBooks(ctx context.Context, filters BookFilters, currentUserID *string) (*BookListResult, error) {
    // ... existing query
    
    books := []Book{}
    // ... fetch books
    
    // If user is logged in, mark wishlisted books
    if currentUserID != nil {
        wishlistedIDs := s.wishlistRepo.GetUserWishlistedBookIDs(ctx, *currentUserID)
        wishlistedMap := make(map[string]bool)
        for _, id := range wishlistedIDs {
            wishlistedMap[id] = true
        }
        
        for i := range books {
            books[i].IsWishlisted = wishlistedMap[books[i].ID]
        }
    }
    
    return &BookListResult{Books: books, Total: total}, nil
}
```


### Acceptance Criteria

- User xem được danh sách wishlist với full book info[^1]
- Pagination hoạt động[^1]
- Sort by added date (newest first)[^1]
- Performance: P95 < 100ms với 100 items[^1]
- Book listing hiển thị trạng thái wishlisted[^1]


### Dependencies

- P2-T018: Wishlist table[^1]
- P2-T019: Add/Remove APIs[^1]


### Effort

0.5 ngày[^1]

***

## 4. Banners Table (P2-T021)

### Mô tả

Tạo database schema cho hệ thống banners/carousel trên homepage.[^1]

### Database Schema

#### 4.1 Banners Table Migration

Tạo file `migrations/000024_create_banners_table.up.sql`:[^1]

```sql
CREATE TABLE banners (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    
    -- Images
    image_url TEXT NOT NULL,
    mobile_image_url TEXT, -- Optional mobile-specific image
    
    -- Action
    link_url TEXT, -- Redirect URL when clicked
    
    -- Placement
    position TEXT NOT NULL CHECK (position IN ('hero', 'sidebar', 'footer')),
    sort_order INT DEFAULT 0,
    
    -- Scheduling
    start_at TIMESTAMPTZ NOT NULL,
    end_at TIMESTAMPTZ NOT NULL,
    
    -- Status
    is_active BOOLEAN DEFAULT true,
    
    -- Analytics
    click_count INT DEFAULT 0,
    view_count INT DEFAULT 0,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT valid_dates CHECK (end_at > start_at)
);

-- Indexes
CREATE INDEX idx_banners_position ON banners(position, sort_order) 
    WHERE is_active = true AND NOW() BETWEEN start_at AND end_at;

CREATE INDEX idx_banners_active ON banners(is_active, start_at, end_at);

CREATE INDEX idx_banners_dates ON banners(start_at, end_at);

-- Trigger auto update updated_at
CREATE TRIGGER update_banners_updated_at
    BEFORE UPDATE ON banners
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```


### Công việc cụ thể

#### 4.2 Domain Model

Tạo file `internal/domains/banner/model/banner.go`:[^1]

```go
package model

import "time"

type BannerPosition string

const (
    BannerPositionHero    BannerPosition = "hero"
    BannerPositionSidebar BannerPosition = "sidebar"
    BannerPositionFooter  BannerPosition = "footer"
)

type Banner struct {
    ID              string         `json:"id"`
    Title           string         `json:"title"`
    ImageURL        string         `json:"image_url"`
    MobileImageURL  *string        `json:"mobile_image_url,omitempty"`
    LinkURL         *string        `json:"link_url,omitempty"`
    Position        BannerPosition `json:"position"`
    SortOrder       int            `json:"sort_order"`
    StartAt         time.Time      `json:"start_at"`
    EndAt           time.Time      `json:"end_at"`
    IsActive        bool           `json:"is_active"`
    ClickCount      int            `json:"click_count"`
    ViewCount       int            `json:"view_count"`
    CreatedAt       time.Time      `json:"created_at"`
    UpdatedAt       time.Time      `json:"updated_at"`
}
```


#### 4.3 Seed Data

Tạo file `seeds/003_banners_seed.sql`:[^1]

```sql
-- Hero banners (main carousel)
INSERT INTO banners (title, image_url, mobile_image_url, link_url, position, sort_order, start_at, end_at, is_active)
VALUES
('Flash Sale Sách Văn Học', 
 'https://cdn.bookstore.com/banners/flash-sale-hero.jpg',
 'https://cdn.bookstore.com/banners/flash-sale-hero-mobile.jpg',
 '/promotions/flash-sale',
 'hero', 1,
 '2025-11-01 00:00:00+07', '2025-11-30 23:59:59+07', true),

('Sách Mới Tháng 11', 
 'https://cdn.bookstore.com/banners/new-arrivals-hero.jpg',
 'https://cdn.bookstore.com/banners/new-arrivals-hero-mobile.jpg',
 '/books?filter=new_arrivals',
 'hero', 2,
 '2025-11-01 00:00:00+07', '2025-11-30 23:59:59+07', true),

('Giảm 50% Sách Thiếu Nhi', 
 'https://cdn.bookstore.com/banners/kids-sale-hero.jpg',
 'https://cdn.bookstore.com/banners/kids-sale-hero-mobile.jpg',
 '/categories/thieu-nhi',
 'hero', 3,
 '2025-11-01 00:00:00+07', '2025-12-31 23:59:59+07', true);

-- Sidebar banners
INSERT INTO banners (title, image_url, link_url, position, sort_order, start_at, end_at, is_active)
VALUES
('Ebook Hot Deal', 
 'https://cdn.bookstore.com/banners/ebook-sidebar.jpg',
 '/ebooks/hot-deals',
 'sidebar', 1,
 '2025-11-01 00:00:00+07', '2025-12-31 23:59:59+07', true),

('Tác Giả Nổi Bật', 
 'https://cdn.bookstore.com/banners/featured-authors-sidebar.jpg',
 '/authors/featured',
 'sidebar', 2,
 '2025-11-01 00:00:00+07', '2025-12-31 23:59:59+07', true);
```


### Acceptance Criteria

- Migration tạo bảng banners thành công[^1]
- Constraint validate dates (end_at > start_at)[^1]
- Indexes optimize queries cho active banners[^1]
- Seed data insert được 5+ banners mẫu[^1]


### Dependencies

- P1-T002: Database setup[^1]


### Effort

0.5 ngày[^1]

***

## 5. Admin: CRUD Banners (P2-T022)

### Mô tả

Admin panel APIs để quản lý banners.[^1]

### API Endpoints

- `GET /v1/admin/banners` - List banners[^1]
- `GET /v1/admin/banners/:id` - Get banner detail[^1]
- `POST /v1/admin/banners` - Create banner[^1]
- `PUT /v1/admin/banners/:id` - Update banner[^1]
- `DELETE /v1/admin/banners/:id` - Delete banner[^1]


### Công việc cụ thể

#### 5.1 Create Banner API

**Request Body**:

```json
{
  "title": "Flash Sale Banner",
  "image_url": "https://cdn.bookstore.com/banners/flash-sale.jpg",
  "mobile_image_url": "https://cdn.bookstore.com/banners/flash-sale-mobile.jpg",
  "link_url": "/promotions/flash-sale",
  "position": "hero",
  "sort_order": 1,
  "start_at": "2025-11-01T00:00:00+07:00",
  "end_at": "2025-11-30T23:59:59+07:00",
  "is_active": true
}
```

**Service**:

```go
func (s *BannerService) CreateBanner(ctx context.Context, req CreateBannerRequest) (*Banner, error) {
    // 1. Validate dates
    if req.EndAt.Before(req.StartAt) {
        return nil, fmt.Errorf("end_at must be after start_at")
    }
    
    // 2. Validate position
    validPositions := []string{"hero", "sidebar", "footer"}
    if !contains(validPositions, req.Position) {
        return nil, fmt.Errorf("invalid position")
    }
    
    // 3. Create banner
    banner := &Banner{
        Title:          req.Title,
        ImageURL:       req.ImageURL,
        MobileImageURL: req.MobileImageURL,
        LinkURL:        req.LinkURL,
        Position:       BannerPosition(req.Position),
        SortOrder:      req.SortOrder,
        StartAt:        req.StartAt,
        EndAt:          req.EndAt,
        IsActive:       req.IsActive,
    }
    
    err := s.bannerRepo.Create(ctx, banner)
    if err != nil {
        return nil, err
    }
    
    return banner, nil
}
```


#### 5.2 Update Banner API

```go
func (s *BannerService) UpdateBanner(ctx context.Context, id string, req UpdateBannerRequest) (*Banner, error) {
    // 1. Get existing banner
    banner, err := s.bannerRepo.FindByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("banner not found")
    }
    
    // 2. Update fields
    if req.Title != nil {
        banner.Title = *req.Title
    }
    if req.ImageURL != nil {
        banner.ImageURL = *req.ImageURL
    }
    if req.MobileImageURL != nil {
        banner.MobileImageURL = req.MobileImageURL
    }
    if req.LinkURL != nil {
        banner.LinkURL = req.LinkURL
    }
    if req.Position != nil {
        banner.Position = BannerPosition(*req.Position)
    }
    if req.SortOrder != nil {
        banner.SortOrder = *req.SortOrder
    }
    if req.StartAt != nil {
        banner.StartAt = *req.StartAt
    }
    if req.EndAt != nil {
        banner.EndAt = *req.EndAt
    }
    if req.IsActive != nil {
        banner.IsActive = *req.IsActive
    }
    
    // 3. Validate dates
    if banner.EndAt.Before(banner.StartAt) {
        return nil, fmt.Errorf("end_at must be after start_at")
    }
    
    // 4. Update
    err = s.bannerRepo.Update(ctx, banner)
    if err != nil {
        return nil, err
    }
    
    return banner, nil
}
```


#### 5.3 List Banners API

**Query Parameters**:

- `?position=hero` - Filter by position[^1]
- `?is_active=true` - Filter by status[^1]
- `?status=active|upcoming|expired` - Filter by time status[^1]
- `?page=1&limit=20` - Pagination[^1]

```go
func (s *BannerService) ListBanners(ctx context.Context, filters BannerFilters) (*BannerListResult, error) {
    query := `
        SELECT * FROM banners
        WHERE 1=1
    `
    
    args := []interface{}{}
    argPos := 1
    
    // Apply filters
    if filters.Position != "" {
        query += fmt.Sprintf(" AND position = $%d", argPos)
        args = append(args, filters.Position)
        argPos++
    }
    
    if filters.IsActive != nil {
        query += fmt.Sprintf(" AND is_active = $%d", argPos)
        args = append(args, *filters.IsActive)
        argPos++
    }
    
    if filters.Status == "active" {
        query += " AND is_active = true AND NOW() BETWEEN start_at AND end_at"
    } else if filters.Status == "upcoming" {
        query += " AND is_active = true AND start_at > NOW()"
    } else if filters.Status == "expired" {
        query += " AND end_at < NOW()"
    }
    
    // Sort
    query += " ORDER BY position, sort_order, created_at DESC"
    
    // Count total
    countQuery := "SELECT COUNT(*) FROM (" + query + ") as count_query"
    var total int
    s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
    
    // Pagination
    query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argPos, argPos+1)
    args = append(args, filters.Limit, (filters.Page-1)*filters.Limit)
    
    // Execute
    rows, err := s.db.QueryContext(ctx, query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    banners := []Banner{}
    for rows.Next() {
        var b Banner
        // Scan...
        banners = append(banners, b)
    }
    
    return &BannerListResult{Banners: banners, Total: total}, nil
}
```


#### 5.4 Delete Banner API

```go
func (s *BannerService) DeleteBanner(ctx context.Context, id string) error {
    // Check if exists
    _, err := s.bannerRepo.FindByID(ctx, id)
    if err != nil {
        return fmt.Errorf("banner not found")
    }
    
    // Hard delete (banners có thể xóa thật)
    err = s.bannerRepo.Delete(ctx, id)
    if err != nil {
        return err
    }
    
    return nil
}
```


#### 5.5 Validation

```go
type CreateBannerRequest struct {
    Title          string    `json:"title"`
    ImageURL       string    `json:"image_url"`
    MobileImageURL *string   `json:"mobile_image_url"`
    LinkURL        *string   `json:"link_url"`
    Position       string    `json:"position"`
    SortOrder      int       `json:"sort_order"`
    StartAt        time.Time `json:"start_at"`
    EndAt          time.Time `json:"end_at"`
    IsActive       bool      `json:"is_active"`
}

func (r CreateBannerRequest) Validate() error {
    return validation.ValidateStruct(&r,
        validation.Field(&r.Title, 
            validation.Required,
            validation.Length(1, 200)),
        validation.Field(&r.ImageURL, 
            validation.Required,
            is.URL),
        validation.Field(&r.MobileImageURL,
            validation.When(r.MobileImageURL != nil, is.URL)),
        validation.Field(&r.LinkURL,
            validation.When(r.LinkURL != nil, validation.Length(1, 500))),
        validation.Field(&r.Position, 
            validation.Required,
            validation.In("hero", "sidebar", "footer")),
        validation.Field(&r.SortOrder,
            validation.Min(0)),
        validation.Field(&r.StartAt,
            validation.Required),
        validation.Field(&r.EndAt,
            validation.Required),
    )
}
```


### Acceptance Criteria

- Admin tạo được banner với validation đầy đủ[^1]
- Update banner (partial update)[^1]
- List banners với filter và pagination[^1]
- Delete banner[^1]
- Audit log ghi lại thay đổi[^1]


### Dependencies

- P2-T021: Banners table[^1]
- P1-T029: RBAC middleware[^1]


### Effort

2 ngày[^1]

***

## 6. Public: Get Active Banners (P2-T023)

### Mô tả

Public API để lấy banners đang active theo position.[^1]

### API Endpoint

`GET /v1/banners?position=hero`[^1]

### Query Parameters

- `position` (optional): hero, sidebar, footer[^1]
- Nếu không có position, trả về tất cả active banners[^1]


### Response Format

```json
{
  "success": true,
  "data": {
    "hero": [
      {
        "id": "uuid",
        "title": "Flash Sale",
        "image_url": "https://...",
        "mobile_image_url": "https://...",
        "link_url": "/promotions/flash-sale",
        "position": "hero",
        "sort_order": 1
      }
    ],
    "sidebar": [
      {
        "id": "uuid",
        "title": "Ebook Deals",
        "image_url": "https://...",
        "link_url": "/ebooks",
        "position": "sidebar",
        "sort_order": 1
      }
    ]
  }
}
```


### Công việc cụ thể

#### 6.1 Service Implementation

```go
func (s *BannerService) GetActiveBanners(ctx context.Context, position *string) (map[string][]Banner, error) {
    query := `
        SELECT 
            id, title, image_url, mobile_image_url, link_url, 
            position, sort_order
        FROM banners
        WHERE is_active = true
        AND NOW() BETWEEN start_at AND end_at
    `
    
    args := []interface{}{}
    
    if position != nil {
        query += " AND position = $1"
        args = append(args, *position)
    }
    
    query += " ORDER BY position, sort_order ASC"
    
    rows, err := s.db.QueryContext(ctx, query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    // Group by position
    bannersMap := make(map[string][]Banner)
    
    for rows.Next() {
        var b Banner
        err := rows.Scan(
            &b.ID, &b.Title, &b.ImageURL, &b.MobileImageURL, &b.LinkURL,
            &b.Position, &b.SortOrder,
        )
        if err != nil {
            return nil, err
        }
        
        pos := string(b.Position)
        bannersMap[pos] = append(bannersMap[pos], b)
    }
    
    return bannersMap, nil
}
```


#### 6.2 Handler Implementation

```go
func (h *BannerHandler) GetActiveBanners(c *gin.Context) {
    position := c.Query("position")
    
    var posPtr *string
    if position != "" {
        posPtr = &position
    }
    
    banners, err := h.bannerService.GetActiveBanners(c.Request.Context(), posPtr)
    if err != nil {
        c.JSON(500, gin.H{"success": false, "error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{
        "success": true,
        "data":    banners,
    })
}
```


#### 6.3 Caching with Redis

Banners ít thay đổi, nên cache để tăng performance:

```go
func (s *BannerService) GetActiveBannersWithCache(ctx context.Context, position *string) (map[string][]Banner, error) {
    // 1. Build cache key
    cacheKey := "banners:active"
    if position != nil {
        cacheKey = fmt.Sprintf("banners:active:%s", *position)
    }
    
    // 2. Try to get from Redis
    cached, err := s.redis.Get(ctx, cacheKey).Result()
    if err == nil {
        // Cache hit
        var banners map[string][]Banner
        json.Unmarshal([]byte(cached), &banners)
        return banners, nil
    }
    
    // 3. Cache miss - fetch from DB
    banners, err := s.GetActiveBanners(ctx, position)
    if err != nil {
        return nil, err
    }
    
    // 4. Store in cache (TTL: 5 minutes)
    bannersJSON, _ := json.Marshal(banners)
    s.redis.Set(ctx, cacheKey, bannersJSON, 5*time.Minute)
    
    return banners, nil
}
```


#### 6.4 Cache Invalidation

Khi admin update banner, invalidate cache:

```go
func (s *BannerService) UpdateBanner(ctx context.Context, id string, req UpdateBannerRequest) (*Banner, error) {
    // ... update logic
    
    // Invalidate cache
    s.redis.Del(ctx, "banners:active")
    s.redis.Del(ctx, "banners:active:hero")
    s.redis.Del(ctx, "banners:active:sidebar")
    s.redis.Del(ctx, "banners:active:footer")
    
    return banner, nil
}
```


### Acceptance Criteria

- Public API trả về active banners (trong time range)[^1]
- Group by position[^1]
- Sort by sort_order[^1]
- Redis caching với TTL 5 phút[^1]
- Performance: P95 < 50ms (with cache)[^1]


### Dependencies

- P2-T021: Banners table[^1]
- P1-T004: Redis setup[^1]


### Effort

0.5 ngày[^1]

***

## 7. Banner Click Tracking (P2-T024)

### Mô tả

Tracking clicks trên banner để analytics và đo effectiveness.[^1]

### API Endpoint

`POST /v1/banners/:id/click`[^1]

### Business Logic

- Increment `click_count` khi user click banner[^1]
- Optional: Track user IP, device, timestamp[^1]
- Async processing với Asynq để không block response[^1]


### Công việc cụ thể

#### 7.1 Click Tracking Table (Optional - for detailed analytics)

```sql
CREATE TABLE banner_clicks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    banner_id UUID NOT NULL REFERENCES banners(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    ip_address INET,
    user_agent TEXT,
    referrer TEXT,
    clicked_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_banner_clicks_banner ON banner_clicks(banner_id, clicked_at DESC);
CREATE INDEX idx_banner_clicks_user ON banner_clicks(user_id) WHERE user_id IS NOT NULL;
```


#### 7.2 Click Handler

```go
func (h *BannerHandler) TrackClick(c *gin.Context) {
    bannerID := c.Param("id")
    
    // Get optional user ID (if logged in)
    userID := c.GetString("user_id") // May be empty
    
    // Get metadata
    ipAddr := c.ClientIP()
    userAgent := c.Request.UserAgent()
    referrer := c.Request.Referer()
    
    // Enqueue background job
    task := asynq.NewTask("banner:track_click", &TrackBannerClickPayload{
        BannerID:  bannerID,
        UserID:    userID,
        IPAddr:    ipAddr,
        UserAgent: userAgent,
        Referrer:  referrer,
    })
    
    h.queueClient.Enqueue(task, queue.QueueDefault)
    
    // Return success immediately (don't wait for job)
    c.JSON(200, gin.H{"success": true})
}
```


#### 7.3 Background Job Handler

```go
func (h *BannerJobHandler) TrackClick(ctx context.Context, task *asynq.Task) error {
    var payload TrackBannerClickPayload
    json.Unmarshal(task.Payload(), &payload)
    
    // 1. Increment click_count in banners table
    err := h.bannerRepo.IncrementClickCount(ctx, payload.BannerID)
    if err != nil {
        return err
    }
    
    // 2. (Optional) Insert detailed click record
    click := &BannerClick{
        BannerID:  payload.BannerID,
        UserID:    payload.UserID,
        IPAddr:    payload.IPAddr,
        UserAgent: payload.UserAgent,
        Referrer:  payload.Referrer,
    }
    
    err = h.bannerClickRepo.Create(ctx, click)
    if err != nil {
        log.Error("Failed to insert banner click", "error", err)
        // Don't return error - still counted in banners table
    }
    
    return nil
}
```


#### 7.4 Repository - Increment Click Count

```go
func (r *BannerRepository) IncrementClickCount(ctx context.Context, bannerID string) error {
    query := `UPDATE banners SET click_count = click_count + 1 WHERE id = $1`
    _, err := r.db.ExecContext(ctx, query, bannerID)
    return err
}
```


#### 7.5 View Tracking (Optional)

Track impressions (banners được hiển thị):

```go
// POST /v1/banners/:id/view
func (h *BannerHandler) TrackView(c *gin.Context) {
    bannerID := c.Param("id")
    
    // Enqueue background job
    task := asynq.NewTask("banner:track_view", &TrackBannerViewPayload{
        BannerID: bannerID,
    })
    
    h.queueClient.Enqueue(task, queue.QueueLow)
    
    c.JSON(200, gin.H{"success": true})
}

// Job handler
func (h *BannerJobHandler) TrackView(ctx context.Context, task *asynq.Task) error {
    var payload TrackBannerViewPayload
    json.Unmarshal(task.Payload(), &payload)
    
    // Increment view_count
    query := `UPDATE banners SET view_count = view_count + 1 WHERE id = $1`
    _, err := h.db.ExecContext(ctx, query, payload.BannerID)
    
    return err
}
```


#### 7.6 Analytics Dashboard (Admin)

`GET /v1/admin/banners/:id/analytics`

```go
func (s *BannerService) GetBannerAnalytics(ctx context.Context, bannerID string) (*BannerAnalytics, error) {
    banner, err := s.bannerRepo.FindByID(ctx, bannerID)
    if err != nil {
        return nil, err
    }
    
    // Calculate CTR (Click-Through Rate)
    ctr := 0.0
    if banner.ViewCount > 0 {
        ctr = (float64(banner.ClickCount) / float64(banner.ViewCount)) * 100
    }
    
    // Get click distribution by date
    clicksByDate := s.bannerClickRepo.GetClicksByDate(ctx, bannerID, 30)
    
    return &BannerAnalytics{
        BannerID:     banner.ID,
        Title:        banner.Title,
        ViewCount:    banner.ViewCount,
        ClickCount:   banner.ClickCount,
        CTR:          ctr,
        ClicksByDate: clicksByDate,
    }, nil
}
```


### Acceptance Criteria

- Track clicks không block user experience (async)[^1]
- Increment click_count chính xác[^1]
- Optional: Lưu detailed click records với IP, user agent[^1]
- Admin xem được analytics (views, clicks, CTR)[^1]


### Dependencies

- P2-T021: Banners table[^1]
- P2-T008: Asynq setup[^1]


### Effort

1 ngày[^1]

***

## 8. Full-Text Search (tsvector) (P2-T025)

### Mô tả

Upgrade search từ LIKE query sang full-text search với tsvector PostgreSQL.[^1]

### Current vs Target

- **Current** (P1-T018): `WHERE title LIKE '%keyword%'` - slow, không ranked[^1]
- **Target**: Full-text search với ranking, stemming, accent-insensitive[^1]


### Công việc cụ thể

#### 8.1 Database Setup (Already exists in books table)

```sql
-- books table already has search_vector column
-- CREATE INDEX idx_books_search ON books USING GIN(search_vector);

-- Trigger already exists to auto-update search_vector
-- CREATE TRIGGER books_search_update ...
```


#### 8.2 Vietnamese Text Search Configuration

PostgreSQL không có built-in Vietnamese, dùng unaccent extension:

```sql
-- Install unaccent extension
CREATE EXTENSION IF NOT EXISTS unaccent;

-- Create custom text search configuration for Vietnamese
CREATE TEXT SEARCH CONFIGURATION vietnamese (COPY = simple);
ALTER TEXT SEARCH CONFIGURATION vietnamese
    ALTER MAPPING FOR hword, hword_part, word
    WITH unaccent, simple;
```


#### 8.3 Update Trigger to Use Vietnamese Config

```sql
CREATE OR REPLACE FUNCTION update_book_search_vector()
RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector := 
        setweight(to_tsvector('vietnamese', COALESCE(NEW.title, '')), 'A') ||
        setweight(to_tsvector('vietnamese', COALESCE(NEW.description, '')), 'B');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Update trigger
DROP TRIGGER IF EXISTS books_search_update ON books;
CREATE TRIGGER books_search_update
    BEFORE INSERT OR UPDATE ON books
    FOR EACH ROW
    EXECUTE FUNCTION update_book_search_vector();

-- Backfill existing data
UPDATE books SET search_vector = 
    setweight(to_tsvector('vietnamese', COALESCE(title, '')), 'A') ||
    setweight(to_tsvector('vietnamese', COALESCE(description, '')), 'B');
```


#### 8.4 Search Service Implementation

```go
func (s *BookService) SearchBooks(ctx context.Context, query string, filters SearchFilters) (*SearchResult, error) {
    // 1. Build full-text search query
    searchQuery := `
        SELECT 
            b.*,
            a.name as author_name,
            c.name as category_name,
            ts_rank(b.search_vector, websearch_to_tsquery('vietnamese', $1)) as rank
        FROM books b
        LEFT JOIN authors a ON b.author_id = a.id
        LEFT JOIN categories c ON b.category_id = c.id
        WHERE b.search_vector @@ websearch_to_tsquery('vietnamese', $1)
        AND b.is_active = true
        AND b.deleted_at IS NULL
    `
    
    args := []interface{}{query}
    argPos := 2
    
    // 2. Apply additional filters
    if filters.CategoryID != nil {
        searchQuery += fmt.Sprintf(" AND b.category_id = $%d", argPos)
        args = append(args, *filters.CategoryID)
        argPos++
    }
    
    if filters.MinPrice != nil {
        searchQuery += fmt.Sprintf(" AND b.price >= $%d", argPos)
        args = append(args, *filters.MinPrice)
        argPos++
    }
    
    if filters.MaxPrice != nil {
        searchQuery += fmt.Sprintf(" AND b.price <= $%d", argPos)
        args = append(args, *filters.MaxPrice)
        argPos++
    }
    
    if filters.Language != nil {
        searchQuery += fmt.Sprintf(" AND b.language = $%d", argPos)
        args = append(args, *filters.Language)
        argPos++
    }
    
    // 3. Sort by relevance (rank) + other criteria
    searchQuery += " ORDER BY rank DESC, b.view_count DESC, b.created_at DESC"
    
    // 4. Pagination
    searchQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argPos, argPos+1)
    args = append(args, filters.Limit, (filters.Page-1)*filters.Limit)
    
    // 5. Execute query
    rows, err := s.db.QueryContext(ctx, searchQuery, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    books := []Book{}
    for rows.Next() {
        var b Book
        var rank float64
        err := rows.Scan(
            // ... book fields
            &b.AuthorName,
            &b.CategoryName,
            &rank,
        )
        if err != nil {
            return nil, err
        }
        
        b.SearchRank = rank
        books = append(books, b)
    }
    
    // 6. Count total results
    countQuery := `
        SELECT COUNT(*)
        FROM books
        WHERE search_vector @@ websearch_to_tsquery('vietnamese', $1)
        AND is_active = true
        AND deleted_at IS NULL
    `
    
    var total int
    s.db.QueryRowContext(ctx, countQuery, query).Scan(&total)
    
    return &SearchResult{
        Books: books,
        Total: total,
        Query: query,
    }, nil
}
```


#### 8.5 Search Suggestions / Autocomplete

```go
func (s *BookService) SearchSuggestions(ctx context.Context, query string, limit int) ([]string, error) {
    // Use trigram similarity for autocomplete
    sqlQuery := `
        SELECT DISTINCT title
        FROM books
        WHERE title ILIKE $1
        AND is_active = true
        ORDER BY similarity(title, $2) DESC
        LIMIT $3
    `
    
    pattern := query + "%"
    rows, err := s.db.QueryContext(ctx, sqlQuery, pattern, query, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    suggestions := []string{}
    for rows.Next() {
        var title string
        rows.Scan(&title)
        suggestions = append(suggestions, title)
    }
    
    return suggestions, nil
}
```


#### 8.6 Search Highlighting

Highlight matched terms trong results:

```go
// Use ts_headline function
searchQuery := `
    SELECT 
        b.*,
        ts_headline('vietnamese', b.title, websearch_to_tsquery('vietnamese', $1), 
            'StartSel=<mark>, StopSel=</mark>') as highlighted_title,
        ts_headline('vietnamese', b.description, websearch_to_tsquery('vietnamese', $1),
            'StartSel=<mark>, StopSel=</mark>, MaxWords=50') as highlighted_description
    FROM books b
    WHERE b.search_vector @@ websearch_to_tsquery('vietnamese', $1)
    ...
`
```


### Acceptance Criteria

- Full-text search với tsvector hoạt động[^1]
- Vietnamese text support (unaccent)[^1]
- Search results ranked by relevance[^1]
- Performance: P95 < 100ms cho 10k books[^1]
- Support multi-word queries: "sách văn học"[^1]
- Autocomplete suggestions[^1]


### Dependencies

- P1-T018: Basic search[^1]
- P1-T003: Books table với search_vector[^1]


### Effort

2 ngày[^1]

***

## 9. Advanced Filters (Price, Language) (P2-T026)

### Mô tả

Mở rộng book listing/search với advanced filters.[^1]

### Filter Options

1. **Price range**: min_price, max_price[^1]
2. **Language**: vi, en[^1]
3. **Format**: paperback, hardcover, ebook[^1]
4. **Publisher**: publisher_id[^1]
5. **Rating**: min_rating (1-5)[^1]
6. **Availability**: in_stock, pre_order[^1]

### API Endpoint

`GET /v1/books?category=xxx&price_min=50000&price_max=200000&language=vi&format=paperback&rating=4&in_stock=true`

### Công việc cụ thể

#### 9.1 Update Book Listing Service

```go
type BookFilters struct {
    // Existing
    CategoryID *string
    Search     *string
    Page       int
    Limit      int
    
    // NEW: Advanced filters
    PriceMin    *float64
    PriceMax    *float64
    Language    *string
    Format      *string
    PublisherID *string
    MinRating   *float64
    InStock     *bool
    SortBy      string // price_asc, price_desc, rating, newest, popular
}

func (s *BookService) ListBooks(ctx context.Context, filters BookFilters) (*BookListResult, error) {
    query := `
        SELECT 
            b.*,
            a.name as author_name,
            c.name as category_name,
            p.name as publisher_name,
            (SELECT SUM(quantity - reserved) FROM warehouse_inventory WHERE book_id = b.id) as available_stock
        FROM books b
        LEFT JOIN authors a ON b.author_id = a.id
        LEFT JOIN categories c ON b.category_id = c.id
        LEFT JOIN publishers p ON b.publisher_id = p.id
        WHERE b.is_active = true
        AND b.deleted_at IS NULL
    `
    
    args := []interface{}{}
    argPos := 1
    
    // Apply filters
    if filters.CategoryID != nil {
        query += fmt.Sprintf(" AND b.category_id = $%d", argPos)
        args = append(args, *filters.CategoryID)
        argPos++
    }
    
    if filters.PriceMin != nil {
        query += fmt.Sprintf(" AND b.price >= $%d", argPos)
        args = append(args, *filters.PriceMin)
        argPos++
    }
    
    if filters.PriceMax != nil {
        query += fmt.Sprintf(" AND b.price <= $%d", argPos)
        args = append(args, *filters.PriceMax)
        argPos++
    }
    
    if filters.Language != nil {
        query += fmt.Sprintf(" AND b.language = $%d", argPos)
        args = append(args, *filters.Language)
        argPos++
    }
    
    if filters.Format != nil {
        query += fmt.Sprintf(" AND b.format = $%d", argPos)
        args = append(args, *filters.Format)
        argPos++
    }
    
    if filters.PublisherID != nil {
        query += fmt.Sprintf(" AND b.publisher_id = $%d", argPos)
        args = append(args, *filters.PublisherID)
        argPos++
    }
    
    if filters.MinRating != nil {
        query += fmt.Sprintf(" AND b.average_rating >= $%d", argPos)
        args = append(args, *filters.MinRating)
        argPos++
    }
    
    if filters.InStock != nil && *filters.InStock {
        query += " AND (SELECT SUM(quantity - reserved) FROM warehouse_inventory WHERE book_id = b.id) > 0"
    }
    
    // Sorting
    switch filters.SortBy {
    case "price_asc":
        query += " ORDER BY b.price ASC"
    case "price_desc":
        query += " ORDER BY b.price DESC"
    case "rating":
        query += " ORDER BY b.average_rating DESC, b.review_count DESC"
    case "newest":
        query += " ORDER BY b.created_at DESC"
    case "popular":
        query += " ORDER BY b.view_count DESC, b.review_count DESC"
    default:
        query += " ORDER BY b.created_at DESC"
    }
    
    // Pagination
    query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argPos, argPos+1)
    args = append(args, filters.Limit, (filters.Page-1)*filters.Limit)
    
    // Execute...
    rows, err := s.db.QueryContext(ctx, query, args...)
    // ... rest of implementation
    
    return &BookListResult{Books: books, Total: total}, nil
}
```


#### 9.2 Filter Aggregations (Facets)

Trả về aggregations để frontend hiển thị filter options:

```go
func (s *BookService) GetFilterAggregations(ctx context.Context, categoryID *string) (*FilterAggregations, error) {
    aggs := &FilterAggregations{}
    
    // Base condition
    baseCondition := "WHERE is_active = true AND deleted_at IS NULL"
    args := []interface{}{}
    if categoryID != nil {
        baseCondition += " AND category_id = $1"
        args = append(args, *categoryID)
    }
    
    // 1. Price range
    priceQuery := fmt.Sprintf(`
        SELECT 
            MIN(price) as min_price,
            MAX(price) as max_price
        FROM books
        %s
    `, baseCondition)
    s.db.QueryRowContext(ctx, priceQuery, args...).Scan(&aggs.PriceMin, &aggs.PriceMax)
    
    // 2. Languages
    langQuery := fmt.Sprintf(`
        SELECT language, COUNT(*) as count
        FROM books
        %s
        GROUP BY language
        ORDER BY count DESC
    `, baseCondition)
    rows, _ := s.db.QueryContext(ctx, langQuery, args...)
    for rows.Next() {
        var lang string
        var count int
        rows.Scan(&lang, &count)
        aggs.Languages = append(aggs.Languages, FilterOption{Value: lang, Count: count})
    }
    rows.Close()
    
    // 3. Formats
    formatQuery := fmt.Sprintf(`
        SELECT format, COUNT(*) as count
        FROM books
        %s
        GROUP BY format
        ORDER BY count DESC
    `, baseCondition)
    rows, _ = s.db.QueryContext(ctx, formatQuery, args...)
    for rows.Next() {
        var format string
        var count int
        rows.Scan(&format, &count)
        aggs.Formats = append(aggs.Formats, FilterOption{Value: format, Count: count})
    }
    rows.Close()
    
    // 4. Publishers (top 10)
    pubQuery := fmt.Sprintf(`
        SELECT p.id, p.name, COUNT(b.id) as count
        FROM books b
        JOIN publishers p ON b.publisher_id = p.id
        %s
        GROUP BY p.id, p.name
        ORDER BY count DESC
        LIMIT 10
    `, baseCondition)
    rows, _ = s.db.QueryContext(ctx, pubQuery, args...)
    for rows.Next() {
        var id, name string
        var count int
        rows.Scan(&id, &name, &count)
        aggs.Publishers = append(aggs.Publishers, PublisherOption{ID: id, Name: name, Count: count})
    }
    rows.Close()
    
    return aggs, nil
}
```


#### 9.3 Response với Aggregations

```json
{
  "success": true,
  "data": {
    "books": [...],
    "aggregations": {
      "price_min": 20000,
      "price_max": 500000,
      "languages": [
        {"value": "vi", "count": 850},
        {"value": "en", "count": 150}
      ],
      "formats": [
        {"value": "paperback", "count": 600},
        {"value": "ebook", "count": 300},
        {"value": "hardcover", "count": 100}
      ],
      "publishers": [
        {"id": "uuid", "name": "NXB Kim Đồng", "count": 120},
        {"id": "uuid", "name": "NXB Trẻ", "count": 95}
      ]
    }
  },
  "meta": {
    "page": 1,
    "limit": 20,
    "total": 150
  }
}
```


### Acceptance Criteria

- Filter books theo price range, language, format, publisher, rating[^1]
- Multiple filters có thể combine[^1]
- Sort by price, rating, newest, popular[^1]
- Return filter aggregations (facets)[^1]
- Performance: P95 < 200ms với multiple filters[^1]


### Dependencies

- P1-T015: Book listing API[^1]
- P2-T025: Full-text search[^1]


### Effort

1 ngày[^1]

***

## 10. Integration Tests (Testcontainers) (P2-T027)

### Mô tả

Viết integration tests với real PostgreSQL và Redis containers sử dụng Testcontainers.[^1]

### Testing Strategy

- **Unit tests**: Mock dependencies (đã có ở P1-T037)[^1]
- **Integration tests**: Real database, real Redis[^1]
- **E2E tests**: Full API flows[^1]


### Công việc cụ thể

#### 10.1 Setup Testcontainers

```go
// tests/integration/setup_test.go
package integration

import (
    "context"
    "testing"
    
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
    "github.com/testcontainers/testcontainers-go/modules/redis"
)

type TestEnv struct {
    PostgresContainer *postgres.PostgresContainer
    RedisContainer    *redis.RedisContainer
    DB                *sql.DB
    RedisClient       *redis.Client
}

func SetupTestEnv(t *testing.T) *TestEnv {
    ctx := context.Background()
    
    // 1. Start PostgreSQL container
    postgresC, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:15-alpine"),
        postgres.WithDatabase("bookstore_test"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
    )
    if err != nil {
        t.Fatal(err)
    }
    
    // 2. Start Redis container
    redisC, err := redis.RunContainer(ctx,
        testcontainers.WithImage("redis:7-alpine"),
    )
    if err != nil {
        t.Fatal(err)
    }
    
    // 3. Connect to PostgreSQL
    connStr, _ := postgresC.ConnectionString(ctx)
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        t.Fatal(err)
    }
    
    // 4. Run migrations
    RunMigrations(db)
    
    // 5. Connect to Redis
    redisAddr, _ := redisC.Endpoint(ctx, "")
    redisClient := redis.NewClient(&redis.Options{
        Addr: redisAddr,
    })
    
    return &TestEnv{
        PostgresContainer: postgresC,
        RedisContainer:    redisC,
        DB:                db,
        RedisClient:       redisClient,
    }
}

func (env *TestEnv) Teardown(t *testing.T) {
    env.DB.Close()
    env.RedisClient.Close()
    env.PostgresContainer.Terminate(context.Background())
    env.RedisContainer.Terminate(context.Background())
}
```


#### 10.2 Integration Test - Full Checkout Flow

```go
// tests/integration/checkout_test.go
func TestCheckoutFlow(t *testing.T) {
    // Setup
    env := SetupTestEnv(t)
    defer env.Teardown(t)
    
    // Seed data
    user := SeedUser(env.DB)
    book := SeedBook(env.DB, 5) // 5 copies in stock
    address := SeedAddress(env.DB, user.ID)
    
    // Initialize services
    cartService := service.NewCartService(env.DB, env.RedisClient)
    orderService := service.NewOrderService(env.DB, env.RedisClient)
    inventoryService := service.NewInventoryService(env.DB)
    
    ctx := context.Background()
    
    // Step 1: Add to cart
    err := cartService.AddItem(ctx, user.ID, book.ID, 2)
    assert.NoError(t, err)
    
    // Verify cart
    cart, _ := cartService.GetCart(ctx, user.ID)
    assert.Len(t, cart.Items, 1)
    assert.Equal(t, 2, cart.Items[^0].Quantity)
    
    // Step 2: Apply promo code
    promo := SeedPromotion(env.DB, "TEST20", 20.0)
    err = cartService.ApplyPromotion(ctx, user.ID, promo.Code)
    assert.NoError(t, err)
    
    // Step 3: Checkout
    order, err := orderService.CreateOrder(ctx, service.CreateOrderParams{
        UserID:        user.ID,
        AddressID:     address.ID,
        PaymentMethod: "cod",
    })
    assert.NoError(t, err)
    assert.NotNil(t, order)
    assert.Equal(t, "pending", order.Status)
    
    // Step 4: Verify inventory reserved
    inventory, _ := inventoryService.GetInventory(ctx, book.ID)
    assert.Equal(t, 2, inventory.Reserved)
    assert.Equal(t, 3, inventory.Available) // 5 - 2
    
    // Step 5: Confirm order
    err = orderService.ConfirmOrder(ctx, order.ID)
    assert.NoError(t, err)
    
    // Step 6: Verify inventory deducted
    inventory, _ = inventoryService.GetInventory(ctx, book.ID)
    assert.Equal(t, 0, inventory.Reserved)
    assert.Equal(t, 3, inventory.Quantity) // Actually deducted
    
    // Step 7: Verify cart cleared
    cart, _ = cartService.GetCart(ctx, user.ID)
    assert.Len(t, cart.Items, 0)
}
```


#### 10.3 Integration Test - Payment Flow

```go
func TestVNPayPaymentFlow(t *testing.T) {
    env := SetupTestEnv(t)
    defer env.Teardown(t)
    
    user := SeedUser(env.DB)
    order := SeedOrder(env.DB, user.ID, "pending")
    
    paymentService := service.NewPaymentService(env.DB, env.RedisClient)
    
    ctx := context.Background()
    
    // Step 1: Create payment
    paymentURL, payment, err := paymentService.CreateVNPayPayment(ctx, service.CreateVNPayPaymentParams{
        UserID:  user.ID,
        OrderID: order.ID,
        IPAddr:  "127.0.0.1",
    })
    
    assert.NoError(t, err)
    assert.NotEmpty(t, paymentURL)
    assert.Equal(t, "pending", payment.Status)
    
    // Step 2: Simulate VNPay IPN callback
    err = paymentService.ProcessVNPayIPN(ctx, service.ProcessVNPayIPNParams{
        TxnRef:        order.OrderNumber,
        Amount:        fmt.Sprintf("%d", int(order.Total*100)),
        ResponseCode:  "00", // Success
        TransactionNo: "VNP123456",
    })
    
    assert.NoError(t, err)
    
    // Step 3: Verify order paid
    updatedOrder, _ := paymentService.GetOrder(ctx, order.ID)
    assert.Equal(t, "confirmed", updatedOrder.Status)
    assert.Equal(t, "paid", updatedOrder.PaymentStatus)
    
    // Step 4: Verify payment record
    updatedPayment, _ := paymentService.GetPayment(ctx, payment.ID)
    assert.Equal(t, "success", updatedPayment.Status)
    assert.Equal(t, "VNP123456", updatedPayment.VnpTransactionNo)
}
```


#### 10.4 Integration Test - Search

```go
func TestFullTextSearch(t *testing.T) {
    env := SetupTestEnv(t)
    defer env.Teardown(t)
    
    // Seed books
    SeedBook(env.DB, Book{Title: "Nhà Giả Kim", Description: "Tiểu thuyết triết lý"})
    SeedBook(env.DB, Book{Title: "Đắc Nhân Tâm", Description: "Sách kỹ năng sống"})
    SeedBook(env.DB, Book{Title: "Sapiens", Description: "Lịch sử loài người"})
    
    bookService := service.NewBookService(env.DB)
    ctx := context.Background()
    
    // Test 1: Search by title
    result, _ := bookService.SearchBooks(ctx, "Nhà Giả Kim", SearchFilters{Page: 1, Limit: 10})
    assert.Len(t, result.Books, 1)
    assert.Contains(t, result.Books[^0].Title, "Nhà Giả Kim")
    
    // Test 2: Search by description
    result, _ = bookService.SearchBooks(ctx, "triết lý", SearchFilters{Page: 1, Limit: 10})
    assert.Len(t, result.Books, 1)
    
    // Test 3: Search with Vietnamese accents
    result, _ = bookService.SearchBooks(ctx, "Dac Nhan Tam", SearchFilters{Page: 1, Limit: 10})
    assert.Len(t, result.Books, 1) // Should find "Đắc Nhân Tâm"
}
```


#### 10.5 Integration Test - Wishlist

```go
func TestWishlistFlow(t *testing.T) {
    env := SetupTestEnv(t)
    defer env.Teardown(t)
    
    user := SeedUser(env.DB)
    book1 := SeedBook(env.DB, Book{Title: "Book 1"})
    book2 := SeedBook(env.DB, Book{Title: "Book 2"})
    
    wishlistService := service.NewWishlistService(env.DB)
    ctx := context.Background()
    
    // Add to wishlist
    err := wishlistService.AddToWishlist(ctx, user.ID, book1.ID)
    assert.NoError(t, err)
    
    err = wishlistService.AddToWishlist(ctx, user.ID, book2.ID)
    assert.NoError(t, err)
    
    // Get wishlist
    wishlist, _ := wishlistService.GetWishlist(ctx, user.ID, 1, 10)
    assert.Len(t, wishlist.Items, 2)
    
    // Remove from wishlist
    err = wishlistService.RemoveFromWishlist(ctx, user.ID, book1.ID)
    assert.NoError(t, err)
    
    wishlist, _ = wishlistService.GetWishlist(ctx, user.ID, 1, 10)
    assert.Len(t, wishlist.Items, 1)
}
```


#### 10.6 Run Integration Tests

```bash
# Run all integration tests
go test -v ./tests/integration/...

# Run with coverage
go test -v -coverprofile=coverage-integration.out ./tests/integration/...

# Run specific test
go test -v -run TestCheckoutFlow ./tests/integration/
```


### Acceptance Criteria

- Integration tests với real database containers[^1]
- Test critical flows: checkout, payment, search, wishlist[^1]
- Tests chạy isolated (mỗi test có own container)[^1]
- Test execution time < 5 minutes[^1]
- Coverage critical paths: 80%+[^1]
- CI/CD integration (GitHub Actions)[^1]


### Dependencies

- All Sprint 13-14 tasks[^1]


### Effort

3 ngày[^1]

***

## SUMMARY

### Total Effort Sprint 13-14

| Task ID | Task | Effort (days) |
| :-- | :-- | :-- |
| P2-T018 | Wishlist table | 0.5 |
| P2-T019 | Add/Remove wishlist APIs | 1 |
| P2-T020 | Get wishlist API | 0.5 |
| P2-T021 | Banners table | 0.5 |
| P2-T022 | Admin CRUD banners | 2 |
| P2-T023 | Public Get active banners | 0.5 |
| P2-T024 | Banner click tracking | 1 |
| P2-T025 | Full-text search (tsvector) | 2 |
| P2-T026 | Advanced filters (price, language) | 1 |
| P2-T027 | Integration tests (Testcontainers) | 3 |
| **TOTAL** |  | **12 days** |

**Sprint duration**: 2 tuần (10 ngày làm việc)[^1]
**Team size**: 2 backend developers (có thể song song hóa tasks)[^1]

### Parallelization Strategy

**Week 1** (5 ngày):

- **Dev 1**: P2-T018 → P2-T019 → P2-T020 (Wishlist features) (0.5+1+0.5 = 2 days) + P2-T025 (Full-text search) (2 days) + Review (1 day)[^1]
- **Dev 2**: P2-T021 → P2-T022 → P2-T023 → P2-T024 (Banners features) (0.5+2+0.5+1 = 4 days) + Review (1 day)[^1]

**Week 2** (5 ngày):

- **Dev 1**: P2-T026 (Advanced filters) (1 day) + P2-T027 (Integration tests) (3 days, focus on wishlist/search) + Bug fixes (1 day)[^1]
- **Dev 2**: P2-T027 (Integration tests) (3 days, focus on banners/filters) + Bug fixes + Documentation (2 days)[^1]


### Deliverables Checklist Sprint 13-14

- ✅ **Wishlist feature** hoàn chỉnh (add, remove, view)[^1]
- ✅ **Banner management** system với admin CRUD và public display[^1]
- ✅ **Click tracking** cho banners với analytics[^1]
- ✅ **Full-text search** với tsvector, Vietnamese support[^1]
- ✅ **Advanced filters** (price, language, format, publisher, rating)[^1]
- ✅ **Integration tests** với Testcontainers coverage > 70%[^1]


### Key Technical Achievements

**Performance**:

- Full-text search: P95 < 100ms (10k books)[^1]
- Advanced filters: P95 < 200ms (multiple filters)[^1]
- Banner API (cached): P95 < 50ms[^1]

**Search Quality**:

- Relevance ranking với ts_rank[^1]
- Vietnamese text support (unaccent)[^1]
- Multi-word queries[^1]
- Autocomplete suggestions[^1]

**Testing**:

- Integration tests với real PostgreSQL \& Redis[^1]
- Test isolation (mỗi test có own containers)[^1]
- Critical flow coverage: checkout, payment, search[^1]


### Next Steps (Sprint 15-16)

Phase 3 sẽ focus vào Multi-Warehouse Inventory và eBook Management.[^1]

<div align="center">⁂</div>

[^1]: USER-REQUIREMENTS-DOCUMENT-URD-PHIEN-BAN-HOA.docx

