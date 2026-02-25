package app

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/qraqula/qla/internal/config"
	"github.com/qraqula/qla/internal/graphql"
	"github.com/qraqula/qla/internal/history"
	"github.com/qraqula/qla/internal/schema"
)

func updateModel(m Model, msg tea.Msg) (Model, tea.Cmd) {
	tm, cmd := m.Update(msg)
	return tm.(Model), cmd
}

func TestNewModel(t *testing.T) {
	m := NewModel()
	if m.focus != PanelEditor {
		t.Errorf("expected initial focus on editor, got %v", m.focus)
	}
	if m.querying {
		t.Error("expected not querying initially")
	}
}

func TestFocusCycle(t *testing.T) {
	m := NewModel()
	// Simulate window size first
	m, _ = updateModel(m, tea.WindowSizeMsg{Width: 120, Height: 40})

	initial := m.focus
	m, _ = updateModel(m, tea.KeyPressMsg{Code: tea.KeyTab})
	if m.focus == initial {
		t.Error("expected focus to change after Tab")
	}
}

func TestQueryResult(t *testing.T) {
	m := NewModel()
	m, _ = updateModel(m, tea.WindowSizeMsg{Width: 120, Height: 40})

	m, _ = updateModel(m, QueryResultMsg{
		Result: &graphql.Result{
			StatusCode: 200,
			Duration:   142 * time.Millisecond,
			Size:       100,
		},
	})

	if m.querying {
		t.Error("expected querying to be false after result")
	}
	view := m.renderView()
	if !strings.Contains(view, "200") {
		t.Errorf("expected status code in view after result")
	}
}

func TestQueryError(t *testing.T) {
	m := NewModel()
	m, _ = updateModel(m, tea.WindowSizeMsg{Width: 120, Height: 40})

	m, _ = updateModel(m, QueryErrorMsg{Err: fmt.Errorf("connection refused")})
	if m.querying {
		t.Error("expected querying to be false after error")
	}
}

func TestToggleDocs(t *testing.T) {
	m := NewModel()
	m, _ = updateModel(m, tea.WindowSizeMsg{Width: 120, Height: 40})

	if m.rightPanelMode != modeResults {
		t.Errorf("expected initial right panel mode to be modeResults, got %v", m.rightPanelMode)
	}

	// Toggle to schema mode with ctrl+d
	m, _ = updateModel(m, tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl})
	if m.rightPanelMode != modeSchema {
		t.Errorf("expected right panel mode to be modeSchema after ctrl+d, got %v", m.rightPanelMode)
	}

	// Toggle back to results mode with ctrl+d
	m, _ = updateModel(m, tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl})
	if m.rightPanelMode != modeResults {
		t.Errorf("expected right panel mode to be modeResults after second ctrl+d, got %v", m.rightPanelMode)
	}
}

func TestSchemaFetchedMsg(t *testing.T) {
	m := NewModel()
	m, _ = updateModel(m, tea.WindowSizeMsg{Width: 120, Height: 40})

	name := "Query"
	s := &schema.Schema{
		QueryType: &schema.TypeRef{Name: &name},
		Types: []schema.FullType{
			{Kind: "OBJECT", Name: "Query"},
			{Kind: "SCALAR", Name: "String"},
		},
	}

	m, _ = updateModel(m, SchemaFetchedMsg{Schema: s})
	m.rightPanelMode = modeSchema

	view := m.renderView()
	if !strings.Contains(view, "Query") {
		t.Errorf("expected view to contain 'Query' after schema loaded, got:\n%s", view)
	}
}

func TestSchemaFetchErrorMsg(t *testing.T) {
	m := NewModel()
	m, _ = updateModel(m, tea.WindowSizeMsg{Width: 120, Height: 40})

	m, _ = updateModel(m, SchemaFetchErrorMsg{Err: fmt.Errorf("connection refused")})

	view := m.renderView()
	if !strings.Contains(view, "Schema fetch failed") {
		t.Errorf("expected view to contain 'Schema fetch failed', got:\n%s", view)
	}
}

func TestAutoFetchSchemaOnEndpointChange(t *testing.T) {
	m := newTestModel(t)

	// Focus on endpoint
	m.setFocus(PanelEndpoint)

	// Type an endpoint URL
	m.endpoint, _ = m.endpoint.Update(tea.KeyPressMsg{Code: 'h', Text: "h"})

	// Tab away from endpoint — should trigger auto-fetch (returns a cmd)
	m, cmd := updateModel(m, tea.KeyPressMsg{Code: tea.KeyTab})

	if m.lastEndpoint != "h" {
		t.Errorf("expected lastEndpoint to be 'h', got %q", m.lastEndpoint)
	}
	if cmd == nil {
		t.Error("expected a command to be returned for schema fetch")
	}
}

