# AsyncAPI 3.1.0 coverage

A point-in-time assessment of how much of the
[AsyncAPI 3.1.0 specification](https://www.asyncapi.com/docs/reference/specification/v3.1.0)
`asyncgo` can model, express through its fluent DSL, and emit.

It answers one question: **given a document a user needs to write, can this
library produce it?**

| | |
| --- | --- |
| **Assessed** | 2026-09-12 |
| **Revision** | `master` @ `d9f9dbd` (assessed) · `master` @ `583d762` (latest revision) |
| **Revised** | 2026-10-04 — `§5 Tooling and pipeline` (validation row), `§8 B12`/`B14`/`B15`, and accuracy fixes in `§1`, `§4.3`, `§9` · 2026-10-04 — `§1`/`§2` security scheme rows, `§7` `SecurityRequirement` deviation, `§8 B1` implemented · 2026-10-05 — `§1`/`§2` traits, tags/externalDocs, and correlation ID rows, `§3` Kafka binding Union types, `§8 B3` implemented / `B13` partially implemented · 2026-10-05 — `§1` Multi Format Schema Object row, `§4.3`, `§8 B4` implemented |
| **Method** | Field-by-field diff of `spec/`, `schema/`, the root DSL package, `internal/cli`, and `internal/discovery` against the normative spec text at `github.com/asyncapi/spec@v3.1.0` (`spec/asyncapi.md`) |
| **Spec source of truth** | <https://github.com/asyncapi/spec/blob/v3.1.0/spec/asyncapi.md> |

This document is an assessment, not a commitment. The backlog in
[§8](#8-backlog) is unordered relative to the repo's design-doc workflow — a
P0 item still needs a design doc before it is built.

## Legend

| Mark | Meaning |
| --- | --- |
| ✅ | Full — modeled, reachable from the DSL, and emitted |
| 🟡 | Partial — some of the object/field set is covered |
| 🟠 | Modeled but unreachable — the struct exists in `spec/`, but no DSL builder can set it, so it can never be emitted |
| ❌ | Absent — not present at all |

"DSL can emit" means there is a builder path from `asyncgo.Spec(...)`. A field
that is only in `spec/*.go` is not reachable by a user writing a catalog.

---

## 1. Object coverage

| AsyncAPI 3.1.0 object | `spec` model | DSL can emit | Gap |
| --- | --- | --- | --- |
| AsyncAPI (root) | ✅ 8/8 +2 extra | 🟡 | `id` unsettable; root `tags`/`externalDocs` modeled but **not in 3.1.0** ([§7](#7-spec-deviations)) |
| Info Object | 🟡 7/8 | 🟡 | no `externalDocs` |
| Contact Object | ✅ 3/3 | ✅ | — |
| License Object | ✅ +`identifier` | ✅ | `identifier` is not a 3.1.0 field ([§7](#7-spec-deviations)) |
| Servers Object | ✅ | ✅ | — |
| Server Object | 🟡 9/12 | 🟡 | no `pathname`, `title`, `summary` |
| Server Variable Object | ✅ 4/4 | ✅ | — |
| Channels Object | ✅ | ✅ | — |
| Channel Object | 🟡 9/10 | 🟡 | no `summary`; `parameters` is 🟠; `address` cannot be `null` |
| Messages Object | ✅ | ✅ | key is the message `name`; collisions overwrite silently |
| Operations Object | ✅ | 🟡 | key is auto-derived `${address}.${action}`; a second `Send`/`Receive` on one channel overwrites |
| Operation Object | ✅ 12/12 | ✅ | — |
| Operation Trait Object | ✅ 7/7 | ✅ | — |
| **Operation Reply Object** | ✅ 3/3 | ✅ | every field is a `$ref`; an inline reply is not expressible |
| **Operation Reply Address Object** | ✅ 2/2 | ✅ | `address` is a `$ref`; an inline address is not expressible |
| Message Object | ✅ 13/13 | 🟡 | `correlationId` is 🟠 |
| Message Trait Object | ✅ 11/11 | ✅ | `correlationId` is a `$ref`; an inline trait is not expressible |
| Message Example Object | ✅ 4/4 | 🟡 | `Example()` sets `name` + `payload` only |
| Tag Object | ✅ 3/3 | ✅ | — |
| External Documentation Object | ✅ | ✅ | — |
| Components Object | 🟡 **12/19** | 🟡 | `schemas`, `securitySchemes`, `replies`, `replyAddresses`, `correlationIds`, `operationTraits` and `messageTraits` are written; no builder for the rest ([§2](#2-components-object)) |
| Reference Object | ✅ (`$ref` only) | 🟡 | correct shape for 3.1.0; internal use only |
| **Multi Format Schema Object** | ✅ | ✅ | — |
| Schema Object | 🟡 Draft-07 subset | ✅ | see [§4](#4-schema-derivation) |
| Security Scheme Object | ✅ 9/9 | ✅ | — |
| OAuth Flows Object | ✅ 4/4 | ✅ | — |
| OAuth Flow Object | ✅ 4/4 | ✅ | — |
| Server Bindings Object | ✅ map | 🟡 | 4 of 20 protocols ([§3](#3-bindings-protocols)) |
| Parameters Object | ✅ | ❌ | 🟠 — modeled, no builder |
| Parameter Object | 🟡 **wrong shape** | ❌ | models `schema`, which 3.1.0 removed; missing `enum`, `default`, `examples` |
| Channel Bindings Object | ✅ map | 🟡 | 4 of 20 protocols |
| Operation Bindings Object | ✅ map | 🟡 | 4 of 20 protocols |
| Message Bindings Object | ✅ map | 🟡 | 4 of 20 protocols |
| Correlation ID Object | ✅ 2/2 | 🟡 | declaration + `$ref` via message traits; `Message.CorrelationID` has no builder and no inline form |
| Replies / Reply Addresses (components) | ✅ | ✅ | — |

## 2. Components Object

3.1.0 defines 19 fields. The library models 12 and populates 7.

| Field | Modeled | Populated |
| --- | :--: | :--: |
| `schemas` | ✅ | ✅ |
| `servers` | ✅ | ❌ |
| `channels` | ✅ | ❌ |
| `operations` | ✅ | ❌ |
| `messages` | ✅ | ❌ |
| `parameters` | ✅ | ❌ |
| `correlationIds` | ✅ | ✅ |
| `securitySchemes` | ✅ | ✅ |
| `serverVariables` | ❌ | ❌ |
| `replies` | ✅ | ✅ |
| `replyAddresses` | ✅ | ✅ |
| `externalDocs` | ❌ | ❌ |
| `tags` | ❌ | ❌ |
| `operationTraits` | ✅ | ✅ |
| `messageTraits` | ✅ | ✅ |
| `serverBindings` | ❌ | ❌ |
| `channelBindings` | ❌ | ❌ |
| `operationBindings` | ❌ | ❌ |
| `messageBindings` | ❌ | ❌ |

The five modeled-but-unpopulated maps exist only so `Merge()` can union them
across catalogs (`internal/discovery/merge.go`); nothing in the DSL or the
generator ever writes them. `components.schemas` is filled by
`builder.components()` (`doc.go`) and `schema.Finalize` (`schema/registry.go`),
`components.securitySchemes` by the `SecuritySchemes(...)` item's `apply`
(`doc.go`), `components.replies` / `components.replyAddresses` by
`Replies(...)` / `ReplyAddresses(...)` (`doc.go`), and
`components.correlationIds` / `components.operationTraits` /
`components.messageTraits` by `CorrelationIDs(...)` / `OperationTraits(...)` /
`MessageTraits(...)` (`doc.go`).

## 3. Bindings protocols

3.1.0 defines 20 protocol keys. The library ships typed structs for 4.

| Protocol | Server | Channel | Operation | Message | Notes |
| --- | :--: | :--: | :--: | :--: | --- |
| `kafka` | ✅ | ✅ | ✅ | 🟡 | message binding missing `schemaIdPayloadEncoding`, `schemaLookupStrategy`; `groupId`/`clientId`/`key` are Union types |
| `amqp` (0-9-1) | ✅ | ✅ | ✅ | ✅ | — |
| `nats` | ✅ | ✅ | ✅ | ✅ | — |
| `mqtt` | ✅ | ✅ | ✅ | ✅ | — |
| `http` | ❌ | ❌ | ❌ | ❌ | `spec.ProtocolHTTP` constant exists, but no binding structs |
| `ws` | ❌ | ❌ | ❌ | ❌ | |
| `amqp1` | ❌ | ❌ | ❌ | ❌ | distinct protocol from `amqp` |
| `mqtt5` | ❌ | ❌ | ❌ | ❌ | distinct protocol from `mqtt` |
| `anypointmq` | ❌ | ❌ | ❌ | ❌ | |
| `jms` | ❌ | ❌ | ❌ | ❌ | |
| `sns` | ❌ | ❌ | ❌ | ❌ | |
| `solace` | ❌ | ❌ | ❌ | ❌ | |
| `sqs` | ❌ | ❌ | ❌ | ❌ | |
| `stomp` | ❌ | ❌ | ❌ | ❌ | |
| `redis` | ❌ | ❌ | ❌ | ❌ | |
| `mercure` | ❌ | ❌ | ❌ | ❌ | |
| `ibmmq` | ❌ | ❌ | ❌ | ❌ | |
| `googlepubsub` | ❌ | ❌ | ❌ | ❌ | |
| `pulsar` | ❌ | ❌ | ❌ | ❌ | |
| `ros2` | ❌ | ❌ | ❌ | ❌ | |

The untyped escape hatch
(`Server/Channel/Operation/Message.Binding(proto string, v any)`, backed by
`spec.*Bindings = map[string]any`) means all 20 *can* be emitted if the user
hand-writes a struct. It is not discoverable, not type-checked, and not
documented.

> **Zero-value hazard.** Every binding field carries `omitempty`, so meaningful
> zero values are silently dropped: `MQTTOperationBinding{QoS: 0}` (a valid QoS),
> `AMQPOperationBinding{Mandatory: false}`, `KafkaChannelBinding{Partitions: 0}`,
> and every `bool` flag set to `false`. See [B8](#b8--binding-zero-values-are-dropped).
>
> **Kafka Union fields.** `KafkaOperationBinding.GroupID`/`ClientID` are
> Schema Object | Reference Object | boolean, and `KafkaMessageBinding.Key` is
> Schema Object | Reference Object. They are modeled as `any` (the same
> reference-side convention as `Message.CorrelationID`), so a boolean or a
> schema value is expressible. The earlier `string` shape was AsyncAPI 2.x and
> emitted documents the 3.1.0 schema rejects; #14 fixed it.

## 4. Schema derivation

`MessageOf(v)` derives the payload schema from `v`'s Go type. This is the
library's differentiating feature and the area with the deepest coverage.

### 4.1 Type mapping

| Go source | Emitted schema | Status |
| --- | --- | --- |
| `string` | `{type: string}` | ✅ |
| `bool` | `{type: boolean}` | ✅ |
| `int`, `int8`…`int64`, `uint`…`uint64` | `{type: integer}` | ✅ (no `int32`/`int64` format) |
| `float32` / `float64` | `{type: number, format: float}` / `double` | ✅ |
| `[]T`, `[N]T` | `{type: array, items: …}` | ✅ |
| `map[K]V` | `{type: object, additionalProperties: …}` | ✅ (key type ignored) |
| named struct | hoisted to `components.schemas`, referenced by `$ref` | ✅ |
| anonymous struct | inlined `object` | ✅ |
| pointer | dereferenced | ✅ |
| `time.Time` | `{type: string, format: date-time}` | ✅ hardcoded, not overridable |
| `[]byte` | `{type: string, format: byte}` | ⚠️ see [B17](#b17--byte-encoding-format) |
| `json.RawMessage` | `{}` | ✅ |
| `interface`, `chan`, `func`, `complex`, `uintptr` | `{}` | ✅ |
| recursive type | terminates (pre-registered) | ✅ |
| embedded struct | flattened, matching `encoding/json` | ✅ |
| embedded struct, `asyncapi:"allOf"` | `allOf: [$ref, own]` | ✅ |
| named non-struct (`type ID string`) | inlined; not hoisted, no provider hook | 🟡 |

### 4.2 Field rules

| Source | Emitted | Status |
| --- | --- | --- |
| `json:"name"` | property name | ✅ |
| `json:"-"`, unexported field | skipped | ✅ |
| field doc comment | `description` | ✅ static AST pass (`internal/discovery/descriptions.go`) |
| `asyncapi:"required"` | `required: [...]` | ✅ |
| `asyncapi:"enum=a\|b"` | `enum` (strings only) | 🟡 no typed / `iota` enum detection |
| `asyncapi:"example=…"` | `example` | ✅ |
| `asyncapi:"format=…"` | `format` | ✅ |
| `asyncapi:"oneOf=A\|B"` / `anyOf` / `allOf` | `$ref` list, members auto-hoisted | ✅ |
| `spec.SchemaProvider` | full schema override, hoisted as-is | ✅ struct types only |

### 4.3 Not expressible

| Capability | Status |
| --- | --- |
| `default` from Go | ❌ only via `SchemaProvider` |
| `minLength`, `maxLength`, `pattern` from Go | ❌ only via `SchemaProvider` |
| `minItems`, `maxItems`, `uniqueItems` | ❌ not even in `spec.Schema` |
| `minProperties`, `maxProperties` | ❌ not in `spec.Schema` |
| validation-tag bridging (`validate:"required,min=1"`, `jsonschema:"…"`) | ❌ |
| `discriminator`, `externalDocs`, `deprecated` (3.1.0 Schema keywords) | ❌ not in `spec.Schema` |
| `$id`, `$schema`, `$comment`, `const`, `if`/`then`/`else`, `contains`, `propertyNames`, `patternProperties`, `dependencies`, `readOnly`, `writeOnly` | ❌ not in `spec.Schema` |
| `$ref` with sibling keywords on one node | ❌ `Schema` carries `Ref` alongside sibling fields, so the pair does serialize — but siblings are a no-op under JSON Reference, and `spec.Ref()` sets none |
| non-JSON-Schema formats from Go (Avro, Protobuf, OpenAPI) | ❌ not derivable; declare the body by hand with `spec.MultiFormat` ([§8 B4](#b4)) |

## 5. Tooling and pipeline

| Capability | Status | Where |
| --- | --- | --- |
| Catalog discovery (`*asyncgo.SpecResult` vars reachable from `main`) | ✅ | `internal/discovery/discover.go` (`go/packages`) |
| Field descriptions from AST doc comments | ✅ | `internal/discovery/descriptions.go` |
| Harness generation + `go run` materialization | ✅ | `internal/discovery/materialize.go` |
| YAML round-trip to preserve integer binding values | ✅ | `internal/discovery/materialize.go` |
| Multi-catalog merge | ✅ | `internal/discovery/merge.go` |
| YAML output, default `asyncapi.yaml` | ✅ | `spec/encode.go` (`goccy/go-yaml`) |
| JSON output, `--format json` (default `asyncapi.json`) | ✅ | `spec/encode.go` (`JSONIndent`), `internal/cli/generate.go` |
| `asyncgo generate [dir] [-o file\|dir/] [--format yaml\|json]` | ✅ | `internal/cli/generate.go` |
| `asyncgo check [dir]` byte-equality drift gate (YAML only) | ✅ | `internal/cli/check.go` |
| `asyncgo version`, `--version`, shell completion | ✅ | `internal/cli/root.go`, Cobra |
| Validate output against the official AsyncAPI 3.1.0 JSON Schema | 🟡 | `test/integration/asyncgo_generate_test.go` — test-time only: the pinned `asyncapi/cli` accepts both the YAML and JSON encodings of every fixture; there is no `asyncgo validate` command |
| Bundle external / multi-file `$ref` | ❌ | — |
| Serve / preview (e.g. AsyncAPI Studio) | ❌ | — |

## 6. Validation

Checks that run at catalog build time (`SpecResult.ValidationErrors()`) or as a
post-pass, and surface as a per-catalog `discovery.CatalogErrors` report.

| Rule | Status |
| --- | --- |
| `info.title`, `info.version` required | ✅ |
| `server.{name,protocol,host}` required | ✅ |
| duplicate server name | ✅ |
| duplicate channel address | ✅ |
| channel `servers` `$ref` resolves to a declared server | ✅ post-pass in `builder.validateServerRefs` |
| `MessageOf(nil)` guarded | ✅ |
| key pattern `^[A-Za-z0-9_\-]+$` (server, channel, message, component keys) | ❌ |
| `operation.messages` ⊆ channel `messages` | ❌ |
| channel address expressions `{p}` ↔ declared `parameters` | ❌ |
| operation / message map-key collision (silent overwrite today) | ❌ |
| required-field presence per object, beyond the checks above | ❌ |
| spec-conformance of the final document (schema validation) | ❌ |

## 7. Spec deviations

Fields the model carries that AsyncAPI 3.1.0 does not define. Harmless today
because no builder sets them, but they would emit non-conformant output if
populated.

| Field | Reality |
| --- | --- |
| `AsyncAPI.Tags` | The 3.1.0 root object has exactly 8 fields: `asyncapi`, `id`, `info`, `servers`, `defaultContentType`, `channels`, `operations`, `components`. There is no root `tags` or `externalDocs`. |
| `AsyncAPI.ExternalDocs` | " |
| `License.Identifier` | Not a 3.1.0 field; 3.1.0 `License` is `name` + `url` (this is an OpenAPI 3.1 field). |
| `Parameter.Schema` | 3.1.0 `Parameter` is `enum`, `default`, `description`, `examples`, `location` (this is a 2.x shape). |

The 3.1.0 `security` field is `[[Security Scheme Object | Reference Object]]` —
an array of schemes or `$ref`s, and 3.1.0 defines no Security Requirement
Object. The original assessment carried `Server.Security`/`Operation.Security`
as `[]SecurityRequirement` (a `map[string][]string`, the 2.x shape) without
listing it here; that type was replaced by `[]*Reference` in
[B1](#b1--security-schemes).

## 8. Backlog

Each item is issue-shaped: gap, spec reference, affected area (using the labels
from `.github/ISSUE_TEMPLATE/feature_request.yml`), and acceptance criteria.

Priority reflects impact on *being able to write a production document*, not
implementation effort.

### P0 — blocks classes of documents

<a id="b1"></a>

#### B1 — Security schemes

- **Implemented** — [#12](https://github.com/RubenRibGarcia/asyncgo/issues/12):
  `spec.SecurityScheme` + `OAuthFlows`/`OAuthFlow`, `Components.SecuritySchemes`,
  `SecurityScheme(...)`/`SecuritySchemes(...)`,
  `Server.Security(...)`/`Operation.Security(...)`, a `validateSecurityRefs`
  post-pass, and one `test/data/security` golden fixture covering all five
  scheme types. The 2.x `SecurityRequirement` type is gone
  ([§7](#7-spec-deviations)).
- **Gap** — No `SecurityScheme`, `OAuthFlows`, or `OAuthFlow` object.
  `Server.Security` and `Operation.Security` are `[]SecurityRequirement` (a
  scheme-name → scopes map), so security can be *referenced* but never
  *declared*. A production document cannot describe authentication.
- **Spec** — Security Scheme Object, OAuth Flows Object, OAuth Flow Object,
  `components/securitySchemes`, `Server.security`, `Operation.security`.
- **Area** — `spec (object model, codecs, bindings)`, `dsl (root package: doc.go, message.go, bindings.go)`
- **Acceptance** — `spec.SecurityScheme` (+ `OAuthFlows`/`OAuthFlow`) and
  `Components.SecuritySchemes`; DSL builders on `Server` and `Operation` plus a
  top-level `SecurityScheme(...)` item; emitted `components.securitySchemes`
  whose `$ref`s from server/operation `security` resolve; golden fixture per
  scheme type (`userPassword`, `apiKey`, `http`, `oauth2`, `openIdConnect`).

<a id="b2"></a>

#### B2 — Request/reply (Operation Reply)

- **Implemented** — [#13](https://github.com/RubenRibGarcia/asyncgo/issues/13):
  `spec.OperationReply` + `OperationReplyAddress`, `Components.Replies` /
  `ReplyAddresses`, `Reply(...)`/`Replies(...)` and
  `ReplyAddress(...)`/`ReplyAddresses(...)`, `Operation.Reply(...)` plus
  `reply.Address(...)`/`Channel(...)`/`Message(...)`, a `validateReplyRefs`
  post-pass, and one `test/data/reply` golden fixture covering a
  `$message.header#/replyTo` address and a channel + messages reply. Every field
  is emitted as a `$ref`: the specification types `reply` and `reply.address` as
  `X | Reference Object`, and the model carries only the reference side, so an
  inline reply or reply address is not expressible.
- **Gap** — `Operation.reply` is not modeled. `OperationReply` and
  `OperationReplyAddress` do not exist, nor do `components.replies` /
  `components.replyAddresses`. Request/reply channels cannot be documented.
- **Spec** — Operation Reply Object, Operation Reply Address Object,
  `Operation.reply`, `components/replies`, `components/replyAddresses`.
- **Area** — `spec (object model, codecs, bindings)`, `dsl (root package: doc.go, message.go, bindings.go)`
- **Acceptance** — `Operation.Reply(...)` builder emitting
  `reply.address.location` (runtime expression), optional `reply.channel` and
  `reply.messages` `$ref`s; the two component maps added; golden fixture showing
  a `$message.header#/replyTo` address.

<a id="b3"></a>

#### B3 — Operation and message traits

- **Gap** — `Operation.Traits` and `Message.Traits` are `[]*Reference`, but the
  trait objects are not modeled and nothing ever writes
  `components.operationTraits` / `components.messageTraits`. The slot exists and
  is unusable.
- **Spec** — Operation Trait Object, Message Trait Object,
  `components/operationTraits`, `components/messageTraits`, traits merge
  mechanism.
- **Area** — `spec (object model, codecs, bindings)`, `dsl (root package: doc.go, message.go, bindings.go)`
- **Acceptance** — trait objects modeled with their fixed-field sets; builders to
  declare and reference them; `$ref`s resolve into the components maps; a golden
  fixture demonstrating a shared trait applied to two operations.
- **Implemented** — #14: `spec.OperationTrait` / `spec.MessageTrait`,
  `components.operationTraits` / `components.messageTraits`, the
  `OperationTraits(...)` / `MessageTraits(...)` declaration items with fluent
  setters, `Operation.Traits(...)` / `Message.Traits(...)` refs, and the
  `validateTraitRefs` post-pass (plus an operation-trait security arm in
  `validateSecurityRefs`). The traits merge mechanism's JSON Merge Patch is left
  to consumers: `$ref`s are emitted, not merged. The `test/data/traits` golden
  applies one operation trait to two operations.

<a id="b4"></a>

#### B4 — Multi Format Schema Object

- **Gap** — Payload and headers are `*spec.Schema` only. There is no
  `schemaFormat`, so Avro and Protobuf payloads — the norm for Kafka, the
  library's flagship protocol — cannot be described.
- **Spec** — Multi Format Schema Object (`schemaFormat`, `schema`),
  `Message.headers`, `Message.payload`, `components/schemas`.
- **Area** — `spec (object model, codecs, bindings)`, `schema (struct -> JSON Schema)`
- **Acceptance** — a multi-format schema type accepted anywhere a schema is
  accepted today; `schemaFormat` emitted verbatim with an opaque `schema` body;
  either a `MessageFrom...` counterpart to `MessageOf` or a documented
  hand-authored path; golden fixture with an Avro payload.
- **Implemented** — #15: `spec.Schema` gained `SchemaFormat` and `Schema`, so the
  same struct models both a Schema Object and a Multi Format Schema Object and is
  accepted at every schema location — `Message.payload`/`headers`,
  `MessageTrait.headers`, and `components.schemas`. `spec.MultiFormat(format,
  body)` emits `schemaFormat` verbatim with the body opaque, and
  `validateSchemaNodes` rejects a node that mixes the two field sets or sets only
  one. The DSL adds `MessageFrom(name, schema)` as the type-free counterpart to
  `MessageOf`, plus `Schema(name, schema)` / `Schemas(...)` for reusable
  components. The `test/data/avro` golden declares one Avro record `$ref`'d from
  two messages, an inline Avro payload, and multi-format headers. The pinned
  `asyncapi/cli` validates Avro multi-format schemas but registers no parser for
  the other formats, so Protobuf emission is covered by `spec/encode_test.go`
  rather than the golden.

### P1 — correctness and spec conformance

<a id="b5"></a>

#### B5 — Complete the Components Object

- **Gap** — 7 of 19 fields modeled, 1 populated, no builder. Six modeled maps are
  dead output paths.
- **Spec** — Components Object (all 19 fixed fields).
- **Area** — `spec (object model, codecs, bindings)`
- **Acceptance** — all 19 fields present; `mergeComponents` unions all of them;
  the docs state plainly which fields are auto-populated (`schemas`) versus
  user-declared.

<a id="b6"></a>

#### B6 — Correct and expose the Parameter Object

- **Gap** — `spec.Parameter` models `schema`, which 3.1.0 removed, and omits
  `enum`, `default`, `examples`. `Channel.Parameters` has no builder, so channel
  parameters can never be emitted even though address expressions are the
  documented way to describe dynamic channels.
- **Spec** — Parameter Object (`enum`, `default`, `description`, `examples`,
  `location`), Parameters Object, `Channel.address` expressions.
- **Area** — `spec (object model, codecs, bindings)`, `dsl (root package: doc.go, message.go, bindings.go)`
- **Acceptance** — fields aligned to 3.1.0; a `Channel.Parameter(name, …)`
  builder; golden fixture with `address: users.{userId}` and a matching
  parameter.

<a id="b7"></a>

#### B7 — Schema Object: add the 3.1.0 and remaining Draft-07 keywords

- **Gap** — Missing the three AsyncAPI-specific keywords (`discriminator`,
  `externalDocs`, `deprecated`) and most structural Draft-07 keywords
  (`const`, `if`/`then`/`else`, `contains`, `propertyNames`, `patternProperties`,
  `minItems`, `maxItems`, `uniqueItems`, `minProperties`, `maxProperties`,
  `readOnly`, `writeOnly`, `$id`, `$schema`).
- **Spec** — Schema Object.
- **Area** — `spec (object model, codecs, bindings)`
- **Acceptance** — keywords added to `spec.Schema` with JSON/YAML round-trip
  tests; `discriminator` at minimum, since it is the JSON Schema counterpart of
  the library's existing `oneOf` support.

<a id="b8"></a>

#### B8 — Binding zero values are dropped

- **Gap** — `omitempty` on every binding field discards meaningful zero values:
  `MQTTOperationBinding{QoS: 0}` (QoS 0 is a real level),
  `AMQPOperationBinding{Mandatory: false}`, `KafkaChannelBinding{Partitions: 0}`,
  `MQTTServerBinding{CleanSession: false}`.
- **Spec** — the per-protocol binding objects.
- **Area** — `spec (object model, codecs, bindings)`
- **Acceptance** — presence-aware types (pointers or wrappers) for fields where
  zero is semantically distinct; a golden fixture asserting `qos: 0` and
  `mandatory: false` survive serialization. The YAML round-trip in
  `materialize.go` already preserves integer types, so this is a struct-tag
  problem only.

<a id="b9"></a>

#### B9 — Remove spec deviations

- **Gap** — `AsyncAPI.Tags`, `AsyncAPI.ExternalDocs`, and `License.Identifier`
  are modeled but are not 3.1.0 fields.
- **Spec** — AsyncAPI Object (8 fields), License Object (`name`, `url`).
- **Area** — `spec (object model, codecs, bindings)`
- **Acceptance** — removed, or retained behind an explicit, documented extension
  mechanism. Either way the exported surface stops implying they are spec fields.

<a id="b10"></a>

#### B10 — Support a null channel address

- **Gap** — `Channel(address)` errors on an empty address, so a channel whose
  address is only known at runtime cannot be expressed. 3.1.0 explicitly allows
  `address: null` or absent for that case.
- **Spec** — `Channel.address` (`string | null`).
- **Area** — `spec (object model, codecs, bindings)`, `dsl (root package: doc.go, message.go, bindings.go)`
- **Acceptance** — an address-less channel builder emits no `address` key and
  passes validation; golden fixture.

### P2 — breadth and ergonomics

<a id="b11"></a>

#### B11 — Extend DSL validation

- **Gap** — No checks for the `^[A-Za-z0-9_\-]+$` key pattern (servers, channels,
  messages, component keys), `operation.messages ⊆ channel.messages`, address
  expressions versus declared parameters, or operation/message map-key
  collisions (which silently overwrite today).
- **Spec** — patterned-field constraints; `Operation.messages`;
  `Channel.parameters`.
- **Area** — `dsl (root package: doc.go, message.go, bindings.go)`, `internal/discovery (catalog discovery + materialization)`
- **Acceptance** — each rule reports through the existing per-catalog
  `CatalogError` path with a message naming the offending object; fixtures in
  `test/data/invalid`.

<a id="b12"></a>

#### B12 — Typed bindings for the remaining protocols

- **Gap** — 4 of 20 protocols have typed structs; `spec.ProtocolHTTP` exists with
  no structs behind it. Everything else needs hand-written `any` values through
  the untyped escape hatch.
- **Spec** — Server/Channel/Operation/Message Bindings Objects.
- **Area** — `spec (object model, codecs, bindings)`, `docs`
- **Acceptance** — all 20 protocols typed (`kafka`, `amqp`, `nats`, `mqtt`,
  `http`, `ws`, `amqp1`, `mqtt5`, `anypointmq`, `jms`, `sns`, `solace`, `sqs`,
  `stomp`, `redis`, `mercure`, `ibmmq`, `googlepubsub`, `pulsar`, `ros2`),
  covering every binding object each protocol defines; the escape hatch retained
  for protocols added to the specification later.

<a id="b13"></a>

#### B13 — Builders for modeled-but-unreachable objects

- **Gap** — `Message.CorrelationID` is a `*Reference` with no way to author or
  hoist a `CorrelationID` from a plain message. The rest of this item landed with
  #14 (see **Partially implemented**).
- **Spec** — External Documentation Object, Tag Object, Correlation ID Object,
  `components/correlationIds`.
- **Area** — `dsl (root package: doc.go, message.go, bindings.go)`
- **Acceptance** — `Message.CorrelationID(...)` accepting an inline
  `spec.CorrelationID` or a `$ref`.
- **Partially implemented** — #14: `.Tags(...)` / `.ExternalDocs(...)` on server,
  channel, operation, message, and both trait builders; `CorrelationID(...)` /
  `CorrelationIDs(...)` declare hoisted components, and
  `MessageTrait.CorrelationID(...)` references them. Still open:
  `Message.CorrelationID(...)`.

<a id="b14"></a>

#### B14 — Conformance validation against the official schema

- **Gap** — No user-facing way to validate a document. The repository's own
  end-to-end test runs the real `asyncapi validate` over every fixture (see
  [§5](#5-tooling-and-pipeline)), but `asyncgo check` only compares bytes
  against the committed artifact, so a structurally invalid document is
  detected as "out of date", not as "invalid".
- **Spec** — the published 3.1.0 JSON Schema.
- **Area** — `internal/cli (generate/check)`, `internal/discovery (catalog discovery + materialization)`
- **Acceptance** — `asyncgo validate [dir]` (or `generate --validate`) validating
  the produced document against a pinned copy of the official schema, with a
  failure message citing the JSON Pointer of each violation. The schema must be
  vendored or pinned, not fetched at run time.

<a id="b15"></a>

#### B15 — JSON output flag

- **Gap** — `generate --format json` emits `asyncapi.json`, but `check` still
  verifies only `asyncapi.yaml`, so an emitted JSON artifact goes unverified.
- **Area** — `internal/cli (check)`
- **Acceptance** — `check` resolves which format to verify (a `--format` flag
  mirroring `generate`, or auto-discovery of whichever artifact exists),
  defaulting to YAML so existing artifacts are unaffected.

<a id="b16"></a>

#### B16 — Named non-struct types

- **Gap** — `type OrderID string` is inlined at every use site rather than
  hoisted, so it cannot carry a reusable constraint. `asSchemaProvider` checks
  only struct types, so a named scalar or collection cannot override derivation
  either.
- **Spec** — n/a (library design).
- **Area** — `schema (struct -> JSON Schema)`
- **Acceptance** — a named non-struct with a `SchemaProvider` is hoisted under
  its fully-qualified name and referenced by `$ref`. This is already recorded as
  Stage 3 deferred in `docs/designdoc/custom-schema-providers.md`, so the work is
  to promote it, not to design it.

<a id="b17"></a>

#### B17 — `[]byte` encoding format

- **Gap** — `[]byte` derives to `{type: string, format: byte}`. `byte` is not a
  defined JSON Schema format; `encoding/json` marshals `[]byte` as base64, which
  Draft-07 spells `contentEncoding: base64` (optionally `contentMediaType`).
  `contentEncoding` is not in `spec.Schema` today.
- **Spec** — Schema Object (`contentEncoding`, `contentMediaType`).
- **Area** — `schema (struct -> JSON Schema)`, `spec (object model, codecs, bindings)`
- **Acceptance** — `[]byte` derives to `contentEncoding: base64`; golden fixture.
  Low risk, but verify against the AsyncAPI docs examples before changing, since
  `format: byte` appears in some AsyncAPI-adjacent material.

## 9. Re-verifying

The spec tables in this document were read from the normative markdown rather
than the rendered site, because the rendered page collapses the fixed-field
tables. To re-check a claim, fetch the raw spec and grep the field anchors:

```bash
curl -sL https://raw.githubusercontent.com/asyncapi/spec/v3.1.0/spec/asyncapi.md \
  | grep -o '<a name="[A-Za-z0-9]*"></a>' | sort -u
```

Every fixed field in the spec is preceded by an `<a name="…">` anchor, so the
anchor set for an object is its authoritative field list. The anchor naming is
not uniform, so the grep output has to be read by hand:

- The **root object's fields** are `A2S`-prefixed: `A2SAsyncAPI`, `A2SId`,
  `A2SInfo`, `A2SServers`, `A2SDefaultContentType`, `A2SChannels`,
  `A2SOperations`, `A2SComponents` — eight fields, which is what makes
  [B9](#b9--remove-spec-deviations) a finding rather than a guess. The same grep
  also matches section-level anchors (`A2SObject`, `A2SIdString`, …), so its
  output is a superset of those eight.
- **Every other object's fields** are anchored by section and name, e.g.
  `infoObjectExternalDocs`, `serverObjectPathname`, `parameterObjectEnum`,
  `componentsReplies`, `operationReplyObjectAddress`. Filter on the
  `<object>Object` prefix rather than on `A2S`.

To re-derive the library side, diff the same object families against:

| Concern | File |
| --- | --- |
| Object model | `spec/spec.go`, `spec/schema.go`, `spec/bindings.go` |
| Fluent DSL builders and their setters | `doc.go`, `message.go`, `bindings.go` |
| Reflection derivation | `schema/derive.go`, `schema/tags.go`, `schema/registry.go` |
| Validation | `doc.go` (`apply` methods, `validateServerRefs`) |
| Emitted output | `test/data/*/asyncapi.yaml` golden fixtures |
