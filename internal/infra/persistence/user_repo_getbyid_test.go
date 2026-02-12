package persistence

import (
	"gophkeeper/internal/domain/user"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestUserRepo_GetByID(t *testing.T) {
	db, ctx := openTestDB(t)
	repo := NewUserRepo(db)

	u := user.User{ID: uuid.New(), Username: "alice", PasswordHash: []byte("hash"), CreatedAt: time.Now()}
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := repo.GetByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Username != "alice" {
		t.Fatalf("unexpected username")
	}
}
