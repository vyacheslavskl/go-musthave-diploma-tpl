package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/apperrors"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/models"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/repository"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/utils"
)

type BalanceServicer interface {
	GetBalance(ctx context.Context, userID string) (models.Balance, error)
	GetWithdrawals(ctx context.Context, userID string) ([]models.Withdrawal, error)
	Withdraw(ctx context.Context, userID string, req models.WithdrawRequest) error
}

type BalanceService struct {
	balanceRepo repository.BalanceRepository
	orderRepo   repository.OrderRepository
}

func NewBalanceService(repo repository.BalanceRepository, orderRepo repository.OrderRepository) BalanceServicer {
	return &BalanceService{balanceRepo: repo, orderRepo: orderRepo}
}

func (s *BalanceService) GetBalance(ctx context.Context, userID string) (models.Balance, error) {
	return s.balanceRepo.GetBalance(ctx, userID)
}

func (s *BalanceService) GetWithdrawals(ctx context.Context, userID string) ([]models.Withdrawal, error) {
	return s.balanceRepo.GetWithdrawals(ctx, userID)
}

func (s *BalanceService) Withdraw(ctx context.Context, userID string, req models.WithdrawRequest) error {
	if !utils.CheckLuhn(req.Order) {
		return apperrors.ErrInvalidOrderNumber
	}
	if req.Sum <= 0 {
		return apperrors.ErrInvalidSum
	}

	existingUser, err := s.orderRepo.GetUserByOrder(ctx, req.Order)
	if err != nil {
		return err
	}

	if existingUser != "" {
		return apperrors.ErrOrderAlreadyExists
	}

	order := models.BalanceOrder{
		TransactionID: uuid.NewString(),
		OrderID:       uuid.NewString(),
		UserID:        userID,
		Number:        req.Order,
		Sum:           req.Sum,
	}
	err = s.balanceRepo.Withdraw(ctx, order)
	if err != nil {
		if errors.Is(err, apperrors.ErrInsufficientFunds) {
			return apperrors.ErrInsufficientFunds
		}
		return err
	}

	return nil
}
