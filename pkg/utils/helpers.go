package utils

import (
	"strings"
)

// ContainsString checks if a string is present in a slice
func ContainsString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// IsSubPath checks if path is a subpath of root
func IsSubPath(path, root string) bool {
	if root == "" {
		return true
	}

	// Remove trailing slash from root for comparison
	root = strings.TrimSuffix(root, "/")

	if len(path) < len(root) {
		return false
	}
	if path[:len(root)] != root {
		return false
	}
	if len(path) == len(root) {
		return true
	}
	return path[len(root)] == '/'
}

// SafeMapLookup safely extracts a value from a map
func SafeMapLookup(m map[string]interface{}, key string) (interface{}, bool) {
	if m == nil {
		return nil, false
	}
	val, ok := m[key]
	return val, ok
}

// MergeMaps merges multiple maps (last one has priority)
func MergeMaps(maps ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

// TrimPathPrefix removes prefix from path
func TrimPathPrefix(path, prefix string) string {
	if !strings.HasPrefix(path, prefix) {
		return path
	}
	trimmed := strings.TrimPrefix(path, prefix)
	return strings.TrimPrefix(trimmed, "/")
}

// EnsureTrailingSlash adds trailing slash if it doesn't exist
func EnsureTrailingSlash(path string) string {
	if path == "" {
		return "/"
	}
	if !strings.HasSuffix(path, "/") {
		return path + "/"
	}
	return path
}

// SplitPath splits path into parts
func SplitPath(path string) []string {
	return strings.FieldsFunc(path, func(r rune) bool {
		return r == '/'
	})
}
