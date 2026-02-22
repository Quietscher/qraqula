package schema

import "testing"

func TestBrowserItemTitle(t *testing.T) {
	item := browserItem{name: "Query", badge: "OBJECT"}
	if item.Title() != "Query" {
		t.Errorf("expected title 'Query', got %q", item.Title())
	}
}

func TestBrowserItemDescription(t *testing.T) {
	item := browserItem{name: "user", desc: "12 fields"}
	if item.Description() != "12 fields" {
		t.Errorf("expected description '12 fields', got %q", item.Description())
	}
}

func TestBrowserItemFilterValue(t *testing.T) {
	item := browserItem{name: "createUser", desc: "User"}
	if item.FilterValue() != "createUser User" {
		t.Errorf("expected filter 'createUser User', got %q", item.FilterValue())
	}
}

func TestBrowserItemDrillable(t *testing.T) {
	drillable := browserItem{name: "User", target: "User"}
	if !drillable.Drillable() {
		t.Error("expected item with target to be drillable")
	}
	scalar := browserItem{name: "id"}
	if scalar.Drillable() {
		t.Error("expected item without target to not be drillable")
	}
}

func TestRootItems(t *testing.T) {
	s := testSchema()
	items := rootItems(s)

	// Should have: Query, Mutation, Variable Types (group)
	if len(items) != 3 {
		t.Fatalf("expected 3 root items, got %d", len(items))
	}
	if items[0].name != "Query" {
		t.Errorf("expected first root item 'Query', got %q", items[0].name)
	}
	if items[0].badge != "OBJECT" {
		t.Errorf("expected badge 'OBJECT', got %q", items[0].badge)
	}
	if items[0].target != "Query" {
		t.Errorf("expected target 'Query', got %q", items[0].target)
	}
	if items[0].desc != "2 fields" {
		t.Errorf("expected desc '2 fields', got %q", items[0].desc)
	}

	// Last item should be the Variable Types group
	vt := items[2]
	if vt.name != "Variable Types" {
		t.Errorf("expected 'Variable Types' group, got %q", vt.name)
	}
	if !vt.Drillable() {
		t.Error("expected Variable Types to be drillable")
	}
	if vt.desc != "2 types" {
		t.Errorf("expected desc '2 types', got %q", vt.desc)
	}
}

func TestVariableTypeItems(t *testing.T) {
	s := testSchema()
	items := variableTypeItems(s)

	foundInput := false
	foundEnum := false
	for _, item := range items {
		if item.name == "CreateUserInput" && item.badge == "INPUT_OBJECT" {
			foundInput = true
		}
		if item.name == "Role" && item.badge == "ENUM" {
			foundEnum = true
		}
	}
	if !foundInput {
		t.Error("expected CreateUserInput in variable type items")
	}
	if !foundEnum {
		t.Error("expected Role enum in variable type items")
	}
}

