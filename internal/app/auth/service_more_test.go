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

type notFoundUserRepo struct{}

func (notFoundUserRepo) Create(_ context.Context, _ user.User) error { return nil }
func (notFoundUserRepo) GetByUsername(_ context.Context, _ string) (user.User, error) {
	return user.User{}, domain.ErrNotFound
}

func (notFoundUserRepo) GetByID(_ context.Context, _ uuid.UUID) (user.User, error) {
	return user.User{}, domain.ErrNotFound
}

type createErrorRepo struct{}

func (createErrorRepo) Create(_ context.Context, _ user.User) error { return errors.New("create") }
func (createErrorRepo) GetByUsername(_ context.Context, _ string) (user.User, error) {
	return user.User{}, domain.ErrNotFound
}

func (createErrorRepo) GetByID(_ context.Context, _ uuid.UUID) (user.User, error) {
	return user.User{}, domain.ErrNotFound
}

type hashErrorHasher struct{}

func (hashErrorHasher) Hash(_ string) ([]byte, error)    { return nil, errors.New("hash") }
func (hashErrorHasher) Compare(_ []byte, _ string) error { return nil }

func TestService_RegisterHashError(t *testing.T) {
	svc := NewService(notFoundUserRepo{}, hashErrorHasher{}, okSigner{}, time.Now, 24*time.Hour)
	if _, err := svc.Register(context.Background(), "u", "p"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestService_RegisterCreateError(t *testing.T) {
	svc := NewService(createErrorRepo{}, fakeHasher{}, okSigner{}, time.Now, 24*time.Hour)
	if _, err := svc.Register(context.Background(), "u", "p"); err == nil {
		t.Fatalf("expected error")
	}
}
