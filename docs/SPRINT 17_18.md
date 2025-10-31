<img src="https://r2cdn.perplexity.ai/pplx-full-logo-primary-dark%402x.png" style="height:64px;margin-right:32px"/>

# TODO LIST CHI TIẾT CHO BACKEND DEVELOPER - SPRINT 17-18: EBOOK MANAGEMENT

Dựa trên URD, dưới đây là danh sách công việc chi tiết và đầy đủ cho backend developer trong Sprint 17-18 (Phase 3, 2 tuần - 10 ngày làm việc).[^1]

## 1. S3/MinIO Setup for eBook Storage (P3-T009)

### Mô tả

Thiết lập object storage (S3 hoặc MinIO) để lưu trữ và phân phối file ebook.[^1]

### Technology Choice

- **Production**: AWS S3 (unlimited capacity, high durability)[^1]
- **Development/Self-hosted**: MinIO (S3-compatible, open source)[^1]


### Công việc cụ thể

#### 1.1 Environment Configuration

Tạo file `config/storage.go`:[^1]

```go
package config

type StorageConfig struct {
    Type            string // "s3" or "minio"
    S3Region        string
    S3Bucket        string
    S3AccessKey     string
    S3SecretKey     string
    MinIOEndpoint   string
    MinIOBucket     string
    MinIOAccessKey  string
    MinIOSecretKey  string
    MinIOUseSSL     bool
}

func LoadStorageConfig() *StorageConfig {
    storageType := getEnv("STORAGE_TYPE", "s3")
    
    return &StorageConfig{
        Type:            storageType,
        S3Region:        getEnv("AWS_REGION", "ap-southeast-1"),
        S3Bucket:        getEnv("S3_BUCKET", "bookstore-ebooks"),
        S3AccessKey:     getEnv("AWS_ACCESS_KEY_ID", ""),
        S3SecretKey:     getEnv("AWS_SECRET_ACCESS_KEY", ""),
        MinIOEndpoint:   getEnv("MINIO_ENDPOINT", "localhost:9000"),
        MinIOBucket:     getEnv("MINIO_BUCKET", "ebooks"),
        MinIOAccessKey:  getEnv("MINIO_ACCESS_KEY", "minioadmin"),
        MinIOSecretKey:  getEnv("MINIO_SECRET_KEY", "minioadmin"),
        MinIOUseSSL:     getEnv("MINIO_USE_SSL", "false") == "true",
    }
}
```


#### 1.2 Storage Service Interface

Tạo file `internal/infrastructure/storage/interface.go`:[^1]

```go
package storage

import (
    "context"
    "io"
    "time"
)

type StorageService interface {
    // Upload file to storage
    Upload(ctx context.Context, key string, file io.Reader, contentType string, size int64) (string, error)
    
    // Generate presigned download URL (valid for limited time)
    GeneratePresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error)
    
    // Delete file from storage
    Delete(ctx context.Context, key string) error
    
    // Check if file exists
    Exists(ctx context.Context, key string) (bool, error)
    
    // Get file metadata
    GetMetadata(ctx context.Context, key string) (*FileMetadata, error)
}

type FileMetadata struct {
    Key          string
    Size         int64
    ContentType  string
    LastModified time.Time
}
```


#### 1.3 AWS S3 Implementation

Tạo file `internal/infrastructure/storage/s3.go`:[^1]

```go
package storage

import (
    "context"
    "fmt"
    "io"
    "time"
    
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/credentials"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/s3"
)

type S3Storage struct {
    client *s3.S3
    bucket string
}

func NewS3Storage(region, bucket, accessKey, secretKey string) (*S3Storage, error) {
    sess, err := session.NewSession(&aws.Config{
        Region:      aws.String(region),
        Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create AWS session: %w", err)
    }
    
    return &S3Storage{
        client: s3.New(sess),
        bucket: bucket,
    }, nil
}

func (s *S3Storage) Upload(ctx context.Context, key string, file io.Reader, contentType string, size int64) (string, error) {
    _, err := s.client.PutObjectWithContext(ctx, &s3.PutObjectInput{
        Bucket:        aws.String(s.bucket),
        Key:           aws.String(key),
        Body:          aws.ReadSeekCloser(file),
        ContentType:   aws.String(contentType),
        ContentLength: aws.Int64(size),
        ServerSideEncryption: aws.String("AES256"), // Enable encryption at rest
    })
    
    if err != nil {
        return "", fmt.Errorf("failed to upload to S3: %w", err)
    }
    
    // Return S3 URL (not used for downloads - use presigned URLs)
    url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, *s.client.Config.Region, key)
    return url, nil
}

func (s *S3Storage) GeneratePresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
    req, _ := s.client.GetObjectRequest(&s3.GetObjectInput{
        Bucket: aws.String(s.bucket),
        Key:    aws.String(key),
    })
    
    url, err := req.Presign(expiration)
    if err != nil {
        return "", fmt.Errorf("failed to generate presigned URL: %w", err)
    }
    
    return url, nil
}

func (s *S3Storage) Delete(ctx context.Context, key string) error {
    _, err := s.client.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
        Bucket: aws.String(s.bucket),
        Key:    aws.String(key),
    })
    
    return err
}

func (s *S3Storage) Exists(ctx context.Context, key string) (bool, error) {
    _, err := s.client.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
        Bucket: aws.String(s.bucket),
        Key:    aws.String(key),
    })
    
    if err != nil {
        if aerr, ok := err.(awserr.Error); ok {
            if aerr.Code() == s3.ErrCodeNoSuchKey || aerr.Code() == "NotFound" {
                return false, nil
            }
        }
        return false, err
    }
    
    return true, nil
}

func (s *S3Storage) GetMetadata(ctx context.Context, key string) (*FileMetadata, error) {
    output, err := s.client.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
        Bucket: aws.String(s.bucket),
        Key:    aws.String(key),
    })
    
    if err != nil {
        return nil, err
    }
    
    return &FileMetadata{
        Key:          key,
        Size:         *output.ContentLength,
        ContentType:  *output.ContentType,
        LastModified: *output.LastModified,
    }, nil
}
```


#### 1.4 MinIO Implementation

Tạo file `internal/infrastructure/storage/minio.go`:[^1]

