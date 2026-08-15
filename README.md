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
drive field names; `asyncapi` tags carry `required`, `enum`, `example`, and
`format`. Field descriptions are read from the field's doc comment.

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

## Layout

| Path | Purpose |
| --- | --- |
| `spec/` | Typed AsyncAPI 3.1.0 model + codecs + Kafka/AMQP/NATS/MQTT bindings |
| `schema/` | `struct → JSON Schema` reflection |
| `cmd/asyncgo/` | `generate` / `check` CLI |
| `internal/discovery/` | catalog discovery + materialization |
| `examples/simple/` | runnable example (its own module) |
| `examples/embedded/` | example with embedded structs (its own module) |
