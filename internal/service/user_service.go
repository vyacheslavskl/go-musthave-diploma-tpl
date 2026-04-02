package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/auth"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

const TokenExp = time.Hour * 12

type UserService struct {
	repo repository.UserRepository
	jwt  *auth.JWTService
}

func NewUserService(repo *repository.UserRepo, jwt *auth.JWTService) *UserService {
	return &UserService{repo: repo, jwt: jwt}
}

func (s *UserService) Register(ctx context.Context, login, password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	user := &repository.User{
		ID:           uuid.NewString(),
		Login:        login,
		PasswordHash: string(hash),
	}

	err = s.repo.CreateUser(context.TODO(), user)
	if err != nil {
		return "", err
	}

	token, err := s.jwt.GenerateToken(user.ID)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *UserService) Login(ctx context.Context, login, password string) (string, error) {
	user, err := s.repo.GetByLogin(ctx, login)
	if err != nil {
		return "", errors.New("unauthorized")
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return "", errors.New("unauthorized")
	}

	return s.jwt.GenerateToken(user.ID)
}
