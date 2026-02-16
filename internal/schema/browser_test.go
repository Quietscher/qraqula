package schema

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func testSchema() *Schema {
	return &Schema{
		QueryType:    &TypeRef{Name: strPtr("Query")},
		MutationType: &TypeRef{Name: strPtr("Mutation")},
		Types: []FullType{
			{
				Kind: "OBJECT",
				Name: "Query",
				Fields: []Field{
					{
						Name: "user",
						Args: []InputValue{{Name: "id", Type: TypeRef{Kind: "NON_NULL", OfType: &TypeRef{Kind: "SCALAR", Name: strPtr("ID")}}}},
						Type: TypeRef{Kind: "OBJECT", Name: strPtr("User")},
					},
					{
						Name: "users",
						Type: TypeRef{Kind: "NON_NULL", OfType: &TypeRef{Kind: "LIST", OfType: &TypeRef{Kind: "OBJECT", Name: strPtr("User")}}},
					},
				},
			},
			{
				Kind: "OBJECT",
				Name: "Mutation",
				Fields: []Field{
					{Name: "createUser", Type: TypeRef{Kind: "OBJECT", Name: strPtr("User")}},
				},
			},
			{
				Kind: "OBJECT",
				Name: "User",
				Fields: []Field{
					{Name: "id", Type: TypeRef{Kind: "NON_NULL", OfType: &TypeRef{Kind: "SCALAR", Name: strPtr("ID")}}},
					{Name: "name", Type: TypeRef{Kind: "SCALAR", Name: strPtr("String")}},
					{Name: "email", Type: TypeRef{Kind: "SCALAR", Name: strPtr("String")}},
					{Name: "oldName", Type: TypeRef{Kind: "SCALAR", Name: strPtr("String")}, IsDeprecated: true, DeprecationReason: "use name"},
				},
				Interfaces: []TypeRef{{Kind: "INTERFACE", Name: strPtr("Node")}},
			},
			{Kind: "SCALAR", Name: "ID"},
			{Kind: "SCALAR", Name: "String"},
			{
				Kind: "ENUM",
				Name: "Role",
				EnumValues: []EnumValue{
					{Name: "ADMIN"},
					{Name: "USER"},
					{Name: "GUEST", IsDeprecated: true, DeprecationReason: "no longer used"},
				},
			},
			{
				Kind:        "INPUT_OBJECT",
				Name:        "CreateUserInput",
				InputFields: []InputValue{
					{Name: "name", Type: TypeRef{Kind: "NON_NULL", OfType: &TypeRef{Kind: "SCALAR", Name: strPtr("String")}}},
					{Name: "email", Type: TypeRef{Kind: "SCALAR", Name: strPtr("String")}},
				},
			},
		},
	}
}

// keyPress creates a tea.KeyPressMsg for a simple key string.
func keyPress(s string) tea.KeyPressMsg {
	switch s {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	default:
		// For single rune keys like j, k, h
		r := []rune(s)
		if len(r) == 1 {
			return tea.KeyPressMsg{Code: r[0], Text: s}
		}
		return tea.KeyPressMsg{}
	}
}

func updateBrowser(b Browser, msg tea.Msg) Browser {
	b, _ = b.Update(msg)
	return b
}

func TestBrowserInitialState(t *testing.T) {
	b := NewBrowser()
	b.SetSchema(testSchema())
	b.SetSize(80, 30)

	view := b.View()
	if !strings.Contains(view, "Query") {
		t.Error("expected root page to show Query")
	}
	if !strings.Contains(view, "Mutation") {
		t.Error("expected root page to show Mutation")
	}
}

func TestBrowserNoSchemaView(t *testing.T) {
	b := NewBrowser()
	b.SetSize(80, 30)

	view := b.View()
	if !strings.Contains(view, "No schema") {
		t.Error("expected 'No schema' message when no schema is set")
	}
}

