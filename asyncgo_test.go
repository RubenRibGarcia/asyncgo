package asyncgo

import (
	"reflect"
	"testing"

	"github.com/RubenRibGarcia/asyncgo/schema"
	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type OrderPlaced struct {
	OrderID string  `json:"order_id" asyncapi:"required"`
	Amount  float64 `json:"amount"   asyncapi:"required"`
	Note    string  `json:"note"`
}

func TestSpecBuildsDocument(t *testing.T) {
	res := Spec(
		Info("Orders Service", "1.0.0").Description("Order lifecycle events"),
		DefaultContentType("application/json"),
		Servers(Server("prod", "kafka", "broker:9092")),
		Channels(
			Channel("order-placed").
				Description("Emitted when an order is placed").
				Send(Operation().
					Message(MessageOf(OrderPlaced{}).Name("OrderPlaced"))).
				Kafka(spec.KafkaChannelBinding{Topic: "order-placed"}),
		),
	)
	require.NoError(t, res.Err)
	doc := res.Doc

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

	// The channel carries a $ref to the hoisted message, whose payload is a
	// $ref to the schema hoisted under the fully-qualified type name.
	entry := ch.Messages["OrderPlaced"]
	assert.Equal(t, "#/components/messages/OrderPlaced", entry.Ref)

	require.Contains(t, doc.Components.Messages, "OrderPlaced")
	msg := doc.Components.Messages["OrderPlaced"]
	require.NotNil(t, msg.Payload)
	assert.NotEmpty(t, msg.Payload.Ref)

	const key = "github.com/RubenRibGarcia/asyncgo.OrderPlaced"
	require.Contains(t, doc.Components.Schemas, key)
	assert.Len(t, doc.Components.Schemas[key].Required, 2)

	// Channel binding survived.
	assert.Contains(t, ch.Bindings, spec.ProtocolKafka)
}

func TestChannelServers(t *testing.T) {
	prod := Server("prod", "kafka", "broker:9092")
	staging := Server("staging", "kafka", "broker-staging:9092")

	res := Spec(
		Info("Orders Service", "1.0.0"),
		Servers(prod, staging),
		Channels(
			Channel("order-placed").
				Servers(prod, staging).
				Send(Operation().Message(MessageOf(OrderPlaced{}).Name("OrderPlaced"))),
		),
	)
	require.NoError(t, res.Err)
	doc := res.Doc

	ch := doc.Channels["order-placed"]
	require.Len(t, ch.Servers, 2)
	assert.Equal(t, "#/servers/prod", ch.Servers[0].Ref)
	assert.Equal(t, "#/servers/staging", ch.Servers[1].Ref)

	// The referenced servers are declared in the root Servers object.
	require.Contains(t, doc.Servers, "prod")
	require.Contains(t, doc.Servers, "staging")
}

func TestMessageOfInfersName(t *testing.T) {
	assert.Equal(t, "OrderPlaced", messageName(MessageOf(OrderPlaced{})))
}

// MappedID stands in for a third-party type a project maps by name: a named byte
// array, which kind derivation renders as an array of integers.
type MappedID [16]byte

// MappedPayload is the payload whose field the mapping applies to.
type MappedPayload struct {
	ID MappedID `json:"id" asyncapi:"required"`
}

// TestFinalizeRedrivesPayloadWithDeclaredMapping pins the derivation order a
// catalog actually goes through: Spec derives the payload while the catalog
// package's own variables initialize, which is before a harness can register any
// declared mapping, so Finalize has to redo that derivation — otherwise a mapped
// type would apply when reachable from Finalize's combinator hoisting but not in
// the payload that most needs it.
func TestFinalizeRedrivesPayloadWithDeclaredMapping(t *testing.T) {
	payloadKey := schema.Name(reflect.TypeFor[MappedPayload]())
	idKey := schema.Name(reflect.TypeFor[MappedID]())

	res := Spec(
		Info("Mapped", "1.0.0"),
		Channels(
			Channel("mapped").Send(Operation().Message(MessageOf(MappedPayload{}))),
		),
	)
	require.NoError(t, res.Err)
	doc := res.Doc

	// Provisional: derived by kind, with no mapping registered yet.
	require.Contains(t, doc.Components.Schemas, payloadKey)
	id := doc.Components.Schemas[payloadKey].Properties["id"]
	require.NotNil(t, id)
	assert.Equal(t, "array", id.Type)

	schema.MapSchemaByName(idKey, &spec.Schema{Type: "string", Format: "uuid"})
	schema.Finalize(doc)

	id = doc.Components.Schemas[payloadKey].Properties["id"]
	require.NotNil(t, id)
	assert.Equal(t, "string", id.Type)
	assert.Equal(t, "uuid", id.Format)
}
