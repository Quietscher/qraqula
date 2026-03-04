package builder

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/qraqula/qla/internal/highlight"
	"github.com/qraqula/qla/internal/schema"
	"github.com/qraqula/qla/internal/statusbar"
)

// Messages emitted by the builder to the parent app.

// CloseMsg is sent when the builder is dismissed without applying.
type CloseMsg struct{}

// ApplyMsg is sent when the user applies the built query.
type ApplyMsg struct {
	Query     string
	Variables string
}

// builderMode tracks the current interaction mode.
type builderMode int

const (
	modePickOperation builderMode = iota
	modeTree
)

// builderPane tracks which pane has focus within the builder.
type builderPane int

const (
	panePreview builderPane = iota
	paneTree
	paneArgs
	paneCount
)

// Model is the Bubble Tea model for the query builder overlay.
type Model struct {
	schema *schema.Schema

	visible bool
	mode    builderMode
	pane    builderPane

	// Operation picker state
	opItems  []opItem // available root fields
	opCursor int
	opType   string // "query", "mutation", "subscription"

	// Tree state
	root    *TreeNode
	opField string     // name of the root operation field
	flat    []FlatNode // visible flattened nodes
	cursor  int        // cursor position in flat list (indexes into visibleFlat())

	// Filter state for field tree search
	filtering   bool
	filterInput textinput.Model

	// Args state
	argCursor int
	argNodes  []schema.InputValue // args for current field (or root)
	argNode   *TreeNode           // the node whose args are displayed
	argScroll int                 // horizontal scroll offset for args display

	// Preview viewport (scrollable)
	preview        viewport.Model
	previewContent string // raw preview text for direct rendering

	// Status bar (reused from statusbar package)
	statusbar statusbar.Model

	// Dimensions
	width  int
	height int
}

type opItem struct {
	OpType    string // "query", "mutation", "subscription"
	FieldName string
	Field     schema.Field
}

// New creates a new builder model (initially not visible).
func New() Model {
	fi := textinput.New()
	fi.Prompt = "/ "
	fi.CharLimit = 100
	fiStyles := fi.Styles()
	fiStyles.Focused.Prompt = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	fiStyles.Cursor.Color = lipgloss.Color("196")
	fi.SetStyles(fiStyles)

	return Model{
		preview:     viewport.New(),
		statusbar:   statusbar.New(),
		filterInput: fi,
	}
}

// IsOpen returns true when the builder overlay is visible.
func (m Model) IsOpen() bool {
	return m.visible
}

// SetSize updates the builder dimensions.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// OpenBlank opens the builder in operation picker mode.
func (m *Model) OpenBlank(s *schema.Schema) {
	m.schema = s
	m.visible = true
	m.mode = modePickOperation
	m.pane = paneTree
	m.buildOpItems()
	m.opCursor = 0
	m.updateStatusHints()
}

// OpenFromSchemaField opens the builder pre-loaded with a specific operation field.
func (m *Model) OpenFromSchemaField(s *schema.Schema, opType string, field schema.Field) {
	m.schema = s
	m.visible = true
	m.mode = modeTree
	m.pane = paneTree
	m.opType = opType
	m.opField = field.Name
	m.root = BuildTreeFromField(s, field)
	m.cursor = 0
	m.rebuildFlat()
	m.updatePreview()
	m.updateStatusHints()
}

// OpenWithQuery opens the builder and parses an existing query to pre-select fields.
// Falls back to operation picker on parse failure.
func (m *Model) OpenWithQuery(s *schema.Schema, queryStr string) {
	root, opType, opField, err := ParseExistingQuery(queryStr, s)
	if err != nil {
		m.OpenBlank(s)
		return
	}
	m.schema = s
	m.visible = true
	m.mode = modeTree
	m.pane = paneTree
	m.opType = opType
	m.opField = opField
	m.root = root
	m.cursor = 0
	m.rebuildFlat()
	m.updatePreview()
	m.updateStatusHints()
}

// Close dismisses the builder.
func (m *Model) Close() {
	m.visible = false
	m.root = nil
	m.flat = nil
	m.filtering = false
	m.filterInput.Blur()
	m.filterInput.SetValue("")
}

