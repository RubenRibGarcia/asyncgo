package validation

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// minimalDoc returns the smallest document the schema accepts.
func minimalDoc() *spec.AsyncAPI {
	return &spec.AsyncAPI{
		AsyncAPI: spec.Version,
		Info:     spec.Info{Title: "Orders", Version: "1.0.0"},
	}
}

// channelWithBindings returns a minimal document with one channel per name,
// each carrying the given channel bindings.
func channelWithBindings(names []string, bindings func() spec.ChannelBindings) *spec.AsyncAPI {
	doc := minimalDoc()
	doc.Channels = make(map[string]*spec.Channel, len(names))
	for _, name := range names {
		doc.Channels[name] = &spec.Channel{Address: name, Bindings: bindings()}
	}
	return doc
}

// violationsOf validates doc and returns the reported violations.
func violationsOf(t *testing.T, doc *spec.AsyncAPI, opts Options) Violations {
	t.Helper()

	err := Validate(doc, opts)
	require.Error(t, err)

	var violations Violations
	require.ErrorAs(t, err, &violations)
	require.NotEmpty(t, violations)
	return violations
}

// pointers returns the pointers of violations, in report order.
func pointers(violations Violations) []string {
	out := make([]string, len(violations))
	for i, violation := range violations {
		out[i] = violation.Pointer
	}
	return out
}

func TestValidate(t *testing.T) {
	t.Run("should_accept_a_minimal_document", func(t *testing.T) {
		require.NoError(t, Validate(minimalDoc(), Options{}))
	})

	t.Run("should_accept_a_reference_with_an_empty_target", func(t *testing.T) {
		// A Reference Object requires $ref, and the typed model always serializes
		// it (an empty Ref becomes "$ref": ""), so a "missing required property"
		// violation is not reachable from *spec.AsyncAPI. The schema accepts the
		// empty string: draft-07 does not assert the uri-reference format.
		doc := minimalDoc()
		doc.Channels = map[string]*spec.Channel{
			"orders": {Address: "orders", Servers: []*spec.Reference{{}}},
		}

		require.NoError(t, Validate(doc, Options{}))
	})

	t.Run("should_report_a_bogus_schema_type_under_a_qualified_key", func(t *testing.T) {
		// A hoisted schema is keyed "<package path>.<type name>", which the
		// published schema's key pattern rejects; relaxComponentKeys is what
		// keeps its body validated.
		doc := minimalDoc()
		doc.Components = &spec.Components{
			Schemas: map[string]*spec.Schema{
				"example.com/orders.OrderPlaced": {Type: "bogus"},
			},
		}

		violations := violationsOf(t, doc, Options{})

		require.Len(t, violations, 1)
		assert.Equal(
			t,
			"/components/schemas/example.com~1orders.OrderPlaced/type",
			violations[0].Pointer,
		)
	})

	t.Run("should_escape_json_pointer_tokens", func(t *testing.T) {
		// kafka is a known protocol, so its contents are validated: topic must be
		// a string. The channel keys exercise both RFC 6901 escapes.
		doc := channelWithBindings([]string{"orders/v1", "orders~v1"}, func() spec.ChannelBindings {
			return spec.ChannelBindings{"kafka": map[string]any{"topic": 5}}
		})

		violations := violationsOf(t, doc, Options{})

		assert.Equal(
			t,
			[]string{
				"/channels/orders~0v1/bindings/kafka/topic",
				"/channels/orders~1v1/bindings/kafka/topic",
			},
			pointers(violations),
		)
	})

	t.Run("should_omit_reference_branch_noise", func(t *testing.T) {
		// An Operations Object value is oneOf [Reference, Operation]. Both branches
		// fail here, and only the real cause — the empty action — is reported.
		doc := minimalDoc()
		doc.Operations = map[string]*spec.Operation{
			"orders.send": {Channel: &spec.Reference{Ref: "#/channels/orders"}},
		}

		violations := violationsOf(t, doc, Options{})

		require.Len(t, violations, 1)
		assert.Equal(t, "/operations/orders.send/action", violations[0].Pointer)
		assert.NotContains(t, violations[0].Message, "$ref")
	})

	t.Run("should_report_each_pointer_once", func(t *testing.T) {
		// The draft-07 metaschema reports a bad type twice: once for the enum of
		// allowed type names and once for the type of the keyword's value.
		doc := minimalDoc()
		doc.Components = &spec.Components{
			Schemas: map[string]*spec.Schema{"X": {Type: "bogus"}},
		}

		violations := violationsOf(t, doc, Options{})

		require.Len(t, violations, 1)
		assert.Equal(t, "/components/schemas/X/type", violations[0].Pointer)
	})

	t.Run("should_ignore_missing_binding_keys_only_when_asked", func(t *testing.T) {
		doc := channelWithBindings([]string{"orders"}, func() spec.ChannelBindings {
			return spec.ChannelBindings{"mercure": map[string]any{"bindingVersion": "0.1.0"}}
		})

		violations := violationsOf(t, doc, Options{})
		require.Len(t, violations, 1)
		assert.Equal(t, "/channels/orders/bindings", violations[0].Pointer)
		assert.Contains(t, violations[0].Message, "mercure")

		require.NoError(t, Validate(doc, Options{IgnoreMissingBindings: true}))
	})

	t.Run("should_not_ignore_other_binding_keys", func(t *testing.T) {
		// kafka is in the published schema, so its contents stay checked even
		// with IgnoreMissingBindings: topic must be a string.
		doc := channelWithBindings([]string{"orders"}, func() spec.ChannelBindings {
			return spec.ChannelBindings{"kafka": map[string]any{"topic": 5}}
		})

		violations := violationsOf(t, doc, Options{IgnoreMissingBindings: true})

		assert.Equal(t, []string{"/channels/orders/bindings/kafka/topic"}, pointers(violations))
	})

	t.Run("should_report_nothing_beyond_the_suppressed_bindings", func(t *testing.T) {
		// The suppression is scoped to the key, not the node: a second, real
		// violation elsewhere in the document still fails validation.
		doc := channelWithBindings([]string{"orders"}, func() spec.ChannelBindings {
			return spec.ChannelBindings{"mqtt5": map[string]any{"bindingVersion": "0.2.0"}}
		})
		doc.Operations = map[string]*spec.Operation{
			"orders.send": {Channel: &spec.Reference{Ref: "#/channels/orders"}},
		}

		violations := violationsOf(t, doc, Options{IgnoreMissingBindings: true})

		assert.Equal(t, []string{"/operations/orders.send/action"}, pointers(violations))
	})

	t.Run("should_return_violations_as_error", func(t *testing.T) {
		doc := channelWithBindings([]string{"orders"}, func() spec.ChannelBindings {
			return spec.ChannelBindings{"mercure": map[string]any{"bindingVersion": "0.1.0"}}
		})

		err := Validate(doc, Options{})
		require.Error(t, err)

		var violations Violations
		require.ErrorAs(t, err, &violations)

		lines := strings.Split(err.Error(), "\n")
		assert.Equal(t, "document is not valid AsyncAPI 3.1.0:", lines[0])
		assert.Len(t, lines, 2)
	})

	t.Run("should_reject_a_nil_document", func(t *testing.T) {
		require.EqualError(t, Validate(nil, Options{}), "validating: nil document")
	})
}

