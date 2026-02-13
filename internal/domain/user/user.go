// Package user contains user aggregate root.
package user

import (
	"time"

	"github.com/google/uuid"
)

// User represents a registered account owner.
type User struct {
	ID           uuid.UUID
	Username     string
	PasswordHash []byte
	CreatedAt    time.Time
}
