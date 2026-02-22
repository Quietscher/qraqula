package history

import (
	"bytes"
	"strings"
	"testing"

	"charm.land/bubbles/v2/list"
)

func TestDelegateHeight(t *testing.T) {
	d := newSidebarDelegate()
	if d.Height() != 1 {
		t.Errorf("expected height 1, got %d", d.Height())
	}
}

func TestDelegateSpacing(t *testing.T) {
	d := newSidebarDelegate()
	if d.Spacing() != 0 {
		t.Errorf("expected spacing 0, got %d", d.Spacing())
	}
}

func TestDelegateRenderFolder(t *testing.T) {
	d := newSidebarDelegate()
	items := []list.Item{
		sidebarItem{kind: kindFolder, name: "MyFolder", collapsed: false},
	}
	l := list.New(items, d, 40, 10)

	var buf bytes.Buffer
	d.Render(&buf, l, 0, items[0])
	output := buf.String()

	if !strings.Contains(output, "MyFolder") {
		t.Errorf("expected folder name in output, got %q", output)
	}
	if !strings.Contains(output, "📂") {
		t.Errorf("expected open folder icon in output, got %q", output)
	}
}

func TestDelegateRenderFolderCollapsed(t *testing.T) {
	d := newSidebarDelegate()
	items := []list.Item{
		sidebarItem{kind: kindFolder, name: "ClosedFolder", collapsed: true},
	}
	l := list.New(items, d, 40, 10)

	var buf bytes.Buffer
	d.Render(&buf, l, 0, items[0])
	output := buf.String()

	if !strings.Contains(output, "📁") {
		t.Errorf("expected closed folder icon in output, got %q", output)
	}
}

func TestDelegateRenderEntry(t *testing.T) {
	d := newSidebarDelegate()
	items := []list.Item{
		sidebarItem{kind: kindEntry, name: "GetUsers", folder: "api", entryID: "abc123", endpoint: "example.com"},
	}
	l := list.New(items, d, 60, 10)

	var buf bytes.Buffer
	d.Render(&buf, l, 0, items[0])
	output := buf.String()

	if !strings.Contains(output, "GetUsers") {
		t.Errorf("expected entry name in output, got %q", output)
	}
}

func TestDelegateRenderEntryUnsorted(t *testing.T) {
	d := newSidebarDelegate()
	items := []list.Item{
		sidebarItem{kind: kindEntry, name: "TestQuery", folder: "", entryID: "abc"},
	}
	l := list.New(items, d, 40, 10)

	var buf bytes.Buffer
	d.Render(&buf, l, 0, items[0])
	output := buf.String()

	if !strings.Contains(output, "TestQuery") {
		t.Errorf("expected entry name in output, got %q", output)
	}
}

func TestDelegateRenderSeparator(t *testing.T) {
	d := newSidebarDelegate()
	items := []list.Item{
		sidebarItem{kind: kindSeparator, name: "Unsorted"},
	}
	l := list.New(items, d, 40, 10)

	var buf bytes.Buffer
	d.Render(&buf, l, 0, items[0])
	output := buf.String()

	if !strings.Contains(output, "Unsorted") {
		t.Errorf("expected separator text in output, got %q", output)
	}
}

func TestSidebarItemFilterValue(t *testing.T) {
	item := sidebarItem{name: "GetUser", endpoint: "example.com"}
	fv := item.FilterValue()
	if !strings.Contains(fv, "GetUser") || !strings.Contains(fv, "example.com") {
		t.Errorf("expected filter value to contain name and endpoint, got %q", fv)
	}
}

func TestMarqueeReturnsFullStringWhenFits(t *testing.T) {
	result := marquee("short", 10, 0)
	if result != "short" {
		t.Errorf("expected %q, got %q", "short", result)
	}
}

func TestMarqueeScrollsFromOffset(t *testing.T) {
	// "abcdefghij" at offset 3 with maxWidth 5 → "defgh" or truncated "defg…"
	s := "abcdefghij"
	result := marquee(s, 5, 3)
	// Should start from 'd' and fit 5 chars
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
	// From offset 8: "ij" which fits in 5
	if result != "ij" {
		t.Errorf("expected %q, got %q", "ij", result)
	}
}

func TestMarqueeWrapsOverLength(t *testing.T) {
	s := "abcdefghij"
	// offset >= len should wrap to 0
	result := marquee(s, 5, 20)
	// Same as offset 0 → "abcde" or "abcd…"
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

func TestDelegateRenderSelectedUsesMarquee(t *testing.T) {
	d := newSidebarDelegate()
	longName := "ThisIsAVeryLongFolderNameThatWillBeTruncated"
	items := []list.Item{
		sidebarItem{kind: kindFolder, name: longName, collapsed: false},
	}
	l := list.New(items, d, 25, 10) // narrow width to force truncation

	// First render at offset 0
	var buf1 bytes.Buffer
	d.Render(&buf1, l, 0, items[0])
	output1 := buf1.String()

	// Advance scroll offset
	d.scroll.offset = 5
	d.scroll.active = true

	var buf2 bytes.Buffer
	d.Render(&buf2, l, 0, items[0])
	output2 := buf2.String()

	// The two renders should differ (scrolled text)
	if output1 == output2 {
		t.Errorf("expected different output after scroll, got same: %q", output1)
	}
}
