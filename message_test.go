package asyncgo

import (
	"reflect"
	"testing"

	"github.com/RubenRibGarcia/asyncgo/internal/jsonpointer"
	"github.com/RubenRibGarcia/asyncgo/schema"
	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageFields(t *testing.T) {
	headers := &spec.Schema{Type: "object"}
	m := MessageOf(OrderPlaced{}).
		Name("n").
		Title("t").
		Summary("s").
		Description("d").
		ContentType("application/json").
		Headers(headers).
		Example("first", map[string]any{"order_id": "1"}).
		Example("second", map[string]any{"order_id": "2"})

	assert.Equal(t, "n", m.name)
	assert.Equal(t, "t", m.title)
	assert.Equal(t, "s", m.summary)
	assert.Equal(t, "d", m.description)
	assert.Equal(t, "application/json", m.contentType)
	assert.Same(t, headers, m.headers)
	require.Len(t, m.examples, 2)
	assert.Equal(t, "first", m.examples[0].Name)
	assert.Equal(t, "second", m.examples[1].Name)
}

func TestMessageName(t *testing.T) {
	t.Run("should_return_explicit_name", func(t *testing.T) {
		assert.Equal(t, "Explicit", messageName(MessageOf(OrderPlaced{}).Name("Explicit")))
	})

	t.Run("should_dereference_pointer_type", func(t *testing.T) {
		assert.Equal(t, "OrderPlaced", messageName(MessageOf(&OrderPlaced{})))
	})

	t.Run("should_fallback_to_message_for_anonymous_type", func(t *testing.T) {
		assert.Equal(t, "message", messageName(MessageOf(struct{ X int }{})))
	})

	t.Run("should_fallback_to_message_for_nil_type", func(t *testing.T) {
		assert.Equal(t, "message", messageName(MessageOf(nil)))
	})
}

func TestMessageKey(t *testing.T) {
	t.Run("should_key_authored_message_by_its_name", func(t *testing.T) {
		assert.Equal(t, "UserSignedUp", messageKey(MessageFrom("UserSignedUp", &spec.Schema{})))
	})

	t.Run("should_key_explicitly_named_message_by_its_name", func(t *testing.T) {
		assert.Equal(t, "Explicit", messageKey(MessageOf(OrderPlaced{}).Name("Explicit")))
	})

	t.Run("should_key_type_derived_message_by_fully_qualified_type_name", func(t *testing.T) {
		assert.Equal(
			t,
			schema.Name(reflect.TypeFor[OrderPlaced]()),
			messageKey(MessageOf(OrderPlaced{})),
		)
	})

	t.Run("should_dereference_pointer_type", func(t *testing.T) {
		assert.Equal(
			t,
			schema.Name(reflect.TypeFor[OrderPlaced]()),
			messageKey(MessageOf(&OrderPlaced{})),
		)
	})

	t.Run("should_fallback_to_message_for_anonymous_type", func(t *testing.T) {
		assert.Equal(t, "message", messageKey(MessageOf(struct{ X int }{})))
	})

	t.Run("should_fallback_to_message_for_nil_type", func(t *testing.T) {
		assert.Equal(t, "message", messageKey(MessageOf(nil)))
	})
}

func TestMessageBuildNilType(t *testing.T) {
	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	sm, err := MessageOf(nil).build(b)
	assert.Nil(t, sm)
	assert.EqualError(t, err, "message: nil payload type or schema")
}

func TestMessageFrom(t *testing.T) {
	t.Run("should_emit_message_from_payload_verbatim", func(t *testing.T) {
		payload := spec.MultiFormat(
			"application/vnd.apache.avro;version=1.9.0",
			map[string]any{"type": "record", "name": "User"},
		)
		b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}

		sm, err := MessageFrom("UserSignedUp", payload).build(b)

		require.NoError(t, err)
		require.NotNil(t, sm)
		assert.Equal(t, "UserSignedUp", sm.Name)
		assert.Same(t, payload, sm.Payload)
	})

	t.Run("should_not_hoist_the_hand_authored_payload", func(t *testing.T) {
		b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}

		_, err := MessageFrom("UserSignedUp", &spec.Schema{Type: "object"}).build(b)

		require.NoError(t, err)
		assert.Empty(t, b.defs)
	})

	t.Run("should_emit_multi_format_headers", func(t *testing.T) {
		headers := spec.MultiFormat(
			"application/vnd.apache.avro;version=1.9.0",
			map[string]any{"type": "record", "name": "Headers"},
		)
		b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}

		sm, err := MessageFrom("UserSignedUp", spec.Ref("#/components/schemas/UserAvro")).
			Headers(headers).
			build(b)

		require.NoError(t, err)
		assert.Same(t, headers, sm.Headers)
		require.NotNil(t, sm.Payload)
		assert.Equal(t, "#/components/schemas/UserAvro", sm.Payload.Ref)
	})

	t.Run("should_require_message_from_name", func(t *testing.T) {
		b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}

		sm, err := MessageFrom("", &spec.Schema{Type: "object"}).build(b)

		assert.Nil(t, sm)
		assert.EqualError(t, err, "message: name is required for a hand-authored payload")
	})

	t.Run("should_return_error_on_nil_message_from_payload", func(t *testing.T) {
		b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}

		sm, err := MessageFrom("UserSignedUp", nil).build(b)

		assert.Nil(t, sm)
		assert.EqualError(t, err, "message: nil payload type or schema")
	})

	t.Run("should_return_error_when_payload_type_and_schema_are_both_set", func(t *testing.T) {
		b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
		m := &message{
			typ:     reflect.TypeFor[OrderPlaced](),
			payload: &spec.Schema{Type: "object"},
		}

		sm, err := m.build(b)

		assert.Nil(t, sm)
		assert.EqualError(t, err,
			"message: payload schema and payload type are mutually exclusive")
	})
}

