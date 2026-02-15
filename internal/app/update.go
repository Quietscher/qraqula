package app

import (
	"context"
	"encoding/json"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/qraqula/qla/internal/graphql"
)

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layoutPanels()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case QueryResultMsg:
		m.querying = false
		m.cancelQuery = nil
		r := msg.Result
		hasErrors := r.Response.HasErrors()

		// Build display content
		raw, _ := json.MarshalIndent(r.Response, "", "  ")
		m.results.SetContent(string(raw))
		m.statusbar.SetResult(r.StatusCode, r.Duration, r.Size, hasErrors)
		return m, nil

	case QueryErrorMsg:
		m.querying = false
		m.cancelQuery = nil
		m.results.SetContent("Error: " + msg.Err.Error())
		m.statusbar.SetError(msg.Err.Error())
		return m, nil

	case QueryAbortedMsg:
		m.querying = false
		m.cancelQuery = nil
		m.results.SetContent("Query aborted")
		m.statusbar.SetAborted()
		return m, nil
	}

	// Forward to focused panel
	cmds = append(cmds, m.updateFocused(msg)...)
	return m, tea.Batch(cmds...)
}

func (m *Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Quit):
		return *m, tea.Quit

	case key.Matches(msg, keys.Abort):
		if m.querying && m.cancelQuery != nil {
			m.cancelQuery()
			return *m, nil
		}
		return *m, tea.Quit

	case key.Matches(msg, keys.Execute):
		return m.executeQuery()

	case key.Matches(msg, keys.Tab):
		m.setFocus(nextPanel(m.focus))
		return *m, nil

	case key.Matches(msg, keys.FocusUp):
		m.setFocus(navigatePanel(m.focus, "up"))
		return *m, nil
	case key.Matches(msg, keys.FocusDown):
		m.setFocus(navigatePanel(m.focus, "down"))
		return *m, nil
	case key.Matches(msg, keys.FocusLeft):
		m.setFocus(navigatePanel(m.focus, "left"))
		return *m, nil
	case key.Matches(msg, keys.FocusRight):
		m.setFocus(navigatePanel(m.focus, "right"))
		return *m, nil
	}

	// Forward to focused panel
	var cmds []tea.Cmd
	cmds = append(cmds, m.updateFocused(tea.Msg(msg))...)
	return *m, tea.Batch(cmds...)
}

func (m *Model) executeQuery() (Model, tea.Cmd) {
	if m.querying {
		return *m, nil
	}
	ep := m.endpoint.Value()
	if ep == "" {
		m.statusbar.SetError("No endpoint configured")
		return *m, nil
	}

	vars, err := m.variables.ParsedVariables()
	if err != nil {
		m.statusbar.SetError("Invalid variables JSON: " + err.Error())
		return *m, nil
	}

	query := m.editor.Value()
	if query == "" {
		return *m, nil
	}

	m.querying = true
	m.statusbar.SetLoading()

	ctx, cancel := context.WithCancel(context.Background())
	m.cancelQuery = cancel

	req := graphql.Request{
		Query:     query,
		Variables: vars,
	}
	client := m.gqlClient

	cmd := func() tea.Msg {
		result, err := client.Execute(ctx, ep, req)
		if err != nil {
			if ctx.Err() != nil {
				return QueryAbortedMsg{}
			}
			return QueryErrorMsg{Err: err}
		}
		return QueryResultMsg{Result: result}
	}

	return *m, cmd
}

func (m *Model) setFocus(p Panel) {
	// Blur current
	switch m.focus {
	case PanelEndpoint:
		m.endpoint.Blur()
	case PanelEditor:
		m.editor.Blur()
	case PanelVariables:
		m.variables.Blur()
	}

	m.focus = p

	// Focus new
	switch p {
	case PanelEndpoint:
		m.endpoint.Focus()
	case PanelEditor:
		m.editor.Focus()
	case PanelVariables:
		m.variables.Focus()
	}
}

func (m *Model) updateFocused(msg tea.Msg) []tea.Cmd {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	switch m.focus {
	case PanelEndpoint:
		m.endpoint, cmd = m.endpoint.Update(msg)
	case PanelEditor:
		m.editor, cmd = m.editor.Update(msg)
	case PanelVariables:
		m.variables, cmd = m.variables.Update(msg)
	case PanelResults:
		m.results, cmd = m.results.Update(msg)
	}
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	return cmds
}

func (m *Model) layoutPanels() {
	// Reserve: 1 line endpoint, 1 line status bar, 2 lines borders
	contentHeight := m.height - 4

	leftWidth := m.width / 2
	rightWidth := m.width - leftWidth

	editorHeight := contentHeight * 6 / 10
	varsHeight := contentHeight - editorHeight

	m.endpoint.SetWidth(m.width)
	m.editor.SetSize(leftWidth, editorHeight)
	m.variables.SetSize(leftWidth, varsHeight)
	m.results.SetSize(rightWidth, contentHeight)
	m.statusbar.SetWidth(m.width)
}
