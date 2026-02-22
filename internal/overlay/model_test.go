package overlay

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/qraqula/qla/internal/config"
)

func keyMsg(k string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: -1, Text: k}
}

func testConfig() config.Config {
	return config.Config{
		ActiveEnv: "dev",
		Environments: []config.Environment{
			{
				Name:     "dev",
				Endpoint: "https://dev.api.com/graphql",
				Headers: []config.Header{
					{Key: "Authorization", Value: "Bearer dev", Enabled: true},
				},
			},
			{
				Name:     "prod",
				Endpoint: "https://api.com/graphql",
				Headers: []config.Header{
					{Key: "Authorization", Value: "Bearer prod", Enabled: true},
				},
			},
		},
		GlobalHeaders: []config.Header{
			{Key: "Accept", Value: "application/json", Enabled: true},
			{Key: "X-Debug", Value: "true", Enabled: false},
		},
	}
}

func TestOpenClose(t *testing.T) {
	m := New()
	if m.IsOpen() {
		t.Error("should be closed initially")
	}

	cfg := testConfig()
	m.Open(&cfg, 100, 40)
	if !m.IsOpen() {
		t.Error("should be open after Open()")
	}
	if m.section != SectionEnvs {
		t.Error("should start on SectionEnvs")
	}

	m.Close()
	if m.IsOpen() {
		t.Error("should be closed after Close()")
	}
}

func TestEscCloses(t *testing.T) {
	m := New()
	cfg := testConfig()
	m.Open(&cfg, 100, 40)

	var gotClose bool
	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(CloseMsg); ok {
			gotClose = true
		}
	}
	if !gotClose {
		t.Error("esc should return CloseMsg")
	}
}

