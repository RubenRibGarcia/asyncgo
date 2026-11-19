# 0015. Derive a null branch for nilable types

- Status: accepted
- Deciders: asyncgo maintainers
- Created: 2026-11-19
- Status updated: 2026-11-19

## Context and Problem Statement

`schema.FromType` strips pointer layers before deriving, so a field, array item
or map value whose Go type can hold `nil` derives a schema that rejects `null`:
`Note *string` and `Note string` both produce `{type: string}`. `encoding/json`
marshals a nil pointer, slice or map as `null` and accepts `null` on unmarshal
for all of them, so the generated document rejects instances the payload type
admits — including the ones asyncgo writes itself:

```go
type P struct{ Note *string `json:"note"` }   // nil Note
json.Marshal(P{})                              // {"note": null} — schema says invalid
```

The forces at play:

- **The dialect.** AsyncAPI 3.1.0's Schema Object is `allOf: [JSON Schema
  draft-07, {x-…}]`. Draft-07 `type` accepts an array of type names and `"null"`
  is one of them, but Draft-07 **ignores keywords beside `$ref`**, so a pointer
  to a hoisted struct cannot carry the null next to its reference.
- **`nullable` is not available.** It was an AsyncAPI 2.x / OpenAPI 3.0 keyword.
  A 3.1.0 validator ignores it as an unknown keyword, so emitting it would
  document nothing to a conforming consumer.
- **Byte-stable goldens.** The `asyncgo check` gate compares bytes, and the
  discovery harness round-trips every document through YAML, so the emitted
  shape has to be expressible by the existing `spec.Schema` struct without a
  custom codec.

## Decision Drivers

- Spec fidelity: the document must cover what the Go type accepts.
- One representation, not two: primitives and references need the same shape.
- No public API change if the model can already express the result.
- Deterministic, minimal output for the types that are not nilable.
- Pre-1.0 freedom to fix a wrong default rather than layer a flag over it.

## Considered Options

- **`oneOf: [<schema>, {type: "null"}]`** at every nilable position.
- **`anyOf: [<schema>, {type: "null"}]`** — equally correct, weaker intent.
- **`type: ["T", "null"]`** for inlined kinds, `anyOf` only where `$ref` forces
  it.
- **`nullable: true`** beside the schema.
- **An `asyncapi:"nullable"` opt-in directive**, leaving pointers unchanged.

## Decision Outcome

Chosen option: "`oneOf: [<schema>, {type: "null"}]`", applied to every nilable
kind rather than only to pointers.

- **`anyOf`** was rejected only on intent: the branches are disjoint, so `oneOf`
  is exact rather than merely permissive.
- **`type: ["T", "null"]`** was rejected because it needs two shapes anyway: a
  `$ref` cannot take a type union, so references would still need a combinator.
  Producing one shape for primitives and another for references would also
  require widening `spec.Schema.Type` from `string` to a string-or-array type —
  a public API break with roughly 250 call sites — and the trigger (a Go pointer)
  is uniform, so the output should be too.
- **`nullable: true`** was rejected as non-conformant: it validates only because
  unknown keywords are ignored.
- **An opt-in directive** was rejected because the current output is a defect,
  not a preference, and a flag would leave the default wrong.

### Consequences

- Good, because the document now covers every instance `encoding/json` accepts
  for the payload type, including the null asyncgo itself emits.
- Good, because no `spec.Schema` field and no public symbol changed: `OneOf` was
  already modelled.
- Good, because nullability is a property of the *position*, so
  `components.schemas` entries and the `FromType` result stay bare and `*T`
  shares `T`'s component; `Message.PayloadExamples` keeps resolving through the
  payload's `$ref`.
- Good, because a tag keyword (`enum`, `format`, `minItems`) is applied to the
  payload branch, so it constrains the value instead of silently excluding null.
- Bad, because it is a breaking change: every document with a nilable position
  changes shape (`anyof`, `oneof`, `bytes` and `mappings` fixtures, plus any
  user document with a pointer, slice or map field). Accepted deliberately while
  the library is pre-1.0; the change carries a `BREAKING CHANGE:` footer.
- Bad, because the wrapper is not free: a plain `*string` field goes from one
  mapping to a `oneOf` pair, which is noisier for a human reading the document
  and for a code generator that does not flatten single-branch combinators.
- Bad, because a schema attached by hand with `asyncgo.Schema` is not bound by
  this rule: derivation is the only producer of the wrapper, so an authored
  component that ought to be nullable is still the author's job.
- Bad, because the guard that skips an already-unconstrained payload has to look
  through one level of `$ref`; a provider that delegates to another reference
  rather than returning a schema is treated as constraining, so a null branch is
  emitted where none was needed. The opposite error — wrapping `{}` — would make
  the position accept nothing, so the conservative side was chosen.

## More Information

- Design doc: docs/designdoc/nullable-derivation.md
- Related: [0001](0001-schema-composition-from-go-structs.md) — the derivation
  and hoisting model this extends; [0006](0006-schema-keywords.md) — modelled
  `oneOf` and the "no keywords beside `$ref`" constraint that forces a
  combinator here.
- Coverage: [§4](../asyncapi-3.1.0-coverage.md#4-schema-derivation).
