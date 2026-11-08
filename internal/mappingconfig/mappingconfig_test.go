package mappingconfig

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Run("should_parse_mappings_sorted_by_fqn", func(t *testing.T) {
		cfg, err := Parse([]byte(`
mappings:
  github.com/example/money.Money:
    type: string
    pattern: '^-?\d+\.\d{2} [A-Z]{3}$'
  github.com/gofrs/uuid.UUID:
    type: string
    format: uuid
`))
		require.NoError(t, err)
		require.Len(t, cfg.Mappings, 2)

		// Sorted, so the emitted harness is deterministic.
		assert.Equal(t, "github.com/example/money.Money", cfg.Mappings[0].FQN)
		assert.JSONEq(
			t,
			`{"type":"string","pattern":"^-?\\d+\\.\\d{2} [A-Z]{3}$"}`,
			cfg.Mappings[0].Schema,
		)
		assert.Equal(t, "github.com/gofrs/uuid.UUID", cfg.Mappings[1].FQN)
		assert.JSONEq(
			t,
			`{"type":"string","format":"uuid"}`,
			cfg.Mappings[1].Schema,
		)
	})

	t.Run("should_produce_a_schema_that_decodes", func(t *testing.T) {
		cfg, err := Parse([]byte(`
mappings:
  math/big.Int:
    type: integer
    minimum: 1
    enum: [1, 2]
    readOnly: true
`))
		require.NoError(t, err)
		require.Len(t, cfg.Mappings, 1)

		var s spec.Schema
		require.NoError(t, json.Unmarshal([]byte(cfg.Mappings[0].Schema), &s))
		assert.Equal(t, "integer", s.Type)
		require.NotNil(t, s.Minimum)
		assert.Equal(t, 1.0, *s.Minimum)
		// Any-valued keywords come back through encoding/json, so a number is a
		// float64 by the time the mapping reaches derivation.
		assert.Equal(t, []any{1.0, 2.0}, s.Enum)
		assert.True(t, s.ReadOnly)
	})

	t.Run("should_accept_every_whitelisted_keyword", func(t *testing.T) {
		// One line per allowed key: a keyword added to the whitelist without being
		// accepted here, or vice versa, fails this test rather than a user's build.
		cfg, err := Parse([]byte(`
mappings:
  example.com/a.T:
    type: string
    format: uuid
    description: a
    pattern: '^a$'
    enum: [a]
    const: a
    default: a
    examples: [a]
    contentEncoding: base64
    contentMediaType: text/plain
    minLength: 1
    maxLength: 2
    minimum: 1
    maximum: 2
    exclusiveMinimum: 1
    exclusiveMaximum: 2
    minItems: 1
    maxItems: 2
    readOnly: true
    writeOnly: true
    deprecated: true
`))
		require.NoError(t, err)
		require.Len(t, cfg.Mappings, 1)

		for _, key := range allowedKeys {
			assert.Contains(t, cfg.Mappings[0].Schema, `"`+key+`"`)
		}
	})

	t.Run("should_return_empty_config_for_an_empty_document", func(t *testing.T) {
		cfg, err := Parse([]byte("  \n"))
		require.NoError(t, err)
		assert.Empty(t, cfg.Mappings)
	})

	t.Run("should_return_empty_config_without_a_mappings_section", func(t *testing.T) {
		cfg, err := Parse([]byte("# nothing configured yet\n"))
		require.NoError(t, err)
		assert.Empty(t, cfg.Mappings)
	})

	t.Run("should_return_empty_config_for_an_empty_mappings_section", func(t *testing.T) {
		cfg, err := Parse([]byte("mappings: {}\n"))
		require.NoError(t, err)
		assert.Empty(t, cfg.Mappings)
	})
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "should_reject_malformed_yaml",
			in:   "mappings: [",
			want: "parsing config",
		},
		{
			name: "should_reject_unknown_section",
			in:   "mappings: {}\nchannels: {}\n",
			want: `unknown section "channels"; allowed: mappings`,
		},
		{
			name: "should_reject_unknown_keyword_with_allowed_list",
			in: `mappings:
  example.com/a.T:
    type: string
    propertys: {}
`,
			want: `unknown keyword "propertys"; allowed: const, contentEncoding`,
		},
		{
			name: "should_reject_semi_qualified_mapping_name",
			in: `mappings:
  a.T:
    type: string
`,
			want: `"a.T" is not a fully-qualified type name; use pkgPath.TypeName`,
		},
		{
			name: "should_reject_a_name_without_a_type",
			in: `mappings:
  example.com/a.:
    type: string
`,
			want: "is not a fully-qualified type name",
		},
		{
			name: "should_reject_a_name_whose_type_is_not_an_identifier",
			in: `mappings:
  example.com/a.T-U:
    type: string
`,
			want: `"T-U" is not a Go type name`,
		},
		{
			name: "should_reject_nested_object",
			in: `mappings:
  example.com/a.T:
    default:
      id: a
`,
			want: `"default" must be a scalar or a list, not a nested object`,
		},
		{
			name: "should_reject_a_nested_object_inside_a_list",
			in: `mappings:
  example.com/a.T:
    enum:
      - {type: string}
`,
			want: `"enum" must be a scalar or a list, not a nested object`,
		},
		{
			name: "should_reject_a_non_mapping_entry",
			in: `mappings:
  example.com/a.T: string
`,
			want: "want a mapping of schema keywords",
		},
		{
			name: "should_reject_a_value_that_is_not_a_schema",
			in: `mappings:
  example.com/a.T:
    minLength: lots
`,
			want: "cannot unmarshal",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := Parse([]byte(tc.in))

			require.Error(t, err)
			assert.Nil(t, cfg)
			assert.True(
				t,
				strings.Contains(err.Error(), tc.want),
				"want %q to contain %q", err.Error(), tc.want,
			)
		})
	}
}
