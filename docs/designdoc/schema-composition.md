# Design: allOf / oneOf / anyOf derivation from Go structs

- **Status**: Accepted
- **Created**: 2026-08-15
- **Status updated**: 2026-08-18
- **Scope**: `schema/`, `spec/`, `internal/discovery/`, `cmd/asyncgo`
- **Raw notes**: `docs/scratch/schema-composition-brainstorm.md` (gitignored)

## Summary

Extend struct→JSON Schema derivation so that Go struct composition can map to
the JSON Schema combinators `allOf`, `oneOf`, and `anyOf`:

- **`allOf` from embedding** — an opt-in marker on an anonymous embedded field
  composes the embedded type and the remaining fields via `allOf`, instead of
  the current flat flattening.
- **`oneOf` / `anyOf` / `allOf` from a tag** — a directive on a field
  (`asyncapi:"oneOf=OrderPlaced|OrderShipped"`) emits the corresponding
  combinator of `$ref`s to the named types.

The model already carries `OneOf`/`AllOf`/`AnyOf` in `spec.Schema`; only
derivation and discovery need to learn about them.

## Background

### Two-pass architecture

Derivation runs in two disjoint passes:

| Pass | When | Sees | Can't see |
|------|------|------|-----------|
| `schema.FromType` (reflection) | harness `go run`, separate process | `reflect.Type`, struct tags, embedded field types | comments, import aliases, name→type resolution, interface implementers |
| `internal/discovery` (`go/packages`+`go/types`+AST) | static, before materialization | doc comments, type relations, ident→`*types.TypeName` | cannot run reflection; operates on the materialized YAML |

This split drives every decision below. The descriptions feature
(`extractDescriptions` → `applyDescriptions`) is the precedent for
"static pass resolves, then injects".

### JSON Schema dialect: Draft 07

AsyncAPI 3.0/3.1's Schema Object is a superset of **JSON Schema Draft 07** (not
2020-12). Two corrections to `spec/schema.go` are folded into this work:

- The doc comment claims Draft 2020-12 — fix to Draft 07.
- `Defs map[string]*Schema`json:"$defs"`` uses `$defs`, a 2019-09+ keyword.
  Draft 07 uses`definitions`. The field is never emitted today; rename to
  `Definitions` with `json:"definitions"`.

## Goals / Non-goals

**Goals**

1. Opt-in `allOf` composition from embedded structs.
2. Tag-driven `oneOf` / `anyOf` / `allOf` over named types.
3. Deterministic output; referenced types hoisted into `components.schemas`.
4. Zero user boilerplate: the generator discovers combinator references and
   generates the harness registrations, consistent with "derive from real
   structs".

**Non-goals (v1)**

- `discriminator` emission (event-envelope unions). Deferred.
- Sealed-interface implementer enumeration (magic unions). Deferred.
- Cross-package selector sugar (`asyncapi:"oneOf=orders.OrderPlaced"`). Deferred;
  v1 accepts fully-qualified names or same-package short names.
- `not` derivation.

## Design decisions

### D1 — Embedding: flatten by default, `allOf` opt-in

Flattening stays the default (wire-accurate, matches `encoding/json`,
backward-compatible). An anonymous embedded field tagged `asyncapi:"allOf"`
switches to `allOf` composition.

**Rationale**: the default must describe the actual JSON, which is flat; `allOf`
is schema-level composition that downstream tooling must resolve. Opt-in keeps
the shared-base relationship available without changing existing output.

### D2 — Unions: tag-driven, resolved by pure string → `$ref`

`oneOf=`/`anyOf=`/`allOf=` directives list type names. Names are resolved to a
fully-qualified name (FQN) by a single shared rule, and emitted as `$ref`s at
derivation time (no type needed). Hoisting of those types is deferred to a
`Finalize` step in the harness.

