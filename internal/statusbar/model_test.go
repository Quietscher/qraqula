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
