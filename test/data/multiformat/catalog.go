// Package multiformat is a discovery fixture for the AsyncAPI 3.1.0 Multi Format
// Schema Object. It carries every format the specification's supported-formats
// table recommends: Avro 1.9.0, OpenAPI 3.0.0, RAML 1.0, and Protocol Buffers.
// Each channel carries one format inline, so the golden shows schemaFormat
// emitted verbatim with an opaque body for all four; the Avro channels also share
// one components.schemas declaration between two messages and use a multi-format
// schema for headers.
//
// The OpenAPI and RAML bodies are objects rather than the idiomatic format text
// because the pinned AsyncAPI CLI resolves a string-valued `schema` as a
// reference and reports a governance issue with an empty error list — an object
// body is the only shape it validates, for every format. Protobuf has no parser
// registered at all: every body shape is rejected the same way, which is why the
// integration test excludes this fixture from the CLI check.
package multiformat

import (
	"github.com/RubenRibGarcia/asyncgo"
	"github.com/RubenRibGarcia/asyncgo/spec"
)

// userAvro is the Avro record the order messages carry, declared once in
// components.schemas and referenced from two messages. The body is opaque:
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

// refundedAvro is an inline Avro payload: a multi-format schema used without a
// components.schemas declaration.
var refundedAvro = spec.MultiFormat("application/vnd.apache.avro;version=1.9.0",
	map[string]any{
		"type": "record",
		"name": "OrderRefunded",
		"fields": []any{
			map[string]any{"name": "orderId", "type": "string"},
		},
	})

// openAPIPayload is an OpenAPI 3.0.0 Schema Object describing an order.
var openAPIPayload = spec.MultiFormat("application/vnd.oai.openapi;version=3.0.0",
	map[string]any{
		"type": "object",
		"properties": map[string]any{
			"orderId": map[string]any{"type": "string", "format": "uuid"},
			"amount":  map[string]any{"type": "number", "format": "double"},
		},
		"required": []any{"orderId"},
	})

// ramlPayload is a RAML 1.0 data type describing an order.
var ramlPayload = spec.MultiFormat("application/raml+yaml;version=1.0",
	map[string]any{
		"type": "object",
		"properties": map[string]any{
			"orderId": map[string]any{"type": "string"},
			"carrier": map[string]any{"type": "string"},
		},
		"required": []any{"orderId"},
	})

// protobufPayload is a Protocol Buffers definition. Its body is the .proto text,
// the shape a Protobuf schema naturally takes, so it spans lines and is emitted
// as a YAML literal block.
var protobufPayload = spec.MultiFormat("application/vnd.google.protobuf;version=3",
	`message OrderCancelled {
  string order_id = 1;
  string reason = 2;
}
`)

// Catalog is the AsyncAPI description of an orders service that carries one
// payload per supported multi-format schema. The asyncgo CLI discovers it and
// generates asyncapi.yaml from it.
var Catalog = asyncgo.Spec(
	asyncgo.Info("Multi Format Orders Service", "1.0.0").
		Description("Order events carried in Avro, OpenAPI, RAML, and Protobuf schema formats"),

	asyncgo.Schemas(
		asyncgo.Schema("UserAvro", userAvro),
	),

	asyncgo.Channels(
		asyncgo.Channel("order-avro-placed").
			Description("Order placements carrying the shared Avro record and Avro headers").
			Send(asyncgo.Operation().
				Message(asyncgo.MessageFrom("OrderPlaced",
					spec.Ref("#/components/schemas/UserAvro")).
					Headers(orderHeaders))),
		asyncgo.Channel("order-avro-shipped").
			Description("Order shipments carrying the shared Avro record").
			Send(asyncgo.Operation().
				Message(asyncgo.MessageFrom("OrderShipped",
					spec.Ref("#/components/schemas/UserAvro")))),
		asyncgo.Channel("order-avro-refunded").
			Description("Order refunds carrying an inline Avro record").
			Send(asyncgo.Operation().
				Message(asyncgo.MessageFrom("OrderRefunded", refundedAvro))),
		asyncgo.Channel("order-openapi").
			Description("Order placements described by an OpenAPI 3.0.0 schema").
			Send(asyncgo.Operation().
				Message(asyncgo.MessageFrom("OrderPlacedOpenAPI", openAPIPayload))),
		asyncgo.Channel("order-raml").
			Description("Order shipments described by a RAML 1.0 data type").
			Send(asyncgo.Operation().
				Message(asyncgo.MessageFrom("OrderShippedRAML", ramlPayload))),
		asyncgo.Channel("order-protobuf").
			Description("Order cancellations described by a Protobuf message").
			Send(asyncgo.Operation().
				Message(asyncgo.MessageFrom("OrderCancelledProtobuf", protobufPayload))),
	),
)
