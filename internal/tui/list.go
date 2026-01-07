package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		// Handle 'gg' key sequence for jump to top
		if m.pendingKey == "g" {
			m.pendingKey = ""
			if key == "g" {
				// 'gg' - jump to top
				m.cursor = 0
				return m, nil
			}
			// Not 'gg', continue processing this key normally
		}

		switch key {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.subscriptions)-1 {
				m.cursor++
			}
		case "g":
			// Wait for second 'g' for 'gg' command
			m.pendingKey = "g"
			return m, nil
		case "G":
			// Jump to bottom
			if len(m.subscriptions) > 0 {
				m.cursor = len(m.subscriptions) - 1
			}
		case "a":
			m.view = ViewAdd
			m.addForm = NewAddForm()
			return m, m.addForm.Init()
		case "e":
			if len(m.subscriptions) > 0 {
				m.view = ViewEdit
				m.editForm = NewEditForm()
				m.editForm.LoadSubscription(m.subscriptions[m.cursor])
				return m, m.editForm.Init()
			}
		case "d":
			if len(m.subscriptions) > 0 {
				return m, m.deleteSubscription(m.subscriptions[m.cursor].ID)
			}
		case "s":
			m.view = ViewSpending
			m.spendingView = NewSpendingView()
			return m, m.spendingView.Init(m.app)
		case "x":
			m.view = ViewExport
			m.exportView = NewExportView()
			return m, nil
		case "c":
			m.view = ViewConfig
			m.configView = NewConfigView()
			return m, m.configView.Init(m.app)
		case "y":
			m.view = ViewSync
			m.syncView = NewSyncView()
			return m, m.syncView.Init(m.app)
		case "?":
			m.view = ViewHelp
			return m, nil
		case "r":
			return m, m.loadSubscriptions
		}
	}
	return m, nil
}

func (m Model) viewList() string {
	var b strings.Builder

	// Title
	title := TitleStyle.Render("Subscription Tracker")
	b.WriteString(title + "\n\n")

	// Message
	if m.message != "" {
		b.WriteString(SuccessStyle.Render(m.message) + "\n\n")
	}

	// Error
	if m.err != nil {
		b.WriteString(ErrorStyle.Render("Error: "+m.err.Error()) + "\n\n")
	}

	// Subscriptions list
	if len(m.subscriptions) == 0 {
		b.WriteString(SubtitleStyle.Render("No subscriptions yet. Press 'a' to add one."))
	} else {
		// Header
		header := fmt.Sprintf("%-4s %-25s %-12s %-10s %-12s",
			"ID", "Name", "Amount", "Cycle", "Renewal")
		b.WriteString(TableHeaderStyle.Render(header) + "\n")

		// Rows
		for i, sub := range m.subscriptions {
			cycle := sub.BillingCycle
			if cycle == "monthly" {
				cycle = MonthlyStyle.Render(cycle)
			} else {
				cycle = YearlyStyle.Render(cycle)
			}

			renewal := "-"
			if sub.NextRenewalDate.Valid {
				renewal = sub.NextRenewalDate.String
			}

			row := fmt.Sprintf("%-4d %-25s %-12s %-10s %-12s",
				sub.ID,
				truncate(sub.Name, 25),
				fmt.Sprintf("%.2f %s", sub.Amount, sub.Currency),
				sub.BillingCycle,
				renewal,
			)

			if i == m.cursor {
				row = SelectedItemStyle.Render(row)
			} else {
				row = NormalItemStyle.Render(row)
			}
			b.WriteString(row + "\n")
		}
	}

	// Help
	help := "\n[↑/↓] navigate  [gg/G] top/bottom  [a]dd  [e]dit  [d]elete  [s]pending  e[x]port  [c]onfig  s[y]nc  [?]help  [q]uit"
	b.WriteString(HelpStyle.Render(help))

	return BoxStyle.Render(b.String())
}

func (m Model) deleteSubscription(id int64) tea.Cmd {
	return func() tea.Msg {
		err := m.app.Queries.DeleteSubscription(context.Background(), id)
		if err != nil {
			return errMsg{err}
		}
		return successMsg{"Subscription deleted successfully"}
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s + strings.Repeat(" ", maxLen-len(s))
	}
	return s[:maxLen-3] + "..."
}

// Ensure BoxStyle exists or create a simple one
var _ = lipgloss.NewStyle()
