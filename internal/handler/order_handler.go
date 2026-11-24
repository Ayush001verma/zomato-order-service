package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/Ayush001verma/zomato-order-service/internal/model"
	"github.com/Ayush001verma/zomato-order-service/internal/repository"
	"github.com/Ayush001verma/zomato-order-service/internal/service"
)

// OrderHandler is the HTTP layer for orders.
type OrderHandler struct {
	svc *service.OrderService
}

// NewOrderHandler builds a handler that calls the given service.
func NewOrderHandler(svc *service.OrderService) *OrderHandler {
	return &OrderHandler{svc: svc}
}

// RegisterRoutes attaches this handler's routes to the mux.
func (h *OrderHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /orders", h.CreateOrder)
	mux.HandleFunc("GET /orders/{id}", h.GetOrder)
	mux.HandleFunc("PATCH /orders/{id}/status", h.UpdateOrderStatus)
}

// CreateOrder handles POST /orders.
func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	// Refuse bodies larger than 1 MB.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields() // reject fields we don't know, e.g. "total_amount_paise"

	var req model.CreateOrderRequest
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	req.IdempotencyKey = strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	order, err := h.svc.CreateOrder(r.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrValidation) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		log.Printf("create order failed: %v", err) // details go to our logs only
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, order)
}

// GetOrder handles GET /orders/{id}.
func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	order, err := h.svc.GetOrder(r.Context(), r.PathValue("id"))
	if err != nil {
		if errors.Is(err, service.ErrValidation) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		log.Printf("get order failed: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, order)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// updateStatusRequest is the JSON body of PATCH /orders/{id}/status.
type updateStatusRequest struct {
	Status model.OrderStatus `json:"status"`
}

// UpdateOrderStatus handles PATCH /orders/{id}/status.
func (h *OrderHandler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req updateStatusRequest
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	order, err := h.svc.UpdateOrderStatus(r.Context(), r.PathValue("id"), req.Status)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, repository.ErrNotFound):
			writeError(w, http.StatusNotFound, "order not found")
		case errors.Is(err, model.ErrInvalidTransition):
			writeError(w, http.StatusConflict, err.Error())
		default:
			log.Printf("update order status failed: %v", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusOK, order)
}
