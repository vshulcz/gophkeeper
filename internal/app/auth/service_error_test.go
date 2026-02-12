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

type errorUserRepo struct{}

func (errorUserRepo) Create(_ context.Context, _ user.User) error { return errors.New("db") }
func (errorUserRepo) GetByUsername(_ context.Context, _ string) (user.User, error) {
	return user.User{}, errors.New("db")
}

func (errorUserRepo) GetByID(_ context.Context, _ uuid.UUID) (user.User, error) {
	return user.User{}, errors.New("db")
}

type errorHasher struct{}

func (errorHasher) Hash(_ string) ([]byte, error)    { return nil, errors.New("hash") }
func (errorHasher) Compare(_ []byte, _ string) error { return errors.New("bad") }

type okSigner struct{}

func (okSigner) Sign(_ string, _ time.Time) (string, error) { return "t", nil }
func (okSigner) Verify(_ string) (string, time.Time, error) { return "", time.Time{}, nil }

func TestService_RegisterErrors(t *testing.T) {
	svc := NewService(errorUserRepo{}, errorHasher{}, okSigner{}, time.Now, 24*time.Hour)
	if _, err := svc.Register(context.Background(), "u", "p"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestService_LoginUnauthorized(t *testing.T) {
	repo := newMemoryUserRepo()
	svc := NewService(repo, fakeHasher{}, okSigner{}, time.Now, 24*time.Hour)
	_, _ = svc.Register(context.Background(), "u", "p")
	_, err := svc.Login(context.Background(), "u", "wrong")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected unauthorized")
	}
}
