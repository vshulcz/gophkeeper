package persistence

import (
	"errors"
	"gophkeeper/internal/domain"
	"gophkeeper/internal/domain/secret"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestItemRepo_GetNotFound(t *testing.T) {
	db, ctx := openTestDB(t)
	repo := NewItemRepo(db)
	_, err := repo.GetByID(ctx, uuid.New(), uuid.New())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestItemRepo_UpsertUpdateAndListSince(t *testing.T) {
	db, ctx := openTestDB(t)
	repo := NewItemRepo(db)

	owner := uuid.New()
	id := uuid.New()
	item := secret.Item{ID: id, OwnerID: owner, Type: secret.TypeText, Payload: []byte("a"), Version: 1, UpdatedAt: time.Now()}
	if _, err := repo.Upsert(ctx, item); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	item.Payload = []byte("b")
	item.Version = 2
	if _, err := repo.Upsert(ctx, item); err != nil {
		t.Fatalf("upsert2: %v", err)
	}
	items, err := repo.ListUpdatedSince(ctx, owner, 1)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item")
	}
}

func TestItemRepo_Delete(t *testing.T) {
	db, ctx := openTestDB(t)
	repo := NewItemRepo(db)

	owner := uuid.New()
	id := uuid.New()
	item := secret.Item{ID: id, OwnerID: owner, Type: secret.TypeText, Payload: []byte("a"), Version: 1, UpdatedAt: time.Now()}
	if _, err := repo.Upsert(ctx, item); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	deleted, err := repo.Delete(ctx, owner, id, 2, time.Now())
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if !deleted.Deleted {
		t.Fatalf("expected deleted")
	}

	items, err := repo.ListUpdatedSince(ctx, owner, 1)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 || !items[0].Deleted {
		t.Fatalf("expected deleted item in list")
	}
}
