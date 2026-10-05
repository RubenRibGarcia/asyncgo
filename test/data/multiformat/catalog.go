// Package multiformat is a discovery fixture for the non-Avro schema formats the
// AsyncAPI 3.1.0 Multi Format Schema Object supports: OpenAPI 3.0.0, RAML 1.0,
// and Protocol Buffers. Each channel carries one format inline, so the golden
// shows schemaFormat emitted verbatim with an opaque body for all three.
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
// the shape a Protobuf schema naturally takes.
var protobufPayload = spec.MultiFormat("application/vnd.google.protobuf;version=3",
	"message OrderCancelled { string order_id = 1; }")

// Catalog is the AsyncAPI description of an orders service that carries one
// payload per non-Avro multi-format schema. The asyncgo CLI discovers it and
// generates asyncapi.yaml from it.
var Catalog = asyncgo.Spec(
	asyncgo.Info("Multi Format Orders Service", "1.0.0").
		Description("Order events carried in OpenAPI, RAML, and Protobuf schema formats"),

	asyncgo.Channels(
		asyncgo.Channel("order-openapi").
			Description("Order placements described by an OpenAPI 3.0.0 schema").
			Send(asyncgo.Operation().
				Message(asyncgo.MessageFrom("OrderPlaced", openAPIPayload))),
		asyncgo.Channel("order-raml").
			Description("Order shipments described by a RAML 1.0 data type").
			Send(asyncgo.Operation().
				Message(asyncgo.MessageFrom("OrderShipped", ramlPayload))),
		asyncgo.Channel("order-protobuf").
			Description("Order cancellations described by a Protobuf message").
			Send(asyncgo.Operation().
				Message(asyncgo.MessageFrom("OrderCancelled", protobufPayload))),
	),
)
