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

type AddForm struct {
	inputs     []textinput.Model
	focusIndex int
	cycleIndex int // 0 = monthly, 1 = yearly
	err        error
}

const (
	addInputName = iota
	addInputAmount
	addInputCurrency
	addInputRenewal
)

var cycles = []string{"monthly", "yearly"}

func NewAddForm() *AddForm {
	inputs := make([]textinput.Model, 4)

	inputs[addInputName] = textinput.New()
	inputs[addInputName].Placeholder = "Netflix"
	inputs[addInputName].Focus()
	inputs[addInputName].CharLimit = 50
	inputs[addInputName].Width = 30
	inputs[addInputName].Prompt = "Name: "

	inputs[addInputAmount] = textinput.New()
	inputs[addInputAmount].Placeholder = "9.99"
	inputs[addInputAmount].CharLimit = 10
	inputs[addInputAmount].Width = 15
	inputs[addInputAmount].Prompt = "Amount: "

	inputs[addInputCurrency] = textinput.New()
	inputs[addInputCurrency].Placeholder = "USD"
	inputs[addInputCurrency].CharLimit = 3
	inputs[addInputCurrency].Width = 5
	inputs[addInputCurrency].Prompt = "Currency: "
	inputs[addInputCurrency].SetValue("USD")

	inputs[addInputRenewal] = textinput.New()
	inputs[addInputRenewal].Placeholder = time.Now().Format("2006-01-02")
	inputs[addInputRenewal].CharLimit = 10
	inputs[addInputRenewal].Width = 12
	inputs[addInputRenewal].Prompt = "Renewal Date (YYYY-MM-DD): "

	return &AddForm{
		inputs:     inputs,
		focusIndex: 0,
		cycleIndex: 0,
	}
}

func (f *AddForm) Init() tea.Cmd {
	return textinput.Blink
}

// nextFocus returns the next focus index in the form
func (f *AddForm) nextFocus(current int) int {
	// Order: Name(0) -> Amount(1) -> Currency(2) -> Cycle(100) -> Renewal(3) -> Name(0)
	switch current {
	case addInputName:
		return addInputAmount
	case addInputAmount:
		return addInputCurrency
	case addInputCurrency:
		return focusCycle
	case focusCycle:
		return addInputRenewal
	case addInputRenewal:
		return addInputName
	default:
		return addInputName
	}
}

// prevFocus returns the previous focus index in the form
func (f *AddForm) prevFocus(current int) int {
	// Reverse order
	switch current {
	case addInputName:
		return addInputRenewal
	case addInputAmount:
		return addInputName
	case addInputCurrency:
		return addInputAmount
	case focusCycle:
		return addInputCurrency
	case addInputRenewal:
		return focusCycle
	default:
		return addInputName
	}
}

const focusCycle = 100 // special index for cycle selector

func (f *AddForm) Update(msg tea.Msg, app interface{}) (bool, tea.Cmd) {
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
			if f.focusIndex == focusCycle {
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

	if f.focusIndex < len(f.inputs) {
		var cmd tea.Cmd
		f.inputs[f.focusIndex], cmd = f.inputs[f.focusIndex].Update(msg)
		return false, cmd
	}
	return false, nil
}

func (f *AddForm) updateFocus() tea.Cmd {
	cmds := make([]tea.Cmd, len(f.inputs))
	for i := range f.inputs {
		if i == f.focusIndex && f.focusIndex != focusCycle {
			cmds[i] = f.inputs[i].Focus()
		} else {
			f.inputs[i].Blur()
		}
	}
	return tea.Batch(cmds...)
}

func (f *AddForm) submit() tea.Cmd {
	return func() tea.Msg {
		name := f.inputs[addInputName].Value()
		if name == "" {
			return errMsg{fmt.Errorf("name is required")}
		}

		amount, err := strconv.ParseFloat(f.inputs[addInputAmount].Value(), 64)
		if err != nil {
			return errMsg{fmt.Errorf("invalid amount: %w", err)}
		}

		dateStr := f.inputs[addInputRenewal].Value()
		if dateStr == "" {
			return errMsg{fmt.Errorf("renewal date is required")}
		}
		_, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			return errMsg{fmt.Errorf("invalid date format (use YYYY-MM-DD): %w", err)}
		}

		params := db.CreateSubscriptionParams{
			Name:            name,
			Amount:          amount,
			Currency:        strings.ToUpper(f.inputs[addInputCurrency].Value()),
			BillingCycle:    cycles[f.cycleIndex],
			NextRenewalDate: sql.NullString{String: dateStr, Valid: true},
		}

		return createSubscriptionMsg{params}
	}
}

func (f *AddForm) View() string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render("Add Subscription") + "\n\n")

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
	if f.focusIndex == focusCycle {
		b.WriteString(FocusedInputStyle.Render(cycleStr) + "\n")
	} else {
		b.WriteString(cycleStr + "\n")
	}

	// Renewal date (always shown)
	if f.focusIndex == addInputRenewal {
		b.WriteString(FocusedInputStyle.Render(f.inputs[addInputRenewal].View()) + "\n")
	} else {
		b.WriteString(BlurredInputStyle.Render(f.inputs[addInputRenewal].View()) + "\n")
	}

	b.WriteString("\n" + HelpStyle.Render("[tab] next  [shift+tab] prev  [←/→] cycle  [ctrl+s] save  [q/esc] cancel"))

	return BoxStyle.Render(b.String())
}