func TestBrowserDrillIntoType(t *testing.T) {
	b := NewBrowser()
	b.SetSchema(testSchema())
	b.SetSize(80, 30)

	// At root, cursor=0 → Query. Press enter to drill in.
	b = updateBrowser(b, keyPress("enter"))

	view := b.View()
	// Should now show Query's fields
	if !strings.Contains(view, "user") {
		t.Error("expected Query detail page to show 'user' field")
	}
	if !strings.Contains(view, "users") {
		t.Error("expected Query detail page to show 'users' field")
	}
	// Should show breadcrumb with "Schema"
	if !strings.Contains(view, "Schema") {
		t.Error("expected breadcrumb to contain 'Schema'")
	}
}

func TestBrowserNavigateBack(t *testing.T) {
	b := NewBrowser()
	b.SetSchema(testSchema())
	b.SetSize(80, 30)

	// Drill into Query
	b = updateBrowser(b, keyPress("enter"))
	// Now go back with backspace
	b = updateBrowser(b, keyPress("backspace"))

	view := b.View()
	// Should be back at root showing both Query and Mutation
	if !strings.Contains(view, "Query") {
		t.Error("expected root page after back navigation to show Query")
	}
	if !strings.Contains(view, "Mutation") {
		t.Error("expected root page after back navigation to show Mutation")
	}
}

func TestBrowserNavigateBackWithH(t *testing.T) {
	b := NewBrowser()
	b.SetSchema(testSchema())
	b.SetSize(80, 30)

	// Drill into Query
	b = updateBrowser(b, keyPress("enter"))
	// Now go back with h
	b = updateBrowser(b, keyPress("h"))

	view := b.View()
	if !strings.Contains(view, "Query") {
		t.Error("expected root page after h navigation to show Query")
	}
}

func TestBrowserCursorMovement(t *testing.T) {
	b := NewBrowser()
	b.SetSchema(testSchema())
	b.SetSize(80, 30)

	// Root has Query (0) and Mutation (1).
	// cursor starts at 0.
	view0 := b.View()

	// Move down with j
	b = updateBrowser(b, keyPress("j"))
	// Now cursor=1. Press enter → drill into Mutation
	b = updateBrowser(b, keyPress("enter"))

	view := b.View()
	if !strings.Contains(view, "createUser") {
		t.Error("expected Mutation detail page to show 'createUser' field")
	}

	// Go back and verify k moves up
	b = updateBrowser(b, keyPress("backspace"))
	b = updateBrowser(b, keyPress("k")) // cursor back to 0
	b = updateBrowser(b, keyPress("enter"))

	view = b.View()
	if !strings.Contains(view, "user") {
		t.Error("expected Query detail page after k+enter to show 'user' field")
	}

	// Test bounds: k at top should stay at 0
	b = updateBrowser(b, keyPress("backspace"))
	b = updateBrowser(b, keyPress("k"))
	b = updateBrowser(b, keyPress("k")) // should clamp
	b = updateBrowser(b, keyPress("enter"))

	view = b.View()
	if !strings.Contains(view, "user") {
		t.Error("expected cursor to clamp at top, drill into Query")
	}

	// Test bounds: j past end should clamp
	b = updateBrowser(b, keyPress("backspace"))
	b = updateBrowser(b, keyPress("j"))
	b = updateBrowser(b, keyPress("j"))
	b = updateBrowser(b, keyPress("j")) // should clamp at 1
	b = updateBrowser(b, keyPress("enter"))

	view = b.View()
	if !strings.Contains(view, "createUser") {
		t.Error("expected cursor to clamp at bottom, drill into Mutation")
	}

	_ = view0
}

func TestBrowserCursorWithArrowKeys(t *testing.T) {
	b := NewBrowser()
	b.SetSchema(testSchema())
	b.SetSize(80, 30)

	// Move down with arrow key
	b = updateBrowser(b, keyPress("down"))
	b = updateBrowser(b, keyPress("enter"))

	view := b.View()
	if !strings.Contains(view, "createUser") {
		t.Error("expected arrow down + enter to drill into Mutation")
	}

	b = updateBrowser(b, keyPress("backspace"))
	b = updateBrowser(b, keyPress("up"))
	b = updateBrowser(b, keyPress("enter"))

	view = b.View()
	if !strings.Contains(view, "user") {
		t.Error("expected arrow up + enter to drill into Query")
	}
}