func TestSchemaPin(t *testing.T) {
	t.Run("should_embed_the_pinned_schema", func(t *testing.T) {
		assert.Equal(
			t,
			pinnedSHA256,
			fmt.Sprintf("%x", sha256.Sum256(schemaJSON)),
			"the embedded schema changed; update pinnedSHA256 and PROVENANCE.md",
		)
	})
}

func TestRelaxComponentKeys(t *testing.T) {
	// componentsDoc builds a schema-shaped document whose components definition
	// holds the given properties.
	componentsDoc := func(properties map[string]any) map[string]any {
		return map[string]any{
			"definitions": map[string]any{
				componentsURL: map[string]any{"properties": properties},
			},
		}
	}
	valueSchema := map[string]any{"$ref": "http://asyncapi.com/definitions/3.1.0/anySchema.json"}

	tests := []struct {
		name string
		doc  map[string]any
		want map[string]any // component name -> expected entry
	}{
		{
			name: "should_add_additional_properties_from_the_single_pattern",
			doc: componentsDoc(map[string]any{
				"schemas": map[string]any{
					"patternProperties": map[string]any{"^[\\w]+$": valueSchema},
				},
			}),
			want: map[string]any{
				"schemas": map[string]any{
					"patternProperties":    map[string]any{"^[\\w]+$": valueSchema},
					"additionalProperties": valueSchema,
				},
			},
		},
		{
			name: "should_keep_an_existing_additional_properties",
			doc: componentsDoc(map[string]any{
				"schemas": map[string]any{
					"patternProperties":    map[string]any{"^[\\w]+$": valueSchema},
					"additionalProperties": false,
				},
			}),
			want: map[string]any{
				"schemas": map[string]any{
					"patternProperties":    map[string]any{"^[\\w]+$": valueSchema},
					"additionalProperties": false,
				},
			},
		},
		{
			name: "should_skip_an_entry_with_several_patterns",
			doc: componentsDoc(map[string]any{
				"schemas": map[string]any{
					"patternProperties": map[string]any{"^a$": valueSchema, "^b$": valueSchema},
				},
			}),
			want: map[string]any{
				"schemas": map[string]any{
					"patternProperties": map[string]any{"^a$": valueSchema, "^b$": valueSchema},
				},
			},
		},
		{
			name: "should_skip_an_entry_without_patterns",
			doc:  componentsDoc(map[string]any{"schemas": map[string]any{"type": "object"}}),
			want: map[string]any{"schemas": map[string]any{"type": "object"}},
		},
		{
			name: "should_skip_a_non_object_entry",
			doc:  componentsDoc(map[string]any{"schemas": "unexpected"}),
			want: map[string]any{"schemas": "unexpected"},
		},
		{
			name: "should_tolerate_a_document_without_definitions",
			doc:  map[string]any{"$id": "x"},
		},
		{
			name: "should_tolerate_a_document_without_the_components_definition",
			doc:  map[string]any{"definitions": map[string]any{}},
		},
		{
			name: "should_tolerate_components_without_properties",
			doc: map[string]any{
				"definitions": map[string]any{componentsURL: map[string]any{}},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			relaxComponentKeys(tc.doc)

			definitions, _ := tc.doc["definitions"].(map[string]any)
			components, _ := definitions[componentsURL].(map[string]any)
			properties, _ := components["properties"].(map[string]any)
			for name, want := range tc.want {
				assert.Equal(t, want, properties[name], "component map %q", name)
			}
		})
	}

	t.Run("should_relax_every_component_map_in_the_embedded_schema", func(t *testing.T) {
		// Pins the assumption recorded in ADR 0007 and PROVENANCE.md: if an
		// upgrade changes how components maps are constrained, this count moves.
		root := make(map[string]any)
		require.NoError(t, json.Unmarshal(schemaJSON, &root))
		relaxComponentKeys(root)

		definitions, ok := root["definitions"].(map[string]any)
		require.True(t, ok)
		components, ok := definitions[componentsURL].(map[string]any)
		require.True(t, ok)
		properties, ok := components["properties"].(map[string]any)
		require.True(t, ok)

		relaxed := 0
		for name, raw := range properties {
			entry, ok := raw.(map[string]any)
			require.True(t, ok, "component map %q", name)
			if _, ok := entry["additionalProperties"]; ok {
				relaxed++
			}
		}
		assert.Equal(t, 19, relaxed)
	})
}