```go
package storage

import (
    "context"
    "fmt"
    "io"
    "time"
    
    "github.com/minio/minio-go/v7"
    "github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOStorage struct {
    client *minio.Client
    bucket string
}

func NewMinIOStorage(endpoint, bucket, accessKey, secretKey string, useSSL bool) (*MinIOStorage, error) {
    client, err := minio.New(endpoint, &minio.Options{
        Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
        Secure: useSSL,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create MinIO client: %w", err)
    }
    
    // Create bucket if not exists
    ctx := context.Background()
    exists, err := client.BucketExists(ctx, bucket)
    if err != nil {
        return nil, err
    }
    
    if !exists {
        err = client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
        if err != nil {
            return nil, fmt.Errorf("failed to create bucket: %w", err)
        }
        log.Info("Created MinIO bucket", "bucket", bucket)
    }
    
    return &MinIOStorage{
        client: client,
        bucket: bucket,
    }, nil
}

func (m *MinIOStorage) Upload(ctx context.Context, key string, file io.Reader, contentType string, size int64) (string, error) {
    _, err := m.client.PutObject(ctx, m.bucket, key, file, size, minio.PutObjectOptions{
        ContentType: contentType,
    })
    
    if err != nil {
        return "", fmt.Errorf("failed to upload to MinIO: %w", err)
    }
    
    return fmt.Sprintf("%s/%s/%s", m.client.EndpointURL(), m.bucket, key), nil
}

func (m *MinIOStorage) GeneratePresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
    url, err := m.client.PresignedGetObject(ctx, m.bucket, key, expiration, nil)
    if err != nil {
        return "", fmt.Errorf("failed to generate presigned URL: %w", err)
    }
    
    return url.String(), nil
}

func (m *MinIOStorage) Delete(ctx context.Context, key string) error {
    return m.client.RemoveObject(ctx, m.bucket, key, minio.RemoveObjectOptions{})
}

func (m *MinIOStorage) Exists(ctx context.Context, key string) (bool, error) {
    _, err := m.client.StatObject(ctx, m.bucket, key, minio.StatObjectOptions{})
    if err != nil {
        if minio.ToErrorResponse(err).Code == "NoSuchKey" {
            return false, nil
        }
        return false, err
    }
    return true, nil
}

func (m *MinIOStorage) GetMetadata(ctx context.Context, key string) (*FileMetadata, error) {
    info, err := m.client.StatObject(ctx, m.bucket, key, minio.StatObjectOptions{})
    if err != nil {
        return nil, err
    }
    
    return &FileMetadata{
        Key:          key,
        Size:         info.Size,
        ContentType:  info.ContentType,
        LastModified: info.LastModified,
    }, nil
}
```


#### 1.5 Factory Pattern

```go
func NewStorageService(cfg *config.StorageConfig) (StorageService, error) {
    switch cfg.Type {
    case "s3":
        return NewS3Storage(cfg.S3Region, cfg.S3Bucket, cfg.S3AccessKey, cfg.S3SecretKey)
    case "minio":
        return NewMinIOStorage(cfg.MinIOEndpoint, cfg.MinIOBucket, cfg.MinIOAccessKey, cfg.MinIOSecretKey, cfg.MinIOUseSSL)
    default:
        return nil, fmt.Errorf("unsupported storage type: %s", cfg.Type)
    }
}
```


#### 1.6 Docker Compose for Local Development

```yaml
# docker-compose.yml
services:
  minio:
    image: minio/minio:latest
    command: server /data --console-address ":9001"
    ports:
      - "9000:9000"
      - "9001:9001"
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    volumes:
      - minio_data:/data
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 30s
      timeout: 20s
      retries: 3

volumes:
  minio_data:
```


### Acceptance Criteria

- S3/MinIO client kết nối thành công[^1]
- Upload file hoạt động (test với file nhỏ)[^1]
- Generate presigned URL valid và downloadable[^1]
- Delete file thành công[^1]
- Local MinIO setup hoạt động với docker-compose[^1]


### Dependencies

- None (độc lập)[^1]


### Effort

1 ngày[^1]

***

## 2. Upload eBook File API (Admin) (P3-T010)

### Mô tả

Admin API để upload file ebook (PDF, EPUB) lên storage.[^1]

### API Endpoint

`POST /v1/admin/books/:book_id/ebook`[^1]

### Request Format

- **Content-Type**: `multipart/form-data`[^1]
- **Fields**:
    - `file`: File upload (required)[^1]
    - `format`: "pdf" hoặc "epub" (optional, auto-detect từ file)[^1]


### Business Rules

- Max file size: 50MB[^1]
- Allowed formats: PDF (.pdf), EPUB (.epub)[^1]
- File được encrypt at rest[^1]
- Filename format: `ebooks/{book_id}/{format}/{timestamp}_{filename}`[^1]


### Công việc cụ thể

#### 2.1 Database Schema Update

```sql
-- Update books table to support ebook
ALTER TABLE books 
ADD COLUMN IF NOT EXISTS ebook_file_key TEXT,
ADD COLUMN IF NOT EXISTS ebook_file_size_mb DECIMAL(5,2),
ADD COLUMN IF NOT EXISTS ebook_format TEXT CHECK (ebook_format IN ('pdf', 'epub')),
ADD COLUMN IF NOT EXISTS ebook_uploaded_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_books_ebook ON books(id) WHERE ebook_file_key IS NOT NULL;
```


#### 2.2 Service Implementation

Tạo file `internal/domains/ebook/service/upload_service.go`:[^1]

```go
package service

import (
    "context"
    "fmt"
    "mime/multipart"
    "path/filepath"
    "strings"
    "time"
)

type EbookUploadService struct {
    bookRepo       *repository.BookRepository
    storageService storage.StorageService
}

const (
    MaxEbookSizeMB = 50
    MaxEbookSizeBytes = MaxEbookSizeMB * 1024 * 1024
)

var AllowedFormats = map[string]string{
    ".pdf":  "application/pdf",
    ".epub": "application/epub+zip",
}

func (s *EbookUploadService) UploadEbook(
    ctx context.Context,
    bookID string,
    file *multipart.FileHeader,
) (*EbookUploadResult, error) {
    
    // 1. Validate book exists
    book, err := s.bookRepo.FindByID(ctx, bookID)
    if err != nil {
        return nil, fmt.Errorf("book not found")
    }
    
    // 2. Validate file size
    if file.Size > MaxEbookSizeBytes {
        return nil, fmt.Errorf("file size exceeds %dMB limit", MaxEbookSizeMB)
    }
    
    // 3. Validate file format
    ext := strings.ToLower(filepath.Ext(file.Filename))
    contentType, ok := AllowedFormats[ext]
    if !ok {
        return nil, fmt.Errorf("unsupported file format: %s. Allowed: .pdf, .epub", ext)
    }
    
    format := strings.TrimPrefix(ext, ".")
    
    // 4. Open file
    src, err := file.Open()
    if err != nil {
        return nil, fmt.Errorf("failed to open file: %w", err)
    }
    defer src.Close()
    
    // 5. Generate storage key
    timestamp := time.Now().Unix()
    sanitizedFilename := sanitizeFilename(file.Filename)
    storageKey := fmt.Sprintf("ebooks/%s/%s/%d_%s", bookID, format, timestamp, sanitizedFilename)
    
    // 6. Upload to storage
    _, err = s.storageService.Upload(ctx, storageKey, src, contentType, file.Size)
    if err != nil {
        return nil, fmt.Errorf("failed to upload file: %w", err)
    }
    
    log.Info("Ebook uploaded to storage",
        "book_id", bookID,
        "storage_key", storageKey,
        "size_mb", float64(file.Size)/(1024*1024),
    )
    
    // 7. Update book record
    book.EbookFileKey = &storageKey
    sizeMB := float64(file.Size) / (1024 * 1024)
    book.EbookFileSizeMB = &sizeMB
    book.EbookFormat = &format
    now := time.Now()
    book.EbookUploadedAt = &now
    
    err = s.bookRepo.Update(ctx, book)
    if err != nil {
        // Try to delete uploaded file
        s.storageService.Delete(ctx, storageKey)
        return nil, fmt.Errorf("failed to update book record: %w", err)
    }
    
    return &EbookUploadResult{
        BookID:     bookID,
        StorageKey: storageKey,
        Format:     format,
        SizeMB:     sizeMB,
    }, nil
}

func sanitizeFilename(filename string) string {
    // Remove special characters, keep only alphanumeric, dash, underscore, dot
    re := regexp.MustCompile(`[^a-zA-Z0-9._-]`)
    return re.ReplaceAllString(filename, "_")
}
```