**Rationale**: reflection cannot map a name string to a `reflect.Type` and
cannot enumerate interface implementers. Resolving names as strings keeps
`FromType` self-contained; deferring hoisting sidesteps Go's package-init order
problem (a generated `Register` in `main` runs *after* catalog `init` has
already derived refs, which is fine because refs are pure strings).

### D3 — Registry surface: global `Register` + `Finalize`

A package-level registry in `schema` keyed by FQN, populated by generated harness
code, and a `Finalize(doc)` that hoists every registered type into
`components.schemas`.

**Rationale**: threading a registry through `Spec`/`builder`/`FromType` is
purer but touches the whole DSL; a global avoids that plumbing while remaining
an internal implementation detail (user code never calls `Register` — the
generator does).

### D4 — `required` inside `allOf`: local per member

Each `allOf` member keeps its own `required` list (Draft-07-correct). We do not
hoist `required` to the outer schema.

**Rationale**: matches JSON Schema semantics; flattening (which hoists
`required`) is exactly the behavior we're *replacing* only when `allOf` is
opt-in.

### D5 — Discriminator: deferred

Deferred to a follow-up. When needed, add `spec.Discriminator` and a
`discriminator=` directive; nothing here blocks that.

## Detailed design

### 1. Tag grammar (`schema/tags.go`)

Existing directives are unchanged: `required`, `enum=a|b`, `example=...`,
`format=...`. New:

| Directive | Form | Meaning |
| ----------- | ------ | --------- |
| `allOf` | bare flag on anonymous embedded field | compose the embedded type via `allOf` instead of flattening |
| `allOf=A\|B` | on a field | `{ allOf: [ $ref A, $ref B ] }`, replaces the field's derived schema |
| `oneOf=A\|B` | on a field | `{ oneOf: [ $ref A, $ref B ] }` |
| `anyOf=A\|B` | on a field | `{ anyOf: [ $ref A, $ref B ] }` |

Name resolution rule (shared, see §4):

- If the name contains `/`, treat it as an FQN (`pkgPath.TypeName`).
- Otherwise resolve against the declaring struct's package:
  `declaringPkgPath + "." + name`.
- Types in anonymous inline structs (no `PkgPath`) must use FQNs.

Add to `tags.go`:

```go
// combinatorNames returns the "|"-separated type names for a directive
// (oneOf=/anyOf=/allOf=), or ok=false when absent.
func combinatorNames(tag, key string) ([]string, bool)
```

### 2. Registry + finalization (`schema/registry.go`, new)

```go
package schema

var registry = map[string]reflect.Type{} // FQN -> reflect.Type

// Register records types by fully-qualified name for combinator resolution.
// Generated by the harness; not intended to be called by hand. Idempotent.
func Register(types ...any)

// Finalize hoists every registered type into doc.Components.Schemas so the
// $refs emitted by FromType resolve. Derives in sorted-FQN order for
// deterministic output.
func Finalize(doc *spec.AsyncAPI)
```

`Register` dereferences pointers and ignores anonymous types (no `Name`/`PkgPath`).
`Finalize` is idempotent via the existing `hoistStruct` defs check.

### 3. `schema.FromType` / `fillObject` changes (`schema/derive.go`)

#### 3a. Combinator directives on named fields

In `fillObject`, after `applyTag`:

```go
if tag := f.Tag.Get("asyncapi"); tag != "" {
    applyTag(prop, tag)
    if hasFlag(tag, "required") {
        s.Required = append(s.Required, name)
    }
    if names, ok := combinatorNames(tag, "oneOf"); ok { prop.OneOf = refs(names, t) }
    if names, ok := combinatorNames(tag, "anyOf"); ok { prop.AnyOf = refs(names, t) }
    if names, ok := combinatorNames(tag, "allOf"); ok { prop.AllOf = refs(names, t) }
}
```

`refs(names, declaring)` resolves each name per §1 and returns
`spec.Ref(RefByName(fqn))`, where `RefByName` is the string-based counterpart of
`Ref`:

