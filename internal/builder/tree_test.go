package builder

import (
	"testing"

	"github.com/qraqula/qla/internal/schema"
)

// Helper to create a *string (TypeRef.Name requires a pointer).
func strPtr(s string) *string { return &s }

// nonNull wraps a TypeRef in a NON_NULL wrapper.
func nonNull(inner schema.TypeRef) schema.TypeRef {
	return schema.TypeRef{Kind: "NON_NULL", OfType: &inner}
}

// listOf wraps a TypeRef in a LIST wrapper.
func listOf(inner schema.TypeRef) schema.TypeRef {
	return schema.TypeRef{Kind: "LIST", OfType: &inner}
}

// namedRef creates a named TypeRef (SCALAR or OBJECT, depending on usage).
func namedRef(name string) schema.TypeRef {
	return schema.TypeRef{Kind: "NAMED", Name: strPtr(name)}
}

// countriesSchema builds a mock schema mimicking the Countries API
// (countries.trevorblades.com) with Query, Country, Continent, and Language types.
func countriesSchema() *schema.Schema {
	s := &schema.Schema{
		QueryType: &schema.TypeRef{Kind: "NAMED", Name: strPtr("Query")},
		Types: []schema.FullType{
			{
				Kind: "OBJECT",
				Name: "Query",
				Fields: []schema.Field{
					{
						Name: "countries",
						Type: nonNull(listOf(nonNull(namedRef("Country")))), // [Country!]!
					},
					{
						Name: "country",
						Type: namedRef("Country"), // Country (nullable)
						Args: []schema.InputValue{
							{
								Name: "code",
								Type: nonNull(namedRef("ID")), // ID!
							},
						},
					},
				},
			},
			{
				Kind: "OBJECT",
				Name: "Country",
				Fields: []schema.Field{
					{Name: "code", Type: nonNull(namedRef("String"))},                            // String!
					{Name: "name", Type: nonNull(namedRef("String"))},                            // String!
					{Name: "continent", Type: nonNull(namedRef("Continent"))},                    // Continent!
					{Name: "languages", Type: nonNull(listOf(nonNull(namedRef("Language"))))},    // [Language!]!
					{Name: "capital", Type: namedRef("String")},                                  // String (nullable)
				},
			},
			{
				Kind: "OBJECT",
				Name: "Continent",
				Fields: []schema.Field{
					{Name: "code", Type: nonNull(namedRef("String"))},                          // String!
					{Name: "name", Type: nonNull(namedRef("String"))},                          // String!
					{Name: "countries", Type: nonNull(listOf(nonNull(namedRef("Country"))))},   // [Country!]!
				},
			},
			{
				Kind: "OBJECT",
				Name: "Language",
				Fields: []schema.Field{
					{Name: "code", Type: nonNull(namedRef("String"))},                          // String!
					{Name: "name", Type: nonNull(namedRef("String"))},                          // String!
					{Name: "countries", Type: nonNull(listOf(nonNull(namedRef("Country"))))},   // [Country!]!
				},
			},
			{Kind: "SCALAR", Name: "String"},
			{Kind: "SCALAR", Name: "ID"},
			{Kind: "SCALAR", Name: "Boolean"},
		},
	}
	return s
}

func TestBuildTreeFromField_CountriesField(t *testing.T) {
	s := countriesSchema()
	queryType := s.TypeByName("Query")
	if queryType == nil {
		t.Fatal("Query type not found in schema")
	}

	// Find the "countries" field on Query.
	var countriesField schema.Field
	found := false
	for _, f := range queryType.Fields {
		if f.Name == "countries" {
			countriesField = f
			found = true
			break
		}
	}
	if !found {
		t.Fatal("countries field not found on Query type")
	}

	root := BuildTreeFromField(s, countriesField)

	// Root should represent the countries field.
	if root.Name != "countries" {
		t.Errorf("root name = %q, want %q", root.Name, "countries")
	}
	if root.TypeName != "Country" {
		t.Errorf("root TypeName = %q, want %q", root.TypeName, "Country")
	}
	if root.TypeKind != "OBJECT" {
		t.Errorf("root TypeKind = %q, want %q", root.TypeKind, "OBJECT")
	}
	if root.IsLeaf {
		t.Error("root should NOT be a leaf (it is an OBJECT type)")
	}

	// Root should be loaded with children (the fields of Country).
	if !root.Loaded {
		t.Error("root should be loaded after BuildTreeFromField")
	}
	if len(root.Children) == 0 {
		t.Fatal("root should have children (Country fields)")
	}

	// Country has 5 fields: code, name, continent, languages, capital
	if len(root.Children) != 5 {
		t.Errorf("root children count = %d, want 5", len(root.Children))
	}
}

