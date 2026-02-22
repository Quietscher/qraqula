package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/qraqula/qla/internal/app"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println("qla " + version)
		return
	}

	m := app.NewModel()
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
