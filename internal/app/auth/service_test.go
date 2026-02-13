package auth

import (
	"context"
	"errors"
	"gophkeeper/internal/domain"
	"gophkeeper/internal/domain/user"
	"testing"
	"time"

	"github.com/google/uuid"
)

type memoryUserRepo struct {
	byID       map[string]user.User
	byUsername map[string]user.User
}

func newMemoryUserRepo() *memoryUserRepo {
	return &memoryUserRepo{byID: map[string]user.User{}, byUsername: map[string]user.User{}}
}

func (m *memoryUserRepo) Create(_ context.Context, user user.User) error {
	if _, ok := m.byUsername[user.Username]; ok {
		return domain.ErrConflict
	}
	m.byID[user.ID.String()] = user
	m.byUsername[user.Username] = user
	return nil
}

func (m *memoryUserRepo) GetByUsername(_ context.Context, username string) (user.User, error) {
	if user, ok := m.byUsername[username]; ok {
		return user, nil
	}
	return user.User{}, domain.ErrNotFound
}

func (m *memoryUserRepo) GetByID(_ context.Context, id uuid.UUID) (user.User, error) {
	if user, ok := m.byID[id.String()]; ok {
		return user, nil
	}
	return user.User{}, domain.ErrNotFound
}

type fakeHasher struct{}

func (fakeHasher) Hash(password string) ([]byte, error) { return []byte("hash:" + password), nil }
func (fakeHasher) Compare(hash []byte, password string) error {
	if string(hash) != "hash:"+password {
		return domain.ErrUnauthorized
	}
	return nil
}

type fakeSigner struct{}

func (fakeSigner) Sign(userID string, _ time.Time) (string, error) { return "token:" + userID, nil }
func (fakeSigner) Verify(token string) (string, time.Time, error)  { return token, time.Now(), nil }

func TestService_RegisterAndLogin(t *testing.T) {
	repo := newMemoryUserRepo()
	svc := NewService(repo, fakeHasher{}, fakeSigner{}, func() time.Time { return time.Unix(1, 0) }, 24*time.Hour)

	tok, err := svc.Register(context.Background(), "alice", "secret")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if tok == "" {
		t.Fatal("empty token")
	}

	tok, err = svc.Login(context.Background(), "alice", "secret")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if tok == "" {
		t.Fatal("empty token")
	}
}

func TestService_Conflict(t *testing.T) {
	repo := newMemoryUserRepo()
	svc := NewService(repo, fakeHasher{}, fakeSigner{}, time.Now, 24*time.Hour)

	_, err := svc.Register(context.Background(), "bob", "secret")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	_, err = svc.Register(context.Background(), "bob", "secret")
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}
