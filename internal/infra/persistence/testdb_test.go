package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"
)

func openTestDB(t *testing.T) (*sql.DB, context.Context) {
	ctx := context.Background()
	dsn := fmt.Sprintf("file:memdb_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := OpenSQLite(ctx, dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	return db, ctx
}
