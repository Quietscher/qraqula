package results

import (
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	m := New(80, 20)
	view := m.View()
	if !strings.Contains(view, "Result") {
		t.Errorf("expected view to contain 'Result', got %q", view)
	}
}

func TestSetContent(t *testing.T) {
	m := New(80, 20)
	m.SetContent(`{"hello":"world"}`)
	view := m.View()
	if !strings.Contains(view, "hello") {
		t.Errorf("expected view to contain 'hello', got %q", view)
	}
}

func TestSetPrettyJSON(t *testing.T) {
	m := New(80, 20)
	err := m.SetPrettyJSON([]byte(`{"hello":"world"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	view := m.View()
	if !strings.Contains(view, "hello") {
		t.Errorf("expected pretty JSON in view")
	}
}

func TestSetPrettyJSONInvalid(t *testing.T) {
	m := New(80, 20)
	err := m.SetPrettyJSON([]byte(`{invalid}`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
