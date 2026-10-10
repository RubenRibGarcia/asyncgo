# asyncgo

Generate an [AsyncAPI 3.1.0](https://www.asyncapi.com/docs/reference/specification/v3.1.0)
specification document from Go code — the *code → spec* direction that most Go
AsyncAPI tooling (which goes *spec → code*) leaves unserved.

`asyncgo` is a **documentation generator**, not a messaging framework. It does
not route your actual messaging; it derives a committed AsyncAPI document
(`asyncapi.yaml`, or `asyncapi.json` with `generate --format json`) from two
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

### Messages

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

The component key is the message's identity, and it is also the channel's
message id. A name you set — `MessageFrom("Name", ...)` or
`MessageOf(T{}).Name("Name")` — is that key; an unpinned `MessageOf` derives it
from the payload type's fully-qualified name, so `Name(...)` is how you keep the
key short. Two channels carrying the same message share one component, while the
same key carrying different content is a catalog validation error. Operations
and replies keep pointing at `#/channels/<id>/messages/<key>` rather than at
`components.messages`: the specification requires their `messages` to be a
subset of the messages defined in the referenced channel.

### Servers on a channel

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

### Security schemes

Declare a scheme once with `SecurityScheme(...)`, register it with
`SecuritySchemes(...)`, and reference it from a server or an operation. The
reference is emitted as a `$ref` into `components.securitySchemes`:

```go
oauth := asyncgo.SecurityScheme("oauth", spec.SecurityScheme{
 Type: "oauth2",
 Flows: &spec.OAuthFlows{
  ClientCredentials: &spec.OAuthFlow{
   TokenURL: "https://auth.example.com/oauth/token",
   AvailableScopes: map[string]string{"read:orders": "Read orders"},
  },
 },
})

var Catalog = asyncgo.Spec(
 asyncgo.SecuritySchemes(oauth),
 asyncgo.Servers(
  asyncgo.Server("prod", "kafka", "broker:9092").Security(oauth),
 ),
)
```

```yaml
# servers/prod:
#   security:
#     - $ref: '#/components/securitySchemes/oauth'
```

`spec.SecurityScheme` models every 3.1.0 field — `type`, `description`, `name`,
`in`, `scheme`, `bearerFormat`, `flows`, `openIdConnectUrl`, `scopes` — so the
other scheme types (`userPassword`, `apiKey`, `http`, `openIdConnect`, …) are
declared the same way. `Server.Security(...)` and `Operation.Security(...)`
accept the same scheme builders, and a reference to a scheme that was never
registered is a catalog validation error.

### Request/reply

Declare a reply address and a reply once, register them with
`ReplyAddresses(...)` and `Replies(...)`, and attach one to an operation with
`Operation.Reply(...)`. The reply is declared in one place and referenced from
the operation, so both sides stay a `$ref`:

```go
replyTo := asyncgo.ReplyAddress("ReplyTo").
 Location("$message.header#/replyTo")

orderReply := asyncgo.Reply("OrderReply").Address(replyTo)

var Catalog = asyncgo.Spec(
 asyncgo.ReplyAddresses(replyTo),
 asyncgo.Replies(orderReply),
 asyncgo.Channels(
  asyncgo.Channel("order-placed").
   Send(asyncgo.Operation().
    Reply(orderReply).
    Message(asyncgo.MessageOf(OrderPlaced{}))),
 ),
)
```

```yaml
# components:
#   replies:
#     OrderReply:
#       address:
#         $ref: '#/components/replyAddresses/ReplyTo'
#   replyAddresses:
#     ReplyTo:
#       location: '$message.header#/replyTo'
# operations:
#   order-placed.send:
#     reply:
#       $ref: '#/components/replies/OrderReply'
```

A reply can instead name the channel it is performed in with
`Reply.Channel(...)`, and the messages it carries with
`Reply.Message(channel, ...)`. The channel argument is what the message `$ref`s
are built from, so the two calls do not depend on each other's order. The
specification forbids combining them though: when a reply declares an address,
the channel it names must have no address — so `Address(...)` and `Channel(...)`
on the same reply is a catalog validation error rather than a valid document.

### Traits

Declare a trait once with `OperationTrait(...)` or `MessageTrait(...)`, register
it with `OperationTraits(...)` / `MessageTraits(...)`, and reference it from any
number of operations or messages. The reference is emitted as a `$ref`, so one
declaration is shared:

```go
kafkaOrders := asyncgo.OperationTrait("KafkaOrders").
 Summary("Shared Kafka settings").
 Kafka(spec.KafkaOperationBinding{GroupID: &spec.Schema{Type: "string"}})

var Catalog = asyncgo.Spec(
 asyncgo.OperationTraits(kafkaOrders),
 asyncgo.Channels(
  asyncgo.Channel("order-placed").Send(asyncgo.Operation().
   Traits(kafkaOrders).
   Message(asyncgo.MessageOf(OrderPlaced{}))),
  asyncgo.Channel("order-shipped").Send(asyncgo.Operation().
   Traits(kafkaOrders).
   Message(asyncgo.MessageOf(OrderShipped{}))),
 ),
)
```

```yaml
# components:
#   operationTraits:
#     KafkaOrders:
#       summary: Shared Kafka settings
# operations:
#   order-placed.send:
#     traits:
#       - $ref: '#/components/operationTraits/KafkaOrders'
#   order-shipped.send:
#     traits:
#       - $ref: '#/components/operationTraits/KafkaOrders'
```

`spec.OperationTrait` models the shareable subset of an operation — `title`,
`summary`, `description`, `security`, `tags`, `externalDocs`, `bindings` — and
`spec.MessageTrait` the shareable subset of a message — `headers`,
`correlationId`, `contentType`, `name`, `title`, `summary`, `description`,
`tags`, `externalDocs`, `bindings`, `examples`. `action`, `channel`, `messages`,
`payload`, and a nested `traits` list are deliberately not settable: the
specification excludes them, and modeling each trait as its own type makes them
unrepresentable.

A message trait can point at a Correlation ID declared with `CorrelationID(...)`
and registered with `CorrelationIDs(...)`:

```go
correlationID := asyncgo.CorrelationID("OrderCorrelationID", spec.CorrelationID{
 Location: "$message.header#/correlationId",
})

traced := asyncgo.MessageTrait("TracedMessage").
 ContentType("application/json").
 CorrelationID(correlationID)
```

A message can carry a correlation id too. `CorrelationID(...)` points at a
component declared with `CorrelationID(...)` / `CorrelationIDs(...)`, while
`CorrelationIDFrom(...)` authors the value inline: the builder hoists it into
`components.correlationIds` under `<name>CorrelationID` — the message's `Name`,

or the payload type's name when unset — and emits a `$ref`, so the id stays
reusable. `MessageTrait` has the same `CorrelationIDFrom(...)`, keyed
`<traitName>CorrelationID`. Authoring the same value twice under the same key
reuses the component; a different value at the same key is a duplicate-name
error.

```go
// Points at the component declared above.
asyncgo.MessageOf(OrderPlaced{}).
 Name("OrderPlaced").
 CorrelationID(correlationID)

// Hoists to components.correlationIds.OrderCancelledCorrelationID.
asyncgo.MessageOf(OrderPlaced{}).
 Name("OrderCancelled").
 CorrelationIDFrom(spec.CorrelationID{
  Location: "$message.header#/correlationId",
 })
```

The specification's Traits Merge Mechanism — a JSON Merge Patch applied in
declaration order, where a trait must not override the target's own property —
is the *consumer's* job, not asyncgo's: the generator emits the `traits` `$ref`
list and leaves merging to the tool reading the document.

### Multi format schemas (Avro, Protobuf)

`MessageOf` derives a payload schema from a Go type. When the payload is not
JSON Schema — Avro and Protobuf are the norm for Kafka — there is no Go type to
derive from, so declare the schema and attach it with `MessageFrom`:

```go
var userAvro = spec.MultiFormat("application/vnd.apache.avro;version=1.9.0",
 map[string]any{
  "type": "record",
  "name": "User",
  "fields": []any{
   map[string]any{"name": "displayName", "type": "string"},
   map[string]any{"name": "age", "type": "int"},
  },
 })

var Catalog = asyncgo.Spec(
 asyncgo.Info("Avro Orders Service", "1.0.0"),

 // Declare it once, reference it from as many messages as you like.
 asyncgo.Schemas(asyncgo.Schema("UserAvro", userAvro)),

 asyncgo.Channels(
  asyncgo.Channel("order-placed").Send(asyncgo.Operation().
   Message(asyncgo.MessageFrom("OrderPlaced",
    spec.Ref("#/components/schemas/UserAvro")))),
  asyncgo.Channel("order-shipped").Send(asyncgo.Operation().
   Message(asyncgo.MessageFrom("OrderShipped",
    spec.MultiFormat("application/vnd.google.protobuf;version=3",
     "message OrderShipped { string order_id = 1; }")))),
 ),
)
```

```yaml
# components:
#   schemas:
#     UserAvro:
#       schemaFormat: application/vnd.apache.avro;version=1.9.0
#       schema:
#         type: record
#         name: User
#   messages:
#     OrderShipped:
#       payload:
#         schemaFormat: application/vnd.google.protobuf;version=3
#         schema: message OrderShipped { string order_id = 1; }
# channels:
#   order-shipped:
#     messages:
#       OrderShipped:
#         $ref: '#/components/messages/OrderShipped'
```

`schemaFormat` is emitted verbatim and `schema` is opaque: asyncgo never parses,
validates, or rewrites the body, so any format the specification lists — or a
custom one — round-trips untouched. A multi-format schema is accepted anywhere a
schema is accepted: a message payload or headers, a message trait's headers, and
`components.schemas` (via `Schema(...)` / `Schemas(...)`).

Two things to know:

- **Inside the body, map keys are emitted sorted.** The discovery harness
  round-trips every document through YAML, so a record declared
  `type, name, fields` is emitted `fields, name, type`. Avro JSON readers are
  order-insensitive — this is cosmetic, not data loss.
- **Declare it once.** An inline multi-format payload is emitted in full on every
  message that uses it; `Schemas(...)` plus `spec.Ref(...)` shares one
  declaration. A declared name must not collide with a hoisted Go type's
  fully-qualified name — that is a catalog validation error.

The pinned `asyncapi validate` used by the integration test checks Avro, OpenAPI
3.0.0, and RAML 1.0 multi-format schemas when the body is a mapping. It resolves
a string-valued `schema` as a reference — so a string body fails for every
format — and it has no Protobuf parser at all, rejecting every Protobuf body with
an empty error list. asyncgo emits all of them correctly; the reference validator
just cannot check Protobuf. The `test/data/multiformat` fixture records that
exclusion explicitly.

### Generate & check

```bash
# write asyncapi.yaml (committed artifact)
asyncgo generate .

# write asyncapi.json instead (2-space indented, newline-terminated)
asyncgo generate . --format json

# write to a custom location (a file path, or a directory with a trailing slash)
asyncgo generate . -o ./docs/asyncapi.yaml
asyncgo generate . -o ./docs/ --format json    # -> ./docs/asyncapi.json

# an explicit -o path wins whatever its extension: --format picks the encoding
asyncgo generate . -o ./docs/spec.yaml --format json

# fail CI when asyncapi.yaml is out of date (check verifies asyncapi.yaml only)
asyncgo check .

# print the asyncgo version (matches the git tag / release)
asyncgo version

# built-in help, --version flag, and shell completion (via Cobra)
asyncgo --help
asyncgo --version
asyncgo completion bash
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
- **Custom types** — a named struct implementing `spec.SchemaProvider` overrides
  derivation: its `AsyncAPISchema()` result is hoisted as-is.

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

AsyncAPI 3.1.0 Schema Objects are a superset of **JSON Schema Draft 07**, where
`$ref` siblings are ignored — so a union field should be typed `any` (or an
interface), not a concrete type.

## Custom schema providers

A type with custom (de)serialization — `MarshalJSON`/`UnmarshalJSON`,
`encoding.TextMarshaler`, and the like — changes its wire format, so
reflection-derived derivation no longer matches. Implement
[`spec.SchemaProvider`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo/spec#SchemaProvider)
to declare the wire schema yourself; it is hoisted under the type's
fully-qualified name like any other named struct.

```go
type Money struct {
 Amount   int64
 Currency string
}

// Wire format is a single "12.34 USD" string, not an object.
func (Money) AsyncAPISchema() *spec.Schema {
 return &spec.Schema{
  Type:     "string",
  Pattern:  `^\d+\.\d{2} [A-Z]{3}$`,
  Examples: []any{"12.34 USD"},
 }
}
```

```yaml
# components.schemas (abridged; "..." elides the fully-qualified name):
#   ...Money:
#     type: string
#     examples:
#     - 12.34 USD
#     pattern: "^\d+\.\d{2} [A-Z]{3}$"
```

`Examples` is the Draft-07 keyword, and the only example keyword AsyncAPI 3.1.0
has — its Schema Object defines no singular `example`. `Message.PayloadExamples(…)`
is the same keyword at the message level: when the payload is a `$ref` it writes
the examples onto the referenced `components.schemas` entry, the only place JSON
Schema applies them.

`AsyncAPISchema` is invoked on a zero value, so it must describe the *type*
(not an instance): be pure, deterministic, and panic-free. Return `nil` to fall
back to reflection-derived derivation, or `&spec.Schema{}` for an explicitly
unconstrained schema. Detection honors both value and pointer receivers, and a
custom type referenced from a `oneOf`/`anyOf`/`allOf` field is hoisted
automatically.
