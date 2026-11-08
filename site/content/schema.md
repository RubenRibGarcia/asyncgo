# Schema derivation

A message payload is derived from a Go type. These are the rules that turn the
type into a JSON Schema, and the composition it reads out of your struct
declarations. Nothing here needs a hand-written schema.

## Derivation rules

- **Fully-qualified names** — hoisted schemas are keyed `pkgPath.TypeName`
  (e.g. `example.com/orders/orders.OrderPlaced`); `$ref` escapes `/` per JSON
  Pointer.
- **Optional by default** — a field is required only with `asyncapi:"required"`.
- **Whitespace-separated directives** — `asyncapi:"required examples=1,000"`
  separates directives with spaces, so a value may itself contain a comma.
- **Descriptions from comments** — a field's `description` is read from its doc
  comment; `enum`, `examples`, `format`, `contentEncoding`, and `contentMediaType`
  still come from the `asyncapi` tag.
- **Binary fields** — a `[]byte` field derives to `type: string` with
  `format: byte` and `contentEncoding: base64`, matching how `encoding/json`
  marshals it. The media type is never derived: a Go type carries none, so set it
  with `asyncapi:"contentMediaType=image/png"` when it is known.
- **Always hoist** named struct types into `components.schemas`; only anonymous
  inline types are inlined. A named non-struct is inlined too, unless it
  implements `spec.SchemaProvider` (see [Custom types](custom-types.md)).
- **Embedding** — embedded structs are flattened by default (matching
  `encoding/json`); tag one with `asyncapi:"allOf"` to compose it instead.
- **Union fields** — `asyncapi:"oneOf=A|B"`, `anyOf=A|B`, or `allOf=A|B` on a
  field emits `$ref`s to the named types, which are hoisted automatically.
- **Custom types** — a named type implementing `spec.SchemaProvider` overrides
  derivation: its `AsyncAPISchema()` result is hoisted under the type's
  fully-qualified name as-is, and referenced via `$ref` from every use site. This
  works for structs and for named non-structs alike — see
  [Custom types](custom-types.md).
- **Declared mappings** — a mapping registered for a type's fully-qualified name
  replaces both the provider and kind derivation for that type, and derives the
  mapped schema inline at every use site. Declare one in `asyncgo.yaml` or with
  `schema.MapSchema`; a type you do not own can be mapped without a wrapper — see
  [Custom types](custom-types.md).

## Schema composition

Go struct composition maps onto the JSON Schema combinators `allOf`, `oneOf`, and
`anyOf` — no hand-written schema required.

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
	Data any `json:"data" asyncapi:"required oneOf=OrderPlaced|OrderCancelled"`
}
```

```yaml
# data:
#   oneOf:
#     - $ref: "#/components/schemas/...OrderPlaced"
#     - $ref: "#/components/schemas/...OrderCancelled"
```

!!! warning "Union fields are typed `any` on purpose"

    AsyncAPI 3.1.0 Schema Objects are a superset of **JSON Schema Draft 07**,
    where `$ref` siblings are ignored — so a union field should be typed `any`
    (or an interface), not a concrete type.

## Next

- **[Custom types](custom-types.md)** — providers for types with custom
  (de)serialization, and mappings for types you do not own.
- **[The full keyword set](https://github.com/RubenRibGarcia/asyncgo/blob/master/docs/adr/0006-schema-keywords.md)**
  — ADR 0006 records the Draft-07 and AsyncAPI Schema keywords `asyncgo` models.
