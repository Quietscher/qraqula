package endpoint

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNew(t *testing.T) {
	m := New()
	if m.Value() != "" {
		t.Errorf("expected empty value, got %q", m.Value())
	}
}

func TestSetFocused(t *testing.T) {
	m := New()
	cmd := m.Focus()
	if cmd == nil {
		// cmd may or may not be nil depending on textinput impl
	}
	if !m.Focused() {
		t.Error("expected focused after Focus()")
	}
	m.Blur()
	if m.Focused() {
		t.Error("expected not focused after Blur()")
	}
}

func TestViewContainsEndpoint(t *testing.T) {
	m := New()
	m.SetWidth(80)
	view := m.View()
	if !strings.Contains(view, "Endpoint") {
		t.Errorf("expected view to contain 'Endpoint', got %q", view)
	}
}

func TestUpdate(t *testing.T) {
	m := New()
	m.Focus()
	// Type a URL character by character
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = updated
	if m.Value() != "h" {
		t.Errorf("expected 'h', got %q", m.Value())
	}
}
