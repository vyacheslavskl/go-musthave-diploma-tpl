package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/apperrors"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/auth"
	models "github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/models"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/service"
)

type BalanceHandler struct {
	svc *service.BalanceService
}

func NewBalanceHandler(svc *service.BalanceService) *BalanceHandler {
	return &BalanceHandler{svc: svc}
}

// GET /api/user/balance
func (h *BalanceHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := ctx.Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	balance, err := h.svc.GetBalance(ctx, userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(balance)
}

// GET /api/user/withdrawals
func (h *BalanceHandler) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := ctx.Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	withdrawals, err := h.svc.GetWithdrawals(r.Context(), userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(withdrawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(withdrawals)
}

// POST /api/user/balance/withdraw
func (h *BalanceHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	const layout = "2006-01-02T15:04:05Z07:00"

	ctx := r.Context()
	userID, ok := ctx.Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var req models.WithdrawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err := h.svc.Withdraw(ctx, userID, req)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrInvalidOrderNumber),
			errors.Is(err, apperrors.ErrOrderAlreadyExists):
			w.WriteHeader(http.StatusUnprocessableEntity)
		case errors.Is(err, apperrors.ErrInsufficientFunds):
			w.WriteHeader(http.StatusPaymentRequired)
		case errors.Is(err, apperrors.ErrInvalidSum):
			w.WriteHeader(http.StatusUnprocessableEntity)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}
