package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/apperrors"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/models"
)

type BalanceRepository interface {
	GetBalance(ctx context.Context, userID string) (models.Balance, error)
	Withdraw(ctx context.Context, order models.BalanceOrder) error
	GetWithdrawals(ctx context.Context, userID string) ([]models.Withdrawal, error)
}

type BalanceRepo struct {
	db *pgxpool.Pool
}

func NewBalanceRepo(db *pgxpool.Pool) *BalanceRepo {
	return &BalanceRepo{db: db}
}

// Получаем общий баланс
func (r *BalanceRepo) GetBalance(ctx context.Context, userID string) (models.Balance, error) {
	row := r.db.QueryRow(ctx, `
		SELECT 
		    COALESCE(SUM(amount), 0),
		    COALESCE(SUM(CASE WHEN amount < 0 THEN -amount ELSE 0 END), 0)
		FROM balance_transactions
		WHERE user_id = $1
	`, userID)

	var b models.Balance
	if err := row.Scan(&b.Current, &b.Withdrawn); err != nil {
		return models.Balance{}, err
	}

	return b, nil
}

// Получаем историю списаний
func (r *BalanceRepo) GetWithdrawals(ctx context.Context, userID string) ([]models.Withdrawal, error) {
	rows, err := r.db.Query(ctx, `
		SELECT o.number, -bt.amount, bt.created_at
		FROM balance_transactions bt
		LEFT JOIN orders o ON o.order_id = bt.order_id
		WHERE bt.user_id = $1 AND bt.amount < 0
		ORDER BY bt.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []models.Withdrawal
	for rows.Next() {
		var w models.Withdrawal
		if err := rows.Scan(&w.Order, &w.Sum, &w.ProcessedAt); err != nil {
			return nil, err
		}
		res = append(res, w)
	}

	return res, nil
}

// списание
func (r *BalanceRepo) Withdraw(ctx context.Context, order models.BalanceOrder) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Блокируем все строки пользователя для предотвращения гонок
	_, err = tx.Exec(ctx, `
		SELECT 1 
		FROM balance_transactions 
		WHERE user_id = $1
		FOR UPDATE
	`, order.UserID)
	if err != nil {
		return err
	}

	var balance float64
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0)
		FROM balance_transactions
		WHERE user_id = $1
	`, order.UserID).Scan(&balance)
	if err != nil {
		return err
	}

	if balance < order.Sum {
		return apperrors.ErrInsufficientFunds
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO orders (order_id, user_id, number) VALUES ($1, $2, $3)`,
		order.OrderID, order.UserID, order.Number,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO balance_transactions (transaction_id, order_id, user_id, amount)
		VALUES ($1, $2, $3, $4)
	`, order.TransactionID, order.OrderID, order.UserID, -order.Sum)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
