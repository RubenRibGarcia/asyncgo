package asyncgo

import (
	"testing"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateMessageRefs covers the post-pass that guards a document assembled
// through the spec package: a channel message entry that is a $ref has to
// resolve to a declared component. The generator hoists and references a message
// in one step, so the dangling document is built by hand.
func TestValidateMessageRefs(t *testing.T) {
	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	b.doc.Channels = map[string]*spec.Channel{
		"order-placed": {
			Address: "order-placed",
			Messages: map[string]*spec.Message{
				"OrderPlaced": {Ref: "#/components/messages/OrderPlaced"},
			},
		},
	}

	errs := b.validateMessageRefs()
	require.Len(t, errs, 1)
	assert.EqualError(
		t,
		errs[0],
		`channel.order-placed.messages.OrderPlaced: references unknown message "OrderPlaced"`,
	)

	b.components().Messages = map[string]*spec.Message{
		"OrderPlaced": {Name: "OrderPlaced"},
	}
	assert.Empty(t, b.validateMessageRefs())
}

// TestValidateOperationMessages covers the post-pass that enforces
// operation.messages ⊆ channel.messages. The generator builds a channel's
// messages map and its operations' message refs in lockstep, so a violating
// document has to be built by hand.
func TestValidateOperationMessages(t *testing.T) {
	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	b.doc.Channels = map[string]*spec.Channel{
		"order-placed": {
			Address: "order-placed",
			Messages: map[string]*spec.Message{
				"OrderPlaced": {Ref: "#/components/messages/OrderPlaced"},
			},
		},
	}
	b.doc.Operations = map[string]*spec.Operation{
		"order-placed.send": {
			Channel:  &spec.Reference{Ref: "#/channels/order-placed"},
			Messages: []*spec.Reference{{Ref: "#/channels/order-placed/messages/OrderShipped"}},
		},
	}

	errs := b.validateOperationMessages()
	require.Len(t, errs, 1)
	assert.EqualError(
		t,
		errs[0],
		`operation.order-placed.send: message "#/channels/order-placed/messages/OrderShipped" `+
			`is not in channel "order-placed"`,
	)

	b.doc.Channels["order-placed"].Messages["OrderShipped"] = &spec.Message{
		Ref: "#/components/messages/OrderShipped",
	}
	assert.Empty(t, b.validateOperationMessages())

	// A ref that names another channel's message fails on the prefix, before the
	// key lookup.
	b.doc.Operations["order-placed.send"].Messages = []*spec.Reference{
		{Ref: "#/channels/order-shipped/messages/OrderPlaced"},
	}
	errs = b.validateOperationMessages()
	require.Len(t, errs, 1)
	assert.EqualError(
		t,
		errs[0],
		`operation.order-placed.send: message "#/channels/order-shipped/messages/OrderPlaced" `+
			`is not in channel "order-placed"`,
	)
}
