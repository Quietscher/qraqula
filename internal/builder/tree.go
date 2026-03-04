package builder

import "github.com/qraqula/qla/internal/schema"

// TreeNode represents a field in the query builder tree.
type TreeNode struct {
	Name        string
	TypeName    string // underlying named type (e.g. "Country")
	TypeDisplay string // full display (e.g. "[Country!]!")
	TypeKind    string // SCALAR, OBJECT, UNION, INTERFACE, ENUM, INPUT_OBJECT
	Description string
	IsLeaf      bool // true for scalar/enum fields (no children)
	IsSpread    bool // true for inline fragment "... on Type"

	Args   []schema.InputValue
	Parent *TreeNode

	Children []*TreeNode
	Depth    int

	Selected bool
	Expanded bool
	Loaded   bool // children have been loaded from schema

	ArgValues     map[string]bool   // which args are enabled (by name)
	ArgEdited     map[string]string // edited arg values (by name)
	AncestorTypes map[string]bool   // type names in the ancestor chain (informational)
}

// FlatNode is a visible tree node with its visual depth, used for rendering.
type FlatNode struct {
	Node  *TreeNode
	Depth int
}

// BuildTreeFromField creates a root TreeNode for a top-level operation field.
// It preloads two levels of children so the tree is immediately navigable.
func BuildTreeFromField(s *schema.Schema, field schema.Field) *TreeNode {
	namedType := field.Type.NamedType()
	kind := resolveTypeKind(s, namedType)

	root := &TreeNode{
		Name:          field.Name,
		TypeName:      namedType,
		TypeDisplay:   field.Type.DisplayName(),
		TypeKind:      kind,
		Description:   field.Description,
		IsLeaf:        kind == "SCALAR" || kind == "ENUM",
		Args:          field.Args,
		Depth:         0,
		Selected:      true,
		Expanded:      true,
		Loaded:        false,
		ArgValues:     make(map[string]bool),
		AncestorTypes: make(map[string]bool),
	}

	EnsureChildrenReady(s, root)
	return root
}

// LoadChildren populates a node's children from the schema, if not already loaded.
// Children are loaded dynamically on expand with no depth limit.
func LoadChildren(s *schema.Schema, node *TreeNode) {
	if node.Loaded || node.IsLeaf {
		return
	}
	node.Loaded = true

	t := s.TypeByName(node.TypeName)
	if t == nil {
		node.IsLeaf = true
		return
	}

	// Track this type in ancestor chain
	ancestors := copyAncestors(node.AncestorTypes)
	ancestors[node.TypeName] = true

	switch t.Kind {
	case "OBJECT", "INTERFACE":
		for _, f := range t.Fields {
			child := fieldToNode(s, f, node, ancestors)
			node.Children = append(node.Children, child)
		}
	case "UNION":
		// Add __typename as a selectable leaf
		typenameNode := &TreeNode{
			Name:          "__typename",
			TypeName:      "String",
			TypeDisplay:   "String!",
			TypeKind:      "SCALAR",
			IsLeaf:        true,
			Parent:        node,
			Depth:         node.Depth + 1,
			ArgValues:     make(map[string]bool),
			AncestorTypes: ancestors,
		}
		node.Children = append(node.Children, typenameNode)

		// Add inline fragment nodes for each possible type
		for _, pt := range t.PossibleTypes {
			ptName := pt.NamedType()
			if ptName == "" {
				continue
			}
			spreadNode := &TreeNode{
				Name:          "... on " + ptName,
				TypeName:      ptName,
				TypeDisplay:   ptName,
				TypeKind:      "OBJECT",
				IsSpread:      true,
				Parent:        node,
				Depth:         node.Depth + 1,
				ArgValues:     make(map[string]bool),
				AncestorTypes: ancestors,
			}
			node.Children = append(node.Children, spreadNode)
		}
	}

	// Safety: if loading produced no children, mark as leaf
	if len(node.Children) == 0 {
		node.IsLeaf = true
	}
}

