package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Ayush001verma/zomato-order-service/internal/model"
)

// ErrNotFound is returned when an order does not exist.
var ErrNotFound = errors.New("order not found")

// OrderRepository handles all database access for orders.
type OrderRepository struct {
	pool *pgxpool.Pool
}

// NewOrderRepository builds a repository that uses the given connection pool.
func NewOrderRepository(pool *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{pool: pool}
}

// Create saves an order and all its items in one transaction.
// On success it fills in the DB-generated fields (IDs and timestamps).
func (r *OrderRepository) Create(ctx context.Context, o *model.Order) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	// If we return early with an error, this undoes everything.
	// If Commit already succeeded, this does nothing.
	defer tx.Rollback(ctx)

	// Insert the order row. The DB generates id, created_at, and updated_at.
	err = tx.QueryRow(ctx,
		`INSERT INTO orders (customer_id, restaurant_id, status, total_amount_paise)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id::text, created_at, updated_at`,
		o.CustomerID, o.RestaurantID, string(o.Status), o.TotalAmountPaise,
	).Scan(&o.ID, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}

	// Insert each item, linked to the order by order_id.
	for i := range o.Items {
		item := &o.Items[i]
		err = tx.QueryRow(ctx,
			`INSERT INTO order_items (order_id, item_id, name, quantity, unit_price_paise)
			 VALUES ($1, $2, $3, $4, $5)
			 RETURNING id`,
			o.ID, item.ItemID, item.Name, item.Quantity, item.UnitPricePaise,
		).Scan(&item.ID)
		if err != nil {
			return fmt.Errorf("insert order item: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// GetByID loads one order together with its items.
func (r *OrderRepository) GetByID(ctx context.Context, id string) (*model.Order, error) {
	var o model.Order
	var status string

	err := r.pool.QueryRow(ctx,
		`SELECT id::text, customer_id, restaurant_id, status, total_amount_paise, created_at, updated_at
		 FROM orders WHERE id = $1::uuid`,
		id,
	).Scan(&o.ID, &o.CustomerID, &o.RestaurantID, &status, &o.TotalAmountPaise, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("select order: %w", err)
	}
	o.Status = model.OrderStatus(status)

	rows, err := r.pool.Query(ctx,
		`SELECT id, item_id, name, quantity, unit_price_paise
		 FROM order_items WHERE order_id = $1::uuid ORDER BY id`,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("select order items: %w", err)
	}
	defer rows.Close()

	o.Items = []model.OrderItem{}
	for rows.Next() {
		var it model.OrderItem
		if err := rows.Scan(&it.ID, &it.ItemID, &it.Name, &it.Quantity, &it.UnitPricePaise); err != nil {
			return nil, fmt.Errorf("scan order item: %w", err)
		}
		o.Items = append(o.Items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate order items: %w", err)
	}

	return &o, nil
}

// UpdateStatus changes an order's status inside one transaction.
// It locks the order row, so two requests can't change it at the same time.
// The decide function receives the current status and returns an error to refuse the change.
func (r *OrderRepository) UpdateStatus(
	ctx context.Context,
	id string,
	newStatus model.OrderStatus,
	decide func(current model.OrderStatus) error,
) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var current string
	err = tx.QueryRow(ctx,
		`SELECT status FROM orders WHERE id = $1::uuid FOR UPDATE`, id,
	).Scan(&current)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("lock order: %w", err)
	}

	if err := decide(model.OrderStatus(current)); err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		`UPDATE orders SET status = $1, updated_at = now() WHERE id = $2::uuid`,
		string(newStatus), id,
	)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}
