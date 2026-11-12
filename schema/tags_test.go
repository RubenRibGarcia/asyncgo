package schema

import (
	"reflect"
	"sort"
	"testing"

	"github.com/RubenRibGarcia/asyncgo/internal/taggrammar"
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
			name: "should_ignore_required",
			tag:  "required",
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
			name: "should_ignore_empty_enum",
			tag:  "enum=",
			verify: func(t *testing.T, s *spec.Schema) {
				assert.Empty(t, s.Enum)
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
			name: "should_apply_content_encoding",
			tag:  "contentEncoding=base64url",
			verify: func(t *testing.T, s *spec.Schema) {
				assert.Equal(t, "base64url", s.ContentEncoding)
			},
		},
		{
			name: "should_apply_content_media_type",
			tag:  "contentMediaType=image/png",
			verify: func(t *testing.T, s *spec.Schema) {
				assert.Equal(t, "image/png", s.ContentMediaType)
			},
		},
		{
			name: "should_ignore_empty_content_media_type",
			tag:  "contentMediaType=",
			verify: func(t *testing.T, s *spec.Schema) {
				assert.Empty(t, s.ContentMediaType)
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
			tag:  "required readOnly writeOnly uniqueItems deprecated",
			verify: func(t *testing.T, s *spec.Schema) {
				assert.True(t, s.ReadOnly)
				assert.True(t, s.WriteOnly)
				assert.True(t, s.UniqueItems)
				assert.True(t, s.Deprecated)
			},
		},
		{
			name: "should_append_repeated_examples",
			tag:  "examples=sku-1 examples=sku-2",
			verify: func(t *testing.T, s *spec.Schema) {
				assert.Equal(t, []any{"sku-1", "sku-2"}, s.Examples)
			},
		},
		{
			name: "should_keep_examples_when_a_removed_example_directive_is_present",
			tag:  "examples=draft-07 example=legacy",
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
			// Whitespace separates directives, so a comma is an ordinary value
			// character: "1,000" survives as a single example.
			name: "should_keep_a_comma_inside_a_value",
			tag:  "examples=1,000",
			verify: func(t *testing.T, s *spec.Schema) {
				assert.Equal(t, []any{"1,000"}, s.Examples)
			},
		},
		{
			// The separator changed from comma to whitespace, so a legacy tag is
			// now one unrecognized directive and sets nothing. Pinned so the
			// silent drop stays deliberate and visible.
			name: "should_ignore_a_legacy_comma_separated_tag",
			tag:  "required,readOnly",
			verify: func(t *testing.T, s *spec.Schema) {
				assert.False(t, s.ReadOnly)
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
			tag:  "minItems=1 maxItems=10 minProperties=1 maxProperties=8",
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
			applyTag(s, taggrammar.Parse(tc.tag))
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
			name:   "should_find_a_directive_after_whitespace",
			tag:    "required anyOf=A",
			key:    "anyOf",
			want:   []string{"A"},
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
			name:   "should_match_the_key_exactly",
			tag:    "anyOf=A",
			key:    "anyOf",
			want:   []string{"A"},
			wantOK: true,
		},
		{
			// The key is compared whole, so a longer directive name is not a
			// combinator of the key it merely starts with.
			name:   "should_not_match_a_longer_key",
			tag:    "anyOfAll=A",
			key:    "anyOf",
			want:   nil,
			wantOK: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := combinatorNames(taggrammar.Parse(tc.tag), tc.key)
			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

// minimalTags is one legal spelling per directive, and therefore the vocabulary
// the package promises to honour. It is a third copy on purpose: DirectiveKeys and
// applyTag's switch are the other two, and TestDirectiveVocabularyMatchesDerivation
// is what fails when they drift apart — the same drift guard ADR 0007 legislated
// for the grammar, applied to the vocabulary.
var minimalTags = map[string]string{
	"allOf":            "allOf=A|B",
	"anyOf":            "anyOf=A|B",
	"const":            "const=x",
	"contentEncoding":  "contentEncoding=base64",
	"contentMediaType": "contentMediaType=image/png",
	"deprecated":       "deprecated",
	"discriminator":    "discriminator=kind",
	"enum":             "enum=a|b",
	"examples":         "examples=x",
	"format":           "format=date-time",
	"maxItems":         "maxItems=1",
	"maxProperties":    "maxProperties=1",
	"minItems":         "minItems=1",
	"minProperties":    "minProperties=1",
	"oneOf":            "oneOf=A|B",
	"readOnly":         "readOnly",
	"required":         "required",
	"uniqueItems":      "uniqueItems",
	"writeOnly":        "writeOnly",
}

func TestDirectiveKeys(t *testing.T) {
	want := make([]string, 0, len(minimalTags))
	for key := range minimalTags {
		want = append(want, key)
	}
	sort.Strings(want)

	assert.Equal(t, want, DirectiveKeys())
}

// TestDirectiveVocabularyMatchesDerivation fails when the vocabulary and the code
// that reads it disagree — a key added to one and not the other, which would make
// CheckTag reject a working directive or accept a dead one.
func TestDirectiveVocabularyMatchesDerivation(t *testing.T) {
	// required and the three combinators are applied by fillFields, not applyTag.
	callerApplied := map[string]bool{
		"required": true,
		"oneOf":    true,
		"anyOf":    true,
		"allOf":    true,
	}

	for _, key := range DirectiveKeys() {
		t.Run("should_handle_"+key, func(t *testing.T) {
			tag := minimalTags[key]
			require.Empty(t, CheckTag(tag), "the vocabulary must accept its own spelling")

			s := &spec.Schema{}
			applyTag(s, taggrammar.Parse(tag))
			applied := !reflect.DeepEqual(*s, spec.Schema{})

			assert.Equal(t, !callerApplied[key], applied,
				"%q applied by applyTag = %v", key, applied)
		})
	}
}

func TestCheckTag(t *testing.T) {
	tests := []struct {
		name string
		tag  string
		// want is one substring per expected error; nil means the tag is clean.
		want []string
	}{
		{
			name: "should_accept_an_empty_tag",
			tag:  "",
		},
		{
			name: "should_accept_a_json_only_tag",
			tag:  "",
		},
		{
			name: "should_accept_every_known_directive_at_once",
			tag: "required readOnly writeOnly uniqueItems deprecated " +
				"enum=draft|sent examples=1,000 format=date-time " +
				"contentEncoding=base64 contentMediaType=image/png const=fixed " +
				"discriminator=kind minItems=1 maxItems=9 minProperties=1 " +
				"maxProperties=9 oneOf=A|B anyOf=A allOf=A",
		},
		{
			// allOf is the one directive legal in both forms: bare composes an
			// embedded struct, valued names the member types.
			name: "should_accept_allOf_bare_and_with_a_value",
			tag:  "allOf",
		},
		{
			name: "should_accept_a_value_containing_a_comma",
			tag:  "examples=1,000",
		},
		{
			name: "should_suggest_the_closest_directive",
			tag:  "requird",
			want: []string{`unknown directive "requird"; did you mean "required"?`},
		},
		{
			// The removed singular still suggests its replacement rather than
			// silently disappearing.
			name: "should_suggest_examples_for_the_removed_singular",
			tag:  "example=x",
			want: []string{`did you mean "examples"?`},
		},
		{
			name: "should_list_the_vocabulary_when_nothing_is_close",
			tag:  "zzz=x",
			want: []string{`unknown directive "zzz"; want one of allOf, anyOf, const`},
		},
		{
			// The old spelling, and the likeliest migration failure.
			name: "should_name_the_whitespace_rule_for_a_comma_separated_tag",
			tag:  "required,examples=x",
			want: []string{`"required,examples": directives are separated by spaces, not commas`},
		},
		{
			name: "should_reject_a_value_on_a_flag",
			tag:  "readOnly=true",
			want: []string{`"readOnly" is a flag and takes no value`},
		},
		{
			name: "should_reject_a_directive_without_a_value",
			tag:  "oneOf",
			want: []string{`"oneOf" requires a value`},
		},
		{
			name: "should_reject_an_empty_value",
			tag:  "format=",
			want: []string{`"format" has an empty value`},
		},
		{
			name: "should_reject_an_empty_list_entry",
			tag:  "oneOf=A|",
			want: []string{`"oneOf" has an empty entry`},
		},
		{
			name: "should_reject_a_directive_with_no_name",
			tag:  "=x",
			want: []string{"directive has no name"},
		},
		{
			name: "should_report_every_violation_not_only_the_first",
			tag:  "requird readOnly=true format=",
			want: []string{
				`unknown directive "requird"`,
				`"readOnly" is a flag and takes no value`,
				`"format" has an empty value`,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := CheckTag(tc.tag)
			require.Len(t, errs, len(tc.want))
			for i, want := range tc.want {
				assert.Contains(t, errs[i].Error(), want)
			}
		})
	}
}
