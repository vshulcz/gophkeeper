package client

import (
	"context"
	"encoding/json"
	"errors"
	"gophkeeper/internal/domain/secret"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Service orchestrates client use cases.
type Service struct {
	api    API
	store  Store
	crypto Crypto
	clock  func() time.Time
	mu     sync.Mutex
}

// NewService creates a Service.
func NewService(api API, store Store, crypto Crypto, clock func() time.Time) *Service {
	return &Service{api: api, store: store, crypto: crypto, clock: clock}
}

// Register registers a new user and persists local state.
func (s *Service) Register(ctx context.Context, username, password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	token, err := s.api.Register(ctx, username, password)
	if err != nil {
		return err
	}
	state, err := s.store.Load(ctx)
	if err != nil {
		return err
	}
	state.Username = username
	state.Token = token
	if _, err := state.EnsureSalt(s.crypto); err != nil {
		return err
	}
	return s.store.Save(ctx, state)
}

// Login logs in existing user and persists local state.
func (s *Service) Login(ctx context.Context, username, password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	token, err := s.api.Login(ctx, username, password)
	if err != nil {
		return err
	}
	state, err := s.store.Load(ctx)
	if err != nil {
		return err
	}
	state.Username = username
	state.Token = token
	if _, err := state.EnsureSalt(s.crypto); err != nil {
		return err
	}
	return s.store.Save(ctx, state)
}

// AddInput describes new item data for add use case.
type AddInput struct {
	Value        secret.Value
	Meta         map[string]string
	UserPassword string
}

// Add stores a new encrypted item and updates local cache.
func (s *Service) Add(ctx context.Context, input AddInput) (uuid.UUID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	resp, err := s.upsert(ctx, uuid.New(), input.Value, input.Meta, input.UserPassword)
	if err != nil {
		return uuid.Nil, err
	}
	return resp.ID, nil
}

// UpdateInput describes item data for update use case.
type UpdateInput struct {
	ID           uuid.UUID
	Value        secret.Value
	Meta         map[string]string
	UserPassword string
}

// Update overwrites an existing encrypted item and updates local cache.
func (s *Service) Update(ctx context.Context, input UpdateInput) (uuid.UUID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if input.ID == uuid.Nil {
		return uuid.Nil, errors.New("id required")
	}
	resp, err := s.upsert(ctx, input.ID, input.Value, input.Meta, input.UserPassword)
	if err != nil {
		return uuid.Nil, err
	}
	return resp.ID, nil
}

// Delete removes an item by ID and updates local cache.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.store.Load(ctx)
	if err != nil {
		return err
	}
	if err := state.RequireToken(); err != nil {
		return err
	}
	if id == uuid.Nil {
		return errors.New("id required")
	}
	version := s.clock().UnixNano()
	resp, err := s.api.DeleteItem(ctx, state.Token, id, version)
	if err != nil {
		if isAuthError(err) {
			_ = s.clearToken(ctx)
			return ErrSessionExpired
		}
		if isConflictError(err) {
			return errors.Join(ErrConflict, err)
		}
		return err
	}
	state.RemoveItem(resp.ID)
	if resp.Version > state.LastVersion {
		state.LastVersion = resp.Version
	}
	return s.store.Save(ctx, state)
}

func (s *Service) upsert(ctx context.Context, id uuid.UUID, value secret.Value, meta map[string]string, userPassword string) (ItemResponse, error) {
	state, err := s.store.Load(ctx)
	if err != nil {
		return ItemResponse{}, err
	}
	if err := state.RequireToken(); err != nil {
		return ItemResponse{}, err
	}
	ciphertext, err := s.encryptPayload(state, value, meta, userPassword)
	if err != nil {
		return ItemResponse{}, err
	}

	version := s.clock().UnixNano()
	resp, err := s.api.UpsertItem(ctx, state.Token, ItemUpsertRequest{
		ID:      id,
		Type:    value.Type(),
		Payload: ciphertext,
		Version: version,
	})
	if err != nil {
		if isAuthError(err) {
			_ = s.clearToken(ctx)
			return ItemResponse{}, ErrSessionExpired
		}
		if isConflictError(err) {
			return ItemResponse{}, errors.Join(ErrConflict, err)
		}
		return ItemResponse{}, err
	}

	if resp.Deleted {
		state.RemoveItem(resp.ID)
	} else {
		state.UpdateOrInsertItem(LocalItemFromResponse(resp))
	}
	if resp.Version > state.LastVersion {
		state.LastVersion = resp.Version
	}
	if err := s.store.Save(ctx, state); err != nil {
		return ItemResponse{}, err
	}
	return resp, nil
}

func (s *Service) encryptPayload(state *StoreState, value secret.Value, meta map[string]string, userPassword string) ([]byte, error) {
	if userPassword == "" {
		return nil, errors.New("user password required")
	}
	if value == nil {
		return nil, errors.New("secret value required")
	}
	if err := value.Validate(); err != nil {
		return nil, err
	}
	salt, err := state.EnsureSalt(s.crypto)
	if err != nil {
		return nil, err
	}
	key, err := s.crypto.DeriveKey(userPassword, salt)
	if err != nil {
		return nil, err
	}
	env := NewPayloadEnvelope(value.Type(), meta, value.Data())
	plain, err := json.Marshal(env)
	if err != nil {
		return nil, err
	}
	return s.crypto.Encrypt(key, plain)
}

// ListItem is a summary of cached item.
type ListItem struct {
	ID      uuid.UUID
	Type    secret.Type
	Version int64
	Meta    map[string]string
	Err     error
}

