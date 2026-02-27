package overlay

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/qraqula/qla/internal/config"
)

// Messages returned to the parent app.
type CloseMsg struct{}
type ConfigChangedMsg struct{ Config config.Config }

// Section of the overlay.
type Section int

const (
	SectionEnvs    Section = iota // environment list
	SectionHeaders                // active env headers
	SectionGlobal                 // global headers
	sectionCount
)

// Mode within the overlay.
type Mode int

const (
	ModeNormal    Mode = iota
	ModeEditKey         // editing a header key
	ModeEditValue       // editing a header value
	ModeCreateEnv       // creating a new environment name
	ModeRenameEnv       // renaming an environment
	ModeEditEnvEndpoint // editing environment endpoint
	ModeEditEnvVars     // editing environment variables
)

// Vampire theme colors (matching app).
var (
	overlayBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("196")).
			Padding(1, 2)

	sectionTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("245"))

	activeSectionTitle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("196"))

	sepLine = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	enabledStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	disabledStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	hintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	activeEnvMarker = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
)

type Model struct {
	config  *config.Config
	width   int
	height  int
	visible bool

	section   Section
	mode      Mode
	envCursor int
	hdrCursor int
	hdrCol    int // 0=key, 1=value

	input textinput.Model
}

func New() Model {
	ti := textinput.New()
	ti.CharLimit = 500
	return Model{input: ti}
}

func (m *Model) Open(cfg *config.Config, w, h int) {
	m.config = cfg
	m.width = w
	m.height = h
	m.visible = true
	m.section = SectionEnvs
	m.mode = ModeNormal
	m.envCursor = 0
	m.hdrCursor = 0
	m.hdrCol = 0
	m.input.SetValue("")
	m.input.Blur()
	m.clampCursors()
}

func (m *Model) Close() {
	m.visible = false
	m.mode = ModeNormal
}

