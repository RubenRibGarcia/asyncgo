package asyncgo

import (
	"reflect"
	"testing"

	"github.com/RubenRibGarcia/asyncgo/schema"
	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemasRegistersComponents(t *testing.T) {
	avro := spec.MultiFormat(
		"application/vnd.apache.avro;version=1.9.0",
		map[string]any{"type": "record", "name": "User"},
	)

	res := Spec(
		Info("Orders", "1.0.0"),
		Schemas(Schema("UserAvro", avro)),
		Channels(
			Channel("order-placed").Send(Operation().Message(
				MessageFrom("OrderPlaced", spec.Ref("#/components/schemas/UserAvro")),
			)),
			Channel("order-shipped").Send(Operation().Message(
				MessageFrom("OrderShipped", spec.Ref("#/components/schemas/UserAvro")),
			)),
		),
	)

	require.NoError(t, res.Err)
	require.NotNil(t, res.Doc.Components)
	require.Contains(t, res.Doc.Components.Schemas, "UserAvro")
	assert.Same(t, avro, res.Doc.Components.Schemas["UserAvro"])

	placed := res.Doc.Components.Messages["OrderPlaced"]
	shipped := res.Doc.Components.Messages["OrderShipped"]
	require.NotNil(t, placed.Payload)
	require.NotNil(t, shipped.Payload)
	assert.Equal(t, "#/components/schemas/UserAvro", placed.Payload.Ref)
	assert.Equal(t, "#/components/schemas/UserAvro", shipped.Payload.Ref)

	// Each channel carries a $ref to its hoisted message instead of the object.
	assert.Equal(
		t,
		"#/components/messages/OrderPlaced",
		res.Doc.Channels["order-placed"].Messages["OrderPlaced"].Ref,
	)
	assert.Equal(
		t,
		"#/components/messages/OrderShipped",
		res.Doc.Channels["order-shipped"].Messages["OrderShipped"].Ref,
	)
}

func TestSchemaValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		spec func() *SpecResult
		want string
	}{
		{
			name: "should_return_error_when_schema_name_is_empty",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Schemas(Schema("", &spec.Schema{Type: "object"})),
				)
			},
			want: "schema.name: is required",
		},
		{
			name: "should_return_error_on_duplicate_schema_name",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Schemas(
						Schema("User", &spec.Schema{Type: "object"}),
						Schema("User", &spec.Schema{Type: "string"}),
					),
				)
			},
			want: "schema.User: duplicate name",
		},
		{
			name: "should_reject_schema_format_without_schema",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Schemas(Schema("Avro", &spec.Schema{
						SchemaFormat: "application/vnd.apache.avro;version=1.9.0",
					})),
				)
			},
			want: "schema.Avro.schema: is required alongside schemaFormat",
		},
		{
			name: "should_reject_schema_without_schema_format",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Schemas(Schema("Avro", &spec.Schema{
						Schema: map[string]any{"type": "record"},
					})),
				)
			},
			want: "schema.Avro.schemaFormat: is required alongside schema",
		},
		{
			name: "should_reject_schema_format_beside_json_schema_keywords",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Schemas(Schema("Avro", &spec.Schema{
						SchemaFormat: "application/vnd.apache.avro;version=1.9.0",
						Schema:       map[string]any{"type": "record"},
						Type:         "object",
					})),
				)
			},
			want: "schema.Avro: multi format schema must not carry JSON Schema keyword(s): type",
		},
		{
			// The content keywords are Draft-07, so a Multi Format body carrying one
			// is rejected like any other JSON Schema keyword. jsonSchemaKeywords
			// reflects over the struct and picks them up without an edit; this case
			// pins that it still does.
			name: "should_reject_content_encoding_on_a_multi_format_schema",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Schemas(Schema("Avro", &spec.Schema{
						SchemaFormat:    "application/vnd.apache.avro;version=1.9.0",
						Schema:          map[string]any{"type": "record"},
						ContentEncoding: "base64",
					})),
				)
			},
			want: "schema.Avro: multi format schema must not carry JSON Schema keyword(s): contentEncoding",
		},
		{
			name: "should_reject_declared_name_colliding_with_hoisted_schema",
			spec: func() *SpecResult {
				fqn := schema.Name(reflect.TypeFor[OrderPlaced]())
				return Spec(
					Info("Orders", "1.0.0"),
					Schemas(Schema(fqn, &spec.Schema{Type: "object"})),
					Channels(
						Channel("order-placed").Send(Operation().Message(
							MessageOf(OrderPlaced{}).Name("OrderPlaced"),
						)),
					),
				)
			},
			want: "schema." + schema.Name(reflect.TypeFor[OrderPlaced]()) +
				": name must match ^[\\w\\d.\\-_]+$" +
				"\nschema." + schema.Name(reflect.TypeFor[OrderPlaced]()) +
				": collides with an auto-hoisted schema of the same name",
		},
		{
			name: "should_reject_multi_format_node_nested_in_if",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Schemas(Schema("Avro", &spec.Schema{
						Type: "object",
						If: &spec.Schema{
							SchemaFormat: "application/vnd.apache.avro;version=1.9.0",
						},
					})),
				)
			},
			want: "schema.Avro.if.schema: is required alongside schemaFormat",
		},
		{
			name: "should_reject_multi_format_node_nested_in_pattern_properties",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Schemas(Schema("Avro", &spec.Schema{
						Type: "object",
						PatternProperties: map[string]*spec.Schema{
							"^x-": {SchemaFormat: "application/vnd.apache.avro;version=1.9.0"},
						},
					})),
				)
			},
			want: "schema.Avro.patternProperties.^x-.schema: is required alongside schemaFormat",
		},
		{
			name: "should_descend_into_dependencies_schema",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Schemas(Schema("Avro", &spec.Schema{
						Type: "object",
						Dependencies: map[string]*spec.Schema{
							"creditCard": {
								SchemaFormat: "application/vnd.apache.avro;version=1.9.0",
							},
						},
					})),
				)
			},
			want: "schema.Avro.dependencies.creditCard.schema: is required alongside schemaFormat",
		},
		{
			name: "should_reject_json_schema_keyword_nested_in_then",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Schemas(Schema("Avro", &spec.Schema{
						Type: "object",
						Then: &spec.Schema{
							SchemaFormat: "application/vnd.apache.avro;version=1.9.0",
							Schema:       map[string]any{"type": "record"},
							Type:         "object",
						},
					})),
				)
			},
			want: "schema.Avro.then: multi format schema must not carry JSON Schema keyword(s): type",
		},
		{
			name: "should_reject_a_schema_name_outside_the_key_pattern",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Schemas(Schema("Order Placed", &spec.Schema{Type: "object"})),
				)
			},
			want: "schema.Order Placed: name must match ^[\\w\\d.\\-_]+$",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := tc.spec()
			if tc.want == "" {
				require.NoError(t, res.Err)
				assert.Nil(t, res.ValidationErrors())
				return
			}
			require.Error(t, res.Err)
			assert.Equal(t, tc.want, res.Err.Error())
		})
	}
}
