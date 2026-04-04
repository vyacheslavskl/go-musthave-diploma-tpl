package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	models "github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/models"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/repository"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/service"
)

type UserHandler struct {
	svc service.UserServicer
}

func NewUserHandler(svc service.UserServicer) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var creds models.UserCreds
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if creds.Login == "" || creds.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	token, err := h.svc.Register(ctx, creds.Login, creds.Password)
	if err != nil {
		var dupErr *repository.DuplicateError
		if errors.As(err, &dupErr) {
			w.WriteHeader(http.StatusConflict)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Authorization", "Bearer "+token)
	w.WriteHeader(http.StatusOK)
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var creds models.UserCreds
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if creds.Login == "" || creds.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	token, err := h.svc.Login(ctx, creds.Login, creds.Password)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	w.Header().Set("Authorization", "Bearer "+token)
	w.WriteHeader(http.StatusOK)
}
