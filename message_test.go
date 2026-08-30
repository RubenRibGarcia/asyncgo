package asyncgo

import (
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
}
