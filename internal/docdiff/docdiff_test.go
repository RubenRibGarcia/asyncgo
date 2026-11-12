package docdiff_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/RubenRibGarcia/asyncgo/internal/docdiff"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// doc is one side of a comparison, kept as source text so a test can mix the two
// encodings the CLI supports.
type doc struct {
	format string
	src    string
}

// decode turns the source into the generic document the walker consumes, using
// the decoder the CLI uses for that format: goccy/go-yaml for YAML, and
// encoding/json with UseNumber for JSON so an integer keeps its digits.
func (d doc) decode(t *testing.T) any {
	t.Helper()
	if d.format == "json" {
		dec := json.NewDecoder(strings.NewReader(d.src))
		dec.UseNumber()
		var out any
		require.NoError(t, dec.Decode(&out))
		return out
	}
	var out any
	require.NoError(t, yaml.Unmarshal([]byte(d.src), &out))
	return out
}

// describe renders changes the way a reviewer reads them: path, verdict, and the
// values on both sides.
func describe(changes []docdiff.Change) []string {
	var lines []string
	for _, c := range changes {
		lines = append(
			lines,
			fmt.Sprintf(
				"%s %s [%s -> %s]",
				c.Path,
				c.Kind,
				docdiff.FormatValue(c.Before),
				docdiff.FormatValue(c.After),
			),
		)
	}
	return lines
}

func TestChanges(t *testing.T) {
	yaml := func(src string) doc { return doc{format: "yaml", src: src} }
	jsonDoc := func(src string) doc { return doc{format: "json", src: src} }

	tt := []struct {
		name   string
		before doc
		after  doc
		want   []string
	}{
		{
			name:   "should_return_nothing_for_identical_documents",
			before: yaml("channels:\n  order-placed:\n    address: orders\n"),
			after:  yaml("channels:\n  order-placed:\n    address: orders\n"),
			want:   nil,
		},
		{
			name:   "should_report_an_added_key",
			before: yaml("channels: {}\n"),
			after:  yaml("channels:\n  order-placed:\n    address: orders\n"),
			want:   []string{`channels.order-placed added [null -> {"address":"orders"}]`},
		},
		{
			name:   "should_report_a_removed_key",
			before: yaml("channels:\n  order-placed:\n    address: orders\n"),
			after:  yaml("channels: {}\n"),
			want:   []string{`channels.order-placed removed [{"address":"orders"} -> null]`},
		},
		{
			name:   "should_locate_a_nested_change",
			before: yaml("channels:\n  order-placed:\n    address: orders\n"),
			after:  yaml("channels:\n  order-placed:\n    address: order.placed\n"),
			want: []string{
				`channels.order-placed.address changed ["orders" -> "order.placed"]`,
			},
		},
		{
			// A whole schema is one entry, not one per property: the removed
			// subtree is the value.
			name: "should_report_a_removed_subtree_once",
			before: yaml(`
components:
  schemas:
    Order:
      properties:
        id: {type: string}
`),
			after: yaml("components:\n  schemas: {}\n"),
			want: []string{
				`components.schemas.Order removed [{"properties":{"id":{"type":"string"}}} -> null]`,
			},
		},
		{
			name:   "should_report_a_sequence_change_as_one_value",
			before: yaml("components:\n  schemas:\n    Order:\n      required: []\n"),
			after:  yaml("components:\n  schemas:\n    Order:\n      required: [id]\n"),
			want: []string{
				`components.schemas.Order.required changed [[] -> ["id"]]`,
			},
		},
		{
			name:   "should_report_a_reordered_sequence",
			before: yaml("components:\n  schemas:\n    Order:\n      required: [id, total]\n"),
			after:  yaml("components:\n  schemas:\n    Order:\n      required: [total, id]\n"),
			want: []string{
				`components.schemas.Order.required changed [["id","total"] -> ["total","id"]]`,
			},
		},
		{
			name:   "should_report_a_type_change",
			before: yaml("channels:\n  c:\n    address: orders\n"),
			after:  yaml("channels:\n  c:\n    address:\n      template: orders\n"),
			want: []string{
				`channels.c.address changed ["orders" -> {"template":"orders"}]`,
			},
		},
		{
			name:   "should_report_a_mapping_replaced_by_a_scalar",
			before: yaml("channels:\n  c:\n    address:\n      template: orders\n"),
			after:  yaml("channels:\n  c:\n    address: orders\n"),
			want: []string{
				`channels.c.address changed [{"template":"orders"} -> "orders"]`,
			},
		},
		{
			name: "should_ignore_an_identical_sequence_of_objects",
			before: yaml(`
components:
  schemas:
    X:
      allOf:
      - type: object
`),
			after: yaml(`
components:
  schemas:
    X:
      allOf:
      - type: object
`),
			want: nil,
		},
		{
			name: "should_report_a_change_inside_a_sequence_of_objects",
			before: yaml(`
components:
  schemas:
    X:
      allOf:
      - type: object
`),
			after: yaml(`
components:
  schemas:
    X:
      allOf:
      - type: string
`),
			want: []string{
				`components.schemas.X.allOf changed [[{"type":"object"}] -> [{"type":"string"}]]`,
			},
		},
		{
			name:   "should_ignore_equal_negative_numbers",
			before: yaml("x: -7\n"),
			after:  yaml("x: -7\n"),
			want:   nil,
		},
		{
			name:   "should_ignore_equal_decimal_numbers",
			before: yaml("x: 3.5\n"),
			after:  yaml("x: 3.5\n"),
			want:   nil,
		},
		{
			// The same number is uint64 in YAML and json.Number in JSON; the
			// comparison is by value, so it is not a change.
			name:   "should_not_report_an_equal_number_across_decoders",
			before: yaml("x: 3\n"),
			after:  jsonDoc(`{"x":3}`),
			want:   nil,
		},
		{
			name:   "should_treat_an_integer_and_a_decimal_as_equal",
			before: yaml("x: 3\n"),
			after:  jsonDoc(`{"x":3.0}`),
			want:   nil,
		},
		{
			// goccy/go-yaml reads 1e3 as a string, not a number; reporting that
			// difference is honest rather than papering over it.
			name:   "should_report_a_yaml_exponent_string_as_changed",
			before: yaml("x: 1e3\n"),
			after:  jsonDoc(`{"x":1000}`),
			want:   []string{`x changed ["1e3" -> 1000]`},
		},
		{
			name:   "should_keep_a_string_and_a_number_apart",
			before: yaml(`x: "3"` + "\n"),
			after:  yaml("x: 3\n"),
			want:   []string{`x changed ["3" -> 3]`},
		},
		{
			name:   "should_quote_a_key_that_is_not_an_identifier",
			before: yaml("components:\n  schemas: {}\n"),
			after: yaml(`
components:
  schemas:
    example.com/orders.Order:
      type: object
`),
			want: []string{
				`components.schemas["example.com/orders.Order"] added [null -> {"type":"object"}]`,
			},
		},
		{
			name:   "should_report_an_added_empty_mapping",
			before: yaml("channels: {}\n"),
			after:  yaml("channels: {}\ncomponents: {}\n"),
			want:   []string{`components added [null -> {}]`},
		},
		{
			name:   "should_report_null_against_a_value",
			before: yaml("channels:\n  c:\n    address: null\n"),
			after:  yaml("channels:\n  c:\n    address: orders\n"),
			want:   []string{`channels.c.address changed [null -> "orders"]`},
		},
		{
			name:   "should_sort_changes_by_path",
			before: yaml("channels:\n  z: {}\n  a: {}\nservers: {}\n"),
			after:  yaml("channels:\n  z: {}\n  a: {}\nservers:\n  kafka: {}\n"),
			want:   []string{`servers.kafka added [null -> {}]`},
		},
		{
			name:   "should_sort_removals_before_additions_alphabetically",
			before: yaml("channels:\n  z: {}\n"),
			after:  yaml("channels:\n  a: {}\n"),
			want: []string{
				`channels.a added [null -> {}]`,
				`channels.z removed [{} -> null]`,
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got := docdiff.Changes(tc.before.decode(t), tc.after.decode(t))
			assert.Equal(t, tc.want, describe(got))
		})
	}
}

