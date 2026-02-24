package app

import (
	"charm.land/bubbles/v2/key"
)

type keyMap struct {
	Execute       key.Binding
	Abort         key.Binding
	Quit          key.Binding
	Tab           key.Binding
	ShiftTab      key.Binding
	FocusUp       key.Binding
	FocusDown     key.Binding
	FocusLeft     key.Binding
	FocusRight    key.Binding
	ToggleDocs    key.Binding
	RefreshSchema key.Binding
	ToggleSidebar key.Binding
	ToggleOverlay key.Binding
	CycleEnv      key.Binding
	OpenEditor    key.Binding
	ToggleSearch  key.Binding
	Prettify      key.Binding
}

var keys = keyMap{
	Execute: key.NewBinding(
		key.WithKeys("alt+enter"),
		key.WithHelp("alt+↵", "execute query"),
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
	ShiftTab: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "previous panel"),
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
	ToggleDocs: key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("ctrl+d", "toggle schema/results"),
	),
	RefreshSchema: key.NewBinding(
		key.WithKeys("ctrl+r"),
		key.WithHelp("ctrl+r", "refresh schema"),
	),
	ToggleSidebar: key.NewBinding(
		key.WithKeys("ctrl+b"),
		key.WithHelp("ctrl+b", "toggle sidebar"),
	),
	ToggleOverlay: key.NewBinding(
		key.WithKeys("ctrl+e"),
		key.WithHelp("ctrl+e", "environments/headers"),
	),
	CycleEnv: key.NewBinding(
		key.WithKeys("ctrl+n"),
		key.WithHelp("ctrl+n", "cycle environment"),
	),
	OpenEditor: key.NewBinding(
		key.WithKeys("ctrl+o"),
		key.WithHelp("ctrl+o", "open in $EDITOR"),
	),
	ToggleSearch: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search results"),
	),
	Prettify: key.NewBinding(
		key.WithKeys("ctrl+p"),
		key.WithHelp("ctrl+p", "prettify"),
	),
}
