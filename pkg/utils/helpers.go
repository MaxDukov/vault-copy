package utils

import (
	"strings"
)

// ContainsString проверяет наличие строки в слайсе
func ContainsString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// IsSubPath проверяет, является ли path подпутем root
func IsSubPath(path, root string) bool {
	if root == "" {
		return true
	}
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

// SafeMapLookup безопасное извлечение значения из map
func SafeMapLookup(m map[string]interface{}, key string) (interface{}, bool) {
	if m == nil {
		return nil, false
	}
	val, ok := m[key]
	return val, ok
}

// MergeMaps объединяет несколько map (последний имеет приоритет)
func MergeMaps(maps ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

// TrimPathPrefix удаляет префикс из пути
func TrimPathPrefix(path, prefix string) string {
	if !strings.HasPrefix(path, prefix) {
		return path
	}
	trimmed := strings.TrimPrefix(path, prefix)
	return strings.TrimPrefix(trimmed, "/")
}

// EnsureTrailingSlash добавляет завершающий слэш если его нет
func EnsureTrailingSlash(path string) string {
	if path == "" {
		return "/"
	}
	if !strings.HasSuffix(path, "/") {
		return path + "/"
	}
	return path
}

// SplitPath разбивает путь на части
func SplitPath(path string) []string {
	return strings.FieldsFunc(path, func(r rune) bool {
		return r == '/'
	})
}
