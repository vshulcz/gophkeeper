//go:build integration
// +build integration

package tui

import (
	"context"
	"fmt"
	"gophkeeper/internal/domain/secret"
	"strings"
	"testing"
	"time"

	appclient "gophkeeper/internal/app/client"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
)

type scenarioService struct {
	loggedIn   bool
	items      []appclient.ListItem
	envByID    map[uuid.UUID]appclient.PayloadEnvelope
	addCalled  bool
	updCalled  bool
	delCalled  bool
	syncCalled bool
}

func (s *scenarioService) Register(context.Context, string, string) error {
	s.loggedIn = true
	return nil
}

func (s *scenarioService) Login(context.Context, string, string) error {
	s.loggedIn = true
	return nil
}
func (s *scenarioService) IsLoggedIn(context.Context) (bool, error) { return s.loggedIn, nil }
func (s *scenarioService) Logout(context.Context) error {
	s.loggedIn = false
	return nil
}

func (s *scenarioService) List(context.Context, string) ([]appclient.ListItem, error) {
	return s.items, nil
}

func (s *scenarioService) Get(_ context.Context, id uuid.UUID, _ string) (appclient.PayloadEnvelope, error) {
	if env, ok := s.envByID[id]; ok {
		return env, nil
	}
	return appclient.PayloadEnvelope{}, fmt.Errorf("not found")
}

func (s *scenarioService) Add(context.Context, appclient.AddInput) (uuid.UUID, error) {
	s.addCalled = true
	return uuid.New(), nil
}

func (s *scenarioService) Update(context.Context, appclient.UpdateInput) (uuid.UUID, error) {
	s.updCalled = true
	return uuid.New(), nil
}

func (s *scenarioService) Delete(context.Context, uuid.UUID) error {
	s.delCalled = true
	return nil
}

func (s *scenarioService) Sync(context.Context) (int, error) {
	s.syncCalled = true
	return len(s.items), nil
}

func newScenarioModel(loggedIn bool) (*Model, *scenarioService) {
	idLogin := uuid.New()
	items := []appclient.ListItem{
		{ID: idLogin, Type: secret.TypeLoginPassword, Version: 1, Meta: map[string]string{"label": "admin", "site": "example.com"}},
	}
	envs := map[uuid.UUID]appclient.PayloadEnvelope{
		idLogin: {
			Type: secret.TypeLoginPassword,
			Data: map[string]string{"login": "admin", "password": "admin"},
			Meta: map[string]string{"label": "admin", "site": "example.com"},
		},
	}
	svc := &scenarioService{loggedIn: loggedIn, items: items, envByID: envs}
	pass := ""
	if loggedIn {
		pass = "pass"
	}
	deps := Deps{
		Now:     func() time.Time { return time.Unix(0, 0) },
		Version: func() (string, string) { return "v1.2.3", "2026-02-06" },
	}
	m := New(svc, pass, deps)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m.Update(m.loadItems())
	return m, svc
}

