package model

import "errors"

// ErrInvalidTransition is returned when a status change is not allowed.
var ErrInvalidTransition = errors.New("invalid status transition")

// allowedTransitions lists, for each status, the statuses it may move to.
// A status with an empty list is final.
var allowedTransitions = map[OrderStatus][]OrderStatus{
	StatusPlaced:    {StatusConfirmed, StatusCancelled},
	StatusConfirmed: {StatusPreparing, StatusCancelled},
	StatusPreparing: {StatusPickedUp, StatusCancelled},
	StatusPickedUp:  {StatusDelivered},
	StatusDelivered: {},
	StatusCancelled: {},
}

// IsValid reports whether s is a status we know about.
func (s OrderStatus) IsValid() bool {
	_, ok := allowedTransitions[s]
	return ok
}

// CanTransition reports whether an order may move from one status to another.
func CanTransition(from, to OrderStatus) bool {
	for _, next := range allowedTransitions[from] {
		if next == to {
			return true
		}
	}
	return false
}
