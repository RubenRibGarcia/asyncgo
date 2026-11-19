package nullable

import "github.com/RubenRibGarcia/asyncgo"

// orderUpdated carries the payload the fixture is named for.
var orderUpdated = asyncgo.MessageOf(OrderUpdated{})

// Catalog is the AsyncAPI description of a service publishing order updates. The
// asyncgo CLI discovers it and generates asyncapi.yaml from it.
var Catalog = asyncgo.Spec(
	asyncgo.Info("Nullable Values", "1.0.0").
		Description("A payload whose nilable positions derive oneOf with a null branch"),

	asyncgo.Channels(
		asyncgo.Channel("order-updated").
			Description("Emitted when an order changes").
			Send(asyncgo.Operation().Message(orderUpdated)),
	),
)
