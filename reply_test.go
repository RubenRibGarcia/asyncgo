package asyncgo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplyFields(t *testing.T) {
	replyTo := ReplyAddress("ReplyTo").Location("$message.header#/replyTo")

	r := Reply("OrderReply").Address(replyTo)

	assert.Equal(t, "OrderReply", r.name)
	require.NotNil(t, r.r.Address)
	assert.Equal(t, "#/components/replyAddresses/ReplyTo", r.r.Address.Ref)
	assert.Nil(t, r.r.Channel)
	assert.Empty(t, r.r.Messages)
}

func TestReplyChannelAndMessages(t *testing.T) {
	orderReplies := Channel("order-replies")
	accepted := MessageOf(OrderPlaced{}).Name("OrderAccepted")
	rejected := MessageOf(OrderPlaced{}).Name("OrderRejected")

	r := Reply("OrderReply").
		Channel(orderReplies).
		Message(orderReplies, accepted).
		Message(orderReplies, rejected)

	require.NotNil(t, r.r.Channel)
	assert.Equal(t, "#/channels/order-replies", r.r.Channel.Ref)
	require.Len(t, r.r.Messages, 2)
	assert.Equal(t, "#/channels/order-replies/messages/OrderAccepted", r.r.Messages[0].Ref)
	assert.Equal(t, "#/channels/order-replies/messages/OrderRejected", r.r.Messages[1].Ref)
	assert.Nil(t, r.r.Address)
}

// Message names its channel, so the refs it builds do not depend on Channel
// having been called first.
func TestReplyMessageRefsAreOrderIndependent(t *testing.T) {
	orderReplies := Channel("order-replies")
	accepted := MessageOf(OrderPlaced{}).Name("OrderAccepted")

	messageFirst := Reply("A").Message(orderReplies, accepted).Channel(orderReplies)
	channelFirst := Reply("B").Channel(orderReplies).Message(orderReplies, accepted)

	assert.Equal(t, messageFirst.r.Messages, channelFirst.r.Messages)
	assert.Equal(t, messageFirst.r.Channel, channelFirst.r.Channel)
}

func TestRepliesRegistersComponents(t *testing.T) {
	replyTo := ReplyAddress("ReplyTo").Location("$message.header#/replyTo")

	res := Spec(
		Info("Orders", "1.0.0"),
		ReplyAddresses(replyTo),
		Replies(Reply("OrderReply").Address(replyTo)),
	)

	require.NoError(t, res.Err)
	require.NotNil(t, res.Doc.Components)
	assert.Len(t, res.Doc.Components.Replies, 1)
	assert.Equal(
		t,
		"#/components/replyAddresses/ReplyTo",
		res.Doc.Components.Replies["OrderReply"].Address.Ref,
	)
}

// TestReplyMessageMustBeOnTheReplyChannel pins the tightened reply check: a
// reply message $ref has to name a message the reply channel actually carries,
// not merely one whose pointer starts with the channel's prefix.
func TestReplyMessageMustBeOnTheReplyChannel(t *testing.T) {
	replyChannel := Channel("order-replies").Receive(Operation().Message(
		MessageOf(OrderPlaced{}).Name("OrderAccepted"),
	))
	missing := MessageOf(OrderPlaced{}).Name("OrderMissing")

	res := Spec(
		Info("Orders", "1.0.0"),
		Replies(Reply("OrderReply").Channel(replyChannel).Message(replyChannel, missing)),
		Channels(replyChannel),
	)

	require.Error(t, res.Err)
	assert.EqualError(
		t,
		res.Err,
		`reply.OrderReply: message "#/channels/order-replies/messages/OrderMissing" is not in channel "order-replies"`,
	)
}

func TestReplyValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		spec func() *SpecResult
		want string
	}{
		{
			name: "should_return_error_when_reply_name_is_empty",
			spec: func() *SpecResult {
				return Spec(Info("Orders", "1.0.0"), Replies(Reply("")))
			},
			want: "reply.name: is required",
		},
		{
			name: "should_return_error_when_reply_name_is_duplicate",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Replies(Reply("OrderReply"), Reply("OrderReply")),
				)
			},
			want: "reply.OrderReply: duplicate name",
		},
		{
			name: "should_return_error_when_reply_references_unknown_reply_address",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Replies(Reply("OrderReply").Address(ReplyAddress("ReplyTo"))),
				)
			},
			want: `reply.OrderReply: references unknown reply address "ReplyTo"`,
		},
		{
			name: "should_return_error_when_reply_references_unknown_channel",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Replies(Reply("OrderReply").Channel(Channel("order-replies"))),
				)
			},
			want: `reply.OrderReply: references unknown channel "order-replies"`,
		},
		{
			name: "should_return_error_when_reply_sets_address_and_channel",
			spec: func() *SpecResult {
				replyTo := ReplyAddress("ReplyTo").Location("$message.header#/replyTo")
				orderReplies := Channel("order-replies")
				return Spec(
					Info("Orders", "1.0.0"),
					ReplyAddresses(replyTo),
					Channels(orderReplies),
					Replies(Reply("OrderReply").Address(replyTo).Channel(orderReplies)),
				)
			},
			want: "reply.OrderReply: address and channel are mutually exclusive (the referenced channel must have no address)",
		},
		{
			name: "should_return_error_when_reply_messages_have_no_channel",
			spec: func() *SpecResult {
				orderReplies := Channel("order-replies")
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(orderReplies),
					Replies(Reply("OrderReply").Message(
						orderReplies,
						MessageOf(OrderPlaced{}).Name("OrderAccepted"),
					)),
				)
			},
			want: "reply.OrderReply: messages require a channel",
		},
		{
			name: "should_return_error_when_reply_message_is_not_in_referenced_channel",
			spec: func() *SpecResult {
				orderPlaced := Channel("order-placed")
				orderReplies := Channel("order-replies")
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(orderPlaced, orderReplies),
					Replies(Reply("OrderReply").
						Channel(orderReplies).
						Message(orderPlaced, MessageOf(OrderPlaced{}).Name("OrderAccepted"))),
				)
			},
			want: `reply.OrderReply: message "#/channels/order-placed/messages/OrderAccepted" is not in channel "order-replies"`,
		},
		{
			name: "should_reject_a_reply_name_outside_the_key_pattern",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Replies(Reply("tenant reply")),
				)
			},
			want: "reply.tenant reply: name must match ^[\\w\\d.\\-_]+$",
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