// TestPayloadExamples covers the schema-level examples builder. MessageOf always
// yields a $ref payload, and JSON Schema does not apply keywords beside a $ref,
// so the interesting assertion in every case is where the examples land — and
// that they never land on the ref node itself.
func TestPayloadExamples(t *testing.T) {
	t.Run("should_put_examples_on_the_hoisted_component", func(t *testing.T) {
		res := Spec(
			Info("Orders", "1.0.0"),
			Channels(Channel("order-placed").Send(Operation().Message(
				MessageOf(OrderPlaced{}).Name("OrderPlaced").
					PayloadExamples(map[string]any{"order_id": "1"}),
			))),
		)
		require.NoError(t, res.Err)

		key := schema.Name(reflect.TypeFor[OrderPlaced]())
		require.NotNil(t, res.Doc.Components)
		target := res.Doc.Components.Schemas[key]
		require.NotNil(t, target, "the payload type must still be hoisted")
		assert.Equal(t, []any{map[string]any{"order_id": "1"}}, target.Examples)

		// The ref node the message points at must stay a bare $ref.
		require.Contains(t, res.Doc.Components.Messages, "OrderPlaced")
		payload := res.Doc.Components.Messages["OrderPlaced"].Payload
		require.NotNil(t, payload)
		assert.NotEmpty(t, payload.Ref, "the payload stays a $ref")
		assert.Empty(t, payload.Examples, "examples must never be a sibling of $ref")
	})

	t.Run("should_put_examples_on_inline_payload", func(t *testing.T) {
		res := Spec(
			Info("Orders", "1.0.0"),
			Channels(Channel("order-placed").Send(Operation().Message(
				MessageFrom("Inline", &spec.Schema{Type: "object"}).
					PayloadExamples("first", "second"),
			))),
		)
		require.NoError(t, res.Err)

		payload := res.Doc.Components.Messages["Inline"].Payload
		require.NotNil(t, payload)
		assert.Empty(t, payload.Ref)
		assert.Equal(t, []any{"first", "second"}, payload.Examples)
	})

	t.Run("should_reject_examples_on_a_multi_format_payload", func(t *testing.T) {
		res := Spec(
			Info("Orders", "1.0.0"),
			Channels(Channel("order-placed").Send(Operation().Message(
				MessageFrom("Avro", spec.MultiFormat(
					"application/vnd.apache.avro;version=1.9.0",
					map[string]any{"type": "record"},
				)).PayloadExamples("a"),
			))),
		)
		require.Error(t, res.Err)
		assert.Contains(t, res.Err.Error(),
			"message.Avro: examples: the payload is a multi format schema")
	})

	t.Run("should_accumulate_across_calls", func(t *testing.T) {
		m := MessageFrom("Inline", &spec.Schema{Type: "object"}).
			PayloadExamples("a").
			PayloadExamples("b", "c")
		assert.Equal(t, []any{"a", "b", "c"}, m.payloadExamples)
	})

	t.Run("should_resolve_a_component_declared_after_the_message", func(t *testing.T) {
		// Item order is the caller's, so resolution is deferred to a post-pass.
		// Declaring the component last is the case a build-time lookup would miss.
		res := Spec(
			Info("Orders", "1.0.0"),
			Channels(Channel("order-placed").Send(Operation().Message(
				MessageFrom("Authored", spec.Ref("#/components/schemas/UserAvro")).
					PayloadExamples("a"),
			))),
			Schemas(Schema("UserAvro", &spec.Schema{Type: "object"})),
		)
		require.NoError(t, res.Err)
		assert.Equal(t, []any{"a"}, res.Doc.Components.Schemas["UserAvro"].Examples)
	})

	t.Run("should_error_when_payload_ref_is_unresolvable", func(t *testing.T) {
		res := Spec(
			Info("Orders", "1.0.0"),
			Channels(Channel("order-placed").Send(Operation().Message(
				MessageFrom("Authored", spec.Ref("#/components/schemas/Nope")).
					PayloadExamples("a"),
			))),
		)
		require.Error(t, res.Err)
		assert.Contains(t, res.Err.Error(), "message.Authored: examples:")
		assert.Contains(t, res.Err.Error(), "#/components/schemas/Nope")
	})

	t.Run("should_reject_a_ref_that_is_not_a_schema", func(t *testing.T) {
		res := Spec(
			Info("Orders", "1.0.0"),
			Channels(Channel("order-placed").Send(Operation().Message(
				MessageFrom("Authored", spec.Ref("#/components/messages/Other")).
					PayloadExamples("a"),
			))),
		)
		require.Error(t, res.Err)
		assert.Contains(t, res.Err.Error(), "is not a components.schemas reference")
	})

	t.Run("should_accept_identical_examples_from_two_messages", func(t *testing.T) {
		res := Spec(
			Info("Orders", "1.0.0"),
			Channels(
				Channel("a").Send(Operation().Message(
					MessageOf(OrderPlaced{}).Name("A").PayloadExamples("same"),
				)),
				Channel("b").Send(Operation().Message(
					MessageOf(OrderPlaced{}).Name("B").PayloadExamples("same"),
				)),
			),
		)
		require.NoError(t, res.Err)

		target := res.Doc.Components.Schemas[schema.Name(reflect.TypeFor[OrderPlaced]())]
		require.NotNil(t, target)
		assert.Equal(t, []any{"same"}, target.Examples, "an identical repeat is idempotent")
	})

	t.Run(
		"should_error_when_two_messages_disagree_on_shared_component_examples",
		func(t *testing.T) {
			res := Spec(
				Info("Orders", "1.0.0"),
				Channels(
					Channel("a").Send(Operation().Message(
						MessageOf(OrderPlaced{}).Name("A").PayloadExamples("one"),
					)),
					Channel("b").Send(Operation().Message(
						MessageOf(OrderPlaced{}).Name("B").PayloadExamples("two"),
					)),
				),
			)
			require.Error(t, res.Err)
			assert.Contains(t, res.Err.Error(),
				"conflicts with the examples already declared on")
		},
	)

	t.Run("should_leave_a_payload_alone_when_no_examples_were_declared", func(t *testing.T) {
		res := Spec(
			Info("Orders", "1.0.0"),
			Channels(Channel("order-placed").Send(Operation().Message(
				MessageOf(OrderPlaced{}).Name("OrderPlaced"),
			))),
		)
		require.NoError(t, res.Err)

		target := res.Doc.Components.Schemas[schema.Name(reflect.TypeFor[OrderPlaced]())]
		require.NotNil(t, target)
		assert.Empty(t, target.Examples)
	})
}

