# Design: Schema Object keyword coverage

- **Status**: Accepted
- **Created**: 2026-10-10
- **Status updated**: 2026-10-10
- **Scope**: `spec/`, `schema/`, `doc.go` (root DSL), `message.go`, `docs/`

## Summary

`spec.Schema` models only part of the JSON Schema Draft-07 vocabulary plus the
three AsyncAPI-specific Schema keywords. Issue
[#18](https://github.com/RubanRibGarcia/asyncgo/issues/18) / coverage item §8 B7
tracks the gap.

This design adds the 23 missing keywords to the object model, makes the
mechanically derivable ones settable from `asyncapi` struct tags, and gives the
Message DSL a way to attach Draft-07 `examples` to a payload schema — including
when the payload is a `$ref` into `components.schemas`, which is what
`MessageOf(v)` always produces.

## Background

### One struct, two objects

`spec.Schema` is deliberately a single struct serving both the Schema Object and
the Multi Format Schema Object (`spec/schema.go` doc comment): the discovery
harness round-trips every document through YAML, and a union type would decode
into a map that re-encodes with sorted keys, rewriting committed golden files.

The consequences that constrain everything below:

- A field added for the Schema Object is also a field on the Multi Format body,
  so the "must not carry a JSON Schema keyword" invariant has to keep holding.
- All 23 new fields must be `omitempty`, so a document that does not use them
  serializes byte-identically. No committed golden fixture may change.

### The two validation helpers are not symmetric

| Helper | File | How it enumerates | Effect of adding fields |
| --- | --- | --- | --- |
| `jsonSchemaKeywords` | `doc.go:505` | **reflection** over every struct field not in `multiFormatFields` and not zero | self-covering — picks up all 23 with no edit |
| `schemaChildren` | `doc.go:469` | **explicit** per-field list | **must be extended**, or validation stops recursing |

`schemaChildren` feeds `schemaNodeErrors` (`doc.go:436`), which walks every
schema reaching the document and reports Multi Format invariant violations at a
path. If it is not extended, an invalid Multi Format node nested under `if`,
`then`, `else`, `contains`, `propertyNames`, `patternProperties`,
`dependencies`, or `additionalItems` is silently accepted. That is a
correctness bug introduced by adding the fields, so the fix ships with them.

### The tag vocabulary today

`schema/tags.go` parses the `asyncapi` struct tag. `applyTag` (`:21`) handles
`key=value` directives (`enum=`, `examples=`, `format=`, `const=`,
`discriminator=`, and the four numeric bounds); `hasFlag` (`:39`) handles bare
flags (`required`, `allOf`, `readOnly`, `writeOnly`, `uniqueItems`,
`deprecated`). `oneOf=`/`anyOf=`/`allOf=` are
recognized by `combinatorNames` (`:54`) and acted on by the **caller**
(`fillFields` in `schema/derive.go`), not by `applyTag`.

`applyTag` returns nothing — the whole `FromType` → `fillFields` → `applyTag`
chain is error-free. That constrains malformed-value handling (D6).

### Spec facts

Read from the normative markdown at `asyncapi/spec@v3.1.0` (`spec/asyncapi.md`,
§Schema Object, lines 1925–1990):

| Keyword | Type in 3.1.0 | Note |
| --- | --- | --- |
| `discriminator` | `string` | **Not** the OpenAPI Discriminator Object. Names the schema property used for polymorphism; that property must also be in `required`. |
| `externalDocs` | `External Documentation Object \| Reference Object` | `spec.ExternalDocs` already exists and is reused by six other objects. |
| `deprecated` | `boolean` | Default `false`. |
| `examples` | array | Draft-07. **`example` (singular) is not a 3.1.0 Schema keyword.** |

3.1.0 also states that `$ref` in a Schema Object *"MUST follow the behavior
described by Reference Object instead of the one in JSON Schema definition"*.
Sibling keywords beside `$ref` therefore do not compose — which is what forces
D7.

`example` (singular) was, until this change, a 2.x carry-over the model emitted:
`spec.Schema.Example`, mapped from `asyncapi:"example=…"`, with a committed
usage in `test/data/provider`. It is not a 3.1.0 keyword, so it was a §7
deviation. D2 removes it and migrates that fixture to `Examples`.

## Goals / Non-goals

**Goals**

1. All 23 keywords in the "Keyword inventory" table exist on `spec.Schema` with
   exact `json`/`yaml` wire names, and survive a JSON **and** YAML round-trip.
2. The mechanically derivable subset is settable from `asyncapi` struct tags.
3. A Message DSL builder attaches Draft-07 `examples` to a payload schema,
   including through a `$ref`, without ever writing a sibling of `$ref`.
4. Validation recursion (`schemaChildren`) covers every new schema-valued
   keyword, so the Multi Format invariant still holds at depth.
5. No *incidental* change to committed golden fixtures: a document that does not
   use the new keywords serializes unchanged. The one deliberate exception is
   the `test/data/provider` fixture migrated from the removed `example` to
   `examples` (D2).

**Non-goals (v1)**

- **The `dependencies` `[string]` shorthand.** The schema form is modelled; the
  property-name array is sugar with an equivalent schema form (D4, O6).
- **Tuple-form `items`.** Draft-07 allows `items` to be an array of schemas;
  `spec.Schema.Items` is a single `*Schema`. Widening it is a separate change
  (see O1).
- **Deriving structural keywords from tags** (`if`/`then`/`else`, `contains`,
  `propertyNames`, `patternProperties`, `dependencies`, `additionalItems`).
  They are reachable through `MessageFrom`, `spec.SchemaProvider`, and
  `Schema(...)`, but not reflected from Go struct tags.
- **2019-09 / 2020-12 keywords.** `$defs`, `unevaluatedProperties`,
  `dependentRequired`, `prefixItems`, and friends are out of scope: 3.1.0 is a
  Draft-07 superset. The repo already uses `definitions`, not `$defs`.
- **Tag value validation errors.** Deferred (D6, O2).
- **A new `test/data` fixture.** Keyword coverage is asserted at the `spec`
  layer; the existing integration suite must stay green unchanged.

## Design decisions

### D1 — Scope: the 19 keywords in the issue plus the 4 remainder

The issue body names 19 keywords; its title says "3.1.0 and remaining Draft-07".
The spec's own bullet list and Draft-07 define four more that the body omits:
`examples`, `additionalItems`, `$comment`, `dependencies`.

**Rationale**: "remaining Draft-07" is the stated intent, and leaving these four
out would leave §4.3 stale and the gap half-closed. All four are additive
`omitempty` fields with no behavior change.

### D2 — `Examples` is the only example keyword; the singular `Example` is removed

`Examples` is the Draft-07 array and the only example keyword AsyncAPI 3.1.0
defines. The 2.x singular `Example` field and its `asyncapi:"example=…"`
directive are **removed**, not deprecated. Its committed usage in
`test/data/provider` was migrated to `Examples`.

**Rationale**: `example` is not a 3.1.0 Schema keyword, so the field made the
model emit non-conformant output — a §7 deviation with no upside once `Examples`
exists — and two fields expressing one idea invites picking the wrong one. This
is a breaking change, taken deliberately while the library is pre-1.0.

**Consequence, accepted knowingly**: `applyTag` ignores an unrecognised
directive, so a catalog still carrying `asyncapi:"example=x"` is silently
dropped — no error, no warning. Mapping `example=` onto `examples=` was the
alternative and was rejected because it keeps the deprecated spelling alive.
Making unknown directives loud (O2) is the real fix for the whole class; until
then a test pins the silent ignore so it stays deliberate and visible.

### D3 — Keyword-to-field names

| Keyword | Field | Why this name |
| --- | --- | --- |
| `$ref` | `Ref` (existing) | — |
| `$id` | `ID` | Drops the sigil, matching `Ref`. |
| `$schema` | `SchemaURI` | `Schema` is taken by the Multi Format body. `$schema` holds a URI (`http://json-schema.org/draft-07/schema#`), so `SchemaURI` is accurate; `SchemaVersion` would invite a version *number*, which is wrong. |
| `$comment` | `Comment` | Drops the sigil. |

**Rationale**: `SchemaURI` is the only candidate whose name does not actively
suggest the wrong value. It is a public API one-way door, so it is called out
here rather than buried in code.

### D4 — `Dependencies` is `map[string]*Schema`

Draft-07 types `dependencies` as `Schema | [string]` per key. The field models
the **schema form only**. `map[string]any` was rejected after measurement: a
schema-valued entry stored in an `any` decodes to a map and re-encodes with
**sorted** keys, so `type: object` / `properties:` swap order between the first
and the second marshal — breaking the byte-stable YAML round-trip that the
one-struct `spec.Schema` design exists to protect (Background). Verified
empirically before deciding, not assumed.

**Rationale**: The `[string]` form is sugar with an exact schema-form
equivalent — `creditCard: [billingAddress]` means the same as
`creditCard: {required: [billingAddress]}` — so narrowing loses no
expressiveness, only a shorthand. That keeps the field type-safe and byte-stable
with no custom codec, for a keyword Draft 2019-09 already deprecated in favour
of `dependentRequired`/`dependentSchemas`.

### D5 — Booleans are `bool`; `Const`/`Default` keep `omitempty`

`deprecated`, `readOnly`, `writeOnly`, `uniqueItems` are plain `bool` with
`omitempty`: absent and `false` are semantically identical for these keywords,
so dropping `false` is correct — this is **not** the
[B8](../asyncapi-3.1.0-coverage.md#b8--binding-zero-values-are-dropped)
zero-value problem, where zero is semantically distinct.

`Const any`, like the existing `Default`, loses a falsy value (`false`, `0`,
`""`) to `omitempty`.

**Rationale**: Consistency with the existing any-valued field, and a one-off
presence wrapper for `Const` alone would be a wart. The general fix belongs with
B8 for both at once (O4). Documented, not hidden.

**Rationale**: Consistency with the two existing any-valued fields, and a
one-off presence wrapper for `Const` alone would be a wart. The general fix
belongs with B8 for all three at once (O4). Documented, not hidden.

### D6 — Derivable set and directive spelling

Derivable from a Go struct tag, all on `*Schema`-valued properties:

| Keyword | Directive | Form |
| --- | --- | --- |
| `readOnly` | `readOnly` | bare flag |
| `writeOnly` | `writeOnly` | bare flag |
| `uniqueItems` | `uniqueItems` | bare flag |
| `deprecated` | `deprecated` | bare flag |
| `minItems` | `minItems=1` | `key=value`, `*uint64` |
| `maxItems` | `maxItems=10` | `key=value`, `*uint64` |
| `minProperties` | `minProperties=1` | `key=value`, `*uint64` |
| `maxProperties` | `maxProperties=8` | `key=value`, `*uint64` |
| `const` | `const=OrderPlaced` | `key=value`, string |
| `examples` | `examples=a,examples=b` | `key=value`, repeatable, appends |
| `discriminator` | `discriminator=kind` | `key=value`, string |

Bare flags use camelCase, matching the existing `allOf` flag; `deprecated` is
lowercase because it is one word. `examples=` is **repeatable** rather than
`|`-separated so an example containing `|` is expressible, unlike `enum=`.

**A malformed value is ignored.** `minItems=abc` sets nothing. `applyTag` has no
error channel and the whole derivation chain is error-free, so surfacing this
would mean threading errors through `FromType` — a much larger change than the
issue warrants. This matches the parser's existing behavior for an unknown
directive. It is a documented limitation, not a silent default (O2).

**Rationale**: Only keywords whose value is expressible in a tag string, or
derivable from a Go type, belong here. `if`/`then`/`else`, `contains`,
`propertyNames`, `patternProperties`, `dependencies`, `additionalItems`,
`$id`, `$schema`, and `externalDocs` need a schema or an object, not a scalar,
so they stay authoring-only.

### D7 — `PayloadExamples` resolves into the hoisted component, and conflicts are errors

`message.PayloadExamples(examples ...any)` attaches Draft-07 `examples` to the
payload schema. Resolution in `(*message).build`:

| Payload shape | Where `examples` is written |
| --- | --- |
| inline (`MessageFrom` with a non-`$ref`, or `SchemaProvider`) | `payload.Examples` |
| `$ref` (`MessageOf`, or `MessageFrom` + `spec.Ref`) | the resolved `*spec.Schema` in `b.defs`, else `b.components().Schemas` |
| `$ref` that resolves to nothing | error — never a silent no-op |

Writing `examples` as a sibling of `$ref` is rejected outright (Background: 3.1.0
`$ref` follows Reference Object semantics, where siblings are ignored; a strict
validator may reject them). Emitting a keyword that does not mean what it says
is worse than erroring.

`MessageOf` hoists **one** `components.schemas` entry per Go type, shared by
every channel carrying that type. Therefore:

- the same set declared twice is idempotent — no error;
- a second message declaring a **different** set on the same component is a hard
  error naming the message and the component.

**Rationale**: Silent concatenation would make a shared component's `examples`
depend on item order and leave a reader unable to tell which message contributed
what. Erroring on genuine conflict keeps the common case (one message owns the
type) frictionless while refusing to guess.

### D8 — `schemaChildren` is extended to every schema-valued keyword

Add to `schemaChildren` (`doc.go:469`), with nil checks, in deterministic order:
`patternProperties` (sorted keys), `additionalItems`, `contains`,
`propertyNames`, `dependencies`, `if`, `then`, `else`.

**Rationale**: `schemaNodeErrors` is the only place the Multi Format invariant
is enforced at depth. Leaving the new branches out would let an invalid
multi-format node hide one level down. `jsonSchemaKeywords` needs no edit — it
reflects over the struct, which is why the new fields are auto-covered on the
"must not carry a keyword" side. The asymmetry is the trap.

### D9 — Declaration order

New fields are inserted next to their existing kin; **no existing field moves**.
Inserting between existing fields preserves their relative order, so emitted
output for a document that does not use the new keywords is unchanged.

```text
SchemaFormat, Schema                                    (Multi Format; stays first)
Ref, ID, SchemaURI                                      identification
Type, Title, Description, Format, Comment, ExternalDocs, Deprecated
Properties, Required, Items, AdditionalProperties, AdditionalItems,
  PatternProperties, PropertyNames, Dependencies, Contains
Enum, Const, Examples, Default
Definitions, OneOf, AllOf, AnyOf, Not, If, Then, Else
MinLength, MaxLength, Pattern                           string constraints
Minimum, Maximum, ExclusiveMinimum, ExclusiveMaximum, MultipleOf
MinItems, MaxItems, UniqueItems                         array constraints
MinProperties, MaxProperties                            object constraints
ReadOnly, WriteOnly
Discriminator
```

**Rationale**: declaration order *is* the emitted key order. The order is pinned
by a test (Testing plan, stage 1) so a later reshuffle cannot silently rewrite
documents.

### D10 — `additionalItems` ships as an authoring-only, currently-inert keyword

`additionalItems` only applies when `items` is the tuple (array) form, which
this model cannot express. It is added because it was in the agreed scope, and
documented as inert until O1 widens `Items`.

**Rationale**: Adding the field is cheap, unblocks hand-authored documents the
day tuple `items` lands, and the alternative — silently dropping a keyword the
issue asked for — is worse. The limitation is stated in godoc and in the
coverage doc so nobody expects it to take effect.

## Detailed design

### 1. `spec/schema.go` — the fields

23 fields, per D1/D3/D5/D9. Exact tags (`json` first, then `yaml`, as in the
existing struct):

```go
// Identification
ID        string `json:"$id,omitempty"     yaml:"$id,omitempty"`
SchemaURI string `json:"$schema,omitempty" yaml:"$schema,omitempty"`

// Annotations
Comment      string        `json:"$comment,omitempty"     yaml:"$comment,omitempty"`
ExternalDocs *ExternalDocs `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
Deprecated   bool          `json:"deprecated,omitempty"   yaml:"deprecated,omitempty"`

// Structure
AdditionalItems   *Schema            `json:"additionalItems,omitempty"   yaml:"additionalItems,omitempty"`
PatternProperties map[string]*Schema `json:"patternProperties,omitempty" yaml:"patternProperties,omitempty"`
PropertyNames     *Schema            `json:"propertyNames,omitempty"     yaml:"propertyNames,omitempty"`
Dependencies      map[string]*Schema `json:"dependencies,omitempty"      yaml:"dependencies,omitempty"`
Contains          *Schema            `json:"contains,omitempty"          yaml:"contains,omitempty"`

// Values
Const    any   `json:"const,omitempty"    yaml:"const,omitempty"`
Examples []any `json:"examples,omitempty" yaml:"examples,omitempty"`

// Combinators
If   *Schema `json:"if,omitempty"   yaml:"if,omitempty"`
Then *Schema `json:"then,omitempty" yaml:"then,omitempty"`
Else *Schema `json:"else,omitempty" yaml:"else,omitempty"`

// Array constraints
MinItems    *uint64 `json:"minItems,omitempty"    yaml:"minItems,omitempty"`
MaxItems    *uint64 `json:"maxItems,omitempty"    yaml:"maxItems,omitempty"`
UniqueItems bool    `json:"uniqueItems,omitempty" yaml:"uniqueItems,omitempty"`

// Object constraints
MinProperties *uint64 `json:"minProperties,omitempty" yaml:"minProperties,omitempty"`
MaxProperties *uint64 `json:"maxProperties,omitempty" yaml:"maxProperties,omitempty"`

// Access
ReadOnly  bool `json:"readOnly,omitempty"  yaml:"readOnly,omitempty"`
WriteOnly bool `json:"writeOnly,omitempty" yaml:"writeOnly,omitempty"`

// AsyncAPI polymorphism
Discriminator string `json:"discriminator,omitempty" yaml:"discriminator,omitempty"`
```

### 2. `doc.go` — validation recursion

Extend `schemaChildren` per D8. Shape (nil-checked, deterministic, mirroring the
existing `slices.Sorted(maps.Keys(...))` idiom):

```go
for _, name := range slices.Sorted(maps.Keys(s.PatternProperties)) {
	children = append(children, schemaChild{"patternProperties." + name, s.PatternProperties[name]})
}
for _, name := range slices.Sorted(maps.Keys(s.Dependencies)) {
	children = append(children, schemaChild{"dependencies." + name, s.Dependencies[name]})
}
for _, n := range []struct {
	name string
	s    *spec.Schema
}{
	{"additionalItems", s.AdditionalItems},
	{"contains", s.Contains},
	{"propertyNames", s.PropertyNames},
	{"if", s.If},
	{"then", s.Then},
	{"else", s.Else},
} {
	if n.s != nil {
		children = append(children, schemaChild{n.name, n.s})
	}
}
```

### 3. `schema/tags.go` — directives

Extend the doc comment (`:9`) and `applyTag` (`:21`). Flags go through
`hasFlag`; scalars reuse the existing `strings.HasPrefix` + `TrimPrefix` shape.
`minItems=`/`maxItems=`/`minProperties=`/`maxProperties=` parse with
`strconv.ParseUint(v, 10, 64)` and **ignore** a parse error (D6).

### 4. `message.go` — `PayloadExamples`

```go
// PayloadExamples attaches JSON Schema Draft-07 examples to the message's
// payload schema. When the payload is a $ref (MessageOf always produces one),
// the examples are written to the referenced components.schemas entry — never
// as a sibling of $ref, where they would be ignored.
func (m *message) PayloadExamples(examples ...any) *message
```

Backed by a `payloadExamples []any` field on `message`, resolved in `build`
after `payload` is computed:

- `payload.Ref == ""` → append to `payload.Examples`.
- `payload.Ref != ""` → parse the component name from the ref, look it up in
  `b.defs` then `b.components().Schemas`, and merge per D7; unresolved →
  `fmt.Errorf("message.%s: examples: cannot resolve payload %q: %w", ...)`.

`b.defs` already holds the target at `build` time for `MessageOf`
(`schema.FromType(m.typ, b.defs)` runs immediately above), so no post-pass is
needed. A `MessageFrom` + `spec.Ref` pointing at a `Schema(...)` item declared
later in the same `Spec(...)` call is the one ordering hazard; if that proves
unresolvable in practice, resolution moves to a post-pass in `Spec` before the
merge at `doc.go:63` — the same pattern as `validateServerRefs`.

## Example (before / after)

```go
type OrderPlaced struct {
	Kind   string   `json:"kind"   asyncapi:"required,const=OrderPlaced"`
	Items  []string `json:"items"  asyncapi:"minItems=1,maxItems=10,uniqueItems,examples=sku-1,examples=sku-2"`
	Note   string   `json:"note"   asyncapi:"deprecated,writeOnly,maxLength=140"`
	Secret string   `json:"secret" asyncapi:"readOnly"`
}

asyncgo.Channels(
	asyncgo.Channel("order-placed").
		Message(asyncgo.MessageOf(OrderPlaced{}).
			PayloadExamples(map[string]any{"kind": "OrderPlaced", "items": []any{"sku-1"}})),
)
```

```yaml
components:
  schemas:
    example.OrderPlaced:
      type: object
      properties:
        kind:   {type: string, const: OrderPlaced}
        items:
          type: array
          items: {type: string}
          minItems: 1
          maxItems: 10
          uniqueItems: true
          examples: [sku-1, sku-2]
        note:   {type: string, maxLength: 140, writeOnly: true, deprecated: true}
        secret: {type: string, readOnly: true}
      required: [kind]
      examples:                      # <- PayloadExamples wrote here, not on the $ref
        - kind: OrderPlaced
          items: [sku-1]
```

## Edge cases

- **A message payload that is a `$ref` to a `components.schemas` entry no item
  declared** — build error naming the message and the ref; never a no-op.
- **Two messages, same hoisted type, different `PayloadExamples`** — error
  (D7). Identical sets are idempotent.
- **A `dependencies` entry written in the `[string]` shorthand** — not
  expressible (D4). Rewrite it as the equivalent schema form
  (`{required: [...]}`), which the model does express.
- **Multi Format node carrying a new keyword** — already caught:
  `jsonSchemaKeywords` reflects over all fields, so no edit is needed, and the
  stage-1 test proves it.
- **Multi Format node nested under `if`/`contains`/`patternProperties`** — caught
  **after** the `schemaChildren` extension; a regression test pins it.
- **`minItems=abc`, `const=`, `examples=` (empty)** — ignored, per D6. The empty
  forms set nothing rather than emitting a zero value.
- **`minItems=0`** — a `*uint64` pointing at `0`; emitted as `minItems: 0`,
  which is meaningful ("no minimum" is the absence of the keyword).
- **`const=0` / `const=false` as a tag** — parsed as the string `"0"`/`"false"`,
  not the number/boolean. A tag cannot express the type; use
  `spec.SchemaProvider` for typed constants. Documented.
- **`additionalItems` without tuple `items`** — emitted and inert (D10).
- **A stale `asyncapi:"example=…"` tag** — the directive is gone (D2), so it is
  silently ignored like any unknown directive. Nothing is emitted and nothing is
  reported; O2 is the fix for the whole class. A test pins this.
- **A document using none of the 23 keywords** — byte-identical output; the
  integration golden fixtures are the proof.

## Rollout plan

One PR. The stages below are the logical increments of the change rather than
separate commits — the repository squash-merges every PR, so intra-branch
granularity is review-only. The `Example` removal D2 calls for landed as a
follow-up commit on the same PR.

1. **Stage 0 — object model** (`feat(spec)`): the 23 fields, per-keyword
   round-trip tests, and the `spec.Schema` wire-name pin.
   No behavior change for existing catalogs.
2. **Stage 1 — validation recursion** (`fix(spec)`): `schemaChildren`
   extension plus the nested-Multi-Format regression test. Fixes a gap the
   stage-0 fields would otherwise open.
3. **Stage 2 — tag directives** (`feat(schema)`): `applyTag` and its doc
   comment, table-driven `schema/tags_test.go` cases.
4. **Stage 3 — Message DSL** (`feat(dsl)`): `PayloadExamples`, resolution,
   `message_test.go` cases.
5. **Stage 4 — docs** (`docs`): design doc accepted, ADR 0006, the two README
   indexes, `docs/asyncapi-3.1.0-coverage.md` (§1, §4.2, §4.3, §7, §8 B7), and
   the `schema-composition.md` deferred-`discriminator` note.

## Testing plan

- **Stage 0** — `spec/encode_test.go`, table-driven:
  - `should_round_trip_<keyword>` × 23, each asserting the emitted key in YAML
    **and** JSON and the decoded field.
  - `should_emit_discriminator_as_a_scalar_string` — guards against a reviewer
    "correcting" it into an OpenAPI-style object.
  - `TestEncodeSchemaExamples` — the removed singular `example` key is not
    emitted, and the `examples` array survives the round-trip.
  - `TestSchemaFieldsMatchSpec` — the full declaration-ordered wire-name list,
    sibling of `TestStructFieldsMatchSpec` (`spec/encode_test.go:778`), reusing
    `specFields` (`:755`).
- **Stage 1** — `doc_test.go`:
  - `should_reject_multi_format_node_nested_in_if`
  - `should_reject_multi_format_node_nested_in_pattern_properties`
  - `should_descend_into_dependencies_schema`
- **Stage 2** — `schema/tags_test.go`, table-driven: one case per directive, plus
  `should_ignore_unknown_directive` and `should_ignore_non_numeric_bound`.
  `derive_test.go` gains one end-to-end case on a tagged struct.
- **Stage 3** — `message_test.go`:
  - `should_put_examples_on_the_hoisted_component`
  - `should_put_examples_on_inline_payload`
  - `should_error_when_payload_ref_is_unresolvable`
  - `should_error_when_two_messages_disagree_on_shared_component_examples`
  - `should_accept_identical_examples_from_two_messages`
- **Full gate** — `go build ./...`, `make test` (gotestsum + coverage threshold;
  `test/integration` runs the real AsyncAPI CLI in Docker and asserts no golden
  diff), `make lint`. Race is on by default in `make test`.

## Open / deferred

- **O1 — tuple-form `items`.** Widen `Items` to a union/array so
  `additionalItems` has meaning. Needs the same "one struct, sorted-key
  round-trip" reasoning as the Multi Format decision.
- **O2 — tag value validation.** Surface `minItems=abc` — and a directive that
  no longer exists, such as the removed `example=` (D2) — as a
  discovery/generator error rather than ignoring it. Requires an error channel
  through `FromType`/`fillFields`/`applyTag`, or a static tag-lint pass in
  `internal/discovery`.
- **O4 — presence-aware `Const`/`Default`.** Apply the B8 fix to both
  any-valued fields so `const: false` and `default: 0` survive.
- **O5 — derive structural keywords.** `if=Type`, `contains=Type`,
  `patternProperties=…` via the `combinatorNames` mechanism. Needs hoisting rules
  for schema-valued keywords that have no counterpart today.
- **O6 — the `dependencies` shorthand.** Model the `[string]` property-dependency
  form, if a document ever needs the shorthand rather than its schema-form
  equivalent (D4).