// Update handles messages when the builder is open.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	// Forward non-key messages to viewport when preview is focused
	if m.pane == panePreview {
		var cmd tea.Cmd
		m.preview, cmd = m.preview.Update(msg)
		return m, cmd
	}

	// Forward non-key messages to filter input when filtering (e.g. cursor blink)
	if m.filtering {
		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	key := msg.String()

	// Global builder keys
	switch key {
	case "esc":
		if m.filtering {
			m.clearFilter()
			m.updateStatusHints()
			return m, nil
		}
		m.Close()
		return m, func() tea.Msg { return CloseMsg{} }
	case "alt+enter":
		if m.mode == modeTree && m.root != nil {
			query, vars := GenerateFromTree(m.schema, m.opType, m.opField, m.root)
			m.Close()
			return m, func() tea.Msg {
				return ApplyMsg{Query: query, Variables: vars}
			}
		}
		return m, nil
	case "tab":
		m.cyclePaneForward()
		m.updateStatusHints()
		return m, nil
	case "shift+tab":
		m.cyclePaneBackward()
		m.updateStatusHints()
		return m, nil
	}

	switch m.mode {
	case modePickOperation:
		return m.handlePickKey(key)
	case modeTree:
		return m.handleTreeKey(msg)
	}

	return m, nil
}

func (m Model) handlePickKey(key string) (Model, tea.Cmd) {
	switch key {
	case "j", "down":
		if m.opCursor < len(m.opItems)-1 {
			m.opCursor++
		}
	case "k", "up":
		if m.opCursor > 0 {
			m.opCursor--
		}
	case "enter", "l", "right":
		if m.opCursor < len(m.opItems) {
			item := m.opItems[m.opCursor]
			m.opType = item.OpType
			m.opField = item.FieldName
			m.root = BuildTreeFromField(m.schema, item.Field)
			m.mode = modeTree
			m.cursor = 0
			m.rebuildFlat()
			m.updatePreview()
			m.updateStatusHints()
		}
	}
	return m, nil
}

func (m Model) handleTreeKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	key := msg.String()

	switch m.pane {
	case panePreview:
		// Forward j/k/up/down to viewport for scrolling
		var cmd tea.Cmd
		m.preview, cmd = m.preview.Update(msg)
		return m, cmd
	case paneTree:
		return m.handleTreePaneKey(msg)
	case paneArgs:
		return m.handleArgsPaneKey(key)
	}
	return m, nil
}

func (m Model) handleTreePaneKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	if m.filtering {
		return m.handleFilterKey(msg)
	}

	key := msg.String()
	switch key {
	case "/":
		m.filtering = true
		m.filterInput.SetValue("")
		m.cursor = 0
		m.updateStatusHints()
		return m, m.filterInput.Focus()
	case "j", "down":
		vis := m.visibleFlat()
		if m.cursor < len(vis)-1 {
			m.cursor++
			m.updateArgsFromVisible()
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			m.updateArgsFromVisible()
		}
	case "space", " ":
		vis := m.visibleFlat()
		if m.cursor < len(vis) {
			node := vis[m.cursor].Node
			ToggleSelected(node)
			m.updatePreview()
		}
	case "S":
		vis := m.visibleFlat()
		if m.cursor < len(vis) {
			node := vis[m.cursor].Node
			ToggleChildrenSelected(m.schema, node)
			m.rebuildFlat()
			m.updatePreview()
		}
	case "l", "enter", "right":
		vis := m.visibleFlat()
		if m.cursor < len(vis) {
			node := vis[m.cursor].Node
			if !node.IsLeaf {
				EnsureChildrenReady(m.schema, node)
				node.Expanded = true
				m.rebuildFlat()
				m.updatePreview()
			}
		}
	case "h", "left":
		vis := m.visibleFlat()
		if m.cursor < len(vis) {
			node := vis[m.cursor].Node
			if node.Expanded && len(node.Children) > 0 {
				node.Expanded = false
				m.rebuildFlat()
				// Keep cursor valid
				vis = m.visibleFlat()
				if m.cursor >= len(vis) {
					m.cursor = max(0, len(vis)-1)
				}
			} else if node.Parent != nil && node.Parent != m.root {
				// Navigate to parent
				for i, fn := range vis {
					if fn.Node == node.Parent {
						m.cursor = i
						break
					}
				}
			}
		}
	case "G":
		vis := m.visibleFlat()
		if len(vis) > 0 {
			m.cursor = len(vis) - 1
			m.updateArgsFromVisible()
		}
	case "g":
		m.cursor = 0
		m.updateArgsFromVisible()
	}
	return m, nil
}

