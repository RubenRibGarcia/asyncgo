# Brainstorm: custom schema providers for types with custom (de)serialization

> Scratch notes. Not a design doc — see `docs/designdoc/custom-schema-providers.md`
> for the distilled, decision-oriented version.

## Premise

A developer can customize how a struct marshals/unmarshals (via `MarshalJSON`,
`UnmarshalJSON`, `encoding.TextMarshaler`, …). When they do, the wire format
diverges from the Go field shape, so `schema.FromType`'s reflection mapping is
wrong. The schema must become a *declaration*, not a derivation. The question is
the surface.

## Ground-truth facts that decide everything

1. Reflection **can** detect interface implementation (`reflect.Type.Implements`).
   So a method hook is pure reflection — same category as `allOf`-from-embedding,
   **not** `oneOf`-from-tag (which needed `Register`/`Finalize`).
2. `FromType` derefs pointers before dispatch, so pointer-receiver methods are
   lost unless probed via `reflect.PointerTo(t)` (mirror `encoding/json`'s
   value-then-addressable rule).
3. `FromType` sees a `reflect.Type`, not a value → the hook runs on a **zero
   value** → the method must be type-level/static, not instance-level.
4. Named structs all funnel through `hoistStruct` → one interception point,
   scoped to named structs for free, preserving FQN hoisting + `$ref`.
5. The harness deliberately avoids user-code execution beyond `init` → calling
   the method is a new, explicit trade-off.

## Mechanism exploration

- **A. Interface → `*spec.Schema`** (`AsyncAPISchema() *Schema`) — full JSON
  Schema power, most stdlib-idiomatic, but executes user code. **Chosen.**
- **B. Interface → wire type** (`AsyncAPISchemaType() reflect.Type`) — reuses
  derivation engine, minimal duplication, less expressive. Rejected: less
  powerful, still executes user code, and returns-a-type is a leakier contract.
- **C. Catalog registration** (`asyncgo.Schema(Money{}, …)`) — no user-code
  execution, deterministic, but boilerplate contradicts "derive from structs".
  Rejected on ergonomics.
- **D. Struct tag** (`asyncapi:"type=string"`) — field-level only, can't
  redefine a whole struct. Rejected for the stated use case (customized struct);
  noted as a possible follow-up for single-field overrides.

## Where the interface lives

`spec` vs `schema`. Chose `spec`: user catalogs already import `spec` (headers,
examples, bindings), so zero new imports, and the method returns `*Schema`
in-package. `schema` would force a second import just to name the interface.

## The zero-value / determinism contract

Because the hook runs on a zero value, `AsyncAPISchema()` must not read receiver
state, must be pure, deterministic, and panic-free. This is the sharpest
semantic difference from `json.Marshaler` and must be documented up front.

## Open questions (answers recorded in the design doc)

1. Mechanism surface → interface returning `*spec.Schema`.
2. Scope → named structs only.
3. Nil return → fall back to reflection (no error channel in the interface).
4. Marshal vs unmarshal → single hook; JSON Schema describes the shared wire
   contract.
5. `$ref` inside a returned schema → not auto-hoisted; document + reuse
   `Register`/`Finalize` when needed.
6. Panic recovery / validation → deferred (document-only in v1).
