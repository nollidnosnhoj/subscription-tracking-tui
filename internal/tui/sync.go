package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"subscription-tracker/internal/app"
	"subscription-tracker/internal/service"
)

type SyncView struct {
	passwordInput textinput.Model
	tokenInput    textinput.Model
	gistIDInput   textinput.Model
	focusIndex    int
	message       string
	err           error
	loading       bool
	gistConfig    *service.GistConfig
}

const (
	syncFocusPassword = iota
	syncFocusToken
	syncFocusGistID
)

func NewSyncView() *SyncView {
	passwordInput := textinput.New()
	passwordInput.Placeholder = "Enter encryption password"
	passwordInput.Focus()
	passwordInput.EchoMode = textinput.EchoPassword
	passwordInput.EchoCharacter = '•'
	passwordInput.CharLimit = 100
	passwordInput.Width = 40
	passwordInput.Prompt = "Password: "

	tokenInput := textinput.New()
	tokenInput.Placeholder = "ghp_xxxxxxxxxxxx"
	tokenInput.EchoMode = textinput.EchoPassword
	tokenInput.EchoCharacter = '•'
	tokenInput.CharLimit = 100
	tokenInput.Width = 40
	tokenInput.Prompt = "GitHub Token: "

	gistIDInput := textinput.New()
	gistIDInput.Placeholder = "Leave empty for new gist"
	gistIDInput.CharLimit = 50
	gistIDInput.Width = 40
	gistIDInput.Prompt = "Gist ID: "

	return &SyncView{
		passwordInput: passwordInput,
		tokenInput:    tokenInput,
		gistIDInput:   gistIDInput,
		focusIndex:    syncFocusPassword,
	}
}

func (v *SyncView) Init(a *app.App) tea.Cmd {
	return v.loadConfig(a)
}

func (v *SyncView) loadConfig(a *app.App) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		config, err := a.SyncService.GetGistConfig(ctx)
		if err != nil {
			return syncErrMsg{err}
		}
		return syncConfigLoadedMsg{config}
	}
}

type syncConfigLoadedMsg struct {
	config *service.GistConfig
}

type syncErrMsg struct {
	err error
}

type syncSuccessMsg struct {
	message string
}

type syncPushCompleteMsg struct {
	gistID string
}

func (v *SyncView) Update(msg tea.Msg, a *app.App) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if v.loading {
			return false, nil // Don't accept input while loading
		}
		switch msg.String() {
		case "tab", "down":
			v.focusIndex = (v.focusIndex + 1) % 3
			return false, v.updateFocus()
		case "shift+tab", "up":
			v.focusIndex = (v.focusIndex + 2) % 3
			return false, v.updateFocus()
		case "ctrl+p":
			// Push to gist
			if v.passwordInput.Value() == "" {
				v.err = fmt.Errorf("password is required")
				return false, nil
			}
			if v.tokenInput.Value() == "" {
				v.err = fmt.Errorf("GitHub token is required")
				return false, nil
			}
			v.loading = true
			v.err = nil
			v.message = ""
			return false, v.pushToGist(a)
		case "ctrl+l":
			// Pull from gist
			if v.passwordInput.Value() == "" {
				v.err = fmt.Errorf("password is required")
				return false, nil
			}
			if v.tokenInput.Value() == "" {
				v.err = fmt.Errorf("GitHub token is required")
				return false, nil
			}
			if v.gistIDInput.Value() == "" {
				v.err = fmt.Errorf("Gist ID is required for pull")
				return false, nil
			}
			v.loading = true
			v.err = nil
			v.message = ""
			return false, v.pullFromGist(a)
		case "q", "esc":
			return true, nil
		}
	case syncConfigLoadedMsg:
		v.gistConfig = msg.config
		if msg.config.Token != "" {
			v.tokenInput.SetValue(msg.config.Token)
		}
		if msg.config.GistID != "" {
			v.gistIDInput.SetValue(msg.config.GistID)
		}
		return false, nil
	case syncPushCompleteMsg:
		v.loading = false
		v.message = fmt.Sprintf("Pushed to gist: %s", msg.gistID)
		v.gistIDInput.SetValue(msg.gistID)
		return false, nil
	case syncSuccessMsg:
		v.loading = false
		v.message = msg.message
		return false, nil
	case syncErrMsg:
		v.loading = false
		v.err = msg.err
		return false, nil
	}

	var cmd tea.Cmd
	switch v.focusIndex {
	case syncFocusPassword:
		v.passwordInput, cmd = v.passwordInput.Update(msg)
	case syncFocusToken:
		v.tokenInput, cmd = v.tokenInput.Update(msg)
	case syncFocusGistID:
		v.gistIDInput, cmd = v.gistIDInput.Update(msg)
	}
	return false, cmd
}