// handleFilterKey handles keys when the tree filter input is active.
func (m Model) handleFilterKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "j", "down":
		vis := m.visibleFlat()
		if m.cursor < len(vis)-1 {
			m.cursor++
			m.updateArgsFromVisible()
		}
		return m, nil
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			m.updateArgsFromVisible()
		}
		return m, nil
	case "space":
		vis := m.visibleFlat()
		if m.cursor < len(vis) {
			node := vis[m.cursor].Node
			ToggleSelected(node)
			m.updatePreview()
		}
		return m, nil
	case "l", "enter":
		vis := m.visibleFlat()
		if m.cursor < len(vis) {
			node := vis[m.cursor].Node
			if !node.IsLeaf {
				EnsureChildrenReady(m.schema, node)
				node.Expanded = true
				m.rebuildFlat()
				m.updatePreview()
				// Clamp cursor to new visible list
				vis = m.visibleFlat()
				if m.cursor >= len(vis) {
					m.cursor = max(0, len(vis)-1)
				}
			}
		}
		return m, nil
	case "h":
		vis := m.visibleFlat()
		if m.cursor < len(vis) {
			node := vis[m.cursor].Node
			if node.Expanded && len(node.Children) > 0 {
				node.Expanded = false
				m.rebuildFlat()
				vis = m.visibleFlat()
				if m.cursor >= len(vis) {
					m.cursor = max(0, len(vis)-1)
				}
			} else if node.Parent != nil && node.Parent != m.root {
				for i, fn := range vis {
					if fn.Node == node.Parent {
						m.cursor = i
						break
					}
				}
			}
		}
		return m, nil
	default:
		// Forward all other keys to the text input
		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		// Clamp cursor after filter text changes
		vis := m.visibleFlat()
		if m.cursor >= len(vis) {
			m.cursor = max(0, len(vis)-1)
		}
		return m, cmd
	}
}

func (m Model) handleArgsPaneKey(key string) (Model, tea.Cmd) {
	switch key {
	case "j", "down", "l", "right":
		if m.argCursor < len(m.argNodes)-1 {
			m.argCursor++
		}
	case "k", "up", "h", "left":
		if m.argCursor > 0 {
			m.argCursor--
		}
	case "space", " ":
		if m.argNode != nil && m.argCursor < len(m.argNodes) {
			argName := m.argNodes[m.argCursor].Name
			m.argNode.ArgValues[argName] = !m.argNode.ArgValues[argName]
			m.updatePreview()
		}
	}
	return m, nil
}

// --- Internal helpers ---

func (m *Model) buildOpItems() {
	m.opItems = nil
	if m.schema == nil {
		return
	}
	roots := m.schema.RootTypes()
	for _, rt := range roots {
		opType := rootTypeToOp(m.schema, rt.Name)
		for _, f := range rt.Fields {
			m.opItems = append(m.opItems, opItem{
				OpType:    opType,
				FieldName: f.Name,
				Field:     f,
			})
		}
	}
}

func (m *Model) rebuildFlat() {
	if m.root == nil {
		m.flat = nil
		return
	}
	m.flat = FlattenVisible(m.root)
	if m.cursor >= len(m.flat) {
		m.cursor = max(0, len(m.flat)-1)
	}
	m.updateArgsList()
}

// visibleFlat returns the flat nodes filtered by the current search query.
// When not filtering or query is empty, returns the full m.flat.
func (m *Model) visibleFlat() []FlatNode {
	if !m.filtering || m.filterInput.Value() == "" {
		return m.flat
	}
	query := strings.ToLower(m.filterInput.Value())
	var result []FlatNode
	for _, fn := range m.flat {
		if strings.Contains(strings.ToLower(fn.Node.Name), query) {
			result = append(result, fn)
		}
	}
	return result
}

// clearFilter exits filter mode, restoring the cursor to the currently selected
// node's position in the full flat list.
func (m *Model) clearFilter() {
	if !m.filtering {
		return
	}
	// Find the node at the current filtered cursor position
	vis := m.visibleFlat()
	var currentNode *TreeNode
	if m.cursor < len(vis) {
		currentNode = vis[m.cursor].Node
	}
	m.filtering = false
	m.filterInput.Blur()
	m.filterInput.SetValue("")
	// Restore cursor to the same node in the full flat list
	if currentNode != nil {
		for i, fn := range m.flat {
			if fn.Node == currentNode {
				m.cursor = i
				break
			}
		}
	}
	if m.cursor >= len(m.flat) {
		m.cursor = max(0, len(m.flat)-1)
	}
}

