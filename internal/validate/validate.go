package validate

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/qraqula/qla/internal/schema"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
)

// SchemaAST wraps a gqlparser schema parsed from an introspection result.
type SchemaAST struct {
	ast    *ast.Schema
	source *schema.Schema
}

// LoadSchema converts an introspection schema to a gqlparser AST schema.
// Returns nil if the schema is nil or conversion fails.
func LoadSchema(s *schema.Schema) *SchemaAST {
	if s == nil {
		return nil
	}
	sdl := IntrospectionToSDL(s)
	parsed, err := gqlparser.LoadSchema(&ast.Source{Input: sdl})
	if err != nil {
		return nil
	}
	return &SchemaAST{ast: parsed, source: s}
}

// Query validates a GraphQL query string against the schema.
// Returns a human-readable error if validation fails, or nil if the query is valid.
// If schemaAST is nil, only syntax validation is performed.
func Query(query string, schemaAST *SchemaAST) error {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}

	if schemaAST == nil {
		// No schema — just parse for syntax errors
		_, err := parser.ParseQuery(&ast.Source{Input: query})
		if err != nil {
			return simplifyError(err.Error())
		}
		return nil
	}

	_, errs := gqlparser.LoadQuery(schemaAST.ast, query)
	if errs != nil {
		return simplifyError(errs.Error())
	}
	return nil
}

// Variables validates variables JSON against the variable definitions in a query.
// It checks:
//   - All required variables (non-null without defaults) are present
//   - No unknown variables are provided
//   - Basic type compatibility (scalars, enums, input objects)
//
// If schemaAST is nil, only JSON syntax is validated.
func Variables(varsJSON string, query string, schemaAST *SchemaAST) error {
	varsJSON = strings.TrimSpace(varsJSON)
	if varsJSON == "" {
		return nil
	}

	// Parse JSON
	var vars map[string]any
	if err := json.Unmarshal([]byte(varsJSON), &vars); err != nil {
		return fmt.Errorf("invalid JSON")
	}

	if schemaAST == nil {
		return nil
	}

	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}

	// Parse the query to extract variable definitions
	doc, errs := gqlparser.LoadQuery(schemaAST.ast, query)
	if errs != nil {
		// Query itself is invalid — skip variable validation
		return nil
	}

	if len(doc.Operations) == 0 {
		return nil
	}

	op := doc.Operations[0]
	defs := op.VariableDefinitions

	// Check for missing required variables
	for _, def := range defs {
		_, provided := vars[def.Variable]
		required := def.Type.NonNull && def.DefaultValue == nil
		if required && !provided {
			return fmt.Errorf("missing required variable $%s (%s)", def.Variable, def.Type.String())
		}
	}

	// Check for unknown variables
	defNames := make(map[string]bool, len(defs))
	for _, def := range defs {
		defNames[def.Variable] = true
	}
	for name := range vars {
		if !defNames[name] {
			return fmt.Errorf("unknown variable $%s", name)
		}
	}

	// Type-check provided variables
	for _, def := range defs {
		val, ok := vars[def.Variable]
		if !ok {
			continue
		}
		if err := checkType(val, def.Type, schemaAST.ast); err != nil {
			return fmt.Errorf("$%s: %w", def.Variable, err)
		}
	}

	return nil
}

// checkType validates a JSON value against an expected GraphQL type.
func checkType(val any, typ *ast.Type, s *ast.Schema) error {
	if val == nil {
		if typ.NonNull {
			return fmt.Errorf("expected %s, got null", typ.String())
		}
		return nil
	}

	// Unwrap NonNull for further checking
	innerType := typ
	if innerType.NonNull {
		innerType = &ast.Type{
			NamedType: typ.NamedType,
			Elem:      typ.Elem,
		}
	}

	// List type
	if innerType.Elem != nil {
		arr, ok := val.([]any)
		if !ok {
			return fmt.Errorf("expected list for %s", typ.String())
		}
		for i, item := range arr {
			if err := checkType(item, innerType.Elem, s); err != nil {
				return fmt.Errorf("[%d]: %w", i, err)
			}
		}
		return nil
	}

	// Named type
	name := innerType.NamedType
	def := s.Types[name]

	switch {
	case isScalar(name):
		return checkScalar(val, name)
	case def != nil && def.Kind == ast.Enum:
		str, ok := val.(string)
		if !ok {
			return fmt.Errorf("expected enum value (string) for %s", name)
		}
		for _, ev := range def.EnumValues {
			if ev.Name == str {
				return nil
			}
		}
		return fmt.Errorf("invalid enum value %q for %s", str, name)
	case def != nil && def.Kind == ast.InputObject:
		obj, ok := val.(map[string]any)
		if !ok {
			return fmt.Errorf("expected object for %s", name)
		}
		return checkInputObject(obj, def, s)
	}

	return nil
}

func checkScalar(val any, name string) error {
	switch name {
	case "String", "ID":
		if _, ok := val.(string); !ok {
			return fmt.Errorf("expected string for %s", name)
		}
	case "Int":
		switch v := val.(type) {
		case float64:
			if v != float64(int64(v)) {
				return fmt.Errorf("expected integer for Int")
			}
		default:
			return fmt.Errorf("expected number for Int")
		}
	case "Float":
		if _, ok := val.(float64); !ok {
			return fmt.Errorf("expected number for Float")
		}
	case "Boolean":
		if _, ok := val.(bool); !ok {
			return fmt.Errorf("expected boolean for Boolean")
		}
	}
	return nil
}

func checkInputObject(obj map[string]any, def *ast.Definition, s *ast.Schema) error {
	// Check for required fields
	for _, field := range def.Fields {
		_, provided := obj[field.Name]
		if field.Type.NonNull && field.DefaultValue == nil && !provided {
			return fmt.Errorf("missing required field %q", field.Name)
		}
	}

	// Check for unknown fields
	fieldNames := make(map[string]bool, len(def.Fields))
	for _, field := range def.Fields {
		fieldNames[field.Name] = true
	}
	for name := range obj {
		if !fieldNames[name] {
			return fmt.Errorf("unknown field %q on %s", name, def.Name)
		}
	}

	// Type-check provided fields
	for _, field := range def.Fields {
		val, ok := obj[field.Name]
		if !ok {
			continue
		}
		if err := checkType(val, field.Type, s); err != nil {
			return fmt.Errorf("%s: %w", field.Name, err)
		}
	}

	return nil
}

func isScalar(name string) bool {
	switch name {
	case "String", "Int", "Float", "Boolean", "ID":
		return true
	}
	return false
}

// simplifyError extracts the first meaningful error message from gqlparser output.
func simplifyError(msg string) error {
	// gqlparser errors often have "input:line:col: message" format
	// Extract just the first error message for status bar display
	lines := strings.Split(msg, "\n")
	if len(lines) == 0 {
		return fmt.Errorf("%s", msg)
	}
	first := lines[0]
	// Strip "input:N: " prefix
	if idx := strings.Index(first, ": "); idx >= 0 {
		rest := first[idx+2:]
		// Check for a second colon prefix (line:col:)
		if idx2 := strings.Index(rest, ": "); idx2 >= 0 {
			// Could be "col: msg" or just "msg"
			// If the part before : is a number, strip it too
			before := rest[:idx2]
			allDigits := true
			for _, c := range before {
				if c < '0' || c > '9' {
					allDigits = false
					break
				}
			}
			if allDigits {
				return fmt.Errorf("%s", rest[idx2+2:])
			}
		}
		return fmt.Errorf("%s", rest)
	}
	return fmt.Errorf("%s", first)
}
