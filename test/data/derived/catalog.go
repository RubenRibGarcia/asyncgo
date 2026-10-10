package derived

import "github.com/RubenRibGarcia/asyncgo"

// orderPlaced names no explicit message name, so its components.messages key is
// the payload type's fully-qualified name — the same key its hoisted schema
// uses. The same value is carried by both channels, so it is hoisted once and
// referenced from two channel messages maps.
var orderPlaced = asyncgo.MessageOf(OrderPlaced{})

// Catalog is the AsyncAPI description of an order service that publishes one
// message on two channels. The asyncgo CLI discovers it and generates
// asyncapi.yaml from it.
var Catalog = asyncgo.Spec(
	asyncgo.Info("Derived Message Keys", "1.0.0").
		Description("One payload type hoisted once and shared by two channels"),

	asyncgo.Channels(
		asyncgo.Channel("order-placed").
			Description("Order placements").
			Send(asyncgo.Operation().Message(orderPlaced)),
		asyncgo.Channel("order-shipped").
			Description("Order shipments carrying the same message").
			Send(asyncgo.Operation().Message(orderPlaced)),
	),
)
