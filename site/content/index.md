---
hide:
  - navigation
  - toc
---

# AsyncAPI from Go, not Go from AsyncAPI

Generate an [AsyncAPI 3.1.0](https://www.asyncapi.com/docs/reference/specification/v3.1.0)
specification document from Go code — the *code → spec* direction that most Go
AsyncAPI tooling (which goes *spec → code*) leaves unserved.

`asyncgo` is a **documentation generator**, not a messaging framework. It does not
route your messaging. It derives a committed `asyncapi.yaml` from two touchpoints
in your code, so the document cannot quietly drift from the service it describes.

[Get started](getting-started.md){ .md-button .md-button--primary }
[pkg.go.dev](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo){ .md-button }

## Two touchpoints

### 1. The struct is the data contract

Message payloads are derived from your structs by reflection. `json` tags name the
fields, field doc comments become descriptions, and an `asyncapi` tag carries the
constraints the JSON encoding cannot express.

```go
// OrderPlaced is the message emitted when an order is placed.
type OrderPlaced struct {
	OrderID string  `json:"order_id" asyncapi:"required"`
	Amount  float64 `json:"amount"   asyncapi:"required"`
	// Optional note from the customer
	Note string `json:"note"`
}
```

### 2. The catalog is the topology

Channels and operations are declared once, in a typed catalog the compiler
checks. `MessageOf` *references* the struct instead of duplicating its shape, so
the schema and the data contract are the same declaration.

```go
package orders

import "github.com/RubenRibGarcia/asyncgo"

// Catalog is the AsyncAPI description of the orders service.
var Catalog = asyncgo.Spec(
	asyncgo.Info("Orders Service", "1.0.0"),
	asyncgo.Servers(asyncgo.Server("prod", "kafka", "broker.example.com:9092")),
	asyncgo.Channels(
		asyncgo.Channel("order-placed").
			Send(asyncgo.Operation().
				Message(asyncgo.MessageOf(OrderPlaced{}).Name("OrderPlaced"))),
	),
)
```

## What you get

`asyncgo generate .` writes this, and `asyncgo check .` keeps it that way:

```yaml
asyncapi: 3.1.0
info:
  title: Orders Service
  version: 1.0.0
servers:
  prod:
    host: broker.example.com:9092
    protocol: kafka
channels:
  order-placed:
    address: order-placed
    messages:
      OrderPlaced:
        $ref: "#/components/messages/OrderPlaced"
operations:
  order-placed.send:
    action: send
    channel:
      $ref: "#/channels/order-placed"
    messages:
    - $ref: "#/channels/order-placed/messages/OrderPlaced"
components:
  schemas:
    github.com/acme/orders.OrderPlaced:
      type: object
      properties:
        amount:
          type: number
          format: double
        note:
          type: string
          description: Optional note from the customer
        order_id:
          type: string
      required:
      - order_id
      - amount
  messages:
    OrderPlaced:
      payload:
        $ref: "#/components/schemas/github.com~1acme~1orders.OrderPlaced"
      name: OrderPlaced
```

Messages are hoisted into `components.messages`, and every channel, operation and
reply that names a message points at it with a `$ref` — so one message is defined
once, however many places reference it.

## What it covers

<div class="grid cards" markdown>

-   **Schema derivation**

    ---

    Structs, embedding (`allOf`), `asyncapi` tags for `enum`, `examples`,
    `format`, `contentEncoding`, `oneOf`/`anyOf`, and the draft-07 keywords —
    derived, not hand-written.

-   **20 protocol bindings**

    ---

    Typed builders for every protocol in the 3.1.0 spec's binding set, from
    Kafka to WebSockets. Unknown keys are a compile error, not a typo in YAML.

-   **Request/reply, traits, security**

    ---

    Reply channels and addresses, reusable operation and message traits, and
    security schemes with typed requirement objects.

-   **Multi-format schemas**

    ---

    Avro and Protobuf schema objects alongside JSON Schema, for the payloads
    that are not JSON.

-   **`asyncgo check` for CI**

    ---

    Byte-for-byte comparison against what `generate` would write, with a
    dedicated exit code for a stale artifact — so "regenerate the spec" is a
    distinct failure from "the build is broken".

-   **Third-party types**

    ---

    Declared mappings for types you do not own — `time.Time`, `uuid.UUID`,
    `decimal.Decimal` — so a foreign type becomes a documented schema, not an
    opaque object.

</div>

## Next

- **[Getting started](getting-started.md)** — install the CLI, declare a catalog,
  generate, and wire `asyncgo check` into CI.
- **[The README](https://github.com/RubenRibGarcia/asyncgo#readme)** — the full
  reference today: bindings, traits, providers, and the derivation rules. It is
  being migrated into this site page by page.
