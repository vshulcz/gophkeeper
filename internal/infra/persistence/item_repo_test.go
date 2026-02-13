package persistence

import (
	"gophkeeper/internal/domain/secret"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestItemRepo_UpsertAndList(t *testing.T) {
	db, ctx := openTestDB(t)
	repo := NewItemRepo(db)

	owner := uuid.New()
	item := secret.Item{
		ID:        uuid.New(),
		OwnerID:   owner,
		Type:      secret.TypeText,
		Payload:   []byte("secret"),
		Version:   1,
		UpdatedAt: time.Now(),
	}
	if _, err := repo.Upsert(ctx, item); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	items, err := repo.ListUpdatedSince(ctx, owner, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
}
