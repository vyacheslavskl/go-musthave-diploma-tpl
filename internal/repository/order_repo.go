package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/models"
)

type OrderRepository interface {
	Create(ctx context.Context, order models.Order) error
	ListByUser(ctx context.Context, userID string) ([]models.Order, error)
	GetUserByOrder(ctx context.Context, number string) (string, error)
}

type OrderRepo struct {
	db *pgxpool.Pool
}

func NewOrderRepo(db *pgxpool.Pool) *OrderRepo {
	return &OrderRepo{db: db}
}

func (r *OrderRepo) GetUserByOrder(ctx context.Context, number string) (string, error) {
	var userID string

	err := r.db.QueryRow(ctx,
		`SELECT user_id FROM orders WHERE number = $1`,
		number,
	).Scan(&userID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}

	return userID, nil
}

func (r *OrderRepo) Create(ctx context.Context, order models.Order) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO orders (order_id, user_id, number) VALUES ($1, $2, $3)`,
		order.OrderID, order.UserID, order.Number,
	)
	return err
}

func (r *OrderRepo) ListByUser(ctx context.Context, userID string) ([]models.Order, error) {
	rows, err := r.db.Query(ctx, `
        SELECT order_id, user_id, number, status, accrual, created_at
        FROM orders
        WHERE user_id=$1
		ORDER BY created_at DESC
    `, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		var accrual *float64
		if err := rows.Scan(&o.OrderID, &o.UserID, &o.Number, &o.Status, &accrual, &o.CreatedAt); err != nil {
			return nil, err
		}
		o.Accrual = accrual
		orders = append(orders, o)
	}
	return orders, nil
}
