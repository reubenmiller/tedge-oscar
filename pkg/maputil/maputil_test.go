package maputil

import (
	"reflect"
	"testing"
)

func TestSetNestedMapValue(t *testing.T) {
	tests := []struct {
		name    string
		path    []string
		value   any
		start   map[string]any
		expect  map[string]any
		wantErr bool
	}{
		{
			name:    "simple set",
			path:    []string{"foo"},
			value:   42,
			start:   map[string]any{},
			expect:  map[string]any{"foo": 42},
			wantErr: false,
		},
		{
			name:    "nested set",
			path:    []string{"a", "b", "c"},
			value:   "bar",
			start:   map[string]any{},
			expect:  map[string]any{"a": map[string]any{"b": map[string]any{"c": "bar"}}},
			wantErr: false,
		},
		{
			name:    "overwrite existing",
			path:    []string{"x", "y"},
			value:   99,
			start:   map[string]any{"x": map[string]any{"y": 1}},
			expect:  map[string]any{"x": map[string]any{"y": 99}},
			wantErr: false,
		},
		{
			name:    "error on non-map",
			path:    []string{"foo", "bar"},
			value:   1,
			start:   map[string]any{"foo": 123},
			expect:  map[string]any{"foo": 123},
			wantErr: true,
		},
		{
			name:    "empty path",
			path:    []string{},
			value:   1,
			start:   map[string]any{},
			expect:  map[string]any{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := copyMap(tt.start)
			err := SetNestedMapValue(m, tt.path, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("expected error: %v, got: %v", tt.wantErr, err)
			}
			if !reflect.DeepEqual(m, tt.expect) {
				t.Errorf("expected map: %#v, got: %#v", tt.expect, m)
			}
		})
	}
}

func copyMap(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	out := make(map[string]any, len(src))
	for k, v := range src {
		if m, ok := v.(map[string]any); ok {
			out[k] = copyMap(m)
		} else {
			out[k] = v
		}
	}
	return out
}
