package spec

import (
	"fmt"
	"reflect"

	"github.com/goccy/go-yaml"
)

// Overlay deep-merges overlay into base and returns a new document. Non-zero
// values in overlay override base; zero values (empty string, false, 0, nil,
// empty slices) leave base untouched. Maps are merged key-by-key (overlay wins
// on collision); non-empty slices replace.
//
// The merge is performed over an untyped map so it applies uniformly across the
// whole document, including the opaque *Bindings maps.
func Overlay(base, overlay *AsyncAPI) (*AsyncAPI, error) {
	bm, err := toMap(base)
	if err != nil {
		return nil, fmt.Errorf("converting base document: %w", err)
	}
	om, err := toMap(overlay)
	if err != nil {
		return nil, fmt.Errorf("converting overlay document: %w", err)
	}
	merged := mergeAny(bm, om).(map[string]any)

	raw, err := yaml.Marshal(merged)
	if err != nil {
		return nil, fmt.Errorf("marshaling merged document: %w", err)
	}
	var out AsyncAPI
	if err := yaml.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("unmarshaling merged document: %w", err)
	}
	return &out, nil
}

func toMap(a *AsyncAPI) (map[string]any, error) {
	if a == nil {
		return map[string]any{}, nil
	}
	raw, err := yaml.Marshal(a)
	if err != nil {
		return nil, fmt.Errorf("marshaling document: %w", err)
	}
	m := map[string]any{}
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("unmarshaling document: %w", err)
	}
	return m, nil
}

// mergeAny merges src into dst and returns the result.
func mergeAny(dst, src any) any {
	switch s := src.(type) {
	case map[string]any:
		d, ok := dst.(map[string]any)
		if !ok || d == nil {
			return s
		}
		for k, v := range s {
			if existing, ok := d[k]; ok {
				d[k] = mergeAny(existing, v)
			} else {
				d[k] = v
			}
		}
		return d
	case []any:
		if len(s) == 0 {
			return dst
		}
		return s
	default:
		if isZero(s) {
			return dst
		}
		return s
	}
}

func isZero(v any) bool {
	if v == nil {
		return true
	}
	return reflect.ValueOf(v).IsZero()
}
