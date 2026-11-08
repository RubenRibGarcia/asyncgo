# Features

What the catalog can express beyond channels and messages: security schemes,
request/reply, traits, and multi-format payloads.

## Security schemes

Declare a scheme once with `SecurityScheme(...)`, register it with
`SecuritySchemes(...)`, and reference it from a server or an operation. The
reference is emitted as a `$ref` into `components.securitySchemes`:

```go
oauth := asyncgo.SecurityScheme("oauth", spec.SecurityScheme{
	Type: "oauth2",
	Flows: &spec.OAuthFlows{
		ClientCredentials: &spec.OAuthFlow{
			TokenURL:        "https://auth.example.com/oauth/token",
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
declared the same way.

`Server.Security(...)` and `Operation.Security(...)` accept the same scheme
builders, and a reference to a scheme that was never registered is a catalog
validation error.

## Request/reply

Declare a reply address and a reply once, register them with `ReplyAddresses(...)`
and `Replies(...)`, and attach one to an operation with `Operation.Reply(...)`.
The reply is declared in one place and referenced from the operation, so both
sides stay a `$ref`:

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
`Reply.Channel(...)`, and the messages it carries with `Reply.Message(channel, ...)`.
The channel argument is what the message `$ref`s are built from, so the two calls
do not depend on each other's order.

!!! warning "Address and channel are mutually exclusive"

    The specification forbids combining them: when a reply declares an address,
    the channel it names must have no address — so `Address(...)` and
    `Channel(...)` on the same reply is a catalog validation error rather than a
    valid document.

## Traits

Declare a trait once with `OperationTrait(...)` or `MessageTrait(...)`, register it
with `OperationTraits(...)` / `MessageTraits(...)`, and reference it from any
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
`correlationId`, `contentType`, `name`, `title`, `summary`, `description`, `tags`,
`externalDocs`, `bindings`, `examples`. `action`, `channel`, `messages`, `payload`,
and a nested `traits` list are deliberately not settable: the specification
excludes them, and modeling each trait as its own type makes them
unrepresentable.

### Correlation IDs

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
`components.correlationIds` under `<name>CorrelationID` — the message's `Name`, or
the payload type's name when unset — and emits a `$ref`, so the id stays reusable.
`MessageTrait` has the same `CorrelationIDFrom(...)`, keyed
`<traitName>CorrelationID`. Authoring the same value twice under the same key
reuses the component; a different value at the same key is a duplicate-name error.

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

!!! note "Merging is the consumer's job"

    The specification's Traits Merge Mechanism — a JSON Merge Patch applied in
    declaration order, where a trait must not override the target's own property
    — is the *consumer's* job, not asyncgo's: the generator emits the `traits`
    `$ref` list and leaves merging to the tool reading the document.

## Multi-format schemas (Avro, Protobuf)

`MessageOf` derives a payload schema from a Go type. When the payload is not JSON
Schema — Avro and Protobuf are the norm for Kafka — there is no Go type to derive
from, so declare the schema and attach it with `MessageFrom`:

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

### Two things to know

- **Inside the body, map keys are emitted sorted.** The discovery harness
  round-trips every document through YAML, so a record declared
  `type, name, fields` is emitted `fields, name, type`. Avro JSON readers are
  order-insensitive — this is cosmetic, not data loss.
- **Declare it once.** An inline multi-format payload is emitted in full on every
  message that uses it; `Schemas(...)` plus `spec.Ref(...)` shares one
  declaration. A declared name must not collide with a hoisted Go type's
  fully-qualified name — that is a catalog validation error.

### Reference-validator gaps

The pinned `asyncapi validate` used by the integration test checks Avro, OpenAPI
3.0.0, and RAML 1.0 multi-format schemas when the body is a mapping. It resolves a
string-valued `schema` as a reference — so a string body fails for every format —
and it has no Protobuf parser at all, rejecting every Protobuf body with an empty
error list.

asyncgo emits all of them correctly; the reference validator just cannot check
Protobuf. The `test/data/multiformat` fixture records that exclusion explicitly.