func TestScalarFieldsAreLeaf(t *testing.T) {
	s := countriesSchema()
	queryType := s.TypeByName("Query")
	var countriesField schema.Field
	for _, f := range queryType.Fields {
		if f.Name == "countries" {
			countriesField = f
			break
		}
	}

	root := BuildTreeFromField(s, countriesField)

	// Find scalar children.
	scalarNames := map[string]bool{"code": true, "name": true, "capital": true}
	for _, child := range root.Children {
		if scalarNames[child.Name] {
			if !child.IsLeaf {
				t.Errorf("field %q (type %s) should be a leaf", child.Name, child.TypeKind)
			}
			if child.TypeKind != "SCALAR" {
				t.Errorf("field %q TypeKind = %q, want SCALAR", child.Name, child.TypeKind)
			}
		}
	}
}

func TestContinentFieldIsExpandable(t *testing.T) {
	s := countriesSchema()
	queryType := s.TypeByName("Query")
	var countriesField schema.Field
	for _, f := range queryType.Fields {
		if f.Name == "countries" {
			countriesField = f
			break
		}
	}

	root := BuildTreeFromField(s, countriesField)

	// Find the continent child.
	var continentNode *TreeNode
	for _, child := range root.Children {
		if child.Name == "continent" {
			continentNode = child
			break
		}
	}
	if continentNode == nil {
		t.Fatal("continent child not found")
	}

	// continent has type Continent (OBJECT) -- it must NOT be a leaf.
	if continentNode.IsLeaf {
		t.Error("continent should NOT be a leaf (type is OBJECT: Continent)")
	}
	if continentNode.TypeKind != "OBJECT" {
		t.Errorf("continent TypeKind = %q, want OBJECT", continentNode.TypeKind)
	}
	if continentNode.TypeName != "Continent" {
		t.Errorf("continent TypeName = %q, want Continent", continentNode.TypeName)
	}

	// Since BuildTreeFromField preloads 2 levels, continent should already be loaded.
	if !continentNode.Loaded {
		t.Error("continent should be loaded (preloaded by BuildTreeFromField)")
	}

	// continent's children should include code, name, and countries.
	if len(continentNode.Children) < 2 {
		t.Errorf("continent should have at least 2 children (code, name), got %d", len(continentNode.Children))
	}
}

func TestLanguagesFieldIsExpandable(t *testing.T) {
	s := countriesSchema()
	queryType := s.TypeByName("Query")
	var countriesField schema.Field
	for _, f := range queryType.Fields {
		if f.Name == "countries" {
			countriesField = f
			break
		}
	}

	root := BuildTreeFromField(s, countriesField)

	// Find the languages child.
	var languagesNode *TreeNode
	for _, child := range root.Children {
		if child.Name == "languages" {
			languagesNode = child
			break
		}
	}
	if languagesNode == nil {
		t.Fatal("languages child not found")
	}

	// languages has type [Language!]! -> Language (OBJECT) -- must NOT be a leaf.
	if languagesNode.IsLeaf {
		t.Error("languages should NOT be a leaf (type is OBJECT: Language)")
	}
	if languagesNode.TypeKind != "OBJECT" {
		t.Errorf("languages TypeKind = %q, want OBJECT", languagesNode.TypeKind)
	}
	if languagesNode.TypeName != "Language" {
		t.Errorf("languages TypeName = %q, want Language", languagesNode.TypeName)
	}

	// Preloaded by BuildTreeFromField (2 levels), so should be loaded.
	if !languagesNode.Loaded {
		t.Error("languages should be loaded (preloaded by BuildTreeFromField)")
	}
	// Language has 3 fields: code, name, countries
	if len(languagesNode.Children) != 3 {
		t.Errorf("languages children count = %d, want 3", len(languagesNode.Children))
	}
}

func TestContinentChildrenAfterExpand(t *testing.T) {
	s := countriesSchema()
	queryType := s.TypeByName("Query")
	var countriesField schema.Field
	for _, f := range queryType.Fields {
		if f.Name == "countries" {
			countriesField = f
			break
		}
	}

	root := BuildTreeFromField(s, countriesField)

	// Find the continent child.
	var continentNode *TreeNode
	for _, child := range root.Children {
		if child.Name == "continent" {
			continentNode = child
			break
		}
	}
	if continentNode == nil {
		t.Fatal("continent child not found")
	}

	// Expand continent and ensure children are loaded.
	continentNode.Expanded = true
	LoadChildren(s, continentNode) // no-op if already loaded, but be explicit

	// Verify children include at least code and name (both scalar).
	childNames := make(map[string]bool)
	for _, child := range continentNode.Children {
		childNames[child.Name] = true
	}
	if !childNames["code"] {
		t.Error("continent should have a 'code' child")
	}
	if !childNames["name"] {
		t.Error("continent should have a 'name' child")
	}

	// code and name should be leaf (SCALAR) nodes.
	for _, child := range continentNode.Children {
		if child.Name == "code" || child.Name == "name" {
			if !child.IsLeaf {
				t.Errorf("continent child %q should be a leaf", child.Name)
			}
		}
	}
}

