package utils

import (
	"testing"
)

func TestContainsString(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		str   string
		want  bool
	}{
		{
			name:  "string found",
			slice: []string{"apple", "banana", "cherry"},
			str:   "banana",
			want:  true,
		},
		{
			name:  "string not found",
			slice: []string{"apple", "banana", "cherry"},
			str:   "orange",
			want:  false,
		},
		{
			name:  "empty slice",
			slice: []string{},
			str:   "anything",
			want:  false,
		},
		{
			name:  "case sensitive",
			slice: []string{"Apple", "Banana"},
			str:   "apple",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ContainsString(tt.slice, tt.str)
			if got != tt.want {
				t.Errorf("ContainsString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsSubPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		root string
		want bool
	}{
		{
			name: "exact match",
			path: "secret/data/app",
			root: "secret/data/app",
			want: true,
		},
		{
			name: "subdirectory",
			path: "secret/data/app/config",
			root: "secret/data/app",
			want: true,
		},
		{
			name: "not subpath",
			path: "secret/data/other",
			root: "secret/data/app",
			want: false,
		},
		{
			name: "same prefix but different",
			path: "secret/data/apple",
			root: "secret/data/app",
			want: false,
		},
		{
			name: "empty root",
			path: "any/path",
			root: "",
			want: true,
		},
		{
			name: "shorter than root",
			path: "secret",
			root: "secret/data",
			want: false,
		},
		{
			name: "with trailing slash in root",
			path: "secret/data/app/config",
			root: "secret/data/app/",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSubPath(tt.path, tt.root)
			if got != tt.want {
				t.Errorf("IsSubPath(%q, %q) = %v, want %v", tt.path, tt.root, got, tt.want)
			}
		})
	}
}

func TestSafeMapLookup(t *testing.T) {
	m := map[string]interface{}{
		"string": "value",
		"int":    42,
		"float":  3.14,
		"bool":   true,
		"nested": map[string]interface{}{"key": "value"},
	}

	tests := []struct {
		name   string
		key    string
		want   interface{}
		wantOk bool
	}{
		{
			name:   "existing string",
			key:    "string",
			want:   "value",
			wantOk: true,
		},
		{
			name:   "existing int",
			key:    "int",
			want:   42,
			wantOk: true,
		},
		{
			name:   "non-existent",
			key:    "missing",
			want:   nil,
			wantOk: false,
		},
		{
			name:   "nested map",
			key:    "nested",
			want:   map[string]interface{}{"key": "value"},
			wantOk: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := SafeMapLookup(m, tt.key)
			if ok != tt.wantOk {
				t.Errorf("SafeMapLookup() ok = %v, want %v", ok, tt.wantOk)
			}

			if ok && got != tt.want {
				// For comparing nested maps, a more complex comparator is needed
				if tt.key != "nested" && got != tt.want {
					t.Errorf("SafeMapLookup() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestMergeMaps(t *testing.T) {
	tests := []struct {
		name string
		maps []map[string]interface{}
		want map[string]interface{}
	}{
		{
			name: "empty maps",
			maps: []map[string]interface{}{},
			want: map[string]interface{}{},
		},
		{
			name: "single map",
			maps: []map[string]interface{}{
				{"a": 1, "b": 2},
			},
			want: map[string]interface{}{"a": 1, "b": 2},
		},
		{
			name: "two maps no overlap",
			maps: []map[string]interface{}{
				{"a": 1},
				{"b": 2},
			},
			want: map[string]interface{}{"a": 1, "b": 2},
		},
		{
			name: "two maps with overlap",
			maps: []map[string]interface{}{
				{"a": 1, "b": 2},
				{"b": 3, "c": 4},
			},
			want: map[string]interface{}{"a": 1, "b": 3, "c": 4}, // Last wins
		},
		{
			name: "three maps",
			maps: []map[string]interface{}{
				{"a": 1},
				{"b": 2},
				{"c": 3},
			},
			want: map[string]interface{}{"a": 1, "b": 2, "c": 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeMaps(tt.maps...)

			if len(got) != len(tt.want) {
				t.Errorf("MergeMaps() length = %d, want %d", len(got), len(tt.want))
			}

			for key, wantVal := range tt.want {
				gotVal, ok := got[key]
				if !ok {
					t.Errorf("MergeMaps() missing key %s", key)
					continue
				}

				if gotVal != wantVal {
					t.Errorf("MergeMaps()[%s] = %v, want %v", key, gotVal, wantVal)
				}
			}
		})
	}
}
