# Concepts

`asyncgo` derives a document from two touchpoints in your code: a **struct** that
is the data contract, and a **catalog** that is the topology. Everything else on
this site is detail about what those two can express.

## 1. The struct (data contract)

Message payloads are derived from your Go structs via reflection. `json` tags
drive field names; `asyncapi` tags carry `required`, `enum`, `examples`, `format`,
`contentEncoding`, `contentMediaType`, and the `allOf`/`oneOf`/`anyOf` composition
directives. Field descriptions are read from the field's doc comment.

```go
type OrderPlaced struct {
	OrderID string  `json:"order_id" asyncapi:"required"`
	Amount  float64 `json:"amount"   asyncapi:"required"`
	// Optional note from the customer.
	Note    string `json:"note"`
}
```

The rules that turn a type into a schema — optional-by-default, binary fields,
hoisting, embedding, unions, providers — are on [Schema derivation](schema.md).

## 2. The catalog (topology)

Channels and operations are declared once in a typed, compiler-checked catalog:

```go
var Catalog = asyncgo.Spec(
	asyncgo.Info("Orders Service", "1.0.0"),
	asyncgo.Servers(asyncgo.Server("prod", "kafka", "broker:9092")),
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

## Messages

Every message is hoisted into `components.messages`. The channel's `messages`
map — and every operation or reply that names the message — carries a `$ref`:

```go
var Catalog = asyncgo.Spec(
	asyncgo.Info("Orders Service", "1.0.0"),
	asyncgo.Channels(
		asyncgo.Channel("order-placed").
			Send(asyncgo.Operation().Message(asyncgo.MessageOf(OrderPlaced{}))),
	),
)
```

```yaml
# channels:
#   order-placed:
#     address: order-placed
#     messages:
#       github.com/acme/orders.OrderPlaced:
#         $ref: '#/components/messages/github.com~1acme~1orders.OrderPlaced'
# operations:
#   order-placed.send:
#     messages:
#     - $ref: '#/channels/order-placed/messages/github.com~1acme~1orders.OrderPlaced'
# components:
#   messages:
#     github.com/acme/orders.OrderPlaced:
#       name: OrderPlaced
#       payload:
#         $ref: '#/components/schemas/github.com~1acme~1orders.OrderPlaced'
```

### Message identity

The component key is the message's identity, and it is also the channel's
message id. A name you set — `MessageFrom("Name", ...)` or
`MessageOf(T{}).Name("Name")` — is that key; an unpinned `MessageOf` derives it
from the payload type's fully-qualified name, so `Name(...)` is how you keep the
key short.

A key you set must match `^[\w\d\.\-_]+$` — the pattern the specification applies
to every `components.*` map — while a key derived from the fully-qualified name is
exempt: the package separators it carries cannot match that pattern.

Two channels carrying the same message share one component, while the same key
carrying different content is a catalog validation error. Operations and replies
keep pointing at `#/channels/<id>/messages/<key>` rather than at
`components.messages`: the specification requires their `messages` to be a subset
of the messages defined in the referenced channel. An operation's key is
`<channel>.<action>`, so a second `Send` on one channel is a catalog validation
error rather than a silent overwrite.

### Payload examples

`Message.PayloadExamples(...)` attaches the Draft-07 `examples` array to the
payload schema. Because a payload is a `$ref` and JSON Schema does not apply
keywords beside a `$ref`, the builder writes the examples onto the referenced
`components.schemas` entry, resolving the ref after every item has run so the
component may be declared later. An unresolvable ref is a build error, and two
messages declaring different examples for one shared component is a conflict
rather than an order-dependent merge.

```go
asyncgo.MessageOf(OrderPlaced{}).
	Name("OrderPlaced").
	PayloadExamples(map[string]any{"order_id": "1", "amount": 12.5})
```

## Servers on a channel

A channel is available on all declared servers by default. To restrict it to a
subset, reference the servers (declared via `Servers(...)`) on the channel:

```go
prod := asyncgo.Server("prod", "kafka", "broker:9092")

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

## Channel parameters

An address may carry a Channel Address Expression — `users.{userId}` — and each
expression needs a matching declared parameter. A parameter is a string
substitution into the address, so its allowed values come from `enum`, `default`,
and `examples` — 3.1.0 has no `schema` on a parameter. Declare it on the channel:

```go
asyncgo.Channel("users.{userId}").
	Parameter("userId", spec.Parameter{
		Enum:        []string{"u_1", "u_2"},
		Default:     "u_1",
		Description: "Id of the user.",
		Examples:    []string{"u_1", "u_2"},
		Location:    "$message.payload#/userId",
	})
```

```yaml
# channels:
#   users.{userId}:
#     address: users.{userId}
#     parameters:
#       userId:
#         enum:
#           - u_1
#           - u_2
#         default: u_1
#         description: Id of the user.
#         examples:
#           - u_1
#           - u_2
#         location: $message.payload#/userId
```

Both directions are checked: an expression with no declared parameter, and a
declared parameter the address never uses, are catalog validation errors.

## Next

- **[Schema derivation](schema.md)** — how each Go type becomes a schema.
- **[Protocol bindings](bindings.md)** — the typed binding helper per protocol.
- **[Features](features.md)** — security, request/reply, traits, multi-format.