func TestTypeItemsObject(t *testing.T) {
	s := testSchema()
	items := typeItems(s, "User")
	if len(items) < 4 {
		t.Fatalf("expected at least 4 items for User, got %d", len(items))
	}
	if items[0].name != "implements Node" {
		t.Errorf("expected first item 'implements Node', got %q", items[0].name)
	}
	found := false
	for _, item := range items {
		if item.deprecated {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected at least one deprecated item")
	}
}

func TestTypeItemsEnum(t *testing.T) {
	s := testSchema()
	items := typeItems(s, "Role")
	if len(items) != 3 {
		t.Fatalf("expected 3 enum items, got %d", len(items))
	}
	if items[0].name != "ADMIN" {
		t.Errorf("expected 'ADMIN', got %q", items[0].name)
	}
	if !items[2].deprecated {
		t.Error("expected GUEST to be deprecated")
	}
}

func TestTypeItemsInputObject(t *testing.T) {
	s := testSchema()
	items := typeItems(s, "CreateUserInput")
	if len(items) != 2 {
		t.Fatalf("expected 2 input items, got %d", len(items))
	}
	if items[0].name != "name" {
		t.Errorf("expected 'name', got %q", items[0].name)
	}
	if items[0].desc != "String!" {
		t.Errorf("expected desc 'String!', got %q", items[0].desc)
	}
}

func TestFieldItemWithArgs(t *testing.T) {
	s := testSchema()
	items := typeItems(s, "Query")
	if len(items) == 0 {
		t.Fatal("expected items for Query")
	}
	item := items[0]
	if item.target != "User" {
		t.Errorf("expected target 'User', got %q", item.target)
	}
	if item.name == "" {
		t.Error("expected non-empty name")
	}
}

func TestFieldItemStructuredData(t *testing.T) {
	s := testSchema()
	items := typeItems(s, "Query")
	if len(items) == 0 {
		t.Fatal("expected items for Query")
	}

	// First field: user(id: ID!): User
	item := items[0]
	if item.fieldName != "user" {
		t.Errorf("expected fieldName 'user', got %q", item.fieldName)
	}
	if item.fieldArgs != "(id: ID!)" {
		t.Errorf("expected fieldArgs '(id: ID!)', got %q", item.fieldArgs)
	}
	if item.fieldType != "User" {
		t.Errorf("expected fieldType 'User', got %q", item.fieldType)
	}
	if item.fieldTypeKind != "OBJECT" {
		t.Errorf("expected fieldTypeKind 'OBJECT', got %q", item.fieldTypeKind)
	}

	// Second field: users: [User]! — no args
	item2 := items[1]
	if item2.fieldName != "users" {
		t.Errorf("expected fieldName 'users', got %q", item2.fieldName)
	}
	if item2.fieldArgs != "" {
		t.Errorf("expected empty fieldArgs, got %q", item2.fieldArgs)
	}
}

func TestFieldItemScalarTypeKind(t *testing.T) {
	s := testSchema()
	items := typeItems(s, "User")

	// Find the "name" field which returns String (SCALAR)
	for _, item := range items {
		if item.fieldName == "name" {
			if item.fieldTypeKind != "SCALAR" {
				t.Errorf("expected fieldTypeKind 'SCALAR' for name field, got %q", item.fieldTypeKind)
			}
			return
		}
	}
	t.Error("expected to find 'name' field in User type")
}

func TestRootItemsHaveNoStructuredFields(t *testing.T) {
	s := testSchema()
	items := rootItems(s)
	for _, item := range items {
		if item.fieldName != "" {
			t.Errorf("root items should not have fieldName, got %q", item.fieldName)
		}
	}
}

// --- allSearchableItems tests ---

func TestAllSearchableItemsBasic(t *testing.T) {
	s := testSchema()
	items := allSearchableItems(s)

	if len(items) == 0 {
		t.Fatal("expected non-empty searchable items")
	}

	// Collect all parent names and item names for inspection
	parents := make(map[string][]string)
	for _, si := range items {
		parents[si.parentName] = append(parents[si.parentName], si.item.name)
	}

	// Query fields: user, users
	if fields, ok := parents["Query"]; !ok {
		t.Error("expected Query fields in searchable items")
	} else {
		assertContainsField(t, fields, "Query", "user")
		assertContainsField(t, fields, "Query", "users")
	}

	// Mutation fields: createUser
	if fields, ok := parents["Mutation"]; !ok {
		t.Error("expected Mutation fields in searchable items")
	} else {
		assertContainsField(t, fields, "Mutation", "createUser")
	}

	// User fields: id, name, email, oldName
	if fields, ok := parents["User"]; !ok {
		t.Error("expected User fields in searchable items")
	} else {
		assertContainsField(t, fields, "User", "id")
		assertContainsField(t, fields, "User", "name")
		assertContainsField(t, fields, "User", "email")
		assertContainsField(t, fields, "User", "oldName")
	}

	// Role enum values: ADMIN, USER, GUEST
	if fields, ok := parents["Role"]; !ok {
		t.Error("expected Role enum values in searchable items")
	} else {
		assertContainsField(t, fields, "Role", "ADMIN")
		assertContainsField(t, fields, "Role", "USER")
		assertContainsField(t, fields, "Role", "GUEST")
	}

	// CreateUserInput fields: name, email
	if fields, ok := parents["CreateUserInput"]; !ok {
		t.Error("expected CreateUserInput fields in searchable items")
	} else {
		assertContainsField(t, fields, "CreateUserInput", "name")
		assertContainsField(t, fields, "CreateUserInput", "email")
	}
}

func TestAllSearchableItemsParentKind(t *testing.T) {
	s := testSchema()
	items := allSearchableItems(s)

	kindByParent := make(map[string]string)
	for _, si := range items {
		kindByParent[si.parentName] = si.parentKind
	}

	tests := map[string]string{
		"Query":           "OBJECT",
		"Mutation":        "OBJECT",
		"User":            "OBJECT",
		"Role":            "ENUM",
		"CreateUserInput": "INPUT_OBJECT",
	}
	for parent, expectedKind := range tests {
		if kind, ok := kindByParent[parent]; !ok {
			t.Errorf("expected parent %q in searchable items", parent)
		} else if kind != expectedKind {
			t.Errorf("expected parentKind %q for %q, got %q", expectedKind, parent, kind)
		}
	}
}

func TestAllSearchableItemsSkipsInternalTypes(t *testing.T) {
	s := &Schema{
		QueryType: &TypeRef{Name: strPtr("Query")},
		Types: []FullType{
			{Kind: "OBJECT", Name: "Query", Fields: []Field{
				{Name: "hello", Type: TypeRef{Kind: "SCALAR", Name: strPtr("String")}},
			}},
			{Kind: "SCALAR", Name: "String"},
			// Internal types should be skipped
			{Kind: "OBJECT", Name: "__Schema", Fields: []Field{
				{Name: "types", Type: TypeRef{Kind: "SCALAR", Name: strPtr("String")}},
			}},
			{Kind: "OBJECT", Name: "__Type", Fields: []Field{
				{Name: "name", Type: TypeRef{Kind: "SCALAR", Name: strPtr("String")}},
			}},
		},
	}

	items := allSearchableItems(s)

	for _, si := range items {
		if si.parentName == "__Schema" || si.parentName == "__Type" {
			t.Errorf("internal type %q should be skipped", si.parentName)
		}
	}

	if len(items) != 1 {
		t.Errorf("expected 1 searchable item (just Query.hello), got %d", len(items))
	}
}

func TestAllSearchableItemsNilSchema(t *testing.T) {
	items := allSearchableItems(nil)
	if items != nil {
		t.Error("expected nil for nil schema")
	}
}

func TestAllSearchableItemsEmptySchema(t *testing.T) {
	s := &Schema{}
	items := allSearchableItems(s)
	if len(items) != 0 {
		t.Errorf("expected 0 searchable items for empty schema, got %d", len(items))
	}
}

// TestAllSearchableItemsRecursiveTypes verifies that circular type references
// don't cause infinite recursion. This tests A→B→A patterns.
func TestAllSearchableItemsRecursiveTypes(t *testing.T) {
	s := &Schema{
		QueryType: &TypeRef{Name: strPtr("Query")},
		Types: []FullType{
			{Kind: "OBJECT", Name: "Query", Fields: []Field{
				{Name: "node", Type: TypeRef{Kind: "OBJECT", Name: strPtr("Node")}},
			}},
			// Node has a field that references itself (self-recursive)
			{Kind: "OBJECT", Name: "Node", Fields: []Field{
				{Name: "id", Type: TypeRef{Kind: "SCALAR", Name: strPtr("ID")}},
				{Name: "parent", Type: TypeRef{Kind: "OBJECT", Name: strPtr("Node")}},
				{Name: "children", Type: TypeRef{Kind: "LIST", OfType: &TypeRef{Kind: "OBJECT", Name: strPtr("Node")}}},
			}},
			{Kind: "SCALAR", Name: "ID"},
		},
	}

	items := allSearchableItems(s)

	// Should complete without infinite loop
	parents := make(map[string]int)
	for _, si := range items {
		parents[si.parentName]++
	}

	// Query has 1 field, Node has 3 fields
	if parents["Query"] != 1 {
		t.Errorf("expected 1 Query field, got %d", parents["Query"])
	}
	if parents["Node"] != 3 {
		t.Errorf("expected 3 Node fields, got %d", parents["Node"])
	}
}

// TestAllSearchableItemsMutuallyRecursiveTypes tests A→B→A mutual recursion.
func TestAllSearchableItemsMutuallyRecursiveTypes(t *testing.T) {
	s := &Schema{
		QueryType: &TypeRef{Name: strPtr("Query")},
		Types: []FullType{
			{Kind: "OBJECT", Name: "Query", Fields: []Field{
				{Name: "person", Type: TypeRef{Kind: "OBJECT", Name: strPtr("Person")}},
			}},
			// Person references Company, Company references Person
			{Kind: "OBJECT", Name: "Person", Fields: []Field{
				{Name: "name", Type: TypeRef{Kind: "SCALAR", Name: strPtr("String")}},
				{Name: "employer", Type: TypeRef{Kind: "OBJECT", Name: strPtr("Company")}},
			}},
			{Kind: "OBJECT", Name: "Company", Fields: []Field{
				{Name: "name", Type: TypeRef{Kind: "SCALAR", Name: strPtr("String")}},
				{Name: "employees", Type: TypeRef{Kind: "LIST", OfType: &TypeRef{Kind: "OBJECT", Name: strPtr("Person")}}},
				{Name: "ceo", Type: TypeRef{Kind: "OBJECT", Name: strPtr("Person")}},
			}},
			{Kind: "SCALAR", Name: "String"},
		},
	}

	items := allSearchableItems(s)

	parents := make(map[string]int)
	for _, si := range items {
		parents[si.parentName]++
	}

	// Query: 1 (person), Person: 2 (name, employer), Company: 3 (name, employees, ceo)
	if parents["Query"] != 1 {
		t.Errorf("expected 1 Query field, got %d", parents["Query"])
	}
	if parents["Person"] != 2 {
		t.Errorf("expected 2 Person fields, got %d", parents["Person"])
	}
	if parents["Company"] != 3 {
		t.Errorf("expected 3 Company fields, got %d", parents["Company"])
	}
}

// TestAllSearchableItemsDeepRecursiveChain tests A→B→C→A deep cycle.
func TestAllSearchableItemsDeepRecursiveChain(t *testing.T) {
	s := &Schema{
		QueryType: &TypeRef{Name: strPtr("Query")},
		Types: []FullType{
			{Kind: "OBJECT", Name: "Query", Fields: []Field{
				{Name: "alpha", Type: TypeRef{Kind: "OBJECT", Name: strPtr("Alpha")}},
			}},
			{Kind: "OBJECT", Name: "Alpha", Fields: []Field{
				{Name: "toBeta", Type: TypeRef{Kind: "OBJECT", Name: strPtr("Beta")}},
			}},
			{Kind: "OBJECT", Name: "Beta", Fields: []Field{
				{Name: "toGamma", Type: TypeRef{Kind: "OBJECT", Name: strPtr("Gamma")}},
			}},
			// Gamma references Alpha, completing the cycle
			{Kind: "OBJECT", Name: "Gamma", Fields: []Field{
				{Name: "toAlpha", Type: TypeRef{Kind: "OBJECT", Name: strPtr("Alpha")}},
				{Name: "value", Type: TypeRef{Kind: "SCALAR", Name: strPtr("String")}},
			}},
			{Kind: "SCALAR", Name: "String"},
		},
	}

	items := allSearchableItems(s)

	parents := make(map[string]int)
	for _, si := range items {
		parents[si.parentName]++
	}

	// Each type should appear exactly once with the correct field count
	if parents["Query"] != 1 {
		t.Errorf("expected 1 Query field, got %d", parents["Query"])
	}
	if parents["Alpha"] != 1 {
		t.Errorf("expected 1 Alpha field, got %d", parents["Alpha"])
	}
	if parents["Beta"] != 1 {
		t.Errorf("expected 1 Beta field, got %d", parents["Beta"])
	}
	if parents["Gamma"] != 2 {
		t.Errorf("expected 2 Gamma fields, got %d", parents["Gamma"])
	}
}

// TestAllSearchableItemsUnionType tests that union possible types are indexed.
func TestAllSearchableItemsUnionType(t *testing.T) {
	s := &Schema{
		QueryType: &TypeRef{Name: strPtr("Query")},
		Types: []FullType{
			{Kind: "OBJECT", Name: "Query", Fields: []Field{
				{Name: "search", Type: TypeRef{Kind: "UNION", Name: strPtr("SearchResult")}},
			}},
			{Kind: "UNION", Name: "SearchResult", PossibleTypes: []TypeRef{
				{Kind: "OBJECT", Name: strPtr("Article")},
				{Kind: "OBJECT", Name: strPtr("Author")},
			}},
			{Kind: "OBJECT", Name: "Article", Fields: []Field{
				{Name: "title", Type: TypeRef{Kind: "SCALAR", Name: strPtr("String")}},
			}},
			{Kind: "OBJECT", Name: "Author", Fields: []Field{
				{Name: "name", Type: TypeRef{Kind: "SCALAR", Name: strPtr("String")}},
			}},
			{Kind: "SCALAR", Name: "String"},
		},
	}

	items := allSearchableItems(s)

	parents := make(map[string]int)
	names := make(map[string][]string)
	for _, si := range items {
		parents[si.parentName]++
		names[si.parentName] = append(names[si.parentName], si.item.name)
	}

	// SearchResult union should have 2 possible types
	if parents["SearchResult"] != 2 {
		t.Errorf("expected 2 SearchResult items, got %d", parents["SearchResult"])
	}
	assertContainsField(t, names["SearchResult"], "SearchResult", "Article")
	assertContainsField(t, names["SearchResult"], "SearchResult", "Author")
}

// TestAllSearchableItemsInterfaceType tests that interface fields are indexed.
func TestAllSearchableItemsInterfaceType(t *testing.T) {
	s := &Schema{
		QueryType: &TypeRef{Name: strPtr("Query")},
		Types: []FullType{
			{Kind: "OBJECT", Name: "Query", Fields: []Field{
				{Name: "node", Type: TypeRef{Kind: "INTERFACE", Name: strPtr("Node")}},
			}},
			{Kind: "INTERFACE", Name: "Node", Fields: []Field{
				{Name: "id", Type: TypeRef{Kind: "NON_NULL", OfType: &TypeRef{Kind: "SCALAR", Name: strPtr("ID")}}},
				{Name: "createdAt", Type: TypeRef{Kind: "SCALAR", Name: strPtr("DateTime")}},
			}},
			{Kind: "SCALAR", Name: "ID"},
			{Kind: "SCALAR", Name: "DateTime"},
		},
	}

	items := allSearchableItems(s)

	parents := make(map[string]int)
	for _, si := range items {
		parents[si.parentName]++
	}

	if parents["Node"] != 2 {
		t.Errorf("expected 2 Node interface fields, got %d", parents["Node"])
	}

	// Verify parentKind is INTERFACE
	for _, si := range items {
		if si.parentName == "Node" && si.parentKind != "INTERFACE" {
			t.Errorf("expected parentKind INTERFACE for Node, got %q", si.parentKind)
		}
	}
}

// TestAllSearchableItemsFieldItemData verifies that field items in the index
// carry structured rendering data (fieldName, fieldArgs, fieldType, etc.).
func TestAllSearchableItemsFieldItemData(t *testing.T) {
	s := testSchema()
	items := allSearchableItems(s)

	// Find the "user" field from Query
	for _, si := range items {
		if si.parentName == "Query" && si.item.fieldName == "user" {
			if si.item.fieldArgs != "(id: ID!)" {
				t.Errorf("expected fieldArgs '(id: ID!)', got %q", si.item.fieldArgs)
			}
			if si.item.fieldType != "User" {
				t.Errorf("expected fieldType 'User', got %q", si.item.fieldType)
			}
			if si.item.fieldTypeKind != "OBJECT" {
				t.Errorf("expected fieldTypeKind 'OBJECT', got %q", si.item.fieldTypeKind)
			}
			return
		}
	}
	t.Error("expected to find Query.user in searchable items")
}

// TestAllSearchableItemsDeprecatedField verifies deprecated items are indexed.
func TestAllSearchableItemsDeprecatedField(t *testing.T) {
	s := testSchema()
	items := allSearchableItems(s)

	for _, si := range items {
		if si.parentName == "User" && si.item.fieldName == "oldName" {
			if !si.item.deprecated {
				t.Error("expected oldName to be marked deprecated")
			}
			if si.item.dimNote == "" {
				t.Error("expected deprecation note for oldName")
			}
			return
		}
	}
	t.Error("expected to find deprecated User.oldName in searchable items")
}

// TestAllSearchableItemsScalarsSkipped verifies SCALAR types are not indexed.
func TestAllSearchableItemsScalarsSkipped(t *testing.T) {
	s := testSchema()
	items := allSearchableItems(s)

	for _, si := range items {
		if si.parentName == "ID" || si.parentName == "String" {
			t.Errorf("SCALAR type %q should not appear as parent", si.parentName)
		}
	}
}

// TestAllSearchableItemsDuplicateTypes verifies that duplicate types in the
// schema are only processed once.
func TestAllSearchableItemsDuplicateTypes(t *testing.T) {
	s := &Schema{
		QueryType: &TypeRef{Name: strPtr("Query")},
		Types: []FullType{
			{Kind: "OBJECT", Name: "Query", Fields: []Field{
				{Name: "hello", Type: TypeRef{Kind: "SCALAR", Name: strPtr("String")}},
			}},
			// Same type listed twice (shouldn't happen in practice, but be safe)
			{Kind: "OBJECT", Name: "Query", Fields: []Field{
				{Name: "hello", Type: TypeRef{Kind: "SCALAR", Name: strPtr("String")}},
			}},
			{Kind: "SCALAR", Name: "String"},
		},
	}

	items := allSearchableItems(s)

	count := 0
	for _, si := range items {
		if si.parentName == "Query" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 Query item (deduped), got %d", count)
	}
}

// TestAllSearchableItemsComplexRecursive tests a realistic schema with
// multiple circular references: User→Post→Comment→User, User→User (friends).
func TestAllSearchableItemsComplexRecursive(t *testing.T) {
	s := &Schema{
		QueryType: &TypeRef{Name: strPtr("Query")},
		Types: []FullType{
			{Kind: "OBJECT", Name: "Query", Fields: []Field{
				{Name: "user", Type: TypeRef{Kind: "OBJECT", Name: strPtr("User")}},
				{Name: "feed", Type: TypeRef{Kind: "LIST", OfType: &TypeRef{Kind: "OBJECT", Name: strPtr("Post")}}},
			}},
			{Kind: "OBJECT", Name: "User", Fields: []Field{
				{Name: "id", Type: TypeRef{Kind: "SCALAR", Name: strPtr("ID")}},
				{Name: "posts", Type: TypeRef{Kind: "LIST", OfType: &TypeRef{Kind: "OBJECT", Name: strPtr("Post")}}},
				{Name: "friends", Type: TypeRef{Kind: "LIST", OfType: &TypeRef{Kind: "OBJECT", Name: strPtr("User")}}},
				{Name: "bestFriend", Type: TypeRef{Kind: "OBJECT", Name: strPtr("User")}},
			}},
			{Kind: "OBJECT", Name: "Post", Fields: []Field{
				{Name: "title", Type: TypeRef{Kind: "SCALAR", Name: strPtr("String")}},
				{Name: "author", Type: TypeRef{Kind: "OBJECT", Name: strPtr("User")}},
				{Name: "comments", Type: TypeRef{Kind: "LIST", OfType: &TypeRef{Kind: "OBJECT", Name: strPtr("Comment")}}},
			}},
			{Kind: "OBJECT", Name: "Comment", Fields: []Field{
				{Name: "text", Type: TypeRef{Kind: "SCALAR", Name: strPtr("String")}},
				{Name: "author", Type: TypeRef{Kind: "OBJECT", Name: strPtr("User")}},
				{Name: "post", Type: TypeRef{Kind: "OBJECT", Name: strPtr("Post")}},
				{Name: "replies", Type: TypeRef{Kind: "LIST", OfType: &TypeRef{Kind: "OBJECT", Name: strPtr("Comment")}}},
			}},
			{Kind: "SCALAR", Name: "ID"},
			{Kind: "SCALAR", Name: "String"},
		},
	}

	items := allSearchableItems(s)

	parents := make(map[string]int)
	for _, si := range items {
		parents[si.parentName]++
	}

	// Query: 2 (user, feed)
	if parents["Query"] != 2 {
		t.Errorf("expected 2 Query fields, got %d", parents["Query"])
	}
	// User: 4 (id, posts, friends, bestFriend)
	if parents["User"] != 4 {
		t.Errorf("expected 4 User fields, got %d", parents["User"])
	}
	// Post: 3 (title, author, comments)
	if parents["Post"] != 3 {
		t.Errorf("expected 3 Post fields, got %d", parents["Post"])
	}
	// Comment: 4 (text, author, post, replies)
	if parents["Comment"] != 4 {
		t.Errorf("expected 4 Comment fields, got %d", parents["Comment"])
	}

	// Total: 2+4+3+4 = 13
	if len(items) != 13 {
		t.Errorf("expected 13 total searchable items, got %d", len(items))
	}
}

// TestAllSearchableItemsMixedKinds tests a schema with all kind types together.
func TestAllSearchableItemsMixedKinds(t *testing.T) {
	s := &Schema{
		QueryType: &TypeRef{Name: strPtr("Query")},
		Types: []FullType{
			{Kind: "OBJECT", Name: "Query", Fields: []Field{
				{Name: "search", Type: TypeRef{Kind: "UNION", Name: strPtr("Result")}},
			}},
			{Kind: "UNION", Name: "Result", PossibleTypes: []TypeRef{
				{Kind: "OBJECT", Name: strPtr("Article")},
			}},
			{Kind: "OBJECT", Name: "Article", Fields: []Field{
				{Name: "title", Type: TypeRef{Kind: "SCALAR", Name: strPtr("String")}},
				{Name: "status", Type: TypeRef{Kind: "ENUM", Name: strPtr("Status")}},
			}},
			{Kind: "ENUM", Name: "Status", EnumValues: []EnumValue{
				{Name: "DRAFT"},
				{Name: "PUBLISHED"},
			}},
			{Kind: "INPUT_OBJECT", Name: "ArticleInput", InputFields: []InputValue{
				{Name: "title", Type: TypeRef{Kind: "SCALAR", Name: strPtr("String")}},
				{Name: "status", Type: TypeRef{Kind: "ENUM", Name: strPtr("Status")}},
			}},
			{Kind: "INTERFACE", Name: "Timestamped", Fields: []Field{
				{Name: "createdAt", Type: TypeRef{Kind: "SCALAR", Name: strPtr("DateTime")}},
			}},
			{Kind: "SCALAR", Name: "String"},
			{Kind: "SCALAR", Name: "DateTime"},
		},
	}

	items := allSearchableItems(s)

	parents := make(map[string]int)
	kinds := make(map[string]string)
	for _, si := range items {
		parents[si.parentName]++
		kinds[si.parentName] = si.parentKind
	}

	expected := map[string]struct{ count int; kind string }{
		"Query":        {1, "OBJECT"},
		"Result":       {1, "UNION"},
		"Article":      {2, "OBJECT"},
		"Status":       {2, "ENUM"},
		"ArticleInput": {2, "INPUT_OBJECT"},
		"Timestamped":  {1, "INTERFACE"},
	}

	for name, exp := range expected {
		if parents[name] != exp.count {
			t.Errorf("expected %d items for %s, got %d", exp.count, name, parents[name])
		}
		if kinds[name] != exp.kind {
			t.Errorf("expected kind %q for %s, got %q", exp.kind, name, kinds[name])
		}
	}
}

func assertContainsField(t *testing.T, fields []string, parent, name string) {
	t.Helper()
	for _, f := range fields {
		if f == name || (len(f) > len(name) && f[:len(name)] == name) {
			return
		}
	}
	t.Errorf("expected %s.%s in searchable items, got fields: %v", parent, name, fields)
}