#### 2.3 Handler Implementation

```go
func (h *EbookHandler) UploadEbook(c *gin.Context) {
    bookID := c.Param("book_id")
    
    // Parse multipart form (max 50MB)
    err := c.Request.ParseMultipartForm(MaxEbookSizeBytes)
    if err != nil {
        c.JSON(400, gin.H{"success": false, "error": "File too large or invalid form"})
        return
    }
    
    // Get file from form
    file, err := c.FormFile("file")
    if err != nil {
        c.JSON(400, gin.H{"success": false, "error": "File is required"})
        return
    }
    
    // Upload
    result, err := h.ebookService.UploadEbook(c.Request.Context(), bookID, file)
    if err != nil {
        c.JSON(400, gin.H{"success": false, "error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{
        "success": true,
        "data": result,
    })
}
```


#### 2.4 Delete Ebook API

`DELETE /v1/admin/books/:book_id/ebook`

```go
func (s *EbookUploadService) DeleteEbook(ctx context.Context, bookID string) error {
    book, err := s.bookRepo.FindByID(ctx, bookID)
    if err != nil {
        return fmt.Errorf("book not found")
    }
    
    if book.EbookFileKey == nil {
        return fmt.Errorf("book has no ebook")
    }
    
    // Delete from storage
    err = s.storageService.Delete(ctx, *book.EbookFileKey)
    if err != nil {
        log.Error("Failed to delete ebook from storage", "error", err, "key", *book.EbookFileKey)
        // Continue to update DB even if storage delete fails
    }
    
    // Update book record
    book.EbookFileKey = nil
    book.EbookFileSizeMB = nil
    book.EbookFormat = nil
    book.EbookUploadedAt = nil
    
    return s.bookRepo.Update(ctx, book)
}
```


### Acceptance Criteria

- Admin upload được file PDF/EPUB[^1]
- Validation file size (max 50MB)[^1]
- Validation file format (chỉ PDF, EPUB)[^1]
- File được lưu vào S3/MinIO với đúng key format[^1]
- Database cập nhật file metadata[^1]
- Admin xóa được ebook đã upload[^1]


### Dependencies

- P3-T009: S3/MinIO setup[^1]
- P1-T029: RBAC middleware[^1]


### Effort

2 ngày[^1]

***

## 3. Generate Presigned Download URL (P3-T011)

### Mô tả

Service để generate presigned URL cho ebook download với expiration time.[^1]

### Technical Details

- **URL Expiration**: 1 hour (configurable)[^1]
- **Security**: URL chỉ valid cho user đã mua sách[^1]
- **No direct access**: Direct S3 URLs không hoạt động[^1]


### Công việc cụ thể

#### 3.1 Database Schema for Download Tracking

```sql
CREATE TABLE ebook_downloads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    book_id UUID NOT NULL REFERENCES books(id),
    order_id UUID NOT NULL REFERENCES orders(id),
    download_url TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    ip_address INET,
    user_agent TEXT,
    downloaded_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_downloads_user ON ebook_downloads(user_id, created_at DESC);
CREATE INDEX idx_downloads_book ON ebook_downloads(book_id);
CREATE INDEX idx_downloads_order ON ebook_downloads(order_id);
CREATE INDEX idx_downloads_expires ON ebook_downloads(expires_at) WHERE downloaded_at IS NULL;
```


#### 3.2 Service Implementation

```go
type EbookDownloadService struct {
    bookRepo        *repository.BookRepository
    orderRepo       *repository.OrderRepository
    downloadRepo    *repository.EbookDownloadRepository
    storageService  storage.StorageService
}

type GenerateDownloadURLParams struct {
    UserID  string
    BookID  string
    IPAddr  string
    UserAgent string
}

func (s *EbookDownloadService) GenerateDownloadURL(
    ctx context.Context,
    params GenerateDownloadURLParams,
) (*DownloadURLResult, error) {
    
    // 1. Validate user has purchased this book
    hasPurchased, orderID, err := s.validatePurchase(ctx, params.UserID, params.BookID)
    if err != nil {
        return nil, err
    }
    
    if !hasPurchased {
        return nil, fmt.Errorf("user has not purchased this book")
    }
    
    // 2. Check download limit (5 downloads per day)
    canDownload, err := s.checkDownloadLimit(ctx, params.UserID, params.BookID)
    if err != nil {
        return nil, err
    }
    
    if !canDownload {
        return nil, fmt.Errorf("daily download limit exceeded (5 downloads/day)")
    }
    
    // 3. Get book ebook file key
    book, err := s.bookRepo.FindByID(ctx, params.BookID)
    if err != nil {
        return nil, fmt.Errorf("book not found")
    }
    
    if book.EbookFileKey == nil {
        return nil, fmt.Errorf("book has no ebook file")
    }
    
    // 4. Generate presigned URL (valid for 1 hour)
    expiration := 1 * time.Hour
    presignedURL, err := s.storageService.GeneratePresignedURL(ctx, *book.EbookFileKey, expiration)
    if err != nil {
        return nil, fmt.Errorf("failed to generate download URL: %w", err)
    }
    
    expiresAt := time.Now().Add(expiration)
    
    // 5. Record download request
    download := &EbookDownload{
        UserID:      params.UserID,
        BookID:      params.BookID,
        OrderID:     orderID,
        DownloadURL: presignedURL,
        ExpiresAt:   expiresAt,
        IPAddress:   params.IPAddr,
        UserAgent:   params.UserAgent,
    }
    
    err = s.downloadRepo.Create(ctx, download)
    if err != nil {
        log.Error("Failed to record download", "error", err)
        // Non-critical, continue
    }
    
    log.Info("Generated ebook download URL",
        "user_id", params.UserID,
        "book_id", params.BookID,
        "expires_at", expiresAt,
    )
    
    return &DownloadURLResult{
        DownloadURL: presignedURL,
        ExpiresAt:   expiresAt,
        BookTitle:   book.Title,
        Format:      *book.EbookFormat,
        SizeMB:      *book.EbookFileSizeMB,
    }, nil
}

func (s *EbookDownloadService) validatePurchase(ctx context.Context, userID string, bookID string) (bool, string, error) {
    // Check if user has completed order containing this book
    query := `
        SELECT o.id
        FROM orders o
        JOIN order_items oi ON o.id = oi.order_id
        WHERE o.user_id = $1
        AND oi.book_id = $2
        AND o.status IN ('completed', 'delivered')
        AND o.payment_status = 'paid'
        ORDER BY o.created_at DESC
        LIMIT 1
    `
    
    var orderID string
    err := s.db.QueryRowContext(ctx, query, userID, bookID).Scan(&orderID)
    
    if err == sql.ErrNoRows {
        return false, "", nil
    }
    if err != nil {
        return false, "", err
    }
    
    return true, orderID, nil
}

func (s *EbookDownloadService) checkDownloadLimit(ctx context.Context, userID string, bookID string) (bool, error) {
    // Count downloads in last 24 hours
    query := `
        SELECT COUNT(*)
        FROM ebook_downloads
        WHERE user_id = $1
        AND book_id = $2
        AND created_at >= NOW() - INTERVAL '24 hours'
    `
    
    var count int
    err := s.db.QueryRowContext(ctx, query, userID, bookID).Scan(&count)
    if err != nil {
        return false, err
    }
    
    return count < 5, nil // Max 5 downloads per day
}
```


