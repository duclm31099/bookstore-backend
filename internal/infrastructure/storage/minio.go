package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"bookstore-backend/internal/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOStorage handles file uploads to MinIO
type MinIOStorage struct {
	client *minio.Client
	bucket string
}

// NewMinIOStorage khởi tạo MinIO client
func NewMinIOStorage(cfg config.MinIOConfig) (*MinIOStorage, error) {
	// Tạo MinIO client với credentials
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL, // false cho local, true cho production
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	// Kiểm tra bucket có tồn tại không, nếu không thì tạo mới
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket: %w", err)
	}

	if !exists {
		// Tạo bucket mới với quyền truy cập public-read
		err = client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return &MinIOStorage{
		client: client,
		bucket: cfg.Bucket,
	}, nil
}

// Upload uploads a file to MinIO
// key: đường dẫn file trong bucket (vd: books/uuid/0_original.jpg)
// data: nội dung file dưới dạng bytes
// contentType: loại file (image/jpeg, image/png...)
func (s *MinIOStorage) Upload(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	reader := bytes.NewReader(data)

	// Upload file lên MinIO
	_, err := s.client.PutObject(
		ctx,
		s.bucket,
		key,
		reader,
		int64(len(data)),
		minio.PutObjectOptions{
			ContentType: contentType,
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to upload to minio: %w", err)
	}

	// Tạo URL truy cập file
	// Format: http://localhost:9000/bookstore/books/uuid/0_original.jpg
	url := fmt.Sprintf("http://%s/%s/%s", s.client.EndpointURL().Host, s.bucket, key)

	return url, nil
}

// Download downloads a file from MinIO
func (s *MinIOStorage) Download(ctx context.Context, key string) ([]byte, error) {
	// Lấy object từ MinIO
	object, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	defer object.Close()

	// Đọc toàn bộ nội dung file vào memory
	data, err := io.ReadAll(object)
	if err != nil {
		return nil, fmt.Errorf("failed to read object: %w", err)
	}

	return data, nil
}

// Delete xóa một file khỏi MinIO
func (s *MinIOStorage) Delete(ctx context.Context, key string) error {
	err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}

// DeleteByPrefix xóa tất cả files có prefix (vd: books/uuid/)
// Dùng khi xóa book, xóa hết ảnh của book đó
func (s *MinIOStorage) DeleteByPrefix(ctx context.Context, prefix string) error {
	// List tất cả objects có prefix
	objectsCh := s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	// Xóa từng object
	for object := range objectsCh {
		if object.Err != nil {
			return fmt.Errorf("error listing objects: %w", object.Err)
		}

		err := s.client.RemoveObject(ctx, s.bucket, object.Key, minio.RemoveObjectOptions{})
		if err != nil {
			return fmt.Errorf("failed to delete object %s: %w", object.Key, err)
		}
	}

	return nil
}

// MoveObject di chuyển object từ key này sang key khác
// Sử dụng CopyObject + RemoveObject
func (s *MinIOStorage) MoveObject(ctx context.Context, fromKey, toKey string) error {
	// 1. Copy object to new location
	srcOpts := minio.CopySrcOptions{
		Bucket: s.bucket,
		Object: fromKey,
	}

	dstOpts := minio.CopyDestOptions{
		Bucket: s.bucket,
		Object: toKey,
	}

	_, err := s.client.CopyObject(ctx, dstOpts, srcOpts)
	if err != nil {
		return fmt.Errorf("failed to copy object: %w", err)
	}

	// 2. Remove old object
	err = s.client.RemoveObject(ctx, s.bucket, fromKey, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to remove old object: %w", err)
	}

	return nil
}

// RemoveObjects xóa nhiều objects cùng lúc (for cleanup)
func (s *MinIOStorage) RemoveObjects(ctx context.Context, keys []string) error {
	objectsCh := make(chan minio.ObjectInfo, len(keys))

	// Send object keys to channel
	go func() {
		defer close(objectsCh)
		for _, key := range keys {
			objectsCh <- minio.ObjectInfo{Key: key}
		}
	}()

	// Remove objects
	errorCh := s.client.RemoveObjects(ctx, s.bucket, objectsCh, minio.RemoveObjectsOptions{})

	// Check for errors
	for rmErr := range errorCh {
		if rmErr.Err != nil {
			return fmt.Errorf("failed to remove %s: %w", rmErr.ObjectName, rmErr.Err)
		}
	}

	return nil
}

// RemoveFolder xóa tất cả objects trong một folder (prefix)
func (s *MinIOStorage) RemoveFolder(ctx context.Context, prefix string) error {
	// List all objects with prefix
	objectsCh := s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	// Collect object keys
	var keys []string
	for object := range objectsCh {
		if object.Err != nil {
			return fmt.Errorf("failed to list objects: %w", object.Err)
		}
		keys = append(keys, object.Key)
	}

	// Remove all objects
	if len(keys) > 0 {
		return s.RemoveObjects(ctx, keys)
	}

	return nil
}
