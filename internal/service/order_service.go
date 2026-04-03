package service

import (
	"context"
	"errors"
	"regexp"

	"github.com/google/uuid"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/models"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/repository"
)

var (
	ErrOrderAlreadyExistsForUser   = errors.New("order already exists for user")
	ErrOrderAlreadyExistsOtherUser = errors.New("order already exists for another user")
	ErrInvalidOrderNumber          = errors.New("invalid order number")
)

type OrderService struct {
	repo *repository.OrderRepo
}

func NewOrderService(repo *repository.OrderRepo) *OrderService {
	return &OrderService{repo: repo}
}

// AddOrder добавляет заказ для пользователя
func (s *OrderService) AddOrder(ctx context.Context, userID, number string) error {
	if !checkLuhn(number) {
		return ErrInvalidOrderNumber
	}

	existingUser, err := s.repo.GetUserByOrder(ctx, number)
	if err != nil {
		return err
	}

	if existingUser == userID {
		return ErrOrderAlreadyExistsForUser
	} else if existingUser != "" {
		return ErrOrderAlreadyExistsOtherUser
	}

	order := models.Order{
		ID:     uuid.NewString(),
		UserID: userID,
		Number: number,
	}
	return s.repo.Create(ctx, order)
}

func (s *OrderService) GetOrders(ctx context.Context, userID string) ([]models.Order, error) {
	return s.repo.ListByUser(ctx, userID)
}

func checkLuhn(number string) bool {
	matched, _ := regexp.MatchString(`^\d+$`, number)
	if !matched {
		return false
	}
	sum := 0
	alt := false

	for i := len(number) - 1; i >= 0; i-- {
		n := int(number[i] - '0')
		if alt {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		alt = !alt
	}
	return sum%10 == 0
}
