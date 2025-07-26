package maputil

import "fmt"

// SetNestedMapValue sets a value in a nested map[string]any given a path of keys.
func SetNestedMapValue(m map[string]any, path []string, value any) error {
	if len(path) == 0 {
		return fmt.Errorf("empty path")
	}
	current := m
	for i, key := range path {
		if i == len(path)-1 {
			current[key] = value
			return nil
		}
		if next, ok := current[key]; ok {
			if nextMap, ok := next.(map[string]any); ok {
				current = nextMap
			} else {
				return fmt.Errorf("key '%s' is not a map at path %v", key, path[:i+1])
			}
		} else {
			nextMap := make(map[string]any)
			current[key] = nextMap
			current = nextMap
		}
	}
	return nil
}
