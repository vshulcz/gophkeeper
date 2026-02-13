package persistence

import (
	"errors"
	"gophkeeper/internal/domain"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestItemRepo_InvalidType(t *testing.T) {
	db, ctx := openTestDB(t)
	repo := NewItemRepo(db)

	owner := uuid.New()
	id := uuid.New()
	_, err := db.ExecContext(ctx, `INSERT INTO items (id, owner_id, type, payload, deleted, version, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id.String(), owner.String(), "bad", []byte("x"), 0, 1, time.Now().Unix())
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	_, err = repo.GetByID(ctx, owner, id)
	if !errors.Is(err, domain.ErrInvalidItemType) {
		t.Fatalf("expected invalid type, got %v", err)
	}
}
