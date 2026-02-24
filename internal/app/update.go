package app

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/qraqula/qla/internal/format"
	"github.com/qraqula/qla/internal/graphql"
	"github.com/qraqula/qla/internal/history"
	"github.com/qraqula/qla/internal/overlay"
	"github.com/qraqula/qla/internal/schema"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layoutPanels()
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case QueryResultMsg:
		m.querying = false
		m.cancelQuery = nil
		r := msg.Result
		hasErrors := r.Response.HasErrors()

		// Build display content with syntax highlighting
		raw, _ := json.Marshal(r.Response)
		if err := m.results.SetPrettyJSON(raw); err != nil {
			m.results.SetContent(string(raw))
		}
		m.statusbar.SetResult(r.StatusCode, r.Duration, r.Size, hasErrors)

		// Auto-save to history
		query := m.editor.Value()
		vars := m.variables.Value()
		ep := m.endpoint.Value()
		if query != "" && !m.histStore.IsDuplicate(query, vars, ep) {
			entry := history.Entry{
				ID:        history.GenerateID(),
				Name:      history.EntryNameFromQuery(query),
				Query:     query,
				Variables: vars,
				Endpoint:  ep,
				CreatedAt: time.Now(),
			}
			_ = m.histStore.AddEntry(entry)
			m.histSidebar.Rebuild()
			// Re-layout in case sidebar just became visible
			m.layoutPanels()
		}
		return m, nil

	case QueryErrorMsg:
		m.querying = false
		m.cancelQuery = nil
		m.results.SetContent("Error: " + msg.Err.Error())
		return m, m.setTimedError(msg.Err.Error())

	case QueryAbortedMsg:
		m.querying = false
		m.cancelQuery = nil
		m.results.SetContent("Query aborted")
		m.statusbar.SetAborted()
		return m, nil

	case SchemaFetchedMsg:
		m.browser.SetSchema(msg.Schema)
		m.statusbar.SetSchemaLoaded(len(msg.Schema.Types))
		return m, nil

	case SchemaFetchErrorMsg:
		return m, m.setTimedError("Schema fetch failed: " + msg.Err.Error())

	case history.LoadEntryMsg:
		m.editor.SetValue(msg.Entry.Query)
		m.variables.SetValue(msg.Entry.Variables)
		m.endpoint.SetValue(msg.Entry.Endpoint)
		m.setFocus(PanelEditor)
		return m, nil

	case history.SidebarUpdatedMsg:
		// Re-layout in case sidebar content changed visibility
		m.layoutPanels()
		return m, nil

	case overlay.CloseMsg:
		m.overlay.Close()
		return m, nil

	case overlay.ConfigChangedMsg:
		m.configStore.Config = msg.Config
		_ = m.configStore.Save()
		if env := m.configStore.Config.ActiveEnvironment(); env != nil {
			m.endpoint.SetValue(env.Endpoint)
			m.endpoint.SetEnvName(env.Name)
		} else {
			m.endpoint.SetEnvName("")
		}
		m.layoutPanels()
		return m, nil

	case EditorFinishedMsg:
		if msg.Err != nil {
			return m, m.setTimedError("Editor: " + msg.Err.Error())
		}
		content := strings.TrimRight(msg.Content, "\n")
		var editCmd tea.Cmd
		switch msg.Panel {
		case PanelEditor:
			m.editor.SetValue(content)
			editCmd = m.editor.StartEditing()
		case PanelVariables:
			m.variables.SetValue(content)
			editCmd = m.variables.StartEditing()
		}
		m.statusbar.SetHints(editingHints)
		return m, editCmd

	case statusClearMsg:
		if msg.gen == m.statusClearGen {
			m.statusbar.Clear()
		}
		return m, nil

	case lintMsg:
		if msg.gen == m.lintGen {
			m.runLint()
		}
		return m, nil
	}

	// Forward to focused panel
	cmds = append(cmds, m.updateFocused(msg)...)
	return m, tea.Batch(cmds...)
}

