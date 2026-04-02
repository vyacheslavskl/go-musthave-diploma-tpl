package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/auth"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/auth/middleware"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/service"
	"go.uber.org/zap"
)

type GopherMartHandler struct {
	userSvc *service.UserService
	jwtSvc  *auth.JWTService
	log     *zap.SugaredLogger
}

func NewGopherMartHandler(userSvc *service.UserService, jwtSvc *auth.JWTService, log *zap.SugaredLogger) *GopherMartHandler {
	return &GopherMartHandler{
		userSvc: userSvc,
		jwtSvc:  jwtSvc,
		log:     log,
	}
}

func (h *GopherMartHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.WithLogging(h.log))

	userHandler := NewUserHandler(h.userSvc)
	r.Group(func(r chi.Router) {
		r.Post("/api/user/register", userHandler.Register)
		r.Post("/api/user/login", userHandler.Login)
	})

	r.Group(func(r chi.Router) {
		r.Use(auth.JWTMiddleware(h.jwtSvc, h.log))
		// r.Post("/api/user/orders", h.AddOrder)
		// r.Get("/api/user/orders", h.GetOrders)
	})

	return r
}