### Acceptance Criteria

- Generate được presigned URL valid[^1]
- URL expire sau 1 hour[^1]
- Validate user đã mua sách[^1]
- Download tracking được lưu database[^1]
- URL download được từ S3/MinIO thành công[^1]


### Dependencies

- P3-T009: S3/MinIO setup[^1]
- P3-T010: Upload ebook API[^1]


### Effort

2 ngày[^1]

***

## 4. Download Link API với Validation (P3-T012)

### Mô tả

Public API để user request download link cho ebook đã mua.[^1]

### API Endpoint

`POST /v1/ebooks/:book_id/download-link`[^1]

### Response Format

```json
{
  "success": true,
  "data": {
    "download_url": "https://bookstore-ebooks.s3.amazonaws.com/ebooks/...",
    "expires_at": "2025-10-31T13:30:00Z",
    "book_title": "Nhà Giả Kim",
    "format": "pdf",
    "size_mb": 2.5,
    "remaining_downloads_today": 3
  }
}
```


### Công việc cụ thể

#### 4.1 Handler Implementation

```go
func (h *EbookHandler) RequestDownloadLink(c *gin.Context) {
    bookID := c.Param("book_id")
    userID := c.GetString("user_id") // From JWT
    
    // Get IP and User-Agent
    ipAddr := c.ClientIP()
    userAgent := c.Request.UserAgent()
    
    // Generate download URL
    result, err := h.ebookService.GenerateDownloadURL(c.Request.Context(), GenerateDownloadURLParams{
        UserID:    userID,
        BookID:    bookID,
        IPAddr:    ipAddr,
        UserAgent: userAgent,
    })
    
    if err != nil {
        statusCode := 400
        if strings.Contains(err.Error(), "not purchased") {
            statusCode = 403
        }
        if strings.Contains(err.Error(), "limit exceeded") {
            statusCode = 429
        }
        
        c.JSON(statusCode, gin.H{
            "success": false,
            "error":   err.Error(),
        })
        return
    }
    
    // Get remaining downloads today
    remaining, _ := h.ebookService.GetRemainingDownloadsToday(c.Request.Context(), userID, bookID)
    
    c.JSON(200, gin.H{
        "success": true,
        "data": gin.H{
            "download_url":              result.DownloadURL,
            "expires_at":                result.ExpiresAt,
            "book_title":                result.BookTitle,
            "format":                    result.Format,
            "size_mb":                   result.SizeMB,
            "remaining_downloads_today": remaining,
        },
    })
}
```


#### 4.2 Get Remaining Downloads

```go
func (s *EbookDownloadService) GetRemainingDownloadsToday(ctx context.Context, userID string, bookID string) (int, error) {
    query := `
        SELECT COUNT(*)
        FROM ebook_downloads
        WHERE user_id = $1
        AND book_id = $2
        AND created_at >= NOW() - INTERVAL '24 hours'
    `
    
    var count int
    err := s.db.QueryRowContext(ctx, query, userID, bookID).Scan(&count)
    if err != nil {
        return 0, err
    }
    
    remaining := 5 - count
    if remaining < 0 {
        remaining = 0
    }
    
    return remaining, nil
}
```


#### 4.3 Mark Download as Completed

Client có thể ping endpoint sau khi download xong:

`POST /v1/ebooks/downloads/:download_id/complete`

```go
func (s *EbookDownloadService) MarkDownloadCompleted(ctx context.Context, downloadID string, userID string) error {
    query := `
        UPDATE ebook_downloads
        SET downloaded_at = NOW()
        WHERE id = $1
        AND user_id = $2
        AND downloaded_at IS NULL
    `
    
    result, err := s.db.ExecContext(ctx, query, downloadID, userID)
    if err != nil {
        return err
    }
    
    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0 {
        return fmt.Errorf("download not found or already marked")
    }
    
    return nil
}
```


### Acceptance Criteria

- User request được download link[^1]
- Error 403 nếu chưa mua sách[^1]
- Error 429 nếu vượt quá 5 downloads/day[^1]
- Response bao gồm expiration time[^1]
- Download URL functional[^1]


### Dependencies

- P3-T011: Generate presigned URL[^1]
- P1-T012: JWT middleware[^1]


### Effort

1 ngày[^1]

***

## 5. Download Limit Tracking (5/day) (P3-T013)

### Mô tả

Implement download limit tracking và enforcement (5 downloads per day per book).[^1]

### Business Rules

- Limit: 5 downloads per 24 hours per book per user[^1]
- Counter reset: Rolling 24h window (not calendar day)[^1]
- Admin bypass: Admins không bị limit[^1]


### Công việc cụ thể

#### 5.1 Enhanced Validation

Already implemented in P3-T011, nhưng thêm features:

