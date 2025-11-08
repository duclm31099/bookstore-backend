package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"bookstore-backend/internal/domains/order/model"
)

// =====================================================
// POSTGRES REPOSITORY IMPLEMENTATION
// =====================================================
type postgresOrderRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresOrderRepository creates a new postgres order repository
func NewPostgresOrderRepository(pool *pgxpool.Pool) OrderRepository {
	return &postgresOrderRepository{
		pool: pool,
	}
}

// =====================================================
// TRANSACTION MANAGEMENT
// =====================================================

func (r *postgresOrderRepository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.pool.Begin(ctx)
}

func (r *postgresOrderRepository) CommitTx(ctx context.Context, tx pgx.Tx) error {
	return tx.Commit(ctx)
}

func (r *postgresOrderRepository) RollbackTx(ctx context.Context, tx pgx.Tx) error {
	return tx.Rollback(ctx)
}

// =====================================================
// CREATE ORDER
// =====================================================

func (r *postgresOrderRepository) CreateOrder(ctx context.Context, order *model.Order) error {
	query := `
		INSERT INTO orders (
			id, user_id, address_id, promotion_id, warehouse_id,
			subtotal, shipping_fee, cod_fee, discount_amount, tax_amount, total,
			payment_method, payment_status, status, customer_note, version
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16
		)
		RETURNING order_number, created_at, updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		order.ID,
		order.UserID,
		order.AddressID,
		order.PromotionID,
		order.WarehouseID,
		order.Subtotal,
		order.ShippingFee,
		order.CODFee,
		order.DiscountAmount,
		order.TaxAmount,
		order.Total,
		order.PaymentMethod,
		order.PaymentStatus,
		order.Status,
		order.CustomerNote,
		order.Version,
	).Scan(&order.OrderNumber, &order.CreatedAt, &order.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}

	return nil
}

func (r *postgresOrderRepository) CreateOrderWithTx(ctx context.Context, tx pgx.Tx, order *model.Order) error {
	query := `
		INSERT INTO orders (
			id, user_id, address_id, promotion_id, warehouse_id,
			subtotal, shipping_fee, cod_fee, discount_amount, tax_amount, total,
			payment_method, payment_status, status, customer_note, version
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16
		)
		RETURNING order_number, created_at, updated_at
	`

	err := tx.QueryRow(ctx, query,
		order.ID,
		order.UserID,
		order.AddressID,
		order.PromotionID,
		order.WarehouseID,
		order.Subtotal,
		order.ShippingFee,
		order.CODFee,
		order.DiscountAmount,
		order.TaxAmount,
		order.Total,
		order.PaymentMethod,
		order.PaymentStatus,
		order.Status,
		order.CustomerNote,
		order.Version,
	).Scan(&order.OrderNumber, &order.CreatedAt, &order.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create order with tx: %w", err)
	}

	return nil
}

// =====================================================
// GET ORDER
// =====================================================

func (r *postgresOrderRepository) GetOrderByID(ctx context.Context, orderID uuid.UUID) (*model.Order, error) {
	query := `
		SELECT 
			id, order_number, user_id, address_id, promotion_id, warehouse_id,
			subtotal, shipping_fee, cod_fee, discount_amount, tax_amount, total,
			payment_method, payment_status, payment_details, paid_at,
			status, tracking_number, estimated_delivery_at, delivered_at,
			customer_note, admin_note, cancellation_reason,
			created_at, updated_at, cancelled_at, version
		FROM orders
		WHERE id = $1
	`

	var order model.Order
	err := r.pool.QueryRow(ctx, query, orderID).Scan(
		&order.ID,
		&order.OrderNumber,
		&order.UserID,
		&order.AddressID,
		&order.PromotionID,
		&order.WarehouseID,
		&order.Subtotal,
		&order.ShippingFee,
		&order.CODFee,
		&order.DiscountAmount,
		&order.TaxAmount,
		&order.Total,
		&order.PaymentMethod,
		&order.PaymentStatus,
		&order.PaymentDetails,
		&order.PaidAt,
		&order.Status,
		&order.TrackingNumber,
		&order.EstimatedDeliveryAt,
		&order.DeliveredAt,
		&order.CustomerNote,
		&order.AdminNote,
		&order.CancellationReason,
		&order.CreatedAt,
		&order.UpdatedAt,
		&order.CancelledAt,
		&order.Version,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrOrderNotFound
		}
		return nil, fmt.Errorf("failed to get order by id: %w", err)
	}

	return &order, nil
}

func (r *postgresOrderRepository) GetOrderByIDAndUserID(ctx context.Context, orderID, userID uuid.UUID) (*model.Order, error) {
	query := `
		SELECT 
			id, order_number, user_id, address_id, promotion_id, warehouse_id,
			subtotal, shipping_fee, cod_fee, discount_amount, tax_amount, total,
			payment_method, payment_status, payment_details, paid_at,
			status, tracking_number, estimated_delivery_at, delivered_at,
			customer_note, admin_note, cancellation_reason,
			created_at, updated_at, cancelled_at, version
		FROM orders
		WHERE id = $1 AND user_id = $2
	`

	var order model.Order
	err := r.pool.QueryRow(ctx, query, orderID, userID).Scan(
		&order.ID,
		&order.OrderNumber,
		&order.UserID,
		&order.AddressID,
		&order.PromotionID,
		&order.WarehouseID,
		&order.Subtotal,
		&order.ShippingFee,
		&order.CODFee,
		&order.DiscountAmount,
		&order.TaxAmount,
		&order.Total,
		&order.PaymentMethod,
		&order.PaymentStatus,
		&order.PaymentDetails,
		&order.PaidAt,
		&order.Status,
		&order.TrackingNumber,
		&order.EstimatedDeliveryAt,
		&order.DeliveredAt,
		&order.CustomerNote,
		&order.AdminNote,
		&order.CancellationReason,
		&order.CreatedAt,
		&order.UpdatedAt,
		&order.CancelledAt,
		&order.Version,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrOrderNotFound
		}
		return nil, fmt.Errorf("failed to get order by id and user id: %w", err)
	}

	return &order, nil
}

func (r *postgresOrderRepository) GetOrderByNumber(ctx context.Context, orderNumber string) (*model.Order, error) {
	query := `
		SELECT 
			id, order_number, user_id, address_id, promotion_id, warehouse_id,
			subtotal, shipping_fee, cod_fee, discount_amount, tax_amount, total,
			payment_method, payment_status, payment_details, paid_at,
			status, tracking_number, estimated_delivery_at, delivered_at,
			customer_note, admin_note, cancellation_reason,
			created_at, updated_at, cancelled_at, version
		FROM orders
		WHERE order_number = $1
	`

	var order model.Order
	err := r.pool.QueryRow(ctx, query, orderNumber).Scan(
		&order.ID,
		&order.OrderNumber,
		&order.UserID,
		&order.AddressID,
		&order.PromotionID,
		&order.WarehouseID,
		&order.Subtotal,
		&order.ShippingFee,
		&order.CODFee,
		&order.DiscountAmount,
		&order.TaxAmount,
		&order.Total,
		&order.PaymentMethod,
		&order.PaymentStatus,
		&order.PaymentDetails,
		&order.PaidAt,
		&order.Status,
		&order.TrackingNumber,
		&order.EstimatedDeliveryAt,
		&order.DeliveredAt,
		&order.CustomerNote,
		&order.AdminNote,
		&order.CancellationReason,
		&order.CreatedAt,
		&order.UpdatedAt,
		&order.CancelledAt,
		&order.Version,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrOrderNotFound
		}
		return nil, fmt.Errorf("failed to get order by number: %w", err)
	}

	return &order, nil
}

// =====================================================
// UPDATE ORDER
// =====================================================

func (r *postgresOrderRepository) UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, status string, version int) error {
	query := `
		UPDATE orders
		SET status = $1, version = version + 1, updated_at = NOW()
		WHERE id = $2 AND version = $3
	`

	result, err := r.pool.Exec(ctx, query, status, orderID, version)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	// Check if row was actually updated (version mismatch = concurrent modification)
	if result.RowsAffected() == 0 {
		return model.ErrVersionMismatch
	}

	return nil
}

func (r *postgresOrderRepository) UpdateOrderStatusWithTx(ctx context.Context, tx pgx.Tx, orderID uuid.UUID, status string, version int) error {
	query := `
		UPDATE orders
		SET status = $1, version = version + 1, updated_at = NOW()
		WHERE id = $2 AND version = $3
	`

	result, err := tx.Exec(ctx, query, status, orderID, version)
	if err != nil {
		return fmt.Errorf("failed to update order status with tx: %w", err)
	}

	if result.RowsAffected() == 0 {
		return model.ErrVersionMismatch
	}

	return nil
}

func (r *postgresOrderRepository) CancelOrder(ctx context.Context, orderID uuid.UUID, reason string, version int) error {
	query := `
		UPDATE orders
		SET status = $1, 
			cancellation_reason = $2,
			cancelled_at = NOW(),
			version = version + 1,
			updated_at = NOW()
		WHERE id = $3 AND version = $4
	`

	result, err := r.pool.Exec(ctx, query, model.OrderStatusCancelled, reason, orderID, version)
	if err != nil {
		return fmt.Errorf("failed to cancel order: %w", err)
	}

	if result.RowsAffected() == 0 {
		return model.ErrVersionMismatch
	}

	return nil
}

func (r *postgresOrderRepository) UpdateOrderTracking(ctx context.Context, orderID uuid.UUID, trackingNumber string, version int) error {
	query := `
		UPDATE orders
		SET tracking_number = $1,
			version = version + 1,
			updated_at = NOW()
		WHERE id = $2 AND version = $3
	`

	result, err := r.pool.Exec(ctx, query, trackingNumber, orderID, version)
	if err != nil {
		return fmt.Errorf("failed to update order tracking: %w", err)
	}

	if result.RowsAffected() == 0 {
		return model.ErrVersionMismatch
	}

	return nil
}

func (r *postgresOrderRepository) UpdateOrderAdminNote(ctx context.Context, orderID uuid.UUID, adminNote string, version int) error {
	query := `
		UPDATE orders
		SET admin_note = $1,
			version = version + 1,
			updated_at = NOW()
		WHERE id = $2 AND version = $3
	`

	result, err := r.pool.Exec(ctx, query, adminNote, orderID, version)
	if err != nil {
		return fmt.Errorf("failed to update order admin note: %w", err)
	}

	if result.RowsAffected() == 0 {
		return model.ErrVersionMismatch
	}

	return nil
}

// =====================================================
// ORDER ITEMS
// =====================================================

func (r *postgresOrderRepository) CreateOrderItems(ctx context.Context, items []model.OrderItem) error {
	// Batch insert using pgx CopyFrom for better performance
	copyCount, err := r.pool.CopyFrom(
		ctx,
		pgx.Identifier{"order_items"},
		[]string{"id", "order_id", "book_id", "book_title", "book_slug", "book_cover_url", "author_name", "quantity", "price", "subtotal"},
		pgx.CopyFromSlice(len(items), func(i int) ([]interface{}, error) {
			return []interface{}{
				items[i].ID,
				items[i].OrderID,
				items[i].BookID,
				items[i].BookTitle,
				items[i].BookSlug,
				items[i].BookCoverURL,
				items[i].AuthorName,
				items[i].Quantity,
				items[i].Price,
				items[i].Subtotal,
			}, nil
		}),
	)

	if err != nil {
		return fmt.Errorf("failed to create order items: %w", err)
	}

	if copyCount != int64(len(items)) {
		return fmt.Errorf("expected to insert %d items, but inserted %d", len(items), copyCount)
	}

	return nil
}

func (r *postgresOrderRepository) CreateOrderItemsWithTx(ctx context.Context, tx pgx.Tx, items []model.OrderItem) error {
	// For transaction, use batch insert instead of CopyFrom
	batch := &pgx.Batch{}
	query := `
		INSERT INTO order_items (
			id, order_id, book_id, book_title, book_slug, 
			book_cover_url, author_name, quantity, price, subtotal
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	for _, item := range items {
		batch.Queue(query,
			item.ID,
			item.OrderID,
			item.BookID,
			item.BookTitle,
			item.BookSlug,
			item.BookCoverURL,
			item.AuthorName,
			item.Quantity,
			item.Price,
			item.Subtotal,
		)
	}

	results := tx.SendBatch(ctx, batch)
	defer results.Close()

	// Check all inserts succeeded
	for i := 0; i < len(items); i++ {
		_, err := results.Exec()
		if err != nil {
			return fmt.Errorf("failed to create order item %d: %w", i, err)
		}
	}

	return nil
}

func (r *postgresOrderRepository) GetOrderItemsByOrderID(ctx context.Context, orderID uuid.UUID) ([]model.OrderItem, error) {
	query := `
		SELECT 
			id, order_id, book_id, book_title, book_slug,
			book_cover_url, author_name, quantity, price, subtotal, created_at
		FROM order_items
		WHERE order_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.pool.Query(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order items: %w", err)
	}
	defer rows.Close()

	var items []model.OrderItem
	for rows.Next() {
		var item model.OrderItem
		err := rows.Scan(
			&item.ID,
			&item.OrderID,
			&item.BookID,
			&item.BookTitle,
			&item.BookSlug,
			&item.BookCoverURL,
			&item.AuthorName,
			&item.Quantity,
			&item.Price,
			&item.Subtotal,
			&item.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order item: %w", err)
		}
		items = append(items, item)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("error iterating order items: %w", rows.Err())
	}

	return items, nil
}

func (r *postgresOrderRepository) CountOrderItemsByOrderID(ctx context.Context, orderID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM order_items WHERE order_id = $1`

	var count int
	err := r.pool.QueryRow(ctx, query, orderID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count order items: %w", err)
	}

	return count, nil
}

// =====================================================
// LIST ORDERS
// =====================================================

func (r *postgresOrderRepository) ListOrdersByUserID(ctx context.Context, userID uuid.UUID, status string, page, limit int) ([]model.Order, int, error) {
	offset := (page - 1) * limit

	// Build query with optional status filter
	queryBuilder := `
		SELECT 
			id, order_number, user_id, address_id, promotion_id, warehouse_id,
			subtotal, shipping_fee, cod_fee, discount_amount, tax_amount, total,
			payment_method, payment_status, paid_at,
			status, tracking_number, estimated_delivery_at, delivered_at,
			customer_note, cancellation_reason,
			created_at, updated_at, cancelled_at, version
		FROM orders
		WHERE user_id = $1
	`

	countQuery := `SELECT COUNT(*) FROM orders WHERE user_id = $1`
	args := []interface{}{userID}
	countArgs := []interface{}{userID}

	// Add status filter if provided
	if status != "" {
		queryBuilder += ` AND status = $2`
		countQuery += ` AND status = $2`
		args = append(args, status)
		countArgs = append(countArgs, status)
	}

	queryBuilder += ` ORDER BY created_at DESC LIMIT $` + fmt.Sprintf("%d", len(args)+1) + ` OFFSET $` + fmt.Sprintf("%d", len(args)+2)
	args = append(args, limit, offset)

	// Get total count
	var total int
	err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count orders: %w", err)
	}

	// Get orders
	rows, err := r.pool.Query(ctx, queryBuilder, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list orders: %w", err)
	}
	defer rows.Close()

	var orders []model.Order
	for rows.Next() {
		var order model.Order
		err := rows.Scan(
			&order.ID,
			&order.OrderNumber,
			&order.UserID,
			&order.AddressID,
			&order.PromotionID,
			&order.WarehouseID,
			&order.Subtotal,
			&order.ShippingFee,
			&order.CODFee,
			&order.DiscountAmount,
			&order.TaxAmount,
			&order.Total,
			&order.PaymentMethod,
			&order.PaymentStatus,
			&order.PaidAt,
			&order.Status,
			&order.TrackingNumber,
			&order.EstimatedDeliveryAt,
			&order.DeliveredAt,
			&order.CustomerNote,
			&order.CancellationReason,
			&order.CreatedAt,
			&order.UpdatedAt,
			&order.CancelledAt,
			&order.Version,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan order: %w", err)
		}
		orders = append(orders, order)
	}

	if rows.Err() != nil {
		return nil, 0, fmt.Errorf("error iterating orders: %w", rows.Err())
	}

	return orders, total, nil
}

