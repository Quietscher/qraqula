package editor

import (
	"testing"
)

func TestNew(t *testing.T) {
	m := New()
	if m.Value() != "" {
		t.Errorf("expected empty value, got %q", m.Value())
	}
}

func TestEditing(t *testing.T) {
	m := New()
	if m.Editing() {
		t.Error("expected not editing initially")
	}
	m.StartEditing()
	if !m.Editing() {
		t.Error("expected editing after StartEditing")
	}
	m.StopEditing()
	if m.Editing() {
		t.Error("expected not editing after StopEditing")
	}
}

func TestBlurStopsEditing(t *testing.T) {
	m := New()
	m.StartEditing()
	m.Blur()
	if m.Editing() {
		t.Error("expected Blur to stop editing")
	}
}

func TestSetSize(t *testing.T) {
	m := New()
	m.SetSize(40, 10)
	// No panic means success — textarea handles sizing internally
}

func TestViewEmptyShowsPlaceholder(t *testing.T) {
	m := New()
	m.SetSize(80, 20)
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view even without content")
	}
}
