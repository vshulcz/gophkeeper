package httpapi

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type ctxKey string

const userIDKey ctxKey = "user_id"

func withUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func userIDFromContext(r *http.Request) uuid.UUID {
	v := r.Context().Value(userIDKey)
	if v == nil {
		return uuid.Nil
	}
	id, ok := v.(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return id
}