func TestPointer(t *testing.T) {
	t.Run("should_render_the_whole_document_as_slash", func(t *testing.T) {
		assert.Equal(t, "/", pointer(nil))
	})

	t.Run("should_join_tokens", func(t *testing.T) {
		assert.Equal(
			t,
			"/channels/orders/address",
			pointer([]string{"channels", "orders", "address"}),
		)
	})

	t.Run("should_escape_tokens", func(t *testing.T) {
		assert.Equal(
			t,
			"/channels/orders~1v1/orders~0v0",
			pointer([]string{"channels", "orders/v1", "orders~v0"}),
		)
	})
}

func TestIsOmittedBinding(t *testing.T) {
	additional := func(names ...string) *kind.AdditionalProperties {
		return &kind.AdditionalProperties{Properties: names}
	}
	tests := []struct {
		name string
		leaf leaf
		want bool
	}{
		{
			name: "should_accept_an_omitted_key_under_bindings",
			leaf: leaf{
				errorKind: additional("mqtt5"),
				violation: Violation{Pointer: "/servers/prod/bindings"},
			},
			want: true,
		},
		{
			name: "should_accept_only_omitted_keys",
			leaf: leaf{
				errorKind: additional("mqtt5", "mercure"),
				violation: Violation{Pointer: "/servers/prod/bindings"},
			},
			want: true,
		},
		{
			name: "should_reject_a_known_protocol",
			leaf: leaf{
				errorKind: additional("mqtt5", "kafka"),
				violation: Violation{Pointer: "/servers/prod/bindings"},
			},
		},
		{
			name: "should_reject_a_pointer_that_is_not_bindings",
			leaf: leaf{
				errorKind: additional("mqtt5"),
				violation: Violation{Pointer: "/servers/prod"},
			},
		},
		{
			name: "should_reject_a_different_keyword",
			leaf: leaf{
				errorKind: &kind.Type{Got: "number", Want: []string{"string"}},
				violation: Violation{Pointer: "/servers/prod/bindings"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isOmittedBinding(tc.leaf))
		})
	}
}

func TestReport(t *testing.T) {
	t.Run("should_drop_a_failed_reference_wrapper", func(t *testing.T) {
		_, ok := report(&jsonschema.ValidationError{
			ErrorKind:        &kind.Reference{Keyword: "$ref"},
			InstanceLocation: []string{"channels", "orders"},
		})
		assert.False(t, ok)
	})
}