func (m *Model) handleKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	// Toggle overlay regardless of overlay state
	if key.Matches(msg, keys.ToggleOverlay) {
		if m.overlay.IsOpen() {
			m.overlay.Close()
		} else {
			m.overlay.Open(&m.configStore.Config, m.width, m.height)
		}
		return *m, nil
	}

	// When overlay is open, route all other input to it
	if m.overlay.IsOpen() {
		var cmd tea.Cmd
		m.overlay, cmd = m.overlay.Update(msg)
		return *m, cmd
	}

	showSidebar := m.shouldShowSidebar()

	switch {
	case key.Matches(msg, keys.Quit):
		return *m, tea.Quit

	case key.Matches(msg, keys.Abort):
		if m.querying && m.cancelQuery != nil {
			m.cancelQuery()
			return *m, nil
		}
		return *m, tea.Quit

	case key.Matches(msg, keys.ToggleSidebar):
		m.sidebarOpen = !m.sidebarOpen
		m.histStore.Meta.SidebarOpen = m.sidebarOpen
		_ = m.histStore.Save()
		if !m.shouldShowSidebar() && m.focus == PanelHistory {
			m.setFocus(PanelEditor)
		}
		m.layoutPanels()
		return *m, nil

	case key.Matches(msg, keys.Execute):
		return m.executeQuery()

	// Enter to start editing in query/variables panels
	case msg.String() == "enter" && m.focus == PanelEditor && !m.editor.Editing():
		cmd := m.editor.StartEditing()
		m.statusbar.SetHints(editingHints)
		return *m, cmd
	case msg.String() == "enter" && m.focus == PanelVariables && !m.variables.Editing():
		cmd := m.variables.StartEditing()
		m.statusbar.SetHints(editingHints)
		return *m, cmd

	case msg.String() == "enter" && m.focus == PanelResults && m.rightPanelMode == modeResults && !m.results.Searching():
		return m.executeQuery()

	// Escape to stop editing + lint
	case msg.String() == "esc" && m.focus == PanelEditor && m.editor.Editing():
		m.editor.StopEditing()
		m.statusbar.SetHints(hintsForFocus(m.focus, m.rightPanelMode))
		var cmd tea.Cmd
		if err := format.ValidateGraphQL(m.editor.Value()); err != nil {
			cmd = m.setTimedError("Query: " + err.Error())
		}
		return *m, cmd
	case msg.String() == "esc" && m.focus == PanelVariables && m.variables.Editing():
		m.variables.StopEditing()
		m.statusbar.SetHints(hintsForFocus(m.focus, m.rightPanelMode))
		var cmd tea.Cmd
		if v := strings.TrimSpace(m.variables.Value()); v != "" {
			if err := format.ValidateJSON(v); err != nil {
				cmd = m.setTimedError("Variables: invalid JSON")
			}
		}
		return *m, cmd

	case key.Matches(msg, keys.Tab):
		if m.isEditing() {
			// Let tab pass through to textarea for indentation
		} else {
			return m.switchFocus(nextPanel(m.focus, showSidebar))
		}

	case key.Matches(msg, keys.ShiftTab):
		if m.isEditing() {
			// Let shift+tab pass through to textarea
		} else {
			return m.switchFocus(prevPanel(m.focus, showSidebar))
		}

	case key.Matches(msg, keys.FocusUp):
		return m.switchFocus(navigatePanel(m.focus, "up", showSidebar))
	case key.Matches(msg, keys.FocusDown):
		return m.switchFocus(navigatePanel(m.focus, "down", showSidebar))
	case key.Matches(msg, keys.FocusLeft):
		return m.switchFocus(navigatePanel(m.focus, "left", showSidebar))
	case key.Matches(msg, keys.FocusRight):
		return m.switchFocus(navigatePanel(m.focus, "right", showSidebar))

	case key.Matches(msg, keys.ToggleDocs):
		if m.rightPanelMode == modeResults {
			m.rightPanelMode = modeSchema
			m.setFocus(PanelResults)
		} else {
			m.rightPanelMode = modeResults
		}
		m.statusbar.SetHints(hintsForFocus(m.focus, m.rightPanelMode))
		return *m, nil

	case key.Matches(msg, keys.CycleEnv):
		return m.cycleEnvironment()

	case key.Matches(msg, keys.RefreshSchema):
		return m.fetchSchema()

	case key.Matches(msg, keys.Prettify):
		if m.focus == PanelEditor {
			formatted := format.GraphQL(m.editor.Value())
			m.editor.SetValue(formatted)
			var cmd tea.Cmd
			if err := format.ValidateGraphQL(formatted); err != nil {
				cmd = m.setTimedError("Query: " + err.Error())
			}
			return *m, cmd
		}
		if m.focus == PanelVariables {
			v := strings.TrimSpace(m.variables.Value())
			if v != "" {
				formatted, err := format.JSON(v)
				if err != nil {
					return *m, m.setTimedError("Variables: invalid JSON")
				}
				m.variables.SetValue(formatted)
			}
			return *m, nil
		}
		return *m, nil

	case key.Matches(msg, keys.OpenEditor):
		if m.focus == PanelEditor || m.focus == PanelVariables {
			return m.openExternalEditor()
		}
		return *m, nil

	case key.Matches(msg, keys.ToggleSearch):
		if m.focus == PanelResults && m.rightPanelMode == modeResults {
			m.results.ToggleSearch()
			m.results.SetSize(m.rightW, m.contentH-2) // recalculate viewport height
			return *m, nil
		}
	}

	// Forward to focused panel
	var cmds []tea.Cmd
	cmds = append(cmds, m.updateFocused(msg)...)
	return *m, tea.Batch(cmds...)
}

