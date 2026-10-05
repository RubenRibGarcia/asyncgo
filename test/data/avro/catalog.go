// Package avro is a discovery fixture for the AsyncAPI 3.1.0 Multi Format Schema
// Object. It declares one Avro record in components.schemas and references it
// from two messages, carries a multi-format headers schema, and gives a third
// message an inline Avro payload. It exercises components.schemas,
// Message.payload, Message.headers, and the hand-authored MessageFrom path end
// to end.
//
// Every body is Avro because the pinned AsyncAPI CLI validates Avro multi-format
// schemas but has no parser registered for the other formats — a Protobuf or RAML
// payload fails validation with no reported error. Protobuf is still emittable
// through spec.MultiFormat; it just cannot be checked by the reference validator.
package avro

import (
	"github.com/RubenRibGarcia/asyncgo"
	"github.com/RubenRibGarcia/asyncgo/spec"
)

// userAvro is the Avro record the order messages carry. The body is opaque:
// asyncgo emits schemaFormat verbatim and never parses or rewrites schema.
var userAvro = spec.MultiFormat("application/vnd.apache.avro;version=1.9.0",
	map[string]any{
		"type": "record",
		"name": "User",
		"fields": []any{
			map[string]any{"name": "displayName", "type": "string"},
			map[string]any{"name": "age", "type": "int"},
		},
	})

// orderHeaders is a Multi Format Schema Object used for headers rather than a
// payload: the same node is accepted in both positions.
var orderHeaders = spec.MultiFormat("application/vnd.apache.avro;version=1.9.0",
	map[string]any{
		"type": "record",
		"name": "OrderHeaders",
		"fields": []any{
			map[string]any{"name": "correlationId", "type": "string"},
		},
	})

// Catalog is the AsyncAPI description of an orders service whose payloads are
// Avro records. The asyncgo CLI discovers it and generates asyncapi.yaml from it.
var Catalog = asyncgo.Spec(
	asyncgo.Info("Avro Orders Service", "1.0.0").
		Description("Order events whose payloads are Avro records"),

	asyncgo.Schemas(
		asyncgo.Schema("UserAvro", userAvro),
	),

	asyncgo.Channels(
		asyncgo.Channel("order-placed").
			Description("Order placements").
			Send(asyncgo.Operation().
				Message(asyncgo.MessageFrom("OrderPlaced",
					spec.Ref("#/components/schemas/UserAvro")).
					Headers(orderHeaders))),
		asyncgo.Channel("order-shipped").
			Description("Order shipments").
			Send(asyncgo.Operation().
				Message(asyncgo.MessageFrom("OrderShipped",
					spec.Ref("#/components/schemas/UserAvro")))),
		asyncgo.Channel("order-cancelled").
			Description("Order cancellations").
			Send(asyncgo.Operation().
				Message(asyncgo.MessageFrom("OrderCancelled",
					spec.MultiFormat("application/vnd.apache.avro;version=1.9.0",
						map[string]any{
							"type": "record",
							"name": "OrderCancelled",
							"fields": []any{
								map[string]any{"name": "orderId", "type": "string"},
							},
						})))),
	),
)
