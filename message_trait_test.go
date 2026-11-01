package asyncgo

import (
	"testing"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageTraitFields(t *testing.T) {
	corr := CorrelationID("CorrelationID", spec.CorrelationID{
		Description: "Correlation ID",
		Location:    "$message.header#/correlationId",
	})

	tr := MessageTrait("Traced").
		ContentType("application/json").
		Name("TracedMessage").
		Title("Traced").
		Summary("Shared headers").
		Description("Applied to every traced message").
		Headers(spec.Ref("#/components/schemas/Headers")).
		CorrelationID(corr).
		Tags(spec.Tag{Name: "traced"}).
		ExternalDocs(spec.ExternalDocs{URL: "https://example.com/traced"}).
		Example("simple", map[string]any{"order_id": "1"}).
		Kafka(spec.KafkaMessageBinding{Key: &spec.Schema{Type: "string"}})

	assert.Equal(t, "Traced", tr.name)
	assert.Equal(t, "application/json", tr.t.ContentType)
	assert.Equal(t, "TracedMessage", tr.t.Name)
	assert.Equal(t, "Traced", tr.t.Title)
	assert.Equal(t, "Shared headers", tr.t.Summary)
	assert.Equal(t, "Applied to every traced message", tr.t.Description)
	require.NotNil(t, tr.t.Headers)
	assert.Equal(t, "#/components/schemas/Headers", tr.t.Headers.Ref)
	require.NotNil(t, tr.t.CorrelationID)
	assert.Equal(t, "#/components/correlationIds/CorrelationID", tr.t.CorrelationID.Ref)
	assert.Equal(t, []spec.Tag{{Name: "traced"}}, tr.t.Tags)
	require.NotNil(t, tr.t.ExternalDocs)
	assert.Equal(t, "https://example.com/traced", tr.t.ExternalDocs.URL)
	require.Len(t, tr.t.Examples, 1)
	assert.Equal(t, "simple", tr.t.Examples[0].Name)
	require.Contains(t, tr.t.Bindings, spec.ProtocolKafka)
}

// TestMessageTraitBindingsReplaces is the message-trait counterpart of
// TestOperationTraitBindingsReplaces.
func TestMessageTraitBindingsReplaces(t *testing.T) {
	tr := MessageTrait("Traced").
		Bindings(spec.MessageBindings{
			spec.ProtocolAMQP: &spec.AMQPMessageBinding{ContentEncoding: "gzip"},
		})

	require.Len(t, tr.t.Bindings, 1)
	assert.Equal(
		t,
		"gzip",
		tr.t.Bindings[spec.ProtocolAMQP].(*spec.AMQPMessageBinding).ContentEncoding,
	)

	tr.Bindings(spec.MessageBindings{
		spec.ProtocolMQTT: &spec.MQTTMessageBinding{BindingVersion: "0.2.0"},
	})

	require.Len(t, tr.t.Bindings, 1)
	assert.NotContains(t, tr.t.Bindings, spec.ProtocolAMQP)
	assert.Equal(
		t,
		"0.2.0",
		tr.t.Bindings[spec.ProtocolMQTT].(*spec.MQTTMessageBinding).BindingVersion,
	)
}

func TestMessageTraitsRegistersComponents(t *testing.T) {
	res := Spec(
		Info("Orders", "1.0.0"),
		MessageTraits(
			MessageTrait("Traced").ContentType("application/json"),
			MessageTrait("Untraced").ContentType("text/plain"),
		),
	)

	require.NoError(t, res.Err)
	require.NotNil(t, res.Doc.Components)
	assert.Len(t, res.Doc.Components.MessageTraits, 2)
	assert.Equal(t, "application/json", res.Doc.Components.MessageTraits["Traced"].ContentType)
}

func TestMessageTraitCorrelationIDFrom(t *testing.T) {
	tr := MessageTrait("Traced").CorrelationIDFrom(spec.CorrelationID{
		Description: "Correlation ID",
		Location:    "$message.header#/correlationId",
	})

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	require.NoError(t, MessageTraits(tr).apply(b))

	require.Contains(t, b.doc.Components.CorrelationIDs, "TracedCorrelationID")
	assert.Equal(
		t,
		"$message.header#/correlationId",
		b.doc.Components.CorrelationIDs["TracedCorrelationID"].Location,
	)

	require.NotNil(t, b.doc.Components.MessageTraits["Traced"].CorrelationID)
	assert.Equal(
		t,
		"#/components/correlationIds/TracedCorrelationID",
		b.doc.Components.MessageTraits["Traced"].CorrelationID.Ref,
	)
}

func TestMessageTraits(t *testing.T) {
	trait := MessageTrait("Traced")

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	msg, err := MessageOf(OrderPlaced{}).Name("OrderPlaced").Traits(trait).build(b)
	require.NoError(t, err)

	require.Len(t, msg.Traits, 1)
	assert.Equal(t, "#/components/messageTraits/Traced", msg.Traits[0].Ref)
}

func TestMessageTraitValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		spec func() *SpecResult
		want string
	}{
		{
			name: "should_return_error_when_message_trait_references_unknown_correlation_id",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					MessageTraits(
						MessageTrait("Traced").CorrelationID(
							CorrelationID("Missing", spec.CorrelationID{
								Location: "$message.header#/correlationId",
							}),
						),
					),
				)
			},
			want: `messageTrait.Traced: references unknown correlation id "Missing"`,
		},
		{
			name: "should_return_error_when_message_trait_name_is_empty",
			spec: func() *SpecResult {
				return Spec(Info("Orders", "1.0.0"), MessageTraits(MessageTrait("")))
			},
			want: "messageTrait.name: is required",
		},
		{
			name: "should_return_error_when_message_trait_name_is_duplicate",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					MessageTraits(MessageTrait("Traced"), MessageTrait("Traced")),
				)
			},
			want: "messageTrait.Traced: duplicate name",
		},
		{
			name: "should_return_error_when_message_trait_sets_both_correlation_forms",
			spec: func() *SpecResult {
				corr := CorrelationID("CorrelationID", spec.CorrelationID{
					Location: "$message.header#/declared",
				})
				return Spec(
					Info("Orders", "1.0.0"),
					MessageTraits(
						MessageTrait("Traced").
							CorrelationID(corr).
							CorrelationIDFrom(spec.CorrelationID{
								Location: "$message.header#/inline",
							}),
					),
					CorrelationIDs(corr),
				)
			},
			want: "messageTrait.Traced: correlation id reference and inline correlation id are mutually exclusive",
		},
		{
			name: "should_reject_a_message_trait_name_outside_the_key_pattern",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					MessageTraits(MessageTrait("tenant trait")),
				)
			},
			want: "messageTrait.tenant trait: name must match ^[\\w\\d.\\-_]+$",
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
