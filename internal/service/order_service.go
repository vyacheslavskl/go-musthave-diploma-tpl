package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/apperrors"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/models"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/repository"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/utils"
)

type OrderServicer interface {
	AddOrder(ctx context.Context, userID, number string) error
	GetOrders(ctx context.Context, userID string) ([]models.Order, error)
}

type OrderService struct {
	repo repository.OrderRepository
}

func NewOrderService(repo repository.OrderRepository) OrderServicer {
	return &OrderService{repo: repo}
}

// AddOrder добавляет заказ для пользователя
func (s *OrderService) AddOrder(ctx context.Context, userID, number string) error {
	if !utils.CheckLuhn(number) {
		return apperrors.ErrInvalidOrderNumber
	}

	existingUser, err := s.repo.GetUserByOrder(ctx, number)
	if err != nil {
		return err
	}

	if existingUser == userID {
		return apperrors.ErrOrderAlreadyExistsForUser
	} else if existingUser != "" {
		return apperrors.ErrOrderAlreadyExistsOtherUser
	}

	order := models.Order{
		OrderID: uuid.NewString(),
		UserID:  userID,
		Number:  number,
	}
	return s.repo.Create(ctx, order)
}

func (s *OrderService) GetOrders(ctx context.Context, userID string) ([]models.Order, error) {
	return s.repo.ListByUser(ctx, userID)
}