```go
func RefByName(fqn string) string { return "#/components/schemas/" + escapePointer(fqn) }
```

A combinator directive **replaces** the field's derived schema (the field type is
typically `any` or an interface). Document that combining a combinator with a
concrete field type is a user error: Draft 07 ignores `$ref` siblings, so a
`$ref` field schema must not also carry `oneOf`.

#### 3b. Opt-in `allOf` from embedding

Restructure `fillObject` so the parent's *own* fields are collected into an
`own` subschema and marked embeds are collected into an `embedded` slice:

```go
func fillObject(s *spec.Schema, t reflect.Type, defs map[string]*spec.Schema) {
    own := &spec.Schema{Type: "object", Properties: map[string]*spec.Schema{}}
    var embedded []*spec.Schema
    for i := 0; i < t.NumField(); i++ {
        f := t.Field(i)
        if f.PkgPath != "" { continue } // unexported

        if f.Anonymous {
            if ft, ok := structOf(f.Type); ok {
                if hasFlag(f.Tag.Get("asyncapi"), "allOf") {
                    embedded = append(embedded, FromType(f.Type, defs)) // $ref
                    continue
                }
                fillObject(own, ft, defs) // flatten, as today
                continue
            }
        }

        name, skip := jsonName(f)
        if skip { continue }
        prop := FromType(f.Type, defs)
        // ... existing tag handling (3a) against `own` ...
        own.Properties[name] = prop
    }

    if len(embedded) == 0 {
        s.Properties = own.Properties
        s.Required = own.Required
    } else {
        s.AllOf = append(embedded, own)
    }
}
```

Rules:

- `allOf` member order: embedded `$ref`s first, then the `own` subschema.
- **Flatten wins for un-marked embeds**: a flattened base is always flattened
  fully (its internal markers are ignored at the point of flattening). Markers
  are honored only at the level where they appear; a marked base's own schema is
  built by `hoistStruct`, which honors markers at *its* top level.

### 4. Discovery + harness (`internal/discovery`)

The static pass gains one responsibility: collect the FQNs referenced by
`oneOf=`/`anyOf=`/`allOf=` directives (the embedded `allOf` marker needs nothing
— the embedded field is already a real type, hoisted by reflection).

- Walk reachable structs (reuse `walkStructTypes`), parse `asyncapi` tags for
  the three directives.
- For each name, apply the **same resolution rule** as §1 (same-package short
  name → `pkg.Path()+"."+name`; FQN passthrough). Split the FQN into
  `(importPath, typeName)` for code generation.
- Generate the harness so that it:
  1. imports each referenced package;
  2. calls `schema.Register(pkgN.TypeName{}, …)` at the top of `main`;
  3. calls `schema.Finalize(doc)` for each catalog before marshaling.

Generated harness shape:

```go
package main

import (
    "os"
    "github.com/goccy/go-yaml"
    "github.com/RubenRibGarcia/asyncgo/schema"

    pkg0 "github.com/acme/orders"
)

func main() {
    schema.Register(pkg0.OrderPlaced{}, pkg0.OrderShipped{})

    docs := []any{ pkg0.Catalog /* , ... */ }
    for _, d := range docs {
        if a, ok := d.(*spec.AsyncAPI); ok { schema.Finalize(a) }
    }
    // ... existing marshal ...
}
```

### 5. `spec/schema.go` corrections

- Fix the Draft 07 comment.
- Rename `Defs` → `Definitions` with `json:"definitions,omitempty"`
  `yaml:"definitions,omitempty"`.
- `OneOf`/`AllOf`/`AnyOf` already exist — no change.

## Example (before / after)

Input:

```go
type BaseSchema struct {
    ID string `json:"id" asyncapi:"required"`
}

type OrderPlaced struct {
    BaseSchema `asyncapi:"allOf"` // opt-in composition
    Amount     float64 `json:"amount" asyncapi:"required"`
}

type Delivery struct {
    OrderPlaced
    Carrier string `json:"carrier"`
}

type Event struct {
    Data any `json:"data" asyncapi:"oneOf=OrderPlaced|OrderCancelled"`
}
```

