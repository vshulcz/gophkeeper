package client

import (
	"context"
	"encoding/base64"
	"errors"
	"gophkeeper/internal/domain/secret"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeAPI struct {
	registerToken string
	loginToken    string
	items         []ItemResponse
}

func (f *fakeAPI) Register(_ context.Context, _, _ string) (string, error) {
	return f.registerToken, nil
}
func (f *fakeAPI) Login(_ context.Context, _, _ string) (string, error) { return f.loginToken, nil }
func (f *fakeAPI) UpsertItem(_ context.Context, _ string, req ItemUpsertRequest) (ItemResponse, error) {
	item := ItemResponse{
		ID:        req.ID,
		Type:      req.Type,
		Payload:   req.Payload,
		Deleted:   false,
		Version:   req.Version,
		UpdatedAt: time.Now(),
	}
	f.items = append(f.items, item)
	return item, nil
}

func (f *fakeAPI) GetItem(_ context.Context, _ string, id uuid.UUID) (ItemResponse, error) {
	for _, item := range f.items {
		if item.ID == id {
			return item, nil
		}
	}
	return ItemResponse{}, errors.New("not found")
}

func (f *fakeAPI) SyncItems(_ context.Context, _ string, _ int64) ([]ItemResponse, error) {
	return f.items, nil
}

func (f *fakeAPI) DeleteItem(_ context.Context, _ string, id uuid.UUID, version int64) (ItemResponse, error) {
	for i := range f.items {
		if f.items[i].ID == id {
			f.items[i].Deleted = true
			f.items[i].Version = version
			return f.items[i], nil
		}
	}
	return ItemResponse{ID: id, Deleted: true, Version: version, UpdatedAt: time.Now()}, nil
}

type memoryStore struct {
	state *StoreState
}

func (m *memoryStore) Load(_ context.Context) (*StoreState, error) {
	if m.state == nil {
		m.state = &StoreState{}
	}
	return m.state, nil
}

func (m *memoryStore) Save(_ context.Context, state *StoreState) error {
	m.state = state
	return nil
}

type fakeCrypto struct{}

func (fakeCrypto) DeriveKey(_ string, _ []byte) ([]byte, error) { return []byte("key"), nil }
func (fakeCrypto) Encrypt(_, plaintext []byte) ([]byte, error)  { return plaintext, nil }
func (fakeCrypto) Decrypt(_, ciphertext []byte) ([]byte, error) { return ciphertext, nil }
func (fakeCrypto) RandomBytes(n int) ([]byte, error)            { return make([]byte, n), nil }

type corruptCrypto struct{ fakeCrypto }

func (corruptCrypto) Decrypt(_, _ []byte) ([]byte, error) { return []byte("{"), nil }

type slowAPI struct{}

func (slowAPI) Register(context.Context, string, string) (string, error) {
	return "", nil
}
func (slowAPI) Login(context.Context, string, string) (string, error) { return "", nil }
func (slowAPI) UpsertItem(context.Context, string, ItemUpsertRequest) (ItemResponse, error) {
	return ItemResponse{}, errors.New("not implemented")
}

func (slowAPI) GetItem(context.Context, string, uuid.UUID) (ItemResponse, error) {
	return ItemResponse{}, errors.New("not implemented")
}

func (slowAPI) SyncItems(ctx context.Context, _ string, _ int64) ([]ItemResponse, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func (slowAPI) DeleteItem(context.Context, string, uuid.UUID, int64) (ItemResponse, error) {
	return ItemResponse{}, errors.New("not implemented")
}

type authErrorAPI struct{}

func (authErrorAPI) Register(context.Context, string, string) (string, error) { return "", nil }
func (authErrorAPI) Login(context.Context, string, string) (string, error)    { return "", nil }
func (authErrorAPI) UpsertItem(context.Context, string, ItemUpsertRequest) (ItemResponse, error) {
	return ItemResponse{}, errors.New("server error: invalid token")
}

func (authErrorAPI) GetItem(context.Context, string, uuid.UUID) (ItemResponse, error) {
	return ItemResponse{}, errors.New("server error: invalid token")
}

func (authErrorAPI) SyncItems(context.Context, string, int64) ([]ItemResponse, error) {
	return nil, errors.New("server error: invalid token")
}

func (authErrorAPI) DeleteItem(context.Context, string, uuid.UUID, int64) (ItemResponse, error) {
	return ItemResponse{}, errors.New("server error: invalid token")
}

type conflictAPI struct{}

func (conflictAPI) Register(context.Context, string, string) (string, error) { return "", nil }
func (conflictAPI) Login(context.Context, string, string) (string, error)    { return "", nil }
func (conflictAPI) UpsertItem(context.Context, string, ItemUpsertRequest) (ItemResponse, error) {
	return ItemResponse{}, errors.New("conflict: stale version")
}

func (conflictAPI) GetItem(context.Context, string, uuid.UUID) (ItemResponse, error) {
	return ItemResponse{}, errors.New("not implemented")
}

func (conflictAPI) SyncItems(context.Context, string, int64) ([]ItemResponse, error) {
	return nil, errors.New("not implemented")
}

func (conflictAPI) DeleteItem(context.Context, string, uuid.UUID, int64) (ItemResponse, error) {
	return ItemResponse{}, errors.New("conflict: stale version")
}

func TestService_RegisterLoginAndAdd(t *testing.T) {
	api := &fakeAPI{registerToken: "t1", loginToken: "t2"}
	store := &memoryStore{}
	crypto := fakeCrypto{}
	svc := NewService(api, store, crypto, time.Now)

	if err := svc.Register(context.Background(), "alice", "pass"); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := svc.Login(context.Background(), "alice", "pass"); err != nil {
		t.Fatalf("login: %v", err)
	}

	value := secret.Text{Text: "hello"}
	id, err := svc.Add(context.Background(), AddInput{Value: value, Meta: map[string]string{"k": "v"}, UserPassword: "pass"})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if id == uuid.Nil {
		t.Fatalf("expected id")
	}
}

func TestService_ListAndGet(t *testing.T) {
	api := &fakeAPI{registerToken: "t1"}
	store := &memoryStore{}
	crypto := fakeCrypto{}
	svc := NewService(api, store, crypto, time.Now)

	_ = svc.Register(context.Background(), "alice", "pass")
	value := secret.Text{Text: "hello"}
	id, _ := svc.Add(context.Background(), AddInput{Value: value, Meta: map[string]string{"k": "v"}, UserPassword: "pass"})

	items, err := svc.List(context.Background(), "pass")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item")
	}

	env, err := svc.Get(context.Background(), id, "pass")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if env.Type != secret.TypeText {
		t.Fatalf("unexpected type")
	}
}

func TestService_Sync(t *testing.T) {
	api := &fakeAPI{registerToken: "t1"}
	store := &memoryStore{}
	crypto := fakeCrypto{}
	svc := NewService(api, store, crypto, time.Now)

	_ = svc.Register(context.Background(), "alice", "pass")
	value := secret.Text{Text: "hello"}
	_, _ = svc.Add(context.Background(), AddInput{Value: value, Meta: map[string]string{"k": "v"}, UserPassword: "pass"})

	count, err := svc.Sync(context.Background())
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 item")
	}
}

