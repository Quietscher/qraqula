package app

import "github.com/qraqula/qla/internal/statusbar"

var editorHints = []statusbar.Hint{
	{Key: "↵", Label: "edit"},
	{Key: "alt+↵", Label: "execute"},
	{Key: "^p", Label: "prettify"},
	{Key: "^y", Label: "copy"},
	{Key: "tab", Label: "next"},
	{Key: "^o", Label: "$EDITOR"},
	{Key: "^e", Label: "env"},
	{Key: "^d", Label: "docs"},
	{Key: "^b", Label: "sidebar"},
	{Key: "^q", Label: "quit"},
}

var variablesHints = []statusbar.Hint{
	{Key: "↵", Label: "edit"},
	{Key: "alt+↵", Label: "execute"},
	{Key: "^p", Label: "prettify"},
	{Key: "^y", Label: "copy"},
	{Key: "tab", Label: "next"},
	{Key: "^o", Label: "$EDITOR"},
	{Key: "^d", Label: "docs"},
	{Key: "^b", Label: "sidebar"},
	{Key: "^q", Label: "quit"},
}

var editingHints = []statusbar.Hint{
	{Key: "esc", Label: "done"},
	{Key: "tab", Label: "indent"},
	{Key: "alt+↵", Label: "execute"},
	{Key: "^p", Label: "prettify"},
	{Key: "^o", Label: "$EDITOR"},
	{Key: "^q", Label: "quit"},
}

var resultsHints = []statusbar.Hint{
	{Key: "↵", Label: "execute"},
	{Key: "tab", Label: "next"},
	{Key: "/", Label: "search"},
	{Key: "^y", Label: "copy"},
	{Key: "^d", Label: "docs"},
	{Key: "j/k", Label: "scroll"},
	{Key: "^c", Label: "abort"},
	{Key: "^q", Label: "quit"},
}

var schemaBrowserHints = []statusbar.Hint{
	{Key: "j/k/↑↓", Label: "navigate"},
	{Key: "l/↵", Label: "drill in"},
	{Key: "h/⌫", Label: "back"},
	{Key: "/", Label: "filter"},
	{Key: "esc", Label: "clear"},
	{Key: "^d", Label: "results"},
	{Key: "^q", Label: "quit"},
}

var endpointHints = []statusbar.Hint{
	{Key: "tab", Label: "next"},
	{Key: "^y", Label: "copy"},
	{Key: "^e", Label: "env"},
	{Key: "^n", Label: "cycle env"},
	{Key: "^r", Label: "schema"},
	{Key: "^q", Label: "quit"},
}

var historyHints = []statusbar.Hint{
	{Key: "j/k", Label: "navigate"},
	{Key: "↵/l", Label: "select"},
	{Key: "N", Label: "new folder"},
	{Key: "r", Label: "rename"},
	{Key: "d", Label: "delete"},
	{Key: "m/M", Label: "move"},
	{Key: "/", Label: "filter"},
	{Key: "^b", Label: "close"},
	{Key: "^q", Label: "quit"},
}

// hintsForFocus returns the appropriate hint set for the given panel and right panel mode.
func hintsForFocus(panel Panel, rpMode rightPanelMode) []statusbar.Hint {
	switch panel {
	case PanelEditor:
		return editorHints
	case PanelVariables:
		return variablesHints
	case PanelResults:
		if rpMode == modeSchema {
			return schemaBrowserHints
		}
		return resultsHints
	case PanelEndpoint:
		return endpointHints
	case PanelHistory:
		return historyHints
	default:
		return editorHints
	}
}
