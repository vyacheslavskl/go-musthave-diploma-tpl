package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	models "github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/model"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/repository"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/service"
)

type UserHandler struct {
	svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var creds models.UserCreds
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if creds.Login == "" || creds.Password == "" {
		http.Error(w, "bad request: missing login or password", http.StatusBadRequest)
		return
	}

	token, err := h.svc.Register(ctx, creds.Login, creds.Password)
	if err != nil {
		var dupErr *repository.DuplicateError
		if errors.As(err, &dupErr) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var creds models.UserCreds
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if creds.Login == "" || creds.Password == "" {
		http.Error(w, "bad request: missing login or password", http.StatusBadRequest)
		return
	}

	token, err := h.svc.Login(ctx, creds.Login, creds.Password)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"token": token})
}
