package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/auth"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/models"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

const TokenExp = time.Hour * 12

type UserService struct {
	repo repository.UserRepository
	jwt  *auth.JWTService
}

func NewUserService(repo repository.UserRepository, jwt *auth.JWTService) *UserService {
	return &UserService{repo: repo, jwt: jwt}
}

func (s *UserService) Register(ctx context.Context, login, password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	user := &models.User{
		UserID:       uuid.NewString(),
		Login:        login,
		PasswordHash: string(hash),
	}

	err = s.repo.CreateUser(context.TODO(), user)
	if err != nil {
		return "", err
	}

	token, err := s.jwt.GenerateToken(user.UserID)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *UserService) Login(ctx context.Context, login, password string) (string, error) {
	user, err := s.repo.GetByLogin(ctx, login)
	if err != nil {
		return "", err
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return "", err
	}

	return s.jwt.GenerateToken(user.UserID)
}
