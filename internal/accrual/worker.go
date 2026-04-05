package accrual

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/google/uuid"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/apperrors"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/models"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/repository"
	"go.uber.org/zap"
)

type Worker struct {
	orderRepo   *repository.OrderRepo
	balanceRepo *repository.BalanceRepo
	client      *Client
	log         *zap.SugaredLogger

	pauseUntil int64 // atomic (UnixNano)
}

func NewWorker(
	orderRepo *repository.OrderRepo,
	balanceRepo *repository.BalanceRepo,
	client *Client,
	log *zap.SugaredLogger,
) *Worker {
	return &Worker{
		orderRepo:   orderRepo,
		balanceRepo: balanceRepo,
		client:      client,
		log:         log,
	}
}

func (w *Worker) Run(ctx context.Context, concurrency int) error {
	tasks := make(chan models.Order, concurrency*2) // буфер для предотвращения блокировок
	g, ctx := errgroup.WithContext(ctx)

	// запуск воркеров
	for range concurrency {
		g.Go(func() error {
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case order, ok := <-tasks:
					if !ok {
						return nil
					}
					w.processOrder(ctx, order)
				}
			}
		})
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			close(tasks) // закрываем канал один раз
			return g.Wait()

		case <-ticker.C:
			orders, err := w.orderRepo.GetPendingOrders(ctx)
			if err != nil {
				w.log.Errorw("get pending orders failed", "err", err)
				continue
			}

			for _, o := range orders {
				select {
				case tasks <- o:
				case <-ctx.Done():
					close(tasks)
					return g.Wait()
				}
			}
		}
	}
}

func (w *Worker) processOrder(ctx context.Context, order models.Order) error {
	w.waitIfPaused()

	res, err := w.client.GetAccrualInfo(ctx, order.Number)
	if err != nil {

		switch {
		case errors.Is(err, apperrors.ErrOrderNotFound):
			w.log.Infof("order not registered yet: %s", order.Number)
			return nil

		case errors.Is(err, apperrors.ErrTooManyRequests):
			w.log.Warnf("rate limited, retry after %s", res.RetryAfter)
			w.setPause(res.RetryAfter)
		case errors.Is(err, apperrors.ErrInternalServer):
			w.log.Warnf("internal server error: %s", order.Number)
			return nil

		default:
			w.log.Errorw("accrual request failed",
				"order", order.Number,
				"err", err,
			)
			return nil
		}
	}

	info := res.Data
	info.Status = mapAccrualStatus(info.Status)

	sum := 0.0
	if info.Accrual != nil {
		sum = *info.Accrual
	}

	// бизнес-логика
	applyAmount := info.Status == "PROCESSED" && sum > 0

	b := models.BalanceOrder{
		TransactionID: uuid.New().String(),
		OrderID:       order.OrderID,
		UserID:        order.UserID,
		Number:        order.Number,
		Status:        info.Status,
		Sum:           sum,
	}

	err = w.balanceRepo.ApplyAccrual(ctx, b, applyAmount)
	if err != nil {
		w.log.Errorw("apply accrual failed",
			"order", order.Number,
			"err", err,
		)
	}
	return nil
}

func (w *Worker) setPause(d time.Duration) {
	newUntil := time.Now().Add(d).UnixNano()

	for {
		current := atomic.LoadInt64(&w.pauseUntil)

		if current >= newUntil {
			return
		}

		if atomic.CompareAndSwapInt64(&w.pauseUntil, current, newUntil) {
			return
		}
	}
}

func (w *Worker) waitIfPaused() {
	until := atomic.LoadInt64(&w.pauseUntil)
	if until == 0 {
		return
	}

	sleep := time.Until(time.Unix(0, until))
	if sleep > 0 {
		time.Sleep(sleep)
	}
}

// mapAccrualStatus конвертирует статусы внешнего сервиса в наши статусы orders
func mapAccrualStatus(accrualStatus string) string {
	switch accrualStatus {
	case "REGISTERED":
		return "PROCESSING"
	case "PROCESSING":
		return "PROCESSING"
	case "INVALID":
		return "INVALID"
	case "PROCESSED":
		return "PROCESSED"
	default:
		return "NEW" // fallback
	}
}
