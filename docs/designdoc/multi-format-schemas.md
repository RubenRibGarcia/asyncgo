# Design: Multi Format Schema Objects (Avro, Protobuf)

- **Status**: Accepted
- **Created**: 2026-10-05
- **Status updated**: 2026-10-05
- **Scope**: `spec/`, `dsl (root: message.go, doc.go)`, `test/data/avro/`, `test/integration/`, `docs/`

## Summary

Let a payload or header name a schema in a non-JSON-Schema format — Avro and
Protobuf first, since Kafka is the library's flagship protocol — by modeling the
AsyncAPI 3.1.0 Multi Format Schema Object. The object model change is two fields
on the existing `spec.Schema` plus a `spec.MultiFormat` constructor, so a
multi-format schema is accepted **anywhere a schema is accepted today** with no
field-type change; the DSL gains `MessageFrom` (a type-free counterpart to
`MessageOf`) and a `Schemas(...)` declaration item so multi-format schemas are
reachable in `components.schemas`.

## Background

### Spec facts

The Multi Format Schema Object represents a schema in a format other than
JSON Schema. Its fixed fields are exactly two, both REQUIRED:

| Field | Type | Note |
| ----- | ---- | ---- |
| `schemaFormat` | `string` | The format identifier, e.g. `application/vnd.apache.avro;version=1.9.0`. |
| `schema` | `any` | The schema body, in that format. |

The spec's *supported schema formats* table recommends Avro 1.9.0, OpenAPI 3.0.0,
RAML 1.0, and Protocol Buffers (`application/vnd.google.protobuf;version=2|3`).

Two spec rules bound the design:

- If `schemaFormat` is **absent from a Schema Object**, it defaults to
  `application/vnd.aai.asyncapi+json;version=3.1.0` — i.e. the object is
  equivalent to a Schema Object. asyncgo therefore keeps omitting `schemaFormat`
  for plain schemas; nothing changes for them.
- A Multi Format Schema Object is a node with **only** those two fields. JSON
  Schema keywords are not siblings of `schemaFormat`; the JSON-Schema flavour is
  the Schema Object, not a multi-format node with keywords.

Locations that accept `Multi Format Schema Object | Schema Object | Reference
Object`, and are therefore in scope here:

| Location | Field |
| -------- | ----- |
| Message Object | `payload`, `headers` |
| Message Trait Object | `headers` |
| Components Object | `schemas` map values |

3.1.0 removed `schema` from the Parameter Object, so parameters are **not** a
multi-format location. That gap is tracked separately (coverage backlog B6).

### Architecture facts that decide the shape

| Fact | Consequence |
| ---- | ----------- |
| `spec.Message.Payload`/`Headers`, `spec.MessageTrait.Headers`, and `spec.Components.Schemas` are all `*spec.Schema`. | Modeling the multi-format node *as* `*spec.Schema` makes it accepted at every location for free — no field-type change anywhere. |
| A catalog is materialized by a generated harness that marshals the document to YAML, and `internal/discovery/materialize.go:63` unmarshals it back into `*spec.AsyncAPI`; the document is re-marshalled from decoded Go values. | Whatever the payload field's Go type is, it must survive **marshal → unmarshal → marshal**. |
| `goccy/go-yaml` marshals `map[string]any` with **sorted keys**, and a struct with fields in **declaration order**. | An `any`-typed node decodes to a map and re-marshals reordered; a struct-typed node decodes back into the struct and round-trips byte-for-byte. |
| `schema.FromType` runs only for `MessageOf`; `schema.Finalize` (`schema/registry.go:35`) walks the Go-type registry of combinator references. | A hand-authored multi-format schema never enters derivation or finalization; no `schema/` code change is needed. |
| Component items follow one shape: `Server(name, …)`, `Channel(address)`, `ReplyAddress(name)` return a builder, paired with a `Servers(...)`/`Channels(...)`/`ReplyAddresses(...)` `Item`. | A `components.schemas` builder should follow the same `Schema(name, …)` + `Schemas(...)` shape. |

### The constraint that rules out the union type

