package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/auth"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/handler"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/migrate"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/repository"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/service"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Run() error {

	log := zap.NewDevelopmentConfig()
	log.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	logger, err := log.Build()
	if err != nil {
		return err
	}
	defer logger.Sync()

	sugar := logger.Sugar()

	_ = godotenv.Load()
	enVserAdr := os.Getenv("RUN_ADDRESS")
	enVdbAdr := os.Getenv("DATABASE_URI")
	enVaccAdr := os.Getenv("ACCRUAL_SYSTEM_ADDRESS")

	var serAdr, dbAdr, accAdr string
	flag.StringVar(&serAdr, "a", enVserAdr, "RUN_ADDRESS")
	flag.StringVar(&dbAdr, "d", enVdbAdr, "dsn for DATABASE_URI")
	flag.StringVar(&accAdr, "r", enVaccAdr, "ACCRUAL_SYSTEM_ADDRESS")
	flag.Parse()

	if serAdr == "" {
		serAdr = "localhost:8080"
	}
	if dbAdr == "" {
		return fmt.Errorf("DATABASE_URI is required")
	}
	jwt := os.Getenv("JWT_KEY")
	if jwt == "" {
		jwt = "secret_temp_key"
	}

	sugar.Infof("got envs  RUN_ADDRESS: %v DATABASE_URI:%v ACCRUAL_SYSTEM_ADDRESS:%v", serAdr, dbAdr, accAdr)

	initCtx, cancelInit := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelInit()
	pool, err := pgxpool.New(initCtx, dbAdr)
	if err != nil {
		return err
	}
	err = pool.Ping(initCtx)
	if err != nil {
		return err
	}
	sugar.Info("db connected succefually")
	migrateCtx, cancelMigrate := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelMigrate()
	err = migrate.Migrate(migrateCtx, pool, sugar)
	if err != nil {
		return err
	}

	repo, err := repository.NewUserRepo(pool)
	if err != nil {
		return err
	}
	jwtSvc := auth.NewJWTService([]byte(jwt))
	userSrc := service.NewUserService(repo, jwtSvc)

	orderRepo := repository.NewOrderRepo(pool)
	orderSvc := service.NewOrderService(orderRepo)

	handler := handler.NewGopherMartHandler(userSrc, orderSvc, jwtSvc, sugar)
	server := &http.Server{
		Addr:    serAdr,
		Handler: handler.Routes(),
	}

	serverErr := make(chan error, 1)
	go func() {
		sugar.Infof("starting server on %s", serAdr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()
	stopCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case <-stopCtx.Done():
		sugar.Info("shutdown signal received")
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			sugar.Errorf("server error: %v", err)
		}
	}

	return nil

}
