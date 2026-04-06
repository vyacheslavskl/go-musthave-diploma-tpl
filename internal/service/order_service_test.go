package service

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/apperrors"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/models"
	mock_repository "github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/service/mocks"
)

func TestOrderService_AddOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_repository.NewMockOrderRepository(ctrl)
	service := NewOrderService(mockRepo)

	userID := "user-123"
	orderNumber := "6011111111111117"

	ctx := context.Background()

	tests := []struct {
		name        string
		setup       func()
		userID      string
		number      string
		expectedErr error
	}{
		{
			name:        "invalid order number (Luhn)",
			userID:      userID,
			number:      "12345678901",
			setup:       func() {},
			expectedErr: apperrors.ErrInvalidOrderNumber,
		},
		{
			name:   "order already exists for same user",
			userID: userID,
			number: orderNumber,
			setup: func() {
				mockRepo.EXPECT().GetUserByOrder(ctx, orderNumber).Return(userID, nil)
			},
			expectedErr: apperrors.ErrOrderAlreadyExistsForUser,
		},
		{
			name:   "order already exists for another user",
			userID: userID,
			number: orderNumber,
			setup: func() {
				mockRepo.EXPECT().GetUserByOrder(ctx, orderNumber).Return("another-user", nil)
			},
			expectedErr: apperrors.ErrOrderAlreadyExistsOtherUser,
		},
		{
			name:   "error from repo in GetUserByOrder",
			userID: userID,
			number: orderNumber,
			setup: func() {
				mockRepo.EXPECT().GetUserByOrder(ctx, orderNumber).Return("", errors.New("db error"))
			},
			expectedErr: errors.New("db error"),
		},
		{
			name:   "successful order creation",
			userID: userID,
			number: orderNumber,
			setup: func() {
				mockRepo.EXPECT().GetUserByOrder(ctx, orderNumber).Return("", nil)
				mockRepo.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, order models.Order) error {
						assert.Equal(t, userID, order.UserID)
						assert.Equal(t, orderNumber, order.Number)
						assert.NotEmpty(t, order.OrderID)
						return nil
					},
				)
			},
			expectedErr: nil,
		},
		{
			name:   "repo Create returns error",
			userID: userID,
			number: orderNumber,
			setup: func() {
				mockRepo.EXPECT().GetUserByOrder(ctx, orderNumber).Return("", nil)
				mockRepo.EXPECT().Create(ctx, gomock.Any()).Return(errors.New("create failed"))
			},
			expectedErr: errors.New("create failed"),
		},
		{
			name:        "empty order number",
			userID:      userID,
			number:      "",
			setup:       func() {},
			expectedErr: apperrors.ErrInvalidOrderNumber,
		},
		{
			name:        "non-digit order number",
			userID:      userID,
			number:      "abc123",
			setup:       func() {},
			expectedErr: apperrors.ErrInvalidOrderNumber,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			err := service.AddOrder(ctx, tt.userID, tt.number)

			if tt.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			}
		})
	}
}

func TestOrderService_GetOrders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_repository.NewMockOrderRepository(ctrl)
	service := NewOrderService(mockRepo)

	userID := "user-123"
	ctx := context.Background()

	tests := []struct {
		name        string
		setup       func()
		userID      string
		expected    []models.Order
		expectedErr error
	}{
		{
			name:   "get orders successfully",
			userID: userID,
			setup: func() {
				orders := []models.Order{
					{OrderID: uuid.NewString(), UserID: userID, Number: "1234567890"},
					{OrderID: uuid.NewString(), UserID: userID, Number: "0987654321"},
				}
				mockRepo.EXPECT().ListByUser(ctx, userID).Return(orders, nil)
			},
			expected:    nil, // проверяется внутри EXPECT
			expectedErr: nil,
		},
		{
			name:   "no orders found",
			userID: userID,
			setup: func() {
				mockRepo.EXPECT().ListByUser(ctx, userID).Return([]models.Order{}, nil)
			},
			expected:    []models.Order{},
			expectedErr: nil,
		},
		{
			name:   "repo error",
			userID: userID,
			setup: func() {
				mockRepo.EXPECT().ListByUser(ctx, userID).Return(nil, errors.New("db error"))
			},
			expected:    nil,
			expectedErr: errors.New("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			result, err := service.GetOrders(ctx, tt.userID)

			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
				if tt.expected != nil {
					assert.Equal(t, tt.expected, result)
				}
			}
		})
	}
}
