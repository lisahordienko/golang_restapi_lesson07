package repository

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"libratrack"
)

func ApplyMigrations(path, databaseURL string) error {
	path = strings.TrimPrefix(path, "file://")
	sourceURL := "file://" + filepath.ToSlash(path)
	instance, err := migrate.New(sourceURL, databaseURL)
	if err != nil {
		return fmt.Errorf("create migration runner: %w", err)
	}
	defer instance.Close()
	if err := instance.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply database migrations: %w", err)
	}
	return nil
}

func ApplyEmbeddedMigrations(databaseURL string) error {
	source, err := iofs.New(libratrack.Files, "migrations")
	if err != nil {
		return fmt.Errorf("create embedded migration source: %w", err)
	}
	instance, err := migrate.NewWithSourceInstance("iofs", source, databaseURL)
	if err != nil {
		return fmt.Errorf("create migration runner: %w", err)
	}
	defer instance.Close()
	if err := instance.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply database migrations: %w", err)
	}
	return nil
}
