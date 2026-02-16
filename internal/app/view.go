package app

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	focusedBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))
	blurredBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240"))
	endpointStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240"))
	statusStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
)

func (m Model) View() tea.View {
	v := tea.NewView(m.renderView())
	v.AltScreen = true
	return v
}

func (m Model) renderView() string {
	// Endpoint bar
	epStyle := endpointStyle.Width(m.width - 2)
	if m.focus == PanelEndpoint {
		epStyle = epStyle.BorderForeground(lipgloss.Color("62"))
	}
	ep := epStyle.Render(m.endpoint.View())

	// Left column
	editorStyle := m.panelStyle(PanelEditor)
	varsStyle := m.panelStyle(PanelVariables)

	leftCol := lipgloss.JoinVertical(lipgloss.Left,
		editorStyle.Render(m.editor.View()),
		varsStyle.Render(m.variables.View()),
	)

	// Right column
	resultsStyle := m.panelStyle(PanelResults)
	var rightContent string
	if m.rightPanelMode == modeSchema {
		rightContent = m.browser.View()
	} else {
		rightContent = m.results.View()
	}
	rightCol := resultsStyle.Render(rightContent)

	// Main content
	content := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, rightCol)

	// Status bar
	status := statusStyle.Width(m.width).Render(m.statusbar.View())

	return lipgloss.JoinVertical(lipgloss.Left, ep, content, status)
}

func (m Model) panelStyle(p Panel) lipgloss.Style {
	if m.focus == p {
		return focusedBorder
	}
	return blurredBorder
}
