package model

import "time"

// OrderStatus is the state of an order in its lifecycle.
type OrderStatus string

const (
	StatusPlaced    OrderStatus = "PLACED"
	StatusConfirmed OrderStatus = "CONFIRMED"
	StatusPreparing OrderStatus = "PREPARING"
	StatusPickedUp  OrderStatus = "PICKED_UP"
	StatusDelivered OrderStatus = "DELIVERED"
	StatusCancelled OrderStatus = "CANCELLED"
)

// OrderItem is one line in an order (e.g. 2 x Paneer Tikka).
// Field names mirror the order_items table columns.
type OrderItem struct {
	ID             int64  `json:"id"`
	ItemID         string `json:"item_id"`
	Name           string `json:"name"`
	Quantity       int    `json:"quantity"`
	UnitPricePaise int64  `json:"unit_price_paise"` // price of ONE unit, in paise
}

// Order is what we store and return from the API.
type Order struct {
	ID               string      `json:"id"`
	CustomerID       string      `json:"customer_id"`
	RestaurantID     string      `json:"restaurant_id"`
	Status           OrderStatus `json:"status"`
	TotalAmountPaise int64       `json:"total_amount_paise"`
	IdempotencyKey   string      `json:"-"`
	Items            []OrderItem `json:"items"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
}

// CreateOrderItem is one item the customer wants to order.
type CreateOrderItem struct {
	ItemID         string `json:"item_id"`
	Name           string `json:"name"`
	Quantity       int    `json:"quantity"`
	UnitPricePaise int64  `json:"unit_price_paise"`
}

// CreateOrderRequest is the JSON body of POST /orders.
// Notice: no id, no status, no total. The server decides those.
type CreateOrderRequest struct {
	CustomerID     string            `json:"customer_id"`
	RestaurantID   string            `json:"restaurant_id"`
	Items          []CreateOrderItem `json:"items"`
	IdempotencyKey string            `json:"-"` // comes from the HTTP header, never from the JSON body
}
