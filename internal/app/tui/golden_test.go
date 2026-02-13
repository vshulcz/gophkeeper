//go:build integration
// +build integration

package tui

import (
	"flag"
	"gophkeeper/internal/domain/secret"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var updateGoldens = flag.Bool("update", false, "update golden files")

func init() {
	lipgloss.SetColorProfile(termenv.TrueColor)
	lipgloss.SetHasDarkBackground(true)
}

func goldenPath(name string) string {
	return filepath.Join("testdata", name+".golden")
}

func assertGolden(t *testing.T, name, got string) {
	t.Helper()
	path := goldenPath(name)
	if *updateGoldens {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if string(want) != got {
		t.Fatalf("golden mismatch for %s", name)
	}
}

func newGoldenModel(loggedIn bool) *Model {
	svc := &recordService{loggedIn: loggedIn}
	deps := Deps{
		Now: func() time.Time { return time.Unix(0, 0) },
	}
	pass := ""
	if loggedIn {
		pass = "pass"
	}
	m := New(svc, pass, deps)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	return m
}

func TestTUI_Golden_ListLoggedOut(t *testing.T) {
	m := newGoldenModel(false)
	m.Update(m.loadItems())
	assertGolden(t, "list_logged_out", m.View())
}

func TestTUI_Golden_ListLoggedIn(t *testing.T) {
	m := newGoldenModel(true)
	id1 := "11111111-1111-1111-1111-111111111111"
	id2 := "22222222-2222-2222-2222-222222222222"
	items := []list.Item{
		secretItem{id: id1, title: "github", desc: "type=login_password v=1", typ: secret.TypeLoginPassword},
		secretItem{id: id2, title: "email", desc: "type=text v=2", typ: secret.TypeText},
	}
	m.list.SetItems(items)
	assertGolden(t, "list_logged_in", m.View())
}

func TestTUI_Golden_FormAddLogin(t *testing.T) {
	m := newGoldenModel(true)
	m.Update(key("a"))
	typeText(m, "login")
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	typeText(m, "site=github")
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	typeText(m, "admin")
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	typeText(m, "secret")
	assertGolden(t, "form_add_login", m.View())
}

func TestTUI_Golden_AuthLogin(t *testing.T) {
	m := newGoldenModel(false)
	m.Update(key("l"))
	typeText(m, "alice")
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	typeText(m, "pass")
	assertGolden(t, "auth_login", m.View())
}

func TestTUI_Golden_Register(t *testing.T) {
	m := newGoldenModel(false)
	m.Update(key("r"))
	typeText(m, "alice")
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	typeText(m, "pass")
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	typeText(m, "pass")
	assertGolden(t, "auth_register", m.View())
}

func TestTUI_Golden_ConfirmDelete(t *testing.T) {
	m := newGoldenModel(true)
	itemID := "33333333-3333-3333-3333-333333333333"
	m.list.SetItems([]list.Item{secretItem{id: itemID, title: "github", desc: "type=login_password v=1", typ: secret.TypeLoginPassword}})
	m.Update(key("d"))
	assertGolden(t, "confirm_delete", m.View())
}

func TestTUI_Golden_DetailView(t *testing.T) {
	m := newGoldenModel(true)
	m.content = "type: login_password\nmeta: map[site:github]\ndata: map[login:admin password:admin]"
	assertGolden(t, "detail_view", m.View())
}

func TestTUI_Golden_NarrowLayout(t *testing.T) {
	m := newGoldenModel(true)
	m.Update(tea.WindowSizeMsg{Width: 70, Height: 30})
	m.list.SetItems([]list.Item{
		secretItem{id: "11111111-1111-1111-1111-111111111111", title: "github", desc: "type=login_password v=1", typ: secret.TypeLoginPassword},
	})
	assertGolden(t, "narrow_layout", m.View())
}

func TestTUI_Golden_Search(t *testing.T) {
	m := newGoldenModel(true)
	m.mode = "search"
	m.searchForm = newSearchForm("Search")
	m.searchForm.fields[0].input.SetValue("admin")
	assertGolden(t, "search_view", m.View())
}

func TestTUI_Golden_Version(t *testing.T) {
	m := newGoldenModel(true)
	m.mode = "version"
	m.versionInfo = "version: v1.2.3\nbuild date: 2026-02-06"
	assertGolden(t, "version_view", m.View())
}

func TestTUI_Golden_EditForm(t *testing.T) {
	m := newGoldenModel(true)
	values := map[string]string{
		"type":   "login",
		"meta":   "site=github",
		"field1": "admin",
		"field2": "secret",
	}
	m.mode = "form"
	m.formAction = "edit"
	m.form = buildForm("Edit Secret", values)
	assertGolden(t, "form_edit_login", m.View())
}

func TestTUI_Golden_FilteredList(t *testing.T) {
	m := newGoldenModel(true)
	item := secretItem{id: "11111111-1111-1111-1111-111111111111", title: "admin", desc: "login · site=example.com", typ: secret.TypeLoginPassword}
	m.filterQuery = "admin"
	m.list.SetItems([]list.Item{item})
	assertGolden(t, "list_filtered", m.View())
}
