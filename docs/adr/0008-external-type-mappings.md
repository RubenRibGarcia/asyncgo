# 0008. Derive third-party types from declared, name-keyed mappings

- Status: accepted
- Deciders: asyncgo maintainers
- Created: 2026-11-08
- Status updated: 2026-11-08

## Context and Problem Statement

A type with a wire format that reflection gets wrong has one way to say so:
implement `spec.SchemaProvider` (ADR 0002). That requires a method *on the type*,
so it cannot be done for a type the project does not own. `gofrs/uuid.UUID` is a
named `[16]byte`, so it derives as an array of integers, and the only workaround
is a wrapper type in every payload signature.

Two facts about the generator constrain any fix. Schemas are derived in a
generated harness process, and that process derives twice: a message payload while
the catalog package's variables are initialized, and the combinator-registered
types in `schema.Finalize`. A declared mapping registered by the project — in an
`init()` or by the harness — is therefore not in place for the payload that most
needs it. Reflection also cannot walk `type OrderID uuid.UUID` back to
`uuid.UUID`, so alias inheritance has to be resolved statically, where full type
information exists.

## Decision Drivers

- The extension point must work for a type nobody can add a method to.
- A mapping must be declarable without adding a dependency on the library it
  describes (`gofrs/uuid`, `shopspring/decimal`, …) to asyncgo or its consumers.
- Two commands (`generate`, `check`) must agree on the source of truth; per-run
  flags invite drift.
- Existing documents must not change unless a project opts in.
- The project's document is the one being published: a deliberate mapping should
  outrank a type's suggested default, but not the other way round.

## Considered Options

- **Declared mappings keyed by fully-qualified type name** — a small Go API
  (`schema.MapSchema`, `schema.MapSchemaByName`), an `asyncgo.yaml` file for
  simple `{type, format}`-shaped schemas, and a built-in table for well-known
  third-party types.
- Keep wrapper types and generated code as the supported workaround.
- Kind-based hooks (`[]byte`, all `uint8` slices) instead of exact-name keying.
- A field-level mapping in the `asyncapi` tag (`asyncapi:"map=..."`).

## Decision Outcome

Chosen option: "Declared mappings keyed by fully-qualified type name", because it
is the only option that reaches a third-party type without a method, without an
import, and without touching every field that uses it.

Concretely:

- **One registry, two entry points.** `schema.MapSchema[T]` (type-safe, rich
  schemas) and `schema.MapSchemaByName(fqn, schema)` (name-keyed, what the config
  file compiles to) write the same `map[string]*spec.Schema`, keyed
  `pkgPath.TypeName` like `schema.Register` already is. Re-registration replaces,
  so the tier order is built-in < Go API < `asyncgo.yaml`, deterministically.
- **Precedence.** Exact mapping > `SchemaProvider` > mapping inherited through a
  derived alias > kind derivation. An exact mapping is a document-level override,
  so it outranks the type's suggested default; an inherited mapping is implicit,
  so it does not.
- **Inline at every use site**, like `time.Time`, with one exception: a mapped
  type a combinator directive names is hoisted by `Finalize` so its `$ref`
  resolves. `time.Time` becomes an ordinary built-in mapping, deleting the
  exact-type special case.
- **`asyncgo.yaml` at the module root** carries a single whitelisted `mappings:`
  section (scalar annotations and constraints only; `$ref`, nested objects and
  combinators are rejected at discovery time). `internal/mappingconfig` is the one
  parser, shared with the embedded built-in table, so a built-in is exactly a
  config entry.
- **Alias inheritance is emitted, not guessed.** Discovery walks the module's `type
  X Y` declarations and emits one `schema.MapAlias(X, Y)` per declaration; chains
  stop at the module boundary, and Go type aliases need no edge.
- **Unused mappings warn on stderr.** Name-keying fails silently by nature; a
  registered but never-consulted user mapping is reported, which catches a typo'd
  FQN. Built-ins are exempt.
- **Payloads are derived provisionally and re-derived by `Finalize`.** Because a
  payload is derived before any mapping can register, `schema.FromTypePayload`
  records the derivation and `Finalize` redoes it with every tier in place,
  rewriting the node in place, replacing the components it hoisted, dropping the
  ones a mapping made obsolete, and carrying the payload's examples across. This
  keeps `Spec`'s validation, collision checks and `PayloadExamples` where they
  are, and leaves a document complete for callers that never finalize.

## Consequences

- **Good**: `uuid.UUID`, `ulid.ULID`, `decimal.Decimal` and `big.Int` derive
  correctly with no configuration and no new dependency.
- **Good**: one extension point covers third-party types and a project's own
  aliases; a derived alias inherits without a second declaration.
- **Good**: a document does not change unless a mapping applies, and a stale
  mapping is reported rather than silently ignored.
- **Bad**: three registration paths to keep in step (built-ins, API, config) and a
  documented ordering rule that only exists because they are all last-wins.
- **Bad**: payloads are derived twice per run.
- **Bad**: `Finalize` must run on each catalog document before it is merged, since
  the deferred payload record is attached to the document object.
- **Neutral**: the built-in table is data, so growing it is a data change — but
  every entry has to be verified against the library's `encoding/json` behavior.
- **Neutral**: kind-based mapping (all `[]byte`, all `uint8` slices) remains out of
  scope; exact-name keying is deliberate.

## More Information

- Design doc: [`docs/designdoc/external-type-mappings.md`](../designdoc/external-type-mappings.md)
- Tracking issue: [#54](https://github.com/RubenRibGarcia/asyncgo/issues/54)
- Related: [ADR 0002](0002-custom-schema-providers.md) — the type-owned extension
  point this fills the gap beside