The obvious modeling of a spec union — `Message.Payload any`, or a sealed
interface satisfied by `*Schema` and a new `*MultiFormatSchema` — fails the
harness round-trip. A throwaway program in this repository marshalled a payload,
unmarshalled it back, and re-marshalled it:

```text
typed *Schema (round-trip):   payload: {type, properties, required}   unchanged
any          (round-trip):   payload: {properties, required, type}   REORDERED
```

An `any` payload becomes `map[string]any` on decode, and the re-marshal sorts the
keys. That would rewrite every committed `asyncapi.yaml` and break the
`asyncgo check` byte-equality drift gate. Keeping the node a **struct** is the
only option under which the eight existing goldens stay byte-identical.

## Goals / Non-goals

**Goals**

1. Emit `schemaFormat` verbatim with an opaque `schema` body; never parse,
   validate, or rewrite the body.
2. Accept a multi-format schema at every location listed above with **no
   field-type change** to `Message`, `MessageTrait`, or `Components`.
3. Make `components.schemas` reachable from the DSL, so one Avro schema can be
   declared once and `$ref`'d from several messages.
4. Provide a `MessageFrom...` counterpart to `MessageOf` for payloads with no Go
   type to derive from.
5. Prove it end to end with a committed Avro golden fixture that passes the
   integration test's golden, codec-equivalence, and `asyncapi validate` checks.
6. Leave every existing generated document byte-identical.

**Non-goals (v1)**

- Deriving an Avro/Protobuf schema *from* a Go type. There is no Go type to
  reflect in these formats; the payload is declared, not derived.
- Validating `schemaFormat` against the spec's recommended table, or shipping
  format constants. Values are emitted verbatim.
- Preserving key order inside the opaque body (see Edge cases).
- Bundling external `$ref`s (`schema: {$ref: ./user.avsc}`); asyncgo does not
  bundle external documents today and this design does not add it.
- `Parameter.schema` — 3.1.0 removed it (backlog B6).

## Design decisions

### D1 — Model the Multi Format Schema Object by extending `spec.Schema`

`spec.Schema` gains two `omitempty` fields at the top of the struct —
`SchemaFormat string` and `Schema any` (declared in that order) — plus a
`spec.MultiFormat(format string, body any) *Schema` constructor. `Payload`,
`Headers`, `MessageTrait.Headers`, and `Components.Schemas` keep their current
types.

```go
type Schema struct {
    // Multi Format Schema Object fields. SchemaFormat is the format
    // identifier, emitted verbatim; Schema is its opaque body. Both are
    // REQUIRED together, and neither combines with the JSON Schema keywords
    // below (see D5).
    SchemaFormat string `json:"schemaFormat,omitempty" yaml:"schemaFormat,omitempty"`
    Schema       any    `json:"schema,omitempty"       yaml:"schema,omitempty"`

    Ref string `json:"$ref,omitempty" yaml:"$ref,omitempty"`
    // ...existing keywords, unchanged
}

// MultiFormat returns a Multi Format Schema Object: schemaFormat is emitted
// verbatim and body is carried opaquely.
func MultiFormat(format string, body any) *Schema {
    return &Schema{SchemaFormat: format, Schema: body}
}
```

**Rationale**: it is the smallest change that satisfies goals 2 and 6
simultaneously. Because the node stays a struct, `schemaFormat` and `schema`
serialize in declaration order, the opaque body round-trips, and plain schemas
(the overwhelming majority of documents) emit exactly the bytes they emit today.
Placing the two fields in their own declaration group at the top keeps `gofmt`'s
tag alignment from reflowing the existing keyword groups, and makes them precede
any keyword set in output.

**Rejected — nominal `spec.MultiFormatSchema` + sealed union**: faithful nominal
typing, but `Message.Payload` et al. become an interface. Interface fields are
`any`-like at the marshalling layer in the sense that the decoder has no concrete
target, so the harness round-trip reorders keys (the probe above), and it would
additionally need custom `MarshalJSON`/`MarshalYAML`/`UnmarshalYAML` on four
locations. Breaking public API for no functional gain.

