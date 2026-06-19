package util

import (
	"encoding/json"
)

// ToStringMap converts any supported type to map[string]any.
// Supports: map[string]any, map[string]string, []byte (JSON), structs (via JSON).
// Returns nil for unsupported types or JSON errors.
// Empty []byte returns an empty map (not nil).
func ToStringMap(data any) map[string]any {
	if data == nil {
		return nil
	}
	switch v := data.(type) {
	case map[string]any:
		return v
	case map[string]string:
		m := make(map[string]any, len(v))
		for k, val := range v {
			m[k] = val
		}
		return m
	case []byte:
		if len(v) == 0 {
			return map[string]any{}
		}
		var m map[string]any
		if err := json.Unmarshal(v, &m); err != nil {
			return nil
		}
		return m
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil
		}
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			return nil
		}
		return m
	}
}
