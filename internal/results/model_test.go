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

func TestSetPrettyJSONHighlighted(t *testing.T) {
	m := New(80, 20)
	data := []byte(`{"key":"value","num":42}`)
	err := m.SetPrettyJSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	view := m.View()
	// Split off the title line; check the content area has ANSI codes
	lines := strings.SplitN(view, "\n", 2)
	if len(lines) < 2 {
		t.Fatal("expected at least two lines in view")
	}
	content := lines[1]
	if !strings.Contains(content, "\x1b[") {
		t.Error("expected syntax-highlighted output with ANSI codes in content area")
	}
}

func TestSearchToggle(t *testing.T) {
	m := New(80, 20)
	m.SetContent("line one\nline two\nline three")

	if m.Searching() {
		t.Error("should not be searching initially")
	}

	m.ToggleSearch()
	if !m.Searching() {
		t.Error("should be searching after toggle")
	}

	m.ToggleSearch()
	if m.Searching() {
		t.Error("should stop searching after second toggle")
	}
}

func TestSearchFindMatches(t *testing.T) {
	m := New(80, 20)
	m.SetContent("apple banana apple cherry apple")

	m.ToggleSearch()
	m.SetSearchQuery("apple")

	if m.MatchCount() != 3 {
		t.Errorf("expected 3 matches, got %d", m.MatchCount())
	}
	if m.CurrentMatch() != 0 {
		t.Errorf("expected match index 0, got %d", m.CurrentMatch())
	}
}

func TestSearchNextPrev(t *testing.T) {
	m := New(80, 20)
	m.SetContent("aa\nbb\naa\ncc\naa")

	m.ToggleSearch()
	m.SetSearchQuery("aa")

	if m.MatchCount() != 3 {
		t.Fatalf("expected 3 matches, got %d", m.MatchCount())
	}

	m.NextMatch()
	if m.CurrentMatch() != 1 {
		t.Errorf("expected match 1, got %d", m.CurrentMatch())
	}

	m.NextMatch()
	if m.CurrentMatch() != 2 {
		t.Errorf("expected match 2, got %d", m.CurrentMatch())
	}

	m.NextMatch() // wraps
	if m.CurrentMatch() != 0 {
		t.Errorf("expected wrap to 0, got %d", m.CurrentMatch())
	}

	m.PrevMatch()
	if m.CurrentMatch() != 2 {
		t.Errorf("expected wrap to 2, got %d", m.CurrentMatch())
	}
}

func TestSearchNoMatches(t *testing.T) {
	m := New(80, 20)
	m.SetContent("hello world")

	m.ToggleSearch()
	m.SetSearchQuery("xyz")

	if m.MatchCount() != 0 {
		t.Errorf("expected 0 matches, got %d", m.MatchCount())
	}
}

func TestInsertMatchHighlights(t *testing.T) {
	// Plain text, no ANSI codes
	line := "hello world hello"
	ranges := [][2]int{{0, 5}, {12, 17}} // both "hello"
	result := insertMatchHighlights(line, ranges)
	expected := matchBgOn + "hello" + matchBgOff + " world " + matchBgOn + "hello" + matchBgOff
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestInsertMatchHighlightsWithANSI(t *testing.T) {
	// Simulated ANSI-coded text: "\x1b[31m" + "he" + "\x1b[0m" + "llo"
	line := "\x1b[31mhe\x1b[0mllo world"
	ranges := [][2]int{{0, 5}} // "hello" spans across the ANSI boundary
	result := insertMatchHighlights(line, ranges)
	// ANSI code comes first in string, then matchBgOn at visPos=0, re-emit after reset
	expected := "\x1b[31m" + matchBgOn + "he" + "\x1b[0m" + matchBgOn + "llo" + matchBgOff + " world"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestSearchHighlightsAppliedToViewport(t *testing.T) {
	m := New(80, 20)
	m.SetContent("apple banana apple")

	m.ToggleSearch()
	m.SetSearchQuery("apple")

	if m.MatchCount() != 2 {
		t.Fatalf("expected 2 matches, got %d", m.MatchCount())
	}

	// The viewport content should contain the yellow background ANSI code
	view := m.View()
	if !strings.Contains(view, matchBgOn) {
		t.Error("expected search highlights in viewport content")
	}
}

func TestSearchHighlightsRemovedOnClose(t *testing.T) {
	m := New(80, 20)
	m.SetContent("apple banana apple")

	m.ToggleSearch()
	m.SetSearchQuery("apple")
	m.ToggleSearch() // close search

	view := m.View()
	if strings.Contains(view, matchBgOn) {
		t.Error("expected search highlights removed after closing search")
	}
}