func TestFlattenVisible(t *testing.T) {
	s := countriesSchema()
	queryType := s.TypeByName("Query")
	var countriesField schema.Field
	for _, f := range queryType.Fields {
		if f.Name == "countries" {
			countriesField = f
			break
		}
	}

	root := BuildTreeFromField(s, countriesField)

	// Initially root is expanded, but none of the children are expanded.
	// FlattenVisible should return only direct children of root.
	flat := FlattenVisible(root)
	if len(flat) != 5 {
		t.Errorf("FlattenVisible with no children expanded: got %d, want 5", len(flat))
	}

	// Expand continent node.
	for _, child := range root.Children {
		if child.Name == "continent" {
			child.Expanded = true
			break
		}
	}

	flat = FlattenVisible(root)
	// Should now include continent's children too.
	if len(flat) <= 5 {
		t.Errorf("FlattenVisible after expanding continent: got %d, want > 5", len(flat))
	}

	// Verify the continent's children appear at depth 1.
	foundContinentChild := false
	for _, fn := range flat {
		if fn.Node.Parent != nil && fn.Node.Parent.Name == "continent" {
			if fn.Depth != 1 {
				t.Errorf("continent child %q depth = %d, want 1", fn.Node.Name, fn.Depth)
			}
			foundContinentChild = true
		}
	}
	if !foundContinentChild {
		t.Error("expected to find continent's children in flattened output")
	}
}

func TestToggleSelected(t *testing.T) {
	s := countriesSchema()
	queryType := s.TypeByName("Query")
	var countriesField schema.Field
	for _, f := range queryType.Fields {
		if f.Name == "countries" {
			countriesField = f
			break
		}
	}

	root := BuildTreeFromField(s, countriesField)

	// Find a scalar child (code) and deselect it.
	var codeNode *TreeNode
	for _, child := range root.Children {
		if child.Name == "code" {
			codeNode = child
			break
		}
	}
	if codeNode == nil {
		t.Fatal("code child not found")
	}

	// Initially selected (inherits from BuildTreeFromField default).
	// Actually, children are NOT auto-selected by BuildTreeFromField --
	// only the root is. Let's verify initial state first.
	initialSelected := codeNode.Selected

	// Toggle: if not selected, selecting should also select ancestors.
	ToggleSelected(codeNode)
	if codeNode.Selected == initialSelected {
		t.Error("ToggleSelected should flip the selected state")
	}

	// Toggle back.
	ToggleSelected(codeNode)
	if codeNode.Selected != initialSelected {
		t.Error("second ToggleSelected should restore original state")
	}
}

func TestToggleChildrenSelected(t *testing.T) {
	s := countriesSchema()
	queryType := s.TypeByName("Query")
	var countriesField schema.Field
	for _, f := range queryType.Fields {
		if f.Name == "countries" {
			countriesField = f
			break
		}
	}

	root := BuildTreeFromField(s, countriesField)

	// Initially no children are selected.
	if HasSelectedChildren(root) {
		t.Log("root initially has selected children")
	}

	// Select all children.
	ToggleChildrenSelected(s, root)

	// All immediate children should now be selected.
	for _, child := range root.Children {
		if !child.Selected {
			t.Errorf("after ToggleChildrenSelected, child %q should be selected", child.Name)
		}
	}
	if !HasSelectedChildren(root) {
		t.Error("HasSelectedChildren should return true after selecting all children")
	}

	// Toggle again: should deselect all children.
	ToggleChildrenSelected(s, root)
	for _, child := range root.Children {
		if child.Selected {
			t.Errorf("after second ToggleChildrenSelected, child %q should be deselected", child.Name)
		}
	}
	if HasSelectedChildren(root) {
		t.Error("HasSelectedChildren should return false after deselecting all children")
	}
}

