package allof

import (
	"github.com/RubenRibGarcia/asyncgo"
	"github.com/RubenRibGarcia/asyncgo/spec"
)

// Catalog is the AsyncAPI description of the orders service. The asyncgo CLI
// discovers it and generates asyncapi.yaml from it.
var Catalog = asyncgo.Spec(
	asyncgo.Info("Orders Service", "1.0.0").
		Description("Order lifecycle events").
		Contact(spec.Contact{Name: "Orders Team", Email: "orders@example.com"}),

	asyncgo.DefaultContentType("application/json"),

	asyncgo.Servers(
		asyncgo.Server("prod", "kafka").
			Host("broker.example.com:9092").
			Description("Production Kafka cluster"),
	),

	asyncgo.Channels(
		asyncgo.Channel("order-placed").
			Description("Emitted when an order is placed").
			Send(asyncgo.Operation().
				Message(asyncgo.MessageOf(OrderPlaced{}).Name("OrderPlaced"))).
			Kafka(spec.KafkaChannelBinding{Topic: "order-placed", Partitions: 3}),
	),
)
