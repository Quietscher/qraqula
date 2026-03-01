package schema

import (
	"encoding/json"
	"strings"

	"github.com/qraqula/qla/internal/format"
)

// GenerateQueryMsg is sent when the user presses 'g' in the schema browser
// to generate a query from the selected field.
type GenerateQueryMsg struct {
	Query     string
	Variables string
}

// GenerateQuery builds a complete GraphQL operation string and example variables
// JSON for the given root-level field. opType is "query", "mutation", or
// "subscription".
func GenerateQuery(s *Schema, opType, rootTypeName string, field Field) (query, variables string) {
	// Collect variable declarations and argument references
	var varDecls []string
	var argRefs []string
	varsMap := make(map[string]any)

	for _, arg := range field.Args {
		varName := arg.Name
		varDecls = append(varDecls, "$"+varName+": "+arg.Type.DisplayName())
		argRefs = append(argRefs, arg.Name+": $"+varName)
		varsMap[varName] = exampleValue(s, arg.Type, make(map[string]bool))
	}

	// Build selection set for the return type
	visited := make(map[string]bool)
	selSet := buildSelectionSet(s, field.Type, visited, 0)

	// Assemble the operation
	var buf strings.Builder

	// Operation name: capitalize field name
	opName := strings.ToUpper(field.Name[:1]) + field.Name[1:]

	buf.WriteString(opType)
	buf.WriteByte(' ')
	buf.WriteString(opName)

	if len(varDecls) > 0 {
		buf.WriteByte('(')
		buf.WriteString(strings.Join(varDecls, ", "))
		buf.WriteByte(')')
	}

	buf.WriteString(" { ")
	buf.WriteString(field.Name)

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

// buildSelectionSet recursively expands fields for a type reference.
// It returns a space-separated list of field selections (without outer braces).
func buildSelectionSet(s *Schema, ref TypeRef, visited map[string]bool, depth int) string {
	// Unwrap NON_NULL and LIST wrappers
	if ref.Kind == "NON_NULL" || ref.Kind == "LIST" {
		if ref.OfType != nil {
			return buildSelectionSet(s, *ref.OfType, visited, depth)
		}
		return ""
	}

	name := ""
	if ref.Name != nil {
		name = *ref.Name
	}
	if name == "" {
		return ""
	}

	t := s.TypeByName(name)
	if t == nil {
		return ""
	}

	switch t.Kind {
	case "SCALAR", "ENUM":
		return "" // leaf type, no sub-selection needed

	case "OBJECT", "INTERFACE":
		if visited[name] || depth >= 5 {
			return ""
		}
		visited[name] = true
		var fields []string
		for _, f := range t.Fields {
			sub := buildSelectionSet(s, f.Type, visited, depth+1)
			if sub != "" {
				fields = append(fields, f.Name+" { "+sub+" }")
			} else {
				// Only include if it's a leaf (scalar/enum)
				leafName := resolveNamedType(s, f.Type)
				if leafName != nil && isLeaf(s, *leafName) {
					fields = append(fields, f.Name)
				}
			}
		}
		delete(visited, name) // backtrack: allow same type in sibling branches
		if len(fields) == 0 {
			return ""
		}
		return strings.Join(fields, " ")

	case "UNION":
		var parts []string
		parts = append(parts, "__typename")
		for _, pt := range t.PossibleTypes {
			ptName := pt.NamedType()
			if ptName == "" {
				continue
			}
			ptRef := TypeRef{Kind: "OBJECT", Name: &ptName}
			sub := buildSelectionSet(s, ptRef, visited, depth+1)
			if sub != "" {
				parts = append(parts, "... on "+ptName+" { "+sub+" }")
			}
		}
		return strings.Join(parts, " ")
	}

	return ""
}

// resolveNamedType unwraps wrappers and returns the inner type name, or nil.
func resolveNamedType(s *Schema, ref TypeRef) *string {
	if ref.Name != nil {
		return ref.Name
	}
	if ref.OfType != nil {
		return resolveNamedType(s, *ref.OfType)
	}
	return nil
}

// isLeaf returns true if the named type is a scalar or enum.
func isLeaf(s *Schema, name string) bool {
	t := s.TypeByName(name)
	if t == nil {
		return true // unknown types treated as leaf
	}
	return t.Kind == "SCALAR" || t.Kind == "ENUM"
}

// exampleValue generates a plausible example value for a type reference,
// suitable for use in a variables JSON object.
func exampleValue(s *Schema, ref TypeRef, visited map[string]bool) any {
	switch ref.Kind {
	case "NON_NULL":
		if ref.OfType != nil {
			return exampleValue(s, *ref.OfType, visited)
		}
		return nil

	case "LIST":
		if ref.OfType != nil {
			return []any{exampleValue(s, *ref.OfType, visited)}
		}
		return []any{}
	}

	name := ""
	if ref.Name != nil {
		name = *ref.Name
	}
	if name == "" {
		return nil
	}

	// Built-in scalars
	switch name {
	case "String":
		return "example"
	case "Int":
		return 42
	case "Float":
		return 3.14
	case "Boolean":
		return false
	case "ID":
		return "1"
	}

	t := s.TypeByName(name)
	if t == nil {
		return "example" // unknown type, treat as custom scalar
	}

	switch t.Kind {
	case "SCALAR":
		return "example" // custom scalar

	case "ENUM":
		if len(t.EnumValues) > 0 {
			return t.EnumValues[0].Name
		}
		return nil

	case "INPUT_OBJECT":
		// Permanent visited marking for input objects to break cycles
		if visited[name] {
			return nil
		}
		visited[name] = true
		obj := make(map[string]any, len(t.InputFields))
		for _, f := range t.InputFields {
			obj[f.Name] = exampleValue(s, f.Type, visited)
		}
		return obj
	}

	return nil
}
