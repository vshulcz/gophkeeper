package persistence

import (
	"context"
	"testing"
)

func TestOpenSQLite_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := OpenSQLite(ctx, "file::memory:?cache=shared")
	if err == nil {
		t.Fatalf("expected error")
	}
}
