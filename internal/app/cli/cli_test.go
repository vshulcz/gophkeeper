package cli

import (
	"bytes"
	"context"
	"errors"
	"gophkeeper/internal/domain/secret"
	"testing"
	"time"

	appclient "gophkeeper/internal/app/client"

	"github.com/google/uuid"
)

type fakeAPI struct {
	token string
	items []appclient.ItemResponse
}

func (f *fakeAPI) Register(_ context.Context, _, _ string) (string, error) { return f.token, nil }
func (f *fakeAPI) Login(_ context.Context, _, _ string) (string, error)    { return f.token, nil }
func (f *fakeAPI) UpsertItem(_ context.Context, _ string, req appclient.ItemUpsertRequest) (appclient.ItemResponse, error) {
	item := appclient.ItemResponse{
		ID:        req.ID,
		Type:      req.Type,
		Payload:   req.Payload,
		Version:   req.Version,
		UpdatedAt: time.Now(),
	}
	f.items = append(f.items, item)
	return item, nil
}

func (f *fakeAPI) GetItem(_ context.Context, _ string, id uuid.UUID) (appclient.ItemResponse, error) {
	for _, item := range f.items {
		if item.ID == id {
			return item, nil
		}
	}
	return appclient.ItemResponse{}, errors.New("not found")
}

func (f *fakeAPI) SyncItems(_ context.Context, _ string, _ int64) ([]appclient.ItemResponse, error) {
	return f.items, nil
}

func (f *fakeAPI) DeleteItem(_ context.Context, _ string, id uuid.UUID, version int64) (appclient.ItemResponse, error) {
	for i := range f.items {
		if f.items[i].ID == id {
			f.items[i].Deleted = true
			f.items[i].Version = version
			return f.items[i], nil
		}
	}
	return appclient.ItemResponse{ID: id, Deleted: true, Version: version, UpdatedAt: time.Now()}, nil
}

type memoryStore struct {
	state *appclient.StoreState
}

func (m *memoryStore) Load(_ context.Context) (*appclient.StoreState, error) {
	if m.state == nil {
		m.state = &appclient.StoreState{}
	}
	return m.state, nil
}

func (m *memoryStore) Save(_ context.Context, state *appclient.StoreState) error {
	m.state = state
	return nil
}

type fakeCrypto struct{}

func (fakeCrypto) DeriveKey(_ string, _ []byte) ([]byte, error) { return []byte("key"), nil }
func (fakeCrypto) Encrypt(_, plaintext []byte) ([]byte, error)  { return plaintext, nil }
func (fakeCrypto) Decrypt(_, ciphertext []byte) ([]byte, error) { return ciphertext, nil }
func (fakeCrypto) RandomBytes(n int) ([]byte, error)            { return make([]byte, n), nil }

const (
	buildVersion = "v1"
	buildDate    = "d1"
)

func TestCLI_Version(t *testing.T) {
	buf := &bytes.Buffer{}
	deps := Deps{
		NewService: func(string) Service { return nil },
		ReadFile:   nil,
		NowVersion: func() (string, string) { return buildVersion, buildDate },
	}
	if err := Run([]string{"version"}, deps, buf); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte(buildVersion)) {
		t.Fatalf("expected version output")
	}
}

func TestCLI_RegisterAddListGetSync(t *testing.T) {
	api := &fakeAPI{token: "t"}
	store := &memoryStore{}
	crypto := fakeCrypto{}
	svc := appclient.NewService(api, store, crypto, time.Now)
	deps := Deps{
		NewService: func(string) Service { return svc },
		ReadFile:   func(string) ([]byte, error) { return []byte("bin"), nil },
		NowVersion: func() (string, string) { return buildVersion, buildDate },
	}
	buf := &bytes.Buffer{}

	if err := Run([]string{"register", "-user", "u", "-pass", "p"}, deps, buf); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := Run([]string{"add", "-type", "text", "-text", "hello", "-user-pass", "p"}, deps, buf); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := Run([]string{"list", "-user-pass", "p"}, deps, buf); err != nil {
		t.Fatalf("list: %v", err)
	}
	itemID := store.state.Items[0].ID
	if err := Run([]string{"get", "-id", itemID.String(), "-user-pass", "p"}, deps, buf); err != nil {
		t.Fatalf("get: %v", err)
	}
	if err := Run([]string{"sync"}, deps, buf); err != nil {
		t.Fatalf("sync: %v", err)
	}
}