// List returns cached items, optionally decrypting metadata.
func (s *Service) List(ctx context.Context, userPassword string) ([]ListItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.store.Load(ctx)
	if err != nil {
		return nil, err
	}
	if len(state.Items) == 0 {
		return nil, nil
	}

	var key []byte
	if userPassword != "" {
		salt, err := state.EnsureSalt(s.crypto)
		if err != nil {
			return nil, err
		}
		key, err = s.crypto.DeriveKey(userPassword, salt)
		if err != nil {
			return nil, err
		}
	}

	items := make([]ListItem, 0, len(state.Items))
	for _, item := range state.Items {
		entry := ListItem{ID: item.ID, Type: item.Type, Version: item.Version}
		if key != nil {
			plain, err := s.crypto.Decrypt(key, item.Payload)
			if err != nil {
				entry.Err = err
				items = append(items, entry)
				continue
			}
			var env PayloadEnvelope
			if err := json.Unmarshal(plain, &env); err != nil {
				entry.Err = err
				items = append(items, entry)
				continue
			}
			entry.Meta = ensureMetaFromEnvelope(env.Meta, env)
		}
		items = append(items, entry)
	}
	return items, nil
}

// IsLoggedIn reports whether the client has an active session.
func (s *Service) IsLoggedIn(ctx context.Context) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.store.Load(ctx)
	if err != nil {
		return false, err
	}
	return state.Token != "", nil
}

// Logout clears the session and cached secrets.
func (s *Service) Logout(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.store.Load(ctx)
	if err != nil {
		return err
	}
	state.Token = ""
	state.Items = nil
	state.LastVersion = 0
	return s.store.Save(ctx, state)
}

// Get returns a decrypted item payload.
func (s *Service) Get(ctx context.Context, id uuid.UUID, userPassword string) (PayloadEnvelope, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.store.Load(ctx)
	if err != nil {
		return PayloadEnvelope{}, err
	}
	if userPassword == "" {
		return PayloadEnvelope{}, errors.New("user password required")
	}
	var item *LocalItem
	for i := range state.Items {
		if state.Items[i].ID == id {
			item = &state.Items[i]
			break
		}
	}
	if item == nil {
		return PayloadEnvelope{}, errors.New("item not found in local store, run sync")
	}
	itemVal := *item

	salt, err := state.EnsureSalt(s.crypto)
	if err != nil {
		return PayloadEnvelope{}, err
	}
	key, err := s.crypto.DeriveKey(userPassword, salt)
	if err != nil {
		return PayloadEnvelope{}, err
	}
	plain, err := s.crypto.Decrypt(key, itemVal.Payload)
	if err != nil {
		return PayloadEnvelope{}, err
	}
	var env PayloadEnvelope
	if err := json.Unmarshal(plain, &env); err != nil {
		return PayloadEnvelope{}, err
	}
	return env, nil
}

// Sync pulls updates from server and updates local cache.
func (s *Service) Sync(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.store.Load(ctx)
	if err != nil {
		return 0, err
	}
	if err := state.RequireToken(); err != nil {
		return 0, err
	}
	items, err := s.api.SyncItems(ctx, state.Token, state.LastVersion)
	if err != nil {
		if isAuthError(err) {
			_ = s.clearToken(ctx)
			return 0, ErrSessionExpired
		}
		return 0, err
	}
	for _, item := range items {
		if item.Deleted {
			state.RemoveItem(item.ID)
		} else {
			state.UpdateOrInsertItem(LocalItemFromResponse(item))
		}
		if item.Version > state.LastVersion {
			state.LastVersion = item.Version
		}
	}
	if err := s.store.Save(ctx, state); err != nil {
		return 0, err
	}
	return len(items), nil
}

func (s *Service) clearToken(ctx context.Context) error {
	state, err := s.store.Load(ctx)
	if err != nil {
		return err
	}
	state.Token = ""
	return s.store.Save(ctx, state)
}

func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unauthorized") ||
		strings.Contains(msg, "not logged in") ||
		strings.Contains(msg, "invalid token") ||
		strings.Contains(msg, "missing token") ||
		strings.Contains(msg, "expired token")
}

func isConflictError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "conflict")
}

// ItemUpsertRequest mirrors server payload.
type ItemUpsertRequest struct {
	ID      uuid.UUID   `json:"id"`
	Type    secret.Type `json:"type"`
	Payload []byte      `json:"payload"`
	Version int64       `json:"version"`
}

// ItemResponse mirrors server response.
type ItemResponse struct {
	ID        uuid.UUID   `json:"id"`
	Type      secret.Type `json:"type"`
	Payload   []byte      `json:"payload"`
	Deleted   bool        `json:"deleted"`
	Version   int64       `json:"version"`
	UpdatedAt time.Time   `json:"updated_at"`
}

func ensureMetaFromEnvelope(meta map[string]string, env PayloadEnvelope) map[string]string {
	if meta == nil {
		meta = map[string]string{}
	}
	if hasDisplayMeta(meta) {
		return meta
	}
	switch env.Type {
	case secret.TypeLoginPassword:
		if v := env.Data["login"]; v != "" {
			meta["label"] = v
		}
	case secret.TypeText:
		if v := env.Data["text"]; v != "" {
			meta["label"] = truncateText(v, 24)
		}
	case secret.TypeCard:
		if v := env.Data["number"]; v != "" {
			meta["label"] = "card ••" + last4(v)
		}
	case secret.TypeBinary:
		meta["label"] = "binary"
	}
	return meta
}

func hasDisplayMeta(meta map[string]string) bool {
	for _, key := range []string{"label", "title", "name", "site"} {
		if strings.TrimSpace(meta[key]) != "" {
			return true
		}
	}
	return false
}

func truncateText(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen < 4 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func last4(s string) string {
	if len(s) <= 4 {
		return s
	}
	return s[len(s)-4:]
}
