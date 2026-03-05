package builder

import (
	"encoding/json"
	"strings"
)

// MergeVariables merges existing variable values into a newly generated variables
// JSON string. For each key in generated: if the key exists in existing and the
// JSON types match, the existing value is preserved. Keys not in generated are dropped.
func MergeVariables(existing, generated string) string {
	existing = strings.TrimSpace(existing)
	generated = strings.TrimSpace(generated)

	if generated == "" {
		return ""
	}
	if existing == "" {
		return generated
	}

	var existingMap, generatedMap map[string]any
	if err := json.Unmarshal([]byte(existing), &existingMap); err != nil {
		return generated
	}
	if err := json.Unmarshal([]byte(generated), &generatedMap); err != nil {
		return generated
	}

	merged := make(map[string]any, len(generatedMap))
	for k, genVal := range generatedMap {
		if exVal, ok := existingMap[k]; ok && jsonTypesMatch(exVal, genVal) {
			merged[k] = exVal
		} else {
			merged[k] = genVal
		}
	}

	out, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return generated
	}
	return string(out)
}

// jsonTypesMatch returns true if a and b have the same JSON type
// (both nil, both bool, both number, both string, both array, or both object).
func jsonTypesMatch(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	switch a.(type) {
	case bool:
		_, ok := b.(bool)
		return ok
	case float64:
		_, ok := b.(float64)
		return ok
	case string:
		_, ok := b.(string)
		return ok
	case []any:
		_, ok := b.([]any)
		return ok
	case map[string]any:
		_, ok := b.(map[string]any)
		return ok
	default:
		return false
	}
}
