package schema

import "testing"

func TestTypeRefDisplayName_Scalar(t *testing.T) {
	name := "String"
	tr := TypeRef{Kind: "SCALAR", Name: &name}
	if got := tr.DisplayName(); got != "String" {
		t.Errorf("got %q, want %q", got, "String")
	}
}

func TestTypeRefDisplayName_NonNull(t *testing.T) {
	name := "String"
	tr := TypeRef{Kind: "NON_NULL", OfType: &TypeRef{Kind: "SCALAR", Name: &name}}
	if got := tr.DisplayName(); got != "String!" {
		t.Errorf("got %q, want %q", got, "String!")
	}
}

func TestTypeRefDisplayName_List(t *testing.T) {
	name := "User"
	tr := TypeRef{Kind: "LIST", OfType: &TypeRef{Kind: "OBJECT", Name: &name}}
	if got := tr.DisplayName(); got != "[User]" {
		t.Errorf("got %q, want %q", got, "[User]")
	}
}

func TestTypeRefDisplayName_NonNullList(t *testing.T) {
	name := "Post"
	tr := TypeRef{
		Kind: "NON_NULL",
		OfType: &TypeRef{
			Kind: "LIST",
			OfType: &TypeRef{
				Kind: "NON_NULL",
				OfType: &TypeRef{Kind: "OBJECT", Name: &name},
			},
		},
	}
	if got := tr.DisplayName(); got != "[Post!]!" {
		t.Errorf("got %q, want %q", got, "[Post!]!")
	}
}

func TestTypeRefNamedType(t *testing.T) {
	name := "User"
	tr := TypeRef{
		Kind: "NON_NULL",
		OfType: &TypeRef{
			Kind: "LIST",
			OfType: &TypeRef{Kind: "OBJECT", Name: &name},
		},
	}
	if got := tr.NamedType(); got != "User" {
		t.Errorf("got %q, want %q", got, "User")
	}
}

func TestSchemaTypeByName(t *testing.T) {
	s := Schema{
		Types: []FullType{
			{Name: "Query", Kind: "OBJECT"},
			{Name: "User", Kind: "OBJECT"},
			{Name: "String", Kind: "SCALAR"},
		},
	}
	if got := s.TypeByName("User"); got == nil || got.Name != "User" {
		t.Errorf("expected to find User type")
	}
	if got := s.TypeByName("Missing"); got != nil {
		t.Errorf("expected nil for missing type")
	}
}

func TestSchemaRootTypes(t *testing.T) {
	qName := "Query"
	mName := "Mutation"
	s := Schema{
		QueryType:    &TypeRef{Name: &qName},
		MutationType: &TypeRef{Name: &mName},
		Types: []FullType{
			{Name: "Query", Kind: "OBJECT", Fields: []Field{{Name: "user"}}},
			{Name: "Mutation", Kind: "OBJECT", Fields: []Field{{Name: "createUser"}}},
		},
	}
	roots := s.RootTypes()
	if len(roots) != 2 {
		t.Errorf("expected 2 root types, got %d", len(roots))
	}
}