func pressKey(m *Model, key string) {
	var msg tea.Msg
	switch key {
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		msg = tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		msg = tea.KeyMsg{Type: tea.KeyShiftTab}
	case "up":
		msg = tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		msg = tea.KeyMsg{Type: tea.KeyDown}
	default:
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
	step(m, msg)
}

func assertBasics(t *testing.T, m *Model, view string) {
	t.Helper()
	if !strings.Contains(view, "GophKeeper") {
		t.Fatalf("missing header")
	}
	if !strings.Contains(view, "l login") || !strings.Contains(view, "q quit") {
		t.Fatalf("missing footer hints")
	}
	if m.mode == "list" && m.loggedIn && len(m.list.Items()) > 0 {
		if count := strings.Count(view, "> "); count != 1 {
			t.Fatalf("expected one selection indicator, got %d\n%s", count, view)
		}
	}
}

func TestTUI_E2E_LoginFlow(t *testing.T) {
	m, _ := newScenarioModel(false)
	pressKey(m, "l")
	typeText(m, "alice")
	pressKey(m, "tab")
	typeText(m, "secret")
	pressKey(m, "enter")

	view := m.View()
	if !strings.Contains(view, "sync complete") && !strings.Contains(view, "syncing") {
		t.Fatalf("expected sync status, got:\n%s", view)
	}
	if !strings.Contains(view, "Session: alice") {
		t.Fatalf("expected session user, got:\n%s", view)
	}
	assertBasics(t, m, view)
}

func TestTUI_E2E_RegisterValidation(t *testing.T) {
	m, _ := newScenarioModel(false)
	pressKey(m, "r")
	typeText(m, "bob")
	pressKey(m, "tab")
	typeText(m, "pw1")
	pressKey(m, "tab")
	typeText(m, "pw2")
	pressKey(m, "enter")
	if !strings.Contains(m.View(), "passwords do not match") {
		t.Fatalf("expected mismatch error")
	}
	m.err = nil
	m.authForm.fields[2].input.SetValue("pw1")
	pressKey(m, "enter")
	view := m.View()
	if !strings.Contains(view, "Session: bob") {
		t.Fatalf("expected session user")
	}
	if !strings.Contains(view, "sync complete") && !strings.Contains(view, "syncing") {
		t.Fatalf("expected sync status")
	}
}

func TestTUI_E2E_AddEditDeleteFlow(t *testing.T) {
	m, svc := newScenarioModel(true)

	pressKey(m, "a")
	typeText(m, "login")
	pressKey(m, "tab")
	typeText(m, "label=siteA")
	pressKey(m, "tab")
	typeText(m, "admin")
	pressKey(m, "tab")
	typeText(m, "admin")
	pressKey(m, "enter")
	if !svc.addCalled {
		t.Fatalf("expected add called, err=%v password=%q mode=%s values=%v", m.err, m.password, m.mode, m.form.values())
	}

	pressKey(m, "e")
	if m.mode != "form" {
		t.Fatalf("expected edit form")
	}
	pressKey(m, "enter")
	if !svc.updCalled {
		t.Fatalf("expected update called")
	}

	pressKey(m, "d")
	if m.mode != "confirm" {
		t.Fatalf("expected confirm mode")
	}
	pressKey(m, "y")
	if !svc.delCalled {
		t.Fatalf("expected delete called")
	}
}

func TestTUI_E2E_FilterAndClear(t *testing.T) {
	m, _ := newScenarioModel(true)
	pressKey(m, "f")
	typeText(m, "admin")
	pressKey(m, "enter")
	if !strings.Contains(m.View(), "Filter: admin") {
		t.Fatalf("expected filter shown")
	}
	pressKey(m, "c")
	if strings.Contains(m.View(), "Filter: admin") {
		t.Fatalf("expected filter cleared")
	}
}

func TestTUI_E2E_VersionSyncLogout(t *testing.T) {
	m, svc := newScenarioModel(true)
	pressKey(m, "v")
	if !strings.Contains(m.View(), "version: v1.2.3") {
		t.Fatalf("expected version view")
	}
	pressKey(m, "esc")
	pressKey(m, "s")
	if !svc.syncCalled {
		t.Fatalf("expected sync called")
	}
	if !strings.Contains(m.View(), "sync complete") {
		t.Fatalf("expected sync status")
	}
	pressKey(m, "x")
	if m.loggedIn {
		t.Fatalf("expected logged out")
	}
	if !strings.Contains(m.View(), "Not logged in") {
		t.Fatalf("expected logged out view")
	}
}

func TestTUI_E2E_InvalidTypeAndBinaryValidation(t *testing.T) {
	m, _ := newScenarioModel(true)

	pressKey(m, "a")
	typeText(m, "unknown")
	pressKey(m, "enter")
	if !strings.Contains(m.View(), "invalid item type") {
		t.Fatalf("expected invalid type error")
	}

	pressKey(m, "esc")
	pressKey(m, "a")
	typeText(m, "binary")
	pressKey(m, "tab")
	typeText(m, "")
	pressKey(m, "tab")
	typeText(m, "not-base64")
	pressKey(m, "enter")
	if !strings.Contains(m.View(), "binary field must be file:/path or b64:base64") {
		t.Fatalf("expected binary validation error")
	}
}

func TestTUI_E2E_CompactBackNavigation(t *testing.T) {
	m, _ := newScenarioModel(true)
	m.Update(tea.WindowSizeMsg{Width: 70, Height: 20})
	m.list.Select(0)
	pressKey(m, "enter")
	if m.panel != "detail" {
		t.Fatalf("expected compact detail panel")
	}
	pressKey(m, "esc")
	if m.panel != "list" {
		t.Fatalf("expected compact list panel")
	}
}

func TestTUI_E2E_NavigationFocusStability(t *testing.T) {
	m, _ := newScenarioModel(true)
	pressKey(m, "down")
	pressKey(m, "up")
	view := m.View()
	if count := strings.Count(view, "> "); count != 1 {
		t.Fatalf("expected single selection indicator, got %d", count)
	}
}

func TestTUI_E2E_KeyMatrix(t *testing.T) {
	keys := []string{"l", "r", "a", "e", "d", "s", "v", "o", "w", "p", "f", "c", "x", "enter", "esc", "tab", "shift+tab", "down", "up"}
	sequences := make([][]string, 0, len(keys)*len(keys)*len(keys))
	for _, a := range keys {
		for _, b := range keys {
			for _, c := range keys {
				sequences = append(sequences, []string{a, b, c})
			}
		}
	}

	for _, loggedIn := range []bool{true, false} {
		for _, seq := range sequences {
			m, _ := newScenarioModel(loggedIn)
			for _, k := range seq {
				pressKey(m, k)
			}
			view := m.View()
			assertBasics(t, m, view)
		}
	}
}

func TestTUI_E2E_KeyMatrixLength4(t *testing.T) {
	keys := []string{"l", "r", "a", "e", "d", "s", "v", "o", "w", "p", "f", "x", "enter", "esc", "tab"}
	sequences := make([][]string, 0, len(keys)*len(keys)*len(keys)*len(keys))
	for _, a := range keys {
		for _, b := range keys {
			for _, c := range keys {
				for _, d := range keys {
					sequences = append(sequences, []string{a, b, c, d})
				}
			}
		}
	}

	for _, loggedIn := range []bool{true, false} {
		for _, seq := range sequences {
			m, _ := newScenarioModel(loggedIn)
			for _, k := range seq {
				pressKey(m, k)
			}
			view := m.View()
			assertBasics(t, m, view)
		}
	}
}
