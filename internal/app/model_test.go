package app

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/qraqula/qla/internal/graphql"
)

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
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	initial := m.focus
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.focus == initial {
		t.Error("expected focus to change after Tab")
	}
}

func TestQueryResult(t *testing.T) {
	m := NewModel()
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	m, _ = m.Update(QueryResultMsg{
		Result: &graphql.Result{
			StatusCode: 200,
			Duration:   142 * time.Millisecond,
			Size:       100,
		},
	})

	if m.querying {
		t.Error("expected querying to be false after result")
	}
	view := m.View()
	if !strings.Contains(view, "200") {
		t.Errorf("expected status code in view after result")
	}
}

func TestQueryError(t *testing.T) {
	m := NewModel()
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	m, _ = m.Update(QueryErrorMsg{Err: fmt.Errorf("connection refused")})
	if m.querying {
		t.Error("expected querying to be false after error")
	}
}
