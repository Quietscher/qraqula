package variables

import (
	"testing"
)

func TestNew(t *testing.T) {
	m := New()
	if m.Value() != "" {
		t.Errorf("expected empty value, got %q", m.Value())
	}
}

func TestParsedVariables(t *testing.T) {
	m := New()
	m.SetValue(`{"key": "value"}`)
	vars, err := m.ParsedVariables()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vars["key"] != "value" {
		t.Errorf("expected key=value, got %v", vars["key"])
	}
}

func TestParsedVariablesEmpty(t *testing.T) {
	m := New()
	vars, err := m.ParsedVariables()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vars != nil {
		t.Errorf("expected nil vars for empty input, got %v", vars)
	}
}

func TestParsedVariablesInvalid(t *testing.T) {
	m := New()
	m.SetValue(`{invalid}`)
	_, err := m.ParsedVariables()
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
