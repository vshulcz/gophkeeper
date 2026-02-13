package persistence

import (
	"errors"
	"gophkeeper/internal/domain"
	"testing"
)

func TestUserRepo_GetByUsernameNotFound(t *testing.T) {
	db, ctx := openTestDB(t)
	repo := NewUserRepo(db)
	_, err := repo.GetByUsername(ctx, "missing")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}
