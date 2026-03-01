package schema

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// browserItem implements list.DefaultItem for the schema browser.
type browserItem struct {
	name       string // display name (field name, type name, enum value)
	desc       string // description line (type signature, field count, etc.)
	badge      string // kind badge text: OBJECT, ENUM, etc. (empty for fields)
	target     string // type name for drill-in (empty = not drillable)
	deprecated bool
	dimNote    string // deprecation reason

	// Structured field data for color-coded rendering (set only for fields)
	fieldName     string // e.g. "user"
	fieldArgs     string // e.g. "(id: ID!)"
	fieldType     string // e.g. "User"
	fieldTypeKind string // e.g. "OBJECT", "SCALAR", "ENUM", etc.

	// Cross-level search: when non-empty, this item came from another type
	// and should render with a "ParentName › " prefix.
	searchParent string
}

// searchableItem pairs a browserItem with its parent context for cross-level search.
type searchableItem struct {
	item       browserItem
	parentName string
	parentKind string
}

func (i browserItem) Title() string       { return i.name }
func (i browserItem) Description() string { return i.desc }
func (i browserItem) FilterValue() string { return i.name + " " + i.desc }
func (i browserItem) Drillable() bool     { return i.target != "" }

// scrollableText returns the text that can be marquee-scrolled for this item.
// For fields: "fieldName(args): " (the type is kept fixed).
// For non-fields: the full name + description.
func (i browserItem) scrollableText() string {
	if i.fieldName != "" {
		s := i.fieldName
		if i.fieldArgs != "" {
			s += i.fieldArgs
		}
		s += ": "
		return s
	}
	// Non-field items: name + desc
	s := i.name
	if i.desc != "" {
		s += "  " + i.desc
	}
	return s
}

// fixedSuffix returns the text that should always remain visible (not scrolled).
// For fields: the type string. For non-fields: empty.
func (i browserItem) fixedSuffix() string {
	if i.fieldName != "" {
		return i.fieldType
	}
	return ""
}

// scrollableWidth returns the available width for the scrollable portion
// given the total content width (after prefix, before right-side badge/arrow).
func (i browserItem) scrollableWidth(contentWidth int) int {
	if i.fieldName == "" {
		return contentWidth
	}
	// Reserve space for the colored type
	typeW := lipgloss.Width(i.fieldType)
	w := contentWidth - typeW
	if w < 1 {
		w = 1
	}
	return w
}

// --- Page-building helpers ---

// targetVariableTypes is a synthetic target for the "Variable Types" group.
const targetVariableTypes = "__variable_types__"

func rootItems(s *Schema) []browserItem {
	roots := s.RootTypes()
	items := make([]browserItem, 0, len(roots)+1)
	for _, rt := range roots {
		count := len(rt.Fields)
		desc := fmt.Sprintf("%d fields", count)
		items = append(items, browserItem{
			name:   rt.Name,
			badge:  rt.Kind,
			target: rt.Name,
			desc:   desc,
		})
	}

	// Count variable types (INPUT_OBJECT and ENUM, excluding internal types)
	count := 0
	for _, t := range s.Types {
		if strings.HasPrefix(t.Name, "__") {
			continue
		}
		if t.Kind == "INPUT_OBJECT" || t.Kind == "ENUM" {
			count++
		}
	}
	if count > 0 {
		items = append(items, browserItem{
			name:   "Variable Types",
			desc:   fmt.Sprintf("%d types", count),
			target: targetVariableTypes,
		})
	}

	return items
}

func variableTypeItems(s *Schema) []browserItem {
	var items []browserItem
	for _, t := range s.Types {
		if strings.HasPrefix(t.Name, "__") {
			continue
		}
		switch t.Kind {
		case "INPUT_OBJECT":
			desc := fmt.Sprintf("%d fields", len(t.InputFields))
			items = append(items, browserItem{
				name:   t.Name,
				badge:  t.Kind,
				target: t.Name,
				desc:   desc,
			})
		case "ENUM":
			desc := fmt.Sprintf("%d values", len(t.EnumValues))
			items = append(items, browserItem{
				name:   t.Name,
				badge:  t.Kind,
				target: t.Name,
				desc:   desc,
			})
		}
	}
	return items
}

