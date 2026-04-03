package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/auth"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/middleware"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/service"
	"go.uber.org/zap"
)

type GopherMartHandler struct {
	userSvc  *service.UserService
	orderSvc *service.OrderService
	jwtSvc   *auth.JWTService
	log      *zap.SugaredLogger
}

func NewGopherMartHandler(userSvc *service.UserService, orderSvc *service.OrderService, jwtSvc *auth.JWTService, log *zap.SugaredLogger) *GopherMartHandler {
	return &GopherMartHandler{
		userSvc:  userSvc,
		orderSvc: orderSvc,
		jwtSvc:   jwtSvc,
		log:      log,
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

	orderHandler := NewOrderHandler(h.orderSvc)

	r.Group(func(r chi.Router) {
		r.Use(auth.JWTMiddleware(h.jwtSvc, h.log))
		r.Post("/api/user/orders", orderHandler.AddOrder)
		r.Get("/api/user/orders", orderHandler.GetOrders)
	})

	return r
}
