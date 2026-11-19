# Design: nilable types derive a null-accepting schema

- **Status**: Accepted
- **Created**: 2026-11-19
- **Status updated**: 2026-11-19
- **Scope**: `schema/`, `test/data/nullable/`, `test/integration/`, `docs/`, `site/`

## Summary

A struct field, array item or map value whose Go type can hold `nil` currently
derives a schema that rejects `null`. `encoding/json` emits `null` for a nil
value and accepts `null` on unmarshal, so the generated document rejects
instances the payload type itself permits — including instances asyncgo writes
itself. Every nilable position now derives `oneOf: [<schema>, {type: "null"}]`.

## Background

### The defect

`schema.FromType` strips pointer layers on entry (`schema/derive.go`, `deref`),
so a pointer and its element derive the identical schema:

| Go | derived before | required |
| --- | --- | --- |
| `Note *string` | `{type: string}` | `oneOf: [{type: string}, {type: "null"}]` |
| `Customer *Customer` | `{$ref: …Customer}` | `oneOf: [{$ref: …}, {type: "null"}]` |
| `Tags []string` | `{type: array, items: …}` | null branch around the array |
| `Attrs map[string]string` | `{type: object, additionalProperties: …}` | null branch around the object |

```go
type P struct{ Note *string `json:"note"` }   // nil Note
json.Marshal(P{})                              // {"note": null}
// derived schema: {note: {type: string}}      // rejects the line above
```

`omitempty` is not the discriminator: it decides only what asyncgo writes. A
consumer sending `{"note": null}` is accepted by the Go type whether or not the
tag is there, so the document has to admit it either way.

Nothing in the repository handled this: no `nullable` anywhere, no pointer case
in `schema/derive_test.go`.

### Dialect facts

- AsyncAPI 3.1.0's Schema Object is `allOf: [JSON Schema draft-07, {x-…}]`
  (`asyncapi/spec-json-schemas` reuses `definitions/3.0.0/schema.json`), so
  Draft-07 semantics govern.
- Draft-07 `type` is a string **or** an array, and `"null"` is one of its seven
  values. `oneOf`/`anyOf` members are ordinary schemas.
- Draft-07 **ignores keywords beside `$ref`** — the constraint already recorded
  in `schema/derive.go` and in ADR 0006 — so nullability cannot be written next
  to a `$ref` and needs a combinator.
- `nullable` is not a Draft-07 or 3.1.0 keyword (it was AsyncAPI 2.x / OpenAPI
  3.0). It would validate only because unknown keywords are ignored, i.e. it
  documents nothing to a conforming consumer.

## Goals / Non-goals

**Goals**

1. A nilable position accepts `null`, so the document covers every instance
   `encoding/json` accepts for the payload type.
2. `components.schemas` output stays byte-stable for types that are not nilable;
   only documents with a nilable position change.
3. No new public API and no new `spec.Schema` field.

**Non-goals (v1)**

- Changing optional-by-default (`AGENT.md` decision 3). A non-pointer field
  without `omitempty` is always emitted, so strictly it should be `required`;
  that is a separate, larger inference change.
- An opt-out directive (`asyncapi:"nonNullable"`). The tag vocabulary is closed
  and a new directive needs `taggrammar` + `CheckTag` + docs.
- Tuple-form `items`, or any change to how embedded structs are flattened.

## Design decisions

### D1 — `oneOf` with `{type: "null"}`, not `anyOf`, not `nullable`

Every nilable position emits `oneOf: [<schema>, {type: "null"}]`.

**Rationale**: the null branch is disjoint from every non-null branch by
construction, so exactly-one holds and `oneOf` is exact rather than merely
permissive. `anyOf` would be equivalent here but weaker in intent; `nullable`
does not exist in 3.1.0 (see Background).

### D2 — every nilable kind, not just pointers

Pointer, slice, map, interface, chan and func all derive a null branch — the six
Go kinds that can marshal to `null`.

**Rationale**: a nil slice in a struct without `omitempty` marshals as `null`
exactly like a nil pointer. Fixing only pointers leaves the same defect in place
one field over. The two kinds that derive an unconstrained `{}` (`interface` and
the `chan`/`func`/`complex`/`uintptr` group) are handled by D5's guard rather
than by a narrower rule, so the set of kinds is complete and the guard decides.

