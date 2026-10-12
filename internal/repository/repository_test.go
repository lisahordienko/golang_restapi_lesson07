package repository

import (
	"errors"
	"testing"

	"gorm.io/gorm"
)

func TestNotFoundMatchesGorm(t *testing.T) {
	if !errors.Is(gorm.ErrRecordNotFound, ErrNotFound) {
		t.Fatal("repository not-found error must match GORM's not-found error")
	}
}