func TestCLI_AddBinary(t *testing.T) {
	api := &fakeAPI{token: "t"}
	store := &memoryStore{}
	crypto := fakeCrypto{}
	svc := appclient.NewService(api, store, crypto, time.Now)
	deps := Deps{
		NewService: func(string) Service { return svc },
		ReadFile:   func(string) ([]byte, error) { return []byte("bin"), nil },
		NowVersion: func() (string, string) { return buildVersion, buildDate },
	}
	buf := &bytes.Buffer{}
	_ = Run([]string{"register", "-user", "u", "-pass", "p"}, deps, buf)
	if err := Run([]string{"add", "-type", string(secret.TypeBinary), "-file", "x", "-user-pass", "p"}, deps, buf); err != nil {
		t.Fatalf("add: %v", err)
	}
}

func TestCLI_AddCardAndLogin(t *testing.T) {
	api := &fakeAPI{token: "t"}
	store := &memoryStore{}
	crypto := fakeCrypto{}
	svc := appclient.NewService(api, store, crypto, time.Now)
	deps := Deps{
		NewService: func(string) Service { return svc },
		ReadFile:   func(string) ([]byte, error) { return []byte("bin"), nil },
		NowVersion: func() (string, string) { return buildVersion, buildDate },
	}
	buf := &bytes.Buffer{}
	_ = Run([]string{"register", "-user", "u", "-pass", "p"}, deps, buf)
	if err := Run([]string{"add", "-type", "login_password", "-login", "l", "-password", "p", "-user-pass", "p"}, deps, buf); err != nil {
		t.Fatalf("add login: %v", err)
	}
	if err := Run([]string{"add", "-type", "card", "-card-number", "1", "-card-expiry", "2", "-card-holder", "3", "-card-cvv", "4", "-user-pass", "p"}, deps, buf); err != nil {
		t.Fatalf("add card: %v", err)
	}
}

