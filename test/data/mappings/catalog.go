package mappings

import "github.com/RubenRibGarcia/asyncgo"

// Catalog is the AsyncAPI description of a service publishing orders whose
// payload includes types asyncgo does not own. The asyncgo CLI discovers it and
// generates asyncapi.yaml from it, reading asyncgo.yaml from the same directory.
var Catalog = asyncgo.Spec(
	asyncgo.Info("External Type Mappings", "1.0.0").
		Description("Payload types shaped by asyncgo.yaml, derived aliases, and built-in mappings"),

	asyncgo.Channels(
		asyncgo.Channel("order-placed").
			Description("Emitted when an order is placed").
			Send(asyncgo.Operation().Message(asyncgo.MessageOf(OrderPlaced{}))),
	),
)
