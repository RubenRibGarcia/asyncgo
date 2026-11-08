# Reference

A curated index of the public API. The authoritative, always-current reference is
generated godoc on
[pkg.go.dev/github.com/RubenRibGarcia/asyncgo](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo);
this page exists to say *which* symbol you want, and to link you straight there.

The same text is available offline:

```bash
go doc github.com/RubenRibGarcia/asyncgo
go doc github.com/RubenRibGarcia/asyncgo/spec KafkaChannelBinding
```

## Document entry points

Top-level functions passed to `asyncgo.Spec(...)`.

| Symbol | Registers |
| --- | --- |
| [`Spec`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo#Spec) | The catalog itself — every other entry point is an argument to it |
| [`Info`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo#Info) | `info`: title, version, description, contact, license |
| [`DefaultContentType`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo#DefaultContentType) | `defaultContentType` |
| [`Servers`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo#Servers) | `servers` — components referenced by channel, operation and security |
| [`Channels`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo#Channels) | `channels` and `operations` |
| [`Schemas`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo#Schemas) | `components.schemas` — hand-declared, e.g. multi-format |
| [`SecuritySchemes`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo#SecuritySchemes) | `components.securitySchemes` |
| [`Replies`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo#Replies) | `components.replies` |
| [`ReplyAddresses`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo#ReplyAddresses) | `components.replyAddresses` |
| [`OperationTraits`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo#OperationTraits) | `components.operationTraits` |
| [`MessageTraits`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo#MessageTraits) | `components.messageTraits` |
| [`CorrelationIDs`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo#CorrelationIDs) | `components.correlationIds` |

## Builders

Each builder returns a handle that the next setter in the chain takes. Most setters
that reference another builder turn it into a JSON Reference immediately, so a
`nil` argument panics at the call site rather than deferring to validation — declare
the referenced item first and pass its builder.

| Symbol | Builds |
| --- | --- |
| [`Channel`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo#Channel) | A channel: address, messages, parameters, servers, bindings |
| [`Operation`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo#Operation) | An operation on a channel: `Send`, `Receive`, reply, traits, security |
| [`MessageOf`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo#MessageOf) | A message whose payload is derived from a Go struct |
| [`MessageFrom`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo#MessageFrom) | A message whose payload you declare, e.g. Avro or Protobuf |
| [`Server`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo#Server) | A server: protocol, host, security, bindings |
| [`SecurityScheme`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo#SecurityScheme) | A security scheme component |
| [`Reply`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo#Reply) | A reply component: address or channel, and messages |
| [`ReplyAddress`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo#ReplyAddress) | A reply-address component |
| [`OperationTrait`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo#OperationTrait) | A shareable subset of an operation |
| [`MessageTrait`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo#MessageTrait) | A shareable subset of a message |
| [`CorrelationID`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo#CorrelationID) | A correlation-id component |
| [`Schema`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo#Schema) | A hand-declared schema component |

## `spec` — the object model

The typed 3.1.0 model the builders write into: every object, every field, plus the
protocol binding structs and the closed vocabularies.

| Symbol | What it is |
| --- | --- |
| [`spec.SchemaProvider`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo/spec#SchemaProvider) | Interface that overrides schema derivation for a type |
| [`spec.Schema`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo/spec#Schema) | The Schema Object, including the Draft-07 keyword set |
| [`spec.MultiFormat`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo/spec#MultiFormat) | A non-JSON-Schema payload: `schemaFormat` plus an opaque body |
| [`spec.Ref`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo/spec#Ref) | A `$ref` to a component |
| [`spec.ReservedBinding`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo/spec#ReservedBinding) | A binding whose field set is not defined — `bindingVersion` only |
| [`spec.Protocols`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo/spec#Protocols) | The 20 protocol keys, as a fresh slice |
| [`spec.KafkaChannelBinding`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo/spec#KafkaChannelBinding) | …and one binding struct per protocol and level |

## `schema` — derivation

| Symbol | What it does |
| --- | --- |
| [`schema.FromType`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo/schema#FromType) | Derive a Schema Object from a `reflect.Type`, registering definitions |
| [`schema.MapSchema`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo/schema#MapSchema) | Declare a mapping for a Go type, keyed by its identity |
| [`schema.MapSchemaByName`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo/schema#MapSchemaByName) | The name-keyed form, which is what `asyncgo.yaml` compiles to |
| [`schema.Finalize`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo/schema#Finalize) | Apply mappings to a document built in process, as the CLI does |

## CLI

See [CLI](../cli.md) for flags, exit codes and verbose logging.
