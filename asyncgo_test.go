package asyncgo

import (
	"testing"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type OrderPlaced struct {
	OrderID string  `json:"order_id" asyncgo:"required"`
	Amount  float64 `json:"amount"   asyncgo:"required"`
	Note    string  `json:"note"`
}

func TestSpecBuildsDocument(t *testing.T) {
	doc := Spec(
		Info("Orders Service", "1.0.0").Description("Order lifecycle events"),
		DefaultContentType("application/json"),
		Servers(Server("prod", "kafka").Host("broker:9092")),
		Channels(
			Channel("order-placed").
				Description("Emitted when an order is placed").
				Send(Operation().
					Message(MessageOf(OrderPlaced{}).Name("OrderPlaced"))).
				Kafka(spec.KafkaChannelBinding{Topic: "order-placed"}),
		),
	)

	assert.Equal(t, "3.1.0", doc.AsyncAPI)
	assert.Equal(t, "Orders Service", doc.Info.Title)
	assert.Equal(t, "application/json", doc.DefaultContentType)

	require.Contains(t, doc.Servers, "prod")
	assert.Equal(t, "broker:9092", doc.Servers["prod"].Host)

	require.Contains(t, doc.Channels, "order-placed")
	ch := doc.Channels["order-placed"]
	assert.Equal(t, "order-placed", ch.Address)
	require.Contains(t, ch.Messages, "OrderPlaced")

	require.Contains(t, doc.Operations, "order-placed.send")
	op := doc.Operations["order-placed.send"]
	assert.Equal(t, spec.ActionSend, op.Action)
	require.NotNil(t, op.Channel)
	assert.Equal(t, "#/channels/order-placed", op.Channel.Ref)

	// Payload is a $ref; schema is hoisted into components under the
	// fully-qualified type name.
	msg := ch.Messages["OrderPlaced"]
	require.NotNil(t, msg.Payload)
	assert.NotEmpty(t, msg.Payload.Ref)

	const key = "github.com/RubenRibGarcia/asyncgo.OrderPlaced"
	require.Contains(t, doc.Components.Schemas, key)
	assert.Len(t, doc.Components.Schemas[key].Required, 2)

	// Channel binding survived.
	assert.Contains(t, ch.Bindings, spec.ProtocolKafka)
}

func TestMessageOfInfersName(t *testing.T) {
	assert.Equal(t, "OrderPlaced", messageName(MessageOf(OrderPlaced{})))
}
