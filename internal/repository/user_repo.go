package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *User) error
	GetByLogin(ctx context.Context, login string) (*User, error)
}

type User struct {
	ID           string
	Login        string
	PasswordHash string
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

func NewRepo(db *pgxpool.Pool) (*UserRepo, error) {
	repo := &UserRepo{}
	if db != nil {
		repo.db = db
	}
	return repo, nil
}

func (r *UserRepo) CreateUser(ctx context.Context, user *User) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO users (id, login, password_hash) VALUES ($1, $2, $3)`,
		user.ID, user.Login, user.PasswordHash,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				return &DuplicateError{Login: user.Login}
			}
		}
	}
	return nil
}

func (r *UserRepo) GetByLogin(ctx context.Context, login string) (*User, error) {
	u := &User{}
	err := r.db.QueryRow(ctx,
		`SELECT id, password_hash FROM users WHERE login=$1`,
		login,
	).Scan(&u.ID, &u.PasswordHash)

	if err != nil {
		return nil, err
	}
	return u, nil
}