// TestMessagesAreHoisted covers the document shape the generator emits:
// components.messages owns the Message Object, the channel's messages map
// carries a $ref to it, and the operation keeps pointing at the message as
// defined in the channel — the specification forbids an operation from
// referencing components.messages directly.
func TestMessagesAreHoisted(t *testing.T) {
	res := Spec(
		Info("Orders", "1.0.0"),
		Channels(
			Channel("order-placed").Send(Operation().Message(MessageOf(OrderPlaced{}))),
		),
	)

	require.NoError(t, res.Err)
	require.NotNil(t, res.Doc.Components)

	key := schema.Name(reflect.TypeFor[OrderPlaced]())
	require.Contains(t, res.Doc.Components.Messages, key)
	hoisted := res.Doc.Components.Messages[key]
	assert.Equal(t, "OrderPlaced", hoisted.Name)
	require.NotNil(t, hoisted.Payload)
	assert.Equal(t, schema.Ref(reflect.TypeFor[OrderPlaced]()), hoisted.Payload.Ref)

	entry := res.Doc.Channels["order-placed"].Messages[key]
	require.NotNil(t, entry)
	assert.Equal(t, "#/components/messages/"+jsonpointer.Escape(key), entry.Ref)

	require.Len(t, res.Doc.Operations["order-placed.send"].Messages, 1)
	assert.Equal(
		t,
		"#/channels/order-placed/messages/"+jsonpointer.Escape(key),
		res.Doc.Operations["order-placed.send"].Messages[0].Ref,
	)
}

