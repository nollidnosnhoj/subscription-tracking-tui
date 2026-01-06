package tui

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"subscription-tracker/internal/app"
	"subscription-tracker/internal/db"
)

type ExportView struct {
	formatIndex int // 0 = CSV, 1 = JSON
	pathInput   textinput.Model
	message     string
	err         error
	exported    bool
}

var exportFormats = []string{"CSV", "JSON"}

func NewExportView() *ExportView {
	pathInput := textinput.New()
	pathInput.Placeholder = "subscriptions.csv"
	pathInput.Focus()
	pathInput.CharLimit = 100
	pathInput.Width = 40
	pathInput.Prompt = "File path: "
	pathInput.SetValue("subscriptions.csv")

	return &ExportView{
		formatIndex: 0,
		pathInput:   pathInput,
	}
}

func (v *ExportView) Init() tea.Cmd {
	return textinput.Blink
}

func (v *ExportView) Update(msg tea.Msg, a *app.App) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			v.formatIndex = (v.formatIndex + 1) % len(exportFormats)
			// Update file extension
			path := v.pathInput.Value()
			if v.formatIndex == 0 {
				path = strings.TrimSuffix(path, ".json") + ".csv"
			} else {
				path = strings.TrimSuffix(path, ".csv") + ".json"
			}
			v.pathInput.SetValue(path)
			return false, nil
		case "enter", "ctrl+s":
			return false, v.export(a)
		case "q", "esc":
			return true, nil
		}
	case exportDoneMsg:
		v.message = msg.message
		v.exported = true
		return false, nil
	case exportErrMsg:
		v.err = msg.err
		return false, nil
	}

	var cmd tea.Cmd
	v.pathInput, cmd = v.pathInput.Update(msg)
	return false, cmd
}

type exportDoneMsg struct {
	message string
}

type exportErrMsg struct {
	err error
}

func (v *ExportView) export(a *app.App) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		subs, err := a.Queries.GetAllSubscriptionsForExport(ctx)
		if err != nil {
			return exportErrMsg{err}
		}

		if len(subs) == 0 {
			return exportErrMsg{fmt.Errorf("no subscriptions to export")}
		}

		path := v.pathInput.Value()
		if path == "" {
			if v.formatIndex == 0 {
				path = "subscriptions.csv"
			} else {
				path = "subscriptions.json"
			}
		}

		file, err := os.Create(path)
		if err != nil {
			return exportErrMsg{fmt.Errorf("failed to create file: %w", err)}
		}
		defer file.Close()

		if v.formatIndex == 0 {
			err = exportCSV(file, subs)
		} else {
			err = exportJSON(file, subs)
		}

		if err != nil {
			return exportErrMsg{err}
		}

		return exportDoneMsg{fmt.Sprintf("Exported %d subscriptions to %s", len(subs), path)}
	}
}

func exportCSV(file *os.File, subs []db.Subscription) error {
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Header
	header := []string{"ID", "Name", "Amount", "Currency", "Billing Cycle", "Next Renewal Date", "Created At", "Updated At"}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Rows
	for _, s := range subs {
		renewalDate := ""
		if s.NextRenewalDate.Valid {
			renewalDate = s.NextRenewalDate.String
		}

		row := []string{
			fmt.Sprintf("%d", s.ID),
			s.Name,
			fmt.Sprintf("%.2f", s.Amount),
			s.Currency,
			s.BillingCycle,
			renewalDate,
			s.CreatedAt,
			s.UpdatedAt,
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

type exportSubscription struct {
	ID              int64   `json:"id"`
	Name            string  `json:"name"`
	Amount          float64 `json:"amount"`
	Currency        string  `json:"currency"`
	BillingCycle    string  `json:"billing_cycle"`
	NextRenewalDate string  `json:"next_renewal_date,omitempty"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

func exportJSON(file *os.File, subs []db.Subscription) error {
	var exportData []exportSubscription

	for _, s := range subs {
		renewalDate := ""
		if s.NextRenewalDate.Valid {
			renewalDate = s.NextRenewalDate.String
		}

		exportData = append(exportData, exportSubscription{
			ID:              s.ID,
			Name:            s.Name,
			Amount:          s.Amount,
			Currency:        s.Currency,
			BillingCycle:    s.BillingCycle,
			NextRenewalDate: renewalDate,
			CreatedAt:       s.CreatedAt,
			UpdatedAt:       s.UpdatedAt,
		})
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(exportData)
}

func (v *ExportView) View() string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render("Export Subscriptions") + "\n\n")

	if v.err != nil {
		b.WriteString(ErrorStyle.Render("Error: "+v.err.Error()) + "\n\n")
	}

	if v.exported {
		b.WriteString(SuccessStyle.Render(v.message) + "\n\n")
		b.WriteString(HelpStyle.Render("[q/esc] back"))
		return BoxStyle.Render(b.String())
	}

	// Format selector
	formatStr := "Format: "
	for i, f := range exportFormats {
		if i == v.formatIndex {
			formatStr += SelectedItemStyle.Render("[" + f + "]")
		} else {
			formatStr += " " + f + " "
		}
	}
	b.WriteString(formatStr + "\n\n")

	// Path input
	b.WriteString(v.pathInput.View() + "\n\n")

	b.WriteString(HelpStyle.Render("[tab] change format  [enter] export  [q/esc] cancel"))

	return BoxStyle.Render(b.String())
}
