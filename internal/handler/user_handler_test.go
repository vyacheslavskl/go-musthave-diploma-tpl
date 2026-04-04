// internal/handler/user_handler_test.go
package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	mock_service "github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/handler/mocks"
	models "github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/models"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/repository"
)

func TestUserHandler_Register(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mock_service.NewMockUserServicer(ctrl)
	handler := NewUserHandler(mockSvc)

	tests := []struct {
		name           string
		input          models.UserCreds
		mockSetup      func()
		expectedStatus int
		expectedHeader string
	}{
		{
			name: "valid credentials",
			input: models.UserCreds{
				Login:    "user1",
				Password: "pass123",
			},
			mockSetup: func() {
				mockSvc.EXPECT().Register(gomock.Any(), "user1", "pass123").
					Return("fake-jwt-token", nil)
			},
			expectedStatus: http.StatusOK,
			expectedHeader: "Bearer fake-jwt-token",
		},
		{
			name: "empty login",
			input: models.UserCreds{
				Login:    "",
				Password: "pass123",
			},
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "empty password",
			input: models.UserCreds{
				Login:    "user1",
				Password: "",
			},
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "duplicate user",
			input: models.UserCreds{
				Login:    "user1",
				Password: "pass123",
			},
			mockSetup: func() {
				mockSvc.EXPECT().Register(gomock.Any(), "user1", "pass123").
					Return("", &repository.DuplicateError{Login: "user already exists"})
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name: "internal error",
			input: models.UserCreds{
				Login:    "user1",
				Password: "pass123",
			},
			mockSetup: func() {
				mockSvc.EXPECT().Register(gomock.Any(), "user1", "pass123").
					Return("", errors.New("unexpected error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			data, _ := json.Marshal(tt.input)
			req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(data))
			w := httptest.NewRecorder()

			handler.Register(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedHeader != "" {
				assert.Equal(t, tt.expectedHeader, w.Header().Get("Authorization"))
			} else {
				assert.Empty(t, w.Header().Get("Authorization"))
			}
		})
	}
}

func TestUserHandler_Login(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mock_service.NewMockUserServicer(ctrl)
	handler := NewUserHandler(mockSvc)

	tests := []struct {
		name           string
		input          models.UserCreds
		mockSetup      func()
		expectedStatus int
		expectedHeader string
	}{
		{
			name: "valid credentials",
			input: models.UserCreds{
				Login:    "user1",
				Password: "pass123",
			},
			mockSetup: func() {
				mockSvc.EXPECT().Login(gomock.Any(), "user1", "pass123").
					Return("fake-jwt-token", nil)
			},
			expectedStatus: http.StatusOK,
			expectedHeader: "Bearer fake-jwt-token",
		},
		{
			name: "empty login",
			input: models.UserCreds{
				Login:    "",
				Password: "pass123",
			},
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "empty password",
			input: models.UserCreds{
				Login:    "user1",
				Password: "",
			},
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid credentials",
			input: models.UserCreds{
				Login:    "user1",
				Password: "wrong",
			},
			mockSetup: func() {
				mockSvc.EXPECT().Login(gomock.Any(), "user1", "wrong").
					Return("", errors.New("invalid credentials"))
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "internal error in service",
			input: models.UserCreds{
				Login:    "user1",
				Password: "pass123",
			},
			mockSetup: func() {
				mockSvc.EXPECT().Login(gomock.Any(), "user1", "pass123").
					Return("", errors.New("db timeout"))
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			data, _ := json.Marshal(tt.input)
			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(data))
			w := httptest.NewRecorder()

			handler.Login(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedHeader != "" {
				assert.Equal(t, tt.expectedHeader, w.Header().Get("Authorization"))
			} else {
				assert.Empty(t, w.Header().Get("Authorization"))
			}
		})
	}
}
