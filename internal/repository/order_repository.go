package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"go-order-service/internal/apperr"
	"go-order-service/internal/model"
)

type OrderRepository interface {
	Create(ctx context.Context, order model.Order) error
	GetByID(ctx context.Context, id string) (model.Order, error)
	List(ctx context.Context, limit, offset int) ([]model.Order, error)
	Count(ctx context.Context) (int, error)
	UpdateStatus(ctx context.Context, id, status string) error
}

type PostgresOrderRepository struct {
	db      *sql.DB
	timeout time.Duration
}

func NewPostgresOrderRepository(db *sql.DB, timeout time.Duration) *PostgresOrderRepository {
	return &PostgresOrderRepository{db: db, timeout: timeout}
}

func (r *PostgresOrderRepository) Create(ctx context.Context, order model.Order) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	const q = `
		INSERT INTO orders (id, customer_name, amount, status, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	if _, err := r.db.ExecContext(ctx, q, order.ID, order.CustomerName, order.Amount, order.Status, order.CreatedAt); err != nil {
		return fmt.Errorf("insert order id=%s: %w", order.ID, err)
	}
	return nil
}

func (r *PostgresOrderRepository) GetByID(ctx context.Context, id string) (model.Order, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	const q = `
		SELECT id, customer_name, amount, status, created_at
		FROM orders
		WHERE id = $1
	`

	var order model.Order
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&order.ID,
		&order.CustomerName,
		&order.Amount,
		&order.Status,
		&order.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Order{}, fmt.Errorf("order id=%s: %w", id, apperr.ErrNotFound)
		}
		return model.Order{}, fmt.Errorf("get order id=%s: %w", id, err)
	}

	return order, nil
}

func (r *PostgresOrderRepository) List(ctx context.Context, limit, offset int) ([]model.Order, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	const q = `
		SELECT id, customer_name, amount, status, created_at
		FROM orders
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, q, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list orders limit=%d offset=%d: %w", limit, offset, err)
	}
	defer rows.Close()

	orders := make([]model.Order, 0, limit)
	for rows.Next() {
		var order model.Order
		if err := rows.Scan(&order.ID, &order.CustomerName, &order.Amount, &order.Status, &order.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan order row: %w", err)
		}
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate order rows: %w", err)
	}

	return orders, nil
}

func (r *PostgresOrderRepository) Count(ctx context.Context) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	const q = `SELECT COUNT(*) FROM orders`
	var total int
	if err := r.db.QueryRowContext(ctx, q).Scan(&total); err != nil {
		return 0, fmt.Errorf("count orders: %w", err)
	}
	return total, nil
}

func (r *PostgresOrderRepository) UpdateStatus(ctx context.Context, id, status string) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	const q = `
		UPDATE orders
		SET status = $2
		WHERE id = $1
	`
	res, err := r.db.ExecContext(ctx, q, id, status)
	if err != nil {
		return fmt.Errorf("update order status id=%s status=%s: %w", id, status, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected id=%s: %w", id, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("order id=%s: %w", id, apperr.ErrNotFound)
	}

	return nil
}
