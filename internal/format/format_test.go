package format

import (
	"testing"
)

func TestJSONPrettify(t *testing.T) {
	input := `{"key":"value","num":42}`
	got, err := JSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "{\n  \"key\": \"value\",\n  \"num\": 42\n}"
	if got != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, got)
	}
}

func TestJSONPrettifyInvalid(t *testing.T) {
	_, err := JSON(`{invalid}`)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestJSONPrettifyEmpty(t *testing.T) {
	got, err := JSON("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestGraphQLPrettifyShorthand(t *testing.T) {
	input := `{ countries { code currency languages { name } continent { name } } }`
	got := GraphQL(input)
	expected := `{
  countries {
    code
    currency
    languages {
      name
    }
    continent {
      name
    }
  }
}`
	if got != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, got)
	}
}

func TestGraphQLPrettifyNamedQuery(t *testing.T) {
	input := `query GetCountries { countries { name code } }`
	got := GraphQL(input)
	expected := `query GetCountries {
  countries {
    name
    code
  }
}`
	if got != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, got)
	}
}

func TestGraphQLPrettifyWithArgs(t *testing.T) {
	input := `query($id: ID!) { user(id: $id) { name email } }`
	got := GraphQL(input)
	expected := `query($id: ID!) {
  user(id: $id) {
    name
    email
  }
}`
	if got != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, got)
	}
}

func TestGraphQLPrettifyEmpty(t *testing.T) {
	got := GraphQL("")
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestGraphQLPrettifyAlreadyFormatted(t *testing.T) {
	input := "{\n  countries {\n    name\n  }\n}"
	got := GraphQL(input)
	if got != input {
		t.Errorf("expected idempotent formatting:\n%s\ngot:\n%s", input, got)
	}
}

func TestValidateGraphQLBalanced(t *testing.T) {
	err := ValidateGraphQL(`{ countries { name } }`)
	if err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
}

func TestValidateGraphQLUnclosedBrace(t *testing.T) {
	err := ValidateGraphQL(`{ countries { name }`)
	if err == nil {
		t.Error("expected error for unclosed brace")
	}
}

func TestValidateGraphQLExtraClose(t *testing.T) {
	err := ValidateGraphQL(`{ countries } }`)
	if err == nil {
		t.Error("expected error for extra closing brace")
	}
}

func TestValidateJSONValid(t *testing.T) {
	err := ValidateJSON(`{"key": "value"}`)
	if err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
}

func TestValidateJSONInvalid(t *testing.T) {
	err := ValidateJSON(`{bad}`)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
