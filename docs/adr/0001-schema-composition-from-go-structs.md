# 0001. Derive allOf / oneOf / anyOf composition from Go structs

- Status: accepted
- Deciders: asyncgo maintainers
- Created: 2026-08-18
- Status updated: 2026-08-18

## Context and Problem Statement

The generator derives JSON Schemas from Go structs via reflection, but only
models the *flat* cases: fields become properties, embedded structs are
flattened, and named types are hoisted into `components.schemas` behind `$ref`.
There is no way to express schema-level composition — an embedded base kept as a
shared `$ref`, or a field that is a tagged union of message payload types —
which are the natural JSON Schema representations of Go struct embedding and
discriminated message envelopes.

The object model (`spec.Schema`) already carries `OneOf`, `AllOf`, and `AnyOf`;
the derivation and discovery passes simply never populate them.

## Decision Drivers

- Backward compatibility: flattening must remain the default so existing
  committed `asyncapi.yaml` artifacts do not change.
- Deterministic output: identical inputs must produce identical documents.
- Zero user boilerplate: the generator, not the user, discovers referenced
  types and emits the harness registrations.
- Reflect the two-pass architecture: reflection (`schema.FromType`) cannot see
  comments or resolve a name string to a `reflect.Type`, so the static
  `internal/discovery` pass must resolve and inject what reflection cannot.
- Draft 07 correctness: AsyncAPI 3.x Schema Objects are a superset of JSON
  Schema Draft 07, so `$ref` siblings and per-member `required` behave per that
  dialect.

## Considered Options

- Tag-driven combinators plus a global registry (`Register`/`Finalize`) populated
  by generated harness code.
- Threading a registry parameter through `Spec`, the builder, and `FromType`.
- Enumerating interface implementers to synthesize unions automatically.

## Decision Outcome

Chosen option: "Tag-driven combinators plus a global registry", because it keeps
`schema.FromType` self-contained (names are resolved as pure strings to `$ref`s)
while the harness defers hoisting until after package `init`, sidestepping Go's
package-init ordering without touching the public DSL.

### Consequences

- Good, because flattening stays the default and `allOf` from embedding is
  opt-in, so existing output is unchanged.
- Good, because `Register`/`Finalize` are an internal implementation detail the
  generator drives; user code never calls them.
- Good, because the registry is idempotent and `Finalize` derives in sorted-FQN
  order, preserving determinism.
- Bad, because a package-level global registry is shared mutable state rather
  than explicit plumbing through the DSL.
- Bad, because union members must be declared explicitly by type name; sealed
  interfaces and discriminators remain unimplemented (deferred).

## More Information

- Design doc: docs/designdoc/schema-composition.md
- Raw notes: docs/scratch/schema-composition-brainstorm.md (gitignored)
