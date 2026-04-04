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
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/accrual"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/auth"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/handler"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/migrate"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/repository"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/service"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Run() error {

	log := zap.NewProductionConfig()
	log.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	log.EncoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.UTC().Format("2006-01-02T15:04:05.0000000Z"))
	} // <-- это ключевое изменение
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
	flag.StringVar(&dbAdr, "d", enVdbAdr, "DATABASE_URI")
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

	sugar.Infof("got envs  RUN_ADDRESS: %v ACCRUAL_SYSTEM_ADDRESS:%v", serAdr, accAdr)

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
	sugar.Info("db connected successfully")
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

	balanceRepo := repository.NewBalanceRepo(pool)
	balanceSvc := service.NewBalanceService(balanceRepo, orderRepo)

	handler := handler.NewGopherMartHandler(userSrc, orderSvc, balanceSvc, jwtSvc, sugar)
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

	accrualClient := accrual.NewClient(accAdr)
	accrualWorker := accrual.NewWorker(orderRepo, balanceRepo, accrualClient, sugar)

	go accrualWorker.Run(stopCtx, 5) // 5 воркеров

	select {
	case <-stopCtx.Done():
		sugar.Info("shutdown signal received")
		if err := server.Shutdown(context.Background()); err != nil {
			sugar.Errorf("server shutdown error: %v", err)
		}
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			sugar.Errorf("server error: %v", err)
		}
		_ = server.Shutdown(context.Background())
	}

	return nil

}
