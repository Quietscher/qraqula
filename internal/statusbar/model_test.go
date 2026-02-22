package statusbar

import (
	"strings"
	"testing"
	"time"
)

func TestNewEmpty(t *testing.T) {
	m := New()
	view := m.View()
	if !strings.Contains(view, "Ready") {
		t.Errorf("expected 'Ready' in empty status bar, got %q", view)
	}
}

func TestSetResult(t *testing.T) {
	m := New()
	m.SetResult(200, 142*time.Millisecond, 3200, false)
	view := m.View()
	if !strings.Contains(view, "200") {
		t.Errorf("expected '200' in view, got %q", view)
	}
	if !strings.Contains(view, "142ms") {
		t.Errorf("expected '142ms' in view, got %q", view)
	}
}

func TestSetResultWithErrors(t *testing.T) {
	m := New()
	m.SetResult(200, 100*time.Millisecond, 500, true)
	view := m.View()
	if !strings.Contains(view, "with errors") {
		t.Errorf("expected 'with errors' in view, got %q", view)
	}
}

func TestSetError(t *testing.T) {
	m := New()
	m.SetError("connection refused")
	view := m.View()
	if !strings.Contains(view, "connection refused") {
		t.Errorf("expected error message in view, got %q", view)
	}
}

func TestSetLoading(t *testing.T) {
	m := New()
	m.SetLoading()
	view := m.View()
	if !strings.Contains(view, "Executing") {
		t.Errorf("expected 'Executing' in view, got %q", view)
	}
}

func TestSetHintsCustom(t *testing.T) {
	m := New()
	m.SetWidth(120)
	m.SetHints([]Hint{
		{Key: "j/k", Label: "navigate"},
		{Key: "l/↵", Label: "drill in"},
		{Key: "^q", Label: "quit"},
	})
	view := m.View()
	if !strings.Contains(view, "navigate") {
		t.Errorf("expected custom hint 'navigate' in view, got %q", view)
	}
	if !strings.Contains(view, "drill in") {
		t.Errorf("expected custom hint 'drill in' in view, got %q", view)
	}
	// Default hints should NOT appear
	if strings.Contains(view, "execute") {
		t.Errorf("expected default hint 'execute' to be replaced, got %q", view)
	}
}

func TestSetHintsEmpty(t *testing.T) {
	m := New()
	m.SetWidth(120)
	// No SetHints — should fall back to defaults
	view := m.View()
	if !strings.Contains(view, "execute") {
		t.Errorf("expected default hint 'execute' in view, got %q", view)
	}
}
