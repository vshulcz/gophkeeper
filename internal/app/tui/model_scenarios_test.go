package tui

import (
	"context"
	"errors"
	"gophkeeper/internal/domain/secret"
	"strings"
	"testing"
	"time"

	appclient "gophkeeper/internal/app/client"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
)

type recordService struct {
	registerCalled bool
	loginCalled    bool
	loggedIn       bool
	addCalled      bool
	updateCalled   bool
	deleteCalled   bool
	syncCalled     bool
	lastPassword   string
	items          []appclient.ListItem
	env            appclient.PayloadEnvelope
}

func (r *recordService) Register(_ context.Context, _, _ string) error {
	r.registerCalled = true
	r.loggedIn = true
	return nil
}

func (r *recordService) Login(_ context.Context, _, _ string) error {
	r.loginCalled = true
	r.loggedIn = true
	return nil
}
func (r *recordService) IsLoggedIn(_ context.Context) (bool, error) { return r.loggedIn, nil }
func (r *recordService) Logout(_ context.Context) error {
	r.loggedIn = false
	return nil
}

func (r *recordService) List(_ context.Context, userPassword string) ([]appclient.ListItem, error) {
	r.lastPassword = userPassword
	return r.items, nil
}

func (r *recordService) Get(_ context.Context, _ uuid.UUID, userPassword string) (appclient.PayloadEnvelope, error) {
	r.lastPassword = userPassword
	return r.env, nil
}

func (r *recordService) Add(_ context.Context, input appclient.AddInput) (uuid.UUID, error) {
	r.addCalled = true
	r.lastPassword = input.UserPassword
	return uuid.New(), nil
}

func (r *recordService) Update(_ context.Context, input appclient.UpdateInput) (uuid.UUID, error) {
	r.updateCalled = true
	r.lastPassword = input.UserPassword
	return uuid.New(), nil
}

func (r *recordService) Delete(_ context.Context, _ uuid.UUID) error {
	r.deleteCalled = true
	return nil
}

func (r *recordService) Sync(_ context.Context) (int, error) {
	r.syncCalled = true
	return 1, nil
}