// TestMessagesAreSharedAcrossChannels pins the reuse half of the component-key
// rule: the same message carried by two channels is hoisted once and referenced
// twice.
func TestMessagesAreSharedAcrossChannels(t *testing.T) {
	msg := MessageOf(OrderPlaced{})

	res := Spec(
		Info("Orders", "1.0.0"),
		Channels(
			Channel("order-placed").Send(Operation().Message(msg)),
			Channel("order-shipped").Receive(Operation().Message(msg)),
		),
	)

	require.NoError(t, res.Err)
	key := schema.Name(reflect.TypeFor[OrderPlaced]())
	assert.Len(t, res.Doc.Components.Messages, 1)
	require.Contains(t, res.Doc.Channels["order-shipped"].Messages, key)
	assert.Equal(
		t,
		"#/components/messages/"+jsonpointer.Escape(key),
		res.Doc.Channels["order-shipped"].Messages[key].Ref,
	)
}

// TestMessagesDuplicateKeyRejected pins the other half: two different messages
// under one key are a duplicate, not a silent overwrite, and the error names the
// channel that declared the second one.
func TestMessagesDuplicateKeyRejected(t *testing.T) {
	res := Spec(
		Info("Orders", "1.0.0"),
		Channels(
			Channel("order-placed").Send(Operation().Message(
				MessageOf(OrderPlaced{}).Name("OrderEvent"),
			)),
			Channel("order-shipped").Send(Operation().Message(
				MessageOf(OrderPlaced{}).Name("OrderEvent").Description("Shipped"),
			)),
		),
	)

	require.Error(t, res.Err)
	assert.EqualError(
		t,
		res.Err,
		"message.OrderEvent: duplicate name with different content (channel.order-shipped)",
	)
}

func TestMessageCorrelationID(t *testing.T) {
	corr := CorrelationID("CorrelationID", spec.CorrelationID{
		Location: "$message.header#/correlationId",
	})

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	require.NoError(t, CorrelationIDs(corr).apply(b))

	msg, err := MessageOf(OrderPlaced{}).
		Name("OrderPlaced").
		CorrelationID(corr).
		build(b)
	require.NoError(t, err)

	require.NotNil(t, msg.CorrelationID)
	assert.Equal(t, "#/components/correlationIds/CorrelationID", msg.CorrelationID.Ref)
}

