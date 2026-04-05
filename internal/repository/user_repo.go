package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/models"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *models.User) error
	GetByLogin(ctx context.Context, login string) (*models.User, error)
}

type UserRepo struct {
	db *pgxpool.Pool
}

type DuplicateError struct {
	Login string
}

func (e *DuplicateError) Error() string {
	return fmt.Sprintf("login already exists: %s", e.Login)
}

func NewUserRepo(db *pgxpool.Pool) (*UserRepo, error) {
	repo := &UserRepo{}
	if db != nil {
		repo.db = db
	}
	return repo, nil
}

func (r *UserRepo) CreateUser(ctx context.Context, user *models.User) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO users (user_id, login, password_hash) VALUES ($1, $2, $3)`,
		user.UserID, user.Login, user.PasswordHash,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				return &DuplicateError{Login: user.Login}
			}
		}
		return err
	}
	return nil
}

func (r *UserRepo) GetByLogin(ctx context.Context, login string) (*models.User, error) {
	u := &models.User{}
	err := r.db.QueryRow(ctx,
		`SELECT user_id, password_hash FROM users WHERE login=$1`,
		login,
	).Scan(&u.UserID, &u.PasswordHash)

	if err != nil {
		return nil, err
	}
	return u, nil
}
