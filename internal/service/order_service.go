package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/Ayush001verma/zomato-order-service/internal/model"
	"github.com/Ayush001verma/zomato-order-service/internal/repository"
)

// ErrValidation marks errors caused by bad input from the client.
// The handler will turn these into HTTP 400 responses.
var ErrValidation = errors.New("validation failed")

const (
	maxItemsPerOrder = 50
	maxQuantity      = 100
)

// OrderService holds the business rules for orders.
type OrderService struct {
	repo *repository.OrderRepository
}

// NewOrderService builds a service that saves orders via the given repository.
func NewOrderService(repo *repository.OrderRepository) *OrderService {
	return &OrderService{repo: repo}
}

// CreateOrder validates the request, computes the total, and saves the order.
func (s *OrderService) CreateOrder(ctx context.Context, req model.CreateOrderRequest) (*model.Order, error) {
	if err := validateCreateRequest(req); err != nil {
		return nil, err
	}

	order := &model.Order{
		CustomerID:     strings.TrimSpace(req.CustomerID),
		RestaurantID:   strings.TrimSpace(req.RestaurantID),
		Status:         model.StatusPlaced, // server decides, never the client
		IdempotencyKey: strings.TrimSpace(req.IdempotencyKey),
	}

	var total int64
	for _, it := range req.Items {
		order.Items = append(order.Items, model.OrderItem{
			ItemID:         strings.TrimSpace(it.ItemID),
			Name:           strings.TrimSpace(it.Name),
			Quantity:       it.Quantity,
			UnitPricePaise: it.UnitPricePaise,
		})
		total += int64(it.Quantity) * it.UnitPricePaise
	}
	order.TotalAmountPaise = total // server calculates, never the client

	if err := s.repo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}
	return order, nil
}

// GetOrder returns one order by id, or repository.ErrNotFound.
func (s *OrderService) GetOrder(ctx context.Context, id string) (*model.Order, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, repository.ErrNotFound // a malformed id can't exist
	}
	order, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("get order: %w", err)
	}
	return order, nil
}

func validateCreateRequest(req model.CreateOrderRequest) error {
	if strings.TrimSpace(req.CustomerID) == "" {
		return fmt.Errorf("%w: customer_id is required", ErrValidation)
	}
	if strings.TrimSpace(req.RestaurantID) == "" {
		return fmt.Errorf("%w: restaurant_id is required", ErrValidation)
	}
	if len(req.Items) == 0 {
		return fmt.Errorf("%w: at least one item is required", ErrValidation)
	}
	if len(req.Items) > maxItemsPerOrder {
		return fmt.Errorf("%w: at most %d items per order", ErrValidation, maxItemsPerOrder)
	}
	for i, it := range req.Items {
		if strings.TrimSpace(it.ItemID) == "" {
			return fmt.Errorf("%w: items[%d].item_id is required", ErrValidation, i)
		}
		if strings.TrimSpace(it.Name) == "" {
			return fmt.Errorf("%w: items[%d].name is required", ErrValidation, i)
		}
		if it.Quantity < 1 || it.Quantity > maxQuantity {
			return fmt.Errorf("%w: items[%d].quantity must be between 1 and %d", ErrValidation, i, maxQuantity)
		}
		if it.UnitPricePaise < 0 {
			return fmt.Errorf("%w: items[%d].unit_price_paise cannot be negative", ErrValidation, i)
		}
	}
	return nil
}

// UpdateOrderStatus moves an order to a new status if the state machine allows it.
func (s *OrderService) UpdateOrderStatus(ctx context.Context, id string, newStatus model.OrderStatus) (*model.Order, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, repository.ErrNotFound // a malformed id can't exist
	}
	if !newStatus.IsValid() {
		return nil, fmt.Errorf("%w: unknown status %q", ErrValidation, newStatus)
	}

	err := s.repo.UpdateStatus(ctx, id, newStatus, func(current model.OrderStatus) error {
		if !model.CanTransition(current, newStatus) {
			return fmt.Errorf("%w: cannot move from %s to %s", model.ErrInvalidTransition, current, newStatus)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) || errors.Is(err, model.ErrInvalidTransition) {
			return nil, err
		}
		return nil, fmt.Errorf("update order status: %w", err)
	}

	return s.GetOrder(ctx, id) // return the order as it is now
}