func TestHasSelectedChildren(t *testing.T) {
	s := countriesSchema()
	queryType := s.TypeByName("Query")
	var countriesField schema.Field
	for _, f := range queryType.Fields {
		if f.Name == "countries" {
			countriesField = f
			break
		}
	}

	root := BuildTreeFromField(s, countriesField)

	// Deselect all children first.
	for _, child := range root.Children {
		child.Selected = false
	}
	if HasSelectedChildren(root) {
		t.Error("HasSelectedChildren should be false when no children are selected")
	}

	// Select just one child.
	root.Children[0].Selected = true
	if !HasSelectedChildren(root) {
		t.Error("HasSelectedChildren should be true when at least one child is selected")
	}
}

func TestCyclicFieldsAreExpandable(t *testing.T) {
	s := countriesSchema()
	queryType := s.TypeByName("Query")
	var countriesField schema.Field
	for _, f := range queryType.Fields {
		if f.Name == "countries" {
			countriesField = f
			break
		}
	}

	root := BuildTreeFromField(s, countriesField)

	// Root is typed as Country. Continent has a "countries" field that points back
	// to [Country!]!. Cyclic fields should NOT be forced to leaf — they should be
	// expandable up to maxDepth.
	var continentNode *TreeNode
	for _, child := range root.Children {
		if child.Name == "continent" {
			continentNode = child
			break
		}
	}
	if continentNode == nil {
		t.Fatal("continent child not found")
	}

	// Find the "countries" field on Continent.
	var countriesOnContinent *TreeNode
	for _, child := range continentNode.Children {
		if child.Name == "countries" {
			countriesOnContinent = child
			break
		}
	}
	if countriesOnContinent == nil {
		t.Fatal("Continent should have a 'countries' child")
	}

	// Cyclic fields should be expandable (not forced to leaf).
	if countriesOnContinent.IsLeaf {
		t.Error("Continent.countries should NOT be a leaf — cyclic fields should be expandable")
	}

	// Expanding it should load children (Country's fields again).
	EnsureChildrenReady(s, countriesOnContinent)
	countriesOnContinent.Expanded = true
	if len(countriesOnContinent.Children) == 0 {
		t.Error("Continent.countries should have children after EnsureChildrenReady")
	}

	// Verify the cyclic field can be expanded multiple levels deep
	// until hitting maxDepth.
	var deepNode *TreeNode
	for _, child := range countriesOnContinent.Children {
		if child.Name == "continent" {
			deepNode = child
			break
		}
	}
	if deepNode == nil {
		t.Fatal("countries.continent child not found at depth 3")
	}
	if deepNode.IsLeaf {
		t.Error("depth-3 continent should still be expandable")
	}
}

func TestDeepCyclicExpansion(t *testing.T) {
	s := countriesSchema()
	queryType := s.TypeByName("Query")
	var countriesField schema.Field
	for _, f := range queryType.Fields {
		if f.Name == "countries" {
			countriesField = f
			break
		}
	}

	root := BuildTreeFromField(s, countriesField)

	// Drill down through cycles 20 levels deep — no depth limit should stop us.
	node := root
	for depth := 0; depth < 20; depth++ {
		EnsureChildrenReady(s, node)
		node.Expanded = true

		if node.IsLeaf {
			t.Errorf("node became leaf at depth %d — there should be no depth limit", node.Depth)
			break
		}

		// Find a non-leaf child to drill into
		var next *TreeNode
		for _, child := range node.Children {
			if !child.IsLeaf {
				next = child
				break
			}
		}
		if next == nil {
			t.Errorf("no non-leaf child found at depth %d", node.Depth)
			break
		}
		node = next
	}

	// After 20 levels of expansion, the last node should still be expandable
	if node.IsLeaf {
		t.Errorf("node at depth %d should still be expandable (no depth limit)", node.Depth)
	}
}

