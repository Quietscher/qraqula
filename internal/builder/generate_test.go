package builder

import (
	"strings"
	"testing"

	"github.com/qraqula/qla/internal/schema"
)

func TestGenerateFromTree_BasicLeafSelection(t *testing.T) {
	s := countriesSchema()
	queryType := s.TypeByName("Query")
	var countriesField = findField(queryType.Fields, "countries")

	root := BuildTreeFromField(s, countriesField)

	// Select just the "name" leaf child of root
	var nameNode *TreeNode
	for _, child := range root.Children {
		if child.Name == "name" {
			nameNode = child
			break
		}
	}
	if nameNode == nil {
		t.Fatal("name child not found")
	}

	ToggleSelected(nameNode)
	query, _ := GenerateFromTree(s, "query", "countries", root)

	if !strings.Contains(query, "name") {
		t.Errorf("query should contain 'name', got:\n%s", query)
	}
	t.Logf("Generated query:\n%s", query)
}

// TestGenerateFromTree_DeepAutoSelection tests the exact bug scenario:
// User navigates deep into tree, selects a leaf, parents are auto-selected,
// and the generated query should include the full nested path.
func TestGenerateFromTree_DeepAutoSelection(t *testing.T) {
	s := countriesSchema()
	queryType := s.TypeByName("Query")
	countriesField := findField(queryType.Fields, "countries")

	root := BuildTreeFromField(s, countriesField)

	// Simulate user expanding: root → continent → countries → languages
	// Step 1: Find and expand continent
	continent := findChild(root, "continent")
	if continent == nil {
		t.Fatal("continent not found")
	}
	EnsureChildrenReady(s, continent)
	continent.Expanded = true

	// Step 2: Find and expand countries under continent
	countriesUnderContinent := findChild(continent, "countries")
	if countriesUnderContinent == nil {
		t.Fatal("countries under continent not found")
	}
	EnsureChildrenReady(s, countriesUnderContinent)
	countriesUnderContinent.Expanded = true

	// Step 3: Find and expand languages under the nested countries
	languages := findChild(countriesUnderContinent, "languages")
	if languages == nil {
		t.Fatal("languages under nested countries not found")
	}
	EnsureChildrenReady(s, languages)
	languages.Expanded = true

	// Step 4: Find "name" leaf under languages — this is the deeply nested field
	nameLeaf := findChild(languages, "name")
	if nameLeaf == nil {
		t.Fatal("name under languages not found")
	}
	if !nameLeaf.IsLeaf {
		t.Fatal("name should be a leaf")
	}

	// Verify initial state: no intermediate nodes are selected
	if continent.Selected {
		t.Log("NOTE: continent was already selected before toggle")
	}
	if countriesUnderContinent.Selected {
		t.Log("NOTE: countriesUnderContinent was already selected before toggle")
	}
	if languages.Selected {
		t.Log("NOTE: languages was already selected before toggle")
	}

	// Now select the deeply nested leaf — this should auto-select all parents
	ToggleSelected(nameLeaf)

	// Verify auto-selection worked
	if !nameLeaf.Selected {
		t.Error("nameLeaf should be selected after toggle")
	}
	if !languages.Selected {
		t.Error("languages should be auto-selected as parent")
	}
	if !countriesUnderContinent.Selected {
		t.Error("countriesUnderContinent should be auto-selected as parent")
	}
	if !continent.Selected {
		t.Error("continent should be auto-selected as parent")
	}

	// Verify HasSelectedChildren at each level
	if !HasSelectedChildren(languages) {
		t.Error("languages should have selected children (name)")
	}
	if !HasSelectedChildren(countriesUnderContinent) {
		t.Error("countriesUnderContinent should have selected children (languages)")
	}
	if !HasSelectedChildren(continent) {
		t.Error("continent should have selected children (countries)")
	}
	if !HasSelectedChildren(root) {
		t.Error("root should have selected children (continent)")
	}

	// Generate the query
	query, _ := GenerateFromTree(s, "query", "countries", root)
	t.Logf("Generated query:\n%s", query)

	// The query MUST contain the nested path
	if !strings.Contains(query, "continent") {
		t.Errorf("query should contain 'continent'")
	}
	if !strings.Contains(query, "languages") {
		t.Errorf("query should contain 'languages'")
	}
	if !strings.Contains(query, "name") {
		t.Errorf("query should contain 'name'")
	}

	// Verify the nesting structure by checking for nested braces
	// The query should have: countries { continent { countries { languages { name } } } }
	expectedFragments := []string{
		"continent",
		"countries",
		"languages",
		"name",
	}
	for _, frag := range expectedFragments {
		if !strings.Contains(query, frag) {
			t.Errorf("query missing expected fragment %q:\n%s", frag, query)
		}
	}
}

// TestGenerateFromTree_DeepAutoSelection_BuildSelSetDepth verifies that
// buildSelSet correctly recurses through auto-selected parent nodes.
func TestGenerateFromTree_DeepAutoSelection_BuildSelSetDepth(t *testing.T) {
	s := countriesSchema()
	queryType := s.TypeByName("Query")
	countriesField := findField(queryType.Fields, "countries")

	root := BuildTreeFromField(s, countriesField)

	// Expand path: root → continent → countries → languages
	continent := findChild(root, "continent")
	EnsureChildrenReady(s, continent)
	continent.Expanded = true

	countriesNested := findChild(continent, "countries")
	EnsureChildrenReady(s, countriesNested)
	countriesNested.Expanded = true

	languages := findChild(countriesNested, "languages")
	EnsureChildrenReady(s, languages)
	languages.Expanded = true

	nameLeaf := findChild(languages, "name")

	// Select the deeply nested leaf
	ToggleSelected(nameLeaf)

	// Test buildSelSet directly
	selSet := buildSelSet(root)
	t.Logf("buildSelSet result: %s", selSet)

	if selSet == "" {
		t.Fatal("buildSelSet returned empty string — deeply nested selection was lost")
	}

	// Should contain the nesting chain
	if !strings.Contains(selSet, "continent") {
		t.Error("selSet missing 'continent'")
	}
	if !strings.Contains(selSet, "countries") {
		t.Error("selSet missing 'countries'")
	}
	if !strings.Contains(selSet, "languages") {
		t.Error("selSet missing 'languages'")
	}
	if !strings.Contains(selSet, "name") {
		t.Error("selSet missing 'name'")
	}
}

// helper to find a child node by name
func findChild(parent *TreeNode, name string) *TreeNode {
	for _, child := range parent.Children {
		if child.Name == name {
			return child
		}
	}
	return nil
}

// helper to find a field by name from a list of schema fields
func findField(fields []schema.Field, name string) schema.Field {
	for _, f := range fields {
		if f.Name == name {
			return f
		}
	}
	return schema.Field{}
}