**Rejected — nominal type + `any` fields**: simplest nominal modeling, and
exactly the shape the probe proves breaks every golden.

**Note on the issue's `schema` label**: `schema/` (struct → JSON Schema) needs no
code change — derivation never sees an opaque body, and `Finalize` only walks the
Go-type registry. The `schema` scope is satisfied by documentation of the
hand-authored path.

### D2 — `schemaFormat` is emitted verbatim; `schema` is opaque

No allow-list, no normalization, no table lookup. `spec.MultiFormat` stores both
values as given, and the fields carry no validation beyond D5's structural
invariants.

**Rationale**: the spec permits custom `schemaFormat` values (implementations may
choose not to support them), so a library-level allow-list would be wrong —
asyncgo is a document generator, not a validator. Emitting verbatim also keeps
this change independent of the spec's recommended-formats table, which can evolve
without an asyncgo release.

### D3 — `MessageFrom(name string, schema *spec.Schema) *message`

A type-free constructor alongside `MessageOf`. It sets a new `payload
*spec.Schema` field that `build` consumes; when `payload` is nil, `build` derives
from `m.typ` as today. The payload is emitted **inline and verbatim**, never
hoisted into `defs`.

```go
// MessageFrom declares a message whose payload is the given schema, emitted
// verbatim. Use it for formats that cannot be derived from a Go type — pass
// spec.MultiFormat(...) for Avro or Protobuf, or spec.Ref(...) to point at a
// schema declared with Schema(...)/Schemas(...). name is required: it is the
// channels.*.messages key.
func MessageFrom(name string, schema *spec.Schema) *message {
    return &message{name: name, payload: schema}
}
```

**Rationale**: `MessageOf` requires a Go type and `build` currently errors on a
nil `typ`; Avro has no Go type. An explicit name is mandatory because
`messageName` cannot derive one and the name becomes the message map key.
Keeping the constructor distinct from `MessageOf` means the two payload sources
are visibly separate rather than a silent override — the alternative
(`MessageOf(v).Payload(s)` from a dummy type) lets the type and the payload
disagree.

### D4 — `components.schemas` gets `Schema(name, *spec.Schema)` + `Schemas(...)`

Following the existing component-item shape:

```go
// Schema declares a reusable Schema or Multi Format Schema Object under name in
// components.schemas. Register it with Schemas(...).
func Schema(name string, s *spec.Schema) *schemaDecl

// Schemas adds reusable schemas to components.schemas.
func Schemas(d ...*schemaDecl) Item

type schemasItem []*schemaDecl
```

`apply` errors on an empty name and on a duplicate key, otherwise writes
`b.components().Schemas[d.name] = d.s`.

**Rationale**: the issue's Spec scope explicitly lists `components/schemas`, and
today that map is populated **only** by auto-hoisting (`Spec` ends with
`maps.Copy(c.Schemas, b.defs)`); there is no user-facing builder. Without D4 an
Avro schema is unshareable — it must be repeated inline on every message. This
mirrors `ReplyAddresses`/`MessageTraits`, so it is a familiar surface rather than
a new concept.

### D5 — Reject invalid multi-format nodes at declaration time

D1's trade-off is that invalid states become representable: `SchemaFormat` can be
set beside JSON Schema keywords, or one of the two fields can be set alone. A
`Spec` post-pass, `validateSchemaNodes`, walks the nodes that enter through the
new surfaces and reports through the existing `SpecResult.ValidationErrors` path:

- `schemaFormat` set without `schema`, or `schema` set without `schemaFormat`;
- a multi-format node carrying any JSON Schema keyword;
- a declared component name that collides with an auto-hoisted FQN schema.

**Rationale**: containing the invalid states at the two new entry points keeps
the unified struct honest without inventing a union type (which D1 rejects for
independent reasons). Reporting through the existing per-catalog `CatalogError`
path means no new error mechanism.

## Detailed design

### 1. `spec/schema.go`

Add the two fields and `MultiFormat` from D1. No other `spec` file changes: the
four locations already hold `*spec.Schema`, and `spec/encode.go` needs nothing
because the codecs are struct-driven.

