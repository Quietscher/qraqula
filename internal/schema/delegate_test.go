package schema

import (
	"bytes"
	"strings"
	"testing"

	"charm.land/bubbles/v2/list"
)

func TestDelegateHeight(t *testing.T) {
	d := newBrowserDelegate()
	if d.Height() != 1 {
		t.Errorf("expected height 1, got %d", d.Height())
	}
}

func TestDelegateSpacing(t *testing.T) {
	d := newBrowserDelegate()
	if d.Spacing() != 0 {
		t.Errorf("expected spacing 0, got %d", d.Spacing())
	}
}

func TestDelegateRenderNormalItem(t *testing.T) {
	d := newBrowserDelegate()
	items := []list.Item{
		browserItem{name: "user(id: ID!): User", desc: "Get a user by ID", target: "User"},
	}
	l := list.New(items, d, 60, 20)

	var buf bytes.Buffer
	d.Render(&buf, l, 0, items[0])
	output := buf.String()
	if !strings.Contains(output, "user") {
		t.Errorf("expected render to contain 'user', got %q", output)
	}
}

func TestDelegateRenderDrillableArrow(t *testing.T) {
	d := newBrowserDelegate()
	items := []list.Item{
		browserItem{name: "User", target: "User", badge: "OBJECT"},
	}
	l := list.New(items, d, 60, 20)

	var buf bytes.Buffer
	d.Render(&buf, l, 0, items[0])
	output := buf.String()
	if !strings.Contains(output, "→") {
		t.Errorf("expected drillable arrow in output, got %q", output)
	}
}

func TestDelegateRenderBadge(t *testing.T) {
	d := newBrowserDelegate()
	items := []list.Item{
		browserItem{name: "Query", badge: "OBJECT", target: "Query"},
	}
	l := list.New(items, d, 60, 20)

	var buf bytes.Buffer
	d.Render(&buf, l, 0, items[0])
	output := buf.String()
	if !strings.Contains(output, "OBJECT") {
		t.Errorf("expected badge 'OBJECT' in output, got %q", output)
	}
}

func TestDelegateRenderDeprecated(t *testing.T) {
	d := newBrowserDelegate()
	items := []list.Item{
		browserItem{name: "oldName: String", deprecated: true, dimNote: "deprecated: use name"},
	}
	l := list.New(items, d, 60, 20)

	var buf bytes.Buffer
	d.Render(&buf, l, 0, items[0])
	output := buf.String()
	if !strings.Contains(output, "deprecated") {
		t.Errorf("expected deprecation note in output, got %q", output)
	}
}

func TestDelegateRenderColorCodedField(t *testing.T) {
	d := newBrowserDelegate()
	items := []list.Item{
		browserItem{
			name:          "user(id: ID!): User",
			fieldName:     "user",
			fieldArgs:     "(id: ID!)",
			fieldType:     "User",
			fieldTypeKind: "OBJECT",
			target:        "User",
		},
	}
	l := list.New(items, d, 60, 20)

	var buf bytes.Buffer
	d.Render(&buf, l, 0, items[0])
	output := buf.String()

	// Should contain the field name and type parts
	if !strings.Contains(output, "user") {
		t.Errorf("expected render to contain 'user', got %q", output)
	}
	if !strings.Contains(output, "User") {
		t.Errorf("expected render to contain return type 'User', got %q", output)
	}
	// Single line — no newlines
	if strings.Contains(output, "\n") {
		t.Errorf("expected single-line render, got multi-line: %q", output)
	}
}

func TestRenderBreadcrumbs(t *testing.T) {
	result := renderBreadcrumbs([]string{"Schema", "Query", "User"}, 80)
	if !strings.Contains(result, "Schema") {
		t.Error("expected breadcrumbs to contain 'Schema'")
	}
	if !strings.Contains(result, "Query") {
		t.Error("expected breadcrumbs to contain 'Query'")
	}
	if !strings.Contains(result, "User") {
		t.Error("expected breadcrumbs to contain 'User'")
	}
	if !strings.Contains(result, "›") {
		t.Error("expected breadcrumbs to contain separator")
	}
}

func TestRenderBreadcrumbsSinglePage(t *testing.T) {
	result := renderBreadcrumbs([]string{"Schema"}, 80)
	if result != "" {
		t.Errorf("expected empty breadcrumbs for single page, got %q", result)
	}
}

func TestDelegateRenderSearchParentPrefix(t *testing.T) {
	d := newBrowserDelegate()
	items := []list.Item{
		browserItem{
			name:         "email",
			desc:         "String",
			searchParent: "User",
		},
	}
	l := list.New(items, d, 60, 20)

	var buf bytes.Buffer
	d.Render(&buf, l, 0, items[0])
	output := stripANSI(buf.String())

	if !strings.Contains(output, "User") {
		t.Errorf("expected render to contain search parent 'User', got %q", output)
	}
	if !strings.Contains(output, "›") {
		t.Errorf("expected render to contain separator '›', got %q", output)
	}
	if !strings.Contains(output, "email") {
		t.Errorf("expected render to contain item name 'email', got %q", output)
	}
}

func TestDelegateRenderSearchParentFieldItem(t *testing.T) {
	d := newBrowserDelegate()
	items := []list.Item{
		browserItem{
			name:          "user(id: ID!): User",
			fieldName:     "user",
			fieldArgs:     "(id: ID!)",
			fieldType:     "User",
			fieldTypeKind: "OBJECT",
			target:        "Query",
			searchParent:  "Query",
		},
	}
	l := list.New(items, d, 80, 20)

	var buf bytes.Buffer
	d.Render(&buf, l, 0, items[0])
	output := stripANSI(buf.String())

	if !strings.Contains(output, "Query") {
		t.Errorf("expected search parent 'Query' in output, got %q", output)
	}
	if !strings.Contains(output, "user") {
		t.Errorf("expected field name 'user' in output, got %q", output)
	}
	if !strings.Contains(output, "›") {
		t.Errorf("expected separator in output, got %q", output)
	}
}

func TestDelegateRenderNoSearchParent(t *testing.T) {
	d := newBrowserDelegate()
	items := []list.Item{
		browserItem{name: "Query", badge: "OBJECT", target: "Query"},
	}
	l := list.New(items, d, 60, 20)

	var buf bytes.Buffer
	d.Render(&buf, l, 0, items[0])
	output := stripANSI(buf.String())

	// Should NOT contain the separator when searchParent is empty
	if strings.Contains(output, "›") {
		t.Errorf("expected no separator for non-search-parent item, got %q", output)
	}
}
