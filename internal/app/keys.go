package app

import (
	"github.com/charmbracelet/bubbles/key"
)

type keyMap struct {
	Execute    key.Binding
	Abort      key.Binding
	Quit       key.Binding
	Tab        key.Binding
	FocusUp    key.Binding
	FocusDown  key.Binding
	FocusLeft  key.Binding
	FocusRight key.Binding
}

var keys = keyMap{
	Execute: key.NewBinding(
		key.WithKeys("ctrl+enter"),
		key.WithHelp("ctrl+enter", "execute query"),
	),
	Abort: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "abort query / quit"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+q"),
		key.WithHelp("ctrl+q", "quit"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next panel"),
	),
	FocusUp: key.NewBinding(
		key.WithKeys("ctrl+k"),
		key.WithHelp("ctrl+k", "focus up"),
	),
	FocusDown: key.NewBinding(
		key.WithKeys("ctrl+j"),
		key.WithHelp("ctrl+j", "focus down"),
	),
	FocusLeft: key.NewBinding(
		key.WithKeys("ctrl+h"),
		key.WithHelp("ctrl+h", "focus left"),
	),
	FocusRight: key.NewBinding(
		key.WithKeys("ctrl+l"),
		key.WithHelp("ctrl+l", "focus right"),
	),
}