func key(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func runCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

func typeText(m *Model, text string) {
	for _, r := range text {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
}

func TestTUI_EmptyListWhenLoggedOut(t *testing.T) {
	svc := &recordService{loggedIn: false}
	m := New(svc, "", Deps{Now: time.Now})
	msg := m.loadItems()
	m.Update(msg)
	if len(m.list.Items()) != 0 {
		t.Fatalf("expected empty list when logged out")
	}
}

func TestTUI_AddRequiresLoginAndResumesForm(t *testing.T) {
	svc := &recordService{loggedIn: false}
	m := New(svc, "", Deps{Now: time.Now})

	m.Update(key("a"))
	if m.mode != "auth" {
		t.Fatalf("expected auth mode")
	}

	_, cmd := m.Update(authMsg{password: "p", status: "ok"})
	_ = cmd
	if m.mode != "form" {
		t.Fatalf("expected to resume form")
	}
	typeText(m, "login")
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	typeText(m, "meta")
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	typeText(m, "admin")
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	typeText(m, "admin")
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg := runCmd(cmd)
	if msg == nil {
		t.Fatalf("expected formMsg")
	}
	m.Update(msg)
	if !svc.addCalled {
		t.Fatalf("expected add called")
	}
}

func TestTUI_FormIsolation(t *testing.T) {
	svc := &recordService{loggedIn: true}
	m := New(svc, "pass", Deps{Now: time.Now})
	m.Update(m.loadItems())
	m.Update(key("a"))
	m.Update(key("o"))
	if got := m.form.fields[0].input.Value(); got == "" {
		t.Fatalf("expected input to receive characters, got empty")
	}
}

func TestTUI_ListSelectionIndicatorSingle(t *testing.T) {
	svc := &recordService{loggedIn: true}
	m := New(svc, "pass", Deps{Now: time.Now})
	m.list.SetItems([]list.Item{
		secretItem{id: "1", title: "alpha", desc: "login", typ: secret.TypeLoginPassword},
		secretItem{id: "2", title: "beta", desc: "text", typ: secret.TypeText},
		secretItem{id: "3", title: "gamma", desc: "card", typ: secret.TypeCard},
	})
	m.list.Select(1)

	view := m.list.View()
	if count := strings.Count(view, "> "); count != 1 {
		t.Fatalf("expected one selection indicator, got %d\n%s", count, view)
	}
}

func TestResolveBinaryField(t *testing.T) {
	b64, err := resolveBinaryField("b64:YWJj", nil)
	if err != nil || b64 != "YWJj" {
		t.Fatalf("expected base64")
	}
	_, err = resolveBinaryField("not-base64", nil)
	if err == nil {
		t.Fatalf("expected error")
	}
	b64, err = resolveBinaryField("file:/tmp/none", func(string) ([]byte, error) {
		return []byte("abc"), nil
	})
	if err != nil || b64 == "" {
		t.Fatalf("expected file base64")
	}
}

func TestDisplayTitle(t *testing.T) {
	if got := displayTitle(map[string]string{"label": "x"}, "id"); got != "x" {
		t.Fatalf("expected label")
	}
	if got := displayTitle(nil, "1234567890"); got != "12345678" {
		t.Fatalf("expected short id")
	}
}

func TestAuthResumeListActionFallback(t *testing.T) {
	svc := &recordService{
		loggedIn: true,
		items:    []appclient.ListItem{{ID: uuid.New(), Type: secret.TypeText, Version: 1}},
		env:      appclient.PayloadEnvelope{Type: secret.TypeText, Data: map[string]string{"text": "ok"}},
	}
	m := New(svc, "pass", Deps{Now: time.Now})
	m.list.SetItems([]list.Item{secretItem{id: svc.items[0].ID.String(), title: "id", desc: "d", typ: secret.TypeText}})
	cmd := m.resumeListAction("detail", "")
	if cmd == nil {
		t.Fatalf("expected cmd")
	}
	msg := cmd()
	if _, ok := msg.(detailMsg); !ok {
		t.Fatalf("expected detailMsg")
	}
}

func TestSubmitAuthValidation(t *testing.T) {
	svc := &recordService{loggedIn: false}
	m := New(svc, "", Deps{Now: time.Now})
	m.setAuthResume("login", "", "")
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg := runCmd(cmd)
	if msg == nil {
		t.Fatalf("expected authMsg")
	}
	auth, ok := msg.(authMsg)
	if !ok || auth.err == nil {
		t.Fatalf("expected auth error")
	}
}

func TestTUI_LoginTriggersSync(t *testing.T) {
	svc := &recordService{loggedIn: false}
	m := New(svc, "", Deps{Now: time.Now})
	_, cmd := m.Update(authMsg{password: "p", status: "ok"})
	msg := runCmd(cmd)
	if msg == nil {
		t.Fatalf("expected sync msg")
	}
	if _, ok := msg.(syncMsg); !ok {
		t.Fatalf("expected syncMsg, got %T", msg)
	}
	if !svc.syncCalled {
		t.Fatalf("expected sync to be called")
	}
}

func TestRegisterConfirmPassword(t *testing.T) {
	svc := &recordService{loggedIn: false}
	m := New(svc, "", Deps{Now: time.Now})
	m.setAuthResume("register", "", "")
	m.authForm.fields[0].input.SetValue("alice")
	m.authForm.fields[1].input.SetValue("pass")
	m.authForm.fields[2].input.SetValue("nope")
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg := runCmd(cmd)
	auth, ok := msg.(authMsg)
	if !ok || auth.err == nil {
		t.Fatalf("expected mismatch error")
	}
}

func TestListShowsErrorState(t *testing.T) {
	svc := &recordService{
		loggedIn: true,
		items:    []appclient.ListItem{{ID: uuid.New(), Type: secret.TypeText, Version: 1, Err: errors.New("decrypt")}},
	}
	m := New(svc, "pass", Deps{Now: time.Now})
	msg := m.loadItems()
	m.Update(msg)
	if len(m.list.Items()) != 1 {
		t.Fatalf("expected list item")
	}
}

func TestTUI_LogoutClearsSession(t *testing.T) {
	svc := &recordService{loggedIn: true}
	m := New(svc, "pass", Deps{Now: time.Now})
	m.Update(m.loadItems())
	_, cmd := m.Update(key("x"))
	msg := runCmd(cmd)
	m.Update(msg)
	if m.password != "" || m.loggedIn {
		t.Fatalf("expected logged out state")
	}
}

func TestTUI_BodyHeightStableWhenLoggedOut(t *testing.T) {
	svc := &recordService{loggedIn: true}
	m := New(svc, "pass", Deps{Now: time.Now})
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m.Update(m.loadItems())

	before := lipgloss.Height(renderBody(m))
	if before < m.bodyHeight() {
		t.Fatalf("expected body height at least %d, got %d", m.bodyHeight(), before)
	}

	_, cmd := m.Update(key("x"))
	msg := runCmd(cmd)
	m.Update(msg)

	after := lipgloss.Height(renderBody(m))
	if after < m.bodyHeight() {
		t.Fatalf("expected body height at least %d after logout, got %d", m.bodyHeight(), after)
	}
}

func TestTUI_ViewHeightStableAfterDetail(t *testing.T) {
	svc := &recordService{
		loggedIn: true,
		env:      appclient.PayloadEnvelope{Type: secret.TypeLoginPassword, Data: map[string]string{"login": "admin", "password": "admin"}},
	}
	m := New(svc, "pass", Deps{Now: time.Now})
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m.list.SetItems([]list.Item{secretItem{id: "1", title: "admin", desc: "login", typ: secret.TypeLoginPassword}})
	m.list.Select(0)

	before := lipgloss.Height(m.View())
	if before > m.height {
		t.Fatalf("expected view height <= %d, got %d", m.height, before)
	}

	m.Update(detailMsg{content: renderDetail(svc.env)})
	after := lipgloss.Height(m.View())
	if after > m.height {
		t.Fatalf("expected view height <= %d after detail, got %d", m.height, after)
	}
	if after != before {
		t.Fatalf("expected stable view height, before=%d after=%d", before, after)
	}
}

func TestTUI_ViewLinesDoNotExceedWidth(t *testing.T) {
	svc := &recordService{loggedIn: true}
	m := New(svc, "pass", Deps{Now: time.Now})
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	m.list.SetItems([]list.Item{
		secretItem{id: "1", title: "admin-super-long-title-that-should-truncate", desc: "login · site=example.com", typ: secret.TypeLoginPassword},
	})
	view := m.View()
	for i, line := range strings.Split(view, "\n") {
		if w := lipgloss.Width(line); w > m.width {
			t.Fatalf("line %d exceeds width: %d > %d\n%s", i, w, m.width, line)
		}
	}
}

func TestTUI_Filtering(t *testing.T) {
	svc := &recordService{loggedIn: true}
	m := New(svc, "pass", Deps{Now: time.Now})
	items := []list.Item{
		secretItem{id: "1", title: "github", desc: "login", typ: secret.TypeLoginPassword},
		secretItem{id: "2", title: "bank", desc: "card", typ: secret.TypeCard},
	}
	m.allItems = items
	m.applyFilter("git")
	if len(m.list.Items()) != 1 {
		t.Fatalf("expected filtered list")
	}
	m.applyFilter("")
	if len(m.list.Items()) != 2 {
		t.Fatalf("expected cleared filter")
	}
}

func TestTUI_VersionSetsContent(t *testing.T) {
	svc := &recordService{loggedIn: true}
	m := New(svc, "pass", Deps{Now: time.Now, Version: func() (string, string) { return "v1", "d1" }})
	m.Update(key("v"))
	if m.mode != "version" || !strings.Contains(m.versionInfo, "v1") {
		t.Fatalf("expected version view state")
	}
}

func TestTUI_SyncDoesNotClobberDetail(t *testing.T) {
	svc := &recordService{loggedIn: true}
	m := New(svc, "pass", Deps{Now: time.Now})
	m.content = "type: login\nmeta: -\ndata:\n  login: admin\n"
	_, cmd := m.Update(syncMsg{count: 0})
	if cmd == nil {
		t.Fatalf("expected sync cmd")
	}
	m.Update(cmd())
	if !strings.Contains(m.content, "login") {
		t.Fatalf("expected detail to remain")
	}
	view := m.View()
	if strings.Count(view, "sync complete") > 1 {
		t.Fatalf("expected single sync status")
	}
}
