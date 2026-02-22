package schema

import (
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// stripANSI removes ANSI escape sequences from a string so we can reliably
// check for substrings in styled terminal output.
var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

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
	case "left":
		return tea.KeyPressMsg{Code: tea.KeyLeft}
	case "right":
		return tea.KeyPressMsg{Code: tea.KeyRight}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	default:
		// For single rune keys like j, k, h, l, /, G
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

	// Test bounds: j past end should clamp at last item (Variable Types)
	b = updateBrowser(b, keyPress("backspace"))
	for i := 0; i < 20; i++ { // press j way past end
		b = updateBrowser(b, keyPress("j"))
	}
	b = updateBrowser(b, keyPress("enter"))

	view = b.View()
	// Last root item is "Variable Types" group
	if !strings.Contains(view, "Role") && !strings.Contains(view, "CreateUserInput") {
		t.Error("expected cursor to clamp at bottom, drill into Variable Types")
	}

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

	view := stripANSI(b.View())
	if !strings.Contains(view, "ADMIN") {
		t.Error("expected enum page to show ADMIN")
	}
	if !strings.Contains(view, "USER") {
		t.Error("expected enum page to show USER")
	}
	if !strings.Contains(view, "GUEST") {
		t.Error("expected enum page to show GUEST")
	}
	// The delegate renders the badge as the kind name directly (e.g. "ENUM"),
	// not wrapped in brackets. Enum items don't carry a badge, but the
	// breadcrumb shows "Role" and items include deprecated notes.
	if !strings.Contains(view, "no longer used") {
		t.Error("expected enum page to show deprecation reason for GUEST")
	}
}

