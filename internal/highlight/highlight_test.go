package highlight

import (
	"strings"
	"testing"
)

func TestHighlightGraphQL(t *testing.T) {
	src := "{ countries { name code } }"
	result := Colorize(src, "graphql")
	if !strings.Contains(result, "\x1b[") {
		t.Error("expected ANSI escape codes in highlighted output")
	}
	plain := StripANSI(result)
	if plain != src {
		t.Errorf("expected plain text %q, got %q", src, plain)
	}
}

func TestHighlightJSON(t *testing.T) {
	src := `{"key": "value", "num": 42, "bool": true, "nil": null}`
	result := Colorize(src, "json")
	if !strings.Contains(result, "\x1b[") {
		t.Error("expected ANSI escape codes")
	}
}

func TestHighlightLines(t *testing.T) {
	src := "{\n  name\n}"
	lines := ColorizeLines(src, "graphql")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
}

func TestHighlightEmptyString(t *testing.T) {
	result := Colorize("", "graphql")
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestHighlightUnknownLexer(t *testing.T) {
	result := Colorize("hello", "nonexistent")
	plain := StripANSI(result)
	if plain != "hello" {
		t.Errorf("expected plain text fallback, got %q", plain)
	}
}