Output (abridged):

```yaml
components:
  schemas:
    github.com/.../embedded.OrderPlaced:
      allOf:
        - $ref: "#/components/schemas/github.com/.../embedded.BaseSchema"
        - type: object
          properties:
            amount: { type: number, format: double }
          required: [amount]
    github.com/.../embedded.Delivery:      # BaseSchema + OrderPlaced both flattened
      type: object
      properties:
        id: { type: string }
        amount: { type: number, format: double }
        carrier: { type: string }
    github.com/.../embedded.Event:
      type: object
      properties:
        data:
          oneOf:
            - $ref: "#/components/schemas/github.com/.../embedded.OrderPlaced"
            - $ref: "#/components/schemas/github.com/.../embedded.OrderCancelled"
      required: [data]  # if tagged required
    # OrderPlaced / OrderCancelled hoisted by Finalize
```

## Edge cases

- **Shadowing**: a named field shadowing an embedded field is ambiguous under
  `allOf`. Flattening (default) sidesteps it; the opt-in `allOf` path documents
  this as the user's responsibility.
- **Recursion**: unchanged — `hoistStruct` pre-registers before filling, so
  `allOf` self/cycle references terminate.
- **Determinism**: `Finalize` derives in sorted-FQN order; `embedded`/`own`
  ordering is fixed.
- **`json:"-"` and unexported fields**: skipped, as today.
- **Anonymous inline structs**: no package path, so combinator names must be
  FQNs (§1).
- **Combinator on a concrete field type**: user error (replaces the type; `$ref`
  siblings ignored in Draft 07). Document, don't enforce in v1.
- **`Register` idempotency / dedup**: same FQN from multiple catalogs registers
  once; `Finalize` over multiple docs is safe.

## Rollout plan

1. **Stage 0 — model groundwork** (`feat(spec)` / `docs(spec)`): Draft 07
   comment fix + `$defs`→`definitions` rename. No behavior change.
2. **Stage 1 — `allOf` from embedding** (`feat(schema)`): §3b restructure +
   marker + `RefByName`. Pure reflection, no discovery change.
3. **Stage 2 — tag combinators** (`feat(schema)` + `feat(internal)`): §1, §2,
   §3a, §4 (registry, `Register`, `Finalize`, harness generation).
4. **Stage 3 — deferred**: `discriminator`, sealed-interface enumeration,
   cross-package selector sugar.

## Testing plan

- **Stage 0**: update any golden YAML if `$defs`/`definitions` appears (none
  emit today); add a `spec` marshal test for `definitions`.
- **Stage 1**: table-driven `derive_test.go` cases:
  `should_compose_embedded_struct_with_allOf`,
  `should_flatten_embedded_struct_by_default`,
  `should_keep_required_local_per_allOf_member`,
  `should_terminate_on_recursive_allOf`.
- **Stage 2**:
  - `schema` unit tests: `should_emit_oneOf_refs`,
    `should_emit_anyOf_refs`, `should_emit_allOf_refs`,
    `should_resolve_same_package_short_name`,
    `should_pass_through_fully_qualified_name`,
    `should_hoist_referenced_types_on_finalize`.
  - `internal/discovery` golden test: extend an example with a union field and
    assert the committed `asyncapi.yaml` reproduces (the existing
    `examples/*/asyncapi.yaml` golden pattern).
- **Race**: `go test ./... -race` per AGENT.md.

## Open / deferred

- Discriminator + `mapping` for event envelopes.
- Sealed-interface (`interface{ method() }`) implementer enumeration.
- Cross-package selector sugar resolved via `go/types` (needs passing resolved
  FQNs into the registry, so the reflection pass can match — deferred).
