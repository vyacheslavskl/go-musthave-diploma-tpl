package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/apperrors"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/auth"
	mock_service "github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/handler/mocks"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/models"
)

const testUserID = "test-user-id"

func TestOrderHandler_AddOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mock_service.NewMockOrderServicer(ctrl)
	handler := NewOrderHandler(mockSvc)

	tests := []struct {
		name           string
		userID         string
		body           string
		mockSetup      func()
		expectedStatus int
	}{
		{
			name:   "valid order number",
			userID: testUserID,
			body:   "123456789012",
			mockSetup: func() {
				mockSvc.EXPECT().AddOrder(gomock.Any(), testUserID, "123456789012").
					Return(nil)
			},
			expectedStatus: http.StatusAccepted, // 202
		},
		{
			name:   "order already exists for user",
			userID: testUserID,
			body:   "123456789012",
			mockSetup: func() {
				mockSvc.EXPECT().AddOrder(gomock.Any(), testUserID, "123456789012").
					Return(apperrors.ErrOrderAlreadyExistsForUser)
			},
			expectedStatus: http.StatusOK, // 200
		},
		{
			name:   "order already exists for another user",
			userID: testUserID,
			body:   "123456789012",
			mockSetup: func() {
				mockSvc.EXPECT().AddOrder(gomock.Any(), testUserID, "123456789012").
					Return(apperrors.ErrOrderAlreadyExistsOtherUser)
			},
			expectedStatus: http.StatusConflict, // 409
		},
		{
			name:   "invalid order number (Luhn)",
			userID: testUserID,
			body:   "123456789013", // invalid Luhn
			mockSetup: func() {
				mockSvc.EXPECT().AddOrder(gomock.Any(), testUserID, "123456789013").
					Return(apperrors.ErrInvalidOrderNumber)
			},
			expectedStatus: http.StatusUnprocessableEntity, // 422
		},
		{
			name:   "internal error",
			userID: testUserID,
			body:   "123456789012",
			mockSetup: func() {
				mockSvc.EXPECT().AddOrder(gomock.Any(), testUserID, "123456789012").
					Return(assert.AnError) // любая ошибка
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "missing user ID in context",
			userID:         "",
			body:           "123456789012",
			mockSetup:      func() {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "empty body",
			userID:         testUserID,
			body:           "",
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "only whitespace in body",
			userID:         testUserID,
			body:           " \n\t  ",
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "trims whitespace",
			userID: testUserID,
			body:   "  123456789012  ",
			mockSetup: func() {
				mockSvc.EXPECT().AddOrder(gomock.Any(), testUserID, "123456789012").
					Return(nil)
			},
			expectedStatus: http.StatusAccepted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			req := httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()

			// Устанавливаем userID в контекст
			ctx := req.Context()
			if tt.userID != "" {
				ctx = context.WithValue(ctx, auth.UserIDKey, tt.userID)
			}
			req = req.WithContext(ctx)

			handler.AddOrder(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestOrderHandler_GetOrders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mock_service.NewMockOrderServicer(ctrl)
	handler := NewOrderHandler(mockSvc)

	now := time.Now().UTC()
	order1 := models.Order{
		OrderID:   "ord-1",
		UserID:    testUserID,
		Number:    "1234567890",
		Status:    "PROCESSED",
		Accrual:   floatPtr(500.0),
		CreatedAt: now,
	}
	order2 := models.Order{
		OrderID:   "ord-2",
		UserID:    testUserID,
		Number:    "9876543210",
		Status:    "REGISTERED",
		Accrual:   nil,
		CreatedAt: now.Add(-1 * time.Hour),
	}

	tests := []struct {
		name           string
		userID         string
		mockSetup      func()
		expectedStatus int
		expectedBody   []OrderResponse
	}{
		{
			name:   "orders exist",
			userID: testUserID,
			mockSetup: func() {
				mockSvc.EXPECT().GetOrders(gomock.Any(), testUserID).
					Return([]models.Order{order1, order2}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: []OrderResponse{
				{
					Number:     "1234567890",
					Status:     "PROCESSED",
					Accrual:    floatPtr(500.0),
					UploadedAt: now.Format(time.RFC3339),
				},
				{
					Number:     "9876543210",
					Status:     "REGISTERED",
					Accrual:    nil,
					UploadedAt: now.Add(-1 * time.Hour).Format(time.RFC3339),
				},
			},
		},
		{
			name:   "no orders",
			userID: testUserID,
			mockSetup: func() {
				mockSvc.EXPECT().GetOrders(gomock.Any(), testUserID).
					Return([]models.Order{}, nil)
			},
			expectedStatus: http.StatusNoContent,
			expectedBody:   nil,
		},
		{
			name:   "internal error",
			userID: testUserID,
			mockSetup: func() {
				mockSvc.EXPECT().GetOrders(gomock.Any(), testUserID).
					Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   nil,
		},
		{
			name:           "missing user ID in context",
			userID:         "",
			mockSetup:      func() {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
			w := httptest.NewRecorder()

			ctx := req.Context()
			if tt.userID != "" {
				ctx = context.WithValue(ctx, auth.UserIDKey, tt.userID)
			}
			req = req.WithContext(ctx)

			handler.GetOrders(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				assert.JSONEq(t, toJSON(tt.expectedBody), w.Body.String())
			} else if tt.expectedStatus == http.StatusOK {
				assert.JSONEq(t, "[]", w.Body.String())
			} else {
				assert.Empty(t, w.Body.String())
			}

			if tt.expectedStatus == http.StatusOK {
				assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
			}
		})
	}
}

// Вспомогательные функции

func floatPtr(f float64) *float64 {
	return &f
}

func toJSON(v interface{}) string {
	data, _ := json.Marshal(v)
	return string(data)
}
