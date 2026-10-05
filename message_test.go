package asyncgo

import (
	"reflect"
	"testing"

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
			typ:     reflect.TypeOf(OrderPlaced{}),
			payload: &spec.Schema{Type: "object"},
		}

		sm, err := m.build(b)

		assert.Nil(t, sm)
		assert.EqualError(t, err,
			"message: payload schema and payload type are mutually exclusive")
	})
}
