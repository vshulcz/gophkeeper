package tui

import (
	"context"
	"errors"
	"fmt"
	"gophkeeper/internal/domain/secret"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	reflowtruncate "github.com/muesli/reflow/truncate"

	appclient "gophkeeper/internal/app/client"
)

// Service exposes client operations for TUI.
type Service interface {
	Register(ctx context.Context, username, password string) error
	Login(ctx context.Context, username, password string) error
	IsLoggedIn(ctx context.Context) (bool, error)
	Logout(ctx context.Context) error
	List(ctx context.Context, userPassword string) ([]appclient.ListItem, error)
	Get(ctx context.Context, id uuid.UUID, userPassword string) (appclient.PayloadEnvelope, error)
	Add(ctx context.Context, input appclient.AddInput) (uuid.UUID, error)
	Update(ctx context.Context, input appclient.UpdateInput) (uuid.UUID, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Sync(ctx context.Context) (int, error)
}

// Deps are external dependencies for the TUI.
type Deps struct {
	Now      func() time.Time
	ReadFile func(string) ([]byte, error)
	Version  func() (string, string)
}

// Model is the Bubble Tea model for the TUI.
type Model struct {
	service    Service
	password   string
	list       list.Model
	content    string
	err        error
	status     string
	loggedIn   bool
	loggedUser string
	width      int
	height     int

	mode string
	form formModel

	formAction string
	formID     string

	allItems    []list.Item
	filterQuery string
	searchForm  formModel
	panel       string

	confirmAction  string
	confirmID      string
	confirmMessage string

	resumeForm   formModel
	resumeAction string
	resumeID     string

	authAction string
	authForm   formModel

	versionInfo string

	deps Deps
}

const (
	modeList    = "list"
	modeForm    = "form"
	modeAuth    = "auth"
	modeSearch  = "search"
	modeConfirm = "confirm"
	modeVersion = "version"

	panelDetail = "detail"

	actionAdd      = "add"
	actionEdit     = "edit"
	actionDelete   = "delete"
	actionLogin    = "login"
	actionRegister = "register"
	actionDetail   = "detail"
	actionSync     = "sync"

	statusNotLoggedIn = "not logged in: press l to login or r to register"
	statusConflict    = "conflict: item updated elsewhere, press s to sync and retry"
	statusSyncing     = "syncing..."
)

// New creates a TUI model.
func New(service Service, password string, deps Deps) *Model {
	items := []list.Item{}
	l := list.New(items, secretDelegate{}, 0, 0)
	l.Title = "Secrets"
	l.SetShowFilter(false)
	l.SetFilteringEnabled(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.DisableQuitKeybindings()
	l.KeyMap.ClearFilter.SetEnabled(false)
	l.KeyMap.Filter.SetEnabled(false)
	l.Styles.Title = styles.Accent
	l.Styles.TitleBar = styles.Accent
	l.SetSize(50, 15)
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if deps.Version == nil {
		deps.Version = func() (string, string) { return "dev", "unknown" }
	}
	status := ""
	if password == "" {
		status = statusNotLoggedIn
	}
	return &Model{service: service, password: password, list: l, mode: modeList, panel: modeList, deps: deps, status: status}
}

// Init initializes the model.
func (m *Model) Init() tea.Cmd {
	return m.loadItems
}

// Update updates the model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.handleWindowSize(msg)
		return m, nil
	case tea.KeyMsg:
		if cmd, handled := m.handleKeyMsg(msg); handled {
			return m, cmd
		}
	case itemsMsg:
		m.handleItemsMsg(msg)
		return m, nil
	case detailMsg:
		m.handleDetailMsg(msg)
		return m, nil
	case authMsg:
		return m, m.handleAuthMsg(msg)
	case authNeededMsg:
		m.handleAuthNeededMsg(msg)
		return m, nil
	case syncMsg:
		return m, m.handleSyncMsg(msg)
	case logoutMsg:
		return m, m.handleLogoutMsg(msg)
	case editFormMsg:
		m.handleEditFormMsg(msg)
		return m, nil
	case formMsg:
		return m, m.handleFormMsg(msg)
	case deleteMsg:
		return m, m.handleDeleteMsg(msg)
	}

	if m.updateActiveForm(msg) {
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *Model) handleWindowSize(msg tea.WindowSizeMsg) {
	m.width = msg.Width
	m.height = msg.Height
	left, _ := layoutWidths(msg.Width)
	headerH := lipgloss.Height(renderHeader(m))
	footerH := lipgloss.Height(renderFooter(m))
	bodyHeight := maxInt(1, msg.Height-headerH-footerH-3)
	frameX, frameY := styles.Panel.GetFrameSize()
	listW := maxInt(20, left-frameX)
	listH := maxInt(1, bodyHeight-frameY)
	m.list.SetSize(listW, listH)
}

func (m *Model) handleKeyMsg(msg tea.KeyMsg) (tea.Cmd, bool) {
	key := msg.String()
	if cmd, ok := m.handleQuitKey(key); ok {
		return cmd, true
	}
	if cmd, ok := m.handleConfirmKey(key); ok {
		return cmd, true
	}
	if cmd, ok := m.handleEnterKey(key); ok {
		return cmd, true
	}
	if cmd, ok := m.handleListShortcut(key); ok {
		return cmd, true
	}
	if m.handleEscapeKey(key) {
		return nil, true
	}
	if m.handleTabKey(key) {
		return nil, true
	}
	return nil, false
}

func (m *Model) handleQuitKey(key string) (tea.Cmd, bool) {
	if key == "q" || key == "ctrl+c" {
		return tea.Quit, true
	}
	return nil, false
}

func (m *Model) handleConfirmKey(key string) (tea.Cmd, bool) {
	switch key {
	case "y":
		if m.mode == modeConfirm && m.confirmAction == actionDelete && m.confirmID != "" {
			return m.deleteItem(m.confirmID), true
		}
	case "n":
		if m.mode == modeConfirm {
			m.mode = modeList
			m.confirmAction = ""
			m.confirmID = ""
			m.confirmMessage = ""
			return nil, true
		}
	}
	return nil, false
}

func (m *Model) handleEnterKey(key string) (tea.Cmd, bool) {
	if key != "enter" {
		return nil, false
	}
	switch m.mode {
	case modeForm:
		return m.submitForm(), true
	case modeAuth:
		return m.submitAuth(), true
	case modeSearch:
		query := m.searchForm.values()["query"]
		m.applyFilter(query)
		m.mode = modeList
		m.searchForm = formModel{}
		return nil, true
	case modeList:
		item, ok := m.list.SelectedItem().(secretItem)
		if !ok {
			return nil, true
		}
		if !m.loggedIn || m.password == "" {
			m.setAuthResume(actionLogin, actionDetail, item.id)
			m.status = "login required to view secrets"
			return nil, true
		}
		if m.compact() {
			m.panel = panelDetail
		}
		return m.loadDetail(item.id), true
	default:
		return nil, false
	}
}

func (m *Model) handleListShortcut(key string) (tea.Cmd, bool) {
	if m.mode != modeList {
		return nil, false
	}
	switch key {
	case "a":
		return m.handleAddShortcut()
	case "f":
		return m.handleFilterShortcut()
	case "c":
		return m.handleClearFilterShortcut()
	case "e":
		return m.handleEditShortcut()
	case "d":
		return m.handleDeleteShortcut()
	case "l":
		return m.handleLoginShortcut()
	case "r":
		return m.handleRegisterShortcut()
	case "s":
		return m.handleSyncShortcut()
	case "v":
		return m.handleVersionShortcut()
	case "x":
		return m.handleLogoutShortcut()
	}
	return nil, false
}

func (m *Model) handleAddShortcut() (tea.Cmd, bool) {
	if !m.loggedIn || m.password == "" {
		m.setAuthResume(actionLogin, actionAdd, "")
		m.status = "login required to add secret"
		return nil, true
	}
	m.mode = modeForm
	m.err = nil
	m.formAction = actionAdd
	m.formID = ""
	m.form = buildForm("Add Secret", map[string]string{})
	return nil, true
}

func (m *Model) handleFilterShortcut() (tea.Cmd, bool) {
	m.mode = modeSearch
	m.searchForm = newSearchForm("Search")
	if m.filterQuery != "" {
		m.searchForm.fields[0].input.SetValue(m.filterQuery)
	}
	return nil, true
}

func (m *Model) handleClearFilterShortcut() (tea.Cmd, bool) {
	m.applyFilter("")
	return nil, true
}

func (m *Model) handleEditShortcut() (tea.Cmd, bool) {
	item, ok := m.list.SelectedItem().(secretItem)
	if !ok {
		return nil, true
	}
	if !m.loggedIn || m.password == "" {
		m.setAuthResume(actionLogin, actionEdit, item.id)
		m.status = "login required to edit secret"
		return nil, true
	}
	m.err = nil
	return m.loadEditForm(item.id), true
}

func (m *Model) handleDeleteShortcut() (tea.Cmd, bool) {
	item, ok := m.list.SelectedItem().(secretItem)
	if !ok {
		return nil, true
	}
	if !m.loggedIn || m.password == "" {
		m.setAuthResume(actionLogin, actionDelete, item.id)
		m.status = "login required to delete secret"
		return nil, true
	}
	m.mode = modeConfirm
	m.confirmAction = actionDelete
	m.confirmID = item.id
	m.confirmMessage = fmt.Sprintf("Delete item %s? (y/n)", item.id)
	return nil, true
}

func (m *Model) handleLoginShortcut() (tea.Cmd, bool) {
	m.err = nil
	m.setAuthResume(actionLogin, "", "")
	return nil, true
}

func (m *Model) handleRegisterShortcut() (tea.Cmd, bool) {
	m.err = nil
	m.setAuthResume(actionRegister, "", "")
	return nil, true
}

func (m *Model) handleSyncShortcut() (tea.Cmd, bool) {
	if !m.loggedIn || m.password == "" {
		m.setAuthResume(actionLogin, actionSync, "")
		m.status = "login required to sync"
		return nil, true
	}
	m.status = statusSyncing
	if strings.TrimSpace(m.content) == "" {
		m.content = m.status
	}
	return m.syncItems(), true
}

func (m *Model) handleVersionShortcut() (tea.Cmd, bool) {
	ver, date := m.deps.Version()
	m.versionInfo = fmt.Sprintf("version: %s\nbuild date: %s", ver, date)
	m.content = ""
	m.status = ""
	m.mode = modeVersion
	return nil, true
}

func (m *Model) handleLogoutShortcut() (tea.Cmd, bool) {
	if !m.loggedIn {
		return nil, false
	}
	return m.logout(), true
}

func (m *Model) handleEscapeKey(key string) bool {
	if key != "esc" {
		return false
	}
	switch m.mode {
	case modeForm:
		m.mode = modeList
		m.form = formModel{}
		m.formAction = ""
		m.formID = ""
	case modeConfirm:
		m.mode = modeList
		m.confirmAction = ""
		m.confirmID = ""
		m.confirmMessage = ""
	case modeAuth:
		if m.resumeForm.title != "" {
			m.mode = modeForm
			m.form = m.resumeForm
			m.formAction = m.resumeAction
			m.formID = m.resumeID
		} else {
			m.mode = modeList
		}
		m.authAction = ""
		m.authForm = formModel{}
		m.resumeForm = formModel{}
		m.resumeAction = ""
		m.resumeID = ""
	case modeSearch:
		m.mode = modeList
		m.searchForm = formModel{}
	case modeVersion:
		m.mode = modeList
		m.versionInfo = ""
	case modeList:
		if m.compact() && m.panel == panelDetail {
			m.panel = modeList
			return true
		}
	}
	return true
}

func (m *Model) handleTabKey(key string) bool {
	switch key {
	case "tab":
		switch m.mode {
		case modeForm:
			m.form.next()
		case modeAuth:
			m.authForm.next()
		case modeSearch:
			m.searchForm.next()
		}
		return true
	case "shift+tab":
		switch m.mode {
		case modeForm:
			m.form.prev()
		case modeAuth:
			m.authForm.prev()
		case modeSearch:
			m.searchForm.prev()
		}
		return true
	default:
		return false
	}
}

func (m *Model) handleItemsMsg(msg itemsMsg) {
	m.list.SetItems(msg.items)
	m.list.Title = fmt.Sprintf("Secrets (%d)", len(msg.items))
	m.allItems = msg.items
	if len(msg.items) == 0 && strings.TrimSpace(m.content) == "" {
		m.content = "No items."
	}
	m.err = msg.err
}

func (m *Model) handleDetailMsg(msg detailMsg) {
	if isSessionExpired(msg.err) {
		m.handleSessionExpired()
		return
	}
	m.content = msg.content
	m.err = msg.err
	if m.compact() {
		m.panel = panelDetail
	}
}

func (m *Model) handleAuthMsg(msg authMsg) tea.Cmd {
	m.err = msg.err
	if msg.err != nil {
		return nil
	}
	m.authAction = ""
	m.authForm = formModel{}
	m.password = msg.password
	m.loggedUser = msg.user
	m.status = msg.status
	m.content = ""
	m.loggedIn = true
	m.panel = modeList
	m.filterQuery = ""
	m.list.SetItems(nil)
	if m.resumeForm.title != "" {
		m.mode = modeForm
		m.form = m.resumeForm
		m.formAction = m.resumeAction
		m.formID = m.resumeID
		m.resumeForm = formModel{}
		m.resumeAction = ""
		m.resumeID = ""
		return nil
	}
	if m.resumeAction != "" {
		action := m.resumeAction
		itemID := m.resumeID
		m.resumeAction = ""
		m.resumeID = ""
		m.mode = modeList
		return m.resumeListAction(action, itemID)
	}
	m.mode = modeList
	return m.syncItems()
}

func (m *Model) handleAuthNeededMsg(msg authNeededMsg) {
	m.err = msg.err
	m.setAuthResume(actionLogin, msg.action, msg.id)
	m.resumeForm = msg.form
	m.resumeAction = msg.action
	m.resumeID = msg.id
	m.status = "login required to continue"
}

func (m *Model) handleSyncMsg(msg syncMsg) tea.Cmd {
	m.err = msg.err
	if isSessionExpired(msg.err) {
		m.handleSessionExpired()
		return nil
	}
	if msg.err == nil {
		m.status = fmt.Sprintf("sync complete: %d items", msg.count)
		if strings.TrimSpace(m.content) == "" {
			m.content = m.status
		}
		return m.loadItems
	}
	m.status = "sync failed: " + msg.err.Error()
	return m.loadItems
}

func (m *Model) handleLogoutMsg(msg logoutMsg) tea.Cmd {
	m.err = msg.err
	if msg.err == nil {
		m.resetAfterLogout()
		return m.loadItems
	}
	return nil
}

func (m *Model) handleEditFormMsg(msg editFormMsg) {
	m.err = msg.err
	if isSessionExpired(msg.err) {
		m.handleSessionExpired()
		return
	}
	if isConflict(msg.err) {
		m.status = statusConflict
		return
	}
	if msg.err == nil {
		m.mode = modeForm
		m.formAction = actionEdit
		m.formID = msg.id
		m.form = msg.form
	}
}

func (m *Model) handleFormMsg(msg formMsg) tea.Cmd {
	m.err = msg.err
	if isSessionExpired(msg.err) {
		m.handleSessionExpired()
		return nil
	}
	if isConflict(msg.err) {
		m.status = statusConflict
		return nil
	}
	if msg.err == nil {
		m.mode = modeList
		m.formAction = ""
		m.formID = ""
		return m.loadItems
	}
	return nil
}

func (m *Model) handleDeleteMsg(msg deleteMsg) tea.Cmd {
	m.err = msg.err
	if isSessionExpired(msg.err) {
		m.handleSessionExpired()
		return nil
	}
	if isConflict(msg.err) {
		m.status = statusConflict
		return nil
	}
	if msg.err == nil {
		m.mode = modeList
		m.confirmAction = ""
		m.confirmID = ""
		m.confirmMessage = ""
		return m.loadItems
	}
	return nil
}

func (m *Model) updateActiveForm(msg tea.Msg) bool {
	switch m.mode {
	case modeForm:
		prevType := formTypeValue(m.form)
		for i := range m.form.fields {
			m.form.fields[i].input, _ = m.form.fields[i].input.Update(msg)
		}
		nextType := formTypeValue(m.form)
		if prevType != nextType {
			values := m.form.values()
			m.form = buildForm(m.form.title, values)
		}
		return true
	case modeAuth:
		for i := range m.authForm.fields {
			m.authForm.fields[i].input, _ = m.authForm.fields[i].input.Update(msg)
		}
		return true
	case modeSearch:
		for i := range m.searchForm.fields {
			m.searchForm.fields[i].input, _ = m.searchForm.fields[i].input.Update(msg)
		}
		return true
	default:
		return false
	}
}

// View renders the TUI.
func (m *Model) View() string {
	header := trimTrailingNewlines(renderHeader(m))
	footer := trimTrailingNewlines(renderFooter(m))
	switch m.mode {
	case modeForm:
		content := renderFormWithError(m.form, m.err)
		return trimTrailingNewlines(renderScreen(header, footer, renderPanelContent(m.form.title, content), m))
	case modeConfirm:
		title := "Confirm"
		if m.confirmAction == actionDelete {
			title = "Delete Secret"
		}
		content := m.confirmMessage + "\n\n" + styles.Subtle.Render("[y] confirm  [n]/[esc] cancel")
		return trimTrailingNewlines(renderScreen(header, footer, renderPanelContent(title, content), m))
	case modeAuth:
		content := renderFormWithError(m.authForm, m.err)
		return trimTrailingNewlines(renderScreen(header, footer, renderPanelContent(m.authForm.title, content), m))
	case modeSearch:
		content := renderFormWithError(m.searchForm, m.err)
		return trimTrailingNewlines(renderScreen(header, footer, renderPanelContent(m.searchForm.title, content), m))
	case modeVersion:
		return trimTrailingNewlines(renderScreen(header, footer, renderPanelContent("Version", renderVersion(m)), m))
	default:
		body := trimTrailingNewlines(renderBody(m))
		return trimTrailingNewlines(lipgloss.JoinVertical(lipgloss.Left, header, body, footer))
	}
}

type itemsMsg struct {
	items []list.Item
	err   error
}

type detailMsg struct {
	content string
	err     error
}

type formMsg struct {
	err error
}

type editFormMsg struct {
	form formModel
	id   string
	err  error
}

type deleteMsg struct {
	err error
}

type authMsg struct {
	err      error
	user     string
	password string
	status   string
}

type authNeededMsg struct {
	form   formModel
	action string
	id     string
	err    error
}

type syncMsg struct {
	count int
	err   error
}

type logoutMsg struct {
	err error
}

type secretItem struct {
	id    string
	title string
	desc  string
	typ   secret.Type
}

func (s secretItem) Title() string       { return s.title }
func (s secretItem) Description() string { return s.desc }
func (s secretItem) FilterValue() string { return strings.ToLower(s.title + " " + s.desc + " " + s.id) }

type secretDelegate struct{}

func (d secretDelegate) Height() int  { return 1 }
func (d secretDelegate) Spacing() int { return 0 }
func (d secretDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

func (d secretDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	it, ok := item.(secretItem)
	if !ok {
		return
	}
	selected := index == m.Index()
	titlePrefix := "  "
	titleStyle := styles.Item
	descStyle := styles.ItemDesc
	if selected {
		titlePrefix = "> "
		titleStyle = styles.Selected
		descStyle = styles.SelectedDesc
	}
	width := m.Width()
	if width <= 0 {
		width = 40
	}
	line := titleStyle.Render(titlePrefix + it.title)
	if it.desc != "" {
		line += descStyle.Render(" · " + it.desc)
	}
	line = lipgloss.NewStyle().Width(width).Render(line)
	_, _ = fmt.Fprint(w, line)
}

func (m *Model) loadItems() tea.Msg {
	loggedIn, err := m.service.IsLoggedIn(context.Background())
	if err != nil || !loggedIn || m.password == "" {
		m.loggedIn = false
		m.status = statusNotLoggedIn
		return itemsMsg{items: []list.Item{}}
	}
	m.loggedIn = true
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	items, err := m.service.List(ctx, m.password)
	if err != nil {
		if isAuthError(err) {
			m.status = statusNotLoggedIn
			return itemsMsg{items: nil, err: nil}
		}
		return itemsMsg{err: err}
	}
	listItems := make([]list.Item, 0, len(items))
	for _, it := range items {
		desc := prettyType(it.Type)
		if len(it.Meta) > 0 {
			desc += " · " + metaSummary(it.Meta)
		}
		if it.Err != nil {
			desc += " · decrypt error"
		}
		title := displayTitle(it.Meta, it.ID.String())
		listItems = append(listItems, secretItem{id: it.ID.String(), title: title, desc: desc, typ: it.Type})
	}
	// Apply existing filter if present.
	if m.filterQuery != "" {
		listItems = filterItems(listItems, m.filterQuery)
	}
	return itemsMsg{items: listItems}
}

func (m *Model) loadDetail(id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		uid, err := uuid.Parse(id)
		if err != nil {
			return detailMsg{err: err}
		}
		env, err := m.service.Get(ctx, uid, m.password)
		if err != nil {
			return detailMsg{err: err}
		}
		content := renderDetail(env)
		return detailMsg{content: content}
	}
}

func (m *Model) loadEditForm(id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		uid, err := uuid.Parse(id)
		if err != nil {
			return editFormMsg{err: err}
		}
		env, err := m.service.Get(ctx, uid, m.password)
		if err != nil {
			return editFormMsg{err: err}
		}
		values := valuesFromEnvelope(env)
		form := buildForm("Edit Secret", values)
		return editFormMsg{form: form, id: id}
	}
}

