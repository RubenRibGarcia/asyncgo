# Brainstorm: mapping allOf / oneOf / anyOf from Go structs

> Scratch notes. Not a design doc — see `docs/designdoc/schema-composition.md`
> for the distilled, decision-oriented version. This folder is gitignored.

## Premise

Go struct embedding is a natural fit for sharing common attributes. AsyncAPI's
Schema Object is a superset of **JSON Schema Draft 07** (confirmed: AsyncAPI
3.0/3.1 spec, §Schema Object), so `allOf` composition is a first-class, expected
mechanism (the spec literally says *"combining and extending model definitions
using the `allOf` property of JSON Schema, in effect offering model
composition"*). `oneOf` and `anyOf` cover the union cases.

## Ground-truth corrections found while investigating

1. `spec/schema.go` doc comment says `Draft 2020-12` — **wrong**, it's Draft 07.
2. `Defs map[string]*Schema`json:"$defs"`` uses `$defs`, a 2019-09/2020-12
   keyword; Draft 07 calls it`definitions`. It's never emitted today (reuse
   goes through`components.schemas`), so it's latent but should be renamed.
3. `OneOf`/`AllOf`/`AnyOf`/`Not` are already modeled in `spec.Schema` but never
   populated — the model is ready, only derivation + discovery are missing.

## The architectural fact that decides everything

Schema derivation runs in two disjoint passes:

| Pass | When | Sees | Can't see |
|------|------|------|-----------|
| `schema.FromType` (reflection) | harness `go run`, separate process | `reflect.Type`, tags, embedded field types | comments, import aliases, name→type resolution, interface implementers |
| `internal/discovery` (`go/packages`+`go/types`+AST) | static, pre-materialization | doc comments, type relations, ident→`*types.TypeName`, interface implementer sets | cannot run reflection; works on materialized YAML |

The descriptions feature is the precedent: discovery computes field→FQN→comment
statically and injects it post-materialization.

**Key asymmetry:**

- `allOf` from **embedding** is reflection-resolvable *today* — the embedded
  field already arrives as a `reflect.Type`, so `FromType` can hoist it and emit
  `allOf: [{$ref Base}, {…own fields}]` with **no static pass**.
- `oneOf`/`anyOf` from a **tag** (`oneOf=OrderPlaced`) is *not*
  reflection-resolvable — reflection has no name→type registry and can't
  enumerate interface implementers.

## Combinator-by-combinator exploration

### allOf (embedding)

Today `fillObject` **flattens** anonymous embedded structs (matching
`encoding/json`): `OrderPlaced{ BaseSchema; Amount }` → flat
`{"id", "amount"}`. Wire-accurate.

`allOf` mapping would instead be:

```yaml
OrderPlaced:
  allOf:
    - $ref: "#/components/schemas/...BaseSchema"
    - type: object
      properties: { amount: {...} }
      required: [amount]
```

Trade-offs:

- **Pro**: preserves shared-base relationship (DRY); `BaseSchema` becomes a
  reusable component; mirrors the OO intent.
- **Con**: schema-level composition over a flat wire format → consumers must
  resolve `allOf` (tooling varies on merging `required`).
- **Con**: Go field promotion/shadowing rules differ from `allOf` merge
  semantics (a shadowed embedded field is ambiguous in `allOf`).

Leaning: **flatten default, `allOf` opt-in via a marker**.

Latent `encoding/json` nuance: an anonymous field *with* a JSON name tag
(`BaseSchema`json:"base"``) is **not** flattened by `encoding/json` (it nests).
`fillObject` currently flattens it regardless — reconcile either way.

### oneOf / anyOf

Go has no sum types. Three mechanisms, increasing magic:

- **A. Tag + type names** — `asyncapi:"oneOf=OrderPlaced|OrderShipped"`.
  Explicit, matches existing `enum=`/`format=` style. The hard part is
  name→type resolution.
- **B. Sealed-interface idiom** — `type Event interface{ event() }`; static pass
  enumerates implementers. Zero boilerplate, but static-pass-only and needs a
  sealed interface for determinism. Stretch.
- **C. Explicit registration** — `asyncgo.Register(A{}, B{})`. Simplest, but
  boilerplate, contradicts "derive from real structs".

### The hoisting coordination problem (sharpest edge)

`oneOf=OrderPlaced` on an `any` field: reflection sees `any`, emits `{}`, never
visits `OrderPlaced`, so it's absent from `components.schemas`. Whoever resolves
the `oneOf` must also arrange hoisting. → Resolution must live in the reflection
pass (fed by a registry), not as post-materialization schema surgery.

## Resolution (how name→type is solved)

Tag values resolve to FQN, then to a `$ref` by pure string manipulation (no type
needed at emit time); the *hoisting* of those types is deferred to a
`Finalize` step in the harness, which has a registry populated by generated
`Register` calls. This sidesteps Go's init-order problem entirely — see the
design doc.

## Open questions (answers recorded in the design doc)

1. Embedding default: flatten-only vs flatten-default-with-opt-in-`allOf`.
2. Discriminator alongside `oneOf` (event envelopes) — in v1 or deferred?
3. `required` semantics inside `allOf` — local-per-member vs hoisted.
4. Registry surface — global `Register`+`Finalize` vs threading through `Spec`.

## Candidate staged plan

1. Fix spec groundwork (Draft 07 comment, `definitions` rename).
2. `allOf` from embedding — pure reflection, no static pass.
3. `oneOf`/`anyOf`/`allOf` via tag + registry + harness `Register`/`Finalize`.
4. Deferred: discriminator, sealed-interface enumeration, cross-package sugar.