The two fields are placed in their own declaration group — no other field moves
and no existing tag alignment changes — and `SchemaFormat` precedes `Schema` so
output matches the spec's example ordering.

### 2. `message.go`

Add `payload *spec.Schema` to `message` (`message.go:12`) and the `MessageFrom`
constructor (D3). `build` (`message.go:66`) becomes:

```go
func (m *message) build(b *builder) (*spec.Message, error) {
    payload := m.payload
    switch {
    case payload != nil && m.typ != nil:
        return nil, fmt.Errorf("message: payload schema and payload type are mutually exclusive")
    case payload == nil && m.typ == nil:
        return nil, fmt.Errorf("message: nil payload type or schema")
    case payload == nil:
        payload = schema.FromType(m.typ, b.defs)
    }
    if m.name == "" && m.typ == nil {
        return nil, fmt.Errorf("message: name is required for a hand-authored payload")
    }
    // ...existing spec.Message construction, with Payload: payload
}
```

`messageName` (`message.go:86`) already tolerates `typ == nil`; the explicit name
check above replaces its `"message"` fallback for the `MessageFrom` path so an
unnamed hand-authored payload fails loudly instead of silently becoming
`messages.message`. `.Headers(...)` needs no change — it already takes
`*spec.Schema`, so multi-format headers work through `spec.MultiFormat`.

### 3. `doc.go`

Add the D4 item next to the other component items (`MessageTraits` at
`doc.go:624`, `Channels` at `doc.go:875`), using `b.components()` (`doc.go:65`).

Add `validateSchemaNodes` (D5) and wire it into `Spec` (`doc.go:46`) beside
`validateServerRefs`/`validateSecurityRefs`/`validateReplyRefs`/
`validateTraitRefs`, before the `maps.Copy(c.Schemas, b.defs)` merge:

```go
errs = append(errs, b.validateSchemaNodes()...)
```

The walk is bounded and sorted (component keys, channel addresses, message names
ascending), because `SpecResult.Err` is compared by exact string in tests. It
covers:

| Source | Path reported |
| ------ | ------------- |
| `components.schemas` values | `schema.<name>.<field>` |
| `channel.messages[*].payload` | `channel.<address>.messages.<name>.payload` |
| `channel.messages[*].headers` | `channel.<address>.messages.<name>.headers` |
| `components.messageTraits[*].headers` | `messageTrait.<name>.headers` |

with recursion into the child keywords `properties`, `items`,
`additionalProperties`, `allOf`, `oneOf`, `anyOf`, `not`, and `definitions`, so a
multi-format node nested inside a plain object schema is still checked.

The keyword check is reflection-based — iterate the `Schema` fields other than
`SchemaFormat`/`Schema` and report those that are non-zero — so it does not need
maintenance when the backlog's B7 adds more Draft-07 keywords. The collision
check compares declared `components.schemas` keys against `b.defs`.

### 4. Tests

| File | Coverage |
| ---- | -------- |
| `spec/encode_test.go` | verbatim `schemaFormat`, field order, opaque-body round-trip, unchanged plain-schema encoding |
| `message_test.go` | `MessageFrom` passthrough, name, error cases, headers |
| `doc_test.go` | `Schemas` registration, duplicates, `$ref` resolution, D5 rejections |

### 5. `test/data/avro/`

A new fixture module in the existing shape (`go.mod` with a `replace` to the
repository root, `catalog.go`, generated `asyncapi.yaml`), registered in
`go.work` and in `fixtures` at
`test/integration/asyncgo_generate_test.go:38`.

The fixture must use an **inline** Avro body, not `schema: {$ref: ./user.avsc}`:
asyncgo emits external `$ref`s verbatim but does not bundle them, and the
containerized `asyncapi validate` cannot resolve a sibling file. The fixture
should exercise all three new surfaces:

1. an Avro component declared with `Schema`/`Schemas`, `$ref`'d from two
   messages via `MessageFrom(name, spec.Ref(...))`;