```go
func (s *EbookDownloadService) checkDownloadLimitDetailed(
    ctx context.Context,
    userID string,
    bookID string,
) (*DownloadLimitInfo, error) {
    
    query := `
        SELECT 
            COUNT(*) as total_downloads,
            MIN(created_at) as oldest_download,
            MAX(created_at) as latest_download
        FROM ebook_downloads
        WHERE user_id = $1
        AND book_id = $2
        AND created_at >= NOW() - INTERVAL '24 hours'
    `
    
    var info DownloadLimitInfo
    var oldestDownload, latestDownload sql.NullTime
    
    err := s.db.QueryRowContext(ctx, query, userID, bookID).Scan(
        &info.TotalDownloads,
        &oldestDownload,
        &latestDownload,
    )
    
    if err != nil {
        return nil, err
    }
    
    info.Remaining = 5 - info.TotalDownloads
    if info.Remaining < 0 {
        info.Remaining = 0
    }
    
    info.CanDownload = info.TotalDownloads < 5
    
    if oldestDownload.Valid {
        info.OldestDownloadAt = &oldestDownload.Time
        // Calculate when limit resets (24h from oldest download)
        resetAt := oldestDownload.Time.Add(24 * time.Hour)
        info.LimitResetsAt = &resetAt
    }
    
    return &info, nil
}

type DownloadLimitInfo struct {
    TotalDownloads   int        `json:"total_downloads"`
    Remaining        int        `json:"remaining"`
    CanDownload      bool       `json:"can_download"`
    OldestDownloadAt *time.Time `json:"oldest_download_at,omitempty"`
    LimitResetsAt    *time.Time `json:"limit_resets_at,omitempty"`
}
```


#### 5.2 Get Download History API

`GET /v1/user/ebooks/:book_id/downloads`

```go
func (h *EbookHandler) GetDownloadHistory(c *gin.Context) {
    bookID := c.Param("book_id")
    userID := c.GetString("user_id")
    
    history, err := h.ebookService.GetDownloadHistory(c.Request.Context(), userID, bookID, 20)
    if err != nil {
        c.JSON(500, gin.H{"success": false, "error": err.Error()})
        return
    }
    
    // Get current limit info
    limitInfo, _ := h.ebookService.GetDownloadLimitInfo(c.Request.Context(), userID, bookID)
    
    c.JSON(200, gin.H{
        "success": true,
        "data": gin.H{
            "downloads":  history,
            "limit_info": limitInfo,
        },
    })
}
```

```go
func (s *EbookDownloadService) GetDownloadHistory(
    ctx context.Context,
    userID string,
    bookID string,
    limit int,
) ([]DownloadRecord, error) {
    
    query := `
        SELECT 
            id,
            created_at,
            downloaded_at,
            expires_at,
            ip_address
        FROM ebook_downloads
        WHERE user_id = $1
        AND book_id = $2
        ORDER BY created_at DESC
        LIMIT $3
    `
    
    rows, err := s.db.QueryContext(ctx, query, userID, bookID, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    records := []DownloadRecord{}
    for rows.Next() {
        var record DownloadRecord
        err := rows.Scan(
            &record.ID,
            &record.CreatedAt,
            &record.DownloadedAt,
            &record.ExpiresAt,
            &record.IPAddress,
        )
        if err != nil {
            return nil, err
        }
        records = append(records, record)
    }
    
    return records, nil
}
```


#### 5.3 Admin Override

Admins có thể reset download limit cho user:

`POST /v1/admin/users/:user_id/ebooks/:book_id/reset-limit`

```go
func (s *AdminEbookService) ResetDownloadLimit(ctx context.Context, userID string, bookID string, reason string) error {
    // Delete download records for this user+book in last 24h
    query := `
        DELETE FROM ebook_downloads
        WHERE user_id = $1
        AND book_id = $2
        AND created_at >= NOW() - INTERVAL '24 hours'
    `
    
    result, err := s.db.ExecContext(ctx, query, userID, bookID)
    if err != nil {
        return err
    }
    
    rowsAffected, _ := result.RowsAffected()
    
    log.Info("Admin reset download limit",
        "user_id", userID,
        "book_id", bookID,
        "deleted_records", rowsAffected,
        "reason", reason,
    )
    
    return nil
}
```


### Acceptance Criteria

- Download limit 5/day được enforce[^1]
- Rolling 24h window (không phải calendar day)[^1]
- Error message rõ ràng khi vượt limit[^1]
- User xem được download history[^1]
- Admin reset được limit cho user[^1]


### Dependencies

- P3-T011: Generate presigned URL[^1]
- P3-T012: Download link API[^1]


### Effort

1 ngày[^1]

***

## 6. Watermark PDF Job (Asynq) (P3-T014)

### Mô tả

Background job để watermark PDF file với user info (email, order ID) để prevent piracy.[^1]

### Watermark Content

- User email[^1]
- Order ID[^1]
- Purchase date[^1]
- Watermark position: Footer center, light opacity[^1]


### Technical Approach

- **Library**: Use `github.com/pdfcpu/pdfcpu` hoặc `github.com/jung-kurt/gofpdf`[^1]
- **Process**: Original file → Watermarked copy → New storage key[^1]
- **Trigger**: After order completed with ebook[^1]


### Công việc cụ thể

#### 6.1 Install PDF Library

```bash
go get github.com/pdfcpu/pdfcpu/pkg/api
```


#### 6.2 Watermark Service

Tạo file `internal/domains/ebook/service/watermark_service.go`:[^1]

