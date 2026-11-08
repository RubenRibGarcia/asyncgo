# Custom types

Two escapes from reflection-derived schemas: a **provider**, for a type you own
whose wire format reflection cannot describe, and a **mapping**, for a type you do
not own.

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

### Named scalars and collections

Any named type can declare a provider, not only a struct. A named scalar or
collection has no fields to reflect, so the point is the reusable constraint: the
schema is declared once and every use site becomes a `$ref` instead of repeating
an inline `{type: string}`.

```go
type OrderID string

func (OrderID) AsyncAPISchema() *spec.Schema {
	return &spec.Schema{Type: "string", Pattern: `^ord_[0-9]{6}$`}
}

type OrderPlaced struct {
	OrderID OrderID `json:"order_id" asyncapi:"required"`
}
```

```yaml
# components.schemas (abridged):
#   ...OrderID:
#     type: string
#     pattern: ^ord_[0-9]{6}$
#   ...OrderPlaced:
#     type: object
#     properties:
#       order_id:
#         $ref: "#/components/schemas/github.com~1acme~1orders.OrderID"
#     required:
#     - order_id
```

A named type *without* a provider keeps deriving inline, so implementing the
interface is the opt-in and existing documents do not change.

!!! warning "`AsyncAPISchema` describes a type, not an instance"

    It is invoked on a zero value, so it must describe the *type*: be pure,
    deterministic, and panic-free. Return `nil` to fall back to
    reflection-derived derivation, or `&spec.Schema{}` for an explicitly
    unconstrained schema.

Detection honors both value and pointer receivers, and a custom type referenced
from a `oneOf`/`anyOf`/`allOf` field is hoisted automatically — including a
non-struct member, which the generated harness registers as `new(T)` because a
composite literal is only valid for struct, array, slice and map types.

## Third-party types and declared mappings

`spec.SchemaProvider` needs a method on the type, which is impossible for a type
you do not own. `gofrs/uuid.UUID` is a named `[16]byte`, so it derives as an array
of integers, and the only workaround is a wrapper type in every payload signature.

A **mapping** declares how a type derives, keyed by its fully-qualified name
(`pkgPath.TypeName`), so nothing about the type has to change. Two entry points:

```yaml
# asyncgo.yaml, at the module root (read by both generate and check; --config
# names a different file)
mappings:
  github.com/gofrs/uuid.UUID:
    type: string
    format: uuid
  github.com/shopspring/decimal.Decimal:
    type: string
    pattern: '^-?\d+(\.\d+)?$'
```

```go
// or in Go, when the schema needs more than the config whitelist
import "github.com/RubenRibGarcia/asyncgo/schema"

func init() {
	schema.MapSchema[uuid.UUID](func() *spec.Schema {
		return &spec.Schema{Type: "string", Format: "uuid"}
	})
	// the name-keyed form: what asyncgo.yaml compiles to, also usable directly
	schema.MapSchemaByName("github.com/shopspring/decimal.Decimal",
		&spec.Schema{Type: "string", Pattern: `^-?\d+(\.\d+)?$`})
}
```

### Where the file lives

`asyncgo.yaml` in the module root is read automatically by both `generate` and
`check`; `--config <path>` names a different file. The path is resolved against
the current directory, it must exist, and it replaces the module-root file rather
than merging with it — one mapping set shapes a run.

A mapped type derives **inline** at every use site — fields, `[]T`, `map[K]T`,
`*T`, and anonymous structs containing it — so a document stays small. The one
exception is a mapped type a `oneOf`/`anyOf`/`allOf` directive names: its `$ref`
still resolves, because the mapped schema is also hoisted into
`components.schemas`.

```go
type OrderPlaced struct {
	ID   uuid.UUID   `json:"id" asyncapi:"required"`
	Refs []uuid.UUID `json:"refs"`
}
```

```yaml
# properties (abridged):
#   id:   { type: string, format: uuid }
#   refs:
#     type: array
#     items: { type: string, format: uuid }
```

### Precedence

An exact mapping beats a provider, which beats a mapping inherited through a
derived alias, which beats kind derivation. The reasoning: the document author is
overriding the type's *suggested* default, while a provider is a schema written
for that exact type. Registering a second mapping for the same name replaces the
first, so config beats the Go API, which beats a built-in.

### Derived aliases inherit

`type OrderID uuid.UUID` derives like `uuid.UUID` without a declaration of its
own, because the generator emits the declared-from link it finds in your module.
Chains follow (`type LineID OrderID`), and a mapping on an intermediate alias
applies to the types derived from it.

Two limits: a Go type alias (`type A = B`) needs no edge (it introduces no new
type), and alias declarations inside a dependency are not scanned, so a chain that
crosses into a third-party package stops there.

### Built-in mappings

These need no configuration and add no dependency — the table is data, so asyncgo
never imports the library it describes:

| Type | Derives as |
| --- | --- |
| `time.Time` | `{type: string, format: date-time}` |
| `github.com/gofrs/uuid.UUID` | `{type: string, format: uuid}` |
| `github.com/gofrs/uuid/v5.UUID` | `{type: string, format: uuid}` |
| `github.com/google/uuid.UUID` | `{type: string, format: uuid}` |
| `github.com/oklog/ulid/v2.ULID` | `{type: string, pattern: Crockford base32}` |
| `github.com/shopspring/decimal.Decimal` | `{type: string, pattern: decimal}` |
| `math/big.Int` | `{type: integer}` |

Each entry documents the library's own default JSON encoding. A project that
changes a library's wire format — `decimal.MarshalJSONWithoutQuotes`, say —
overrides the entry in its own `asyncgo.yaml`.

### Scope

Mappings are exact-name keyed: there is no kind-based mapping, so `[]byte` keeps
deriving as a base64 string and a named `type Raw []byte` derives the same way. A
field of a mappable type that needs a *different* schema should use a local
wrapper type or a provider — field-level mappings are not supported.

The `asyncgo.yaml` whitelist is scalar-shaped (`type`, `format`, `description`,
`pattern`, `enum`, `const`, `default`, `examples`, `contentEncoding`,
`contentMediaType`, `minLength`, `maxLength`, `minimum`, `maximum`,
`exclusiveMinimum`, `exclusiveMaximum`, `minItems`, `maxItems`, `readOnly`,
`writeOnly`, `deprecated`); `$ref`, nested objects and combinators are rejected by
`generate` and `check` before anything is built.

A mapping whose name never matched a derived type is reported as a warning on
stderr — a typo'd fully-qualified name would otherwise fail silently, since the
type may live in a package the harness never imports:

```text
asyncgo: schema mapping for github.com/gofrs/uuid.UUID was never used: no derived type matched that name
```

Mappings are applied when the document is finalized: `asyncgo generate` and
`asyncgo check` do that for you, and a program that builds documents in process
calls `schema.Finalize(doc)` to get the same result.
