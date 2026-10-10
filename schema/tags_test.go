package schema

import (
	"testing"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			// example= was removed along with spec.Schema.Example: 3.1.0 defines only
			// the Draft-07 `examples` array. applyTag ignores an unknown directive, so
			// a stale example= is dropped rather than quietly mapped onto examples=.
			// Pinned so that silent-ignore stays deliberate and visible.
			name: "should_ignore_the_removed_example_directive",
			tag:  "example=hello",
			verify: func(t *testing.T, s *spec.Schema) {
				assert.Empty(t, s.Examples)
				assert.Empty(t, s.Enum)
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
			},
		},
		{
			name: "should_apply_read_only_flag",
			tag:  "readOnly",
			verify: func(t *testing.T, s *spec.Schema) {
				assert.True(t, s.ReadOnly)
			},
		},
		{
			name: "should_apply_write_only_flag",
			tag:  "writeOnly",
			verify: func(t *testing.T, s *spec.Schema) {
				assert.True(t, s.WriteOnly)
			},
		},
		{
			name: "should_apply_unique_items_flag",
			tag:  "uniqueItems",
			verify: func(t *testing.T, s *spec.Schema) {
				assert.True(t, s.UniqueItems)
			},
		},
		{
			name: "should_apply_deprecated_flag",
			tag:  "deprecated",
			verify: func(t *testing.T, s *spec.Schema) {
				assert.True(t, s.Deprecated)
			},
		},
		{
			name: "should_apply_all_flags_together",
			tag:  "required,readOnly,writeOnly,uniqueItems,deprecated",
			verify: func(t *testing.T, s *spec.Schema) {
				assert.True(t, s.ReadOnly)
				assert.True(t, s.WriteOnly)
				assert.True(t, s.UniqueItems)
				assert.True(t, s.Deprecated)
			},
		},
		{
			name: "should_append_repeated_examples",
			tag:  "examples=sku-1,examples=sku-2",
			verify: func(t *testing.T, s *spec.Schema) {
				assert.Equal(t, []any{"sku-1", "sku-2"}, s.Examples)
			},
		},
		{
			name: "should_keep_examples_when_a_removed_example_directive_is_present",
			tag:  "examples=draft-07,example=legacy",
			verify: func(t *testing.T, s *spec.Schema) {
				assert.Equal(t, []any{"draft-07"}, s.Examples,
					"the removed example= is ignored, not merged into examples=")
			},
		},
		{
			name: "should_ignore_empty_examples",
			tag:  "examples=",
			verify: func(t *testing.T, s *spec.Schema) {
				assert.Empty(t, s.Examples)
			},
		},
		{
			name: "should_apply_const",
			tag:  "const=OrderPlaced",
			verify: func(t *testing.T, s *spec.Schema) {
				assert.Equal(t, "OrderPlaced", s.Const)
			},
		},
		{
			name: "should_ignore_empty_const",
			tag:  "const=",
			verify: func(t *testing.T, s *spec.Schema) {
				assert.Nil(t, s.Const)
			},
		},
		{
			name: "should_apply_discriminator",
			tag:  "discriminator=kind",
			verify: func(t *testing.T, s *spec.Schema) {
				assert.Equal(t, "kind", s.Discriminator)
			},
		},
		{
			name: "should_apply_numeric_bounds",
			tag:  "minItems=1,maxItems=10,minProperties=1,maxProperties=8",
			verify: func(t *testing.T, s *spec.Schema) {
				require.NotNil(t, s.MinItems)
				require.NotNil(t, s.MaxItems)
				require.NotNil(t, s.MinProperties)
				require.NotNil(t, s.MaxProperties)
				assert.Equal(t, uint64(1), *s.MinItems)
				assert.Equal(t, uint64(10), *s.MaxItems)
				assert.Equal(t, uint64(1), *s.MinProperties)
				assert.Equal(t, uint64(8), *s.MaxProperties)
			},
		},
		{
			name: "should_apply_zero_bound",
			tag:  "minItems=0",
			verify: func(t *testing.T, s *spec.Schema) {
				require.NotNil(t, s.MinItems, "zero is a real bound, not an absent one")
				assert.Equal(t, uint64(0), *s.MinItems)
			},
		},
		{
			name: "should_ignore_non_numeric_bound",
			tag:  "minItems=abc",
			verify: func(t *testing.T, s *spec.Schema) {
				assert.Nil(t, s.MinItems)
			},
		},
		{
			name: "should_ignore_negative_bound",
			tag:  "maxItems=-1",
			verify: func(t *testing.T, s *spec.Schema) {
				assert.Nil(t, s.MaxItems)
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
