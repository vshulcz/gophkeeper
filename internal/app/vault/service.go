package vault

import (
	"context"
	"gophkeeper/internal/domain"
	"gophkeeper/internal/domain/secret"
	"gophkeeper/internal/ports"

	"github.com/google/uuid"
)

// Service manages encrypted items.
type Service struct {
	items ports.ItemRepository
	clock ports.Clock
}

// NewService creates a vault service.
func NewService(items ports.ItemRepository, clock ports.Clock) *Service {
	return &Service{items: items, clock: clock}
}

// Upsert stores or updates an encrypted item.
func (s *Service) Upsert(ctx context.Context, ownerID, itemID uuid.UUID, itemType secret.Type, payload []byte, version int64) (secret.Item, error) {
	if version == 0 {
		version = s.clock().UnixNano()
	}
	item := secret.Item{
		ID:        itemID,
		OwnerID:   ownerID,
		Type:      itemType,
		Payload:   payload,
		Deleted:   false,
		Version:   version,
		UpdatedAt: s.clock(),
	}
	return s.items.Upsert(ctx, item)
}

// Get fetches a single encrypted item.
func (s *Service) Get(ctx context.Context, ownerID, itemID uuid.UUID) (secret.Item, error) {
	item, err := s.items.GetByID(ctx, ownerID, itemID)
	if err != nil {
		return secret.Item{}, err
	}
	if item.Deleted {
		return secret.Item{}, domain.ErrNotFound
	}
	return item, nil
}

// SyncSince returns items updated since the given version.
func (s *Service) SyncSince(ctx context.Context, ownerID uuid.UUID, since int64) ([]secret.Item, error) {
	return s.items.ListUpdatedSince(ctx, ownerID, since)
}

// Delete marks an item as deleted.
func (s *Service) Delete(ctx context.Context, ownerID, itemID uuid.UUID, version int64) (secret.Item, error) {
	if version == 0 {
		version = s.clock().UnixNano()
	}
	return s.items.Delete(ctx, ownerID, itemID, version, s.clock())
}