func TestEnterOnResultsOnlyExecutesInResultsMode(t *testing.T) {
	m := NewModel()
	m, _ = updateModel(m, tea.WindowSizeMsg{Width: 120, Height: 40})

	// Set focus to results panel
	m.setFocus(PanelResults)
	m.rightPanelMode = modeSchema

	// Enter should NOT execute query when in schema mode
	m, cmd := updateModel(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.querying {
		t.Error("expected enter not to execute query when in schema mode")
	}
	// The cmd should be nil or a browser update, not a query execution
	_ = cmd
}

func newTestModel(t *testing.T) Model {
	t.Helper()
	dir := t.TempDir()
	store := history.NewStore(dir)
	_ = store.Load()
	store.Meta.SidebarOpen = true
	cfgStore := config.NewStore(filepath.Join(dir, "config"))
	_ = cfgStore.Load()
	m := NewModelWithStores(store, cfgStore)
	m.sidebarOpen = true
	m, _ = updateModel(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	return m
}

func TestToggleSidebar(t *testing.T) {
	m := newTestModel(t)

	if !m.sidebarOpen {
		t.Error("expected sidebar open initially")
	}

	// Toggle sidebar with ctrl+b
	m, _ = updateModel(m, tea.KeyPressMsg{Code: 'b', Mod: tea.ModCtrl})
	if m.sidebarOpen {
		t.Error("expected sidebar closed after ctrl+b")
	}

	// Toggle back
	m, _ = updateModel(m, tea.KeyPressMsg{Code: 'b', Mod: tea.ModCtrl})
	if !m.sidebarOpen {
		t.Error("expected sidebar open after second ctrl+b")
	}
}

func TestToggleSidebarMovesFocusWhenClosing(t *testing.T) {
	m := newTestModel(t)

	// Focus on history
	m.setFocus(PanelHistory)
	if m.focus != PanelHistory {
		t.Fatal("expected focus on history")
	}

	// Close sidebar — should move focus to editor
	m, _ = updateModel(m, tea.KeyPressMsg{Code: 'b', Mod: tea.ModCtrl})
	if m.focus == PanelHistory {
		t.Error("expected focus to move away from history when sidebar closes")
	}
	if m.focus != PanelEditor {
		t.Errorf("expected focus on editor, got %v", m.focus)
	}
}

func TestSidebarFocusNavigation(t *testing.T) {
	m := newTestModel(t)

	// Add content so shouldShowSidebar() returns true
	_ = m.histStore.AddEntry(history.Entry{
		ID: history.GenerateID(), Name: "Test", Query: "{ test }",
		CreatedAt: time.Now(),
	})
	m.histSidebar.Rebuild()

	// From editor, ctrl+h should go to history when sidebar is open
	m.setFocus(PanelEditor)
	m, _ = updateModel(m, tea.KeyPressMsg{Code: 'h', Mod: tea.ModCtrl})
	if m.focus != PanelHistory {
		t.Errorf("expected focus on history after ctrl+h from editor, got %v", m.focus)
	}

	// Close sidebar and try again — should stay on editor
	m.setFocus(PanelEditor)
	m.sidebarOpen = false
	m, _ = updateModel(m, tea.KeyPressMsg{Code: 'h', Mod: tea.ModCtrl})
	if m.focus != PanelEditor {
		t.Errorf("expected focus to stay on editor when sidebar closed, got %v", m.focus)
	}
}

func TestAutoSaveOnQueryResult(t *testing.T) {
	m := newTestModel(t)

	// Set up editor with a query
	m.editor.SetValue("{ users { name } }")
	m.endpoint.SetValue("https://example.com/graphql")

	// Simulate query result
	m, _ = updateModel(m, QueryResultMsg{
		Result: &graphql.Result{
			Response: graphql.Response{Data: json.RawMessage(`{"users":[]}`)},
			StatusCode: 200,
			Duration:   100 * time.Millisecond,
			Size:       50,
		},
	})

	// Check that an entry was saved
	all := m.histStore.AllEntries()
	if len(all) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(all))
	}
	if all[0].Query != "{ users { name } }" {
		t.Errorf("expected query in history, got %q", all[0].Query)
	}
	if all[0].Endpoint != "https://example.com/graphql" {
		t.Errorf("expected endpoint in history, got %q", all[0].Endpoint)
	}
}

func TestAutoSaveDeduplicate(t *testing.T) {
	m := newTestModel(t)

	m.editor.SetValue("{ users { name } }")
	m.endpoint.SetValue("https://example.com/graphql")

	result := QueryResultMsg{
		Result: &graphql.Result{
			Response: graphql.Response{Data: json.RawMessage(`{"users":[]}`)},
			StatusCode: 200,
			Duration:   100 * time.Millisecond,
			Size:       50,
		},
	}

	// First query result
	m, _ = updateModel(m, result)

	// Second identical query result
	m, _ = updateModel(m, result)

	all := m.histStore.AllEntries()
	if len(all) != 1 {
		t.Errorf("expected 1 history entry (deduplicated), got %d", len(all))
	}
}

