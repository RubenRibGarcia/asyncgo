# 0004. Fluent DSL validation with a SpecResult type

- Status: accepted
- Deciders: asyncgo maintainers
- Created: 2026-09-03
- Status updated: 2026-09-03

## Context and Problem Statement

The fluent DSL (`asyncgo`) cannot report that a catalog was assembled
incorrectly. `Item.apply` returns nothing and `Spec` returns only a
`*spec.AsyncAPI`, so invalid catalogs — e.g. a server missing its required
`host` — silently produce invalid AsyncAPI documents. Validation errors must
reach generation time (`asyncgo generate` / `asyncgo check`), but the harness
only materializes already-built catalog variables, and a Go package-level `var`
holds a single value.

## Decision Drivers

- Errors are returned, never panicked (library convention).
- The catalog must remain a single package-level variable discoverable by type.
- Violations are reported per catalog with the specific offending fields named.
- Required fields (AsyncAPI 3.1.0: `info.title`, `info.version`, `server.host`,
  `server.protocol`) are enforced.

## Considered Options

- `SpecResult` wrapper type (`*asyncgo.SpecResult{Doc, Err}`)
- Catalog as a function returning `(*spec.AsyncAPI, error)`
- Two exported variables (`Catalog`, `CatalogErr`)
- A separate `Validate(doc) error` pass

## Decision Outcome

Chosen option: "`SpecResult` wrapper type", because it keeps the catalog a
single value while carrying both the document and its validation errors to the
harness.

- `Item.apply` returns `error`; `Spec` returns `*SpecResult`.
- `SpecResult` has exported `Doc`/`Err` plus `ValidationErrors() []error` for
  the per-catalog report.
- `server.host` becomes a required constructor argument
  (`Server(name, protocol, host)`).
- Discovery detects `*asyncgo.SpecResult`; the harness emits a structured
  per-catalog envelope and `Materialize` returns a `CatalogErrors` report.

- Option "Catalog as a function" was rejected because it changes the discovery
  contract from "value" to "call".
- Option "Two exported variables" was rejected because it leaks an exported
  error variable and needs fragile name-pairing.
- Option "Separate Validate pass" was rejected because it loses the apply-time
  context the returned errors must carry.

### Consequences

- Good, because invalid catalogs fail generation with a per-catalog,
  per-field report.
- Good, because `host` omission becomes a compile error instead of a silent bug.
- Bad, because the public `Spec` return type changes (acceptable: asyncgo has
  no released version yet).
- Bad, because hand-built `*spec.AsyncAPI` variables are no longer discovered.

## More Information

- Design doc: docs/designdoc/fluent-dsl-validation.md