```go
package service

import (
    "bytes"
    "context"
    "fmt"
    "io"
    "time"
    
    "github.com/pdfcpu/pdfcpu/pkg/api"
    "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

type WatermarkService struct {
    storageService storage.StorageService
}

type WatermarkParams struct {
    UserEmail    string
    OrderID      string
    PurchaseDate time.Time
}

func (s *WatermarkService) WatermarkPDF(
    ctx context.Context,
    originalKey string,
    params WatermarkParams,
) (string, error) {
    
    // 1. Download original file from storage
    originalFile, err := s.downloadFile(ctx, originalKey)
    if err != nil {
        return "", fmt.Errorf("failed to download original: %w", err)
    }
    
    // 2. Create watermark text
    watermarkText := fmt.Sprintf(
        "Licensed to: %s | Order: %s | Date: %s",
        params.UserEmail,
        params.OrderID,
        params.PurchaseDate.Format("2006-01-02"),
    )
    
    // 3. Apply watermark
    watermarkedFile, err := s.applyWatermark(originalFile, watermarkText)
    if err != nil {
        return "", fmt.Errorf("failed to apply watermark: %w", err)
    }
    
    // 4. Generate new storage key
    watermarkedKey := s.generateWatermarkedKey(originalKey, params.OrderID)
    
    // 5. Upload watermarked file
    _, err = s.storageService.Upload(
        ctx,
        watermarkedKey,
        bytes.NewReader(watermarkedFile),
        "application/pdf",
        int64(len(watermarkedFile)),
    )
    
    if err != nil {
        return "", fmt.Errorf("failed to upload watermarked file: %w", err)
    }
    
    log.Info("PDF watermarked successfully",
        "original_key", originalKey,
        "watermarked_key", watermarkedKey,
        "user_email", params.UserEmail,
    )
    
    return watermarkedKey, nil
}

func (s *WatermarkService) downloadFile(ctx context.Context, key string) ([]byte, error) {
    // Generate presigned URL for internal download
    url, err := s.storageService.GeneratePresignedURL(ctx, key, 10*time.Minute)
    if err != nil {
        return nil, err
    }
    
    // Download file
    resp, err := http.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    return io.ReadAll(resp.Body)
}

func (s *WatermarkService) applyWatermark(pdfData []byte, text string) ([]byte, error) {
    // Create watermark
    wm, err := pdfcpu.ParseTextWatermarkDetails(text, "font:Helvetica, points:10, color:0.7 0.7 0.7, opacity:0.5, pos:bc", false)
    if err != nil {
        return nil, err
    }
    
    // Read PDF
    inputReader := bytes.NewReader(pdfData)
    
    // Apply watermark
    var outputBuffer bytes.Buffer
    err = api.AddWatermarks(inputReader, &outputBuffer, nil, wm, nil)
    if err != nil {
        return nil, err
    }
    
    return outputBuffer.Bytes(), nil
}

func (s *WatermarkService) generateWatermarkedKey(originalKey string, orderID string) string {
    // ebooks/book123/pdf/original.pdf → ebooks/book123/pdf/order456_watermarked.pdf
    dir := filepath.Dir(originalKey)
    ext := filepath.Ext(originalKey)
    return fmt.Sprintf("%s/%s_watermarked%s", dir, orderID, ext)
}
```


#### 6.3 Asynq Job Handler

```go
const TypeWatermarkPDF = "ebook:watermark"

type WatermarkPDFPayload struct {
    OrderID  string `json:"order_id"`
    BookID   string `json:"book_id"`
    UserID   string `json:"user_id"`
}

func (h *EbookJobHandler) WatermarkPDF(ctx context.Context, task *asynq.Task) error {
    var payload WatermarkPDFPayload
    if err := json.Unmarshal(task.Payload(), &payload); err != nil {
        return fmt.Errorf("unmarshal error: %w", err)
    }
    
    log.Info("Processing PDF watermark job",
        "order_id", payload.OrderID,
        "book_id", payload.BookID,
    )
    
    // 1. Get order details
    order, err := h.orderRepo.FindByID(ctx, payload.OrderID)
    if err != nil {
        return fmt.Errorf("order not found: %w", err)
    }
    
    // 2. Get book details
    book, err := h.bookRepo.FindByID(ctx, payload.BookID)
    if err != nil {
        return fmt.Errorf("book not found: %w", err)
    }
    
    if book.EbookFileKey == nil {
        return fmt.Errorf("book has no ebook file")
    }
    
    // Only watermark PDFs
    if book.EbookFormat == nil || *book.EbookFormat != "pdf" {
        log.Info("Skipping watermark for non-PDF ebook", "format", *book.EbookFormat)
        return nil
    }
    
    // 3. Get user details
    user, err := h.userRepo.FindByID(ctx, payload.UserID)
    if err != nil {
        return fmt.Errorf("user not found: %w", err)
    }
    
    // 4. Apply watermark
    watermarkedKey, err := h.watermarkService.WatermarkPDF(ctx, *book.EbookFileKey, WatermarkParams{
        UserEmail:    user.Email,
        OrderID:      order.OrderNumber,
        PurchaseDate: order.CreatedAt,
    })
    
    if err != nil {
        return fmt.Errorf("watermark failed: %w", err)
    }
    
    // 5. Store watermarked key per user/order
    err = h.ebookRepo.CreateWatermarkedCopy(ctx, &WatermarkedCopy{
        UserID:         user.ID,
        BookID:         book.ID,
        OrderID:        order.ID,
        OriginalKey:    *book.EbookFileKey,
        WatermarkedKey: watermarkedKey,
    })
    
    if err != nil {
        return fmt.Errorf("failed to save watermarked copy: %w", err)
    }
    
    log.Info("PDF watermarked successfully",
        "order_id", payload.OrderID,
        "watermarked_key", watermarkedKey,
    )
    
    return nil
}
```


#### 6.4 Watermarked Copies Table

```sql
CREATE TABLE ebook_watermarked_copies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    book_id UUID NOT NULL REFERENCES books(id),
    order_id UUID NOT NULL REFERENCES orders(id),
    original_key TEXT NOT NULL,
    watermarked_key TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE (user_id, book_id, order_id)
);

CREATE INDEX idx_watermarked_user ON ebook_watermarked_copies(user_id);
CREATE INDEX idx_watermarked_order ON ebook_watermarked_copies(order_id);
```


#### 6.5 Trigger Watermark Job

Trong order service, sau khi order completed:

```go
// After order status = "completed"
for _, item := range order.Items {
    if item.Book.EbookFileKey != nil {
        // Enqueue watermark job
        task := asynq.NewTask(jobs.TypeWatermarkPDF, &WatermarkPDFPayload{
            OrderID: order.ID,
            BookID:  item.BookID,
            UserID:  order.UserID,
        })
        
        queueClient.Enqueue(task, queue.QueueHigh, asynq.MaxRetry(3))
    }
}
```


#### 6.6 Use Watermarked Copy for Download

Update `GenerateDownloadURL` để ưu tiên watermarked copy:

```go
func (s *EbookDownloadService) GenerateDownloadURL(...) {
    // ... existing code
    
    // Try to get watermarked copy first
    watermarkedCopy, err := s.ebookRepo.FindWatermarkedCopy(ctx, params.UserID, params.BookID)
    
    var fileKey string
    if err == nil && watermarkedCopy != nil {
        // Use watermarked version
        fileKey = watermarkedCopy.WatermarkedKey
    } else {
        // Fallback to original (for EPUB or if watermark not ready)
        fileKey = *book.EbookFileKey
    }
    
    // Generate presigned URL for fileKey
    presignedURL, err := s.storageService.GeneratePresignedURL(ctx, fileKey, expiration)
    // ...
}
```


### Acceptance Criteria

- PDF được watermark với user email + order ID[^1]
- Watermark visible nhưng không ảnh hưởng readability[^1]
- Job chạy async, không block order completion[^1]
- Watermarked copy được lưu riêng per user/order[^1]
- EPUB không bị watermark (skip)[^1]
- Download URL serve watermarked copy[^1]


### Dependencies

- P3-T009: S3/MinIO setup[^1]
- P3-T010: Upload ebook API[^1]
- P2-T008: Asynq setup[^1]


### Effort

