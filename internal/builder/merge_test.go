package builder

import (
	"encoding/json"
	"testing"
)

func TestMergeVariables(t *testing.T) {
	tests := []struct {
		name      string
		existing  string
		generated string
		want      map[string]any
	}{
		{
			name:      "empty existing uses generated",
			existing:  "",
			generated: `{"country": "example", "limit": 42}`,
			want:      map[string]any{"country": "example", "limit": float64(42)},
		},
		{
			name:      "empty generated returns empty",
			existing:  `{"country": "US"}`,
			generated: "",
			want:      nil, // empty string returned
		},
		{
			name:      "matching types preserve existing values",
			existing:  `{"country": "US", "limit": 10}`,
			generated: `{"country": "example", "limit": 42}`,
			want:      map[string]any{"country": "US", "limit": float64(10)},
		},
		{
			name:      "type mismatch uses generated",
			existing:  `{"country": 123}`,
			generated: `{"country": "example"}`,
			want:      map[string]any{"country": "example"},
		},
		{
			name:      "extra existing keys dropped",
			existing:  `{"country": "US", "obsolete": "gone"}`,
			generated: `{"country": "example"}`,
			want:      map[string]any{"country": "US"},
		},
		{
			name:      "new generated keys added",
			existing:  `{"country": "US"}`,
			generated: `{"country": "example", "limit": 42}`,
			want:      map[string]any{"country": "US", "limit": float64(42)},
		},
		{
			name:      "object type preserved",
			existing:  `{"filter": {"status": "active"}}`,
			generated: `{"filter": {"status": "example"}}`,
			want:      map[string]any{"filter": map[string]any{"status": "active"}},
		},
		{
			name:      "invalid JSON existing falls back to generated",
			existing:  `{not valid json`,
			generated: `{"country": "example"}`,
			want:      map[string]any{"country": "example"},
		},
		{
			name:      "bool preserved",
			existing:  `{"verbose": true}`,
			generated: `{"verbose": false}`,
			want:      map[string]any{"verbose": true},
		},
		{
			name:      "array preserved",
			existing:  `{"ids": [1, 2, 3]}`,
			generated: `{"ids": [42]}`,
			want:      map[string]any{"ids": []any{float64(1), float64(2), float64(3)}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeVariables(tt.existing, tt.generated)

			if tt.want == nil {
				if got != "" {
					t.Errorf("expected empty string, got %q", got)
				}
				return
			}

			var gotMap map[string]any
			if err := json.Unmarshal([]byte(got), &gotMap); err != nil {
				t.Fatalf("result is not valid JSON: %v\ngot: %s", err, got)
			}

			wantJSON, _ := json.Marshal(tt.want)
			gotJSON, _ := json.Marshal(gotMap)
			if string(gotJSON) != string(wantJSON) {
				t.Errorf("mismatch\nwant: %s\ngot:  %s", wantJSON, gotJSON)
			}
		})
	}
}