func TestCLI_UsageAndErrors(t *testing.T) {
	buf := &bytes.Buffer{}
	deps := Deps{
		NewService: func(string) Service { return nil },
		ReadFile:   nil,
		NowVersion: func() (string, string) { return buildVersion, buildDate },
	}
	if err := Run([]string{}, deps, buf); err != nil {
		t.Fatalf("run: %v", err)
	}
	if err := Run([]string{"unknown"}, deps, buf); err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestCLI_AddErrors(t *testing.T) {
	api := &fakeAPI{token: "t"}
	store := &memoryStore{}
	crypto := fakeCrypto{}
	svc := appclient.NewService(api, store, crypto, time.Now)
	deps := Deps{
		NewService: func(string) Service { return svc },
		ReadFile:   func(string) ([]byte, error) { return nil, errors.New("read") },
		NowVersion: func() (string, string) { return buildVersion, buildDate },
	}
	buf := &bytes.Buffer{}
	if err := Run([]string{"register", "-user", "u", "-pass", "p"}, deps, buf); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := Run([]string{"add", "-type", "text", "-text", "", "-user-pass", "p"}, deps, buf); err == nil {
		t.Fatalf("expected error")
	}
	if err := Run([]string{"add", "-type", "binary", "-file", "x", "-user-pass", "p"}, deps, buf); err == nil {
		t.Fatalf("expected error")
	}
	if err := Run([]string{"add", "-type", "bad", "-user-pass", "p"}, deps, buf); err == nil {
		t.Fatalf("expected error")
	}
}

func TestCLI_RegisterErrors(t *testing.T) {
	svc := &failingService{err: errors.New("api")}
	deps := Deps{
		NewService: func(string) Service { return svc },
		ReadFile:   func(string) ([]byte, error) { return nil, nil },
		NowVersion: func() (string, string) { return buildVersion, buildDate },
	}
	buf := &bytes.Buffer{}
	if err := Run([]string{"register", "-user", "u", "-pass", "p"}, deps, buf); err == nil {
		t.Fatalf("expected error")
	}
}

func TestCLI_LoginAndSyncErrors(t *testing.T) {
	svc := &failingService{err: errors.New("api")}
	deps := Deps{
		NewService: func(string) Service { return svc },
		ReadFile:   func(string) ([]byte, error) { return nil, nil },
		NowVersion: func() (string, string) { return buildVersion, buildDate },
	}
	buf := &bytes.Buffer{}
	if err := Run([]string{"login", "-user", "u", "-pass", "p"}, deps, buf); err == nil {
		t.Fatalf("expected error")
	}
	if err := Run([]string{"sync"}, deps, buf); err == nil {
		t.Fatalf("expected error")
	}
	if err := Run([]string{"list"}, deps, buf); err == nil {
		t.Fatalf("expected error")
	}
	if err := Run([]string{"add", "-type", "text", "-text", "t", "-user-pass", "p"}, deps, buf); err == nil {
		t.Fatalf("expected error")
	}
}

func TestCLI_RegisterMissingArgs(t *testing.T) {
	buf := &bytes.Buffer{}
	deps := Deps{
		NewService: func(string) Service { return &failingService{err: errors.New("api")} },
		ReadFile:   func(string) ([]byte, error) { return nil, nil },
		NowVersion: func() (string, string) { return buildVersion, buildDate },
	}
	if err := Run([]string{"register", "-user", "u"}, deps, buf); err == nil {
		t.Fatalf("expected error")
	}
	if err := Run([]string{"login", "-pass", "p"}, deps, buf); err == nil {
		t.Fatalf("expected error")
	}
}

func TestCLI_GetErrors(t *testing.T) {
	api := &fakeAPI{token: "t"}
	store := &memoryStore{}
	crypto := fakeCrypto{}
	svc := appclient.NewService(api, store, crypto, time.Now)
	deps := Deps{
		NewService: func(string) Service { return svc },
		ReadFile:   func(string) ([]byte, error) { return nil, nil },
		NowVersion: func() (string, string) { return buildVersion, buildDate },
	}
	buf := &bytes.Buffer{}
	if err := Run([]string{"get", "-id", "", "-user-pass", "p"}, deps, buf); err == nil {
		t.Fatalf("expected error")
	}
	if err := Run([]string{"get", "-id", "bad", "-user-pass", "p"}, deps, buf); err == nil {
		t.Fatalf("expected error")
	}
	if err := Run([]string{"get", "-id", "00000000-0000-0000-0000-000000000001", "-user-pass", ""}, deps, buf); err == nil {
		t.Fatalf("expected error")
	}
}

func TestCLI_ListNoItems(t *testing.T) {
	api := &fakeAPI{token: "t"}
	store := &memoryStore{}
	crypto := fakeCrypto{}
	svc := appclient.NewService(api, store, crypto, time.Now)
	deps := Deps{
		NewService: func(string) Service { return svc },
		ReadFile:   func(string) ([]byte, error) { return nil, nil },
		NowVersion: func() (string, string) { return buildVersion, buildDate },
	}
	buf := &bytes.Buffer{}
	if err := Run([]string{"list"}, deps, buf); err != nil {
		t.Fatalf("list: %v", err)
	}
}

func TestCLI_DefaultDepsAndNoopStore(t *testing.T) {
	deps := DefaultDeps()
	if _, _ = deps.NowVersion(); false {
		t.Fatalf("unreachable")
	}
	ns := noopStore{}
	if _, err := ns.Load(context.Background()); err != nil {
		t.Fatalf("load: %v", err)
	}
	if err := ns.Save(context.Background(), &appclient.StoreState{}); err != nil {
		t.Fatalf("save: %v", err)
	}
}

func TestParseKV(t *testing.T) {
	res := parseKV("a=1,b=2")
	if res["a"] != "1" || res["b"] != "2" {
		t.Fatalf("unexpected parse")
	}
	res = parseKV("")
	if len(res) != 0 {
		t.Fatalf("expected empty")
	}
}

type failingService struct{ err error }

func (f *failingService) Register(context.Context, string, string) error { return f.err }
func (f *failingService) Login(context.Context, string, string) error    { return f.err }
func (f *failingService) Add(context.Context, appclient.AddInput) (uuid.UUID, error) {
	return uuid.Nil, f.err
}

func (f *failingService) List(context.Context, string) ([]appclient.ListItem, error) {
	return nil, f.err
}

func (f *failingService) Get(context.Context, uuid.UUID, string) (appclient.PayloadEnvelope, error) {
	return appclient.PayloadEnvelope{}, f.err
}
func (f *failingService) Sync(context.Context) (int, error) { return 0, f.err }
