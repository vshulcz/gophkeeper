package domain

import "errors"

// Common domain errors.
var (
	// ErrNotFound is returned when an entity is not found.
	ErrNotFound = errors.New("not found")
	// ErrConflict is returned when a uniqueness conflict occurs.
	ErrConflict = errors.New("conflict")
	// ErrUnauthorized is returned when authentication or authorization fails.
	ErrUnauthorized = errors.New("unauthorized")
	// ErrInvalidItemType is returned when item type is not supported.
	ErrInvalidItemType = errors.New("invalid item type")
)
