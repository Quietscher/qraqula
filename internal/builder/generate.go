package builder

import (
	"encoding/json"
	"strings"

	"github.com/qraqula/qla/internal/format"
	"github.com/qraqula/qla/internal/schema"
)

// GenerateFromTree builds a GraphQL query string and variables JSON from a tree.
func GenerateFromTree(s *schema.Schema, opType string, opFieldName string, root *TreeNode) (query, variables string) {
	// Collect variable declarations and argument references from the root
	var varDecls []string
	var argRefs []string
	varsMap := make(map[string]any)

	for _, arg := range root.Args {
		if !root.ArgValues[arg.Name] {
			continue
		}
		varName := arg.Name
		varDecls = append(varDecls, "$"+varName+": "+arg.Type.DisplayName())
		argRefs = append(argRefs, arg.Name+": $"+varName)
		varsMap[varName] = schema.ExampleValue(s, arg.Type, make(map[string]bool))
	}

	// Also collect args from child nodes
	collectChildArgs(s, root, varDecls, argRefs, varsMap, &varDecls, &argRefs)

	// Build the selection set from selected children
	selSet := buildSelSet(root)

	var buf strings.Builder

	// Operation header
	if opType != "" {
		opName := strings.ToUpper(opFieldName[:1]) + opFieldName[1:]
		buf.WriteString(opType)
		buf.WriteByte(' ')
		buf.WriteString(opName)

		if len(varDecls) > 0 {
			buf.WriteByte('(')
			buf.WriteString(strings.Join(varDecls, ", "))
			buf.WriteByte(')')
		}
	}

	buf.WriteString(" { ")
	buf.WriteString(opFieldName)

	if len(argRefs) > 0 {
		buf.WriteByte('(')
		buf.WriteString(strings.Join(argRefs, ", "))
		buf.WriteByte(')')
	}

	if selSet != "" {
		buf.WriteString(" { ")
		buf.WriteString(selSet)
		buf.WriteString(" }")
	}

	buf.WriteString(" }")

	query = format.GraphQL(buf.String())

	if len(varsMap) > 0 {
		b, _ := json.MarshalIndent(varsMap, "", "  ")
		variables = string(b)
	}

	return query, variables
}

// collectChildArgs recursively collects enabled arguments from child nodes.
func collectChildArgs(s *schema.Schema, node *TreeNode, existDecls, existRefs []string, varsMap map[string]any, varDecls *[]string, argRefs *[]string) {
	for _, child := range node.Children {
		if !child.Selected || child.IsSpread {
			continue
		}
		// Child args are handled inline in the selection set via buildSelSet,
		// so we don't collect them here. Only root-level args become variables.
		if child.Expanded && len(child.Children) > 0 {
			collectChildArgs(s, child, existDecls, existRefs, varsMap, varDecls, argRefs)
		}
	}
}

// buildSelSet recursively builds a space-separated selection set from selected children.
func buildSelSet(node *TreeNode) string {
	var parts []string

	for _, child := range node.Children {
		if !child.Selected {
			continue
		}

		if child.IsSpread {
			// Inline fragment: ... on TypeName { ... }
			sub := buildSelSet(child)
			if sub != "" {
				parts = append(parts, child.Name+" { "+sub+" }")
			}
			continue
		}

		if child.IsLeaf {
			parts = append(parts, child.Name)
			continue
		}

		// Object/interface/union field — only include if it has selected children
		if HasSelectedChildren(child) {
			sub := buildSelSet(child)
			if sub != "" {
				parts = append(parts, child.Name+" { "+sub+" }")
			}
		}
	}

	return strings.Join(parts, " ")
}
