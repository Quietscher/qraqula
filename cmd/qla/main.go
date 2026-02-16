package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/qraqula/qla/internal/app"
)

func main() {
	m := app.NewModel()
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
