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
	envBadgeStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("235")).
		Background(lipgloss.Color("196"))
)

func (m Model) View() tea.View {
	v := tea.NewView(m.renderView())
	v.AltScreen = true
	return v
}

func (m Model) renderView() string {
	// Endpoint bar with optional env badge to the right
	var envBadge string
	badgeW := 0
	if m.endpoint.EnvName() != "" {
		envBadge = envBadgeStyle.Render(" " + m.endpoint.EnvName() + " ")
		badgeW = lipgloss.Width(envBadge) + 1
	}

	epStyle := endpointStyle.Width(m.width - 2 - badgeW)
	if m.focus == PanelEndpoint {
		epStyle = epStyle.BorderForeground(lipgloss.Color("62"))
	}
	ep := epStyle.Render(m.endpoint.View())
	if envBadge != "" {
		ep = lipgloss.JoinHorizontal(lipgloss.Center, ep, " ", envBadge)
	}

	// Right column content
	var rightContent string
	if m.rightPanelMode == modeSchema {
		rightContent = m.browser.View()
	} else {
		rightContent = m.results.View()
	}

	var content string
	if m.shouldShowSidebar() {
		// 3-column layout: sidebar | editor+vars | results
		sidebarCol := m.panelStyle(PanelHistory, m.sidebarW, m.contentH).Render(m.histSidebar.View())
		midCol := lipgloss.JoinVertical(lipgloss.Left,
			m.panelStyle(PanelEditor, m.midW, m.editorH).Render(m.editor.View()),
			m.panelStyle(PanelVariables, m.midW, m.varsH).Render(m.variables.View()),
		)
		rightCol := m.panelStyle(PanelResults, m.rightW, m.contentH).Render(rightContent)
		content = lipgloss.JoinHorizontal(lipgloss.Top, sidebarCol, midCol, rightCol)
	} else {
		// 2-column layout: editor+vars | results
		midCol := lipgloss.JoinVertical(lipgloss.Left,
			m.panelStyle(PanelEditor, m.leftW, m.editorH).Render(m.editor.View()),
			m.panelStyle(PanelVariables, m.leftW, m.varsH).Render(m.variables.View()),
		)
		rightCol := m.panelStyle(PanelResults, m.rightW, m.contentH).Render(rightContent)
		content = lipgloss.JoinHorizontal(lipgloss.Top, midCol, rightCol)
	}

	// Status bar
	status := statusStyle.Width(m.width).Render(m.statusbar.View())

	base := lipgloss.JoinVertical(lipgloss.Left, ep, content, status)
	if m.overlay.IsOpen() {
		return m.overlay.RenderOver(base)
	}
	return base
}

func (m Model) panelStyle(p Panel, width, height int) lipgloss.Style {
	if m.focus == p {
		return focusedBorder.Width(width).Height(height)
	}
	return blurredBorder.Width(width).Height(height)
}