func TestMessageCorrelationIDFrom(t *testing.T) {
	tests := []struct {
		name string
		msg  *message
		key  string
	}{
		{
			name: "should_derive_the_key_from_the_explicit_name",
			msg: MessageOf(OrderPlaced{}).
				Name("Placed").
				CorrelationIDFrom(spec.CorrelationID{Location: "$message.header#/correlationId"}),
			key: "PlacedCorrelationID",
		},
		{
			name: "should_derive_the_key_from_the_payload_type_name",
			msg: MessageOf(OrderPlaced{}).
				CorrelationIDFrom(spec.CorrelationID{Location: "$message.header#/correlationId"}),
			key: "OrderPlacedCorrelationID",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}

			msg, err := tc.msg.build(b)
			require.NoError(t, err)

			require.NotNil(t, msg.CorrelationID)
			assert.Equal(t, "#/components/correlationIds/"+tc.key, msg.CorrelationID.Ref)
			require.Contains(t, b.doc.Components.CorrelationIDs, tc.key)
			assert.Equal(
				t,
				"$message.header#/correlationId",
				b.doc.Components.CorrelationIDs[tc.key].Location,
			)
		})
	}
}

func TestMessageTagsAndExternalDocs(t *testing.T) {
	m := MessageOf(OrderPlaced{}).Name("OrderPlaced").
		Tags(spec.Tag{Name: "orders"}).
		ExternalDocs(spec.ExternalDocs{URL: "https://example.com/message"})

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	msg, err := m.build(b)
	require.NoError(t, err)

	assert.Equal(t, []spec.Tag{{Name: "orders"}}, msg.Tags)
	require.NotNil(t, msg.ExternalDocs)
	assert.Equal(t, "https://example.com/message", msg.ExternalDocs.URL)
}

func TestMessageValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		spec func() *SpecResult
		want string
	}{
		{
			name: "should_return_error_when_message_payload_type_is_nil",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(
						Channel("order-placed").
							Send(Operation().Message(MessageOf(nil))),
					),
				)
			},
			want: "message: nil payload type or schema",
		},
		{
			name: "should_return_error_when_message_references_unknown_message_trait",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(
						Channel("order-placed").
							Send(Operation().
								Message(MessageOf(OrderPlaced{}).
									Name("OrderPlaced").
									Traits(MessageTrait("Missing")))),
					),
				)
			},
			want: `message.OrderPlaced: references unknown message trait "Missing"`,
		},
		{
			name: "should_return_error_when_message_correlation_id_references_unknown_component",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(
						Channel("order-placed").
							Send(Operation().
								Message(MessageOf(OrderPlaced{}).
									Name("OrderPlaced").
									CorrelationID(CorrelationID("Missing", spec.CorrelationID{
										Location: "$message.header#/correlationId",
									})))),
					),
				)
			},
			want: `message.OrderPlaced: references unknown correlation id "Missing"`,
		},
		{
			name: "should_return_error_when_message_sets_both_correlation_forms",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(
						Channel("order-placed").
							Send(Operation().
								Message(MessageOf(OrderPlaced{}).
									Name("OrderPlaced").
									CorrelationID(CorrelationID("CorrelationID", spec.CorrelationID{
										Location: "$message.header#/declared",
									})).
									CorrelationIDFrom(spec.CorrelationID{
										Location: "$message.header#/inline",
									}))),
					),
				)
			},
			want: "message: correlation id reference and inline correlation id are mutually exclusive",
		},
		{
			name: "should_reject_a_message_name_outside_the_key_pattern",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(Channel("order-placed").Send(Operation().Message(
						MessageOf(OrderPlaced{}).Name("Order Placed"),
					))),
				)
			},
			want: "message.Order Placed: name must match ^[\\w\\d.\\-_]+$",
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
