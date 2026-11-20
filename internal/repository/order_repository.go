package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Ayush001verma/zomato-order-service/internal/model"
)

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
