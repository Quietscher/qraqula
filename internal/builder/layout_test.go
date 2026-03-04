package builder

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/qraqula/qla/internal/schema"
)

// bigSchema builds a schema with many fields and long type names to stress-test layout.
func bigSchema() *schema.Schema {
	// Create a type with many fields (some with long type names)
	var userFields []schema.Field
	fieldNames := []string{
		"id", "username", "email", "firstName", "lastName",
		"avatarUrl", "bio", "createdAt", "updatedAt", "isActive",
		"role", "department", "phoneNumber", "address", "city",
		"country", "zipCode", "latitude", "longitude", "timezone",
	}
	for _, name := range fieldNames {
		userFields = append(userFields, schema.Field{
			Name: name,
			Type: nonNull(namedRef("String")),
		})
	}
	// Add a nested object field
	userFields = append(userFields, schema.Field{
		Name: "organization",
		Type: namedRef("Organization"),
	})

	s := &schema.Schema{
		QueryType: &schema.TypeRef{Kind: "NAMED", Name: strPtr("Query")},
		Types: []schema.FullType{
			{
				Kind: "OBJECT",
				Name: "Query",
				Fields: []schema.Field{
					{
						Name: "users",
						Type: nonNull(listOf(nonNull(namedRef("User")))),
						Args: []schema.InputValue{
							{Name: "filter", Type: namedRef("UserFilterInput")},
							{Name: "sortBy", Type: namedRef("SortOrder")},
							{Name: "limit", Type: namedRef("Int")},
							{Name: "offset", Type: namedRef("Int")},
							{Name: "includeInactive", Type: namedRef("Boolean")},
						},
					},
				},
			},
			{
				Kind: "OBJECT",
				Name: "User",
				Fields: userFields,
			},
			{
				Kind: "OBJECT",
				Name: "Organization",
				Fields: []schema.Field{
					{Name: "id", Type: nonNull(namedRef("String"))},
					{Name: "name", Type: nonNull(namedRef("String"))},
					{Name: "description", Type: namedRef("String")},
				},
			},
			{Kind: "SCALAR", Name: "String"},
			{Kind: "SCALAR", Name: "Int"},
			{Kind: "SCALAR", Name: "Boolean"},
			{Kind: "INPUT_OBJECT", Name: "UserFilterInput"},
			{Kind: "ENUM", Name: "SortOrder"},
		},
	}
	return s
}

// schemaWithLongArgTypes builds a schema with fields that have long argument type names.
func schemaWithLongArgTypes() *schema.Schema {
	s := &schema.Schema{
		QueryType: &schema.TypeRef{Kind: "NAMED", Name: strPtr("Query")},
		Types: []schema.FullType{
			{
				Kind: "OBJECT",
				Name: "Query",
				Fields: []schema.Field{
					{
						Name: "searchEntities",
						Type: nonNull(listOf(nonNull(namedRef("Entity")))),
						Args: []schema.InputValue{
							{Name: "filterConfiguration", Type: nonNull(namedRef("EntitySearchFilterConfigurationInput"))},
							{Name: "sortOrderPreference", Type: namedRef("EntitySortOrderPreferenceEnum")},
							{Name: "paginationSettings", Type: namedRef("PaginationSettingsInput")},
						},
					},
				},
			},
			{
				Kind: "OBJECT",
				Name: "Entity",
				Fields: []schema.Field{
					{Name: "id", Type: nonNull(namedRef("String"))},
					{Name: "name", Type: nonNull(namedRef("String"))},
				},
			},
			{Kind: "SCALAR", Name: "String"},
			{Kind: "INPUT_OBJECT", Name: "EntitySearchFilterConfigurationInput"},
			{Kind: "ENUM", Name: "EntitySortOrderPreferenceEnum"},
			{Kind: "INPUT_OBJECT", Name: "PaginationSettingsInput"},
		},
	}
	return s
}

// setupBuilderWithManyFields creates a builder model with many fields selected.
func setupBuilderWithManyFields(t *testing.T, w, h int) Model {
	t.Helper()
	s := bigSchema()
	m := New()
	m.SetSize(w, h)

	// Find the "users" field
	queryType := s.TypeByName("Query")
	var usersField schema.Field
	for _, f := range queryType.Fields {
		if f.Name == "users" {
			usersField = f
			break
		}
	}

	m.schema = s
	m.visible = true
	m.mode = modeTree
	m.pane = paneTree
	m.opType = "query"
	m.opField = usersField.Name
	m.root = BuildTreeFromField(s, usersField)

	// Select all children to generate a long preview
	for _, child := range m.root.Children {
		child.Selected = true
	}
	// Expand the organization child
	for _, child := range m.root.Children {
		if child.Name == "organization" {
			child.Expanded = true
			EnsureChildrenReady(s, child)
			for _, grandchild := range child.Children {
				grandchild.Selected = true
			}
			break
		}
	}

	m.rebuildFlat()
	m.updatePreview()
	m.updateStatusHints()
	return m
}

func TestTreeOverlayHeight_ExactDimensions(t *testing.T) {
	// Test that renderTreeOverlay output has EXACTLY h lines for various sizes.
	for _, tc := range []struct {
		name string
		w, h int
	}{
		{"small_terminal", 80, 24},
		{"medium_terminal", 120, 35},
		{"large_terminal", 160, 50},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := setupBuilderWithManyFields(t, tc.w, tc.h)
			output := m.renderTreeOverlay(tc.w, tc.h)
			lines := strings.Split(output, "\n")
			if len(lines) != tc.h {
				t.Errorf("renderTreeOverlay(%d, %d) produced %d lines, want exactly %d",
					tc.w, tc.h, len(lines), tc.h)
			}
		})
	}
}

