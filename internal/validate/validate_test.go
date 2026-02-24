package validate

import (
	"testing"

	"github.com/qraqula/qla/internal/schema"
)

func ptr(s string) *string { return &s }

// testSchema returns a minimal schema for testing.
func testSchema() *schema.Schema {
	return &schema.Schema{
		QueryType: &schema.TypeRef{Kind: "OBJECT", Name: ptr("Query")},
		Types: []schema.FullType{
			{
				Kind: "OBJECT",
				Name: "Query",
				Fields: []schema.Field{
					{
						Name: "user",
						Args: []schema.InputValue{
							{Name: "id", Type: schema.TypeRef{Kind: "NON_NULL", OfType: &schema.TypeRef{Kind: "SCALAR", Name: ptr("ID")}}},
						},
						Type: schema.TypeRef{Kind: "OBJECT", Name: ptr("User")},
					},
					{
						Name: "users",
						Args: []schema.InputValue{
							{Name: "role", Type: schema.TypeRef{Kind: "ENUM", Name: ptr("Role")}},
						},
						Type: schema.TypeRef{Kind: "NON_NULL", OfType: &schema.TypeRef{Kind: "LIST", OfType: &schema.TypeRef{Kind: "NON_NULL", OfType: &schema.TypeRef{Kind: "OBJECT", Name: ptr("User")}}}},
					},
				},
			},
			{
				Kind: "OBJECT",
				Name: "User",
				Fields: []schema.Field{
					{Name: "id", Type: schema.TypeRef{Kind: "NON_NULL", OfType: &schema.TypeRef{Kind: "SCALAR", Name: ptr("ID")}}},
					{Name: "name", Type: schema.TypeRef{Kind: "NON_NULL", OfType: &schema.TypeRef{Kind: "SCALAR", Name: ptr("String")}}},
					{Name: "email", Type: schema.TypeRef{Kind: "SCALAR", Name: ptr("String")}},
					{Name: "role", Type: schema.TypeRef{Kind: "NON_NULL", OfType: &schema.TypeRef{Kind: "ENUM", Name: ptr("Role")}}},
				},
			},
			{
				Kind: "ENUM",
				Name: "Role",
				EnumValues: []schema.EnumValue{
					{Name: "ADMIN"},
					{Name: "USER"},
				},
			},
			{
				Kind: "INPUT_OBJECT",
				Name: "CreateUserInput",
				InputFields: []schema.InputValue{
					{Name: "name", Type: schema.TypeRef{Kind: "NON_NULL", OfType: &schema.TypeRef{Kind: "SCALAR", Name: ptr("String")}}},
					{Name: "email", Type: schema.TypeRef{Kind: "SCALAR", Name: ptr("String")}},
					{Name: "role", Type: schema.TypeRef{Kind: "ENUM", Name: ptr("Role")}},
				},
			},
		},
	}
}

func TestIntrospectionToSDL(t *testing.T) {
	s := testSchema()
	sdl := IntrospectionToSDL(s)
	if sdl == "" {
		t.Fatal("expected non-empty SDL")
	}
	// Should contain schema definition
	if !contains(sdl, "schema {") {
		t.Error("expected schema definition")
	}
	if !contains(sdl, "query: Query") {
		t.Error("expected query root type")
	}
	// Should contain types
	if !contains(sdl, "type Query") {
		t.Error("expected Query type")
	}
	if !contains(sdl, "type User") {
		t.Error("expected User type")
	}
	if !contains(sdl, "enum Role") {
		t.Error("expected Role enum")
	}
	if !contains(sdl, "input CreateUserInput") {
		t.Error("expected CreateUserInput input")
	}
}

func TestLoadSchema(t *testing.T) {
	s := testSchema()
	ast := LoadSchema(s)
	if ast == nil {
		t.Fatal("expected non-nil SchemaAST")
	}
}

func TestLoadSchemaNil(t *testing.T) {
	ast := LoadSchema(nil)
	if ast != nil {
		t.Error("expected nil for nil schema")
	}
}

func TestQueryValid(t *testing.T) {
	ast := LoadSchema(testSchema())
	err := Query(`{ user(id: "1") { id name } }`, ast)
	if err != nil {
		t.Errorf("expected valid query, got: %v", err)
	}
}

func TestQueryInvalidField(t *testing.T) {
	ast := LoadSchema(testSchema())
	err := Query(`{ user(id: "1") { id nonexistent } }`, ast)
	if err == nil {
		t.Error("expected error for nonexistent field")
	}
}

func TestQueryMissingRequiredArg(t *testing.T) {
	ast := LoadSchema(testSchema())
	err := Query(`{ user { id } }`, ast)
	if err == nil {
		t.Error("expected error for missing required argument")
	}
}

func TestQuerySyntaxError(t *testing.T) {
	ast := LoadSchema(testSchema())
	err := Query(`{ user(id: "1") { id `, ast)
	if err == nil {
		t.Error("expected syntax error")
	}
}

