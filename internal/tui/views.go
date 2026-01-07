package tui

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"subscription-tracker/internal/db"
)

// updateAdd handles updates for the add form view
func (m Model) updateAdd(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" {
			m.view = ViewList
			return m, nil
		}
	case createSubscriptionMsg:
		_, err := m.app.Queries.CreateSubscription(context.Background(), msg.params)
		if err != nil {
			m.err = err
			return m, nil
		}
		m.message = "Subscription added successfully"
		m.view = ViewList
		return m, m.loadSubscriptions
	}

	done, cmd := m.addForm.Update(msg, m.app)
	if done {
		// Check if we got a create message
		return m, cmd
	}
	return m, cmd
}

// updateEdit handles updates for the edit form view
func (m Model) updateEdit(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" {
			m.view = ViewList
			return m, nil
		}
	case updateSubscriptionMsg:
		_, err := m.app.Queries.UpdateSubscription(context.Background(), msg.params)
		if err != nil {
			m.err = err
			return m, nil
		}
		m.message = "Subscription updated successfully"
		m.view = ViewList
		return m, m.loadSubscriptions
	}

	done, cmd := m.editForm.Update(msg, m.app)
	if done {
		return m, cmd
	}
	return m, cmd
}

// updateSpending handles updates for the spending view
func (m Model) updateSpending(msg tea.Msg) (tea.Model, tea.Cmd) {
	done, cmd := m.spendingView.Update(msg, m.app)
	if done {
		m.view = ViewList
		return m, nil
	}
	return m, cmd
}

// updateExport handles updates for the export view
func (m Model) updateExport(msg tea.Msg) (tea.Model, tea.Cmd) {
	done, cmd := m.exportView.Update(msg, m.app)
	if done {
		m.view = ViewList
		return m, nil
	}
	return m, cmd
}

// updateConfig handles updates for the config view
func (m Model) updateConfig(msg tea.Msg) (tea.Model, tea.Cmd) {
	done, cmd := m.configView.Update(msg, m.app)
	if done {
		m.view = ViewList
		return m, nil
	}
	return m, cmd
}

// updateHelp handles updates for the help view
func (m Model) updateHelp(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "?":
			m.view = ViewList
			return m, nil
		}
	}
	return m, nil
}

// viewAdd renders the add form
func (m Model) viewAdd() string {
	return m.addForm.View()
}

// viewEdit renders the edit form
func (m Model) viewEdit() string {
	return m.editForm.View()
}

// viewSpending renders the spending view
func (m Model) viewSpending() string {
	return m.spendingView.View()
}

// viewExport renders the export view
func (m Model) viewExport() string {
	return m.exportView.View()
}

// viewConfig renders the config view
func (m Model) viewConfig() string {
	return m.configView.View()
}

// viewHelp renders the help view
func (m Model) viewHelp() string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render("Help") + "\n\n")

	help := `Keyboard Shortcuts:

List View (VIM motions supported):
  ↓/j      Move cursor down
  ↑/k      Move cursor up
  gg       Jump to first item
  G        Jump to last item
  a        Add new subscription
  e        Edit selected subscription
  d        Delete selected subscription
  s        View spending summary
  x        Export subscriptions
  c        Configuration (payday, salary)
  y        Sync to GitHub Gist (encrypted)
  r        Refresh list
  ?        Show this help
  q        Quit

Add/Edit Form:
  ↓/Tab    Next field
  ↑/Shift+Tab  Previous field
  ←/→      Toggle billing cycle (monthly/yearly)
  Ctrl+S   Save
  Esc      Cancel

Spending View:
  ←/h      Previous month
  →/l      Next month
  q/Esc    Back to list

Export View:
  Tab      Change format (CSV/JSON)
  Enter    Export
  q/Esc    Cancel

Sync View:
  ↓/Tab    Next field
  ↑/Shift+Tab  Previous field
  Ctrl+P   Push to GitHub Gist
  Ctrl+L   Pull from GitHub Gist
  q/Esc    Cancel

Config:
  ↓/Tab    Next field
  ↑/Shift+Tab  Previous field
  Ctrl+S   Save
  q/Esc    Cancel
`
	b.WriteString(help)
	b.WriteString("\n" + HelpStyle.Render("[q/esc/?] back to list"))

	return BoxStyle.Render(b.String())
}

// Message type for creating subscriptions from add form
type createSubscriptionMsg struct {
	params db.CreateSubscriptionParams
}
