package app

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/qraqula/qla/internal/graphql"
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
