package migrate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

func Migrate(ctx context.Context, pool *pgxpool.Pool, sugar *zap.SugaredLogger) (err error) {
	db := stdlib.OpenDB(*pool.Config().ConnConfig)
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return errors.New("failed to get runtime caller")
	}
	path := filepath.Dir(filename)
	fullPath := filepath.Join(path, "..", "..", "migrations")
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return fmt.Errorf("migrations directory not found: %s", fullPath)
	}
	src := "file://" + filepath.ToSlash(fullPath)

	m, err := migrate.NewWithDatabaseInstance(src, "postgres", driver)
	if err != nil {
		sugar.Errorln("Migrations failed", err)
		return err
	}
	m.Up()
	sugar.Infow("Migrations applied")
	return nil
}
