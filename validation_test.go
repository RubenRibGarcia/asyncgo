package asyncgo

import (
	"testing"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidatorsSkipNilEntries covers the post-passes on a hand-assembled
// document: every validator walks maps whose values can be nil (the DSL never
// produces one, but `spec` users can), and a nil entry must be skipped rather
// than dereferenced.
func TestValidatorsSkipNilEntries(t *testing.T) {
	tests := []struct {
		name     string
		validate func(*builder) []error
		mutate   func(*spec.AsyncAPI)
	}{
		{
			name:     "should_skip_a_nil_channel_in_validate_server_refs",
			validate: (*builder).validateServerRefs,
			mutate: func(doc *spec.AsyncAPI) {
				doc.Channels = map[string]*spec.Channel{"order-placed": nil}
			},
		},
		{
			name:     "should_skip_a_nil_server_in_validate_security_refs",
			validate: (*builder).validateSecurityRefs,
			mutate: func(doc *spec.AsyncAPI) {
				doc.Servers = map[string]*spec.Server{"prod": nil}
			},
		},
		{
			name:     "should_skip_a_nil_operation_in_validate_security_refs",
			validate: (*builder).validateSecurityRefs,
			mutate: func(doc *spec.AsyncAPI) {
				doc.Operations = map[string]*spec.Operation{"send": nil}
			},
		},
		{
			name:     "should_skip_a_nil_operation_trait_in_validate_security_refs",
			validate: (*builder).validateSecurityRefs,
			mutate: func(doc *spec.AsyncAPI) {
				doc.Components = &spec.Components{
					OperationTraits: map[string]*spec.OperationTrait{"base": nil},
				}
			},
		},
		{
			name:     "should_skip_a_nil_operation_in_validate_reply_refs",
			validate: (*builder).validateReplyRefs,
			mutate: func(doc *spec.AsyncAPI) {
				doc.Operations = map[string]*spec.Operation{"send": nil}
			},
		},
		{
			name:     "should_skip_a_nil_reply_in_validate_reply_refs",
			validate: (*builder).validateReplyRefs,
			mutate: func(doc *spec.AsyncAPI) {
				doc.Components = &spec.Components{
					Replies: map[string]*spec.OperationReply{"ack": nil},
				}
			},
		},
		{
			name:     "should_skip_a_nil_message_in_validate_message_refs",
			validate: (*builder).validateMessageRefs,
			mutate: func(doc *spec.AsyncAPI) {
				doc.Channels = map[string]*spec.Channel{
					"order-placed": {Messages: map[string]*spec.Message{"OrderPlaced": nil}},
				}
			},
		},
		{
			name:     "should_skip_a_nil_operation_in_validate_operation_messages",
			validate: (*builder).validateOperationMessages,
			mutate: func(doc *spec.AsyncAPI) {
				doc.Operations = map[string]*spec.Operation{"send": nil}
			},
		},
		{
			name:     "should_skip_a_nil_operation_in_validate_trait_refs",
			validate: (*builder).validateTraitRefs,
			mutate: func(doc *spec.AsyncAPI) {
				doc.Operations = map[string]*spec.Operation{"send": nil}
			},
		},
		{
			name:     "should_skip_a_nil_message_trait_in_validate_trait_refs",
			validate: (*builder).validateTraitRefs,
			mutate: func(doc *spec.AsyncAPI) {
				doc.Components = &spec.Components{
					MessageTraits: map[string]*spec.MessageTrait{"base": nil},
				}
			},
		},
		{
			name:     "should_skip_a_nil_message_in_validate_schema_nodes",
			validate: (*builder).validateSchemaNodes,
			mutate: func(doc *spec.AsyncAPI) {
				doc.Components = &spec.Components{
					Messages: map[string]*spec.Message{"OrderPlaced": nil},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := spec.New()
			tt.mutate(doc)
			b := &builder{doc: doc, defs: map[string]*spec.Schema{}}

			var errs []error
			assert.NotPanics(t, func() { errs = tt.validate(b) })
			assert.Empty(t, errs, "a nil entry is skipped, not reported")
		})
	}
}

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

	// A nil entry is skipped rather than dereferenced, as everywhere else in this
	// file. The reply pass shares channelMessageErrors, so it inherits the guard.
	b.doc.Operations["order-placed.send"].Messages = []*spec.Reference{nil}
	assert.Empty(t, b.validateOperationMessages())
}
