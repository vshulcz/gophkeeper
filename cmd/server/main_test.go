package main

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestEnvOrDefault(t *testing.T) {
	if err := os.Setenv("TEST_ENV", "x"); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Unsetenv("TEST_ENV"); err != nil {
			t.Fatalf("unsetenv: %v", err)
		}
	})
	if v := envOrDefault("TEST_ENV", "d"); v != "x" {
		t.Fatalf("expected env value")
	}
	if v := envOrDefault("MISSING_ENV", "d"); v != "d" {
		t.Fatalf("expected default")
	}
}

func TestDurationOrDefault(t *testing.T) {
	if err := os.Setenv("TEST_DUR", "1s"); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Unsetenv("TEST_DUR"); err != nil {
			t.Fatalf("unsetenv: %v", err)
		}
	})
	if v := durationOrDefault("TEST_DUR", 2*time.Second); v != time.Second {
		t.Fatalf("expected parsed duration")
	}
	if err := os.Setenv("TEST_DUR", "bad"); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	if v := durationOrDefault("TEST_DUR", 2*time.Second); v != 2*time.Second {
		t.Fatalf("expected default")
	}
}

func TestRunServer(t *testing.T) {
	if err := os.Setenv("GOPHKEEPER_ADDR", "127.0.0.1:0"); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	if err := os.Setenv("GOPHKEEPER_DB", "file::memory:?cache=shared"); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	if err := os.Setenv("GOPHKEEPER_TOKEN_SECRET", "s"); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	if err := os.Setenv("GOPHKEEPER_TOKEN_TTL", "1s"); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Unsetenv("GOPHKEEPER_ADDR"); err != nil {
			t.Fatalf("unsetenv: %v", err)
		}
		if err := os.Unsetenv("GOPHKEEPER_DB"); err != nil {
			t.Fatalf("unsetenv: %v", err)
		}
		if err := os.Unsetenv("GOPHKEEPER_TOKEN_SECRET"); err != nil {
			t.Fatalf("unsetenv: %v", err)
		}
		if err := os.Unsetenv("GOPHKEEPER_TOKEN_TTL"); err != nil {
			t.Fatalf("unsetenv: %v", err)
		}
	})

	err := runServer(context.Background(), func(s *http.Server) error {
		if s.Addr == "" {
			t.Fatalf("expected addr")
		}
		return http.ErrServerClosed
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
}
