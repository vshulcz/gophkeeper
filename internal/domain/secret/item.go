package secret

import (
	"time"

	"github.com/google/uuid"
)

// Item represents a stored encrypted secret.
type Item struct {
	ID        uuid.UUID
	OwnerID   uuid.UUID
	Type      Type
	Payload   []byte // encrypted blob
	Deleted   bool
	Version   int64
	UpdatedAt time.Time
}