func (m *Model) submitForm() tea.Cmd {
	return func() tea.Msg {
		if m.password == "" {
			return authNeededMsg{
				form:   m.form,
				action: m.formAction,
				id:     m.formID,
				err:    errors.New("login required: press l to login or set GOPHKEEPER_USER_PASS"),
			}
		}
		values := m.form.values()
		meta := parseMeta(values["meta"])
		value, err := buildValue(values, func(raw string) (string, error) {
			return resolveBinaryField(raw, m.deps.ReadFile)
		})
		if err != nil {
			return formMsg{err: err}
		}
		meta = enrichMeta(meta, value)
		var opErr error
		switch m.formAction {
		case actionEdit:
			if m.formID == "" {
				return formMsg{err: errors.New("missing item id")}
			}
			uid, parseErr := uuid.Parse(m.formID)
			if parseErr != nil {
				return formMsg{err: parseErr}
			}
			_, opErr = m.service.Update(context.Background(), appclient.UpdateInput{ID: uid, Value: value, Meta: meta, UserPassword: m.password})
		default:
			_, opErr = m.service.Add(context.Background(), appclient.AddInput{Value: value, Meta: meta, UserPassword: m.password})
		}
		return formMsg{err: opErr}
	}
}

func (m *Model) submitAuth() tea.Cmd {
	return func() tea.Msg {
		values := m.authForm.values()
		username := values["username"]
		password := values["password"]
		if username == "" || password == "" {
			return authMsg{err: errors.New("username and password required")}
		}
		if m.authAction == actionRegister {
			confirm := values["confirm"]
			if confirm == "" {
				return authMsg{err: errors.New("confirm password required")}
			}
			if confirm != password {
				return authMsg{err: errors.New("passwords do not match")}
			}
		}
		var err error
		switch m.authAction {
		case actionRegister:
			err = m.service.Register(context.Background(), username, password)
		default:
			err = m.service.Login(context.Background(), username, password)
		}
		status := fmt.Sprintf("%s successful for %s. syncing...", m.authAction, username)
		return authMsg{err: err, user: username, password: password, status: status}
	}
}