### D3 — a breaking default

The derivation changes for every document that has a nilable position: no flag,
no `asyncgo.yaml` switch.

**Rationale**: the current output is a defect, not a preference — it rejects the
library's own runtime output. Pre-1.0 is when the break is cheapest, and a flag
would double every fixture and leave the default wrong.

### D4 — nullability belongs to the position, never to the type

The `FromType` result and every `components.schemas` entry stay non-nullable.
Exactly three positions add the wrapper: a struct field, `items`, and
`additionalProperties`.

**Rationale**: `*T` and `T` share one hoisted component. A nullable component
would poison every non-nilable use of the same type. Keeping the root bare also
keeps a `MessageOf[*T]` payload a plain `$ref`, which is what
`Message.PayloadExamples` resolves through — a nullable payload node would have
moved examples onto the wrapper and out of the component they belong to.

### D5 — never wrap a schema that already admits null

The wrapper is skipped when the payload branch constrains nothing, including
when it constrains nothing only because it is a `$ref` to an empty component.

**Rationale**: `oneOf: [{}, {type: "null"}]` matches *every* instance against
two branches, so exactly-one fails and the position accepts nothing at all. That
is worse than the bug being fixed, and it is silent. The `$ref` case matters
because a provider returning an unconstrained schema is hoisted as an empty
component while the use site holds only the reference.

### D6 — tag keywords land on the payload branch

`fillFields` derives, applies the tag, applies combinator directives, and only
then wraps.

**Rationale**: `enum`, `format`, `minItems` and friends restrict an instance.
Written beside the `oneOf` they would intersect with the whole instance, so
`enum=web,mobile` on a nullable field would exclude the `null` the wrapper had
just added.

### D7 — combinator directives nest

`asyncapi:"oneOf=A,B"` on a nilable field yields
`oneOf: [{oneOf: [A, B]}, {type: "null"}]`.

**Rationale**: merging the null branch into the directive's member list is
equivalent for `oneOf` and `anyOf` — the null member is disjoint, so
exactly-one still holds — but it is **wrong** for `allOf`, where every member
must pass and a null member would make the position unsatisfiable. Nesting is
uniform across the three and correct for all of them.

### D8 — embedded fields are unaffected

Flattened fields and `asyncgo:"allOf"` members keep their current shape.

**Rationale**: a nil embedded pointer contributes no keys at all; it does not
emit `null`. There is no null to describe.

## Detailed design

### 1. `schema/derive.go`

Three helpers beside `deref`:

```go
// nullSchema is the Draft-07 schema of JSON null. Never handed out as-is.
var nullSchema = &spec.Schema{Type: "null"}

func nilableKind(t reflect.Type) bool          // Pointer, Slice, Map, Interface, Chan, Func
func nullable(t reflect.Type, s *spec.Schema, defs map[string]*spec.Schema) *spec.Schema
func constrains(s *spec.Schema, defs map[string]*spec.Schema) bool
```

`nullable` returns `s` unchanged unless `t` is nilable *and* `s` constrains
something; otherwise it returns `oneOf: [s, {type: "null"}]` with a fresh copy
of the null node per position.

`constrains` folds away the annotation keywords (`title`, `description`,
`$comment`, `externalDocs`, `examples`, `default`, `readOnly`, `writeOnly`,
`deprecated`, `discriminator`, `$id`, `$schema`) and compares the rest to the
zero schema, so a keyword added to `spec.Schema` later counts as a constraint by
default — the safe direction, since the worst case is an unnecessary null branch
rather than an unsatisfiable one. A `$ref` is resolved one level through `defs`
(D5).

Three call sites, each passing the **undereferenced** position type, because
`FromType` dereferences on entry:

```go
// FromType, slice/array and map branches
Items:                nullable(t.Elem(), FromType(t.Elem(), defs), defs)
AdditionalProperties: nullable(t.Elem(), FromType(t.Elem(), defs), defs)

// fillFields, after applyTag and the combinator directives (D6)
own.Properties[name] = nullable(f.Type, prop, defs)
```

### 2. Package documentation

`schema/derive.go`'s package comment is the canonical record of the derivation
rules, so the null rule is written there, and `FromType`'s comment states that
its result describes a non-nil instance.

### 3. Fixtures