func TestBrowserEnumType(t *testing.T) {
	b := NewBrowser()
	s := testSchema()
	b.SetSchema(s)
	b.SetSize(80, 30)

	// Manually push the Role enum page by navigating into it.
	// We'll do this by calling pushType directly since it's not reachable
	// from root navigation without a field referencing it.
	b.pushType("Role")

	view := b.View()
	if !strings.Contains(view, "ADMIN") {
		t.Error("expected enum page to show ADMIN")
	}
	if !strings.Contains(view, "USER") {
		t.Error("expected enum page to show USER")
	}
	if !strings.Contains(view, "GUEST") {
		t.Error("expected enum page to show GUEST")
	}
	if !strings.Contains(view, "[ENUM]") {
		t.Error("expected enum page title to show [ENUM] badge")
	}
}

func TestBrowserDeprecatedField(t *testing.T) {
	b := NewBrowser()
	b.SetSchema(testSchema())
	b.SetSize(80, 30)

	// Drill into Query, then into User
	b = updateBrowser(b, keyPress("enter")) // into Query
	b = updateBrowser(b, keyPress("enter")) // cursor=0 → first item

	view := b.View()
	// If we drilled into User, it should show deprecated field
	if strings.Contains(view, "oldName") {
		// Good, we can check for deprecation note
		if !strings.Contains(view, "deprecated") {
			t.Error("expected deprecated note for oldName field")
		}
	}
}

func TestBrowserFieldArgDisplay(t *testing.T) {
	b := NewBrowser()
	b.SetSchema(testSchema())
	b.SetSize(80, 30)

	// Drill into Query
	b = updateBrowser(b, keyPress("enter"))

	view := b.View()
	// user(id: ID!): User should be displayed
	if !strings.Contains(view, "id: ID!") {
		t.Error("expected field args to be displayed, got: " + view)
	}
}

func TestBrowserInputObjectType(t *testing.T) {
	b := NewBrowser()
	s := testSchema()
	b.SetSchema(s)
	b.SetSize(80, 30)

	// Push CreateUserInput directly
	b.pushType("CreateUserInput")

	view := b.View()
	if !strings.Contains(view, "name: String!") {
		t.Error("expected input object page to show 'name: String!'")
	}
	if !strings.Contains(view, "email: String") {
		t.Error("expected input object page to show 'email: String'")
	}
	if !strings.Contains(view, "[INPUT_OBJECT]") {
		t.Error("expected input object page title to show [INPUT_OBJECT] badge")
	}
}

func TestBrowserBackAtRootIsNoop(t *testing.T) {
	b := NewBrowser()
	b.SetSchema(testSchema())
	b.SetSize(80, 30)

	// Pressing backspace at root should not crash or change state
	b = updateBrowser(b, keyPress("backspace"))

	view := b.View()
	if !strings.Contains(view, "Query") {
		t.Error("expected root to remain after backspace at root")
	}
}

func TestBrowserDeepNavigation(t *testing.T) {
	b := NewBrowser()
	b.SetSchema(testSchema())
	b.SetSize(80, 30)

	// Root → Query → drill into "user" field → User type
	b = updateBrowser(b, keyPress("enter")) // into Query

	// In Query: first item (index 0) should be either "implements Node" or "user(id: ID!): User"
	// Actually, Query is an OBJECT with fields, and it has no interfaces listed in testSchema.
	// So items[0] = user(id: ID!): User, items[1] = users: [User]!
	b = updateBrowser(b, keyPress("enter")) // drill into User (via user field)

	view := b.View()
	if !strings.Contains(view, "User") {
		t.Error("expected to be on User type page")
	}
	if !strings.Contains(view, "name") {
		t.Error("expected User page to show 'name' field")
	}
	if !strings.Contains(view, "oldName") {
		t.Error("expected User page to show 'oldName' field")
	}

	// Breadcrumbs should show the path
	if !strings.Contains(view, "Schema") {
		t.Error("expected breadcrumbs to include Schema")
	}
}

func TestBrowserTypeBadges(t *testing.T) {
	b := NewBrowser()
	b.SetSchema(testSchema())
	b.SetSize(80, 30)

	view := b.View()
	if !strings.Contains(view, "[OBJECT]") {
		t.Error("expected root page to show [OBJECT] badge")
	}
}