// EnsureChildrenReady loads a node's children and one level of grandchildren,
// so the UI can show which children are expandable.
func EnsureChildrenReady(s *schema.Schema, node *TreeNode) {
	if node.IsLeaf {
		return
	}
	if !node.Loaded {
		LoadChildren(s, node)
	}
	for _, child := range node.Children {
		if !child.IsLeaf && !child.Loaded {
			LoadChildren(s, child)
		}
	}
}

// fieldToNode converts a schema Field into a TreeNode.
func fieldToNode(s *schema.Schema, f schema.Field, parent *TreeNode, ancestors map[string]bool) *TreeNode {
	namedType := f.Type.NamedType()
	kind := resolveTypeKind(s, namedType)
	isLeaf := kind == "SCALAR" || kind == "ENUM"

	return &TreeNode{
		Name:          f.Name,
		TypeName:      namedType,
		TypeDisplay:   f.Type.DisplayName(),
		TypeKind:      kind,
		Description:   f.Description,
		IsLeaf:        isLeaf,
		Args:          f.Args,
		Parent:        parent,
		Depth:         parent.Depth + 1,
		ArgValues:     make(map[string]bool),
		AncestorTypes: ancestors,
	}
}

// FlattenVisible returns all visible nodes (those whose ancestors are all expanded).
// The root node itself is not included — only its descendants.
func FlattenVisible(root *TreeNode) []FlatNode {
	var result []FlatNode
	flattenChildren(root, 0, &result)
	return result
}

func flattenChildren(node *TreeNode, depth int, result *[]FlatNode) {
	for _, child := range node.Children {
		*result = append(*result, FlatNode{Node: child, Depth: depth})
		if child.Expanded && len(child.Children) > 0 {
			flattenChildren(child, depth+1, result)
		}
	}
}

// ToggleSelected toggles the selected state of a node.
// When selecting: auto-select the entire parent chain.
// When deselecting: recursively deselect all descendants.
// Selecting a parent does NOT auto-select children.
func ToggleSelected(node *TreeNode) {
	if node.Selected {
		// Deselect: recursively deselect descendants
		deselectSubtree(node)
	} else {
		// Select: also select all ancestors
		node.Selected = true
		selectAncestors(node)
	}
}

func deselectSubtree(node *TreeNode) {
	node.Selected = false
	for _, child := range node.Children {
		deselectSubtree(child)
	}
}

func selectAncestors(node *TreeNode) {
	for p := node.Parent; p != nil; p = p.Parent {
		p.Selected = true
	}
}

// HasSelectedChildren returns true if any direct child is selected.
func HasSelectedChildren(node *TreeNode) bool {
	for _, child := range node.Children {
		if child.Selected {
			return true
		}
	}
	return false
}

// ToggleChildrenSelected selects or deselects all immediate children of a node.
// If any child is selected, deselects all children (and their descendants).
// Otherwise, selects all immediate children and auto-expands the node.
func ToggleChildrenSelected(s *schema.Schema, node *TreeNode) {
	if node.IsLeaf {
		return
	}
	EnsureChildrenReady(s, node)
	if len(node.Children) == 0 {
		return
	}

	anySelected := false
	for _, child := range node.Children {
		if child.Selected {
			anySelected = true
			break
		}
	}

	if anySelected {
		for _, child := range node.Children {
			deselectSubtree(child)
		}
	} else {
		node.Selected = true
		node.Expanded = true
		selectAncestors(node)
		for _, child := range node.Children {
			child.Selected = true
		}
	}
}

// resolveTypeKind returns the Kind of a named type from the schema.
func resolveTypeKind(s *schema.Schema, name string) string {
	t := s.TypeByName(name)
	if t == nil {
		return "SCALAR" // unknown types treated as scalar
	}
	return t.Kind
}

func copyAncestors(src map[string]bool) map[string]bool {
	dst := make(map[string]bool, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
