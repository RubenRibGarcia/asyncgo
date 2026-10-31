# Design: Custom schema providers for types with custom (de)serialization

- **Status**: Accepted
- **Created**: 2026-08-29
- **Status updated**: 2026-10-31
- **Scope**: `spec/`, `schema/`
- **Raw notes**: `docs/scratch/custom-schema-providers-brainstorm.md`

## Summary

Let a named struct type declare its own wire schema — overriding
reflection-derived derivation — so that a type with a custom
`MarshalJSON`/`UnmarshalJSON` (or `encoding.TextMarshaler`, or any other wire
customization) can still produce a correct AsyncAPI Schema Object. A new
`spec.SchemaProvider` interface is detected via reflection in `schema.FromType`;
no static/discovery pass, registry, or harness change is required. A named
non-struct (`type OrderID string`, a named slice or map) is hoisted through the
same interface, so a scalar or collection can carry a reusable constraint
instead of being inlined at every use site ([#27]).

[#27]: https://github.com/RubenRibGarcia/asyncgo/issues/27

## Background

asyncgo's core bet is *"derive from real structs, no duplication"*:
`schema.FromType` maps a `reflect.Type` to a `*spec.Schema`, and that mapping is
only correct while there is a 1:1 contract between Go field shape and JSON wire
shape. Custom marshaling breaks that contract:

```go
type Money struct {
    Amount   int64
    Currency string
}

// Wire format is a single string, not an object.
func (m Money) MarshalJSON() ([]byte, error) {
    return json.Marshal(fmt.Sprintf("%d.%02d %s", m.Amount/100, m.Amount%100, m.Currency))
}
func (m *Money) UnmarshalJSON(b []byte) error { /* ... */ }
```

Reflection derives `{type: object, properties: {amount, currency}}`, but the
actual wire format is a string. Once the developer customizes the wire format,
the schema is no longer *derivable* — it must be **declared**. The only design
question is the surface through which it is declared.

### Architecture facts that decide the shape

| Fact | Consequence |
| ---- | ----------- |
| Reflection can detect interface implementation (`reflect.Type.Implements`). | A method-based hook is **pure reflection** — same category as `allOf`-from-embedding, *not* `oneOf`-from-tag (which needed the `Register`/`Finalize` registry). No discovery pass. |
| `FromType` dereferences pointers before dispatch. | Pointer-receiver methods are lost unless probed explicitly via `reflect.PointerTo(t)`, mirroring `encoding/json`'s value-then-addressable rule. |
| `FromType` works on a `reflect.Type`, not a value. | The hook can only be invoked on a **zero value**, so the method must describe the *type*, not an instance. |
| Named structs all route through `hoistStruct`. | A single interception point scopes the feature to named structs for free and preserves FQN hoisting. Non-struct kinds need their own interception at the top of `FromType` (D7). |
| The harness avoids executing user code beyond package `init`. | Calling the method is the first user-code execution beyond `init` and must be an explicit, accepted trade-off. |

### The two-pass split (unchanged)

This feature needs **only the reflection pass**. It changes nothing in
`internal/discovery`: interface satisfaction is visible to `reflect`, and no
name→type resolution or doc-comment injection is involved.

## Goals / Non-goals

**Goals**

1. A named type — struct or non-struct — can declare a `*spec.Schema` that
   replaces reflection derivation, hoisted under its fully-qualified name
   (`pkgPath.TypeName`).
2. The hook is detected on both value and pointer receivers.
3. Existing output is unchanged for types that do not implement the interface.
4. No discovery, registry, or harness changes: the feature composes with the
   existing `Register`/`Finalize` machinery for combinator-referenced types.
5. One hook covers both marshal and unmarshal directions.

**Non-goals (v1)**

- Auto-hoisting of `$ref`s referenced *inside* a returned custom schema.
- Validation of the returned schema, or panic recovery around the method call.
  The contract is documented, not enforced.
- Separate marshal vs unmarshal hooks.
- Field-level type override via struct tag (`asyncapi:"type=..."`).
- Anonymous inline structs — Go cannot define methods on unnamed types.

## Design decisions

### D1 — Interface lives in `spec`, method `AsyncAPISchema() *Schema`

```go
package spec

// SchemaProvider is implemented by a named type that customizes its
// wire representation (e.g. via MarshalJSON). AsyncAPISchema returns the
// schema asyncgo emits in place of reflection-derived derivation.
type SchemaProvider interface {
    AsyncAPISchema() *Schema
}
```

**Rationale**: user catalogs already import `spec` (headers, examples,
bindings), so this adds **zero new imports**, and the method returns its own
package's type. Placing it in `schema` would force a second import just to name
the interface while still needing `spec` for the return type.

### D2 — Detect on `t` and `*t`, invoking on a zero value

`FromType` sees a `reflect.Type`, so the only receiver available is a zero
value. The contract is therefore **type-level and static**: the method must not
depend on receiver state, must be pure and deterministic, and must not panic.
Detection probes the value type first, then the pointer type — mirroring
`encoding/json`'s value-then-addressable behavior.

**Rationale**: `MarshalJSON` is commonly on a pointer receiver; probing both
directions keeps the feature working regardless of receiver, and stating the
zero-value contract up front closes the gap between "instance method" (like
`json.Marshaler`) and "type description" (what schema derivation needs).

### D3 — Intercept in `hoistStruct`, not at the top of `FromType`

The provider check lives in `hoistStruct`, so the custom schema is registered
under the type's FQN and a `$ref` is returned — preserving the "named types are
always hoisted" invariant. It also scopes the feature to named structs
automatically, since only named structs reach `hoistStruct`.

**Rationale**: intercepting at the top of `FromType` would return the custom
schema *unhoisted*, breaking `$ref` resolution and diverging from how every
other named struct behaves.

**Promotion for non-structs ([#27])**: a non-struct has no `hoistStruct` to
intercept, so its provider check sits at the top of `FromType`, guarded to kinds
other than struct and interface. It hoists before returning, so D3's invariant —
never hand back an unhoisted custom schema — still holds, and the struct path
keeps its placement unchanged.

### D4 — `nil` falls back to reflection; empty schema is unconstrained

A `nil` return is treated as "not customized" and `hoistStruct` proceeds with
`fillObject` as today. A non-nil empty `&spec.Schema{}` is *distinct*: it is
hoisted as `{}`, i.e. an explicitly unconstrained schema.

**Rationale**: there is no error channel in the interface, so `nil` is the
natural "no-op" sentinel. Keeping `nil` and `&spec.Schema{}` distinct gives the
developer both "derive for me" and "any value" without an extra flag.

### D5 — One hook covers marshal and unmarshal

No separate `AsyncapiMarshalSchema()`/`AsyncapiUnmarshalSchema()`. JSON Schema
describes the shared wire contract; asymmetric needs are expressed inside the
returned schema via `oneOf`/`anyOf`.

**Rationale**: AsyncAPI documents a single wire format; splitting the hook
would surface a distinction that Draft 07 schemas do not carry and that
practically never differs.

### D6 — Accept user-code execution; document the determinism contract

Calling `AsyncAPISchema()` executes user code during generation (in the catalog
package's `init`, and again via `Finalize`). This is accepted as the cost of the
interface approach, and the method is documented as pure, deterministic, and
panic-free.

**Rationale**: the alternative (catalog-side registration) avoids executing
user code but reintroduces per-type boilerplate that contradicts "derive from
real structs". The interface is the more idiomatic surface and mirrors the
stdlib; the trade-off is recorded here rather than made implicitly.

## Detailed design

### 1. `spec/provider.go` (new)

Add the interface from D1. `spec` already owns `Schema` and `Ref`, so no new
dependencies. This is additive and changes no existing behavior.

### 2. `schema/derive.go`

Add a package-level type and a detection helper:

```go
var schemaProviderType = reflect.TypeFor[spec.SchemaProvider]()

// asSchemaProvider detects a SchemaProvider on t or *t, mirroring
// encoding/json's value-then-addressable rule. It returns the interface and
// ok=false when t is not a provider.
func asSchemaProvider(t reflect.Type) (spec.SchemaProvider, bool) {
    if t.Implements(schemaProviderType) {
        return reflect.New(t).Elem().Interface().(spec.SchemaProvider), true
    }
    if reflect.PointerTo(t).Implements(schemaProviderType) {
        return reflect.New(t).Interface().(spec.SchemaProvider), true
    }
    return nil, false
}
```

Modify `hoistStruct` to consult the provider before `fillObject`:

```go
func hoistStruct(t reflect.Type, defs map[string]*spec.Schema) *spec.Schema {
    name := Name(t)
    if _, ok := defs[name]; ok {
        return spec.Ref(Ref(t))
    }

    s := &spec.Schema{Type: "object"}
    defs[name] = s // pre-register for recursion termination

    var custom *spec.Schema
    if p, ok := asSchemaProvider(t); ok {
        custom = p.AsyncAPISchema()
    }
    if custom != nil {
        s = custom
        defs[name] = s
    } else {
        fillObject(s, t, defs)
    }

    return spec.Ref(Ref(t))
}
```

Notes:

- `defs[name]` is pre-registered before any derivation, so recursion (the
  non-provider `fillObject` path) terminates exactly as today.
- A provider's returned schema is **not traversed**, so a self-referential
  provider cannot loop.
- `inlineStruct` is untouched: anonymous structs cannot implement the interface.

A non-struct has no `hoistStruct`, so its check sits at the top of `FromType`:

```go
if isNamed(t) && t.Kind() != reflect.Struct && t.Kind() != reflect.Interface {
    if _, hoisted := defs[Name(t)]; hoisted {
        return spec.Ref(Ref(t))
    }
    if p, ok := asSchemaProvider(t); ok {
        if custom := p.AsyncAPISchema(); custom != nil {
            defs[Name(t)] = custom
            return spec.Ref(Ref(t))
        }
    }
}
```

- `reflect.Interface` is excluded: an interface whose method set includes
  `AsyncAPISchema` satisfies the provider type, but its zero value is a nil
  interface, so `reflect.New(t).Elem().Interface().(spec.SchemaProvider)` would
  panic. The guard keeps such a type deriving `{}` as today.
- A `nil` return from a non-struct provider keeps the current output: the call
  falls through to the kind switch and the schema stays **inline**, with no
  `$ref` and no `components.schemas` entry.
- Only named non-structs are intercepted, so a plain `type OrderID string` — the
  common case — is unchanged. Implementing the interface is the opt-in.

### 3. `schema/registry.go` — no behavior change, composes for free

`Finalize` calls `FromType` on every registered type, which routes through
`hoistStruct` (structs) or the top-of-`FromType` check (non-struct providers), so
a combinator-referenced (`oneOf=`/`anyOf=`/`allOf=`) custom type is hoisted with
its custom schema. No discovery or registry modification is needed.

The generated harness (`internal/discovery/materialize.go`) registers each
combinator-referenced type as `new(pkg.T)` rather than `pkg.T{}`: a composite
literal is only valid for struct, array, slice and map types, so a directive
naming a non-struct would otherwise emit code that does not compile. `Register`
already dereferences pointers, so `new(T)` registers `T` itself.

## Example (before / after)

Input:

```go
type Money struct {
    Amount   int64
    Currency string
}

// Custom wire format: "12.34 USD".
func (m Money) MarshalJSON() ([]byte, error) { /* ... */ }
func (m *Money) UnmarshalJSON(b []byte) error { /* ... */ }

// Declare the schema for the customized wire format.
func (Money) AsyncAPISchema() *spec.Schema {
    return &spec.Schema{
        Type:     "string",
        Pattern:  `^\d+\.\d{2} [A-Z]{3}$`,
        Examples: []any{"12.34 USD"},
    }
}

type Order struct {
    ID    string `json:"id" asyncapi:"required"`
    Price Money  `json:"price"`
}
```

Before (wrong — object schema from the struct fields):

```yaml
github.com/acme/orders.Money:
  type: object
  properties:
    amount: { type: integer }
    currency: { type: string }
```

After:

```yaml
github.com/acme/orders.Money:
  type: string
  examples:
  - 12.34 USD
  pattern: "^\d+\.\d{2} [A-Z]{3}$"

github.com/acme/orders.Order:
  type: object
  properties:
    id: { type: string }
    price:
      $ref: "#/components/schemas/github.com/acme/orders.Money"
  required: [id]
```

## Edge cases

- **`AsyncAPISchema()` returns `nil`** — fall back to reflection derivation
  (D4); the type is treated as not customized. A non-nil `&spec.Schema{}` is
  hoisted as `{}` (unconstrained) — distinct from `nil`.
- **Returns a `$ref` schema** — hoisted as-is, producing a `ref → ref` double
  indirection. Documented expectation: return the *resolved* schema; asyncgo
  owns hoisting.
- **Custom schema references other named types** (`$ref` inside `items`,
  `oneOf`, `properties`, …) — those types are not auto-hoisted; they must be
  `Register`ed (the same `Finalize` path as combinators).
- **Pointer receiver** — detected via `reflect.PointerTo(t)`; a value-only field
  whose provider method is on `*T` still works.
- **Zero-value receiver / repeat calls** — the method must not read receiver
  state (D2); reading it sees only zero values and is non-deterministic by
  contract. It may be invoked more than once per generation (once from a
  message/field reach, and again via `Finalize`), so it must be pure.
- **Descriptions** — the discovery pass's `applyDescriptions` matches derived
  fields; a custom schema has none, so injection is a no-op. The developer sets
  `Description` inside the returned schema.
- **Recursion** — the returned schema is not traversed, so self-referential
  providers terminate.
- **Promoted methods** — a method promoted from an embedded provider type
  counts (standard Go method-set semantics, matching `encoding/json`); the
  outer type then inherits the embedded type's schema unless it shadows the
  method.
- **Provider vs `allOf` markers** — a provider's returned schema fully replaces
  derivation, so `allOf`/embedding markers on that type are ignored.
- **Panic in the method** — surfaces as a harness/generation failure with a
  stack trace (the harness already panics on unrecoverable errors). Documented,
  not recovered, in v1.
- **Anonymous structs** — out of scope: Go cannot define methods on an unnamed
  type, so they always derive inline.
- **Named non-struct without a provider** — unchanged: `type OrderID string`
  still derives inline as `{type: string}`, with no `$ref` and no
  `components.schemas` entry. Implementing the interface is the opt-in.
- **Interface kind with a provider method set** — detection is skipped, so
  `type P interface { AsyncAPISchema() *Schema }` derives `{}` as today instead
  of panicking on the nil zero value.

## Rollout plan

1. **Stage 0 — add the interface** (`feat(spec)`): `spec/provider.go` with
   `SchemaProvider`. Additive; no behavior change.
2. **Stage 1 — detect and intercept** (`feat(schema)`): `schemaProviderType`,
   `asSchemaProvider`, and the `hoistStruct` change, plus unit tests.
3. **Stage 2 — document** (`docs`): a README section on custom types and a
   `godoc` note on the interface contract.
4. **Stage 3 — deferred**: auto-hoisting of `$ref`s inside returned schemas,
   validation/recovery, field-level `type=` tag override.
5. **Stage 4 — named non-struct providers** (`feat(schema)`, [#27]): the
   `FromType` non-struct interception, plus harness registration via `new(T)` so
   a combinator directive can name a non-struct member.

## Testing plan

- **Stage 0**: no behavior change; existing tests must still pass. Optionally a
  compile-time assertion that a sample type satisfies the interface.
- **Stage 1**: table-driven `schema/derive_test.go` cases:
  - `should_use_custom_schema_for_provider_struct`
  - `should_hoist_custom_schema_under_fqn`
  - `should_detect_pointer_receiver_provider`
  - `should_fall_back_to_reflection_on_nil_schema`
  - `should_fall_back_to_reflection_when_not_provider`
  - `should_hoist_empty_schema_as_unconstrained`
  - `should_honor_provider_via_finalize`
- **Stage 4** ([#27]):
  - `should_hoist_named_string_provider`
  - `should_hoist_named_slice_provider` / `should_hoist_named_map_provider`
  - `should_detect_pointer_receiver_on_named_non_struct`
  - `should_fall_back_to_inline_on_nil_provider`
  - `should_leave_named_non_struct_inline_without_provider`
  - `should_not_panic_on_interface_with_provider_method_set`
  - `should_hoist_once_and_ref_from_every_use_site`
  - `should_hoist_registered_non_struct_provider` (via `Finalize`)
  - `should_resolve_combinator_ref_to_non_struct_provider`
  - `internal/discovery`: harness source contains `new(pkg.T)`, never `pkg.T{}`
- **Race**: `go test ./... -race` per AGENT.md.

## Open / deferred

- Auto-hoisting of `$ref`s referenced inside a returned custom schema.
- Schema validation and panic recovery around the method call.
- Separate marshal/unmarshal hooks, if a real asymmetry ever materializes.
- Field-level type override (`asyncapi:"type=string"`) for the common
  single-field case without a full custom type.