func (m *Model) syncItems() tea.Cmd {
	return func() tea.Msg {
		count, err := m.service.Sync(context.Background())
		return syncMsg{count: count, err: err}
	}
}

func (m *Model) logout() tea.Cmd {
	return func() tea.Msg {
		if err := m.service.Logout(context.Background()); err != nil {
			return logoutMsg{err: err}
		}
		return logoutMsg{}
	}
}

func (m *Model) deleteItem(id string) tea.Cmd {
	return func() tea.Msg {
		uid, err := uuid.Parse(id)
		if err != nil {
			return deleteMsg{err: err}
		}
		if err := m.service.Delete(context.Background(), uid); err != nil {
			return deleteMsg{err: err}
		}
		return deleteMsg{}
	}
}

func parseMeta(raw string) map[string]string {
	res := map[string]string{}
	if raw == "" {
		return res
	}
	pairs := strings.Split(raw, ",")
	for _, p := range pairs {
		parts := strings.SplitN(p, "=", 2)
		if len(parts) != 2 {
			continue
		}
		res[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return res
}

func defaultFormLabels() []string {
	return []string{
		"type (text/login/binary/card)",
		"meta (k=v,...)",
	}
}

func defaultFormKeys() []string {
	return []string{"type", "meta"}
}

func buildValue(values map[string]string, binaryResolver func(string) (string, error)) (secret.Value, error) {
	typ, err := secret.ParseType(values["type"])
	if err != nil {
		return nil, err
	}
	switch typ {
	case secret.TypeText:
		return secret.Text{Text: values["field1"]}, nil
	case secret.TypeLoginPassword:
		return secret.LoginPassword{Login: values["field1"], Password: values["field2"]}, nil
	case secret.TypeBinary:
		base64Val := values["field1"]
		if binaryResolver != nil {
			resolved, err := binaryResolver(base64Val)
			if err != nil {
				return nil, err
			}
			base64Val = resolved
		}
		return secret.Binary{Base64: base64Val}, nil
	case secret.TypeCard:
		return secret.Card{Number: values["field1"], Expiry: values["field2"], Holder: values["field3"], CVV: values["field4"]}, nil
	default:
		return nil, errors.New("unknown type")
	}
}

func renderForm(form formModel) string {
	b := strings.Builder{}
	if hasTypeField(form) {
		if hint := formHint(form); hint != "" {
			b.WriteString(styles.Subtle.Render(hint) + "\n")
		}
	}
	labelWidth := 0
	for i := range form.fields {
		field := &form.fields[i]
		if w := lipgloss.Width(field.label); w > labelWidth {
			labelWidth = w
		}
	}
	for i := range form.fields {
		field := &form.fields[i]
		label := field.label
		if pad := labelWidth - lipgloss.Width(label); pad > 0 {
			label += strings.Repeat(" ", pad)
		}
		b.WriteString(styles.Subtle.Render(label) + ": " + field.input.View() + "\n")
	}
	b.WriteString(styles.Subtle.Render("[tab] next  [shift+tab] prev  [enter] submit  [esc] cancel") + "\n")
	return b.String()
}

func renderFormWithError(form formModel, err error) string {
	out := renderForm(form)
	if err == nil {
		return out
	}
	errLine := styles.Error.Render("error: " + err.Error())
	return errLine + "\n\n" + out
}

func padLines(content string, width int) string {
	if width <= 0 {
		return content
	}
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if lipgloss.Width(line) > width {
			lines[i] = reflowtruncate.StringWithTail(line, uint(width), "...")
			line = lines[i]
		}
		if pad := width - lipgloss.Width(line); pad > 0 {
			lines[i] = line + strings.Repeat(" ", pad)
		}
	}
	return strings.Join(lines, "\n")
}

func trimTrailingNewlines(s string) string {
	return strings.TrimRight(s, "\n")
}

func fitHeight(content string, target int) string {
	if target <= 0 {
		return content
	}
	lines := strings.Split(content, "\n")
	if len(lines) > target {
		lines = lines[:target]
	}
	if len(lines) < target {
		lines = append(lines, make([]string, target-len(lines))...)
	}
	return strings.Join(lines, "\n")
}

func renderScreen(header, footer, content string, m *Model) string {
	width := m.width
	if width == 0 {
		width = 100
	}
	panelWidth := minInt(maxInt(40, width-4), width-2)
	panelHeight := m.bodyHeight()
	frameX, frameY := styles.Panel.GetFrameSize()
	contentWidth := panelWidth - frameX
	if contentWidth > 0 {
		content = padLines(content, contentWidth)
	}
	if panelHeight > frameY {
		content = fitHeight(content, panelHeight-frameY)
	}
	if contentWidth < 1 {
		contentWidth = 1
	}
	panel := trimTrailingNewlines(styles.Panel.Width(contentWidth).Render(content))
	return lipgloss.JoinVertical(lipgloss.Left, header, panel, footer)
}

func renderVersion(m *Model) string {
	content := strings.TrimSpace(m.versionInfo)
	if content == "" {
		content = "version: unknown\nbuild date: unknown"
	}
	hint := styles.Subtle.Render("[esc] back")
	return content + "\n\n" + hint
}

func renderPanelContent(title, content string) string {
	if strings.TrimSpace(title) == "" {
		return content
	}
	line := styles.Subtle.Render(strings.Repeat("─", 24))
	titleStyle := styles.Subtle.Bold(true)
	return titleStyle.Render(title) + "\n" + line + "\n" + content
}

func formHint(form formModel) string {
	values := form.values()
	typ, err := secret.ParseType(values["type"])
	if err != nil {
		typ = ""
	}
	switch typ {
	case secret.TypeText:
		return "fields: field1=text"
	case secret.TypeLoginPassword:
		return "fields: field1=login, field2=password"
	case secret.TypeBinary:
		return "fields: field1=file:/path or b64:base64"
	case secret.TypeCard:
		return "fields: field1=number, field2=expiry, field3=holder, field4=cvv"
	default:
		return "fields: set type to see hints"
	}
}

func valuesFromEnvelope(env appclient.PayloadEnvelope) map[string]string {
	values := map[string]string{
		"type": prettyType(env.Type),
		"meta": formatMeta(env.Meta),
	}
	switch env.Type {
	case secret.TypeText:
		values["field1"] = env.Data["text"]
	case secret.TypeLoginPassword:
		values["field1"] = env.Data["login"]
		values["field2"] = env.Data["password"]
	case secret.TypeBinary:
		if env.Data["file"] != "" {
			values["field1"] = "b64:" + env.Data["file"]
		}
	case secret.TypeCard:
		values["field1"] = env.Data["number"]
		values["field2"] = env.Data["expiry"]
		values["field3"] = env.Data["holder"]
		values["field4"] = env.Data["cvv"]
	}
	return values
}

func buildForm(title string, values map[string]string) formModel {
	typ, err := secret.ParseType(values["type"])
	if err != nil {
		typ = ""
	}
	labels := defaultFormLabels()
	keys := defaultFormKeys()
	switch typ {
	case secret.TypeText:
		labels = append(labels, "text")
		keys = append(keys, "field1")
	case secret.TypeLoginPassword:
		labels = append(labels, "login", "password")
		keys = append(keys, "field1", "field2")
	case secret.TypeBinary:
		labels = append(labels, "file:/path or b64:base64")
		keys = append(keys, "field1")
	case secret.TypeCard:
		labels = append(labels, "number", "expiry", "holder", "cvv")
		keys = append(keys, "field1", "field2", "field3", "field4")
	default:
	}
	return newFormWithValues(title, labels, keys, values)
}

func formTypeValue(form formModel) string {
	for i := range form.fields {
		field := &form.fields[i]
		if field.key == "type" {
			return strings.TrimSpace(field.input.Value())
		}
	}
	return ""
}

func hasTypeField(form formModel) bool {
	for i := range form.fields {
		field := &form.fields[i]
		if field.key == "type" {
			return true
		}
	}
	return false
}

func formatMeta(meta map[string]string) string {
	if len(meta) == 0 {
		return ""
	}
	parts := make([]string, 0, len(meta))
	for k, v := range meta {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(parts, ", ")
}

func formatMetaWithout(meta map[string]string, skip map[string]bool) string {
	if len(meta) == 0 {
		return ""
	}
	parts := make([]string, 0, len(meta))
	for k, v := range meta {
		if skip[k] {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(parts, ", ")
}

func metaSummary(meta map[string]string) string {
	if len(meta) == 0 {
		return ""
	}
	return formatMetaWithout(meta, map[string]bool{"label": true, "title": true, "name": true})
}

func newAuthForm(title string) formModel {
	fields := []string{"username", "password"}
	keys := []string{"username", "password"}
	if strings.Contains(strings.ToLower(title), actionRegister) {
		fields = append(fields, "confirm password")
		keys = append(keys, "confirm")
	}
	form := newForm(title, fields, keys)
	for i := range form.fields {
		if form.fields[i].key == "password" {
			form.fields[i].input.EchoMode = textinput.EchoPassword
			form.fields[i].input.EchoCharacter = '*'
		}
		if form.fields[i].key == "confirm" {
			form.fields[i].input.EchoMode = textinput.EchoPassword
			form.fields[i].input.EchoCharacter = '*'
		}
	}
	return form
}

func newSearchForm(title string) formModel {
	return newForm(title, []string{"query"}, []string{"query"})
}

func resolveBinaryField(raw string, readFile func(string) ([]byte, error)) (string, error) {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "b64:") {
		return strings.TrimPrefix(raw, "b64:"), nil
	}
	if strings.HasPrefix(raw, "file:") {
		if readFile == nil {
			return "", errors.New("file reader not configured")
		}
		path := strings.TrimPrefix(raw, "file:")
		data, err := readFile(path)
		if err != nil {
			return "", err
		}
		return appclient.EncodePayload(data), nil
	}
	if readFile != nil {
		if data, err := readFile(raw); err == nil {
			return appclient.EncodePayload(data), nil
		}
	}
	if _, err := appclient.DecodePayload(raw); err == nil {
		return raw, nil
	}
	return "", errors.New("binary field must be file:/path or b64:base64")
}

func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not logged in") ||
		strings.Contains(msg, "unauthorized") ||
		errors.Is(err, appclient.ErrSessionExpired)
}

func isSessionExpired(err error) bool {
	return errors.Is(err, appclient.ErrSessionExpired)
}

func isConflict(err error) bool {
	return errors.Is(err, appclient.ErrConflict)
}

func renderSide(m *Model, panelWidth, contentHeight int) string {
	content := m.content
	if content == "" {
		content = "Select an item to view details."
	}
	if m.filterQuery != "" {
		content = "Filter: " + m.filterQuery + "\n" + strings.Repeat("─", 16) + "\n\n" + content
	}
	if m.status != "" && strings.TrimSpace(content) != strings.TrimSpace(m.status) {
		content = content + "\n\n" + styles.Status.Render(m.status)
	}
	if m.err != nil {
		content = content + "\n\n" + styles.Error.Render("error: "+m.err.Error())
	}
	body := renderPanelContent("Info", content)
	frameX, _ := styles.Panel.GetFrameSize()
	contentWidth := panelWidth - frameX
	if contentWidth > 0 {
		body = padLines(body, contentWidth)
	}
	if contentHeight > 0 {
		body = fitHeight(body, contentHeight)
	}
	if contentWidth < 1 {
		contentWidth = 1
	}
	return styles.Panel.Width(contentWidth).Render(body)
}

func renderHeader(m *Model) string {
	width := m.width
	if width == 0 {
		width = 100
	}
	session := "Session: none"
	if m.loggedIn {
		if m.loggedUser != "" {
			session = "Session: " + m.loggedUser
		} else {
			session = "Session: active"
		}
	}
	title := styles.Header.Render("GophKeeper")
	sub := styles.Subtle.Render("Password Manager · " + session)
	line := styles.Subtle.Render(strings.Repeat("─", minInt(width-2, 80)))
	block := title + "\n" + sub + "\n" + line
	return padLines(block, width)
}

func renderFooter(m *Model) string {
	lines := []string{
		"l login  r register  x logout  s sync  v version",
		"a add  e edit  d delete  enter view  q quit",
		"f filter  c clear-filter  esc back (compact view)",
	}
	out := styles.Subtle.Render(strings.Join(lines, "\n"))
	if m.width > 0 {
		panelWidth := maxInt(4, m.width)
		frameX, _ := styles.Panel.GetFrameSize()
		contentWidth := panelWidth - frameX
		if contentWidth > 0 {
			out = padLines(out, contentWidth)
		}
		if contentWidth < 1 {
			contentWidth = 1
		}
		return styles.Panel.Width(contentWidth).Render(out)
	}
	return out
}

func renderBody(m *Model) string {
	contentHeight := contentHeightForBody(m)
	statusLines, extraLines := listStatusLines(m)
	listHeight := maxInt(1, contentHeight-extraLines)
	m.list.SetSize(m.list.Width(), listHeight)

	leftContent := renderLeftContent(m, statusLines)
	leftWidth, rightWidth := layoutWidths(m.width)
	left := renderPanelWithWidth(leftContent, leftWidth, contentHeight)

	if m.compact() {
		if m.panel == panelDetail {
			return renderSide(m, leftWidth, contentHeight)
		}
		return left
	}
	if rightWidth == 0 {
		return left
	}
	right := renderSide(m, rightWidth, contentHeight)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func contentHeightForBody(m *Model) int {
	contentHeight := m.bodyHeight()
	_, frameY := styles.Panel.GetFrameSize()
	if contentHeight > frameY {
		contentHeight -= frameY
	}
	return maxInt(1, contentHeight)
}

func listStatusLines(m *Model) (string, int) {
	if !m.compact() || m.panel != modeList || !m.loggedIn {
		return "", 0
	}
	extraLines := 0
	statusLines := ""
	if m.status != "" {
		statusLines = styles.Status.Render(m.status)
		extraLines++
	}
	if m.err != nil {
		if statusLines != "" {
			statusLines += "\n"
		}
		statusLines += styles.Error.Render("error: " + m.err.Error())
		extraLines++
	}
	return statusLines, extraLines
}

func renderLeftContent(m *Model, statusLines string) string {
	leftContent := m.list.View()
	if !m.loggedIn {
		return styles.Subtle.Render("Not logged in.\nPress l to login or r to register.")
	}
	if statusLines != "" {
		leftContent = leftContent + "\n" + statusLines
	}
	return leftContent
}

func renderPanelWithWidth(content string, panelWidth, contentHeight int) string {
	frameX, _ := styles.Panel.GetFrameSize()
	contentWidth := panelWidth - frameX
	if contentWidth > 0 {
		content = padLines(content, contentWidth)
	}
	if contentHeight > 0 {
		content = fitHeight(content, contentHeight)
	}
	if contentWidth < 1 {
		contentWidth = 1
	}
	return styles.Panel.Width(contentWidth).Render(content)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func layoutWidths(total int) (int, int) {
	if total == 0 {
		total = 100
	}
	left := total / 2
	if left < 40 {
		left = total
	}
	right := total - left
	if right < 30 {
		right = 0
	}
	return left, right
}

func (m *Model) compact() bool {
	if m.width == 0 || m.height == 0 {
		return false
	}
	return m.width < 90 || m.height < 24
}

func (m *Model) bodyHeight() int {
	if m.height > 0 {
		headerH := lipgloss.Height(renderHeader(m))
		footerH := lipgloss.Height(renderFooter(m))
		return maxInt(1, m.height-headerH-footerH-3)
	}
	if h := m.list.Height(); h > 0 {
		_, frameY := styles.Panel.GetFrameSize()
		return h + frameY
	}
	return 0
}

func (m *Model) setAuthResume(action, resumeAction, resumeID string) {
	m.mode = modeAuth
	if action == "" {
		action = actionLogin
	}
	m.authAction = action
	title := "Login"
	if action == actionRegister {
		title = "Register"
	}
	m.authForm = newAuthForm(title)
	m.resumeAction = resumeAction
	m.resumeID = resumeID
}

func (m *Model) resumeListAction(action, id string) tea.Cmd {
	if id == "" {
		item, ok := m.list.SelectedItem().(secretItem)
		if ok {
			id = item.id
		}
	}
	switch action {
	case actionSync:
		return m.syncItems()
	case actionAdd:
		m.mode = modeForm
		m.formAction = actionAdd
		m.formID = ""
		m.form = buildForm("Add Secret", map[string]string{})
		return nil
	case actionDelete:
		if id != "" {
			m.mode = modeConfirm
			m.confirmAction = actionDelete
			m.confirmID = id
			m.confirmMessage = fmt.Sprintf("Delete secret %s?", id)
		}
		return nil
	case actionDetail:
		if id != "" {
			return m.loadDetail(id)
		}
	case actionEdit:
		if id != "" {
			return m.loadEditForm(id)
		}
	}
	return m.loadItems
}

func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

func displayTitle(meta map[string]string, id string) string {
	if meta != nil {
		if v := strings.TrimSpace(meta["label"]); v != "" {
			return v
		}
		if v := strings.TrimSpace(meta["site"]); v != "" {
			return v
		}
		if v := strings.TrimSpace(meta["name"]); v != "" {
			return v
		}
		if v := strings.TrimSpace(meta["title"]); v != "" {
			return v
		}
	}
	return shortID(id)
}

func renderDetail(env appclient.PayloadEnvelope) string {
	typ := prettyType(env.Type)
	meta := formatMeta(env.Meta)
	var b strings.Builder
	b.WriteString("type: " + typ + "\n")
	if meta == "" {
		b.WriteString("meta: -\n")
	} else {
		b.WriteString("meta: " + meta + "\n")
	}
	b.WriteString("data:\n")
	switch env.Type {
	case secret.TypeLoginPassword:
		b.WriteString("  login: " + env.Data["login"] + "\n")
		b.WriteString("  password: " + env.Data["password"] + "\n")
	case secret.TypeText:
		b.WriteString("  text: " + env.Data["text"] + "\n")
	case secret.TypeBinary:
		if v := env.Data["file"]; v != "" {
			b.WriteString("  file: " + v + "\n")
		} else {
			b.WriteString("  b64: " + env.Data["b64"] + "\n")
		}
	case secret.TypeCard:
		b.WriteString("  number: " + env.Data["number"] + "\n")
		b.WriteString("  expiry: " + env.Data["expiry"] + "\n")
		b.WriteString("  holder: " + env.Data["holder"] + "\n")
		b.WriteString("  cvv: " + env.Data["cvv"] + "\n")
	default:
		for k, v := range env.Data {
			b.WriteString("  " + k + ": " + v + "\n")
		}
	}
	return b.String()
}

func prettyType(t secret.Type) string {
	if t == secret.TypeLoginPassword {
		return "login"
	}
	return string(t)
}

func enrichMeta(meta map[string]string, value secret.Value) map[string]string {
	if meta == nil {
		meta = map[string]string{}
	}
	if hasDisplayMeta(meta) {
		return meta
	}
	switch v := value.(type) {
	case secret.LoginPassword:
		if v.Login != "" {
			meta["label"] = v.Login
		}
	case secret.Text:
		if v.Text != "" {
			meta["label"] = truncate(v.Text, 24)
		}
	case secret.Card:
		if v.Number != "" {
			meta["label"] = "card ••" + last4(v.Number)
		}
	case secret.Binary:
		meta["label"] = "binary"
	}
	return meta
}

func hasDisplayMeta(meta map[string]string) bool {
	for _, key := range []string{"label", "title", "name", "site"} {
		if strings.TrimSpace(meta[key]) != "" {
			return true
		}
	}
	return false
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func last4(s string) string {
	if len(s) <= 4 {
		return s
	}
	return s[len(s)-4:]
}

func (m *Model) applyFilter(query string) {
	m.filterQuery = strings.TrimSpace(query)
	if m.filterQuery == "" {
		m.list.SetItems(m.allItems)
		m.list.Title = fmt.Sprintf("Secrets (%d)", len(m.allItems))
		m.status = "filter cleared"
		return
	}
	filtered := filterItems(m.allItems, m.filterQuery)
	m.list.SetItems(filtered)
	m.list.Title = fmt.Sprintf("Secrets (%d)", len(filtered))
	m.status = fmt.Sprintf("filter: %s (%d/%d)", m.filterQuery, len(filtered), len(m.allItems))
}

func filterItems(items []list.Item, query string) []list.Item {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return items
	}
	filtered := make([]list.Item, 0, len(items))
	for _, it := range items {
		if strings.Contains(it.FilterValue(), q) {
			filtered = append(filtered, it)
		}
	}
	return filtered
}

// Start runs the TUI program.
func Start(model *Model) error {
	_, err := tea.NewProgram(model, tea.WithAltScreen()).Run()
	return err
}

func (m *Model) resetAfterLogout() {
	m.password = ""
	m.loggedIn = false
	m.loggedUser = ""
	m.status = statusNotLoggedIn
	m.content = ""
	m.err = nil
	m.filterQuery = ""
	m.list.SetItems([]list.Item{})
	m.list.Title = "Secrets (0)"
	m.allItems = nil
	m.mode = modeList
	m.panel = modeList
	m.form = formModel{}
	m.formAction = ""
	m.formID = ""
	m.confirmAction = ""
	m.confirmID = ""
	m.confirmMessage = ""
	m.resumeForm = formModel{}
	m.resumeAction = ""
	m.resumeID = ""
	m.authAction = ""
	m.authForm = formModel{}
	m.searchForm = formModel{}
	m.versionInfo = ""
}

func (m *Model) handleSessionExpired() {
	m.resetAfterLogout()
	m.status = "session expired: press l to login"
}