func TestService_UpdateAndDelete(t *testing.T) {
	api := &fakeAPI{registerToken: "t1"}
	store := &memoryStore{}
	crypto := fakeCrypto{}
	svc := NewService(api, store, crypto, time.Now)

	_ = svc.Register(context.Background(), "alice", "pass")
	value := secret.Text{Text: "hello"}
	id, _ := svc.Add(context.Background(), AddInput{Value: value, Meta: map[string]string{"k": "v"}, UserPassword: "pass"})

	if _, err := svc.Update(context.Background(), UpdateInput{ID: id, Value: secret.Text{Text: "new"}, Meta: map[string]string{}, UserPassword: "pass"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := svc.Delete(context.Background(), id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	items, err := svc.List(context.Background(), "pass")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items after delete")
	}
}

func TestService_ListCorruptedPayload(t *testing.T) {
	itemID := uuid.New()
	state := &StoreState{
		Salt:  base64.StdEncoding.EncodeToString([]byte("1234567890abcdef")),
		Items: []LocalItem{{ID: itemID, Type: secret.TypeText, Payload: []byte("cipher"), Version: 1}},
	}
	store := &memoryStore{state: state}
	api := &fakeAPI{}
	crypto := corruptCrypto{}
	svc := NewService(api, store, crypto, time.Now)

	items, err := svc.List(context.Background(), "pass")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item")
	}
	if items[0].Err == nil {
		t.Fatalf("expected error for corrupted payload")
	}
}

func TestService_SyncTimeout(t *testing.T) {
	store := &memoryStore{state: &StoreState{Token: "t"}}
	svc := NewService(slowAPI{}, store, fakeCrypto{}, time.Now)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err := svc.Sync(ctx)
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline error, got %v", err)
	}
}

func TestService_AuthErrorClearsToken(t *testing.T) {
	store := &memoryStore{state: &StoreState{Token: "t"}}
	svc := NewService(authErrorAPI{}, store, fakeCrypto{}, time.Now)

	_, err := svc.Sync(context.Background())
	if err == nil || !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("expected session expired error, got %v", err)
	}
	if store.state.Token != "" {
		t.Fatalf("expected token cleared")
	}
}

func TestService_AuthErrorOnAddReturnsSessionExpired(t *testing.T) {
	store := &memoryStore{state: &StoreState{Token: "t"}}
	svc := NewService(authErrorAPI{}, store, fakeCrypto{}, time.Now)

	_, err := svc.Add(context.Background(), AddInput{Value: secret.Text{Text: "hello"}, UserPassword: "pass"})
	if err == nil || !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("expected session expired error, got %v", err)
	}
	if store.state.Token != "" {
		t.Fatalf("expected token cleared")
	}
}

func TestService_ConflictOnAddReturnsConflict(t *testing.T) {
	store := &memoryStore{state: &StoreState{Token: "t"}}
	svc := NewService(conflictAPI{}, store, fakeCrypto{}, time.Now)

	_, err := svc.Add(context.Background(), AddInput{Value: secret.Text{Text: "hello"}, UserPassword: "pass"})
	if err == nil || !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict error, got %v", err)
	}
}

func TestService_ConcurrentSync(t *testing.T) {
	api := &fakeAPI{items: []ItemResponse{
		{ID: uuid.New(), Type: secret.TypeText, Payload: []byte("x"), Version: 10, UpdatedAt: time.Now()},
		{ID: uuid.New(), Type: secret.TypeText, Payload: []byte("y"), Version: 20, UpdatedAt: time.Now()},
	}}
	store := &memoryStore{state: &StoreState{Token: "t"}}
	svc := NewService(api, store, fakeCrypto{}, time.Now)

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = svc.Sync(context.Background())
		}()
	}
	wg.Wait()

	if store.state.LastVersion != 20 {
		t.Fatalf("expected last version 20, got %d", store.state.LastVersion)
	}
	if len(store.state.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(store.state.Items))
	}
}
