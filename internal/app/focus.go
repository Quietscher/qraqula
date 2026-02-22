package app

type Panel int

const (
	PanelEndpoint Panel = iota
	PanelEditor
	PanelVariables
	PanelResults
	PanelHistory
	panelCount
)

func (p Panel) String() string {
	switch p {
	case PanelEndpoint:
		return "endpoint"
	case PanelEditor:
		return "editor"
	case PanelVariables:
		return "variables"
	case PanelResults:
		return "results"
	case PanelHistory:
		return "history"
	default:
		return "unknown"
	}
}

// nextPanel cycles focus forward. Skips PanelHistory when sidebar is closed.
func nextPanel(p Panel, sidebarOpen bool) Panel {
	for {
		p = (p + 1) % panelCount
		if p == PanelHistory && !sidebarOpen {
			continue
		}
		return p
	}
}

// prevPanel cycles focus backward. Skips PanelHistory when sidebar is closed.
func prevPanel(p Panel, sidebarOpen bool) Panel {
	for {
		p = (p - 1 + panelCount) % panelCount
		if p == PanelHistory && !sidebarOpen {
			continue
		}
		return p
	}
}

// navigatePanel moves focus directionally.
//
// Layout (sidebar open):
//
//	endpoint (top, full width)
//	history (left) | editor (center top) | results (right)
//	history (left) | variables (center bottom) | results (right)
//
// Layout (sidebar closed): same as before (no history column)
func navigatePanel(current Panel, direction string, sidebarOpen bool) Panel {
	switch current {
	case PanelEndpoint:
		switch direction {
		case "down":
			return PanelEditor
		}
	case PanelEditor:
		switch direction {
		case "up":
			return PanelEndpoint
		case "down":
			return PanelVariables
		case "right":
			return PanelResults
		case "left":
			if sidebarOpen {
				return PanelHistory
			}
		}
	case PanelVariables:
		switch direction {
		case "up":
			return PanelEditor
		case "right":
			return PanelResults
		case "left":
			if sidebarOpen {
				return PanelHistory
			}
		}
	case PanelResults:
		switch direction {
		case "up":
			return PanelEndpoint
		case "left":
			return PanelEditor
		}
	case PanelHistory:
		switch direction {
		case "right":
			return PanelEditor
		case "up":
			return PanelEndpoint
		}
	}
	return current
}
