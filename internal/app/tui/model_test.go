package tui

import (
	"context"
	"gophkeeper/internal/domain/secret"
	"testing"
	"time"

	appclient "gophkeeper/internal/app/client"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
)

type fakeService struct{}

func (fakeService) Register(context.Context, string, string) error { return nil }
func (fakeService) Login(context.Context, string, string) error    { return nil }
func (fakeService) IsLoggedIn(context.Context) (bool, error)       { return true, nil }
func (fakeService) Logout(context.Context) error                   { return nil }

func (fakeService) List(context.Context, string) ([]appclient.ListItem, error) {
	return []appclient.ListItem{{ID: uuid.New(), Type: secret.TypeText, Version: 1}}, nil
}

func (fakeService) Get(context.Context, uuid.UUID, string) (appclient.PayloadEnvelope, error) {
	return appclient.PayloadEnvelope{Type: secret.TypeText, Meta: map[string]string{}, Data: map[string]string{"text": "hello"}}, nil
}

func (fakeService) Add(context.Context, appclient.AddInput) (uuid.UUID, error) {
	return uuid.New(), nil
}

func (fakeService) Update(context.Context, appclient.UpdateInput) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (fakeService) Delete(context.Context, uuid.UUID) error { return nil }
func (fakeService) Sync(context.Context) (int, error)       { return 0, nil }

func TestModel_InitUpdate(t *testing.T) {
	m := New(fakeService{}, "pass", Deps{
		Now:      time.Now,
		ReadFile: func(string) ([]byte, error) { return []byte("x"), nil },
		Version:  func() (string, string) { return "v1", "d1" },
	})
	if cmd := m.Init(); cmd == nil {
		t.Fatalf("expected cmd")
	}
	model, _ := m.Update(itemsMsg{items: []list.Item{secretItem{id: "id", title: "id", desc: "d", typ: secret.TypeText}}})
	_ = model
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	_, _ = m.Update(detailMsg{content: "x"})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	_, _ = m.Update(authNeededMsg{form: buildForm("Add", map[string]string{"type": "text"}), action: "add"})
	_, _ = m.Update(formMsg{})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	_, _ = m.Update(editFormMsg{form: buildForm("Edit", map[string]string{"type": "text"}), id: uuid.New().String()})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	_, _ = m.Update(deleteMsg{})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	_, _ = m.Update(authMsg{status: "ok", password: "p"})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	_, _ = m.Update(syncMsg{count: 1})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	view := m.View()
	if view == "" {
		t.Fatalf("expected view")
	}
}