func (m Model) IsOpen() bool {
	return m.visible
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	if m.mode != ModeNormal {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	// When editing, delegate to input
	if m.mode != ModeNormal {
		return m.handleEditKey(msg)
	}

	switch msg.String() {
	case "esc":
		m.Close()
		return m, func() tea.Msg { return CloseMsg{} }

	case "tab":
		m.section = (m.section + 1) % sectionCount
		m.hdrCursor = 0
		m.hdrCol = 0
		m.clampCursors()
		return m, nil

	case "shift+tab":
		m.section = (m.section - 1 + sectionCount) % sectionCount
		m.hdrCursor = 0
		m.hdrCol = 0
		m.clampCursors()
		return m, nil

	case "j", "down":
		m.moveCursorDown()
		return m, nil

	case "k", "up":
		m.moveCursorUp()
		return m, nil

	case "l", "right":
		if m.section != SectionEnvs && m.hdrCol < 1 {
			m.hdrCol++
		}
		return m, nil

	case "h", "left":
		if m.section != SectionEnvs && m.hdrCol > 0 {
			m.hdrCol--
		}
		return m, nil

	case "enter":
		return m.handleEnter()

	case "n":
		return m.handleNew()

	case "a":
		if m.section != SectionEnvs {
			return m.handleAddHeader()
		}
		return m, nil

	case "r":
		if m.section == SectionEnvs {
			return m.startRenameEnv()
		}
		return m, nil

	case "d":
		return m.handleDelete()

	case "space", " ":
		if m.section != SectionEnvs {
			return m.toggleHeader()
		}
		return m, nil

	case "e":
		if m.section == SectionEnvs {
			return m.startEditEnvEndpoint()
		}
		return m, nil

	case "v":
		if m.section == SectionEnvs {
			return m.startEditEnvVars()
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handleEditKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		return m.confirmEdit()
	case "esc":
		m.mode = ModeNormal
		m.input.Blur()
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) handleEnter() (Model, tea.Cmd) {
	switch m.section {
	case SectionEnvs:
		// Select/activate environment
		if m.config == nil || len(m.config.Environments) == 0 {
			return m, nil
		}
		if m.envCursor < len(m.config.Environments) {
			env := m.config.Environments[m.envCursor]
			if m.config.ActiveEnv == env.Name {
				m.config.ActiveEnv = "" // deselect
			} else {
				m.config.ActiveEnv = env.Name
			}
			return m, m.emitChanged()
		}

	case SectionHeaders, SectionGlobal:
		// Edit header key or value
		hdrs := m.currentHeaders()
		if hdrs == nil || m.hdrCursor >= len(*hdrs) {
			return m, nil
		}
		h := (*hdrs)[m.hdrCursor]
		if m.hdrCol == 0 {
			m.mode = ModeEditKey
			m.input.SetValue(h.Key)
		} else {
			m.mode = ModeEditValue
			m.input.SetValue(h.Value)
		}
		m.setInputWidth()
		cmd := m.input.Focus()
		m.input.CursorEnd()
		return m, cmd
	}
	return m, nil
}

func (m Model) handleNew() (Model, tea.Cmd) {
	if m.config == nil {
		return m, nil
	}
	switch m.section {
	case SectionEnvs:
		m.mode = ModeCreateEnv
		m.input.SetValue("")
		m.input.Placeholder = "name"
		m.setInputWidth()
		cmd := m.input.Focus()
		return m, cmd

	case SectionHeaders, SectionGlobal:
		return m.handleAddHeader()
	}
	return m, nil
}

func (m Model) handleAddHeader() (Model, tea.Cmd) {
	if m.config == nil {
		return m, nil
	}
	newH := config.Header{Key: "", Value: "", Enabled: true}
	switch m.section {
	case SectionHeaders:
		env := m.config.ActiveEnvironment()
		if env == nil {
			return m, nil
		}
		env.Headers = append(env.Headers, newH)
		m.hdrCursor = len(env.Headers) - 1
	case SectionGlobal:
		m.config.GlobalHeaders = append(m.config.GlobalHeaders, newH)
		m.hdrCursor = len(m.config.GlobalHeaders) - 1
	}
	m.hdrCol = 0
	// Start editing the key immediately
	m.mode = ModeEditKey
	m.input.SetValue("")
	m.input.Placeholder = "header key"
	m.setInputWidth()
	cmd := m.input.Focus()
	return m, cmd
}

func (m Model) handleDelete() (Model, tea.Cmd) {
	if m.config == nil {
		return m, nil
	}
	switch m.section {
	case SectionEnvs:
		if len(m.config.Environments) == 0 {
			return m, nil
		}
		if m.envCursor >= len(m.config.Environments) {
			return m, nil
		}
		name := m.config.Environments[m.envCursor].Name
		m.config.Environments = append(m.config.Environments[:m.envCursor], m.config.Environments[m.envCursor+1:]...)
		if m.config.ActiveEnv == name {
			m.config.ActiveEnv = ""
		}
		m.clampCursors()
		return m, m.emitChanged()

	case SectionHeaders:
		env := m.config.ActiveEnvironment()
		if env == nil || m.hdrCursor >= len(env.Headers) {
			return m, nil
		}
		env.Headers = append(env.Headers[:m.hdrCursor], env.Headers[m.hdrCursor+1:]...)
		m.clampCursors()
		return m, m.emitChanged()

	case SectionGlobal:
		if m.hdrCursor >= len(m.config.GlobalHeaders) {
			return m, nil
		}
		m.config.GlobalHeaders = append(m.config.GlobalHeaders[:m.hdrCursor], m.config.GlobalHeaders[m.hdrCursor+1:]...)
		m.clampCursors()
		return m, m.emitChanged()
	}
	return m, nil
}

func (m Model) toggleHeader() (Model, tea.Cmd) {
	hdrs := m.currentHeaders()
	if hdrs == nil || m.hdrCursor >= len(*hdrs) {
		return m, nil
	}
	(*hdrs)[m.hdrCursor].Enabled = !(*hdrs)[m.hdrCursor].Enabled
	return m, m.emitChanged()
}

func (m Model) startRenameEnv() (Model, tea.Cmd) {
	if m.config == nil || len(m.config.Environments) == 0 || m.envCursor >= len(m.config.Environments) {
		return m, nil
	}
	m.mode = ModeRenameEnv
	m.input.SetValue(m.config.Environments[m.envCursor].Name)
	m.setInputWidth()
	cmd := m.input.Focus()
	m.input.CursorEnd()
	return m, cmd
}

func (m Model) startEditEnvEndpoint() (Model, tea.Cmd) {
	if m.config == nil || len(m.config.Environments) == 0 || m.envCursor >= len(m.config.Environments) {
		return m, nil
	}
	m.mode = ModeEditEnvEndpoint
	m.input.SetValue(m.config.Environments[m.envCursor].Endpoint)
	m.input.Placeholder = "https://api.example.com/graphql"
	m.setInputWidth()
	cmd := m.input.Focus()
	m.input.CursorEnd()
	return m, cmd
}

func (m Model) startEditEnvVars() (Model, tea.Cmd) {
	if m.config == nil || len(m.config.Environments) == 0 || m.envCursor >= len(m.config.Environments) {
		return m, nil
	}
	m.mode = ModeEditEnvVars
	m.input.SetValue(m.config.Environments[m.envCursor].Variables)
	m.input.Placeholder = `{"key": "value"}`
	m.setInputWidth()
	cmd := m.input.Focus()
	m.input.CursorEnd()
	return m, cmd
}

func (m Model) confirmEdit() (Model, tea.Cmd) {
	val := m.input.Value()
	m.input.Blur()

	switch m.mode {
	case ModeCreateEnv:
		m.mode = ModeNormal
		if val == "" {
			return m, nil
		}
		m.config.Environments = append(m.config.Environments, config.Environment{Name: val})
		m.envCursor = len(m.config.Environments) - 1
		return m, m.emitChanged()

	case ModeRenameEnv:
		m.mode = ModeNormal
		if val == "" || m.envCursor >= len(m.config.Environments) {
			return m, nil
		}
		old := m.config.Environments[m.envCursor].Name
		m.config.Environments[m.envCursor].Name = val
		if m.config.ActiveEnv == old {
			m.config.ActiveEnv = val
		}
		return m, m.emitChanged()

	case ModeEditEnvEndpoint:
		m.mode = ModeNormal
		if m.envCursor >= len(m.config.Environments) {
			return m, nil
		}
		m.config.Environments[m.envCursor].Endpoint = val
		return m, m.emitChanged()

	case ModeEditEnvVars:
		m.mode = ModeNormal
		if m.envCursor >= len(m.config.Environments) {
			return m, nil
		}
		m.config.Environments[m.envCursor].Variables = val
		return m, m.emitChanged()

	case ModeEditKey:
		hdrs := m.currentHeaders()
		if hdrs == nil || m.hdrCursor >= len(*hdrs) {
			m.mode = ModeNormal
			return m, nil
		}
		(*hdrs)[m.hdrCursor].Key = val
		// Auto-advance to value editing
		m.mode = ModeEditValue
		m.input.SetValue("")
		m.input.Placeholder = "header value"
		m.setInputWidth()
		focusCmd := m.input.Focus()
		return m, tea.Batch(m.emitChanged(), focusCmd)

	case ModeEditValue:
		m.mode = ModeNormal
		hdrs := m.currentHeaders()
		if hdrs == nil || m.hdrCursor >= len(*hdrs) {
			return m, nil
		}
		(*hdrs)[m.hdrCursor].Value = val
		return m, m.emitChanged()
	}

	m.mode = ModeNormal
	return m, nil
}

func (m *Model) currentHeaders() *[]config.Header {
	if m.config == nil {
		return nil
	}
	switch m.section {
	case SectionHeaders:
		env := m.config.ActiveEnvironment()
		if env == nil {
			return nil
		}
		return &env.Headers
	case SectionGlobal:
		return &m.config.GlobalHeaders
	}
	return nil
}

func (m Model) emitChanged() tea.Cmd {
	cfg := *m.config
	return func() tea.Msg { return ConfigChangedMsg{Config: cfg} }
}

func (m *Model) clampCursors() {
	if m.config == nil {
		return
	}
	if m.envCursor >= len(m.config.Environments) {
		m.envCursor = max(0, len(m.config.Environments)-1)
	}
	hdrs := m.currentHeaders()
	if hdrs != nil {
		if m.hdrCursor >= len(*hdrs) {
			m.hdrCursor = max(0, len(*hdrs)-1)
		}
	} else {
		m.hdrCursor = 0
	}
}

func (m *Model) moveCursorDown() {
	switch m.section {
	case SectionEnvs:
		if m.config != nil && m.envCursor < len(m.config.Environments)-1 {
			m.envCursor++
		}
	default:
		hdrs := m.currentHeaders()
		if hdrs != nil && m.hdrCursor < len(*hdrs)-1 {
			m.hdrCursor++
		}
	}
}

func (m *Model) moveCursorUp() {
	switch m.section {
	case SectionEnvs:
		if m.envCursor > 0 {
			m.envCursor--
		}
	default:
		if m.hdrCursor > 0 {
			m.hdrCursor--
		}
	}
}

// --- View ---

func (m Model) View() string {
	if !m.visible || m.config == nil {
		return ""
	}
	return m.renderContent()
}

func (m Model) RenderOver(background string) string {
	if !m.visible {
		return background
	}

	overlayW := m.width * 4 / 5
	if overlayW < 40 {
		overlayW = 40
	}
	overlayH := m.height * 7 / 10
	if overlayH < 15 {
		overlayH = 15
	}

	content := m.renderContent()
	box := overlayBorder.
		Width(overlayW - 4). // account for border + padding
		Height(overlayH - 4).
		Render(content)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceChars(" "),
	)
}

func (m Model) renderContent() string {
	var sections []string

	// Title
	sections = append(sections, m.renderEnvSection())
	sections = append(sections, "")
	sections = append(sections, m.renderHeaderSection())
	sections = append(sections, "")
	sections = append(sections, m.renderGlobalSection())

	// Input line if editing
	if m.mode != ModeNormal {
		sections = append(sections, "")
		sections = append(sections, m.renderEditLine())
	}

	// Hints
	sections = append(sections, "")
	sections = append(sections, m.renderHints())

	return strings.Join(sections, "\n")
}

func (m Model) renderEnvSection() string {
	var lines []string
	cw := m.contentWidth()

	title := "ENVIRONMENTS"
	if m.section == SectionEnvs {
		lines = append(lines, activeSectionTitle.Render(title))
	} else {
		lines = append(lines, sectionTitle.Render(title))
	}
	lines = append(lines, sepLine.Render(strings.Repeat("─", cw)))

	if m.config == nil || len(m.config.Environments) == 0 {
		lines = append(lines, dimStyle.Render("  (no environments)"))
		return strings.Join(lines, "\n")
	}

	for i, env := range m.config.Environments {
		var marker string
		if env.Name == m.config.ActiveEnv {
			marker = activeEnvMarker.Render("● ")
		} else {
			marker = "  "
		}

		name := env.Name
		// Truncate endpoint to fill remaining width after marker + name + gap
		epMax := cw - lipgloss.Width(marker) - lipgloss.Width(name) - 2
		if epMax < 10 {
			epMax = 10
		}
		ep := dimStyle.Render(truncate(env.Endpoint, epMax))

		if m.section == SectionEnvs && i == m.envCursor {
			line := marker + selectedStyle.Render(name) + "  " + ep
			lines = append(lines, line)
		} else {
			line := marker + normalStyle.Render(name) + "  " + ep
			lines = append(lines, line)
		}
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderHeaderSection() string {
	var lines []string
	cw := m.contentWidth()

	envName := "(none)"
	if m.config.ActiveEnv != "" {
		envName = m.config.ActiveEnv
	}
	title := fmt.Sprintf("HEADERS (%s)", envName)
	if m.section == SectionHeaders {
		lines = append(lines, activeSectionTitle.Render(title))
	} else {
		lines = append(lines, sectionTitle.Render(title))
	}
	lines = append(lines, sepLine.Render(strings.Repeat("─", cw)))

	env := m.config.ActiveEnvironment()
	if env == nil {
		lines = append(lines, dimStyle.Render("  (select an environment first)"))
		return strings.Join(lines, "\n")
	}

	if len(env.Headers) == 0 {
		lines = append(lines, dimStyle.Render("  (no headers)"))
		return strings.Join(lines, "\n")
	}

	for i, h := range env.Headers {
		lines = append(lines, m.renderHeaderRow(i, h, m.section == SectionHeaders))
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderGlobalSection() string {
	var lines []string
	cw := m.contentWidth()

	title := "GLOBAL HEADERS"
	if m.section == SectionGlobal {
		lines = append(lines, activeSectionTitle.Render(title))
	} else {
		lines = append(lines, sectionTitle.Render(title))
	}
	lines = append(lines, sepLine.Render(strings.Repeat("─", cw)))

	if len(m.config.GlobalHeaders) == 0 {
		lines = append(lines, dimStyle.Render("  (no global headers)"))
		return strings.Join(lines, "\n")
	}

	for i, h := range m.config.GlobalHeaders {
		lines = append(lines, m.renderHeaderRow(i, h, m.section == SectionGlobal))
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderHeaderRow(idx int, h config.Header, isActiveSection bool) string {
	var check string
	if h.Enabled {
		check = enabledStyle.Render("[✓]")
	} else {
		check = disabledStyle.Render("[ ]")
	}

	key := h.Key
	val := h.Value

	// Dynamic truncation: overhead is "  " + check + " " + key + "  " = ~8 + keyWidth
	cw := m.contentWidth()
	valMax := cw - 8 - lipgloss.Width(key)
	if valMax < 10 {
		valMax = 10
	}

	isSelected := isActiveSection && idx == m.hdrCursor
	if isSelected {
		if m.hdrCol == 0 {
			key = selectedStyle.Render(key)
			val = dimStyle.Render(truncate(val, valMax))
		} else {
			key = normalStyle.Render(key)
			val = selectedStyle.Render(truncate(val, valMax))
		}
		return fmt.Sprintf("  %s %s  %s", check, key, val)
	}

	return fmt.Sprintf("  %s %s  %s", check, normalStyle.Render(key), dimStyle.Render(truncate(val, valMax)))
}

func (m Model) renderEditLine() string {
	label := m.editLabel()
	return sectionTitle.Render(label) + m.input.View()
}

// editLabel returns the label for the current edit mode.
func (m Model) editLabel() string {
	switch m.mode {
	case ModeCreateEnv:
		return "New environment: "
	case ModeRenameEnv:
		return "Rename: "
	case ModeEditEnvEndpoint:
		return "Endpoint: "
	case ModeEditEnvVars:
		return "Variables: "
	case ModeEditKey:
		return "Key: "
	case ModeEditValue:
		return "Value: "
	}
	return ""
}

// setInputWidth sizes the text input to fill the available overlay width.
func (m *Model) setInputWidth() {
	cw := m.contentWidth()
	w := cw - lipgloss.Width(m.editLabel())
	if w < 10 {
		w = 10
	}
	m.input.SetWidth(w)
}

func (m Model) renderHints() string {
	var hints []string
	switch m.section {
	case SectionEnvs:
		hints = []string{"tab section", "j/k nav", "↵ select", "n new", "r rename", "e endpoint", "v vars", "d del", "esc close"}
	default:
		hints = []string{"tab section", "j/k nav", "h/l col", "↵ edit", "a/n add", "d del", "space toggle", "esc close"}
	}
	return hintStyle.Render(strings.Join(hints, "  "))
}

// contentWidth returns the usable text width inside the overlay.
func (m Model) contentWidth() int {
	overlayW := m.width * 4 / 5
	if overlayW < 40 {
		overlayW = 40
	}
	// Width(overlayW-4) sets outer width; border takes 2, padding(1,2) takes 4 horizontal
	cw := overlayW - 10
	if cw < 20 {
		cw = 20
	}
	return cw
}

func truncate(s string, maxW int) string {
	if lipgloss.Width(s) <= maxW {
		return s
	}
	runes := []rune(s)
	for i := len(runes) - 1; i >= 0; i-- {
		candidate := string(runes[:i]) + "…"
		if lipgloss.Width(candidate) <= maxW {
			return candidate
		}
	}
	return "…"
}
