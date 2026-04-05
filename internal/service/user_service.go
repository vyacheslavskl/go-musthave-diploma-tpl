package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/apperrors"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/auth"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/models"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type UserServicer interface {
	Register(ctx context.Context, login, password string) (string, error)
	Login(ctx context.Context, login, password string) (string, error)
}

const TokenExp = time.Hour * 12

type UserService struct {
	repo repository.UserRepository
	jwt  auth.JWTServiceInterface
}

func NewUserService(repo repository.UserRepository, jwt auth.JWTServiceInterface) UserServicer {
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

	err = s.repo.CreateUser(ctx, user)
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
		return "", apperrors.ErrInvalidCredentials
	}

	return s.jwt.GenerateToken(user.UserID)
}
