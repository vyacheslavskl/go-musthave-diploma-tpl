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
	userSvc    *service.UserService
	orderSvc   *service.OrderService
	balanceSvc *service.BalanceService
	jwtSvc     *auth.JWTService
	log        *zap.SugaredLogger
}

func NewGopherMartHandler(userSvc *service.UserService, orderSvc *service.OrderService,
	balanceSvc *service.BalanceService, jwtSvc *auth.JWTService, log *zap.SugaredLogger) *GopherMartHandler {
	return &GopherMartHandler{
		userSvc:    userSvc,
		orderSvc:   orderSvc,
		balanceSvc: balanceSvc,
		jwtSvc:     jwtSvc,
		log:        log,
	}
}

func (h *GopherMartHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.WithLogging(h.log))

	userHandler := NewUserHandler(h.userSvc)
	orderHandler := NewOrderHandler(h.orderSvc)
	balanceHandler := NewBalanceHandler(h.balanceSvc)

	r.Route("/api/user", func(r chi.Router) {
		// Публичные роуты (без JWTMiddleware)
		r.Group(func(r chi.Router) {
			r.Post("/register", userHandler.Register)
			r.Post("/login", userHandler.Login)
		})

		// Защищённые роуты (JWT middleware)
		r.Group(func(r chi.Router) {
			r.Use(auth.JWTMiddleware(h.jwtSvc, h.log))

			r.Post("/orders", orderHandler.AddOrder)
			r.Get("/orders", orderHandler.GetOrders)

			r.Get("/balance", balanceHandler.GetBalance)
			r.Post("/balance/withdraw", balanceHandler.Withdraw)
			r.Get("/withdrawals", balanceHandler.GetWithdrawals)
		})
	})

	return r
}
