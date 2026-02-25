package history

import (
	"strings"
	"testing"
)

func TestRenderFolderLine(t *testing.T) {
	si := sidebarItem{kind: kindFolder, name: "MyFolder", collapsed: false}
	output := renderFolderLine(si, false, 40, 0)

	if !strings.Contains(output, "MyFolder") {
		t.Errorf("expected folder name in output, got %q", output)
	}
	if !strings.Contains(output, "📂") {
		t.Errorf("expected open folder icon in output, got %q", output)
	}
}

func TestRenderFolderLineCollapsed(t *testing.T) {
	si := sidebarItem{kind: kindFolder, name: "ClosedFolder", collapsed: true}
	output := renderFolderLine(si, false, 40, 0)

	if !strings.Contains(output, "📁") {
		t.Errorf("expected closed folder icon in output, got %q", output)
	}
}

func TestRenderEntryLine(t *testing.T) {
	si := sidebarItem{kind: kindEntry, name: "GetUsers", folder: "api", entryID: "abc123", endpoint: "example.com"}
	output := renderEntryLine(si, false, 60, 0)

	if !strings.Contains(output, "GetUsers") {
		t.Errorf("expected entry name in output, got %q", output)
	}
}

func TestRenderEntryLineUnsorted(t *testing.T) {
	si := sidebarItem{kind: kindEntry, name: "TestQuery", folder: "", entryID: "abc"}
	output := renderEntryLine(si, false, 40, 0)

	if !strings.Contains(output, "TestQuery") {
		t.Errorf("expected entry name in output, got %q", output)
	}
}

func TestMarqueeReturnsFullStringWhenFits(t *testing.T) {
	result := marquee("short", 10, 0)
	if result != "short" {
		t.Errorf("expected %q, got %q", "short", result)
	}
}

func TestMarqueeScrollsFromOffset(t *testing.T) {
	s := "abcdefghij"
	result := marquee(s, 5, 3)
	if len(result) == 0 {
		t.Error("expected non-empty result")
	}
	if !strings.HasPrefix(result, "d") {
		t.Errorf("expected result to start with 'd', got %q", result)
	}
}

func TestMarqueeWrapsAtEnd(t *testing.T) {
	s := "abcdefghij"
	result := marquee(s, 5, 8)
	if result != "ij" {
		t.Errorf("expected %q, got %q", "ij", result)
	}
}

func TestMarqueeWrapsOverLength(t *testing.T) {
	s := "abcdefghij"
	result := marquee(s, 5, 20)
	if len(result) == 0 {
		t.Error("expected non-empty result")
	}
}

func TestNeedsScroll(t *testing.T) {
	if needsScroll("short", 20) {
		t.Error("short string should not need scroll at width 20")
	}
	if !needsScroll("this is a very long name that exceeds width", 10) {
		t.Error("long string should need scroll at width 10")
	}
}

func TestRenderFolderLineSelectedUsesMarquee(t *testing.T) {
	longName := "ThisIsAVeryLongFolderNameThatWillBeTruncated"
	si := sidebarItem{kind: kindFolder, name: longName, collapsed: false}

	// Render at offset 0
	output1 := renderFolderLine(si, true, 25, 0)

	// Render at offset 5
	output2 := renderFolderLine(si, true, 25, 5)

	// The two renders should differ (scrolled text)
	if output1 == output2 {
		t.Errorf("expected different output after scroll, got same: %q", output1)
	}
}