// updateArgsFromVisible updates the args list based on the current cursor in visibleFlat.
func (m *Model) updateArgsFromVisible() {
	m.argNodes = nil
	m.argCursor = 0
	if m.root != nil && len(m.root.Args) > 0 {
		m.argNodes = m.root.Args
		m.argNode = m.root
	}
	vis := m.visibleFlat()
	if m.cursor < len(vis) {
		node := vis[m.cursor].Node
		if len(node.Args) > 0 {
			m.argNodes = node.Args
			m.argNode = node
		}
	}
}

func (m *Model) updateArgsList() {
	m.argNodes = nil
	m.argCursor = 0
	// Always include root-level args (the operation's arguments)
	if m.root != nil && len(m.root.Args) > 0 {
		m.argNodes = m.root.Args
		m.argNode = m.root
	}
	// If cursor is on a different node with args, show those instead
	if m.cursor < len(m.flat) {
		node := m.flat[m.cursor].Node
		if len(node.Args) > 0 {
			m.argNodes = node.Args
			m.argNode = node
		}
	}
}

func (m *Model) updatePreview() {
	if m.root == nil {
		m.previewContent = ""
		m.preview.SetContent("")
		return
	}
	query, vars := GenerateFromTree(m.schema, m.opType, m.opField, m.root)
	var buf strings.Builder
	buf.WriteString(highlight.Colorize(query, "graphql"))
	if vars != "" {
		buf.WriteString("\n\n")
		buf.WriteString(dimStyle.Render("Variables:"))
		buf.WriteString("\n")
		buf.WriteString(highlight.Colorize(vars, "json"))
	}
	m.previewContent = buf.String()
	m.preview.SetContent(m.previewContent)
}

func (m *Model) cyclePaneForward() {
	if m.mode != modeTree {
		return
	}
	m.pane = (m.pane + 1) % paneCount
	// Skip args pane if no args
	if m.pane == paneArgs && len(m.argNodes) == 0 {
		m.pane = (m.pane + 1) % paneCount
	}
}

func (m *Model) cyclePaneBackward() {
	if m.mode != modeTree {
		return
	}
	m.pane = (m.pane - 1 + paneCount) % paneCount
	if m.pane == paneArgs && len(m.argNodes) == 0 {
		m.pane = (m.pane - 1 + paneCount) % paneCount
	}
}

// updateStatusHints sets the statusbar hints for the current mode/pane.
func (m *Model) updateStatusHints() {
	m.statusbar.SetWidth(m.width)

	switch m.mode {
	case modePickOperation:
		m.statusbar.SetHints([]statusbar.Hint{
			{Key: "j/k", Label: "navigate"},
			{Key: "↵", Label: "select"},
			{Key: "esc", Label: "cancel"},
		})
	case modeTree:
		switch m.pane {
		case panePreview:
			m.statusbar.SetHints([]statusbar.Hint{
				{Key: "j/k", Label: "scroll"},
				{Key: "tab", Label: "pane"},
				{Key: "alt+↵", Label: "apply"},
				{Key: "esc", Label: "cancel"},
			})
		case paneTree:
			if m.filtering {
				m.statusbar.SetHints([]statusbar.Hint{
					{Key: "j/k", Label: "navigate"},
					{Key: "space", Label: "toggle"},
					{Key: "l", Label: "expand"},
					{Key: "h", Label: "collapse"},
					{Key: "esc", Label: "clear filter"},
				})
			} else {
				m.statusbar.SetHints([]statusbar.Hint{
					{Key: "j/k", Label: "navigate"},
					{Key: "space", Label: "toggle"},
					{Key: "S", Label: "children"},
					{Key: "l/→", Label: "expand"},
					{Key: "h/←", Label: "collapse"},
					{Key: "/", Label: "filter"},
					{Key: "tab", Label: "pane"},
					{Key: "alt+↵", Label: "apply"},
					{Key: "esc", Label: "cancel"},
				})
			}
		case paneArgs:
			m.statusbar.SetHints([]statusbar.Hint{
				{Key: "h/l", Label: "navigate"},
				{Key: "space", Label: "toggle"},
				{Key: "tab", Label: "pane"},
				{Key: "alt+↵", Label: "apply"},
				{Key: "esc", Label: "cancel"},
			})
		}
	}

	m.statusbar.SetInfo("Query Builder")
}

