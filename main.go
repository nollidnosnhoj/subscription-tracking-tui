package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"subscription-tracker/internal/app"
	"subscription-tracker/internal/tui"
)

func main() {
	application, err := app.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing app: %v\n", err)
		os.Exit(1)
	}
	defer application.Close()

	model := tui.New(application)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running app: %v\n", err)
		os.Exit(1)
	}
}
