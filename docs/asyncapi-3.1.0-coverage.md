# AsyncAPI 3.1.0 coverage

A point-in-time assessment of how much of the
[AsyncAPI 3.1.0 specification](https://www.asyncapi.com/docs/reference/specification/v3.1.0)
`asyncgo` can model, express through its fluent DSL, and emit.

It answers one question: **given a document a user needs to write, can this
library produce it?**

| | |
| --- | --- |
| **Assessed** | 2026-09-12 |
| **Revision** | `master` @ `d9f9dbd` |
| **Method** | Field-by-field diff of `spec/`, `schema/`, and the root DSL package against the normative spec text at `github.com/asyncapi/spec@v3.1.0` (`spec/asyncapi.md`) |
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
| Server Object | 🟡 8/12 | 🟡 | no `pathname`, `title`, `summary`, `externalDocs` |
| Server Variable Object | ✅ 4/4 | ✅ | — |
| Channels Object | ✅ | ✅ | — |
| Channel Object | 🟡 8/10 | 🟡 | no `summary`, `externalDocs`; `parameters` is 🟠; `address` cannot be `null` |
| Messages Object | ✅ | ✅ | key is the message `name`; collisions overwrite silently |
| Operations Object | ✅ | 🟡 | key is auto-derived `${address}.${action}`; a second `Send`/`Receive` on one channel overwrites |
| Operation Object | 🟡 11/12 | 🟡 | **no `reply`**; `security`/`tags`/`externalDocs`/`traits` are 🟠 |
| Operation Trait Object | ❌ | ❌ | only a `[]*Reference` slot exists on Operation |
| **Operation Reply Object** | ❌ | ❌ | request/reply pattern unsupported |
| **Operation Reply Address Object** | ❌ | ❌ | " |
| Message Object | ✅ 13/13 | 🟡 | `tags`, `externalDocs`, `traits` are 🟠; `correlationId` is 🟠 |
| Message Trait Object | ❌ | ❌ | only a `[]*Reference` slot exists on Message |
| Message Example Object | ✅ 4/4 | 🟡 | `Example()` sets `name` + `payload` only |
| Tag Object | ✅ 3/3 | 🟡 | settable only via `Info().Tags(...)` |
| External Documentation Object | ✅ | ❌ | no builder anywhere |
| Components Object | 🟡 **7/19** | 🟡 | only `schemas` is ever written; no builder at all ([§2](#2-components-object)) |
| Reference Object | ✅ (`$ref` only) | 🟡 | correct shape for 3.1.0; internal use only |
| **Multi Format Schema Object** | ❌ | ❌ | Avro / Protobuf / `schemaFormat` unsupported |
| Schema Object | 🟡 Draft-07 subset | ✅ | see [§4](#4-schema-derivation) |
| Security Scheme Object | ❌ | ❌ | no way to document auth |
| OAuth Flows Object | ❌ | ❌ | " |
| OAuth Flow Object | ❌ | ❌ | " |
| Server Bindings Object | ✅ map | 🟡 | 4 of 20 protocols ([§3](#3-bindings-protocols)) |
| Parameters Object | ✅ | ❌ | 🟠 — modeled, no builder |
| Parameter Object | 🟡 **wrong shape** | ❌ | models `schema`, which 3.1.0 removed; missing `enum`, `default`, `examples` |
| Channel Bindings Object | ✅ map | 🟡 | 4 of 20 protocols |
| Operation Bindings Object | ✅ map | 🟡 | 4 of 20 protocols |
| Message Bindings Object | ✅ map | 🟡 | 4 of 20 protocols |
| Correlation ID Object | ✅ 2/2 | ❌ | `Message.CorrelationID` is `*Reference` only — no builder, no inline form |
| Replies / Reply Addresses (components) | ❌ | ❌ | — |

## 2. Components Object

3.1.0 defines 19 fields. The library models 7 and populates 1.

| Field | Modeled | Populated |
| --- | :--: | :--: |
| `schemas` | ✅ | ✅ (the only one) |
| `servers` | ✅ | ❌ |
| `channels` | ✅ | ❌ |
| `operations` | ✅ | ❌ |
| `messages` | ✅ | ❌ |
| `parameters` | ✅ | ❌ |
| `correlationIds` | ✅ | ❌ |
| `securitySchemes` | ❌ | ❌ |
| `serverVariables` | ❌ | ❌ |
| `replies` | ❌ | ❌ |
| `replyAddresses` | ❌ | ❌ |
| `externalDocs` | ❌ | ❌ |
| `tags` | ❌ | ❌ |
| `operationTraits` | ❌ | ❌ |
| `messageTraits` | ❌ | ❌ |
| `serverBindings` | ❌ | ❌ |
| `channelBindings` | ❌ | ❌ |
| `operationBindings` | ❌ | ❌ |
| `messageBindings` | ❌ | ❌ |

The six modeled-but-unpopulated maps exist only so `Merge()` can union them
across catalogs (`internal/discovery/merge.go`); nothing in the DSL or the
generator ever writes them. `components.schemas` is filled by
`builder.components()` (`doc.go`) and `schema.Finalize` (`schema/registry.go`).

## 3. Bindings protocols

3.1.0 defines 20 protocol keys. The library ships typed structs for 4.

| Protocol | Server | Channel | Operation | Message | Notes |
| --- | :--: | :--: | :--: | :--: | --- |
| `kafka` | ✅ | ✅ | ✅ | 🟡 | message binding missing `schemaIdPayloadEncoding`, `schemaLookupStrategy` |
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
| `$ref` with sibling keywords on one node | ❌ `Schema.Ref` is a plain string field, so a reference cannot carry siblings |
| non-JSON-Schema formats (Avro, Protobuf, OpenAPI) | ❌ no Multi Format Schema Object |

## 5. Tooling and pipeline

| Capability | Status | Where |
| --- | --- | --- |
| Catalog discovery (`*asyncgo.SpecResult` vars reachable from `main`) | ✅ | `internal/discovery/discover.go` (`go/packages`) |
| Field descriptions from AST doc comments | ✅ | `internal/discovery/descriptions.go` |
| Harness generation + `go run` materialization | ✅ | `internal/discovery/materialize.go` |
| YAML round-trip to preserve integer binding values | ✅ | `internal/discovery/materialize.go` |
| Multi-catalog merge | ✅ | `internal/discovery/merge.go` |
| YAML output, default `asyncapi.yaml` | ✅ | `spec/encode.go` (`goccy/go-yaml`) |
| JSON output | 🟡 | `spec.AsyncAPI.JSON()` exists; no CLI flag |
| `asyncgo generate [dir] [-o file\|dir/]` | ✅ | `internal/cli/generate.go` |
| `asyncgo check [dir]` byte-equality drift gate | ✅ | `internal/cli/check.go` |
| `asyncgo version`, `--version`, shell completion | ✅ | `internal/cli/root.go`, Cobra |
| Validate output against the official AsyncAPI 3.1.0 JSON Schema | ❌ | — |
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

## 8. Backlog

Each item is issue-shaped: gap, spec reference, affected area (using the labels
from `.github/ISSUE_TEMPLATE/feature_request.yml`), and acceptance criteria.

Priority reflects impact on *being able to write a production document*, not
implementation effort.

### P0 — blocks classes of documents

<a id="b1"></a>

#### B1 — Security schemes

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
- **Acceptance** — at minimum `http`, `ws`, `mqtt5`, `amqp1` typed (the four a Go
  service most plausibly uses, and the two that shadow already-typed protocols);
  the escape hatch documented for the rest.

<a id="b13"></a>

#### B13 — Builders for modeled-but-unreachable objects

- **Gap** — `ExternalDocs` has no builder anywhere; `Tag` is settable only on
  `Info`; `Message.CorrelationID` is a `*Reference` with no way to author or
  hoist a `CorrelationID`.
- **Spec** — External Documentation Object, Tag Object, Correlation ID Object,
  `components/correlationIds`.
- **Area** — `dsl (root package: doc.go, message.go, bindings.go)`
- **Acceptance** — `.ExternalDocs(...)` and `.Tags(...)` on server, channel,
  operation, and message; `Message.CorrelationID(...)` accepting an inline
  `spec.CorrelationID` or a `$ref`.

<a id="b14"></a>

#### B14 — Conformance validation against the official schema

- **Gap** — Nothing verifies the generated document is a valid AsyncAPI
  document. `asyncgo check` only compares bytes against the committed artifact,
  so a structurally invalid document is detected as "out of date", not as
  "invalid".
- **Spec** — the published 3.1.0 JSON Schema.
- **Area** — `internal/cli (generate/check)`, `internal/discovery (catalog discovery + materialization)`
- **Acceptance** — `asyncgo validate [dir]` (or `generate --validate`) validating
  the produced document against a pinned copy of the official schema, with a
  failure message citing the JSON Pointer of each violation. The schema must be
  vendored or pinned, not fetched at run time.

<a id="b15"></a>

#### B15 — JSON output flag

- **Gap** — `spec.AsyncAPI.JSON()` exists; the CLI cannot reach it.
- **Area** — `internal/cli (generate/check)`
- **Acceptance** — `--format yaml|json` on `generate` and `check`, defaulting to
  YAML so existing artifacts are unaffected.

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
  | grep -o '<a name="A2S[A-Za-z0-9]*"></a>' | sort -u
```

Every fixed field in the spec is preceded by an `<a name="…">` anchor, so the
anchor set for an object is its authoritative field list. For example, the root
object yields exactly eight (`A2SAsyncAPI`, `A2SId`, `A2SInfo`, `A2SServers`,
`A2SDefaultContentType`, `A2SChannels`, `A2SOperations`, `A2SComponents`) — which
is what makes [B9](#b9--remove-spec-deviations) a finding rather than a guess.

To re-derive the library side, diff the same object families against:

| Concern | File |
| --- | --- |
| Object model | `spec/spec.go`, `spec/schema.go`, `spec/bindings.go` |
| Fluent DSL builders and their setters | `doc.go`, `message.go`, `bindings.go` |
| Reflection derivation | `schema/derive.go`, `schema/tags.go`, `schema/registry.go` |
| Validation | `doc.go` (`apply` methods, `validateServerRefs`) |
| Emitted output | `test/data/*/asyncapi.yaml` golden fixtures |
