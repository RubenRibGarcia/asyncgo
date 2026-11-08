package taggrammar

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	t.Run("should_split_each_part_at_its_first_equals", func(t *testing.T) {
		assert.Equal(t, []Directive{
			{Key: "required"},
			{Key: "format", Value: "date-time", HasValue: true},
			{Key: "examples", Value: "a=b", HasValue: true},
			{Key: "format", HasValue: true},
		}, Parse("required format=date-time examples=a=b format="))
	})

	t.Run("should_return_nil_for_an_empty_tag", func(t *testing.T) {
		assert.Nil(t, Parse(""))
		assert.Nil(t, Parse("   "))
	})

	t.Run("should_keep_a_comma_inside_a_value", func(t *testing.T) {
		assert.Equal(
			t,
			[]Directive{{Key: "examples", Value: "1,000", HasValue: true}},
			Parse("examples=1,000"),
		)
	})
}

func TestFlag(t *testing.T) {
	tests := []struct {
		name string
		tag  string
		flag string
		want bool
	}{
		{name: "should_find_a_bare_flag", tag: "required readOnly", flag: "required", want: true},
		{name: "should_not_find_a_value_directive", tag: "required=true", flag: "required"},
		// "allOf" marks an embed for composition, "allOf=A" composes types.
		{name: "should_not_confuse_the_combinator_with_the_flag", tag: "allOf=A", flag: "allOf"},
		{name: "should_find_the_embed_flag", tag: "allOf", flag: "allOf", want: true},
		{name: "should_not_find_an_absent_flag", tag: "readOnly", flag: "required"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, Flag(Parse(tc.tag), tc.flag))
		})
	}
}

func TestValue(t *testing.T) {
	tests := []struct {
		name   string
		tag    string
		key    string
		want   string
		wantOK bool
	}{
		{
			name:   "should_return_the_value",
			tag:    "oneOf=A|B",
			key:    "oneOf",
			want:   "A|B",
			wantOK: true,
		},
		{
			name:   "should_find_a_directive_after_whitespace",
			tag:    "required anyOf=A",
			key:    "anyOf",
			want:   "A",
			wantOK: true,
		},
		{name: "should_report_an_empty_value", tag: "oneOf=", key: "oneOf", wantOK: true},
		{name: "should_not_match_a_bare_flag", tag: "oneOf", key: "oneOf"},
		{name: "should_not_match_a_longer_key", tag: "anyOfAll=A", key: "anyOf"},
		{name: "should_return_false_when_absent", tag: "required", key: "oneOf"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := Value(Parse(tc.tag), tc.key)
			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestJSONName(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		wantName string
		wantSkip bool
	}{
		{name: "should_return_the_declared_name", tag: "orderId,omitempty", wantName: "orderId"},
		{name: "should_skip_a_dash", tag: "-", wantSkip: true},
		// Only an exact "-" skips; "-" with options names a field "-", as before.
		{name: "should_name_a_dash_with_options", tag: "-,omitempty", wantName: "-"},
		// No name declared: the caller falls back to the Go field name.
		{name: "should_report_no_name_for_options_only", tag: ",omitempty"},
		{name: "should_report_no_name_for_an_empty_tag", tag: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			name, skip := JSONName(tc.tag)
			assert.Equal(t, tc.wantName, name)
			assert.Equal(t, tc.wantSkip, skip)
		})
	}
}