func typeItems(s *Schema, name string) []browserItem {
	t := s.TypeByName(name)
	if t == nil {
		return nil
	}
	var items []browserItem

	switch t.Kind {
	case "OBJECT", "INTERFACE":
		for _, iface := range t.Interfaces {
			ifName := iface.NamedType()
			items = append(items, browserItem{
				name:   "implements " + ifName,
				badge:  "INTERFACE",
				target: ifName,
			})
		}
		for _, f := range t.Fields {
			items = append(items, fieldItem(s, f))
		}

	case "ENUM":
		for _, ev := range t.EnumValues {
			items = append(items, browserItem{
				name:       ev.Name,
				desc:       ev.Description,
				deprecated: ev.IsDeprecated,
				dimNote:    ev.DeprecationReason,
			})
		}

	case "INPUT_OBJECT":
		for _, iv := range t.InputFields {
			named := iv.Type.NamedType()
			target := ""
			if isDrillable(s, named) {
				target = named
			}
			items = append(items, browserItem{
				name:   iv.Name,
				desc:   iv.Type.DisplayName(),
				target: target,
			})
		}

	case "UNION":
		for _, pt := range t.PossibleTypes {
			ptName := pt.NamedType()
			items = append(items, browserItem{
				name:   ptName,
				badge:  "OBJECT",
				target: ptName,
			})
		}
	}

	return items
}

func fieldItem(s *Schema, f Field) browserItem {
	title := f.Name
	if len(f.Args) > 0 {
		title += "(" + formatArgs(f.Args) + ")"
	}
	title += ": " + f.Type.DisplayName()

	named := f.Type.NamedType()
	target := ""
	if isDrillable(s, named) {
		target = named
	}

	dimNote := ""
	if f.DeprecationReason != "" {
		dimNote = "deprecated: " + f.DeprecationReason
	}

	// Build structured field data for color-coded rendering
	fArgs := ""
	if len(f.Args) > 0 {
		fArgs = "(" + formatArgs(f.Args) + ")"
	}

	// Resolve the kind of the named return type
	typeKind := ""
	if named != "" {
		if t := s.TypeByName(named); t != nil {
			typeKind = t.Kind
		}
	}

	return browserItem{
		name:          title,
		desc:          f.Description,
		target:        target,
		deprecated:    f.IsDeprecated,
		dimNote:       dimNote,
		fieldName:     f.Name,
		fieldArgs:     fArgs,
		fieldType:     f.Type.DisplayName(),
		fieldTypeKind: typeKind,
	}
}

func formatArgs(args []InputValue) string {
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = a.Name + ": " + a.Type.DisplayName()
	}
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += ", "
		}
		result += p
	}
	return result
}

func isDrillable(s *Schema, name string) bool {
	if name == "" {
		return false
	}
	t := s.TypeByName(name)
	if t == nil {
		return false
	}
	switch t.Kind {
	case "OBJECT", "INTERFACE", "ENUM", "INPUT_OBJECT", "UNION":
		return true
	}
	return false
}

// allSearchableItems builds a flat index of every searchable item across the
// schema, each tagged with its parent type name and kind. Internal types
// (prefixed with __) are skipped. Each type is visited at most once to avoid
// infinite recursion from circular type references.
func allSearchableItems(s *Schema) []searchableItem {
	if s == nil {
		return nil
	}
	visited := make(map[string]bool, len(s.Types))
	var result []searchableItem

	for _, t := range s.Types {
		if strings.HasPrefix(t.Name, "__") {
			continue
		}
		if visited[t.Name] {
			continue
		}
		visited[t.Name] = true

		switch t.Kind {
		case "OBJECT", "INTERFACE":
			for _, f := range t.Fields {
				result = append(result, searchableItem{
					item:       fieldItem(s, f),
					parentName: t.Name,
					parentKind: t.Kind,
				})
			}
		case "ENUM":
			// Add the type itself as a searchable item
			result = append(result, searchableItem{
				item: browserItem{
					name:   t.Name,
					badge:  t.Kind,
					target: t.Name,
					desc:   fmt.Sprintf("%d values", len(t.EnumValues)),
				},
				parentName: "Variable Types",
			})
			for _, ev := range t.EnumValues {
				result = append(result, searchableItem{
					item: browserItem{
						name:       ev.Name,
						desc:       ev.Description,
						deprecated: ev.IsDeprecated,
						dimNote:    ev.DeprecationReason,
					},
					parentName: t.Name,
					parentKind: t.Kind,
				})
			}
		case "INPUT_OBJECT":
			// Add the type itself as a searchable item
			result = append(result, searchableItem{
				item: browserItem{
					name:   t.Name,
					badge:  t.Kind,
					target: t.Name,
					desc:   fmt.Sprintf("%d fields", len(t.InputFields)),
				},
				parentName: "Variable Types",
			})
			for _, iv := range t.InputFields {
				named := iv.Type.NamedType()
				target := ""
				if isDrillable(s, named) {
					target = named
				}
				result = append(result, searchableItem{
					item: browserItem{
						name:   iv.Name,
						desc:   iv.Type.DisplayName(),
						target: target,
					},
					parentName: t.Name,
					parentKind: t.Kind,
				})
			}
		case "UNION":
			for _, pt := range t.PossibleTypes {
				ptName := pt.NamedType()
				result = append(result, searchableItem{
					item: browserItem{
						name:   ptName,
						badge:  "OBJECT",
						target: ptName,
					},
					parentName: t.Name,
					parentKind: t.Kind,
				})
			}
		}
	}

	return result
}
