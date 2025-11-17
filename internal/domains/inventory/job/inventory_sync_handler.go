package job

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"

	repo "bookstore-backend/internal/domains/inventory/repository"
	"bookstore-backend/internal/shared"
	"bookstore-backend/pkg/cache"
	"bookstore-backend/pkg/logger"
)

// InventorySyncHandler xử lý job đồng bộ tổng tồn của book ra Redis.
type InventorySyncHandler struct {
	repo  repo.RepositoryInterface
	cache cache.Cache
}

// NewInventorySyncHandler tạo handler mới với dependency từ container.
func NewInventorySyncHandler(
	repo repo.RepositoryInterface,
	cache cache.Cache,
) *InventorySyncHandler {
	return &InventorySyncHandler{
		repo:  repo,
		cache: cache,
	}
}

// bookStockCacheDTO là cấu trúc JSON lưu trong Redis.
type bookStockCacheDTO struct {
	BookID              string    `json:"book_id"`
	TotalQuantity       int       `json:"total_quantity"`
	TotalReserved       int       `json:"total_reserved"`
	Available           int       `json:"available"`
	WarehouseCount      int       `json:"warehouse_count"`
	WarehousesWithStock []string  `json:"warehouses_with_stock"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// ProcessTask xử lý background job sync tồn kho.
// 1. Parse payload.
// 2. Đọc tổng tồn từ view books_total_stock.
// 3. Ghi JSON vào Redis key inventory:book:{book_id}:total (không TTL).
func (h *InventorySyncHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	// 1. Parse payload
	var payload shared.InventorySyncPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		logger.Error("InventorySync: Failed to unmarshal payload", err)
		// Payload hỏng → không retry, vì retry cũng không sửa dữ liệu
		return fmt.Errorf("unmarshal InventorySync payload: %w", err)
	}

	if payload.BookID == "" {
		err := fmt.Errorf("InventorySync: empty book_id in payload")
		logger.Error("InventorySync: Invalid payload", err)
		// Payload invalid → không có ý nghĩa retry
		return err
	}

	// 2. Đọc total stock từ repository (view)
	stock, err := h.repo.GetBookTotalStock(ctx, payload.BookID)
	if err != nil {
		// Lỗi DB (network, query, v.v.) → cho phép retry
		logger.Error("InventorySync: GetBookTotalStock failed", err)
		return err
	}

	var cacheDTO bookStockCacheDTO

	if stock == nil {
		// Không có row trong view → coi như stock = 0
		cacheDTO = bookStockCacheDTO{
			BookID:              payload.BookID,
			TotalQuantity:       0,
			TotalReserved:       0,
			Available:           0,
			WarehouseCount:      0,
			WarehousesWithStock: []string{},
			UpdatedAt:           time.Now().UTC(),
		}
	} else {
		cacheDTO = bookStockCacheDTO{
			BookID:              stock.BookID,
			TotalQuantity:       stock.TotalQuantity,
			TotalReserved:       stock.TotalReserved,
			Available:           stock.Available,
			WarehouseCount:      stock.WarehouseCount,
			WarehousesWithStock: stock.WarehousesWithStock,
			UpdatedAt:           time.Now().UTC(),
		}
	}

	// 3. Ghi vào Redis cache
	key := fmt.Sprintf("inventory:book:%s:total", payload.BookID)

	// TTL = 0 → không hết hạn, rely 100% vào event.
	if err := h.cache.Set(ctx, key, cacheDTO, 0); err != nil {
		// Redis Set trong implement hiện tại swallow error (log nội bộ và return nil),
		// nhưng vẫn log lại ở đây để dễ trace.
		logger.Error("InventorySync: failed to set cache", err)
		// Dù Redis lỗi, không nên fail job vì DB vẫn là source of truth.
		// Tuy nhiên, nếu bạn muốn retry khi Redis lỗi thực sự, cần sửa RedisCache.Set để trả error.
		// Với implement hiện tại, err hầu như sẽ là nil.
	}

	// Logging info để quan sát
	logger.Info("InventorySync: cache updated", map[string]interface{}{
		"book_id":     payload.BookID,
		"available":   cacheDTO.Available,
		"total":       cacheDTO.TotalQuantity,
		"warehouses":  cacheDTO.WarehouseCount,
		"source":      payload.Source,
		"correlation": payload.CorrelationID,
	})

	return nil
}