func TestChangesForGoValues(t *testing.T) {
	t.Run("should_treat_an_int_and_a_uint64_as_equal", func(t *testing.T) {
		got := docdiff.Changes(
			map[string]any{"x": int(3)},
			map[string]any{"x": uint64(3)},
		)

		assert.Empty(t, got)
	})
}

func TestFormatValue(t *testing.T) {
	t.Run("should_render_an_object_as_compact_sorted_json", func(t *testing.T) {
		value := map[string]any{"order": "v2", "address": "orders"}

		assert.Equal(t, `{"address":"orders","order":"v2"}`, docdiff.FormatValue(value))
	})

	t.Run("should_render_null", func(t *testing.T) {
		assert.Equal(t, "null", docdiff.FormatValue(nil))
	})

	t.Run("should_render_a_yaml_binary_scalar_as_base64", func(t *testing.T) {
		assert.Equal(t, `"aGVsbG8="`, docdiff.FormatValue([]byte("hello")))
	})

	t.Run("should_not_html_escape_a_string", func(t *testing.T) {
		assert.Equal(t, `"<orders&payments>"`, docdiff.FormatValue("<orders&payments>"))
	})

	t.Run("should_truncate_a_long_value_on_a_rune_boundary", func(t *testing.T) {
		got := docdiff.FormatValue(strings.Repeat("é", 100))

		assert.Len(t, []rune(got), 60)
		assert.True(t, strings.HasSuffix(got, `"...`), "a clipped string stays a complete literal")
	})

	t.Run("should_fall_back_for_a_value_json_cannot_encode", func(t *testing.T) {
		assert.Equal(t, "NaN", docdiff.FormatValue(math.NaN()))
	})

	t.Run("should_clip_a_long_binary_value_as_base64", func(t *testing.T) {
		got := docdiff.FormatValue(bytes.Repeat([]byte("x"), 100))

		assert.True(t, strings.HasPrefix(got, `"`))
		assert.True(t, strings.HasSuffix(got, `...`))
	})

	t.Run("should_clip_a_long_scalar", func(t *testing.T) {
		got := docdiff.FormatValue(json.Number(strings.Repeat("1", 100)))

		assert.Len(t, got, 60)
		assert.True(t, strings.HasSuffix(got, "..."))
	})

	t.Run("should_summarise_a_large_object_by_its_key_count", func(t *testing.T) {
		value := map[string]any{
			"address":  strings.Repeat("x", 40),
			"bindings": strings.Repeat("y", 40),
			"messages": strings.Repeat("z", 40),
		}

		assert.Equal(t, "{3 keys}", docdiff.FormatValue(value))
	})

	t.Run("should_summarise_a_large_sequence_by_its_length", func(t *testing.T) {
		value := make([]any, 60)
		for i := range value {
			value[i] = strings.Repeat("x", 5)
		}

		assert.Equal(t, "[60 items]", docdiff.FormatValue(value))
	})
}