func TestTabCyclesSections(t *testing.T) {
	m := New()
	cfg := testConfig()
	m.Open(&cfg, 100, 40)

	if m.section != SectionEnvs {
		t.Fatalf("expected SectionEnvs, got %d", m.section)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if m.section != SectionHeaders {
		t.Errorf("expected SectionHeaders after tab, got %d", m.section)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if m.section != SectionGlobal {
		t.Errorf("expected SectionGlobal after tab, got %d", m.section)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if m.section != SectionEnvs {
		t.Errorf("expected SectionEnvs after wrap, got %d", m.section)
	}
}

func TestEnvNavigation(t *testing.T) {
	m := New()
	cfg := testConfig()
	m.Open(&cfg, 100, 40)

	if m.envCursor != 0 {
		t.Fatalf("expected cursor 0, got %d", m.envCursor)
	}

	m, _ = m.Update(keyMsg("j"))
	if m.envCursor != 1 {
		t.Errorf("expected cursor 1, got %d", m.envCursor)
	}

	m, _ = m.Update(keyMsg("j"))
	if m.envCursor != 1 {
		t.Errorf("cursor should not exceed max, got %d", m.envCursor)
	}

	m, _ = m.Update(keyMsg("k"))
	if m.envCursor != 0 {
		t.Errorf("expected cursor 0, got %d", m.envCursor)
	}
}

func TestSelectEnvActivates(t *testing.T) {
	m := New()
	cfg := testConfig()
	cfg.ActiveEnv = ""
	m.Open(&cfg, 100, 40)

	// Select first env (dev)
	var gotChanged bool
	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd != nil {
		msg := cmd()
		if changed, ok := msg.(ConfigChangedMsg); ok {
			gotChanged = true
			if changed.Config.ActiveEnv != "dev" {
				t.Errorf("expected ActiveEnv=dev, got %q", changed.Config.ActiveEnv)
			}
		}
	}
	if !gotChanged {
		t.Error("enter on env should emit ConfigChangedMsg")
	}
}

func TestSelectEnvDeactivates(t *testing.T) {
	m := New()
	cfg := testConfig() // ActiveEnv = "dev"
	m.Open(&cfg, 100, 40)

	// Enter on already-active env should deactivate
	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd != nil {
		msg := cmd()
		if changed, ok := msg.(ConfigChangedMsg); ok {
			if changed.Config.ActiveEnv != "" {
				t.Errorf("expected ActiveEnv to be unset, got %q", changed.Config.ActiveEnv)
			}
		}
	}
}

func TestDeleteEnv(t *testing.T) {
	m := New()
	cfg := testConfig()
	m.Open(&cfg, 100, 40)

	m, cmd := m.Update(keyMsg("d"))
	if cmd == nil {
		t.Fatal("expected ConfigChangedMsg")
	}
	msg := cmd()
	changed, ok := msg.(ConfigChangedMsg)
	if !ok {
		t.Fatal("expected ConfigChangedMsg")
	}
	if len(changed.Config.Environments) != 1 {
		t.Errorf("expected 1 env after delete, got %d", len(changed.Config.Environments))
	}
	// Active env was dev (deleted), should be unset
	if changed.Config.ActiveEnv != "" {
		t.Errorf("expected unset active env after deleting active, got %q", changed.Config.ActiveEnv)
	}
}

func TestToggleGlobalHeader(t *testing.T) {
	m := New()
	cfg := testConfig()
	m.Open(&cfg, 100, 40)

	// Go to global section
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if m.section != SectionGlobal {
		t.Fatalf("expected SectionGlobal, got %d", m.section)
	}

	// First global header (Accept) is enabled
	if !cfg.GlobalHeaders[0].Enabled {
		t.Fatal("expected Accept enabled")
	}

	m, cmd := m.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	if cmd == nil {
		t.Fatal("expected ConfigChangedMsg")
	}
	msg := cmd()
	changed, ok := msg.(ConfigChangedMsg)
	if !ok {
		t.Fatal("expected ConfigChangedMsg")
	}
	if changed.Config.GlobalHeaders[0].Enabled {
		t.Error("expected Accept to be disabled after toggle")
	}
}

func TestDeleteHeader(t *testing.T) {
	m := New()
	cfg := testConfig()
	m.Open(&cfg, 100, 40)

	// Go to global headers
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})

	origCount := len(cfg.GlobalHeaders)
	m, cmd := m.Update(keyMsg("d"))
	if cmd == nil {
		t.Fatal("expected ConfigChangedMsg")
	}
	msg := cmd()
	changed, ok := msg.(ConfigChangedMsg)
	if !ok {
		t.Fatal("expected ConfigChangedMsg")
	}
	if len(changed.Config.GlobalHeaders) != origCount-1 {
		t.Errorf("expected %d headers after delete, got %d", origCount-1, len(changed.Config.GlobalHeaders))
	}
}

func TestHeaderColumnNavigation(t *testing.T) {
	m := New()
	cfg := testConfig()
	m.Open(&cfg, 100, 40)

	// Go to env headers
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if m.section != SectionHeaders {
		t.Fatalf("expected SectionHeaders, got %d", m.section)
	}

	if m.hdrCol != 0 {
		t.Errorf("expected col 0, got %d", m.hdrCol)
	}

	m, _ = m.Update(keyMsg("l"))
	if m.hdrCol != 1 {
		t.Errorf("expected col 1, got %d", m.hdrCol)
	}

	m, _ = m.Update(keyMsg("l"))
	if m.hdrCol != 1 {
		t.Errorf("should not exceed max col, got %d", m.hdrCol)
	}

	m, _ = m.Update(keyMsg("h"))
	if m.hdrCol != 0 {
		t.Errorf("expected col 0 after h, got %d", m.hdrCol)
	}
}

func TestViewRendersContent(t *testing.T) {
	m := New()
	cfg := testConfig()
	m.Open(&cfg, 100, 40)

	view := m.View()
	if view == "" {
		t.Error("expected non-empty view")
	}

	// Should contain section titles
	if !contains(view, "ENVIRONMENTS") {
		t.Error("expected ENVIRONMENTS in view")
	}
	if !contains(view, "HEADERS") {
		t.Error("expected HEADERS in view")
	}
	if !contains(view, "GLOBAL") {
		t.Error("expected GLOBAL in view")
	}

	// Should contain env names
	if !contains(view, "dev") {
		t.Error("expected 'dev' in view")
	}
	if !contains(view, "prod") {
		t.Error("expected 'prod' in view")
	}
}

func TestRenderOverComposites(t *testing.T) {
	m := New()
	cfg := testConfig()
	m.Open(&cfg, 100, 40)

	bg := "background content"
	result := m.RenderOver(bg)
	if result == bg {
		t.Error("expected overlay to change the output")
	}
	if !contains(result, "ENVIRONMENTS") {
		t.Error("expected overlay content in composited view")
	}
}

func TestRenderOverPassthroughWhenClosed(t *testing.T) {
	m := New()
	bg := "background content"
	result := m.RenderOver(bg)
	if result != bg {
		t.Error("expected passthrough when overlay is closed")
	}
}

func TestCreateEnvFlow(t *testing.T) {
	m := New()
	cfg := config.Config{}
	m.Open(&cfg, 100, 40)

	// Press n to create
	m, _ = m.Update(keyMsg("n"))
	if m.mode != ModeCreateEnv {
		t.Fatalf("expected ModeCreateEnv, got %d", m.mode)
	}

	// Type name
	m.input.SetValue("staging")

	// Confirm
	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != ModeNormal {
		t.Errorf("expected ModeNormal after confirm, got %d", m.mode)
	}
	if cmd == nil {
		t.Fatal("expected ConfigChangedMsg")
	}
	msg := cmd()
	changed, ok := msg.(ConfigChangedMsg)
	if !ok {
		t.Fatal("expected ConfigChangedMsg")
	}
	if len(changed.Config.Environments) != 1 {
		t.Errorf("expected 1 env, got %d", len(changed.Config.Environments))
	}
	if changed.Config.Environments[0].Name != "staging" {
		t.Errorf("expected name=staging, got %q", changed.Config.Environments[0].Name)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchString(s, sub)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