// --- View ---

// RenderOver renders the builder on top of the given background string.
func (m Model) RenderOver(background string) string {
	if !m.visible {
		return background
	}

	w := m.width
	h := m.height

	switch m.mode {
	case modePickOperation:
		return m.renderPickerOverlay(w, h)
	case modeTree:
		return m.renderTreeOverlay(w, h)
	}
	return background
}

func (m Model) renderPickerOverlay(w, h int) string {
	// Single bordered box for the operation picker, centered
	boxW := w*95/100 - 2
	boxH := h*9/10 - 2
	contentW := boxW - 2
	contentH := boxH

	pickerContent := m.renderPicker(contentW, contentH)
	box := builderBorder.
		Width(boxW).
		Height(boxH).
		Render(pickerContent)

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceChars(" "),
	)
}

func (m Model) renderTreeOverlay(w, h int) string {
	// Full-screen layout:
	//   Top row: preview (left) | field tree (right) — same height
	//   Bottom row: args (full width, horizontal) — only if args exist, fixed 3 lines
	//   Status bar (1 line)
	statusH := 1
	argsOuterH := 0
	hasArgs := len(m.argNodes) > 0
	if hasArgs {
		argsOuterH = 3 // border(2) + 1 content line (horizontal args)
	}

	topH := h - statusH - argsOuterH

	// Split top row: preview 40%, tree 60%
	previewOuterW := w * 40 / 100
	treeOuterW := w - previewOuterW

	previewInnerW := previewOuterW - 4 // -2 border -2 padding
	previewInnerH := topH - 2          // -2 border
	treeInnerW := treeOuterW - 4
	treeInnerH := topH - 2

	// Render preview pane — always use viewport for proper height clipping
	m.preview.SetWidth(previewInnerW)
	m.preview.SetHeight(previewInnerH)
	previewRendered := padToHeight(m.preview.View(), previewInnerH)
	previewBox := m.paneBorderStyle(panePreview, previewOuterW-2, topH-2).Render(previewRendered)

	// Render tree pane (already has scroll via scrollWindow)
	treeContent := padToHeight(m.renderFieldTree(treeInnerW, treeInnerH), treeInnerH)
	treeBox := m.paneBorderStyle(paneTree, treeOuterW-2, topH-2).Render(treeContent)

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, previewBox, treeBox)

	// Render args pane (horizontal layout, if args exist)
	var argsBox string
	if hasArgs {
		argsInnerW := w - 4 // -2 border -2 padding
		argsContent := m.renderArgsHorizontal(argsInnerW)
		argsBox = m.paneBorderStyle(paneArgs, w-2, argsOuterH-2).Render(argsContent)
	}

	// Render statusbar
	m.statusbar.SetWidth(w)
	status := lipgloss.NewStyle().Width(w).Foreground(lipgloss.Color("245")).Render(m.statusbar.View())

	if hasArgs {
		return lipgloss.JoinVertical(lipgloss.Left, topRow, argsBox, status)
	}
	return lipgloss.JoinVertical(lipgloss.Left, topRow, status)
}

// paneBorderStyle returns the border style for a pane, focused or blurred.
func (m Model) paneBorderStyle(p builderPane, w, h int) lipgloss.Style {
	base := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Width(w).
		Height(h)

	if m.pane == p {
		return base.BorderForeground(lipgloss.Color("62"))
	}
	return base.BorderForeground(lipgloss.Color("240"))
}

