package persistence

import (
	"gophkeeper/internal/domain/secret"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestItemRepo_GetByID(t *testing.T) {
	db, ctx := openTestDB(t)
	repo := NewItemRepo(db)

	owner := uuid.New()
	item := secret.Item{ID: uuid.New(), OwnerID: owner, Type: secret.TypeText, Payload: []byte("x"), Version: 1, UpdatedAt: time.Now()}
	if _, err := repo.Upsert(ctx, item); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	got, err := repo.GetByID(ctx, owner, item.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Version != 1 {
		t.Fatalf("unexpected version")
	}
}
