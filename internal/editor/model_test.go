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

func TestFocus(t *testing.T) {
	m := New()
	m.Focus()
	if !m.Focused() {
		t.Error("expected focused")
	}
	m.Blur()
	if m.Focused() {
		t.Error("expected not focused")
	}
}

func TestSetSize(t *testing.T) {
	m := New()
	m.SetSize(40, 10)
	// No panic means success — textarea handles sizing internally
}
