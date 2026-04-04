package service

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/models"
	mock_jwt "github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/service/mocks"
	mock_repo "github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/service/mocks"
	"golang.org/x/crypto/bcrypt"
)

const testLogin = "testuser"
const testPassword = "securepass123"

func TestUserService_Register(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_repo.NewMockUserRepository(ctrl)
	mockJWT := mock_jwt.NewMockJWTServiceInterface(ctrl)

	service := NewUserService(mockRepo, mockJWT).(*UserService)

	tests := []struct {
		name          string
		login         string
		password      string
		mockSetup     func()
		expectError   bool
		expectedToken string
	}{
		{
			name:     "valid credentials",
			login:    testLogin,
			password: testPassword,
			mockSetup: func() {
				mockRepo.EXPECT().CreateUser(
					gomock.Any(),
					gomock.AssignableToTypeOf(&models.User{}),
				).DoAndReturn(func(ctx context.Context, u *models.User) error {
					assert.Equal(t, testLogin, u.Login)
					assert.NotEmpty(t, u.UserID)
					assert.NotEmpty(t, u.PasswordHash)

					err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(testPassword))
					assert.NoError(t, err, "password hash should be valid")

					return nil
				})

				mockJWT.EXPECT().GenerateToken(gomock.Any()).Return("fake-jwt", nil)
			},
			expectError:   false,
			expectedToken: "fake-jwt",
		},
		{
			name:     "repo error on create",
			login:    testLogin,
			password: testPassword,
			mockSetup: func() {
				mockRepo.EXPECT().CreateUser(gomock.Any(), gomock.AssignableToTypeOf(&models.User{})).
					Return(errors.New("db error"))
			},
			expectError: true,
		},
		{
			name:     "jwt generation error",
			login:    testLogin,
			password: testPassword,
			mockSetup: func() {
				mockRepo.EXPECT().CreateUser(gomock.Any(), gomock.AssignableToTypeOf(&models.User{})).
					Return(nil)
				mockJWT.EXPECT().GenerateToken(gomock.Any()).Return("", errors.New("jwt error"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			token, err := service.Register(context.Background(), tt.login, tt.password)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedToken, token)
			}
		})
	}
}

func TestUserService_Login(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_repo.NewMockUserRepository(ctrl)
	mockJWT := mock_jwt.NewMockJWTServiceInterface(ctrl)

	service := NewUserService(mockRepo, mockJWT).(*UserService)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.DefaultCost)
	validUser := &models.User{
		UserID:       "user-123",
		Login:        testLogin,
		PasswordHash: string(hashedPassword),
	}

	tests := []struct {
		name          string
		login         string
		password      string
		mockSetup     func()
		expectError   bool
		expectedToken string
	}{
		{
			name:     "valid credentials",
			login:    testLogin,
			password: testPassword,
			mockSetup: func() {
				mockRepo.EXPECT().GetByLogin(gomock.Any(), testLogin).Return(validUser, nil)
				mockJWT.EXPECT().GenerateToken("user-123").Return("login-token", nil)
			},
			expectError:   false,
			expectedToken: "login-token",
		},
		{
			name:     "user not found",
			login:    "unknown",
			password: testPassword,
			mockSetup: func() {
				mockRepo.EXPECT().GetByLogin(gomock.Any(), "unknown").Return(nil, errors.New("not found"))
			},
			expectError: true,
		},
		{
			name:     "wrong password",
			login:    testLogin,
			password: "wrongpass",
			mockSetup: func() {
				mockRepo.EXPECT().GetByLogin(gomock.Any(), testLogin).Return(validUser, nil)
			},
			expectError: true,
		},
		{
			name:     "jwt error on login",
			login:    testLogin,
			password: testPassword,
			mockSetup: func() {
				mockRepo.EXPECT().GetByLogin(gomock.Any(), testLogin).Return(validUser, nil)
				mockJWT.EXPECT().GenerateToken("user-123").Return("", errors.New("jwt error"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			token, err := service.Login(context.Background(), tt.login, tt.password)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedToken, token)
			}
		})
	}
}
