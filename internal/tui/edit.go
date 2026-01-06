package tui

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"subscription-tracker/internal/db"
)

type EditForm struct {
	inputs     []textinput.Model
	focusIndex int
	cycleIndex int
	subID      int64
	err        error
}

const (
	editInputName = iota
	editInputAmount
	editInputCurrency
	editInputRenewal
)

func NewEditForm() *EditForm {
	inputs := make([]textinput.Model, 4)

	inputs[editInputName] = textinput.New()
	inputs[editInputName].CharLimit = 50
	inputs[editInputName].Width = 30
	inputs[editInputName].Prompt = "Name: "

	inputs[editInputAmount] = textinput.New()
	inputs[editInputAmount].CharLimit = 10
	inputs[editInputAmount].Width = 15
	inputs[editInputAmount].Prompt = "Amount: "

	inputs[editInputCurrency] = textinput.New()
	inputs[editInputCurrency].CharLimit = 3
	inputs[editInputCurrency].Width = 5
	inputs[editInputCurrency].Prompt = "Currency: "

	inputs[editInputRenewal] = textinput.New()
	inputs[editInputRenewal].CharLimit = 10
	inputs[editInputRenewal].Width = 12
	inputs[editInputRenewal].Prompt = "Renewal Date (YYYY-MM-DD): "

	return &EditForm{
		inputs:     inputs,
		focusIndex: 0,
		cycleIndex: 0,
	}
}

func (f *EditForm) LoadSubscription(sub db.Subscription) {
	f.subID = sub.ID
	f.inputs[editInputName].SetValue(sub.Name)
	f.inputs[editInputAmount].SetValue(fmt.Sprintf("%.2f", sub.Amount))
	f.inputs[editInputCurrency].SetValue(sub.Currency)
	if sub.NextRenewalDate.Valid {
		f.inputs[editInputRenewal].SetValue(sub.NextRenewalDate.String)
	}
	if sub.BillingCycle == "yearly" {
		f.cycleIndex = 1
	} else {
		f.cycleIndex = 0
	}
	f.inputs[editInputName].Focus()
}

func (f *EditForm) Init() tea.Cmd {
	return textinput.Blink
}

const editFocusCycle = 100 // special index for cycle selector

// nextFocus returns the next focus index in the form
func (f *EditForm) nextFocus(current int) int {
	// Order: Name(0) -> Amount(1) -> Currency(2) -> Cycle(100) -> Renewal(3) -> Name(0)
	switch current {
	case editInputName:
		return editInputAmount
	case editInputAmount:
		return editInputCurrency
	case editInputCurrency:
		return editFocusCycle
	case editFocusCycle:
		return editInputRenewal
	case editInputRenewal:
		return editInputName
	default:
		return editInputName
	}
}

// prevFocus returns the previous focus index in the form
func (f *EditForm) prevFocus(current int) int {
	// Reverse order
	switch current {
	case editInputName:
		return editInputRenewal
	case editInputAmount:
		return editInputName
	case editInputCurrency:
		return editInputAmount
	case editFocusCycle:
		return editInputCurrency
	case editInputRenewal:
		return editFocusCycle
	default:
		return editInputName
	}
}

func (f *EditForm) Update(msg tea.Msg, app interface{}) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			f.focusIndex = f.nextFocus(f.focusIndex)
			return false, f.updateFocus()
		case "shift+tab", "up":
			f.focusIndex = f.prevFocus(f.focusIndex)
			return false, f.updateFocus()
		case "left", "right":
			if f.focusIndex == editFocusCycle {
				f.cycleIndex = 1 - f.cycleIndex
			}
			return false, nil
		case "enter":
			f.focusIndex = f.nextFocus(f.focusIndex)
			return false, f.updateFocus()
		case "ctrl+s":
			return true, f.submit()
		}
	}

	if f.focusIndex < len(f.inputs) && f.focusIndex != editFocusCycle {
		var cmd tea.Cmd
		f.inputs[f.focusIndex], cmd = f.inputs[f.focusIndex].Update(msg)
		return false, cmd
	}
	return false, nil
}

func (f *EditForm) updateFocus() tea.Cmd {
	cmds := make([]tea.Cmd, len(f.inputs))
	for i := range f.inputs {
		if i == f.focusIndex && f.focusIndex != editFocusCycle {
			cmds[i] = f.inputs[i].Focus()
		} else {
			f.inputs[i].Blur()
		}
	}
	return tea.Batch(cmds...)
}

func (f *EditForm) submit() tea.Cmd {
	return func() tea.Msg {
		amount, err := strconv.ParseFloat(f.inputs[editInputAmount].Value(), 64)
		if err != nil {
			return errMsg{fmt.Errorf("invalid amount: %w", err)}
		}

		dateStr := f.inputs[editInputRenewal].Value()
		if dateStr == "" {
			return errMsg{fmt.Errorf("renewal date is required")}
		}
		_, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			return errMsg{fmt.Errorf("invalid date format (use YYYY-MM-DD): %w", err)}
		}

		params := db.UpdateSubscriptionParams{
			ID:              f.subID,
			Name:            f.inputs[editInputName].Value(),
			Amount:          amount,
			Currency:        strings.ToUpper(f.inputs[editInputCurrency].Value()),
			BillingCycle:    cycles[f.cycleIndex],
			NextRenewalDate: sql.NullString{String: dateStr, Valid: true},
		}

		return updateSubscriptionMsg{params}
	}
}

type updateSubscriptionMsg struct {
	params db.UpdateSubscriptionParams
}

func (f *EditForm) View() string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render("Edit Subscription") + "\n\n")

	if f.err != nil {
		b.WriteString(ErrorStyle.Render("Error: "+f.err.Error()) + "\n\n")
	}

	// Name, Amount, Currency
	for i := 0; i < 3; i++ {
		if i == f.focusIndex {
			b.WriteString(FocusedInputStyle.Render(f.inputs[i].View()) + "\n")
		} else {
			b.WriteString(BlurredInputStyle.Render(f.inputs[i].View()) + "\n")
		}
	}

	// Cycle selector
	cycleStr := "Billing Cycle: "
	for i, c := range cycles {
		if i == f.cycleIndex {
			cycleStr += SelectedItemStyle.Render("[" + c + "]")
		} else {
			cycleStr += " " + c + " "
		}
	}
	if f.focusIndex == editFocusCycle {
		b.WriteString(FocusedInputStyle.Render(cycleStr) + "\n")
	} else {
		b.WriteString(cycleStr + "\n")
	}

	// Renewal date (always shown)
	if f.focusIndex == editInputRenewal {
		b.WriteString(FocusedInputStyle.Render(f.inputs[editInputRenewal].View()) + "\n")
	} else {
		b.WriteString(BlurredInputStyle.Render(f.inputs[editInputRenewal].View()) + "\n")
	}

	b.WriteString("\n" + HelpStyle.Render("[tab] next  [shift+tab] prev  [←/→] cycle  [ctrl+s] save  [q/esc] cancel"))

	return BoxStyle.Render(b.String())
}
