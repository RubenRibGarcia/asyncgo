# asyncgo

Generate an [AsyncAPI 3.1.0](https://www.asyncapi.com/docs/reference/specification/v3.1.0)
specification document from Go code — the *code → spec* direction that most Go
AsyncAPI tooling (which goes *spec → code*) leaves unserved.

`asyncgo` is a **documentation generator**, not a messaging framework. It does
not route your actual messaging; it derives a committed `asyncapi.yaml` from two
touchpoints in your code.

## How it works

### 1. The struct (data contract)

Message payloads are derived from your Go structs via reflection. `json` tags
drive field names; `asyncapi` tags carry `required`, `enum`, `example`, `format`,
and the `allOf`/`oneOf`/`anyOf` composition directives. Field descriptions are
read from the field's doc comment.

```go
type OrderPlaced struct {
 OrderID string  `json:"order_id" asyncapi:"required"`
 Amount  float64 `json:"amount"   asyncapi:"required"`
 // Optional note from the customer.
 Note    string  `json:"note"`
}
```

### 2. The catalog (topology)

Channels and operations are declared once in a typed, compiler-checked catalog:

```go
var Catalog = asyncgo.Spec(
 asyncgo.Info("Orders Service", "1.0.0"),
 asyncgo.Servers(asyncgo.Server("prod", "kafka").Host("broker:9092")),
 asyncgo.Channels(
  asyncgo.Channel("order-placed").
   Send(asyncgo.Operation().
    Message(asyncgo.MessageOf(OrderPlaced{}).Name("OrderPlaced"))).
   Kafka(spec.KafkaChannelBinding{Topic: "order-placed"}),
 ),
)
```

`MessageOf` *references* your struct rather than duplicating the shape, so the
schema cannot drift from the data contract.

### Servers on a channel

A channel is available on all declared servers by default. To restrict it to a
subset, reference the servers (declared via `Servers(...)`) on the channel:

```go
prod := asyncgo.Server("prod", "kafka").Host("broker:9092")

var Catalog = asyncgo.Spec(
 asyncgo.Servers(prod),
 asyncgo.Channels(
  asyncgo.Channel("order-placed").
   Servers(prod).
   Send(asyncgo.Operation().Message(asyncgo.MessageOf(OrderPlaced{}))),
 ),
)
```

```yaml
# channels/order-placed:
#   servers:
#     - $ref: '#/servers/prod'
```

### Generate & check

```bash
# write asyncapi.yaml (committed artifact)
asyncgo generate ./...

# fail CI when asyncapi.yaml is out of date
asyncgo check ./...
```

The generator discovers catalogs **reachable from `main`**, then runs a small
harness to materialize them (never executing your `main` package).

## Schema derivation rules

- **Fully-qualified names** — hoisted schemas are keyed `pkgPath.TypeName`
  (e.g. `example.com/orders/orders.OrderPlaced`); `$ref` escapes `/` per JSON
  Pointer.
- **Optional by default** — a field is required only with `asyncapi:"required"`.
- **Descriptions from comments** — a field's `description` is read from its doc
  comment; `enum`, `example`, and `format` still come from the `asyncapi` tag.
- **Always hoist** named struct types into `components.schemas`; only anonymous
  inline types are inlined.
- **Embedding** — embedded structs are flattened by default (matching
  `encoding/json`); tag one with `asyncapi:"allOf"` to compose it instead.
- **Union fields** — `asyncapi:"oneOf=A|B"`, `anyOf=A|B`, or `allOf=A|B` on a
  field emits `$ref`s to the named types, which are hoisted automatically.

## Schema composition

Go struct composition maps onto the JSON Schema combinators `allOf`, `oneOf`,
and `anyOf` — no hand-written schema required.

### allOf from embedding

Anonymous embedded fields are **flattened** by default (matching
`encoding/json`). Tag an embedded field with `asyncapi:"allOf"` to keep it as a
shared `$ref` and compose it via `allOf` instead. `required` stays local to each
`allOf` member, per JSON Schema semantics.

```go
type Base struct {
 ID string `json:"id" asyncapi:"required"`
}

type OrderPlaced struct {
 Base   `asyncapi:"allOf"`
 Amount float64 `json:"amount" asyncapi:"required"`
}
```

```yaml
# components.schemas (abridged; "..." elides the fully-qualified name):
#   ...Base:
#     type: object
#     properties: { id: { type: string } }
#     required: [id]
#   ...OrderPlaced:
#     type: object
#     allOf:
#       - $ref: "#/components/schemas/...Base"
#       - type: object
#         properties: { amount: { type: number } }
#         required: [amount]
```

### oneOf / anyOf / allOf from a tag

On a field, `oneOf=`, `anyOf=`, and `allOf=` emit the corresponding combinator of
`$ref`s. Names may be same-package short names or fully-qualified
`pkgPath.TypeName`. The generator discovers the referenced types and hoists them
into `components.schemas` automatically — zero registration boilerplate.

```go
type OrderCancelled struct {
 OrderID string `json:"order_id" asyncapi:"required"`
}

type OrderEvent struct {
 Data any `json:"data" asyncapi:"required,oneOf=OrderPlaced|OrderCancelled"`
}
```

```yaml
# data:
#   oneOf:
#     - $ref: "#/components/schemas/...OrderPlaced"
#     - $ref: "#/components/schemas/...OrderCancelled"
```

AsyncAPI 3.x Schema Objects are a superset of **JSON Schema Draft 07**, where
`$ref` siblings are ignored — so a union field should be typed `any` (or an
interface), not a concrete type.

## Layout

| Path | Purpose |
| --- | --- |
| `spec/` | Typed AsyncAPI 3.1.0 model + codecs + Kafka/AMQP/NATS/MQTT bindings |
| `schema/` | `struct → JSON Schema` reflection |
| `cmd/asyncgo/` | `generate` / `check` CLI |
| `internal/discovery/` | catalog discovery + materialization |
| `examples/simple/` | runnable example (its own module) |
| `examples/allof/` | embedded struct composed via `allOf` (its own module) |
| `examples/oneof/` | `oneOf` union field (its own module) |
| `examples/anyof/` | `anyOf` union field (its own module) |