func (m *Model) isEditing() bool {
	return (m.focus == PanelEditor && m.editor.Editing()) ||
		(m.focus == PanelVariables && m.variables.Editing())
}

// setTimedError shows an error in the status bar that auto-clears after 3 seconds.
func (m *Model) setTimedError(msg string) tea.Cmd {
	m.statusbar.SetError(msg)
	m.statusClearGen++
	gen := m.statusClearGen
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg {
		return statusClearMsg{gen: gen}
	})
}

// scheduleLint starts a 500ms debounce timer for linting.
func (m *Model) scheduleLint() tea.Cmd {
	m.lintGen++
	gen := m.lintGen
	return tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
		return lintMsg{gen: gen}
	})
}

// runLint validates the content of the currently focused editor.
func (m *Model) runLint() {
	switch m.focus {
	case PanelEditor:
		if err := format.ValidateGraphQL(m.editor.Value()); err != nil {
			m.statusbar.SetError("Query: " + err.Error())
		} else {
			m.statusbar.Clear()
		}
	case PanelVariables:
		if v := strings.TrimSpace(m.variables.Value()); v != "" {
			if err := format.ValidateJSON(v); err != nil {
				m.statusbar.SetError("Variables: invalid JSON")
			} else {
				m.statusbar.Clear()
			}
		} else {
			m.statusbar.Clear()
		}
	}
}

// switchFocus changes panel focus and auto-fetches schema when leaving the endpoint panel.
func (m *Model) switchFocus(target Panel) (Model, tea.Cmd) {
	prev := m.focus
	m.setFocus(target)
	if prev == PanelEndpoint && m.focus != PanelEndpoint {
		ep := m.endpoint.Value()
		if ep != "" && ep != m.lastEndpoint {
			m.lastEndpoint = ep
			return m.fetchSchema()
		}
	}
	return *m, nil
}