func TestBrowserDeprecatedField(t *testing.T) {
	b := NewBrowser()
	b.SetSchema(testSchema())
	b.SetSize(80, 30)

	// Drill into Query, then into User
	b = updateBrowser(b, keyPress("enter")) // into Query
	b = updateBrowser(b, keyPress("enter")) // cursor=0 → first item

	view := stripANSI(b.View())
	if !strings.Contains(view, "oldName") {
		t.Error("expected User page to show deprecated 'oldName' field")
	}
	if !strings.Contains(view, "deprecated") {
		t.Error("expected deprecated note for oldName field")
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

	view := stripANSI(b.View())
	// Input object items render field name as title and type as description.
	if !strings.Contains(view, "name") {
		t.Error("expected input object page to show 'name' field")
	}
	if !strings.Contains(view, "String!") {
		t.Error("expected input object page to show 'String!' type")
	}
	if !strings.Contains(view, "email") {
		t.Error("expected input object page to show 'email' field")
	}
	// Breadcrumb should show the type name
	if !strings.Contains(view, "CreateUserInput") {
		t.Error("expected breadcrumb to show 'CreateUserInput'")
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

	view := stripANSI(b.View())
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

	view := stripANSI(b.View())
	// The delegate renders the badge as the kind name directly (no brackets).
	if !strings.Contains(view, "OBJECT") {
		t.Error("expected root page to show OBJECT badge")
	}
}

func TestBrowserDrillInWithRight(t *testing.T) {
	b := NewBrowser()
	b.SetSchema(testSchema())
	b.SetSize(80, 30)

	b = updateBrowser(b, keyPress("right"))

	view := b.View()
	if !strings.Contains(view, "user") {
		t.Error("expected right arrow to drill into Query, showing 'user' field")
	}
}

func TestBrowserNavigateBackWithLeft(t *testing.T) {
	b := NewBrowser()
	b.SetSchema(testSchema())
	b.SetSize(80, 30)

	b = updateBrowser(b, keyPress("enter"))
	b = updateBrowser(b, keyPress("left"))

	view := b.View()
	if !strings.Contains(view, "Query") {
		t.Error("expected left arrow to navigate back to root")
	}
	if !strings.Contains(view, "Mutation") {
		t.Error("expected root page after left arrow to show Mutation")
	}
}

func TestBrowserDrillInWithL(t *testing.T) {
	b := NewBrowser()
	b.SetSchema(testSchema())
	b.SetSize(80, 30)

	// At root, cursor=0 → Query. Press 'l' to drill in.
	b = updateBrowser(b, keyPress("l"))

	view := b.View()
	if !strings.Contains(view, "user") {
		t.Error("expected 'l' to drill into Query, showing 'user' field")
	}
}

func TestBrowserCursorEnd(t *testing.T) {
	b := NewBrowser()
	b.SetSchema(testSchema())
	b.SetSize(80, 30)

	// Press G to go to end of root list, then enter to drill into Variable Types group
	b = updateBrowser(b, keyPress("G"))
	b = updateBrowser(b, keyPress("enter"))

	view := stripANSI(b.View())
	// Last root item is "Variable Types" group, drilling in shows input/enum types
	if !strings.Contains(view, "CreateUserInput") && !strings.Contains(view, "Role") {
		t.Error("expected G to move cursor to end, drill into Variable Types group")
	}
}

func TestBrowserSearchFiltering(t *testing.T) {
	b := NewBrowser()
	b.SetSchema(testSchema())
	b.SetSize(80, 30)

	// Drill into Query to get fields
	b = updateBrowser(b, keyPress("enter"))

	// Start search with /
	b = updateBrowser(b, keyPress("/"))
	if !b.list.SettingFilter() {
		t.Error("expected filter mode to be active after /")
	}

	// Type "users" to filter
	for _, ch := range "users" {
		b = updateBrowser(b, keyPress(string(ch)))
	}

	view := b.View()
	if !strings.Contains(view, "users") {
		t.Error("expected filtered view to show 'users'")
	}

	// Confirm search with enter
	b = updateBrowser(b, keyPress("enter"))
	if b.list.SettingFilter() {
		t.Error("expected filter mode to be inactive after enter")
	}
}

func TestBrowserSearchEscape(t *testing.T) {
	b := NewBrowser()
	b.SetSchema(testSchema())
	b.SetSize(80, 30)

	// Drill into Query
	b = updateBrowser(b, keyPress("enter"))

	// Start search and type something
	b = updateBrowser(b, keyPress("/"))
	b = updateBrowser(b, keyPress("u"))

	// Escape should cancel filtering
	b = updateBrowser(b, keyPress("esc"))
	if b.list.SettingFilter() {
		t.Error("expected filter mode to be inactive after esc")
	}

	// All items should be visible again
	view := b.View()
	if !strings.Contains(view, "user") {
		t.Error("expected all items visible after esc, 'user' missing")
	}
}

func TestBrowserItemCount(t *testing.T) {
	b := NewBrowser()
	b.SetSchema(testSchema())
	b.SetSize(80, 30)

	view := b.View()
	// Root has: Query, Mutation, Variable Types = 3 items
	if !strings.Contains(view, "3 items") {
		t.Error("expected root page to show item count '3 items', got: " + view)
	}
}

func TestBrowserViewFillsHeight(t *testing.T) {
	b := NewBrowser()
	b.SetSchema(testSchema())
	b.SetSize(80, 30)

	view := b.View()
	lines := strings.Split(view, "\n")
	// View should fill the panel (height - 2 for border)
	expectedLines := 28 // 30 - 2 (border)
	if len(lines) < expectedLines {
		t.Errorf("expected view to have at least %d lines for proper sizing, got %d", expectedLines, len(lines))
	}
}

func TestBrowserAllItemsBuiltOnSetSchema(t *testing.T) {
	b := NewBrowser()
	if len(b.allItems) != 0 {
		t.Error("expected no allItems before schema is set")
	}

	b.SetSchema(testSchema())
	if len(b.allItems) == 0 {
		t.Error("expected allItems to be populated after SetSchema")
	}

	// Verify cross-level items include fields from nested types
	foundUserEmail := false
	foundRoleAdmin := false
	for _, si := range b.allItems {
		if si.parentName == "User" && si.item.fieldName == "email" {
			foundUserEmail = true
		}
		if si.parentName == "Role" && si.item.name == "ADMIN" {
			foundRoleAdmin = true
		}
	}
	if !foundUserEmail {
		t.Error("expected allItems to contain User.email")
	}
	if !foundRoleAdmin {
		t.Error("expected allItems to contain Role.ADMIN")
	}
}

func TestBrowserAllItemsClearedOnNilSchema(t *testing.T) {
	b := NewBrowser()
	b.SetSchema(testSchema())
	if len(b.allItems) == 0 {
		t.Fatal("setup: expected allItems after SetSchema")
	}

	b.SetSchema(nil)
	if len(b.allItems) != 0 {
		t.Error("expected allItems to be cleared after SetSchema(nil)")
	}
	if b.filterAugmented {
		t.Error("expected filterAugmented to be false after SetSchema(nil)")
	}
}

func TestBrowserFilterAugmentsItems(t *testing.T) {
	b := NewBrowser()
	b.SetSchema(testSchema())
	b.SetSize(80, 30)

	// At root page, we have 3 items (Query, Mutation, Variable Types)
	initialItems := b.list.Items()
	initialCount := len(initialItems)
	if initialCount != 3 {
		t.Fatalf("expected 3 root items, got %d", initialCount)
	}

	// Enter filter mode by pressing /
	b = updateBrowser(b, keyPress("/"))
	if !b.list.SettingFilter() {
		t.Fatal("expected filter mode to be active after /")
	}

	// After entering filter mode, items should be augmented with cross-level items
	augmentedItems := b.list.Items()
	if len(augmentedItems) <= initialCount {
		t.Errorf("expected augmented items (%d) to be more than initial (%d)",
			len(augmentedItems), initialCount)
	}
	if !b.filterAugmented {
		t.Error("expected filterAugmented to be true")
	}
}

func TestBrowserFilterRestoresItemsOnEsc(t *testing.T) {
	b := NewBrowser()
	b.SetSchema(testSchema())
	b.SetSize(80, 30)

	// Enter filter mode
	b = updateBrowser(b, keyPress("/"))
	b = updateBrowser(b, keyPress("u")) // type something

	// Exit with Esc
	b = updateBrowser(b, keyPress("esc"))
	if b.list.SettingFilter() {
		t.Error("expected filter mode to be inactive after esc")
	}

	// Items should be restored to original page items
	items := b.list.Items()
	if len(items) != 3 {
		t.Errorf("expected 3 root items after filter cancel, got %d", len(items))
	}
	if b.filterAugmented {
		t.Error("expected filterAugmented to be false after esc")
	}
}

func TestBrowserFilterDrillInRestoresState(t *testing.T) {
	b := NewBrowser()
	b.SetSchema(testSchema())
	b.SetSize(80, 30)

	// Drill into Query
	b = updateBrowser(b, keyPress("enter"))

	// Enter filter mode
	b = updateBrowser(b, keyPress("/"))
	if !b.filterAugmented {
		t.Fatal("expected filterAugmented after entering filter mode")
	}

	// Type something then confirm filter
	b = updateBrowser(b, keyPress("u"))
	b = updateBrowser(b, keyPress("enter")) // confirm filter

	// Now navigate to a result (enter to drill in)
	b = updateBrowser(b, keyPress("enter"))

	// After drill-in, filter state should be clean
	if b.filterAugmented {
		t.Error("expected filterAugmented to be false after drill-in")
	}
}

func TestBrowserSearchParentItemsHaveTarget(t *testing.T) {
	b := NewBrowser()
	b.SetSchema(testSchema())
	b.SetSize(80, 30)

	// Enter filter mode at root
	b = updateBrowser(b, keyPress("/"))

	// Check that cross-level items have target set to parent type
	for _, item := range b.list.Items() {
		bi, ok := item.(browserItem)
		if !ok {
			continue
		}
		if bi.searchParent != "" && bi.target == "" {
			t.Errorf("cross-level item %q (parent: %s) should have target set",
				bi.name, bi.searchParent)
		}
	}
}

func TestBrowserCrossLevelSearchRecursiveSchema(t *testing.T) {
	// Test with a recursive schema to ensure no infinite loops
	s := &Schema{
		QueryType: &TypeRef{Name: strPtr("Query")},
		Types: []FullType{
			{Kind: "OBJECT", Name: "Query", Fields: []Field{
				{Name: "tree", Type: TypeRef{Kind: "OBJECT", Name: strPtr("TreeNode")}},
			}},
			{Kind: "OBJECT", Name: "TreeNode", Fields: []Field{
				{Name: "value", Type: TypeRef{Kind: "SCALAR", Name: strPtr("String")}},
				{Name: "parent", Type: TypeRef{Kind: "OBJECT", Name: strPtr("TreeNode")}},
				{Name: "children", Type: TypeRef{Kind: "LIST", OfType: &TypeRef{Kind: "OBJECT", Name: strPtr("TreeNode")}}},
			}},
			{Kind: "SCALAR", Name: "String"},
		},
	}

	b := NewBrowser()
	b.SetSchema(s)
	b.SetSize(80, 30)

	// Should not panic or infinite loop
	if len(b.allItems) == 0 {
		t.Fatal("expected allItems to be populated for recursive schema")
	}

	// Enter filter mode
	b = updateBrowser(b, keyPress("/"))
	if !b.filterAugmented {
		t.Error("expected filter augmentation for recursive schema")
	}

	// Type "value" to search for TreeNode.value from root level
	for _, ch := range "value" {
		b = updateBrowser(b, keyPress(string(ch)))
	}

	// View should still render without panic
	view := b.View()
	if view == "" {
		t.Error("expected non-empty view during cross-level search")
	}
}
