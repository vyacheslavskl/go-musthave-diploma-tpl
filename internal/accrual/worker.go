package accrual

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

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

func (w *Worker) Run(ctx context.Context, concurrency int) {
	tasks := make(chan models.Order)

	var wg sync.WaitGroup

	// workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for order := range tasks {
				w.processOrder(ctx, order)
			}
		}()
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			close(tasks)
			wg.Wait()
			w.log.Info("worker stopped")
			return

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
					wg.Wait()
					return
				}
			}
		}
	}
}

func (w *Worker) processOrder(ctx context.Context, order models.Order) {
	w.waitIfPaused()

	res, err := w.client.GetAccrualInfo(ctx, order.Number)
	if err != nil {

		switch {
		case errors.Is(err, apperrors.ErrOrderNotFound):
			w.log.Infof("order not registered yet: %s", order.Number)
			return

		case errors.Is(err, apperrors.ErrTooManyRequests):
			w.log.Warnf("rate limited, retry after %s", res.RetryAfter)
			w.setPause(res.RetryAfter)
			return

		default:
			w.log.Errorw("accrual request failed",
				"order", order.Number,
				"err", err,
			)
			return
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