2. one inline multi-format payload on a third message, so the inline path is
   covered by the golden too;
3. a multi-format `headers` schema, proving D1's "accepted anywhere".

## Example (before / after)

Before, an Avro payload is inexpressible — `MessageOf` needs a Go type and
`spec.Schema` has no `schemaFormat`:

```go
// Fixed fields only; no place to put an Avro record.
type Order struct{ ID string `json:"id"` }
```

After:

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
    asyncgo.Schemas(asyncgo.Schema("UserAvro", userAvro)),
    asyncgo.Channels(
        asyncgo.Channel("order-placed").Send(asyncgo.Operation().
            Message(asyncgo.MessageFrom("OrderPlaced",
                spec.Ref("#/components/schemas/UserAvro")).
                Headers(spec.MultiFormat("application/vnd.apache.avro;version=1.9.0",
                    map[string]any{"type": "record", "name": "Headers"})))),
        asyncgo.Channel("order-shipped").Send(asyncgo.Operation().
            Message(asyncgo.MessageFrom("OrderShipped",
                spec.Ref("#/components/schemas/UserAvro")))),
    ),
)
```

Emitted (abridged), with the two surfaces annotated:

```yaml
channels:
  order-placed:
    address: order-placed
    messages:
      OrderPlaced:
        # D1: a multi-format schema is accepted in headers too.
        headers:
          schemaFormat: application/vnd.apache.avro;version=1.9.0
          schema:
            name: Headers
            type: record
        # D4: one declaration, $ref'd from two messages.
        payload:
          $ref: "#/components/schemas/UserAvro"
        name: OrderPlaced
  order-shipped:
    address: order-shipped
    messages:
      OrderShipped:
        payload:
          $ref: "#/components/schemas/UserAvro"
        name: OrderShipped
components:
  schemas:
    UserAvro:
      # D2: schemaFormat verbatim, body opaque.
      schemaFormat: application/vnd.apache.avro;version=1.9.0
      schema:
        # The body's map keys are sorted after materialization, never
        # reordered back into `type, name, fields`.
        fields:
        - name: displayName
          type: string
        - name: age
          type: int
        name: User
        type: record
