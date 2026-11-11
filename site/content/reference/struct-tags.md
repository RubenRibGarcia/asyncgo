# Struct tags

A message payload is derived from a Go type, and every keyword the derivation
cannot infer from the type itself comes from a struct tag. `asyncgo` reads exactly
two of them:

| Tag | What it decides |
| --- | --- |
| `json` | the property name, and whether the field appears at all |
| `asyncapi` | the JSON Schema keywords the Go type does not carry |

Nothing else is tagged. Channels, operations, servers, messages, traits, security
and protocol bindings are declared with the DSL — see the
[API index](index.md) for the builders. A struct tag can only shape a payload
schema.

Two passes read the same tags: the reflection pass in `schema/` derives the schema
from `reflect.StructField`, and the static pass in `internal/discovery` walks the
source to validate the vocabulary and to resolve the types a combinator names.
Both use one grammar, so they cannot disagree about how a tag is split
(ADR 0007).

!!! note "Grammar"

    Directives are separated by **whitespace**. The first `=` splits a directive
    into key and value, so a value may contain a comma — `examples=1,000` is one
    example — but not a space. The old comma-separated form is rejected with a
    message that says so.

## A tagged struct and the schema it produces

```go
// OrderPlaced is the payload of the order-placed message.
type OrderPlaced struct {
	// OrderID is the merchant's identifier.
	OrderID string `json:"order_id" asyncapi:"required format=uuid"`
	// Sku names the product.
	Sku string `json:"sku" asyncapi:"required enum=sku-1|sku-2"`
	// Tags are free-form labels.
	Tags []string `json:"tags" asyncapi:"minItems=1 uniqueItems"`
	// Note is shown to the customer.
	Note string `json:"note,omitempty" asyncapi:"deprecated examples=none"`
	// Receipt is a PNG; []byte is base64 by kind.
	Receipt []byte `json:"receipt" asyncapi:"contentMediaType=image/png"`
	// PlacedAt is when the order was accepted.
	PlacedAt time.Time `json:"placed_at" asyncapi:"required"`
}
```

```yaml
# components.schemas (abridged; "..." elides the fully-qualified name):
#   ...OrderPlaced:
type: object
properties:
  note:
    type: string
    description: Note is shown to the customer.
    deprecated: true
    examples:
    - none
  order_id:
    type: string
    description: OrderID is the merchant's identifier.
    format: uuid
  placed_at:
    type: string
    description: PlacedAt is when the order was accepted.
    format: date-time
  receipt:
    type: string
    description: Receipt is a PNG; []byte is base64 by kind.
    format: byte
    contentEncoding: base64
    contentMediaType: image/png
  sku:
    type: string
    description: Sku names the product.
    enum:
    - sku-1
    - sku-2
  tags:
    type: array
    description: Tags are free-form labels.
    items:
      type: string
    minItems: 1
    uniqueItems: true
required:
- order_id
- sku
- placed_at
```

Two things in that output are not tags at all. The `description` is the field's
doc comment — reflection cannot see comments, so the static pass supplies it. And
`format: date-time` on `PlacedAt` is derived from the type. A tag only ever adds
what the type cannot say.

## Flags

A flag is a bare key. Giving it a value is an error — `readOnly=true` fails
generation rather than being ignored.

| Directive | Emits | Notes |
| --- | --- | --- |
| `required` | the property name in the parent object's `required` array | Fields are optional by default |
| `readOnly` | `readOnly: true` | |
| `writeOnly` | `writeOnly: true` | |
| `uniqueItems` | `uniqueItems: true` | Meaningful on arrays |
| `deprecated` | `deprecated: true` | |