// TestExactUserPath simulates the exact expansion path the user reported as broken:
// countries(D0) → continent(D1) → countries(D2) → languages(D3) → countries(D4)
// The countries field at D4 must be expandable (IsLeaf=false).
func TestExactUserPath(t *testing.T) {
	s := countriesSchema()
	queryType := s.TypeByName("Query")
	var countriesField schema.Field
	for _, f := range queryType.Fields {
		if f.Name == "countries" {
			countriesField = f
			break
		}
	}

	root := BuildTreeFromField(s, countriesField)

	// Step 1: root is countries (Country, D0) — already expanded by BuildTreeFromField
	if root.TypeName != "Country" || root.Depth != 0 {
		t.Fatalf("root: TypeName=%q Depth=%d, want Country/0", root.TypeName, root.Depth)
	}

	// Step 2: Find and expand continent (D1)
	var continent1 *TreeNode
	for _, c := range root.Children {
		if c.Name == "continent" {
			continent1 = c
			break
		}
	}
	if continent1 == nil {
		t.Fatal("continent not found at D1")
	}
	if continent1.IsLeaf {
		t.Fatal("continent at D1 should NOT be a leaf")
	}
	EnsureChildrenReady(s, continent1)
	continent1.Expanded = true
	t.Logf("continent D1: Depth=%d TypeName=%s Loaded=%v IsLeaf=%v Children=%d",
		continent1.Depth, continent1.TypeName, continent1.Loaded, continent1.IsLeaf, len(continent1.Children))

	// Step 3: Find and expand countries (D2, under continent)
	var countries2 *TreeNode
	for _, c := range continent1.Children {
		if c.Name == "countries" {
			countries2 = c
			break
		}
	}
	if countries2 == nil {
		t.Fatal("countries not found at D2 under continent")
	}
	if countries2.IsLeaf {
		t.Fatal("countries at D2 should NOT be a leaf")
	}
	EnsureChildrenReady(s, countries2)
	countries2.Expanded = true
	t.Logf("countries D2: Depth=%d TypeName=%s Loaded=%v IsLeaf=%v Children=%d",
		countries2.Depth, countries2.TypeName, countries2.Loaded, countries2.IsLeaf, len(countries2.Children))

	// Step 4: Find and expand languages (D3, under countries D2)
	var languages3 *TreeNode
	for _, c := range countries2.Children {
		if c.Name == "languages" {
			languages3 = c
			break
		}
	}
	if languages3 == nil {
		t.Fatal("languages not found at D3 under countries")
	}
	if languages3.IsLeaf {
		t.Fatal("languages at D3 should NOT be a leaf")
	}
	EnsureChildrenReady(s, languages3)
	languages3.Expanded = true
	t.Logf("languages D3: Depth=%d TypeName=%s Loaded=%v IsLeaf=%v Children=%d",
		languages3.Depth, languages3.TypeName, languages3.Loaded, languages3.IsLeaf, len(languages3.Children))

	// Step 5: Find countries (D4, under languages D3) — THIS IS THE FIELD THE USER SAYS DOESN'T EXPAND
	var countries4 *TreeNode
	for _, c := range languages3.Children {
		if c.Name == "countries" {
			countries4 = c
			break
		}
	}
	if countries4 == nil {
		t.Fatal("countries not found at D4 under languages")
	}
	t.Logf("countries D4: Depth=%d TypeName=%s Loaded=%v IsLeaf=%v Children=%d",
		countries4.Depth, countries4.TypeName, countries4.Loaded, countries4.IsLeaf, len(countries4.Children))

	// THIS IS THE KEY ASSERTION: countries at D4 must NOT be a leaf
	if countries4.IsLeaf {
		t.Errorf("countries at D4 (Depth=%d) should NOT be a leaf — Country is an OBJECT type", countries4.Depth)
	}

	// Expanding it should work and produce children
	EnsureChildrenReady(s, countries4)
	countries4.Expanded = true
	if len(countries4.Children) == 0 {
		t.Error("countries at D4 should have children after expansion")
	}
	t.Logf("countries D4 after expand: Children=%d", len(countries4.Children))

	// Verify the flat list includes countries4's children
	flat := FlattenVisible(root)
	foundChildOfCountries4 := false
	for _, fn := range flat {
		if fn.Node.Parent == countries4 {
			foundChildOfCountries4 = true
			break
		}
	}
	if !foundChildOfCountries4 {
		t.Error("countries4's children should appear in the flat list after expansion")
	}
}

func TestCountryFieldWithArgs(t *testing.T) {
	s := countriesSchema()
	queryType := s.TypeByName("Query")
	var countryField schema.Field
	for _, f := range queryType.Fields {
		if f.Name == "country" {
			countryField = f
			break
		}
	}

	root := BuildTreeFromField(s, countryField)

	if root.Name != "country" {
		t.Errorf("root name = %q, want %q", root.Name, "country")
	}

	// country field has one argument: code (ID!)
	if len(root.Args) != 1 {
		t.Fatalf("root args count = %d, want 1", len(root.Args))
	}
	if root.Args[0].Name != "code" {
		t.Errorf("root arg name = %q, want %q", root.Args[0].Name, "code")
	}

	// TypeName should be Country (nullable return, but named type is still Country).
	if root.TypeName != "Country" {
		t.Errorf("root TypeName = %q, want Country", root.TypeName)
	}

	// Should have the same children as the countries field.
	if len(root.Children) != 5 {
		t.Errorf("root children count = %d, want 5", len(root.Children))
	}
}