func TestQueryNoSchema(t *testing.T) {
	// Without schema, only syntax is checked
	err := Query(`{ user { id } }`, nil)
	if err != nil {
		t.Errorf("expected syntax-only pass, got: %v", err)
	}
}

func TestQuerySyntaxErrorNoSchema(t *testing.T) {
	err := Query(`{ broken `, nil)
	if err == nil {
		t.Error("expected syntax error without schema")
	}
}

func TestVariablesValid(t *testing.T) {
	ast := LoadSchema(testSchema())
	err := Variables(`{"id": "123"}`, `query($id: ID!) { user(id: $id) { name } }`, ast)
	if err != nil {
		t.Errorf("expected valid variables, got: %v", err)
	}
}

func TestVariablesMissingRequired(t *testing.T) {
	ast := LoadSchema(testSchema())
	err := Variables(`{}`, `query($id: ID!) { user(id: $id) { name } }`, ast)
	if err == nil {
		t.Error("expected error for missing required variable")
	}
	if !contains(err.Error(), "$id") {
		t.Errorf("error should mention $id: %v", err)
	}
}

func TestVariablesUnknown(t *testing.T) {
	ast := LoadSchema(testSchema())
	err := Variables(`{"id": "1", "extra": true}`, `query($id: ID!) { user(id: $id) { name } }`, ast)
	if err == nil {
		t.Error("expected error for unknown variable")
	}
	if !contains(err.Error(), "$extra") {
		t.Errorf("error should mention $extra: %v", err)
	}
}

func TestVariablesWrongType(t *testing.T) {
	ast := LoadSchema(testSchema())
	err := Variables(`{"id": 123}`, `query($id: ID!) { user(id: $id) { name } }`, ast)
	if err == nil {
		t.Error("expected error for wrong type (number for ID)")
	}
}

func TestVariablesEnumValid(t *testing.T) {
	ast := LoadSchema(testSchema())
	err := Variables(`{"role": "ADMIN"}`, `query($role: Role) { users(role: $role) { name } }`, ast)
	if err != nil {
		t.Errorf("expected valid enum variable, got: %v", err)
	}
}

func TestVariablesEnumInvalid(t *testing.T) {
	ast := LoadSchema(testSchema())
	err := Variables(`{"role": "INVALID"}`, `query($role: Role) { users(role: $role) { name } }`, ast)
	if err == nil {
		t.Error("expected error for invalid enum value")
	}
}

func TestVariablesInvalidJSON(t *testing.T) {
	ast := LoadSchema(testSchema())
	err := Variables(`{broken`, `{ user(id: "1") { id } }`, ast)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestVariablesEmpty(t *testing.T) {
	ast := LoadSchema(testSchema())
	err := Variables(``, `{ user(id: "1") { id } }`, ast)
	if err != nil {
		t.Errorf("expected nil for empty variables, got: %v", err)
	}
}

func TestVariablesNoSchema(t *testing.T) {
	err := Variables(`{"id": "1"}`, `query($id: ID!) { user(id: $id) { name } }`, nil)
	if err != nil {
		t.Errorf("expected pass without schema, got: %v", err)
	}
}

func TestVariablesIntValid(t *testing.T) {
	// Int should accept integer values
	s := &schema.Schema{
		QueryType: &schema.TypeRef{Kind: "OBJECT", Name: ptr("Query")},
		Types: []schema.FullType{
			{
				Kind: "OBJECT",
				Name: "Query",
				Fields: []schema.Field{
					{
						Name: "item",
						Args: []schema.InputValue{
							{Name: "count", Type: schema.TypeRef{Kind: "SCALAR", Name: ptr("Int")}},
						},
						Type: schema.TypeRef{Kind: "SCALAR", Name: ptr("String")},
					},
				},
			},
		},
	}
	ast := LoadSchema(s)
	err := Variables(`{"count": 5}`, `query($count: Int) { item(count: $count) }`, ast)
	if err != nil {
		t.Errorf("expected valid int, got: %v", err)
	}
}

func TestVariablesIntFloat(t *testing.T) {
	s := &schema.Schema{
		QueryType: &schema.TypeRef{Kind: "OBJECT", Name: ptr("Query")},
		Types: []schema.FullType{
			{
				Kind: "OBJECT",
				Name: "Query",
				Fields: []schema.Field{
					{
						Name: "item",
						Args: []schema.InputValue{
							{Name: "count", Type: schema.TypeRef{Kind: "SCALAR", Name: ptr("Int")}},
						},
						Type: schema.TypeRef{Kind: "SCALAR", Name: ptr("String")},
					},
				},
			},
		},
	}
	ast := LoadSchema(s)
	err := Variables(`{"count": 5.5}`, `query($count: Int) { item(count: $count) }`, ast)
	if err == nil {
		t.Error("expected error for float where Int expected")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchContains(s, substr)
}

func searchContains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
