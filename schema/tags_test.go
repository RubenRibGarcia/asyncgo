package schema

import (
	"testing"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
)

func TestApplyTag(t *testing.T) {
	tests := []struct {
		name   string
		tag    string
		verify func(*testing.T, *spec.Schema)
	}{
		{
			name: "should_ignore_required_and_empty_parts",
			tag:  "required,,",
			verify: func(t *testing.T, s *spec.Schema) {
				assert.Empty(t, s.Enum)
				assert.Nil(t, s.Example)
				assert.Empty(t, s.Format)
			},
		},
		{
			name: "should_apply_enum_values",
			tag:  "enum=a|b|c",
			verify: func(t *testing.T, s *spec.Schema) {
				assert.Equal(t, []any{"a", "b", "c"}, s.Enum)
			},
		},
		{
			name: "should_apply_example",
			tag:  "example=hello",
			verify: func(t *testing.T, s *spec.Schema) {
				assert.Equal(t, "hello", s.Example)
			},
		},
		{
			name: "should_apply_format",
			tag:  "format=uuid",
			verify: func(t *testing.T, s *spec.Schema) {
				assert.Equal(t, "uuid", s.Format)
			},
		},
		{
			name: "should_ignore_unknown_directives",
			tag:  "unknown=xyz",
			verify: func(t *testing.T, s *spec.Schema) {
				assert.Empty(t, s.Enum)
				assert.Empty(t, s.Format)
				assert.Nil(t, s.Example)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &spec.Schema{}
			applyTag(s, tc.tag)
			tc.verify(t, s)
		})
	}
}

func TestCombinatorNames(t *testing.T) {
	tests := []struct {
		name   string
		tag    string
		key    string
		want   []string
		wantOK bool
	}{
		{
			name:   "should_return_names_when_present",
			tag:    "oneOf=A|B",
			key:    "oneOf",
			want:   []string{"A", "B"},
			wantOK: true,
		},
		{
			name:   "should_return_false_when_absent",
			tag:    "required",
			key:    "oneOf",
			want:   nil,
			wantOK: false,
		},
		{
			name:   "should_return_false_when_value_empty",
			tag:    "oneOf=",
			key:    "oneOf",
			want:   nil,
			wantOK: false,
		},
		{
			name:   "should_match_key_by_prefix",
			tag:    "anyOf=A",
			key:    "anyOf",
			want:   []string{"A"},
			wantOK: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := combinatorNames(tc.tag, tc.key)
			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}
