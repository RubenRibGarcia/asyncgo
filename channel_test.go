package asyncgo

import (
	"testing"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelReceive(t *testing.T) {
	c := Channel("order-placed").
		Title("Order placed").
		Receive(Operation().Message(MessageOf(OrderPlaced{}).Name("OrderPlaced")))

	assert.Equal(t, "Order placed", c.s.Title)

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	require.NoError(t, c.apply(b))

	require.Contains(t, b.doc.Operations, "order-placed.receive")
	op := b.doc.Operations["order-placed.receive"]
	assert.Equal(t, spec.ActionReceive, op.Action)
	assert.Equal(t, "#/channels/order-placed", op.Channel.Ref)
	require.Contains(t, b.doc.Channels["order-placed"].Messages, "OrderPlaced")
}

// TestChannelParameterEmitsParameter covers the success half of the address
// expression rule: the two rejection paths live in the TestChannelValidationErrors table,
// so without this a parameter that validates could still fail to reach the
// document.
func TestChannelParameterEmitsParameter(t *testing.T) {
	userId := spec.Parameter{
		Enum:        []string{"u_1", "u_2"},
		Default:     "u_1",
		Description: "Id of the user.",
		Examples:    []string{"u_1"},
		Location:    "$message.payload#/userId",
	}

	res := Spec(
		Info("Orders", "1.0.0"),
		Channels(Channel("users.{userId}").Parameter("userId", userId)),
	)

	require.NoError(t, res.Err)
	require.Contains(t, res.Doc.Channels, "users.{userId}")
	assert.Equal(t, &userId, res.Doc.Channels["users.{userId}"].Parameters["userId"])
}

func TestChannelTagsAndExternalDocs(t *testing.T) {
	c := Channel("order-placed").
		Tags(spec.Tag{Name: "orders"}).
		ExternalDocs(spec.ExternalDocs{URL: "https://example.com/channel"})

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	require.NoError(t, c.apply(b))

	assert.Equal(t, []spec.Tag{{Name: "orders"}}, b.doc.Channels["order-placed"].Tags)
	require.NotNil(t, b.doc.Channels["order-placed"].ExternalDocs)
	assert.Equal(t, "https://example.com/channel", b.doc.Channels["order-placed"].ExternalDocs.URL)
}

func TestChannelValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		spec func() *SpecResult
		want string
	}{
		{
			name: "should_return_error_when_channel_address_is_duplicate",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(
						Channel("order-placed"),
						Channel("order-placed"),
					),
				)
			},
			want: "channel.order-placed: duplicate address",
		},
		{
			name: "should_return_error_when_channel_address_is_duplicate_across_channels_calls",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(Channel("order-placed")),
					Channels(Channel("order-placed")),
				)
			},
			want: "channel.order-placed: duplicate address",
		},
		{
			name: "should_return_error_when_channel_references_unknown_server",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(
						Channel("order-placed").
							Servers(Server("prod", "kafka", "broker:9092")),
					),
				)
			},
			want: `channel.order-placed: references unknown server "prod"`,
		},
		{
			name: "should_join_multiple_unknown_server_references",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(
						Channel("order-placed").
							Servers(
								Server("prod", "kafka", "broker:9092"),
								Server("staging", "kafka", "broker-staging:9092"),
							),
					),
				)
			},
			want: "channel.order-placed: references unknown server \"prod\"\n" +
				"channel.order-placed: references unknown server \"staging\"",
		},
		{
			name: "should_join_unknown_server_references_across_channels_in_sorted_order",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(
						Channel("order-shipped").
							Servers(Server("staging", "kafka", "broker-staging:9092")),
						Channel("order-placed").
							Servers(Server("prod", "kafka", "broker:9092")),
					),
				)
			},
			want: "channel.order-placed: references unknown server \"prod\"\n" +
				"channel.order-shipped: references unknown server \"staging\"",
		},
		{
			// The one reachable shape: a duplicate channel address whose second
			// declaration uses a different action. That operation is registered under
			// its own key, but the message it carries was written into the duplicate
			// channel that the first declaration displaced.
			name: "should_return_error_when_an_operation_carries_a_message_the_channel_lacks",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(
						Channel("order-placed").Send(Operation().Message(
							MessageOf(OrderPlaced{}).Name("OrderPlaced"),
						)),
						Channel("order-placed").Receive(Operation().Message(
							MessageOf(OrderPlaced{}).Name("OrderShipped"),
						)),
					),
				)
			},
			want: "channel.order-placed: duplicate address\n" +
				`operation.order-placed.receive: message "#/channels/order-placed/messages/OrderShipped" ` +
				`is not in channel "order-placed"`,
		},
		{
			name: "should_return_error_when_an_address_expression_has_no_parameter",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(Channel("orders/{orderId}")),
				)
			},
			want: "channel.orders/{orderId}: address expression {orderId} has no declared parameter",
		},
		{
			name: "should_return_error_when_a_parameter_is_not_used_in_the_address",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(Channel("orders").Parameter(
						"orderId",
						spec.Parameter{Description: "The order id"},
					)),
				)
			},
			want: `channel.orders: parameter "orderId" is not used in the address`,
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