func (v *SyncView) updateFocus() tea.Cmd {
	v.passwordInput.Blur()
	v.tokenInput.Blur()
	v.gistIDInput.Blur()

	switch v.focusIndex {
	case syncFocusPassword:
		return v.passwordInput.Focus()
	case syncFocusToken:
		return v.tokenInput.Focus()
	case syncFocusGistID:
		return v.gistIDInput.Focus()
	}
	return nil
}

func (v *SyncView) pushToGist(a *app.App) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		config := service.GistConfig{
			Token:  v.tokenInput.Value(),
			GistID: v.gistIDInput.Value(),
		}

		gistID, err := a.SyncService.PushToGist(ctx, v.passwordInput.Value(), config)
		if err != nil {
			return syncErrMsg{err}
		}

		// Save the config
		config.GistID = gistID
		if err := a.SyncService.SaveGistConfig(ctx, &config); err != nil {
			return syncErrMsg{fmt.Errorf("pushed but failed to save config: %w", err)}
		}

		return syncPushCompleteMsg{gistID}
	}
}

func (v *SyncView) pullFromGist(a *app.App) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		config := service.GistConfig{
			Token:  v.tokenInput.Value(),
			GistID: v.gistIDInput.Value(),
		}

		if err := a.SyncService.PullFromGist(ctx, v.passwordInput.Value(), config); err != nil {
			return syncErrMsg{err}
		}

		// Save the config
		if err := a.SyncService.SaveGistConfig(ctx, &config); err != nil {
			return syncErrMsg{fmt.Errorf("pulled but failed to save config: %w", err)}
		}

		return syncSuccessMsg{"Data pulled and imported successfully!"}
	}
}

func (v *SyncView) View() string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render("Sync to GitHub Gist") + "\n\n")

	if v.loading {
		b.WriteString("Syncing...\n\n")
		return BoxStyle.Render(b.String())
	}

	if v.err != nil {
		b.WriteString(ErrorStyle.Render("Error: "+v.err.Error()) + "\n\n")
	}

	if v.message != "" {
		b.WriteString(SuccessStyle.Render(v.message) + "\n\n")
	}

	b.WriteString("Your data is encrypted locally before being uploaded.\n")
	b.WriteString("Use the same password on both machines.\n\n")

	// Password input
	if v.focusIndex == syncFocusPassword {
		b.WriteString(FocusedInputStyle.Render(v.passwordInput.View()) + "\n")
	} else {
		b.WriteString(BlurredInputStyle.Render(v.passwordInput.View()) + "\n")
	}

	b.WriteString("\n" + SubtitleStyle.Render("GitHub Settings") + "\n")
	b.WriteString(HelpStyle.Render("Create a token at: https://github.com/settings/tokens") + "\n")
	b.WriteString(HelpStyle.Render("Required scope: 'gist'") + "\n\n")

	// Token input
	if v.focusIndex == syncFocusToken {
		b.WriteString(FocusedInputStyle.Render(v.tokenInput.View()) + "\n")
	} else {
		b.WriteString(BlurredInputStyle.Render(v.tokenInput.View()) + "\n")
	}

	// Gist ID input
	if v.focusIndex == syncFocusGistID {
		b.WriteString(FocusedInputStyle.Render(v.gistIDInput.View()) + "\n")
	} else {
		b.WriteString(BlurredInputStyle.Render(v.gistIDInput.View()) + "\n")
	}

	b.WriteString("\n" + HelpStyle.Render("[tab] next field  [ctrl+p] push  [ctrl+l] pull  [q/esc] back"))

	return BoxStyle.Render(b.String())
}
