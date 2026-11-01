package asyncgo

import (
	"testing"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOperationTraitFields(t *testing.T) {
	basic := SecurityScheme("basic", spec.SecurityScheme{Type: "http", Scheme: "basic"})

	tr := OperationTrait("KafkaOrders").
		Title("Kafka orders").
		Summary("Shared settings").
		Description("Applied to every orders operation").
		Security(basic).
		Tags(spec.Tag{Name: "orders"}).
		ExternalDocs(spec.ExternalDocs{URL: "https://example.com/orders"}).
		Kafka(spec.KafkaOperationBinding{
			GroupID:  &spec.Schema{Type: "string"},
			ClientID: &spec.Schema{Type: "string"},
		})

	assert.Equal(t, "KafkaOrders", tr.name)
	assert.Equal(t, "Kafka orders", tr.t.Title)
	assert.Equal(t, "Shared settings", tr.t.Summary)
	assert.Equal(t, "Applied to every orders operation", tr.t.Description)
	require.Len(t, tr.t.Security, 1)
	assert.Equal(t, "#/components/securitySchemes/basic", tr.t.Security[0].Ref)
	assert.Equal(t, []spec.Tag{{Name: "orders"}}, tr.t.Tags)
	require.NotNil(t, tr.t.ExternalDocs)
	assert.Equal(t, "https://example.com/orders", tr.t.ExternalDocs.URL)
	require.Contains(t, tr.t.Bindings, spec.ProtocolKafka)
	assert.Equal(
		t,
		&spec.Schema{Type: "string"},
		tr.t.Bindings[spec.ProtocolKafka].(*spec.KafkaOperationBinding).GroupID,
	)
}

// TestOperationTraitBindingsReplaces pins the documented whole-map semantics:
// Bindings replaces the trait's protocol bindings, it does not merge them.
func TestOperationTraitBindingsReplaces(t *testing.T) {
	tr := OperationTrait("KafkaOrders").
		Bindings(spec.OperationBindings{
			spec.ProtocolAMQP: &spec.AMQPOperationBinding{Expiration: 30},
		})

	require.Len(t, tr.t.Bindings, 1)
	assert.Equal(
		t,
		30,
		tr.t.Bindings[spec.ProtocolAMQP].(*spec.AMQPOperationBinding).Expiration,
	)

	tr.Bindings(spec.OperationBindings{
		spec.ProtocolNATS: &spec.NATSOperationBinding{Queue: "q"},
	})

	require.Len(t, tr.t.Bindings, 1)
	assert.NotContains(t, tr.t.Bindings, spec.ProtocolAMQP)
	assert.Equal(t, "q", tr.t.Bindings[spec.ProtocolNATS].(*spec.NATSOperationBinding).Queue)
}

func TestOperationTraitsRegistersComponents(t *testing.T) {
	res := Spec(
		Info("Orders", "1.0.0"),
		OperationTraits(
			OperationTrait("Kafka").Summary("kafka settings"),
			OperationTrait("AMQP").Summary("amqp settings"),
		),
	)

	require.NoError(t, res.Err)
	require.NotNil(t, res.Doc.Components)
	assert.Len(t, res.Doc.Components.OperationTraits, 2)
	assert.Equal(t, "kafka settings", res.Doc.Components.OperationTraits["Kafka"].Summary)
}

func TestOperationTraits(t *testing.T) {
	trait := OperationTrait("Kafka")

	c := Channel("order-placed").
		Send(Operation().
			Traits(trait).
			Message(MessageOf(OrderPlaced{}).Name("OrderPlaced")))

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	require.NoError(t, c.apply(b))

	require.Contains(t, b.doc.Operations, "order-placed.send")
	require.Len(t, b.doc.Operations["order-placed.send"].Traits, 1)
	assert.Equal(
		t,
		"#/components/operationTraits/Kafka",
		b.doc.Operations["order-placed.send"].Traits[0].Ref,
	)
}

// TestSharedOperationTraitOnTwoOperations is the unit-level counterpart of the
// fixture's headline case: one component, referenced by two operations.
func TestSharedOperationTraitOnTwoOperations(t *testing.T) {
	trait := OperationTrait("Kafka")

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	require.NoError(t, Channels(
		Channel("order-placed").Send(Operation().
			Traits(trait).
			Message(MessageOf(OrderPlaced{}).Name("OrderPlaced"))),
		Channel("order-shipped").Send(Operation().
			Traits(trait).
			Message(MessageOf(OrderPlaced{}).Name("OrderShipped"))),
	).apply(b))

	assert.Equal(
		t,
		"#/components/operationTraits/Kafka",
		b.doc.Operations["order-placed.send"].Traits[0].Ref,
	)
	assert.Equal(
		t,
		"#/components/operationTraits/Kafka",
		b.doc.Operations["order-shipped.send"].Traits[0].Ref,
	)
}

func TestOperationTraitValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		spec func() *SpecResult
		want string
	}{
		{
			name: "should_return_error_when_operation_trait_references_unknown_security_scheme",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					OperationTraits(
						OperationTrait("Kafka").
							Security(SecurityScheme("basic", spec.SecurityScheme{Type: "http"})),
					),
				)
			},
			want: `operationTrait.Kafka: references unknown security scheme "basic"`,
		},
		{
			name: "should_return_error_when_operation_trait_name_is_empty",
			spec: func() *SpecResult {
				return Spec(Info("Orders", "1.0.0"), OperationTraits(OperationTrait("")))
			},
			want: "operationTrait.name: is required",
		},
		{
			name: "should_return_error_when_operation_trait_name_is_duplicate",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					OperationTraits(OperationTrait("Kafka"), OperationTrait("Kafka")),
				)
			},
			want: "operationTrait.Kafka: duplicate name",
		},
		{
			name: "should_reject_an_operation_trait_name_outside_the_key_pattern",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					OperationTraits(OperationTrait("tenant trait")),
				)
			},
			want: "operationTrait.tenant trait: name must match ^[\\w\\d.\\-_]+$",
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
