package persistence

import (
	"context"
	"database/sql"
	"errors"
	"gophkeeper/internal/domain"
	"gophkeeper/internal/domain/secret"
	"time"

	"github.com/google/uuid"
)

// ItemRepo is a SQLite-backed repository for items.
type ItemRepo struct {
	db *sql.DB
}

// NewItemRepo creates an ItemRepo.
func NewItemRepo(db *sql.DB) *ItemRepo {
	return &ItemRepo{db: db}
}

// Upsert inserts or updates an item.
func (r *ItemRepo) Upsert(ctx context.Context, item secret.Item) (secret.Item, error) {
	var existingVersion int64
	row := r.db.QueryRowContext(ctx,
		`SELECT version FROM items WHERE id = ? AND owner_id = ?`,
		item.ID.String(), item.OwnerID.String())
	if err := row.Scan(&existingVersion); err == nil {
		if item.Version <= existingVersion {
			return secret.Item{}, domain.ErrConflict
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return secret.Item{}, err
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO items (id, owner_id, type, payload, deleted, version, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id, owner_id) DO UPDATE SET
		 type=excluded.type,
		 payload=excluded.payload,
		 deleted=excluded.deleted,
		 version=excluded.version,
		 updated_at=excluded.updated_at`,
		item.ID.String(), item.OwnerID.String(), string(item.Type), item.Payload, boolToInt(item.Deleted), item.Version, item.UpdatedAt.Unix())
	if err != nil {
		return secret.Item{}, err
	}
	return item, nil
}

// GetByID fetches an item by id and owner.
func (r *ItemRepo) GetByID(ctx context.Context, ownerID, itemID uuid.UUID) (secret.Item, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, owner_id, type, payload, deleted, version, updated_at FROM items WHERE id = ? AND owner_id = ?`,
		itemID.String(), ownerID.String())
	var id, owner, typ string
	var payload []byte
	var deleted int
	var version int64
	var updatedAt int64
	if err := row.Scan(&id, &owner, &typ, &payload, &deleted, &version, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return secret.Item{}, domain.ErrNotFound
		}
		return secret.Item{}, err
	}
	itemUUID, err := uuid.Parse(id)
	if err != nil {
		return secret.Item{}, err
	}
	ownerUUID, err := uuid.Parse(owner)
	if err != nil {
		return secret.Item{}, err
	}
	typVal, err := secret.ParseType(typ)
	if err != nil {
		return secret.Item{}, err
	}
	return secret.Item{
		ID:        itemUUID,
		OwnerID:   ownerUUID,
		Type:      typVal,
		Payload:   payload,
		Deleted:   deleted == 1,
		Version:   version,
		UpdatedAt: time.Unix(updatedAt, 0),
	}, nil
}

// ListUpdatedSince returns all items updated after a given version.
func (r *ItemRepo) ListUpdatedSince(ctx context.Context, ownerID uuid.UUID, since int64) ([]secret.Item, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, owner_id, type, payload, deleted, version, updated_at FROM items WHERE owner_id = ? AND version > ? ORDER BY version ASC`,
		ownerID.String(), since)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var items []secret.Item
	for rows.Next() {
		var id, owner, typ string
		var payload []byte
		var deleted int
		var version int64
		var updatedAt int64
		if err := rows.Scan(&id, &owner, &typ, &payload, &deleted, &version, &updatedAt); err != nil {
			return nil, err
		}
		itemUUID, err := uuid.Parse(id)
		if err != nil {
			return nil, err
		}
		ownerUUID, err := uuid.Parse(owner)
		if err != nil {
			return nil, err
		}
		typVal, err := secret.ParseType(typ)
		if err != nil {
			return nil, err
		}
		items = append(items, secret.Item{
			ID:        itemUUID,
			OwnerID:   ownerUUID,
			Type:      typVal,
			Payload:   payload,
			Deleted:   deleted == 1,
			Version:   version,
			UpdatedAt: time.Unix(updatedAt, 0),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// Delete marks an item as deleted with a new version.
func (r *ItemRepo) Delete(ctx context.Context, ownerID, itemID uuid.UUID, version int64, updatedAt time.Time) (secret.Item, error) {
	item, err := r.GetByID(ctx, ownerID, itemID)
	if err != nil {
		return secret.Item{}, err
	}
	item.Payload = []byte{}
	item.Deleted = true
	item.Version = version
	item.UpdatedAt = updatedAt
	return r.Upsert(ctx, item)
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
