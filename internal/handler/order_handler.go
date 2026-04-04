package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/apperrors"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/auth"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/service"
)

type OrderHandler struct {
	svc service.OrderServicer
}

type OrderResponse struct {
	Number     string   `json:"number"`
	Status     string   `json:"status"`
	Accrual    *float64 `json:"accrual,omitempty"`
	UploadedAt string   `json:"uploaded_at"`
}

func NewOrderHandler(svc service.OrderServicer) *OrderHandler {
	return &OrderHandler{svc: svc}
}

// POST /api/user/orders (text/plain)
func (h *OrderHandler) AddOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := ctx.Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	number := strings.TrimSpace(string(body))
	if number == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = h.svc.AddOrder(ctx, userID, number)
	switch err {
	case nil:
		w.WriteHeader(http.StatusAccepted) // 202
	case apperrors.ErrOrderAlreadyExistsForUser:
		w.WriteHeader(http.StatusOK) // 200
	case apperrors.ErrOrderAlreadyExistsOtherUser:
		w.WriteHeader(http.StatusConflict) // 409
	case apperrors.ErrInvalidOrderNumber:
		w.WriteHeader(http.StatusUnprocessableEntity) // 422
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// GET /api/user/orders
func (h *OrderHandler) GetOrders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := ctx.Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	orders, err := h.svc.GetOrders(ctx, userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	resp := make([]OrderResponse, len(orders))
	for i, o := range orders {
		resp[i] = OrderResponse{
			Number:     o.Number,
			Status:     o.Status,
			Accrual:    o.Accrual,
			UploadedAt: o.CreatedAt.Format(time.RFC3339),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
