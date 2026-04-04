// internal/service/balance_service_test.go
package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/apperrors"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/models"
	mock_repository "github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/service/mocks"
)

func mustParseTime(s string) time.Time {
	t, err := time.Parse("2006-01-02T15:04:05Z", s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestBalanceService_GetBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockBalanceRepo := mock_repository.NewMockBalanceRepository(ctrl)
	service := NewBalanceService(mockBalanceRepo, nil)

	ctx := context.Background()
	userID := "user-123"

	tests := []struct {
		name        string
		setup       func()
		userID      string
		expected    models.Balance
		expectedErr error
	}{
		{
			name:   "get balance successfully",
			userID: userID,
			setup: func() {
				balance := models.Balance{Current: 100.5, Withdrawn: 20.0}
				mockBalanceRepo.EXPECT().GetBalance(ctx, userID).Return(balance, nil)
			},
			expected:    models.Balance{Current: 100.5, Withdrawn: 20.0},
			expectedErr: nil,
		},
		{
			name:   "repo returns error",
			userID: userID,
			setup: func() {
				mockBalanceRepo.EXPECT().GetBalance(ctx, userID).Return(models.Balance{}, errors.New("db error"))
			},
			expected:    models.Balance{},
			expectedErr: errors.New("db error"),
		},
		{
			name:        "empty user ID",
			userID:      "",
			setup:       func() { mockBalanceRepo.EXPECT().GetBalance(ctx, "").Return(models.Balance{}, nil) },
			expected:    models.Balance{},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			result, err := service.GetBalance(ctx, tt.userID)

			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestBalanceService_GetWithdrawals(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockBalanceRepo := mock_repository.NewMockBalanceRepository(ctrl)
	service := NewBalanceService(mockBalanceRepo, nil)

	ctx := context.Background()
	userID := "user-123"

	successfulWithdrawals := []models.Withdrawal{
		{Order: "6011111111111117", Sum: 10.5, ProcessedAt: mustParseTime("2024-01-01T12:00:00Z")},
		{Order: "0987654321", Sum: 5.0, ProcessedAt: mustParseTime("2024-01-02T12:00:00Z")},
	}

	tests := []struct {
		name        string
		setup       func()
		userID      string
		expected    []models.Withdrawal
		expectedErr error
	}{
		{
			name:   "get withdrawals successfully",
			userID: userID,
			setup: func() {
				mockBalanceRepo.EXPECT().GetWithdrawals(ctx, userID).Return(successfulWithdrawals, nil)
			},
			expected:    successfulWithdrawals,
			expectedErr: nil,
		},
		{
			name:   "no withdrawals",
			userID: userID,
			setup: func() {
				mockBalanceRepo.EXPECT().GetWithdrawals(ctx, userID).Return([]models.Withdrawal{}, nil)
			},
			expected:    []models.Withdrawal{},
			expectedErr: nil,
		},
		{
			name:   "repo error",
			userID: userID,
			setup: func() {
				mockBalanceRepo.EXPECT().GetWithdrawals(ctx, userID).Return(nil, errors.New("db error"))
			},
			expected:    nil,
			expectedErr: errors.New("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			result, err := service.GetWithdrawals(ctx, tt.userID)

			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestBalanceService_Withdraw(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockBalanceRepo := mock_repository.NewMockBalanceRepository(ctrl)
	mockOrderRepo := mock_repository.NewMockOrderRepository(ctrl)
	service := NewBalanceService(mockBalanceRepo, mockOrderRepo)

	ctx := context.Background()
	userID := "user-123"

	validOrder := "6011111111111117"
	invalidOrder := "12345678901" // fails Luhn
	req := models.WithdrawRequest{
		Order: validOrder,
		Sum:   50.0,
	}

	tests := []struct {
		name        string
		setup       func()
		userID      string
		req         models.WithdrawRequest
		expectedErr error
	}{
		{
			name:        "invalid order number (Luhn)",
			userID:      userID,
			req:         models.WithdrawRequest{Order: invalidOrder, Sum: 10.0},
			setup:       func() {},
			expectedErr: apperrors.ErrInvalidOrderNumber,
		},
		{
			name:        "empty order number",
			userID:      userID,
			req:         models.WithdrawRequest{Order: "", Sum: 10.0},
			setup:       func() {},
			expectedErr: apperrors.ErrInvalidOrderNumber,
		},
		{
			name:        "non-digit order number",
			userID:      userID,
			req:         models.WithdrawRequest{Order: "abc123", Sum: 10.0},
			setup:       func() {},
			expectedErr: apperrors.ErrInvalidOrderNumber,
		},
		{
			name:        "zero sum",
			userID:      userID,
			req:         models.WithdrawRequest{Order: validOrder, Sum: 0},
			setup:       func() {},
			expectedErr: apperrors.ErrInvalidSum,
		},
		{
			name:        "negative sum",
			userID:      userID,
			req:         models.WithdrawRequest{Order: validOrder, Sum: -10.0},
			setup:       func() {},
			expectedErr: apperrors.ErrInvalidSum,
		},
		{
			name:   "order already exists for another user",
			userID: userID,
			req:    req,
			setup: func() {
				mockOrderRepo.EXPECT().GetUserByOrder(ctx, validOrder).Return("another-user", nil)
			},
			expectedErr: apperrors.ErrOrderAlreadyExists,
		},
		{
			name:   "error in GetUserByOrder",
			userID: userID,
			req:    req,
			setup: func() {
				mockOrderRepo.EXPECT().GetUserByOrder(ctx, validOrder).Return("", errors.New("db error"))
			},
			expectedErr: errors.New("db error"),
		},
		{
			name:   "insufficient funds",
			userID: userID,
			req:    req,
			setup: func() {
				mockOrderRepo.EXPECT().GetUserByOrder(ctx, validOrder).Return("", nil)
				mockBalanceRepo.EXPECT().Withdraw(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, order models.BalanceOrder) error {
						assert.Equal(t, userID, order.UserID)
						assert.Equal(t, validOrder, order.Number)
						assert.Equal(t, 50.0, order.Sum)
						return apperrors.ErrInsufficientFunds
					},
				)
			},
			expectedErr: apperrors.ErrInsufficientFunds,
		},
		{
			name:   "other repo error on withdraw",
			userID: userID,
			req:    req,
			setup: func() {
				mockOrderRepo.EXPECT().GetUserByOrder(ctx, validOrder).Return("", nil)
				mockBalanceRepo.EXPECT().Withdraw(ctx, gomock.Any()).Return(errors.New("unknown error"))
			},
			expectedErr: errors.New("unknown error"),
		},
		{
			name:   "successful withdrawal",
			userID: userID,
			req:    req,
			setup: func() {
				mockOrderRepo.EXPECT().GetUserByOrder(ctx, validOrder).Return("", nil)
				mockBalanceRepo.EXPECT().Withdraw(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, order models.BalanceOrder) error {
						assert.Equal(t, userID, order.UserID)
						assert.Equal(t, validOrder, order.Number)
						assert.Equal(t, 50.0, order.Sum)
						assert.NotEmpty(t, order.TransactionID)
						assert.NotEmpty(t, order.OrderID)
						return nil
					},
				)
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			err := service.Withdraw(ctx, tt.userID, tt.req)

			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
