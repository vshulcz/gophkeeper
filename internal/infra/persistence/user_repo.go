package persistence

import (
	"context"
	"database/sql"
	"errors"
	"gophkeeper/internal/domain"
	"gophkeeper/internal/domain/user"
	"strings"
	"time"

	"github.com/google/uuid"
)

// UserRepo is a SQLite-backed repository for users.
type UserRepo struct {
	db *sql.DB
}

// NewUserRepo creates a UserRepo.
func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

// Create inserts a new user.
func (r *UserRepo) Create(ctx context.Context, user user.User) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, username, password_hash, created_at) VALUES (?, ?, ?, ?)`,
		user.ID.String(), user.Username, user.PasswordHash, user.CreatedAt.Unix())
	if isUniqueViolation(err) {
		return domain.ErrConflict
	}
	return err
}

// GetByUsername returns a user by username.
func (r *UserRepo) GetByUsername(ctx context.Context, username string) (user.User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, created_at FROM users WHERE username = ?`, username)
	var id string
	var createdAt int64
	var hash []byte
	if err := row.Scan(&id, &username, &hash, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return user.User{}, domain.ErrNotFound
		}
		return user.User{}, err
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return user.User{}, err
	}
	return user.User{ID: uid, Username: username, PasswordHash: hash, CreatedAt: time.Unix(createdAt, 0)}, nil
}

// GetByID returns a user by ID.
func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (user.User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, created_at FROM users WHERE id = ?`, id.String())
	var uid string
	var username string
	var createdAt int64
	var hash []byte
	if err := row.Scan(&uid, &username, &hash, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return user.User{}, domain.ErrNotFound
		}
		return user.User{}, err
	}
	parsed, err := uuid.Parse(uid)
	if err != nil {
		return user.User{}, err
	}
	return user.User{ID: parsed, Username: username, PasswordHash: hash, CreatedAt: time.Unix(createdAt, 0)}, nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	// SQLite uses "UNIQUE constraint failed" in error messages.
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}
