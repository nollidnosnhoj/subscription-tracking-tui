package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"subscription-tracker/internal/app"
)

type ConfigView struct {
	cutoffInput   textinput.Model
	salaryInput   textinput.Model
	focusIndex    int
	currentDay    int
	currentSalary float64
	message       string
	err           error
	saved         bool
}

const (
	configFocusCutoff = iota
	configFocusSalary
)

func NewConfigView() *ConfigView {
	cutoffInput := textinput.New()
	cutoffInput.Placeholder = "1"
	cutoffInput.Focus()
	cutoffInput.CharLimit = 2
	cutoffInput.Width = 5
	cutoffInput.Prompt = "Payday (1-28): "

	salaryInput := textinput.New()
	salaryInput.Placeholder = "0.00"
	salaryInput.CharLimit = 12
	salaryInput.Width = 15
	salaryInput.Prompt = "Monthly Salary: "

	return &ConfigView{
		cutoffInput: cutoffInput,
		salaryInput: salaryInput,
		focusIndex:  configFocusCutoff,
	}
}

func (v *ConfigView) Init(a *app.App) tea.Cmd {
	return v.loadConfig(a)
}

func (v *ConfigView) loadConfig(a *app.App) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		day, err := a.ConfigService.GetMonthCutoffDay(ctx)
		if err != nil {
			return configErrMsg{err}
		}
		salary, err := a.ConfigService.GetMonthlySalary(ctx)
		if err != nil {
			return configErrMsg{err}
		}
		return configLoadedMsg{cutoffDay: day, salary: salary}
	}
}

type configLoadedMsg struct {
	cutoffDay int
	salary    float64
}

type configErrMsg struct {
	err error
}

type configSavedMsg struct {
	message string
}

func (v *ConfigView) Update(msg tea.Msg, a *app.App) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			v.focusIndex = (v.focusIndex + 1) % 2
			return false, v.updateFocus()
		case "shift+tab", "up":
			v.focusIndex = (v.focusIndex + 1) % 2
			return false, v.updateFocus()
		case "ctrl+s":
			return false, v.save(a)
		case "q", "esc":
			return true, nil
		}
	case configLoadedMsg:
		v.currentDay = msg.cutoffDay
		v.currentSalary = msg.salary
		v.cutoffInput.SetValue(strconv.Itoa(msg.cutoffDay))
		if msg.salary > 0 {
			v.salaryInput.SetValue(strconv.FormatFloat(msg.salary, 'f', 2, 64))
		}
		return false, nil
	case configSavedMsg:
		v.message = msg.message
		v.saved = true
		return false, nil
	case configErrMsg:
		v.err = msg.err
		return false, nil
	}

	var cmd tea.Cmd
	switch v.focusIndex {
	case configFocusCutoff:
		v.cutoffInput, cmd = v.cutoffInput.Update(msg)
	case configFocusSalary:
		v.salaryInput, cmd = v.salaryInput.Update(msg)
	}
	return false, cmd
}

func (v *ConfigView) updateFocus() tea.Cmd {
	if v.focusIndex == configFocusCutoff {
		v.salaryInput.Blur()
		return v.cutoffInput.Focus()
	}
	v.cutoffInput.Blur()
	return v.salaryInput.Focus()
}

func (v *ConfigView) save(a *app.App) tea.Cmd {
	return func() tea.Msg {
		day, err := strconv.Atoi(v.cutoffInput.Value())
		if err != nil {
			return configErrMsg{fmt.Errorf("invalid cutoff day")}
		}

		salary := 0.0
		if v.salaryInput.Value() != "" {
			salary, err = strconv.ParseFloat(v.salaryInput.Value(), 64)
			if err != nil {
				return configErrMsg{fmt.Errorf("invalid salary amount")}
			}
		}

		ctx := context.Background()
		if err := a.ConfigService.SetMonthCutoffDay(ctx, day); err != nil {
			return configErrMsg{err}
		}
		if err := a.ConfigService.SetMonthlySalary(ctx, salary); err != nil {
			return configErrMsg{err}
		}

		return configSavedMsg{"Settings saved!"}
	}
}

func (v *ConfigView) View() string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render("Configuration") + "\n\n")

	if v.err != nil {
		b.WriteString(ErrorStyle.Render("Error: "+v.err.Error()) + "\n\n")
	}

	if v.saved {
		b.WriteString(SuccessStyle.Render(v.message) + "\n\n")
	}

	b.WriteString("Configure your pay stub settings.\n")
	b.WriteString("The payday determines when your billing period starts.\n")
	b.WriteString("The salary is used to calculate remaining money after subscriptions.\n\n")

	// Cutoff day input
	if v.focusIndex == configFocusCutoff {
		b.WriteString(FocusedInputStyle.Render(v.cutoffInput.View()) + "\n")
	} else {
		b.WriteString(BlurredInputStyle.Render(v.cutoffInput.View()) + "\n")
	}

	// Salary input
	if v.focusIndex == configFocusSalary {
		b.WriteString(FocusedInputStyle.Render(v.salaryInput.View()) + "\n")
	} else {
		b.WriteString(BlurredInputStyle.Render(v.salaryInput.View()) + "\n")
	}

	b.WriteString("\n" + HelpStyle.Render("[tab] next field  [ctrl+s] save  [q/esc] back"))

	return BoxStyle.Render(b.String())
}