New `test/data/nullable/` (own module, catalog + committed golden) covering a
pointer scalar, `**string`, pointer-to-hoisted-struct, `[]*T`, `map[string]*T`,
`*[]T`, `*time.Time`, `[]byte`, `map[string]string`, `any`, and
required-and-nullable. Registered in `go.work` and in the `fixtures` slice of
`test/integration/asyncgo_generate_test.go`, so the real `asyncapi validate`
proves `{type: "null"}` against the published 3.1.0 schema.

## Example (before / after)

```yaml
# asyncapi:"required" on Order.Customer, plus a nilable slice and a bare any
properties:
  customer:
    oneOf:
    - $ref: "#/components/schemas/…Customer"
    - type: "null"
  tags:
    oneOf:
    - type: array
      items:
        type: string
    - type: "null"
  channel:                    # *string asyncapi:"enum=web,mobile"
    oneOf:
    - type: string
      enum: [web, mobile]     # the keyword is on the value, not the wrapper
    - type: "null"
  metadata:                   # any — accepts null already, left alone
    description: …
required:
- customer                    # present and possibly null
```

## Edge cases

- Unconstrained payload (`interface{}`, `chan`, `func`, `json.RawMessage`,
  provider returning `{}`) — no wrapper (D5).
- `$ref` to an unconstrained component — resolved one level, no wrapper (D5).
- `[][]string` — outer and inner array both nilable, so both carry a null
  branch; a `string` item does not.
- `**string` — one wrapper, however many pointer layers.
- Named pointer type (`type OrderRef *UUID`) — nullable, because the declared
  type of the position is a pointer.
- Mapped type behind a pointer (`*time.Time`, `*big.Int`) — the mapping's shape
  becomes the payload branch.
- `omitempty` — no effect (D2's rationale).
- Embedded pointer — no wrapper (D8).
- `oneOf`/`anyOf`/`allOf` directive on a nilable field — nested (D7).

## Rollout plan

Single change, `schema` scope: the derivation, its tests, the fixture, the
regenerated goldens, and the docs.

**`fix(schema)!: derive oneOf with null for nilable types`**, with a
`BREAKING CHANGE:` footer naming the four regenerated fixtures and the affected
shapes.

## Testing plan

- `schema/nullable_test.go` — table-driven over one holder struct that places
  every nilable kind in a field position:
  - `should_wrap_a_pointer_scalar`, `should_wrap_a_pointer_once_however_deep`,
    `should_wrap_a_pointer_to_a_hoisted_struct`, `should_wrap_a_slice`,
    `should_wrap_a_byte_slice`, `should_wrap_a_map`,
    `should_wrap_a_pointer_slice_element`, `should_wrap_a_pointer_map_value`,
    `should_wrap_a_pointer_to_a_slice`, `should_wrap_a_pointer_to_a_map`,
    `should_keep_a_mapped_shape_on_the_payload_branch`,
    `should_wrap_a_named_pointer_type`, `should_wrap_despite_omitempty`
  - `should_keep_a_required_pointer_in_required`
  - `should_not_wrap_an_interface`, `should_not_wrap_json_raw_message`,
    `should_not_wrap_a_chan`, `should_not_wrap_a_func`,
    `should_not_wrap_a_non_nilable_scalar`, `should_not_wrap_a_non_nilable_struct`
  - `should_wrap_a_map_whose_values_admit_anything`,
    `should_wrap_a_slice_whose_elements_admit_anything`
  - `should_keep_tag_keywords_on_the_payload_branch`,
    `should_nest_a_combinator_directive`, `should_look_through_a_ref_to_an_empty_component`,
    `should_keep_the_root_and_components_bare`, `should_never_share_the_null_node`
- Existing tests that asserted the un-wrapped shape for `any` fields with
  combinator directives were updated to unwrap through a helper, which is the
  behaviour change they were pinning.
- Integration: the `nullable` fixture is regenerated, compared byte-for-byte,
  codec-checked, and validated by `asyncapi validate` (`asyncapi/cli:6.1.0`) in
  a container, in both YAML and JSON.
- Race: covered by `make test` (`-race`).

## Open / deferred

- Optional-by-default inference (a non-omitempty field always present).
- An opt-out directive for a project that needs the previous shape.
- Whether a *root* payload should ever be nullable. It is not today (D4); a
  null payload is better expressed by omitting `payload`.
