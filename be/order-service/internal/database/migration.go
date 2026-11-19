package database

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog/log"
)

func Migrate(pool *pgxpool.Pool, fs embed.FS, dir string) error {
	log.Info().Msgf("Running Database migrations using goose from directory: %s", dir)

	goose.SetLogger(&GooseLogger{})

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("database migrations: could not set dialect: %v", err)
	}

	goose.SetBaseFS(fs)

	sqlDB := stdlib.OpenDBFromPool(pool)
	defer sqlDB.Close()

	if err := runGooseUp(sqlDB, dir); err != nil {
		return err
	}

	log.Info().Msgf("Database migrations from '%s' applied successfully", dir)

	return nil
}

func runGooseUp(db *sql.DB, dir string) error {
	if err := goose.Up(db, dir); err != nil {
		return fmt.Errorf("database migrations: could not apply migrations: %v", err)
	}

	return nil
}
