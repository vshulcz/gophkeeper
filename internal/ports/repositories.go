package ports

import (
	"context"
	"gophkeeper/internal/domain/secret"
	"gophkeeper/internal/domain/user"
	"time"

	"github.com/google/uuid"
)

// UserRepository persists users.
type UserRepository interface {
	Create(ctx context.Context, user user.User) error
	GetByUsername(ctx context.Context, username string) (user.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (user.User, error)
}

// ItemRepository persists encrypted items.
type ItemRepository interface {
	Upsert(ctx context.Context, item secret.Item) (secret.Item, error)
	GetByID(ctx context.Context, ownerID, itemID uuid.UUID) (secret.Item, error)
	ListUpdatedSince(ctx context.Context, ownerID uuid.UUID, since int64) ([]secret.Item, error)
	Delete(ctx context.Context, ownerID, itemID uuid.UUID, version int64, updatedAt time.Time) (secret.Item, error)
}
