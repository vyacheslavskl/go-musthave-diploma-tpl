// internal/handler/balance_handler_test.go
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

func TestBalanceHandler_GetBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mock_service.NewMockBalanceServicer(ctrl)
	handler := NewBalanceHandler(mockSvc)

	tests := []struct {
		name           string
		userID         string
		mockSetup      func()
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name:   "balance available",
			userID: testUserID,
			mockSetup: func() {
				balance := models.Balance{
					Current:   500.0,
					Withdrawn: 100.0,
				}
				mockSvc.EXPECT().GetBalance(gomock.Any(), testUserID).Return(balance, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"current":   500.0,
				"withdrawn": 100.0,
			},
		},
		{
			name:   "zero balance",
			userID: testUserID,
			mockSetup: func() {
				balance := models.Balance{
					Current:   0.0,
					Withdrawn: 0.0,
				}
				mockSvc.EXPECT().GetBalance(gomock.Any(), testUserID).Return(balance, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"current":   0.0,
				"withdrawn": 0.0,
			},
		},
		{
			name:           "missing user ID in context",
			userID:         "",
			mockSetup:      func() {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   nil,
		},
		{
			name:   "internal error",
			userID: testUserID,
			mockSetup: func() {
				mockSvc.EXPECT().GetBalance(gomock.Any(), testUserID).Return(models.Balance{}, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
			w := httptest.NewRecorder()

			ctx := req.Context()
			if tt.userID != "" {
				ctx = context.WithValue(ctx, auth.UserIDKey, tt.userID)
			}
			req = req.WithContext(ctx)

			handler.GetBalance(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				assert.JSONEq(t, toJSON(tt.expectedBody), w.Body.String())
			} else {
				assert.Empty(t, w.Body.String())
			}

			if tt.expectedStatus == http.StatusOK {
				assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
			}
		})
	}
}

func TestBalanceHandler_GetWithdrawals(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mock_service.NewMockBalanceServicer(ctrl)
	handler := NewBalanceHandler(mockSvc)

	now := time.Now().UTC()
	withdrawal1 := models.Withdrawal{
		Order:       "1234567890",
		Sum:         500.0,
		ProcessedAt: now,
	}
	withdrawal2 := models.Withdrawal{
		Order:       "9876543210",
		Sum:         300.0,
		ProcessedAt: now.Add(-1 * time.Hour),
	}

	tests := []struct {
		name           string
		userID         string
		mockSetup      func()
		expectedStatus int
		expectedBody   []models.Withdrawal
	}{
		{
			name:   "withdrawals exist",
			userID: testUserID,
			mockSetup: func() {
				mockSvc.EXPECT().GetWithdrawals(gomock.Any(), testUserID).
					Return([]models.Withdrawal{withdrawal1, withdrawal2}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   []models.Withdrawal{withdrawal1, withdrawal2},
		},
		{
			name:   "no withdrawals",
			userID: testUserID,
			mockSetup: func() {
				mockSvc.EXPECT().GetWithdrawals(gomock.Any(), testUserID).
					Return([]models.Withdrawal{}, nil)
			},
			expectedStatus: http.StatusNoContent,
			expectedBody:   nil,
		},
		{
			name:   "internal error",
			userID: testUserID,
			mockSetup: func() {
				mockSvc.EXPECT().GetWithdrawals(gomock.Any(), testUserID).
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

			req := httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)
			w := httptest.NewRecorder()

			ctx := req.Context()
			if tt.userID != "" {
				ctx = context.WithValue(ctx, auth.UserIDKey, tt.userID)
			}
			req = req.WithContext(ctx)

			handler.GetWithdrawals(w, req)

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

func TestBalanceHandler_Withdraw(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mock_service.NewMockBalanceServicer(ctrl)
	handler := NewBalanceHandler(mockSvc)

	tests := []struct {
		name           string
		userID         string
		requestBody    interface{}
		mockSetup      func()
		expectedStatus int
	}{
		{
			name:   "valid withdrawal",
			userID: testUserID,
			requestBody: models.WithdrawRequest{
				Order: "1234567890",
				Sum:   500.0,
			},
			mockSetup: func() {
				req := models.WithdrawRequest{
					Order: "1234567890",
					Sum:   500.0,
				}
				mockSvc.EXPECT().Withdraw(gomock.Any(), testUserID, req).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "invalid order number",
			userID:      testUserID,
			requestBody: models.WithdrawRequest{Order: "12345678901", Sum: 500.0},
			mockSetup: func() {
				req := models.WithdrawRequest{Order: "12345678901", Sum: 500.0}
				mockSvc.EXPECT().Withdraw(gomock.Any(), testUserID, req).
					Return(apperrors.ErrInvalidOrderNumber)
			},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:        "order already exists",
			userID:      testUserID,
			requestBody: models.WithdrawRequest{Order: "1234567890", Sum: 500.0},
			mockSetup: func() {
				req := models.WithdrawRequest{Order: "1234567890", Sum: 500.0}
				mockSvc.EXPECT().Withdraw(gomock.Any(), testUserID, req).
					Return(apperrors.ErrOrderAlreadyExists)
			},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:        "insufficient funds",
			userID:      testUserID,
			requestBody: models.WithdrawRequest{Order: "1234567890", Sum: 1000.0},
			mockSetup: func() {
				req := models.WithdrawRequest{Order: "1234567890", Sum: 1000.0}
				mockSvc.EXPECT().Withdraw(gomock.Any(), testUserID, req).
					Return(apperrors.ErrInsufficientFunds)
			},
			expectedStatus: http.StatusPaymentRequired,
		},
		{
			name:        "invalid sum (zero)",
			userID:      testUserID,
			requestBody: models.WithdrawRequest{Order: "1234567890", Sum: 0.0},
			mockSetup: func() {
				req := models.WithdrawRequest{Order: "1234567890", Sum: 0.0}
				mockSvc.EXPECT().Withdraw(gomock.Any(), testUserID, req).
					Return(apperrors.ErrInvalidSum)
			},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:        "negative sum",
			userID:      testUserID,
			requestBody: models.WithdrawRequest{Order: "1234567890", Sum: -100.0},
			mockSetup: func() {
				req := models.WithdrawRequest{Order: "1234567890", Sum: -100.0}
				mockSvc.EXPECT().Withdraw(gomock.Any(), testUserID, req).
					Return(apperrors.ErrInvalidSum)
			},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:        "internal error",
			userID:      testUserID,
			requestBody: models.WithdrawRequest{Order: "1234567890", Sum: 500.0},
			mockSetup: func() {
				req := models.WithdrawRequest{Order: "1234567890", Sum: 500.0}
				mockSvc.EXPECT().Withdraw(gomock.Any(), testUserID, req).
					Return(assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "missing user ID in context",
			userID:         "",
			requestBody:    models.WithdrawRequest{Order: "1234567890", Sum: 500.0},
			mockSetup:      func() {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid JSON body",
			userID:         testUserID,
			requestBody:    "invalid-json", // будет передан как строка
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			var body []byte
			if reqBody, ok := tt.requestBody.(models.WithdrawRequest); ok {
				body, _ = json.Marshal(reqBody)
			} else {
				body = []byte("invalid-json") // для теста с битым JSON
			}

			req := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			ctx := req.Context()
			if tt.userID != "" {
				ctx = context.WithValue(ctx, auth.UserIDKey, tt.userID)
			}
			req = req.WithContext(ctx)

			handler.Withdraw(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
