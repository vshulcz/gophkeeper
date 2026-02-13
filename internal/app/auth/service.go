package auth

import (
	"context"
	"errors"
	"gophkeeper/internal/domain"
	"gophkeeper/internal/domain/user"
	"gophkeeper/internal/ports"
	"time"

	"github.com/google/uuid"
)

// Service handles registration and login.
type Service struct {
	users  ports.UserRepository
	hasher ports.PasswordHasher
	signer ports.TokenSigner
	clock  ports.Clock
	ttl    time.Duration
}

// NewService creates an auth service.
func NewService(users ports.UserRepository, hasher ports.PasswordHasher, signer ports.TokenSigner, clock ports.Clock, ttl time.Duration) *Service {
	return &Service{users: users, hasher: hasher, signer: signer, clock: clock, ttl: ttl}
}

// Register creates a new user and returns an auth token.
func (s *Service) Register(ctx context.Context, username, password string) (string, error) {
	_, err := s.users.GetByUsername(ctx, username)
	if err == nil {
		return "", domain.ErrConflict
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return "", err
	}

	hash, err := s.hasher.Hash(password)
	if err != nil {
		return "", err
	}

	user := user.User{
		ID:           uuid.New(),
		Username:     username,
		PasswordHash: hash,
		CreatedAt:    s.clock(),
	}
	if err := s.users.Create(ctx, user); err != nil {
		return "", err
	}

	return s.signer.Sign(user.ID.String(), s.clock().Add(s.ttl))
}

// Login authenticates a user and returns an auth token.
func (s *Service) Login(ctx context.Context, username, password string) (string, error) {
	user, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		return "", err
	}
	if err := s.hasher.Compare(user.PasswordHash, password); err != nil {
		return "", domain.ErrUnauthorized
	}

	return s.signer.Sign(user.ID.String(), s.clock().Add(s.ttl))
}