func (m Model) renderPicker(w, h int) string {
	var lines []string
	lines = append(lines, titleStyle.Render("Select Operation"))
	lines = append(lines, "")

	// Scrollable list area
	listH := h - 4 // title, blank line, blank line, hints
	start, end := scrollWindow(m.opCursor, len(m.opItems), listH)

	if start > 0 {
		lines = append(lines, dimStyle.Render("  "+upArrow+fmt.Sprintf(" %d more", start)))
	}

	for i := start; i < end; i++ {
		item := m.opItems[i]

		var prefix string
		if i == m.opCursor {
			prefix = selectedBarStyle.String()
		} else {
			prefix = normalPrefixStyle.String()
		}

		name := item.FieldName
		typeDisplay := item.Field.Type.DisplayName()
		opBadge := opBadgeFor(item.OpType)

		var styledName string
		if i == m.opCursor {
			styledName = selectedFieldStyle.Render(name)
		} else {
			styledName = normalFieldStyle.Render(name)
		}

		line := prefix + styledName + dimStyle.Render(": ") + typeAnnotationStyle.Render(typeDisplay) + "  " + opBadge
		lines = append(lines, line)
	}

	remaining := len(m.opItems) - end
	if remaining > 0 {
		lines = append(lines, dimStyle.Render("  "+downArrow+fmt.Sprintf(" %d more", remaining)))
	}

	lines = append(lines, "")
	lines = append(lines, hintStyle.Render("j/k navigate  "+enterKey+" select  esc cancel"))

	return strings.Join(lines, "\n")
}

// renderArgsHorizontal renders args in a single horizontal row with scroll.
func (m Model) renderArgsHorizontal(w int) string {
	if len(m.argNodes) == 0 {
		return dimStyle.Render("(no arguments)")
	}

	// Build each arg as a styled segment
	type argSegment struct {
		text string
		w    int
	}
	var segments []argSegment
	for i, arg := range m.argNodes {
		enabled := false
		if m.argNode != nil {
			enabled = m.argNode.ArgValues[arg.Name]
		}

		var check string
		if enabled {
			check = checkOnStyle.Render("[x]")
		} else {
			check = checkOffStyle.Render("[ ]")
		}

		required := ""
		if isRequired(arg.Type) {
			required = requiredStyle.Render("!")
		}

		name := arg.Name + ": " + arg.Type.DisplayName() + required
		var styledName string
		if i == m.argCursor && m.pane == paneArgs {
			styledName = selectedFieldStyle.Render(name)
		} else {
			styledName = normalFieldStyle.Render(name)
		}

		seg := check + " " + styledName
		segments = append(segments, argSegment{text: seg, w: lipgloss.Width(seg)})
	}

	// Build the full line, scrolling horizontally to keep cursor visible
	sep := "  "
	sepW := 2

	// Calculate total width
	totalW := 0
	for i, s := range segments {
		totalW += s.w
		if i < len(segments)-1 {
			totalW += sepW
		}
	}

	if totalW <= w {
		// Everything fits — render all
		parts := make([]string, len(segments))
		for i, s := range segments {
			parts[i] = s.text
		}
		return strings.Join(parts, sep)
	}

	// Need horizontal scrolling — find window around cursor
	// Calculate the start position and width of cursor segment
	cursorStart := 0
	for i := 0; i < m.argCursor && i < len(segments); i++ {
		cursorStart += segments[i].w + sepW
	}

	// Center cursor segment in the available width
	var cursorW int
	if m.argCursor < len(segments) {
		cursorW = segments[m.argCursor].w
	}
	scrollStart := cursorStart - (w-cursorW)/2
	if scrollStart < 0 {
		scrollStart = 0
	}

	// Render segments within the scroll window
	var result strings.Builder
	pos := 0
	for i, s := range segments {
		segEnd := pos + s.w
		if i > 0 {
			segEnd += sepW
		}

		if pos >= scrollStart+w {
			break
		}
		if segEnd > scrollStart {
			if i > 0 && pos >= scrollStart {
				result.WriteString(sep)
			}
			result.WriteString(s.text)
		}
		pos = segEnd
	}

	// Add scroll indicators
	line := result.String()
	if scrollStart > 0 {
		line = dimStyle.Render("◀ ") + line
	}
	if scrollStart+w < totalW {
		// Trim to fit and add indicator
		line = line + dimStyle.Render(" ▶")
	}

	return line
}

