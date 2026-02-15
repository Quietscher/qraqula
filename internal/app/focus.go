package app

type Panel int

const (
	PanelEndpoint Panel = iota
	PanelEditor
	PanelVariables
	PanelResults
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
	default:
		return "unknown"
	}
}

// nextPanel cycles focus forward.
func nextPanel(p Panel) Panel {
	return (p + 1) % panelCount
}

// navigatePanel moves focus directionally.
// Layout:
//
//	endpoint (top, full width)
//	editor (left top) | results (right)
//	variables (left bottom) | results (right)
func navigatePanel(current Panel, direction string) Panel {
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
		}
	case PanelVariables:
		switch direction {
		case "up":
			return PanelEditor
		case "right":
			return PanelResults
		}
	case PanelResults:
		switch direction {
		case "up":
			return PanelEndpoint
		case "left":
			return PanelEditor
		}
	}
	return current
}
