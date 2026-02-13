// Package dto defines HTTP transport objects for the API.
package dto

import "time"

// CredentialsRequest is the auth request payload.
type CredentialsRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// TokenResponse is the auth response payload.
type TokenResponse struct {
	Token            string `json:"token"`
	ExpiresInSeconds int64  `json:"expires_in_seconds"`
}

// ItemUpsertRequest is the item upsert request.
type ItemUpsertRequest struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Payload []byte `json:"payload"`
	Version int64  `json:"version"`
}

// ItemResponse is the item response payload.
type ItemResponse struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Payload   []byte    `json:"payload"`
	Deleted   bool      `json:"deleted"`
	Version   int64     `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
}