func (m Model) renderFieldTree(w, h int) string {
	vis := m.visibleFlat()

	// When filtering, render filter input at the top and reduce available height
	var filterLine string
	if m.filtering {
		filterLine = m.filterInput.View()
		h-- // reserve one line for the filter input
	}

	if len(vis) == 0 {
		empty := dimStyle.Render("(no fields)")
		if filterLine != "" {
			return filterLine + "\n" + empty
		}
		return empty
	}

	// Reserve space for scroll indicators so content + indicators fit within h lines
	total := len(vis)
	availH := h
	if total > h {
		availH = h - 2 // reserve for both top/bottom scroll indicators
		if availH < 1 {
			availH = 1
		}
	}

	// Scroll window around cursor
	start, end := scrollWindow(m.cursor, total, availH)

	var lines []string
	for i := start; i < end; i++ {
		fn := vis[i]
		node := fn.Node

		// Cursor indicator
		var prefix string
		if i == m.cursor && m.pane == paneTree {
			prefix = selectedBarStyle.String()
		} else {
			prefix = normalPrefixStyle.String()
		}

		// Indentation
		indent := strings.Repeat("  ", fn.Depth)

		// Checkbox
		var check string
		if node.Selected {
			check = checkOnStyle.Render("[x]")
		} else {
			check = checkOffStyle.Render("[ ]")
		}

		// Field name + expand indicator
		name := node.Name
		expand := ""
		if !node.IsLeaf && !node.IsSpread {
			if node.Expanded {
				expand = " " + dimStyle.Render("▼")
			} else {
				expand = " " + dimStyle.Render("▶")
			}
		}

		// Style the field name
		var styledName string
		if i == m.cursor && m.pane == paneTree {
			styledName = selectedFieldStyle.Render(name)
		} else if node.Selected {
			styledName = normalFieldStyle.Render(name)
		} else {
			styledName = dimFieldStyle.Render(name)
		}

		line := prefix + indent + check + " " + styledName + expand

		// Type annotation (if space allows)
		typeStr := ""
		if !node.IsSpread && node.TypeDisplay != "" {
			typeStr = " " + typeAnnotationStyle.Render(node.TypeDisplay)
		}

		lineW := lipgloss.Width(line + typeStr)
		if lineW <= w {
			line += typeStr
		}

		lines = append(lines, line)
	}

	// Add scroll indicators
	if start > 0 {
		lines = append([]string{dimStyle.Render("  " + upArrow + fmt.Sprintf(" %d more", start))}, lines...)
	}
	remaining := len(vis) - end
	if remaining > 0 {
		lines = append(lines, dimStyle.Render("  "+downArrow+fmt.Sprintf(" %d more", remaining)))
	}

	result := strings.Join(lines, "\n")
	if filterLine != "" {
		return filterLine + "\n" + result
	}
	return result
}

// --- Styles ---

var (
	builderBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Padding(0, 1)

	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("196"))

	// Cursor prefix matching schema browser's ▌ style
	selectedBarStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).SetString("▌ ")
	normalPrefixStyle = lipgloss.NewStyle().SetString("  ")

	selectedFieldStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Bold(true)

	normalFieldStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	dimFieldStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	dimStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	checkOnStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("42"))

	checkOffStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	typeAnnotationStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	hintStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	requiredStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)
)

// Hint key labels
const (
	upArrow    = "↑"
	downArrow  = "↓"
	leftArrow  = "←"
	rightArrow = "→"
	enterKey   = "↵"
)

// opBadgeFor returns a styled operation type badge.
func opBadgeFor(opType string) string {
	switch opType {
	case "query":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true).Render("query")
	case "mutation":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("162")).Bold(true).Render("mutation")
	case "subscription":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Bold(true).Render("subscription")
	default:
		return dimStyle.Render(opType)
	}
}

// --- Utilities ---

func scrollWindow(cursor, total, height int) (start, end int) {
	if total <= height {
		return 0, total
	}
	half := height / 2
	start = cursor - half
	if start < 0 {
		start = 0
	}
	end = start + height
	if end > total {
		end = total
		start = end - height
	}
	return start, end
}

func isRequired(ref schema.TypeRef) bool {
	return ref.Kind == "NON_NULL"
}

// padToHeight ensures content has exactly h lines (pads or truncates).
func padToHeight(content string, h int) string {
	if h <= 0 {
		return ""
	}
	lines := strings.Split(content, "\n")
	for len(lines) < h {
		lines = append(lines, "")
	}
	if len(lines) > h {
		lines = lines[:h]
	}
	return strings.Join(lines, "\n")
}

func rootTypeToOp(s *schema.Schema, typeName string) string {
	if s.QueryType != nil && s.QueryType.Name != nil && *s.QueryType.Name == typeName {
		return "query"
	}
	if s.MutationType != nil && s.MutationType.Name != nil && *s.MutationType.Name == typeName {
		return "mutation"
	}
	if s.SubscriptionType != nil && s.SubscriptionType.Name != nil && *s.SubscriptionType.Name == typeName {
		return "subscription"
	}
	return "query"
}
