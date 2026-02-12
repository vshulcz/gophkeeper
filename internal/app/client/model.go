package client

import (
	"encoding/base64"
	"errors"
	"gophkeeper/internal/domain/secret"
	"time"

	"github.com/google/uuid"
)

// StoreState keeps client state and cached items.
type StoreState struct {
	Username    string      `json:"username"`
	Token       string      `json:"token"`
	Salt        string      `json:"salt"`
	LastVersion int64       `json:"last_version"`
	Items       []LocalItem `json:"items"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// LocalItem is a cached encrypted item.
type LocalItem struct {
	ID        uuid.UUID   `json:"id"`
	Type      secret.Type `json:"type"`
	Payload   []byte      `json:"payload"`
	Version   int64       `json:"version"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// PayloadEnvelope is encrypted and stored in the server.
type PayloadEnvelope struct {
	Type secret.Type       `json:"type"`
	Meta map[string]string `json:"meta"`
	Data map[string]string `json:"data"`
}

// NewPayloadEnvelope builds an envelope.
func NewPayloadEnvelope(typ secret.Type, meta, data map[string]string) PayloadEnvelope {
	return PayloadEnvelope{Type: typ, Meta: meta, Data: data}
}

// EncodePayload encodes binary payload to base64 for JSON.
func EncodePayload(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

// DecodePayload decodes base64 payload.
func DecodePayload(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// EnsureSalt sets a new salt if missing and returns bytes.
func (s *StoreState) EnsureSalt(rng Crypto) ([]byte, error) {
	if s.Salt != "" {
		return base64.StdEncoding.DecodeString(s.Salt)
	}
	salt, err := rng.RandomBytes(16)
	if err != nil {
		return nil, err
	}
	s.Salt = base64.StdEncoding.EncodeToString(salt)
	return salt, nil
}

// UpdateOrInsertItem updates local cache.
func (s *StoreState) UpdateOrInsertItem(item LocalItem) {
	for i := range s.Items {
		if s.Items[i].ID == item.ID {
			s.Items[i] = item
			return
		}
	}
	s.Items = append(s.Items, item)
}

// RemoveItem deletes an item from local cache.
func (s *StoreState) RemoveItem(id uuid.UUID) {
	for i := range s.Items {
		if s.Items[i].ID == id {
			s.Items = append(s.Items[:i], s.Items[i+1:]...)
			return
		}
	}
}

// LocalItemFromResponse converts API response to local item.
func LocalItemFromResponse(resp ItemResponse) LocalItem {
	return LocalItem{
		ID:        resp.ID,
		Type:      resp.Type,
		Payload:   resp.Payload,
		Version:   resp.Version,
		UpdatedAt: resp.UpdatedAt,
	}
}

// RequireToken validates that user is logged in.
func (s *StoreState) RequireToken() error {
	if s.Token == "" {
		return errors.New("not logged in")
	}
	return nil
}
