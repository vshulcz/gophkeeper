package vault

import (
	"context"
	"gophkeeper/internal/domain"
	"gophkeeper/internal/domain/secret"
	"testing"
	"time"

	"github.com/google/uuid"
)

type memoryItemRepo struct {
	items map[string]secret.Item
}

func newMemoryItemRepo() *memoryItemRepo {
	return &memoryItemRepo{items: map[string]secret.Item{}}
}

func (m *memoryItemRepo) Upsert(_ context.Context, item secret.Item) (secret.Item, error) {
	key := item.OwnerID.String() + ":" + item.ID.String()
	m.items[key] = item
	return item, nil
}

func (m *memoryItemRepo) GetByID(_ context.Context, ownerID, itemID uuid.UUID) (secret.Item, error) {
	key := ownerID.String() + ":" + itemID.String()
	item, ok := m.items[key]
	if !ok {
		return secret.Item{}, domain.ErrNotFound
	}
	return item, nil
}

func (m *memoryItemRepo) ListUpdatedSince(_ context.Context, ownerID uuid.UUID, since int64) ([]secret.Item, error) {
	var items []secret.Item
	for _, item := range m.items {
		if item.OwnerID == ownerID && item.Version > since {
			items = append(items, item)
		}
	}
	return items, nil
}

func (m *memoryItemRepo) Delete(ctx context.Context, ownerID, itemID uuid.UUID, version int64, updatedAt time.Time) (secret.Item, error) {
	item, err := m.GetByID(ctx, ownerID, itemID)
	if err != nil {
		return secret.Item{}, err
	}
	item.Deleted = true
	item.Version = version
	item.UpdatedAt = updatedAt
	return m.Upsert(ctx, item)
}

func TestService_UpsertGetSync(t *testing.T) {
	repo := newMemoryItemRepo()
	svc := NewService(repo, func() time.Time { return time.Unix(1, 0) })

	owner := uuid.New()
	itemID := uuid.New()
	item, err := svc.Upsert(context.Background(), owner, itemID, secret.TypeText, []byte("data"), 2)
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if item.ID != itemID {
		t.Fatalf("unexpected id")
	}

	got, err := svc.Get(context.Background(), owner, itemID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Type != secret.TypeText {
		t.Fatalf("unexpected type")
	}

	items, err := svc.SyncSince(context.Background(), owner, 1)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item")
	}
}
