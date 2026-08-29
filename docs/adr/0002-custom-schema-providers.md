# 0002. Custom schema providers for types with custom (de)serialization

- Status: accepted
- Deciders: asyncgo maintainers
- Created: 2026-08-29
- Status updated: 2026-08-29

## Context and Problem Statement

The generator derives JSON Schemas from Go structs via reflection. That mapping
is only correct while the Go field shape matches the JSON wire shape. A type
with a custom `MarshalJSON`/`UnmarshalJSON` (or `encoding.TextMarshaler`) breaks
the contract: reflection derives an object from the struct's fields, but the
wire format is something else (e.g. a single string).

Once a developer customizes serialization, the schema is no longer *derivable* —
it must be **declared**. The project needs a surface for that declaration.

## Decision Drivers

- Idiomatic surface: mirror the stdlib (`json.Marshaler`) so the feature feels
  native to Go developers.
- Zero new imports for the common case: user catalogs already import `spec`.
- Preserve hoisting: custom schemas must remain referenced via `$ref` under
  `pkgPath.TypeName`.
- No discovery/harness change: interface satisfaction is visible to reflection.
- Deterministic output and backward compatibility for types that do not opt in.

## Considered Options

- Interface returning `*spec.Schema` (`spec.SchemaProvider`, method
  `AsyncAPISchema() *Schema`).
- Interface returning a wire type (`AsyncAPISchemaType() reflect.Type`).
- Catalog-side registration (`asyncgo.Schema(Money{}, …)`).
- Struct-tag field override (`asyncapi:"type=..."`).

## Decision Outcome

Chosen option: "Interface returning `*spec.Schema`", because it gives full JSON
Schema expressiveness with a stdlib-idiomatic surface while requiring no
discovery pass and no new imports, at the accepted cost of executing user code
during generation.

- The wire-type option was rejected because it is less expressive and leaks a
  `reflect.Type` contract, while still executing user code.
- Catalog registration was rejected because it reintroduces per-type boilerplate
  that contradicts "derive from real structs".
- The struct-tag option was rejected because it is field-level only and cannot
  redefine a whole struct.

### Consequences

- Good, because a named struct can declare any schema (scalar, object,
  combinator) and it is hoisted under its FQN, preserving `$ref` resolution.
- Good, because detection is pure reflection (`reflect.Type.Implements` on `t`
  and `*t`), composing with the existing `Register`/`Finalize` machinery with no
  changes.
- Good, because `nil` versus `&spec.Schema{}` is a clean sentinel split:
  "derive for me" versus "explicitly unconstrained".
- Bad, because the method runs on a zero value, so it must be type-level, pure,
  deterministic, and panic-free — a contract that differs from `json.Marshaler`
  and must be documented.
- Bad, because it executes user code during generation (the first such execution
  beyond package `init`).
- Bad, because `$ref`s referenced inside a returned schema are not auto-hoisted;
  developers must register those types.

## More Information

- Design doc: docs/designdoc/custom-schema-providers.md
- Raw notes: docs/scratch/custom-schema-providers-brainstorm.md
