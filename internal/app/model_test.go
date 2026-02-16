package app

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/qraqula/qla/internal/graphql"
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