3 ngày[^1]

***

## 7. List User's Purchased eBooks (P3-T015)

### Mô tả

API để user xem danh sách ebooks đã mua.[^1]

### API Endpoint

`GET /v1/user/ebooks`[^1]

### Query Parameters

- `?page=1&limit=20` - Pagination[^1]
- `?format=pdf` - Filter by format[^1]


### Response Format

```json
{
  "success": true,
  "data": {
    "ebooks": [
      {
        "book_id": "uuid",
        "title": "Nhà Giả Kim",
        "cover_url": "https://...",
        "format": "pdf",
        "size_mb": 2.5,
        "purchased_at": "2025-10-20T10:00:00Z",
        "order_id": "uuid",
        "order_number": "ORD-20251020-001",
        "can_download": true,
        "downloads_remaining_today": 4
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

#### 7.1 Service Implementation

```go
func (s *EbookService) ListUserEbooks(ctx context.Context, userID string, filters EbookFilters) (*EbookListResult, error) {
    query := `
        SELECT 
            b.id as book_id,
            b.title,
            b.cover_url,
            b.ebook_format,
            b.ebook_file_size_mb,
            o.id as order_id,
            o.order_number,
            o.created_at as purchased_at
        FROM orders o
        JOIN order_items oi ON o.id = oi.order_id
        JOIN books b ON oi.book_id = b.id
        WHERE o.user_id = $1
        AND o.status IN ('completed', 'delivered')
        AND o.payment_status = 'paid'
        AND b.ebook_file_key IS NOT NULL
    `
    
    args := []interface{}{userID}
    argPos := 2
    
    // Filter by format
    if filters.Format != nil {
        query += fmt.Sprintf(" AND b.ebook_format = $%d", argPos)
        args = append(args, *filters.Format)
        argPos++
    }
    
    query += " ORDER BY o.created_at DESC"
    
    // Pagination
    query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argPos, argPos+1)
    args = append(args, filters.Limit, (filters.Page-1)*filters.Limit)
    
    rows, err := s.db.QueryContext(ctx, query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    ebooks := []UserEbook{}
    for rows.Next() {
        var ebook UserEbook
        err := rows.Scan(
            &ebook.BookID,
            &ebook.Title,
            &ebook.CoverURL,
            &ebook.Format,
            &ebook.SizeMB,
            &ebook.OrderID,
            &ebook.OrderNumber,
            &ebook.PurchasedAt,
        )
        if err != nil {
            return nil, err
        }
        
        // Check if can download today
        limitInfo, _ := s.downloadService.GetDownloadLimitInfo(ctx, userID, ebook.BookID)
        if limitInfo != nil {
            ebook.CanDownload = limitInfo.CanDownload
            ebook.DownloadsRemainingToday = limitInfo.Remaining
        }
        
        ebooks = append(ebooks, ebook)
    }
    
    // Count total
    countQuery := `
        SELECT COUNT(DISTINCT b.id)
        FROM orders o
        JOIN order_items oi ON o.id = oi.order_id
        JOIN books b ON oi.book_id = b.id
        WHERE o.user_id = $1
        AND o.status IN ('completed', 'delivered')
        AND o.payment_status = 'paid'
        AND b.ebook_file_key IS NOT NULL
    `
    
    var total int
    s.db.QueryRowContext(ctx, countQuery, userID).Scan(&total)
    
    return &EbookListResult{
        Ebooks: ebooks,
        Total:  total,
    }, nil
}
```


#### 7.2 Handler Implementation

```go
func (h *EbookHandler) ListMyEbooks(c *gin.Context) {
    userID := c.GetString("user_id")
    
    // Parse filters
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
    format := c.Query("format")
    
    var formatPtr *string
    if format != "" {
        formatPtr = &format
    }
    
    result, err := h.ebookService.ListUserEbooks(c.Request.Context(), userID, EbookFilters{
        Page:   page,
        Limit:  limit,
        Format: formatPtr,
    })
    
    if err != nil {
        c.JSON(500, gin.H{"success": false, "error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{
        "success": true,
        "data": gin.H{
            "ebooks": result.Ebooks,
        },
        "meta": gin.H{
            "page":  page,
            "limit": limit,
            "total": result.Total,
        },
    })
}
```


### Acceptance Criteria

- User xem được list ebooks đã mua[^1]
- Hiển thị download status (can_download, remaining)[^1]
- Pagination hoạt động[^1]
- Filter by format (PDF/EPUB)[^1]
- Sort by purchase date (newest first)[^1]


### Dependencies

- P3-T011: Generate presigned URL[^1]
- P3-T013: Download limit tracking[^1]


### Effort

1 ngày[^1]

***

## 8. eBook Format Support (PDF, EPUB) (P3-T016)

### Mô tả

Đảm bảo system support cả PDF và EPUB formats với handling khác nhau.[^1]

### Format Differences

| Feature | PDF | EPUB |
| :-- | :-- | :-- |
| Watermark | ✅ Yes (pdfcpu) | ❌ No (complex) |
| Max Size | 50MB | 50MB |
| MIME Type | application/pdf | application/epub+zip |
| Preview | Possible | Limited |

### Công việc cụ thể

#### 8.1 Format Detection

```go
func DetectEbookFormat(filename string, fileData []byte) (string, error) {
    ext := strings.ToLower(filepath.Ext(filename))
    
    switch ext {
    case ".pdf":
        // Validate PDF magic bytes
        if !bytes.HasPrefix(fileData, []byte("%PDF")) {
            return "", fmt.Errorf("invalid PDF file")
        }
        return "pdf", nil
        
    case ".epub":
        // Validate EPUB (it's a ZIP file)
        if !bytes.HasPrefix(fileData, []byte("PK")) {
            return "", fmt.Errorf("invalid EPUB file")
        }
        // Further validation: check for mimetype file inside
        return "epub", nil
        
    default:
        return "", fmt.Errorf("unsupported format: %s", ext)
    }
}
```


#### 8.2 EPUB Validation

```go
func ValidateEPUB(file io.Reader) error {
    // EPUB is a ZIP archive
    zipReader, err := zip.NewReader(file, size)
    if err != nil {
        return fmt.Errorf("invalid EPUB: not a valid ZIP")
    }
    
    // Check for required files
    requiredFiles := []string{"mimetype", "META-INF/container.xml"}
    found := make(map[string]bool)
    
    for _, f := range zipReader.File {
        for _, req := range requiredFiles {
            if f.Name == req {
                found[req] = true
            }
        }
    }
    
    for _, req := range requiredFiles {
        if !found[req] {
            return fmt.Errorf("invalid EPUB: missing %s", req)
        }
    }
    
    return nil
}
```


#### 8.3 Format-Specific Processing

```go
func (s *EbookUploadService) processEbook(
    ctx context.Context,
    file *multipart.FileHeader,
    format string,
) error {
    
    switch format {
    case "pdf":
        // Additional PDF validation
        return s.validatePDF(file)
        
    case "epub":
        // Additional EPUB validation
        return s.validateEPUB(file)
        
    default:
        return fmt.Errorf("unsupported format: %s", format)
    }
}
```


#### 8.4 Download Handling Per Format

```go
func (s *EbookDownloadService) GenerateDownloadURL(...) {
    // ... existing code
    
    // For PDF: prefer watermarked version
    if *book.EbookFormat == "pdf" {
        watermarkedCopy, err := s.ebookRepo.FindWatermarkedCopy(ctx, params.UserID, params.BookID)
        if err == nil && watermarkedCopy != nil {
            fileKey = watermarkedCopy.WatermarkedKey
        }
    }
    
    // For EPUB: always use original (no watermark)
    if *book.EbookFormat == "epub" {
        fileKey = *book.EbookFileKey
    }
    
    // Generate presigned URL...
}
```


#### 8.5 Admin: Format Statistics

`GET /v1/admin/ebooks/stats`

```go
func (s *AdminEbookService) GetFormatStats(ctx context.Context) (*FormatStats, error) {
    query := `
        SELECT 
            ebook_format,
            COUNT(*) as count,
            SUM(ebook_file_size_mb) as total_size_mb,
            AVG(ebook_file_size_mb) as avg_size_mb
        FROM books
        WHERE ebook_file_key IS NOT NULL
        AND ebook_format IS NOT NULL
        GROUP BY ebook_format
    `
    
    rows, err := s.db.QueryContext(ctx, query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    stats := &FormatStats{
        Formats: make(map[string]FormatStat),
    }
    
    for rows.Next() {
        var format string
        var stat FormatStat
        
        rows.Scan(&format, &stat.Count, &stat.TotalSizeMB, &stat.AvgSizeMB)
        stats.Formats[format] = stat
    }
    
    return stats, nil
}
```

**Response**:

```json
{
  "success": true,
  "data": {
    "formats": {
      "pdf": {
        "count": 850,
        "total_size_mb": 2125.5,
        "avg_size_mb": 2.5
      },
      "epub": {
        "count": 150,
        "total_size_mb": 300.0,
        "avg_size_mb": 2.0
      }
    }
  }
}
```


### Acceptance Criteria

- System support cả PDF và EPUB[^1]
- Format detection chính xác[^1]
- Validation cho từng format[^1]
- PDF được watermark, EPUB không[^1]
- Admin xem được statistics per format[^1]


### Dependencies

- P3-T010: Upload ebook API[^1]
- P3-T014: Watermark PDF job[^1]


### Effort

1 ngày[^1]

***

## SUMMARY

### Total Effort Sprint 17-18

| Task ID | Task | Effort (days) |
| :-- | :-- | :-- |
| P3-T009 | S3/MinIO setup for ebook storage | 1 |
| P3-T010 | Upload ebook file API (Admin) | 2 |
| P3-T011 | Generate presigned download URL | 2 |
| P3-T012 | Download link API với validation | 1 |
| P3-T013 | Download limit tracking (5/day) | 1 |
| P3-T014 | Watermark PDF job (Asynq) | 3 |
| P3-T015 | List user's purchased ebooks | 1 |
| P3-T016 | ebook format support (PDF, EPUB) | 1 |
| **TOTAL** |  | **12 days** |

**Sprint duration**: 2 tuần (10 ngày làm việc)[^1]
**Team size**: 2 backend developers (có thể song song hóa tasks)[^1]

### Parallelization Strategy

**Week 1** (5 ngày):

- **Dev 1**: P3-T009 → P3-T010 → P3-T011 (Storage \& Upload \& Presigned URL) (1+2+2 = 5 days)[^1]
- **Dev 2**: P3-T016 (Format support) (1 day) → P3-T014 (Watermark job - preparation) (3 days) + Code review (1 day)[^1]

**Week 2** (5 ngày):

- **Dev 1**: P3-T012 → P3-T013 → P3-T015 (Download APIs \& Tracking \& List) (1+1+1 = 3 days) + Integration testing (2 days)[^1]
- **Dev 2**: P3-T014 (Watermark job - finalization) (1 day) + Integration testing + Bug fixes (4 days)[^1]


### Deliverables Checklist Sprint 17-18

- ✅ **S3/MinIO storage** fully configured và operational[^1]
- ✅ **Admin upload** ebooks (PDF, EPUB) với validation[^1]
- ✅ **Presigned download URLs** với expiration (1 hour)[^1]
- ✅ **Download limit** 5 per day per book enforced[^1]
- ✅ **PDF watermarking** với user info (email, order ID)[^1]
- ✅ **User library** hiển thị purchased ebooks[^1]
- ✅ **Multi-format support** (PDF với watermark, EPUB original)[^1]
- ✅ **Download tracking** và audit logs[^1]


### Key Technical Achievements

**Security**:

- Presigned URLs với short expiration (1 hour)[^1]
- Purchase validation before download[^1]
- Watermarking để trace piracy[^1]
- Download limit prevents abuse[^1]
- Encryption at rest (S3 AES-256)[^1]

**Storage**:

- S3-compatible interface (swap S3 ↔ MinIO)[^1]
- Efficient storage with presigned URLs[^1]
- Per-user watermarked copies[^1]
- Max 50MB file size[^1]

**User Experience**:

- Unlimited downloads within limit window[^1]
- Download history tracking[^1]
- Format-specific handling (PDF, EPUB)[^1]
- Clear remaining downloads display[^1]


### Database Changes Summary

- ✅ `books` table updated với ebook fields[^1]
- ✅ `ebook_downloads` table cho tracking[^1]
- ✅ `ebook_watermarked_copies` table cho per-user copies[^1]


### Environment Variables Added

```bash
# Storage
STORAGE_TYPE=s3  # or "minio"
AWS_REGION=ap-southeast-1
S3_BUCKET=bookstore-ebooks
AWS_ACCESS_KEY_ID=xxx
AWS_SECRET_ACCESS_KEY=xxx

# MinIO (if using)
MINIO_ENDPOINT=localhost:9000
MINIO_BUCKET=ebooks
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_USE_SSL=false

# eBook Settings
EBOOK_MAX_SIZE_MB=50
EBOOK_DOWNLOAD_LIMIT_PER_DAY=5
EBOOK_PRESIGNED_URL_EXPIRATION=1h
```


### Next Steps (Phase 4)

Phase 4 sẽ focus vào **Production-Ready** features: Security Hardening, Monitoring/Observability, Performance Optimization.[^1]

<div align="center">⁂</div>

[^1]: USER-REQUIREMENTS-DOCUMENT-URD-PHIEN-BAN-HOA.docx