func (m *Model) executeQuery() (Model, tea.Cmd) {
	if m.querying {
		return *m, nil
	}
	ep := m.endpoint.Value()
	if ep == "" {
		return *m, m.setTimedError("No endpoint configured")
	}

	vars, err := m.variables.ParsedVariables()
	if err != nil {
		return *m, m.setTimedError("Invalid variables JSON: " + err.Error())
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
	headers := m.configStore.Config.MergedHeaders()

	cmd := func() tea.Msg {
		result, err := client.Execute(ctx, ep, req, headers)
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

func (m *Model) cycleEnvironment() (Model, tea.Cmd) {
	names := m.configStore.Config.EnvNames()
	if len(names) == 0 {
		return *m, m.setTimedError("No environments configured (ctrl+e to add)")
	}

	current := m.configStore.Config.ActiveEnv
	// Find current index; cycle includes a "none" slot at the end
	nextIdx := 0
	for i, n := range names {
		if n == current {
			nextIdx = i + 1
			break
		}
	}

	if nextIdx >= len(names) {
		// Unset — cycle to "none"
		m.configStore.Config.ActiveEnv = ""
		m.endpoint.SetEnvName("")
	} else {
		m.configStore.Config.ActiveEnv = names[nextIdx]
		env := m.configStore.Config.ActiveEnvironment()
		if env != nil {
			m.endpoint.SetValue(env.Endpoint)
			m.endpoint.SetEnvName(env.Name)
		}
	}
	_ = m.configStore.Save()
	m.layoutPanels()
	return *m, nil
}

func (m *Model) fetchSchema() (Model, tea.Cmd) {
	ep := m.endpoint.Value()
	if ep == "" {
		return *m, m.setTimedError("No endpoint configured")
	}
	m.lastEndpoint = ep
	m.statusbar.SetSchemaLoading()
	client := m.gqlClient
	headers := m.configStore.Config.MergedHeaders()
	cmd := func() tea.Msg {
		s, err := schema.FetchSchema(context.Background(), client, ep, headers)
		if err != nil {
			return SchemaFetchErrorMsg{Err: err}
		}
		return SchemaFetchedMsg{Schema: s}
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
	case PanelResults:
		m.browser.Blur()
	case PanelHistory:
		m.histSidebar.Blur()
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
	case PanelHistory:
		m.histSidebar.Focus()
	}

	// Update status bar hints for the focused panel
	m.statusbar.SetHints(hintsForFocus(p, m.rightPanelMode))
}

func (m *Model) updateFocused(msg tea.Msg) []tea.Cmd {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	switch m.focus {
	case PanelEndpoint:
		m.endpoint, cmd = m.endpoint.Update(msg)
	case PanelEditor:
		prev := m.editor.Value()
		m.editor, cmd = m.editor.Update(msg)
		if m.editor.Editing() && m.editor.Value() != prev {
			cmds = append(cmds, m.scheduleLint())
		}
	case PanelVariables:
		prev := m.variables.Value()
		m.variables, cmd = m.variables.Update(msg)
		if m.variables.Editing() && m.variables.Value() != prev {
			cmds = append(cmds, m.scheduleLint())
		}
	case PanelResults:
		if m.rightPanelMode == modeSchema {
			m.browser, cmd = m.browser.Update(msg)
		} else {
			m.results, cmd = m.results.Update(msg)
		}
	case PanelHistory:
		m.histSidebar, cmd = m.histSidebar.Update(msg)
	}
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	return cmds
}

func (m *Model) layoutPanels() {
	// Vertical budget:
	//   endpoint bar: 3 lines (1 content + 2 border)
	//   status bar: 1 line
	//   content area: m.height - 4
	//
	// lipgloss .Height() sets TOTAL height (including border).
	// Sidebar and results each get the full content area height.
	// Editor + variables split the same height between them.
	totalH := m.height - 4
	if totalH < 4 {
		totalH = 4
	}

	m.contentH = totalH
	m.editorH = totalH * 6 / 10
	m.varsH = totalH - m.editorH

	if m.shouldShowSidebar() {
		// Each panel border = 2 chars wide (left+right), 3 panels = 6
		available := m.width - 6
		if available < 30 {
			available = 30
		}

		m.sidebarW = available / 5
		if m.sidebarW < 18 {
			m.sidebarW = 18
		}
		if m.sidebarW > 33 {
			m.sidebarW = 33
		}
		remaining := available - m.sidebarW
		m.midW = remaining / 2
		m.rightW = remaining - m.midW

		m.histSidebar.SetSize(m.sidebarW, m.contentH-2)
		m.editor.SetSize(m.midW, m.editorH-2)
		m.variables.SetSize(m.midW, m.varsH-2)
		m.results.SetSize(m.rightW, m.contentH-2)
		m.browser.SetSize(m.rightW, m.contentH-2)
	} else {
		// Each panel border = 2 chars wide, 2 panels = 4
		available := m.width - 4
		if available < 20 {
			available = 20
		}

		m.leftW = available / 2
		m.rightW = available - m.leftW

		m.editor.SetSize(m.leftW, m.editorH-2)
		m.variables.SetSize(m.leftW, m.varsH-2)
		m.results.SetSize(m.rightW, m.contentH-2)
		m.browser.SetSize(m.rightW, m.contentH-2)
	}

	epW := m.width
	if m.endpoint.EnvName() != "" {
		epW -= len(m.endpoint.EnvName()) + 3 // badge " name " + gap
	}
	m.endpoint.SetWidth(epW)
	m.statusbar.SetWidth(m.width)
}

func (m *Model) openExternalEditor() (Model, tea.Cmd) {
	var content string
	var ext string
	panel := m.focus

	switch panel {
	case PanelEditor:
		content = m.editor.Value()
		ext = ".graphql"
	case PanelVariables:
		content = m.variables.Value()
		ext = ".json"
	default:
		return *m, nil
	}

	tmpFile, err := os.CreateTemp("", "qla-*"+ext)
	if err != nil {
		return *m, m.setTimedError("Failed to create temp file: " + err.Error())
	}
	tmpFile.WriteString(content)
	tmpFile.Close()

	editorCmd := os.Getenv("VISUAL")
	if editorCmd == "" {
		editorCmd = os.Getenv("EDITOR")
	}
	if editorCmd == "" {
		editorCmd = "vim"
	}

	c := exec.Command(editorCmd, tmpFile.Name())
	path := tmpFile.Name()

	return *m, tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			return EditorFinishedMsg{Err: err, Panel: panel}
		}
		data, readErr := os.ReadFile(path)
		os.Remove(path)
		if readErr != nil {
			return EditorFinishedMsg{Err: readErr, Panel: panel}
		}
		return EditorFinishedMsg{Content: string(data), Panel: panel}
	})
}