func TestLoadEntryMsg(t *testing.T) {
	m := newTestModel(t)

	entry := history.Entry{
		Query:     "{ posts { title } }",
		Variables: `{"limit": 10}`,
		Endpoint:  "https://api.example.com/graphql",
	}

	m, _ = updateModel(m, history.LoadEntryMsg{Entry: entry})

	if m.editor.Value() != entry.Query {
		t.Errorf("expected editor to have query %q, got %q", entry.Query, m.editor.Value())
	}
	if m.variables.Value() != entry.Variables {
		t.Errorf("expected variables to have %q, got %q", entry.Variables, m.variables.Value())
	}
	if m.endpoint.Value() != entry.Endpoint {
		t.Errorf("expected endpoint to have %q, got %q", entry.Endpoint, m.endpoint.Value())
	}
	if m.focus != PanelEditor {
		t.Errorf("expected focus on editor after load, got %v", m.focus)
	}
}

func TestToggleOverlay(t *testing.T) {
	m := newTestModel(t)

	if m.overlay.IsOpen() {
		t.Error("expected overlay closed initially")
	}

	// Open with ctrl+e
	m, _ = updateModel(m, tea.KeyPressMsg{Code: 'e', Mod: tea.ModCtrl})
	if !m.overlay.IsOpen() {
		t.Error("expected overlay open after ctrl+e")
	}

	// Close with ctrl+e
	m, _ = updateModel(m, tea.KeyPressMsg{Code: 'e', Mod: tea.ModCtrl})
	if m.overlay.IsOpen() {
		t.Error("expected overlay closed after second ctrl+e")
	}
}

func TestCycleEnv(t *testing.T) {
	m := newTestModel(t)

	// Add environments
	m.configStore.Config.Environments = []config.Environment{
		{Name: "dev", Endpoint: "https://dev.api.com"},
		{Name: "prod", Endpoint: "https://api.com"},
	}
	m.configStore.Config.ActiveEnv = ""

	// Cycle: none → dev
	m, _ = updateModel(m, tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl})
	if m.configStore.Config.ActiveEnv != "dev" {
		t.Errorf("expected ActiveEnv=dev, got %q", m.configStore.Config.ActiveEnv)
	}
	if m.endpoint.Value() != "https://dev.api.com" {
		t.Errorf("expected endpoint updated, got %q", m.endpoint.Value())
	}
	if m.endpoint.EnvName() != "dev" {
		t.Errorf("expected badge=dev, got %q", m.endpoint.EnvName())
	}

	// Cycle: dev → prod
	m, _ = updateModel(m, tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl})
	if m.configStore.Config.ActiveEnv != "prod" {
		t.Errorf("expected ActiveEnv=prod, got %q", m.configStore.Config.ActiveEnv)
	}

	// Cycle: prod → none (unset)
	m, _ = updateModel(m, tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl})
	if m.configStore.Config.ActiveEnv != "" {
		t.Errorf("expected ActiveEnv unset, got %q", m.configStore.Config.ActiveEnv)
	}
	if m.endpoint.EnvName() != "" {
		t.Errorf("expected empty badge, got %q", m.endpoint.EnvName())
	}
}

func TestCycleEnvNoEnvs(t *testing.T) {
	m := newTestModel(t)
	// No environments configured
	m, _ = updateModel(m, tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl})
	// Should not crash, just show error in statusbar
}

func TestOverlayInterceptsKeys(t *testing.T) {
	m := newTestModel(t)

	// Open overlay
	m, _ = updateModel(m, tea.KeyPressMsg{Code: 'e', Mod: tea.ModCtrl})
	if !m.overlay.IsOpen() {
		t.Fatal("expected overlay open")
	}

	// Press tab — should cycle overlay sections, not app panels
	prevFocus := m.focus
	m, _ = updateModel(m, tea.KeyPressMsg{Code: tea.KeyTab})
	if m.focus != prevFocus {
		t.Error("expected app focus unchanged when overlay is open")
	}
}

func TestSidebarTabCycleSkipsHistoryWhenClosed(t *testing.T) {
	m := newTestModel(t)
	m.sidebarOpen = false
	m.setFocus(PanelResults)

	// Tab from results when sidebar is closed should not land on history
	m, _ = updateModel(m, tea.KeyPressMsg{Code: tea.KeyTab})
	if m.focus == PanelHistory {
		t.Error("expected tab to skip history panel when sidebar is closed")
	}
}