func (r *postgresOrderRepository) ListAllOrders(ctx context.Context, status string, page, limit int) ([]model.Order, int, error) {
	offset := (page - 1) * limit

	queryBuilder := `
		SELECT 
			id, order_number, user_id, address_id, promotion_id, warehouse_id,
			subtotal, shipping_fee, cod_fee, discount_amount, tax_amount, total,
			payment_method, payment_status, paid_at,
			status, tracking_number, estimated_delivery_at, delivered_at,
			customer_note, admin_note, cancellation_reason,
			created_at, updated_at, cancelled_at, version
		FROM orders
		WHERE 1=1
	`

	countQuery := `SELECT COUNT(*) FROM orders WHERE 1=1`
	args := []interface{}{}
	countArgs := []interface{}{}

	// Add status filter if provided
	if status != "" {
		queryBuilder += ` AND status = $1`
		countQuery += ` AND status = $1`
		args = append(args, status)
		countArgs = append(countArgs, status)
	}

	queryBuilder += ` ORDER BY created_at DESC LIMIT $` + fmt.Sprintf("%d", len(args)+1) + ` OFFSET $` + fmt.Sprintf("%d", len(args)+2)
	args = append(args, limit, offset)

	// Get total count
	var total int
	err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count all orders: %w", err)
	}

	// Get orders
	rows, err := r.pool.Query(ctx, queryBuilder, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list all orders: %w", err)
	}
	defer rows.Close()

	var orders []model.Order
	for rows.Next() {
		var order model.Order
		err := rows.Scan(
			&order.ID,
			&order.OrderNumber,
			&order.UserID,
			&order.AddressID,
			&order.PromotionID,
			&order.WarehouseID,
			&order.Subtotal,
			&order.ShippingFee,
			&order.CODFee,
			&order.DiscountAmount,
			&order.TaxAmount,
			&order.Total,
			&order.PaymentMethod,
			&order.PaymentStatus,
			&order.PaidAt,
			&order.Status,
			&order.TrackingNumber,
			&order.EstimatedDeliveryAt,
			&order.DeliveredAt,
			&order.CustomerNote,
			&order.AdminNote,
			&order.CancellationReason,
			&order.CreatedAt,
			&order.UpdatedAt,
			&order.CancelledAt,
			&order.Version,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan order: %w", err)
		}
		orders = append(orders, order)
	}

	if rows.Err() != nil {
		return nil, 0, fmt.Errorf("error iterating orders: %w", rows.Err())
	}

	return orders, total, nil
}

