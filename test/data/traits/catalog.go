// Package traits is a discovery fixture for the AsyncAPI 3.1.0 traits: one
// Operation Trait shared by two operations, a Message Trait carrying a
// Correlation ID, and Tags/ExternalDocs on the server, channel, operation, and
// message. It exercises components.operationTraits, components.messageTraits,
// and components.correlationIds end to end.
package traits

import (
	"github.com/RubenRibGarcia/asyncgo"
	"github.com/RubenRibGarcia/asyncgo/spec"
)

// orderCorrelationID is the correlation id a traced message is matched by. It
// is declared once and referenced from the message trait.
var orderCorrelationID = asyncgo.CorrelationID("OrderCorrelationID", spec.CorrelationID{
	Description: "Correlation ID carried in the message headers",
	Location:    "$message.header#/correlationId",
})

// tracedMessage is the Message Trait both order messages apply: a shared
// content type, the correlation id, and a shared tag.
var tracedMessage = asyncgo.MessageTrait("TracedMessage").
	ContentType("application/json").
	CorrelationID(orderCorrelationID).
	Tags(spec.Tag{Name: "traced"})

// kafkaOrders is the Operation Trait both operations share: one declaration in
// components.operationTraits, referenced twice. groupId/clientId are 3.1.0
// Schema Objects, so they are given as schemas rather than strings.
var kafkaOrders = asyncgo.OperationTrait("KafkaOrders").
	Summary("Shared Kafka settings for order operations").
	Tags(spec.Tag{Name: "orders"}).
	Kafka(spec.KafkaOperationBinding{
		GroupID:  &spec.Schema{Type: "string"},
		ClientID: &spec.Schema{Type: "string"},
	})

// prod is the Kafka cluster both channels are available on.
var prod = asyncgo.Server("prod", "kafka", "broker.example.com:9092").
	Description("Production Kafka cluster").
	Tags(spec.Tag{Name: "prod"}).
	ExternalDocs(spec.ExternalDocs{URL: "https://example.com/servers/prod"})

// placedChannel carries order placements. Its send operation and its message
// both carry the shared operation/message traits.
var placedChannel = asyncgo.Channel("order-placed").
	Description("Order placements").
	Tags(spec.Tag{Name: "orders"}).
	ExternalDocs(spec.ExternalDocs{URL: "https://example.com/channels/order-placed"}).
	Servers(prod).
	Send(asyncgo.Operation().
		Traits(kafkaOrders).
		Tags(spec.Tag{Name: "write"}).
		ExternalDocs(spec.ExternalDocs{URL: "https://example.com/operations/order-placed-send"}).
		Message(asyncgo.MessageOf(OrderPlaced{}).
			Name("OrderPlaced").
			Traits(tracedMessage).
			Tags(spec.Tag{Name: "orders"}).
			ExternalDocs(spec.ExternalDocs{URL: "https://example.com/messages/order-placed"})))

// shippedChannel carries order shipments and shares the same operation trait as
// placedChannel, so the golden shows one component referenced by two operations.
var shippedChannel = asyncgo.Channel("order-shipped").
	Description("Order shipments").
	Servers(prod).
	Send(asyncgo.Operation().
		Traits(kafkaOrders).
		Message(asyncgo.MessageOf(OrderShipped{}).
			Name("OrderShipped").
			Traits(tracedMessage)))

// Catalog is the AsyncAPI description of an orders service whose operations
// share a Kafka operation trait. The asyncgo CLI discovers it and generates
// asyncapi.yaml from it.
var Catalog = asyncgo.Spec(
	asyncgo.Info("Traited Orders Service", "1.0.0").
		Description("Order events whose operations share a Kafka trait"),

	asyncgo.OperationTraits(kafkaOrders),
	asyncgo.MessageTraits(tracedMessage),
	asyncgo.CorrelationIDs(orderCorrelationID),

	asyncgo.Servers(prod),

	asyncgo.Channels(placedChannel, shippedChannel),
)
