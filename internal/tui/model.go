package tui

import (
	"context"
	"subscription-tracker/internal/app"
	"subscription-tracker/internal/db"

	tea "github.com/charmbracelet/bubbletea"
)

// View represents the current screen
type View int

const (
	ViewList View = iota
	ViewAdd
	ViewEdit
	ViewSpending
	ViewExport
	ViewConfig
	ViewSync
	ViewHelp
)

// Model is the main application model
type Model struct {
	app           *app.App
	view          View
	subscriptions []db.Subscription
	cursor        int
	width         int
	height        int
	err           error
	message       string
	pendingKey    string // For VIM key sequences like 'gg'

	// Sub-models
	addForm      *AddForm
	editForm     *EditForm
	spendingView *SpendingView
	exportView   *ExportView
	configView   *ConfigView
	syncView     *SyncView
}

// New creates a new TUI model
func New(application *app.App) Model {
	return Model{
		app:          application,
		view:         ViewList,
		addForm:      NewAddForm(),
		editForm:     NewEditForm(),
		spendingView: NewSpendingView(),
		exportView:   NewExportView(),
		configView:   NewConfigView(),
		syncView:     NewSyncView(),
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return m.loadSubscriptions
}

// loadSubscriptions fetches subscriptions from the database
func (m Model) loadSubscriptions() tea.Msg {
	subs, err := m.app.Queries.ListSubscriptions(context.Background())
	if err != nil {
		return errMsg{err}
	}
	return subscriptionsLoadedMsg{subs}
}

// Messages
type subscriptionsLoadedMsg struct {
	subscriptions []db.Subscription
}

type errMsg struct {
	err error
}

type successMsg struct {
	message string
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global key bindings
		switch msg.String() {
		case "ctrl+c", "q":
			if m.view == ViewList {
				return m, tea.Quit
			}
			// Return to list view from any other view
			m.view = ViewList
			m.message = ""
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case subscriptionsLoadedMsg:
		m.subscriptions = msg.subscriptions
		m.err = nil
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil

	case successMsg:
		m.message = msg.message
		m.view = ViewList
		return m, m.loadSubscriptions
	}

	// Delegate to the appropriate view
	switch m.view {
	case ViewList:
		return m.updateList(msg)
	case ViewAdd:
		return m.updateAdd(msg)
	case ViewEdit:
		return m.updateEdit(msg)
	case ViewSpending:
		return m.updateSpending(msg)
	case ViewExport:
		return m.updateExport(msg)
	case ViewConfig:
		return m.updateConfig(msg)
	case ViewSync:
		return m.updateSync(msg)
	case ViewHelp:
		return m.updateHelp(msg)
	}

	return m, nil
}

// View renders the current view
func (m Model) View() string {
	switch m.view {
	case ViewList:
		return m.viewList()
	case ViewAdd:
		return m.viewAdd()
	case ViewEdit:
		return m.viewEdit()
	case ViewSpending:
		return m.viewSpending()
	case ViewExport:
		return m.viewExport()
	case ViewConfig:
		return m.viewConfig()
	case ViewSync:
		return m.viewSync()
	case ViewHelp:
		return m.viewHelp()
	}
	return ""
}

// updateSync handles sync view updates
func (m Model) updateSync(msg tea.Msg) (tea.Model, tea.Cmd) {
	done, cmd := m.syncView.Update(msg, m.app)
	if done {
		m.view = ViewList
		return m, m.loadSubscriptions
	}
	return m, cmd
}

// viewSync renders the sync view
func (m Model) viewSync() string {
	return m.syncView.View()
}