// =====================================================
// ORDER STATUS HISTORY
// =====================================================

func (r *postgresOrderRepository) CreateOrderStatusHistory(ctx context.Context, history *model.OrderStatusHistory) error {
	query := `
		INSERT INTO order_status_history (
			id, order_id, from_status, to_status, changed_by, notes
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING changed_at
	`

	err := r.pool.QueryRow(ctx, query,
		history.ID,
		history.OrderID,
		history.FromStatus,
		history.ToStatus,
		history.ChangedBy,
		history.Notes,
	).Scan(&history.ChangedAt)

	if err != nil {
		return fmt.Errorf("failed to create order status history: %w", err)
	}

	return nil
}

func (r *postgresOrderRepository) CreateOrderStatusHistoryWithTx(ctx context.Context, tx pgx.Tx, history *model.OrderStatusHistory) error {
	query := `
		INSERT INTO order_status_history (
		 order_id, from_status, to_status, changed_by, notes
		) VALUES ($1, $2, $3, $4, $5)
		RETURNING changed_at
	`

	err := tx.QueryRow(ctx, query,
		history.OrderID,
		history.FromStatus,
		history.ToStatus,
		history.ChangedBy,
		history.Notes,
	).Scan(&history.ChangedAt)

	if err != nil {
		return fmt.Errorf("failed to create order status history with tx: %w", err)
	}

	return nil
}

func (r *postgresOrderRepository) GetOrderStatusHistory(ctx context.Context, orderID uuid.UUID) ([]model.OrderStatusHistory, error) {
	query := `
		SELECT 
			id, order_id, from_status, to_status, changed_by, notes, changed_at
		FROM order_status_history
		WHERE order_id = $1
		ORDER BY changed_at ASC
	`

	rows, err := r.pool.Query(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order status history: %w", err)
	}
	defer rows.Close()

	var histories []model.OrderStatusHistory
	for rows.Next() {
		var history model.OrderStatusHistory
		err := rows.Scan(
			&history.ID,
			&history.OrderID,
			&history.FromStatus,
			&history.ToStatus,
			&history.ChangedBy,
			&history.Notes,
			&history.ChangedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order status history: %w", err)
		}
		histories = append(histories, history)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("error iterating order status history: %w", rows.Err())
	}

	return histories, nil
}
