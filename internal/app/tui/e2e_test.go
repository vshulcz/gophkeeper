//go:build integration
// +build integration

package tui

import (
	"context"
	"gophkeeper/internal/domain/secret"
	"strings"
	"testing"
	"time"

	appclient "gophkeeper/internal/app/client"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
)

type e2eService struct {
	items []appclient.ListItem
	env   appclient.PayloadEnvelope
}

func (s *e2eService) Register(context.Context, string, string) error { return nil }
func (s *e2eService) Login(context.Context, string, string) error    { return nil }
func (s *e2eService) IsLoggedIn(context.Context) (bool, error)       { return true, nil }
func (s *e2eService) Logout(context.Context) error                   { return nil }
func (s *e2eService) List(context.Context, string) ([]appclient.ListItem, error) {
	return s.items, nil
}

func (s *e2eService) Get(context.Context, uuid.UUID, string) (appclient.PayloadEnvelope, error) {
	return s.env, nil
}

func (s *e2eService) Add(context.Context, appclient.AddInput) (uuid.UUID, error) {
	return uuid.New(), nil
}

func (s *e2eService) Update(context.Context, appclient.UpdateInput) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (s *e2eService) Delete(context.Context, uuid.UUID) error { return nil }
func (s *e2eService) Sync(context.Context) (int, error)       { return 1, nil }

func step(m *Model, msg tea.Msg) {
	_, cmd := m.Update(msg)
	if cmd != nil {
		out := cmd()
		m.Update(out)
	}
}

func TestTUI_E2E_VersionAndView(t *testing.T) {
	id := uuid.New()
	svc := &e2eService{
		items: []appclient.ListItem{{ID: id, Type: secret.TypeLoginPassword, Version: 1, Meta: map[string]string{"label": "admin"}}},
		env:   appclient.PayloadEnvelope{Type: secret.TypeLoginPassword, Data: map[string]string{"login": "admin", "password": "admin"}},
	}
	m := New(svc, "pass", Deps{Now: time.Now, Version: func() (string, string) { return "v1", "d1" }})
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m.Update(m.loadItems())

	step(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	if !strings.Contains(m.View(), "version: v1") {
		t.Fatalf("expected version in output")
	}

	step(m, tea.KeyMsg{Type: tea.KeyEsc})
	step(m, tea.KeyMsg{Type: tea.KeyEnter})
	if !strings.Contains(m.View(), "data:") {
		t.Fatalf("expected detail view")
	}
}

func TestTUI_E2E_FilterInputDoesNotCaptureF(t *testing.T) {
	id := uuid.New()
	svc := &e2eService{
		items: []appclient.ListItem{{ID: id, Type: secret.TypeText, Version: 1, Meta: map[string]string{"label": "note"}}},
		env:   appclient.PayloadEnvelope{Type: secret.TypeText, Data: map[string]string{"text": "note"}},
	}
	m := New(svc, "pass", Deps{Now: time.Now})
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m.Update(m.loadItems())

	step(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	step(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	step(m, tea.KeyMsg{Type: tea.KeyEnter})
	if !strings.Contains(m.View(), "Filter: n") {
		t.Fatalf("expected filter in output")
	}
}
