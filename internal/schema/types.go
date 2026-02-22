package schema

// Schema represents a parsed GraphQL introspection schema.
type Schema struct {
	QueryType        *TypeRef   `json:"queryType"`
	MutationType     *TypeRef   `json:"mutationType"`
	SubscriptionType *TypeRef   `json:"subscriptionType"`
	Types            []FullType `json:"types"`
}

// TypeByName returns the type with the given name, or nil.
func (s *Schema) TypeByName(name string) *FullType {
	for i := range s.Types {
		if s.Types[i].Name == name {
			return &s.Types[i]
		}
	}
	return nil
}

// RootTypes returns the root operation types (Query, Mutation, Subscription)
// that exist in this schema, in order.
func (s *Schema) RootTypes() []FullType {
	var roots []FullType
	for _, ref := range []*TypeRef{s.QueryType, s.MutationType, s.SubscriptionType} {
		if ref != nil && ref.Name != nil {
			if t := s.TypeByName(*ref.Name); t != nil {
				roots = append(roots, *t)
			}
		}
	}
	return roots
}

// FullType represents a complete type from the introspection schema.
type FullType struct {
	Kind          string       `json:"kind"`
	Name          string       `json:"name"`
	Description   string       `json:"description"`
	Fields        []Field      `json:"fields"`
	InputFields   []InputValue `json:"inputFields"`
	EnumValues    []EnumValue  `json:"enumValues"`
	PossibleTypes []TypeRef    `json:"possibleTypes"`
	Interfaces    []TypeRef    `json:"interfaces"`
}

// Field represents a field on an OBJECT or INTERFACE type.
type Field struct {
	Name              string       `json:"name"`
	Description       string       `json:"description"`
	Args              []InputValue `json:"args"`
	Type              TypeRef      `json:"type"`
	IsDeprecated      bool         `json:"isDeprecated"`
	DeprecationReason string       `json:"deprecationReason"`
}

// InputValue represents a field argument or input object field.
type InputValue struct {
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	Type         TypeRef `json:"type"`
	DefaultValue *string `json:"defaultValue"`
}

// TypeRef represents a type reference that may be wrapped in NON_NULL/LIST.
type TypeRef struct {
	Kind   string   `json:"kind"`
	Name   *string  `json:"name"`
	OfType *TypeRef `json:"ofType"`
}

// DisplayName renders the type reference as a human-readable string.
func (t TypeRef) DisplayName() string {
	switch t.Kind {
	case "NON_NULL":
		if t.OfType != nil {
			return t.OfType.DisplayName() + "!"
		}
	case "LIST":
		if t.OfType != nil {
			return "[" + t.OfType.DisplayName() + "]"
		}
	default:
		if t.Name != nil {
			return *t.Name
		}
	}
	return "Unknown"
}

// NamedType unwraps NON_NULL/LIST wrappers and returns the innermost type name.
func (t TypeRef) NamedType() string {
	if t.Name != nil {
		return *t.Name
	}
	if t.OfType != nil {
		return t.OfType.NamedType()
	}
	return ""
}

// EnumValue represents a value of an ENUM type.
type EnumValue struct {
	Name              string `json:"name"`
	Description       string `json:"description"`
	IsDeprecated      bool   `json:"isDeprecated"`
	DeprecationReason string `json:"deprecationReason"`
}