`required` is the one that is not a JSON Schema keyword on the property: it adds
the field to its parent's `required` list, so a field that the parent never
derives (see [`json:"-"`](#the-json-tag)) cannot be marked required.

## Value directives

A value directive requires a non-empty value. `format=` with nothing after it is
an error.

| Directive | Emits | Notes |
| --- | --- | --- |
| `enum=a\|b\|c` | `enum: [a, b, c]` | Split on `\|`; an empty member is an error |
| `examples=x` | appends to the `examples` array | **Repeatable** — `examples=a examples=b` emits two entries; unlike `enum`, a value may contain `\|` |
| `format=date-time` | `format: date-time` | Any Draft-07 format; not validated against the vocabulary |
| `contentEncoding=base64` | `contentEncoding: base64` | On a `[]byte` field it does not clear the derived `format: byte` |
| `contentMediaType=image/png` | `contentMediaType: image/png` | Never derived — the media type is not in the Go type |
| `const=accepted` | `const: accepted` | Always a string, even for `0` or `false` |
| `discriminator=kind` | `discriminator: kind` | The property name, not the value |
| `minItems=1` | `minItems: 1` | Array length bound |
| `maxItems=10` | `maxItems: 10` | Array length bound |
| `minProperties=1` | `minProperties: 1` | Object size bound |
| `maxProperties=8` | `maxProperties: 8` | Object size bound |

!!! warning "String bounds are not derivable"

    `minLength`, `maxLength`, `pattern`, `minimum`, `maximum` and `default` have
    no directive. There is no tag for them, so write those keywords with
    [`spec.SchemaProvider`](../custom-types.md) when a payload needs them.

## Combinators

`oneOf=`, `anyOf=` and `allOf=` name other **types** on a field, separated by `|`.
The named types are hoisted into `components.schemas` automatically, so the
document is valid even if nothing else references them.

```go
type Payment struct {
	Identified
	State any `json:"state" asyncapi:"oneOf=Paid|Cancelled"`
}
```

```yaml
# ...Payment:
type: object
properties:
  id:
    type: string
  state:
    oneOf:
    - $ref: "#/components/schemas/...Paid"
    - $ref: "#/components/schemas/...Cancelled"
required:
- id
```

A member name is resolved one of two ways:

| Written | Resolves to |
| --- | --- |
| `oneOf=Paid` | `Paid` in the **declaring package** |
| `oneOf=example.com/acme/pay.Paid` | that type in that package — a name containing `/` is treated as fully qualified `pkgPath.TypeName` |
| `oneOf=pay.Paid` | **rejected** — a dot without an import path is neither form, and generation fails telling you to use the full path |

## Embedding: flatten or compose

An embedded struct is flattened by default, matching `encoding/json`:

```go
type Payment struct {
	Identified // → id becomes a property of Payment, required: [id]
}
```

Tag it `asyncapi:"allOf"` to compose it instead:

```go
type Refund struct {
	Identified `asyncapi:"allOf"`
	Reason     string `json:"reason" asyncapi:"required"`
}
```

```yaml
# ...Refund:
type: object
allOf:
- $ref: "#/components/schemas/...Identified"
- type: object
  properties:
    reason:
      type: string
  required:
  - reason
```

The distinction matters for validation: a flattened `id` is required on the child
object, while an `allOf` composition says the child *is* an `Identified` plus
more. An embedded struct whose `json` tag names a field is a regular property in
either case, because `encoding/json` does not inline it either.

## The `json` tag

`asyncgo` follows `encoding/json` rather than inventing its own rules:

| Tag | Result |
| --- | --- |
| `json:"order_id"` | property `order_id` |
| `json:",omitempty"` | names no field, so the **Go field name** stands; options after the comma are ignored |
| `json:"-"` | the field is skipped entirely |
| unexported field | never derived, tag or not |

An embedded struct with an explicit `json` name (`Base `json:"base"``) is not
flattened — it becomes a `base` property, which is what `encoding/json` does with
it as well.

Because a skipped field is dropped before its `asyncapi` tag is read,
`json:"-" asyncapi:"required"` marks nothing required. The directive is still
validated as a key, but it has no effect — don't write it.

## What tags cannot do

| You might expect | Where it lives instead |
| --- | --- |
| A property `description` | the field's doc comment |
| `default` | no directive; use [`spec.SchemaProvider`](../custom-types.md) |
| Message, channel, operation or server metadata | the DSL builders — see the [API index](index.md) |
| Protocol bindings | the DSL, one struct per protocol and level |
| Overriding a derivation for a third-party type | [`asyncgo.yaml` mappings](../custom-types.md#third-party-types-and-declared-mappings) |

## Rejected spellings

The directive vocabulary is closed and validated by the static pass, before the
generator's harness is built. An unknown or misspelt directive is a **catalog
build error**, not a silently dropped keyword: `generate` and `check` exit `1` and
name the file, line and column.

```console
$ asyncgo generate .
asyncgo: generating: bad.go:4:17: asyncapi:"requird": unknown directive "requird"; did you mean "required"?
asyncgo: generating: bad.go:5:17: asyncapi:"required,format=uuid": "required,format": directives are separated by spaces, not commas
asyncgo: generating: bad.go:6:17: asyncapi:"readOnly=true": "readOnly" is a flag and takes no value
asyncgo: generating: bad.go:7:17: asyncapi:"example=x": unknown directive "example"; did you mean "examples"?
asyncgo: generating: bad.go:8:17: asyncapi:"maxLength=140": unknown directive "maxLength"; want one of allOf, anyOf, const, contentEncoding, contentMediaType, deprecated, discriminator, enum, examples, format, maxItems, maxProperties, minItems, minProperties, oneOf, readOnly, required, uniqueItems, writeOnly
asyncgo: generating: bad.go:9:17: asyncapi:"oneOf": "oneOf" requires a value
```

Every violation in the module is reported in one run, so a catalog migrating from
an older spelling is fixed in one pass.

| Written | Why it fails |
| --- | --- |
| `required,format=uuid` | The pre-ADR-0007 comma-separated form |
| `example=x` | The singular `example` was removed in ADR 0006 — `examples` replaced it |
| `maxLength=140`, `pattern=…`, `default=…` | Never implemented; string and numeric bounds have no directive |
| `type=string`, `map=…` | Rejected alternatives in ADR 0002 and ADR 0008 — use `spec.SchemaProvider` or a declared mapping |
| `readOnly=true` | A flag takes no value |
| `oneOf`, `format=` | A value directive needs a non-empty value |
| `oneOf=A\|` | An empty member is a typo, not a type named `""` |

!!! note "Validation needs the source"

    The check runs in the static pass, which walks the module's packages. Two
    consequences: a payload type imported from a dependency is not checked (declare
    a [mapping](../custom-types.md#third-party-types-and-declared-mappings) for it),
    and a document built in process by calling the builders is not checked either —
    run `asyncgo check` in CI to get the guarantee.

## Next

- [Schema derivation](../schema.md) — the rules that turn a type into a schema
- [Custom types](../custom-types.md) — `spec.SchemaProvider` and declared mappings
- [Concepts](../concepts.md) — how a catalog becomes a document
