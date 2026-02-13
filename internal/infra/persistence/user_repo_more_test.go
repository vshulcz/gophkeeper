package persistence

import (
	"errors"
	"gophkeeper/internal/domain"
	"gophkeeper/internal/domain/user"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestUserRepo_ConflictAndNotFound(t *testing.T) {
	db, ctx := openTestDB(t)
	repo := NewUserRepo(db)

	u := user.User{ID: uuid.New(), Username: "alice", PasswordHash: []byte("hash"), CreatedAt: time.Now()}
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := repo.Create(ctx, u); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	_, err := repo.GetByID(ctx, uuid.New())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}