```

## Edge cases

- **`schemaFormat` set to the spec's default value** (`application/vnd.aai.asyncapi+json;version=3.1.0`)
  with a JSON Schema body — a legal Multi Format Schema Object *only* if the body
  is under `schema`. Setting it beside keywords is rejected by D5; the
  recommended spelling for the JSON-Schema flavour is the plain Schema Object with
  `schemaFormat` omitted.
- **Only one of the two fields set** — rejected by D5. Unset means `""`/`nil`;
  a caller cannot express "present but empty" accidentally through
  `spec.MultiFormat`, since it always sets both.
- **Nested multi-format node inside a plain schema** (e.g. an Avro property) —
  covered by D5's recursion, and emitted correctly because the node is the same
  struct at any depth.
- **External `$ref` inside the body** (`schema: {$ref: ./user.avsc}`) — emitted
  verbatim, but asyncgo does not bundle external documents, so a consumer
  (including `asyncapi validate`) may fail to resolve it. The fixture uses an
  inline body; bundling is deferred.
- **Opaque body key order** — the harness round-trip decodes the body into
  `map[string]any`, so keys are emitted sorted (`fields, name, type`). This is
  cosmetic for Avro JSON (readers are order-insensitive) and identical to how
  `Schema.Properties` keys already behave; it is documented, not fixed.
- **Declared/auto-hoisted name collision** — `Spec` ends with
  `maps.Copy(c.Schemas, b.defs)`, so an FQN-keyed hoisted schema would silently
  overwrite a user-declared component of the same key. D5 closes this by
  comparing `components.schemas` keys against `b.defs` and reporting an error.
- **Repeated inline multi-format payload** — emitted inline on every message, so
  a schema used more than once should be declared with `Schemas(...)` and
  `$ref`'d. Documented in the README.
- **`MessageFrom` with an empty name or nil schema** — both are build errors
  (D3); neither silently produces `messages.message` or a null payload.
- **A Go struct passed as the opaque body** — JSON/YAML marshal by reflection, so
  it works; after the round-trip it becomes a map, like any other body. Not the
  intended spelling, which is a literal body.
- **Component names needing JSON Pointer escaping** (`/`, `~`) — consistent with
  existing component items, `Schema` does not escape; a caller writing the `$ref`
  must escape it.

## Rollout plan

1. **Stage 0 — design doc** (`docs`): this document, accepted, and the
   `docs/designdoc/README.md` index row.
2. **Stage 1 — object model** (`feat(spec)`): `spec.Schema` fields,
   `spec.MultiFormat`, and `spec/encode_test.go` round-trip coverage. Additive;
   existing output unchanged (goal 6 provable here in isolation).
3. **Stage 2 — DSL builders** (`feat(dsl)`): `MessageFrom` in `message.go`;
   `Schema`/`Schemas` and `validateSchemaNodes` in `doc.go`; `message_test.go`
   and `doc_test.go` coverage.
4. **Stage 3 — fixture and integration** (`test`): `test/data/avro/`, `go.work`,
   the `fixtures` list, and a regression check that the eight existing
   `asyncapi.yaml` files are untouched.
5. **Stage 4 — docs** (`docs`): README section, the `AGENT.md` Public API and
   `test/data` lists, and the `docs/asyncapi-3.1.0-coverage.md` §1, §4.3, and §8
   B4 updates.
6. **Stage 5 — deferred**: `schemaFormat` table validation and format constants,
   ordered body preservation, Avro-from-Go derivation, external `$ref` bundling,
   `Parameter.schema` (B6).

## Testing plan

- **Stage 1** — table-driven `spec/encode_test.go` cases:
  - `should_emit_schema_format_verbatim`
  - `should_emit_schema_format_before_schema`
  - `should_round_trip_opaque_body_through_yaml`
  - `should_round_trip_multi_format_through_json`
  - `should_leave_plain_schema_encoding_unchanged`
- **Stage 2** — `message_test.go` and `doc_test.go` cases:
  - `should_emit_message_from_payload_verbatim`
  - `should_require_message_from_name`
  - `should_return_error_on_nil_message_from_payload`
  - `should_emit_multi_format_headers`
  - `should_register_declared_schema_in_components`
  - `should_return_error_on_duplicate_schema_name`
  - `should_reject_schema_format_without_schema`
  - `should_reject_schema_without_schema_format`
  - `should_reject_schema_format_beside_json_schema_keywords`
  - `should_reject_declared_name_colliding_with_hoisted_schema`
  - `should_resolve_ref_to_declared_schema_from_two_messages`
- **Stage 3** — `make test`: the `avro` fixture satisfies the integration test's
  three independent assertions (golden, codec equivalence, `asyncapi validate`
  for YAML and JSON), and `git diff --stat test/data` shows no change to the
  eight existing fixtures. Docker is required; the test fails rather than skips
  without a daemon.
- **Race**: `go test ./... -race` per AGENT.md.

## Open / deferred

- **`schemaFormat` allow-list and constants** — e.g. `spec.SchemaFormatAvro`,
  `spec.SchemaFormatProtobuf`, and a warning (not an error) for values outside
  the spec's recommended table.
- **Ordered body preservation** — if sorted body keys ever matter, it needs an
  ordered-map or raw-node representation, which is a separate design because it
  affects the harness round-trip contract.
- **Avro/Protobuf generation from Go types** — the inverse direction; explicitly
  out of scope, and the reason `MessageFrom` exists.
- **External `$ref` bundling** — `schema: {$ref: ./x.avsc}` and friends.
- **`Parameter.schema`** — 3.1.0 removed it and the Parameter Object is otherwise
  wrong-shaped; tracked as B6, not here.
- **Coverage doc drift** — `docs/asyncapi-3.1.0-coverage.md` §1 currently marks
  the Multi Format Schema Object ❌/❌, §4.3 lists non-JSON-Schema formats as
  absent, and §8 B4 has no `Implemented` note; all three change in Stage 4.
