package validate

import (
	"fmt"
	"strings"

	"github.com/qraqula/qla/internal/schema"
)

// IntrospectionToSDL converts an introspection schema to a GraphQL SDL string
// suitable for parsing by gqlparser.
func IntrospectionToSDL(s *schema.Schema) string {
	var buf strings.Builder

	// Schema definition
	buf.WriteString("schema {\n")
	if s.QueryType != nil && s.QueryType.Name != nil {
		fmt.Fprintf(&buf, "  query: %s\n", *s.QueryType.Name)
	}
	if s.MutationType != nil && s.MutationType.Name != nil {
		fmt.Fprintf(&buf, "  mutation: %s\n", *s.MutationType.Name)
	}
	if s.SubscriptionType != nil && s.SubscriptionType.Name != nil {
		fmt.Fprintf(&buf, "  subscription: %s\n", *s.SubscriptionType.Name)
	}
	buf.WriteString("}\n\n")

	for _, t := range s.Types {
		// Skip introspection types
		if strings.HasPrefix(t.Name, "__") {
			continue
		}
		// Skip built-in scalars
		if isBuiltinScalar(t.Name) && t.Kind == "SCALAR" {
			continue
		}

		switch t.Kind {
		case "OBJECT":
			writeObject(&buf, t)
		case "INPUT_OBJECT":
			writeInputObject(&buf, t)
		case "ENUM":
			writeEnum(&buf, t)
		case "INTERFACE":
			writeInterface(&buf, t)
		case "UNION":
			writeUnion(&buf, t)
		case "SCALAR":
			fmt.Fprintf(&buf, "scalar %s\n\n", t.Name)
		}
	}

	return buf.String()
}

func isBuiltinScalar(name string) bool {
	switch name {
	case "String", "Int", "Float", "Boolean", "ID":
		return true
	}
	return false
}

func writeObject(buf *strings.Builder, t schema.FullType) {
	fmt.Fprintf(buf, "type %s", t.Name)
	if len(t.Interfaces) > 0 {
		names := make([]string, len(t.Interfaces))
		for i, iface := range t.Interfaces {
			names[i] = iface.NamedType()
		}
		fmt.Fprintf(buf, " implements %s", strings.Join(names, " & "))
	}
	buf.WriteString(" {\n")
	for _, f := range t.Fields {
		writeField(buf, f)
	}
	buf.WriteString("}\n\n")
}

func writeInputObject(buf *strings.Builder, t schema.FullType) {
	fmt.Fprintf(buf, "input %s {\n", t.Name)
	for _, f := range t.InputFields {
		fmt.Fprintf(buf, "  %s: %s", f.Name, typeRefToSDL(f.Type))
		if f.DefaultValue != nil {
			fmt.Fprintf(buf, " = %s", *f.DefaultValue)
		}
		buf.WriteByte('\n')
	}
	buf.WriteString("}\n\n")
}

func writeEnum(buf *strings.Builder, t schema.FullType) {
	fmt.Fprintf(buf, "enum %s {\n", t.Name)
	for _, v := range t.EnumValues {
		fmt.Fprintf(buf, "  %s\n", v.Name)
	}
	buf.WriteString("}\n\n")
}

func writeInterface(buf *strings.Builder, t schema.FullType) {
	fmt.Fprintf(buf, "interface %s {\n", t.Name)
	for _, f := range t.Fields {
		writeField(buf, f)
	}
	buf.WriteString("}\n\n")
}

func writeUnion(buf *strings.Builder, t schema.FullType) {
	names := make([]string, len(t.PossibleTypes))
	for i, pt := range t.PossibleTypes {
		names[i] = pt.NamedType()
	}
	fmt.Fprintf(buf, "union %s = %s\n\n", t.Name, strings.Join(names, " | "))
}

func writeField(buf *strings.Builder, f schema.Field) {
	fmt.Fprintf(buf, "  %s", f.Name)
	if len(f.Args) > 0 {
		buf.WriteByte('(')
		for i, arg := range f.Args {
			if i > 0 {
				buf.WriteString(", ")
			}
			fmt.Fprintf(buf, "%s: %s", arg.Name, typeRefToSDL(arg.Type))
			if arg.DefaultValue != nil {
				fmt.Fprintf(buf, " = %s", *arg.DefaultValue)
			}
		}
		buf.WriteByte(')')
	}
	fmt.Fprintf(buf, ": %s\n", typeRefToSDL(f.Type))
}

func typeRefToSDL(t schema.TypeRef) string {
	switch t.Kind {
	case "NON_NULL":
		if t.OfType != nil {
			return typeRefToSDL(*t.OfType) + "!"
		}
	case "LIST":
		if t.OfType != nil {
			return "[" + typeRefToSDL(*t.OfType) + "]"
		}
	default:
		if t.Name != nil {
			return *t.Name
		}
	}
	return "Unknown"
}
