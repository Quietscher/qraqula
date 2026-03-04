package builder

import (
	"strings"

	"github.com/qraqula/qla/internal/schema"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
)

// ParseExistingQuery attempts to parse a query string and build a pre-selected tree.
// Returns the root tree node, the operation type ("query"/"mutation"/"subscription"),
// the operation field name, and an error if parsing fails.
func ParseExistingQuery(queryStr string, s *schema.Schema) (root *TreeNode, opType, opField string, err error) {
	queryStr = strings.TrimSpace(queryStr)
	if queryStr == "" {
		return nil, "", "", errParseFailed("empty query")
	}

	doc, parseErr := parser.ParseQuery(&ast.Source{Input: queryStr})
	if parseErr != nil {
		return nil, "", "", parseErr
	}

	if len(doc.Operations) == 0 {
		return nil, "", "", errParseFailed("no operations found")
	}

	op := doc.Operations[0]
	opType = string(op.Operation)
	if opType == "" {
		opType = "query"
	}

	// Find the root type in the schema
	rootTypeName := rootTypeNameForOp(s, opType)
	if rootTypeName == "" {
		return nil, "", "", errParseFailed("schema has no " + opType + " type")
	}
	rootType := s.TypeByName(rootTypeName)
	if rootType == nil {
		return nil, "", "", errParseFailed("root type not found: " + rootTypeName)
	}

	// Expect exactly one field selection at the root
	if len(op.SelectionSet) == 0 {
		return nil, "", "", errParseFailed("empty selection set")
	}

	// Find the first field in the selection set
	var firstField *ast.Field
	for _, sel := range op.SelectionSet {
		if f, ok := sel.(*ast.Field); ok {
			firstField = f
			break
		}
	}
	if firstField == nil {
		return nil, "", "", errParseFailed("no field in selection set")
	}

	opField = firstField.Name

	// Find the corresponding schema field
	var schemaField *schema.Field
	for i := range rootType.Fields {
		if rootType.Fields[i].Name == opField {
			schemaField = &rootType.Fields[i]
			break
		}
	}
	if schemaField == nil {
		return nil, "", "", errParseFailed("field " + opField + " not found on " + rootTypeName)
	}

	// Build the tree from this field
	root = BuildTreeFromField(s, *schemaField)

	// Now mark selected fields based on the AST
	// First, deselect everything (BuildTreeFromField selects root)
	deselectAll(root)
	root.Selected = true

	// Walk the AST selection set and mark matching tree nodes
	markFromAST(s, root, firstField.SelectionSet)

	// Mark enabled arguments from the AST
	markArgsFromAST(root, firstField.Arguments)

	return root, opType, opField, nil
}

// deselectAll recursively deselects all nodes.
func deselectAll(node *TreeNode) {
	node.Selected = false
	for _, child := range node.Children {
		deselectAll(child)
	}
}

// markFromAST walks the AST selection set and marks matching tree nodes as selected and expanded.
func markFromAST(s *schema.Schema, node *TreeNode, selections ast.SelectionSet) {
	for _, sel := range selections {
		switch sel := sel.(type) {
		case *ast.Field:
			// Find matching child
			for _, child := range node.Children {
				if child.Name == sel.Name {
					child.Selected = true
					if len(sel.SelectionSet) > 0 {
						child.Expanded = true
						LoadChildren(s, child)
						markFromAST(s, child, sel.SelectionSet)
					}
					markArgsFromAST(child, sel.Arguments)
					break
				}
			}
		case *ast.InlineFragment:
			// Find matching spread node
			if sel.TypeCondition != "" {
				spreadName := "... on " + sel.TypeCondition
				for _, child := range node.Children {
					if child.Name == spreadName {
						child.Selected = true
						child.Expanded = true
						LoadChildren(s, child)
						markFromAST(s, child, sel.SelectionSet)
						break
					}
				}
			}
		}
	}
}

// markArgsFromAST marks arguments as enabled in the node's ArgValues map.
func markArgsFromAST(node *TreeNode, args ast.ArgumentList) {
	for _, arg := range args {
		node.ArgValues[arg.Name] = true
	}
}

// rootTypeNameForOp returns the root type name for the given operation type.
func rootTypeNameForOp(s *schema.Schema, opType string) string {
	switch opType {
	case "query":
		if s.QueryType != nil && s.QueryType.Name != nil {
			return *s.QueryType.Name
		}
	case "mutation":
		if s.MutationType != nil && s.MutationType.Name != nil {
			return *s.MutationType.Name
		}
	case "subscription":
		if s.SubscriptionType != nil && s.SubscriptionType.Name != nil {
			return *s.SubscriptionType.Name
		}
	}
	return ""
}

type parseError struct {
	msg string
}

func (e *parseError) Error() string { return e.msg }

func errParseFailed(msg string) error {
	return &parseError{msg: msg}
}