func TestTreeOverlay_PreviewBoxHeight(t *testing.T) {
	// Verify the preview box doesn't exceed its allocated height.
	w, h := 120, 35
	m := setupBuilderWithManyFields(t, w, h)

	// Calculate expected dimensions
	statusH := 1
	argsOuterH := 3
	topH := h - statusH - argsOuterH // the height available for preview+tree row

	previewOuterW := w * 40 / 100
	previewInnerW := previewOuterW - 4
	previewInnerH := topH - 2

	// Render preview content the same way renderTreeOverlay does
	m.preview.SetWidth(previewInnerW)
	m.preview.SetHeight(previewInnerH)
	previewView := m.preview.View()
	clipped := clipContent(previewView, previewInnerW, previewInnerH)

	clippedLines := strings.Split(clipped, "\n")
	if len(clippedLines) != previewInnerH {
		t.Errorf("clipped preview has %d lines, want %d", len(clippedLines), previewInnerH)
	}

	// Now render through the border style and check the final box height
	previewBox := m.paneBorderStyle(panePreview, previewOuterW-2, topH-2).Render(clipped)
	boxLines := strings.Split(previewBox, "\n")
	if len(boxLines) != topH {
		t.Errorf("preview box has %d lines, want %d (topH). Content was %d inner lines",
			len(boxLines), topH, len(clippedLines))
	}

	// Verify each line doesn't exceed the outer width
	for i, line := range boxLines {
		lineW := lipgloss.Width(line)
		if lineW > previewOuterW {
			t.Errorf("preview box line %d has visible width %d, exceeds outer width %d: %q",
				i, lineW, previewOuterW, line)
		}
	}
}

func TestTreeOverlay_ArgsBoxHeight(t *testing.T) {
	// Verify the args box is exactly 3 lines (border + 1 content line).
	w, h := 120, 35
	m := setupBuilderWithManyFields(t, w, h)

	argsInnerW := w - 4
	argsContent := m.renderArgsHorizontal(argsInnerW)

	// Args content must be a single line
	argsLines := strings.Split(argsContent, "\n")
	if len(argsLines) != 1 {
		t.Errorf("renderArgsHorizontal produced %d lines, want 1", len(argsLines))
	}

	// Check the visible width doesn't exceed inner width
	argsW := lipgloss.Width(argsContent)
	if argsW > argsInnerW {
		t.Errorf("args content visible width %d exceeds inner width %d", argsW, argsInnerW)
	}

	// Render through border style and check height
	argsBox := m.paneBorderStyle(paneArgs, w-2, 1).Render(argsContent)
	boxLines := strings.Split(argsBox, "\n")
	if len(boxLines) != 3 {
		t.Errorf("args box has %d lines, want 3", len(boxLines))
	}
}

func TestTreeOverlay_ArgsWithLongTypeNames(t *testing.T) {
	// Test that args with very long type names don't overflow the overlay.
	s := schemaWithLongArgTypes()
	m := New()
	w, h := 80, 24 // small terminal
	m.SetSize(w, h)

	queryType := s.TypeByName("Query")
	var field schema.Field
	for _, f := range queryType.Fields {
		if f.Name == "searchEntities" {
			field = f
			break
		}
	}

	m.schema = s
	m.visible = true
	m.mode = modeTree
	m.pane = paneTree
	m.opType = "query"
	m.opField = field.Name
	m.root = BuildTreeFromField(s, field)
	m.rebuildFlat()
	m.updatePreview()
	m.updateStatusHints()

	// Verify the full overlay height — this is the key assertion
	output := m.renderTreeOverlay(w, h)
	lines := strings.Split(output, "\n")
	if len(lines) != h {
		t.Errorf("overlay with long arg types: %d lines, want %d", len(lines), h)
	}

	// Verify the truncated args content width (as used in renderTreeOverlay)
	argsInnerW := w - 4
	argsContent := ansi.Truncate(m.renderArgsHorizontal(argsInnerW), argsInnerW, "")
	argsW := lipgloss.Width(argsContent)
	if argsW > argsInnerW {
		t.Errorf("truncated args content: visible width %d exceeds inner width %d",
			argsW, argsInnerW)
	}
}

func TestTreeOverlay_TreeBoxHeight(t *testing.T) {
	// Verify the tree box doesn't exceed its allocated height even with deep nesting.
	w, h := 80, 24
	s := countriesSchema()
	m := New()
	m.SetSize(w, h)

	queryType := s.TypeByName("Query")
	var countriesField schema.Field
	for _, f := range queryType.Fields {
		if f.Name == "countries" {
			countriesField = f
			break
		}
	}

	m.schema = s
	m.visible = true
	m.mode = modeTree
	m.pane = paneTree
	m.opType = "query"
	m.opField = countriesField.Name
	m.root = BuildTreeFromField(s, countriesField)

	// Expand deeply to create indented lines that might exceed width
	node := m.root
	for i := 0; i < 5; i++ {
		EnsureChildrenReady(s, node)
		node.Expanded = true
		for _, child := range node.Children {
			if !child.IsLeaf {
				node = child
				break
			}
		}
	}

	m.rebuildFlat()
	m.updatePreview()
	m.updateStatusHints()

	output := m.renderTreeOverlay(w, h)
	lines := strings.Split(output, "\n")
	if len(lines) != h {
		t.Errorf("overlay with deep nesting: %d lines, want %d", len(lines), h)
	}
}
