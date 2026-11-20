package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/Ayush001verma/zomato-order-service/internal/model"
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

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
