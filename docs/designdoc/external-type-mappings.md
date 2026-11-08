# Design: External type mappings

- **Status**: Accepted
- **Created**: 2026-11-08
- **Status updated**: 2026-11-08
- **Scope**: `schema/`, `internal/discovery/`, `internal/cli/`

## Summary

Let the asyncgo user declare how a type they do not own — `gofrs/uuid.UUID`,
`github.com/google/uuid.UUID`, a decimal type, a custom time format — derives
into a Schema Object, instead of getting whatever the type's Go kind suggests
(`uuid.UUID` is a named `[16]byte`, so it derives as an array of integers
today). Two entry points: a type-safe Go registry for rich schemas, and a
declarative `asyncgo.yaml` config for simple `{type, format}`-style mappings.

Two consequences fall out of name-keying and are part of the design: derived
aliases (`type OrderID uuid.UUID`) inherit the base type's mapping, and
asyncgo ships built-in mappings for a small set of well-known third-party
types — without importing those libraries, so consumers gain no dependency.

## Background

Three facts constrain the design:

**Derivation happens in a separate harness process.** `asyncgo generate` does
not derive schemas in-process. It discovers catalogs
(`internal/discovery.Find`), *generates* a `main.go` harness
(`internal/discovery/materialize.go`), compiles and runs it, and the harness
calls `schema.Register(...)` for combinator-referenced types and
`schema.Finalize(doc)` per document. Anything that must influence derivation has
to execute inside that harness process, which imports only two kinds of
packages: every package that declares a catalog variable, and every package
referenced by a combinator directive (`new(pkg.T)` registrations).

**That process derives twice, at two different times.** A message payload is
derived while the catalog package's own variables are initialized — `Spec(...)`
is a package-level variable, and it calls `FromType` for each payload. The types
`schema.Register` recorded are derived in `Finalize`. Only the second happens
after the harness can register a mapping: a project's API mappings register in
some package's `init`, and its `asyncgo.yaml` mappings register in the harness's
`main()`, both after the catalog package has finished initializing. Registration
therefore cannot be the whole story — see D10.

**The existing extension point is type-owned.** `spec.SchemaProvider`
(`AsyncAPISchema()`, see `docs/designdoc/custom-schema-providers.md`) requires
implementing a method *on the type*. That is impossible for a third-party type:
nobody can add a method to `gofrs/uuid.UUID` from outside its package. The
workaround today is a wrapper type (`type OrderID uuid.UUID`), which pollutes
every struct field and payload signature.

**Special cases are exact-type today.** `FromType` maps `time.Time` to
`date-time` by exact `reflect.Type` and byte slices to `byte` by *kind*. A
mapping registry keyed by exact type generalizes the first special case but
cannot express the second.

**Reflection cannot walk the defined-from chain.** For `type OrderID
uuid.UUID`, `reflect` exposes only the `OrderID` named type and its kind —
`reflect.Type` has no `Underlying()` and no link back to the type it was
declared from. Alias-chain resolution is impossible at derivation time and
must happen where full type information exists: the discovery pass, which
already loads `NeedSyntax | NeedTypesInfo` via `golang.org/x/tools/go/packages`.

**Built-in mappings must not create dependencies.** Any mapping asyncgo ships
for a third-party type has to be name-keyed data (an embedded config), never a
Go import of that library — otherwise every asyncgo consumer would pull in
gofrs/uuid, shopspring/decimal, and friends. Name-keyed lookups also cost
nothing when the library is not used: the registry is only consulted for
types `FromType` actually encounters.

Precedent for name-keyed registries: `schema.Register` already keys types by
fully-qualified name (`pkgPath.TypeName`), and combinator directives
(`oneOf=A|B`) already resolve such names, rejecting semi-qualified ones.

## Goals / Non-goals

**Goals**

1. A user of asyncgo can map a named third-party type to a schema without
   touching the type: `uuid.UUID` → `{type: string, format: uuid}`.
2. The mapping is transitive: fields, `[]T`, `map[K]T`, `*T`, and anonymous
   structs containing `T` all derive the mapped schema in place.
3. The mapping is deterministic and can be declared either in Go (rich
   schemas) or in a config file (simple schemas) that both `generate` and
   `check` honor identically.
4. A derived alias (`type OrderID uuid.UUID`) derives like its base type
   without an extra declaration, unless it declares its own.
5. asyncgo ships built-in mappings for well-known third-party types
   (`uuid`, `ulid`, `decimal`, `big.Int`) that work with zero configuration
   and add zero dependencies.

**Non-goals (v1)**

- Kind-based mappings (`[]byte`, all `uint8` slices) — exact-name keying only.
- Hoisting mapped schemas into components by default (inline always; see D3
  for the one exception).
- Config coverage of combinator directives or other future config sections —
  the file is shaped for growth but ships with `mappings` only.
- Field-level mapping (`asyncapi:"map=..."` tag): a field of a mappable type
  that wants a *different* schema should use a local wrapper type or the
  provider path.

## Design decisions

### D1 — One name-keyed registry, two entry points

A package-level registry in `schema`, keyed by fully-qualified name
(`pkgPath.TypeName`), consulted by `FromType` after pointer deref:

```go
// Type-safe, for rich schemas. The function is called lazily at derivation
// time and must satisfy the same purity contract as SchemaProvider.
schema.MapSchema[uuid.UUID](func() *spec.Schema {
    return &spec.Schema{Type: "string", Format: "uuid"}
})

// Name-keyed, value-shaped. This is the target the config path emits into
// the generated harness; also usable directly.
schema.MapSchemaByName("github.com/gofrs/uuid.UUID",
    &spec.Schema{Type: "string", Format: "uuid"})
```

`MapSchema[T]` computes the key via `reflect.TypeFor[T]` and the existing
`Name()`. One registry serves both; name-keying means the config path needs no
compile-time reference to the third-party package.

**Rationale**: mirrors the existing `Register`/combinator name-keyed design
rather than introducing a second keying scheme. Exact-name keying gives
per-type opt-in: `type OrderID uuid.UUID` is *not* affected — it can get its
own mapping or a `SchemaProvider`, deliberately.

### D2 — Precedence: exact mapping > SchemaProvider > inherited mapping > kind derivation

`FromType` consults the registry for the type's exact FQN immediately after
`deref(t)`, before the named-non-struct provider interception, before
`hoistStruct`, and before the byte-slice/kind switch. An external mapping
therefore overrides a `SchemaProvider` on the same type.

The full order, with the alias inheritance of D8 and the built-ins of D9:

1. Exact mapping for the type's own FQN (config/API/built-in — see D5 for
   which registration wins within this tier)
2. `SchemaProvider` implemented on the type itself
3. Mapping inherited through the alias chain (D8)
4. Kind-based derivation, including the byte-slice rule

**Rationale**: the asyncgo user is the author of the published document; the
provider is the type's *suggested* default. The provider path remains the
primary tool for types the user owns; the mapping fills the gap where the
interface cannot be implemented at all. An inherited mapping ranks *below*
the provider because it is implicit: a user who writes `AsyncAPISchema()` on
their own alias declared a schema for that exact type, and that beats a
default picked up from the base type. An exact mapping ranks *above* the
provider because it is a deliberate document-level override — the same rule
reviewed for the original design. Implementation note: tiers 1, 3 and 4 live
in `FromType`; tier 2 sits between them (the provider is intercepted in two
places today — the named-non-struct block and `hoistStruct` — so the tier-3
alias lookup gets a consultation point in each of those two paths, after the
provider check and before derivation).

### D3 — Inline at every derivation site; Finalize hoists registered mapped types

A mapped type derives **inline** at every use site, like `time.Time` — never
auto-hoisted, no `$ref`. Exception: a mapped type that is also
combinator-referenced (so `Register` recorded it and a directive emitted a
`$ref`) must still resolve that `$ref`. `Finalize` therefore hoists the mapped
schema under the type's FQN for any registered mapped type. The inline schema
and the hoisted component carry identical content, so both shapes agree.

**Rationale**: inline matches the dominant use case (`uuid.UUID` fields) and
keeps documents small; refusing to hoist anything would leave dangling `$ref`s
for combinator members, which is a correctness bug, not a style choice.

### D4 — `time.Time` becomes a built-in mapping

The exact-type special case in `FromType` is deleted; `time.Time` is
registered as a built-in mapping in the package's `init()`. Users can override
it like any other mapping. Byte slices stay kind-based in `FromType` (D1
rules out kind keying for them).

**Rationale**: one consultation path instead of two; the built-in registration
doubles as the reference example in godoc.

### D5 — Re-registration replaces; config wins ties

Registering a second mapping for the same FQN replaces the first (idempotent,
no panic). Ordering follows Go's own rules, which gives the three mapping
sources a deterministic precedence within tier 1 of D2: built-ins register in
the `schema` package `init()` (first), user API mappings register in an
imported package's `init()`, and config-emitted registrations run in harness
`main()` — so for the same FQN: config > API > built-in. Payloads derived before
the harness reaches `main()` are re-derived by `Finalize` (D10), which is why the
observed document honours that order rather than freezing the pre-`main()` state.

**Rationale**: silent first-wins would make the config path unable to correct
a mapping inherited from an imported package; last-wins with a deterministic
ordering (inits → main) is predictable and documented. Every tier has registered
by the time `Finalize` derives anything, which is what makes the order observable
in the output: a payload derived earlier in the process is re-derived there
(D10).

### D6 — Config file: `asyncgo.yaml` at the module root, `mappings:` only

Both `generate` and `check` auto-discover `<dir>/asyncgo.yaml`; no new flag in
v1. Only a `mappings:` section is defined, whose values are whitelisted
annotation/constraint keywords:

```yaml
mappings:
  github.com/gofrs/uuid.UUID:
    type: string
    format: uuid
  github.com/shopspring/decimal.Decimal:
    type: string
    pattern: '^-?\d+(\.\d+)?$'
```

Allowed value keys: `type`, `format`, `description`, `pattern`, `enum`,
`const`, `default`, `examples`, `contentEncoding`, `contentMediaType`,
`minLength`, `maxLength`, `minimum`, `maximum`, `exclusiveMinimum`,
`exclusiveMaximum`, `minItems`, `maxItems`, `readOnly`, `writeOnly`,
`deprecated`. Unknown keys, nested objects, combinators, and `$ref` are
rejected with the allowed list in the error, at discovery time — before any
harness is generated.

Discovery emits the validated mapping into the harness as a JSON payload
rather than as a `spec.Schema` struct literal:

```go
// generated harness main(), before catalog derivation
if err := schema.MapSchemaJSON("github.com/gofrs/uuid.UUID",
    `{"type":"string","format":"uuid"}`); err != nil {
    panic(err)
}
```

**Rationale**: a config file honored by both commands beats per-invocation
flags (CI would have to repeat them, and `generate`/`check` could drift).
Emitting JSON and unmarshaling into `spec.Schema` avoids hand-rolling Go
literals for twenty keywords (regex escaping, `[]any` for `enum`) — one new
function covers the whitelist, and the whitelist itself is enforced before
the harness ever runs, not by the runtime call. Naming the file `asyncgo.yaml` keeps it
next to the artifact it shapes.

### D7 — Never-consulted mappings surface as warnings

`FromType` records every FQN it resolves through the mapping registry.
`schema.MappingWarnings()` returns mappings registered but never consulted —
the harness reports them on stderr, one line each, so a typo'd FQN in config (or
a mapping declared in a package the harness never imports) is visible in
`generate`/`check` output instead of silently ignored. They stay warnings: the
document is materialized and the command succeeds.

**Rationale**: name-keying fails silently by nature; the type may be
third-party and absent from the module, so "does the type exist" cannot be
checked statically. Consultation tracking is the only reliable catch, and it
is a warning (not an error) because a mapping can legitimately be unused in a
given module.

Built-in mappings (D9) are excluded from warnings: they are always registered,
and most modules use none of them — warning about every unencountered built-in
would fire on essentially every run. Only user-declared mappings (config and
API) participate. Alias edges (D8) are not mappings and are not tracked; an
alias whose chain dead-ends without a mapping simply derives by kind.

### D8 — Derived aliases inherit the base type's mapping via discovery-emitted edges

For every type declared in the module as `type X Y` — where `Y` resolves to a
*defined* named type, directly or behind a pointer — discovery emits an alias
edge into the harness:

```go
// generated harness main()
schema.MapAlias("github.com/acme/orders.OrderID", "github.com/gofrs/uuid.UUID")
```

At derivation time, when no exact mapping or provider applies (tiers 1–2 of
D2), `mappingFor` follows the alias chain — `OrderID` → `uuid.UUID` — and
uses the mapping found at its end, with a depth guard against cycles
(`type A B; type B A` is a compile error, but the guard is cheap defense).
One edge is emitted per declaration, so intermediate aliases participate:
`type A OrderID` emits `A → OrderID`, and if the user API-maps `OrderID`
itself, `A` inherits `OrderID`'s mapping, not `uuid.UUID`'s.

Go type aliases (`type A = B`) need no edges: they introduce no new type, so
`reflect` already sees `B`. Alias declarations inside third-party packages are
not scanned (discovery only walks module packages), so chains that cross a
package boundary inside a dependency stop there — the documented limit.

**Rationale**: `reflect` cannot resolve the declared-from chain (Background),
so runtime-only resolution is impossible, and the discovery pass already has
the syntax and type information needed. Emitting one edge per declaration
(rather than resolving each alias to its ultimate base at discovery time)
keeps the harness generator dumb and lets runtime API mappings on
intermediate aliases compose. The alternative — handling aliases only by
explicit per-alias declarations — pushes bookkeeping onto the user for
something the toolchain can see.

### D9 — Built-in mappings for well-known third-party types, shipped as embedded config

`schema` embeds a `builtin_mappings.yaml` (`go:embed`) parsed at package
`init()` through the same whitelist parser the user config uses (see
`internal/mappingconfig` below). The v1 set, each verified against the
library's actual `encoding/json` behavior — a built-in that documents a shape
the library does not produce is a lie in every generated document:

| FQN | Mapping | Verified behavior |
| --- | --- | --- |
| `github.com/gofrs/uuid.UUID` | `{type: string, format: uuid}` | `MarshalText` → RFC 9562 lowercase hex-and-dash string |
| `github.com/google/uuid.UUID` | `{type: string, format: uuid}` | same contract as gofrs |
| `github.com/oklog/ulid/v2.ULID` | `{type: string, pattern: "^[0-9ABCDEFGHJKMNPQRSTVWXYZ]{26}$"}` | `MarshalText` → 26-char Crockford base32 |
| `github.com/shopspring/decimal.Decimal` | `{type: string, pattern: "^-?\\d+(\\.\\d+)?$"}` | `MarshalJSON` → JSON string by default (`MarshalJSONWithoutQuotes=false`) |
| `math/big.Int` | `{type: integer}` | `MarshalJSON` → bare digits as a JSON number (golang/go#28154); today it derives as an empty object — a struct with unexported fields |

**Rationale**: the user asked for exactly this — defaults that make common
third-party types derive correctly without any per-project declaration. The
embedded-config shape (not Go code importing those libraries) is what makes
it dependency-free, and extending the set later is a data-only contribution.
The list starts deliberately small: every entry needs its marshal behavior
verified per library version, so the set grows on evidence, not popularity.

### D10 — Payload schemas are derived provisionally and re-derived by `Finalize`

A message payload is the one reflection-derived node a catalog computes while its
own package variables initialize (Background). No declared mapping can be
registered by then, so a payload derived there would miss every non-built-in
mapping — the exact case the feature exists for, `uuid.UUID` inside a payload
struct. Deferring the derivation outright is worse: it would move `Spec`'s
validation, its auto-hoist collision check and `PayloadExamples` after the build,
and leave a document without payload schemas for every caller that never
finalizes (an in-process user, or a test asserting on the assembled document).

Payload derivation therefore goes through `schema.FromTypePayload(doc, t, defs)`:
it derives the schema now, so `Spec` stays whole, and records the derivation
against the document. `Finalize` re-derives each recorded type with every mapping
tier in place, rewrites the payload node's *contents* in place (the message keeps
the pointer it always had), and replaces the components the first derivation
hoisted — dropping the ones a mapping made obsolete, such as the component of a
type that now derives inline. Examples already written onto the payload are
carried across, so `PayloadExamples` survives the rewrite.

**Rationale**: it is the smallest change that makes a declared mapping reach the
payload that motivated the feature, without moving validation, collision checks or
examples out of `Spec`, and without changing what a document looks like to a
caller that never finalizes. The cost is deriving payloads twice, which is
rounding error for a document generator, plus one implicit rule: `Finalize` must
run on a catalog document before it is merged, because the record is attached to
the document object rather than to its contents. The harness already finalizes
each catalog before merging.

## Detailed design

### 1. `schema/mapping.go` (new)

```go
// mappingRegistry maps pkgPath.TypeName to the schema that overrides
// derivation for that type. Written by MapSchema/MapSchemaByName (built-ins,
// user init() and generated harness main()), read by FromType. Not safe for
// concurrent use, like registry in registry.go: registration finishes before
// derivation.
var mappingRegistry = map[string]*spec.Schema{}

// aliasRegistry maps a derived alias FQN to the FQN it was declared from
// ("pkg.OrderID" -> "github.com/gofrs/uuid.UUID"), emitted by discovery.
var aliasRegistry = map[string]string{}

// consulted records every FQN resolved through mappingRegistry, for
// MappingWarnings; builtins records where a mapping came from, so built-ins
// can be excluded from warnings (D7).
var consulted = map[string]bool{}
var builtins = map[string]bool{}

func MapSchema[T any](fn func() *spec.Schema)
func MapSchemaByName(fqn string, s *spec.Schema)
func MapAlias(aliasFQN, baseFQN string)          // runtime form of a discovery edge
func MapSchemaJSON(fqn, jsonSchema string) error // unmarshal into spec.Schema
func MappingWarnings() []string                  // registered user mappings minus consulted, sorted
func exactMapping(t reflect.Type) *spec.Schema   // D2 tier 1 only; nil when unmapped
func mappingFor(t reflect.Type) *spec.Schema     // D2 tiers 1 and 3 (alias chain); nil when unmapped
```

`MapSchema` stores the function's *result* at registration time (it is called
once, eagerly — the purity contract makes laziness pointless, and eager
calling surfaces impure/panicking functions at registration with a clear
stack). Two lookups exist because D2 registers the provider tier *between* them:
`exactMapping` answers tier 1 and nothing else, while `mappingFor` also follows
the alias chain (tier 3) with a depth guard, derefs pointers, and marks the
resolved FQN consulted. A nil schema passed to `MapSchemaByName` panics — the
config path cannot produce one and the API path is a programmer error, and it is
the one place the library panics deliberately: a silently ignored mapping would
fail at derivation, far from the mistake.

Payload re-derivation (D10) lives in `schema/payload.go`:

```go
func FromTypePayload(doc *spec.AsyncAPI, t reflect.Type, defs map[string]*spec.Schema) *spec.Schema
func rederivePayloads(doc *spec.AsyncAPI) // called by Finalize
```

### 2. `schema/derive.go` — consultation points

In `FromType`, immediately after `t = deref(t)` and the nil check (D2 tier 1):

```go
if m := exactMapping(t); m != nil {
    return m // already a copy: callers must not share the node
}
```

The named-non-struct provider block stays next (D2 tier 2). A `mappingFor`
call — the D8 alias lookup, D2 tier 3 — goes after it, before the
`rawMessageType`/byte-slice/kind switch, for every kind except a struct (a
struct's provider is checked in `hoistStruct`, which consults `mappingFor`
after its provider check, so a mapped struct-alias keeps its provider
precedence). Everything else stays as-is.
The `timeType` case and its `timeType` var are deleted; `init()` in the
package registers the built-in:

```go
func init() {
    MapSchemaByName("time.Time", &spec.Schema{Type: "string", Format: "date-time"})
}
```

Note: built-in `time.Time` derives inline today and stays inline (D3), so no
observable behavior changes — but registered types that are `time.Time`-valued
combinator members would now get a hoisted `time.Time` component. No combinator
directive can name `time.Time` today unless a user writes it; acceptable and
consistent.

### 3. `schema/registry.go` — Finalize hoists registered mapped types

In `Finalize`, before the existing `FromType` loop, for each registered name:

```go
for _, name := range names {
    if s := mappingFor(registry[name]); s != nil {
        defs[name] = s // a copy already; D3: combinator $refs resolve
        continue
    }
    FromType(registry[name], defs)
}
```

`mappingFor` (not a direct registry hit) so a combinator directive naming an
alias — `oneOf=OrderID` — hoists the base type's mapping under the alias's
own FQN, and the alias FQN is marked consulted for D7.

`Finalize` has a second job (D10): before that loop it calls
`rederivePayloads(doc)`, which re-derives every payload `FromTypePayload`
recorded, rewrites the payload node in place, replaces the components the
payload derivation hoisted and drops the ones a mapping made obsolete. A mapped
payload type therefore ends up inline in the message (D3) and absent from
`components.schemas` — unless a combinator directive names it, in which case the
loop above re-hoists it.

### 4. `internal/mappingconfig` (new) — one whitelist parser, two consumers

Package holding the config surface: `Parse(data []byte) (*Config, error)` —
locate/parse `mappings:`, validate every FQN (same shape rule as combinator
directives: `pkgPath.TypeName` — semi-qualified names rejected), validate
value keys against the D6 whitelist, and expose each mapping as compact JSON
for emission. Imported by `schema` (embedded built-ins) and
`internal/discovery` (user config); it depends only on `spec`.

### 5. `schema/builtin_mappings.yaml` — embedded built-ins

The D9 table as YAML in the D6 shape, loaded via `go:embed` in the package
`init()` before anything user-controlled registers. Adding a library is a
data-only change to this file plus a verification note.

### 6. `internal/discovery` — config, alias scan, and harness emission

`config.go` wraps `mappingconfig.Parse` for `<dir>/asyncgo.yaml`; invalid
config aborts generation in `buildDoc` before any harness runs.

New `aliases.go` scans the reachable module packages (the same walk as the
catalog scan, so no extra pass over syntax): for each `GenDecl` with `Tok ==
token.TYPE`, each `TypeSpec` whose type expression is an `Ident`,
`SelectorExpr`, or a `*T` of either, resolve through `TypesInfo.Uses` to a
`*types.TypeName` that is a defined type (`IsAlias() == false` — Go type
aliases need no edge) and emit `aliasFQN → baseFQN`, deduplicated and sorted.

`materialize.go`'s `harness()` gains two emission blocks before the
`schema.Register(...)` block:

```go
if err := schema.MapSchemaJSON("github.com/gofrs/uuid.UUID",
    `{"type":"string","format":"uuid"}`); err != nil { panic(err) }
schema.MapAlias("github.com/acme/orders.OrderID", "github.com/gofrs/uuid.UUID")
```

Emitting JSON and unmarshaling into `spec.Schema` avoids hand-rolling Go
literals for twenty keywords (regex escaping, `[]any` for `enum`) — and the
whitelist itself is enforced by `mappingconfig`, not by the runtime call.
Naming the file `asyncgo.yaml` keeps it next to the artifact it shapes.

### 7. `internal/cli` — no change

`generate` and `check` already funnel through `buildDoc` → discovery; the
config is picked up there. No flags in v1 (D6).

## Example (before / after)

Input:

```go
import "github.com/gofrs/uuid"

type OrderPlaced struct {
    ID    uuid.UUID `json:"id" asyncapi:"required"`
    Refs  []uuid.UUID `json:"refs"`
}
```

With v1's built-in mappings (D9), `gofrs/uuid.UUID` is covered with zero
configuration — no `asyncgo.yaml`, no API call. The config file is for
libraries that are not built in (or for overriding a built-in), e.g. mapping
an imaginary decimal library:

```yaml
mappings:
  github.com/example/money.Money:
    type: string
    pattern: '^-?\d+\.\d{2} [A-Z]{3}$'
```

Before (kind derivation — `uuid.UUID` is a named `[16]byte`):

```yaml
github.com/acme/orders.OrderPlaced:
  type: object
  properties:
    id:
      type: array
      items: { type: integer }
    refs:
      type: array
      items:
        type: array
        items: { type: integer }
  required: [id]
```

After:

```yaml
github.com/acme/orders.OrderPlaced:
  type: object
  properties:
    id:    { type: string, format: uuid }
    refs:
      type: array
      items: { type: string, format: uuid }
  required: [id]
```

## Edge cases

- **Derived alias** — `type OrderID uuid.UUID` inherits the base mapping via
  the discovery-emitted edge (D8): `OrderID` derives `{type: string, format:
  uuid}` with no extra declaration. An exact mapping or `SchemaProvider` on
  the alias itself outranks the inheritance (D2 tiers 1–2).
- **Alias chains** — one edge per declaration; `type A OrderID` follows
  `A → OrderID → uuid.UUID` at derivation time, so an API mapping on the
  intermediate `OrderID` applies to `A` too (D8). Chains stop at the module
  boundary: alias declarations inside third-party packages are not scanned.
- **Go type aliases** — `type A = B` introduces no new type; no edge, no
  special case, `reflect` already sees `B`.
- **Mapped struct alias** — `type Money2 SomeMappedStruct` keeps tier
  precedence: its own provider (if any) beats the inherited mapping; the
  alias check in `hoistStruct` runs after the provider check (D2 note).
- **Combinator directive naming an alias** — `oneOf=OrderID` resolves because
  `Finalize` hoists via `mappingFor`, which follows the chain and hoists the
  base mapping under the alias's own FQN.
- **Pointer fields** — `*uuid.UUID` derives the mapped schema: `mappingFor`
  derefs before lookup, matching `FromType`'s deref.
- **Mapped named struct** — bypasses `hoistStruct`; the mapped schema is
  inlined at every use site and the struct's fields are never derived (D2, D3).
- **Mapped type in a combinator directive** — the directive's `$ref` resolves
  because `Finalize` hoists registered mapped types (D3); the inline use sites
  carry the same schema as the component.
- **Mapped payload type** — a payload whose whole type is mapped derives inline
  in the message, and the component the provisional derivation hoisted is
  dropped when nothing else references it (D10). A combinator directive naming
  the type re-hoists it, so the `$ref` still resolves.
- **Payload examples** — `PayloadExamples` are written onto the schema `Spec`
  derived; `Finalize`'s re-derivation carries them across rather than losing
  them with the replaced node (D10). A mapping that declares its own `examples`
  is overridden by the message's, which is the more specific declaration.
- **In-process callers** — `Spec` derives a provisional payload; `schema.Finalize`
  is what applies declared mappings, and each catalog document must be finalized
  before it is merged into another.
- **Built-in not triggered** — a built-in mapping only fires when `FromType`
  meets a type with that exact FQN; a module that does not import the library
  sees no effect and no warning (D7 excludes built-ins).
- **Library behavior differs from the built-in** — the shopspring case: if
  the user flips `MarshalJSONWithoutQuotes`, the built-in `{type: string}`
  mapping no longer matches the wire format. Built-ins encode the library's
  default behavior; a project in that situation overrides the FQN in its own
  config (D5: config wins).
- **Named interface** — name-keyed lookup works for named interface types
  (no zero-value assertion is involved, unlike the provider path); a mapping
  turns the default unconstrained schema into the mapped one.
- **Mapping declared via API in a package the harness does not import** — the
  mapping never runs; `MappingWarnings` (D7) surfaces it. Documented rule: put
  API mappings in the catalog package or any package it imports.
- **Duplicate registration** — last registration wins (D5): `init()` order
  first, then config emissions in `main()`.
- **Mapping for an unregistered type** — works without any `Register` call;
  consultation happens on demand wherever `FromType` meets the type.
- **Schema sharing** — the registry returns a clone per consultation;
  mutating a returned schema cannot corrupt other use sites.

## Rollout plan

1. **Stage 0 — registry and derivation** (`feat(schema)`):
   `schema/mapping.go` (exact mappings, alias registry, `mappingFor`),
   `FromType` consultation points, `Finalize` mapped-type hoisting, payload
   re-derivation (`schema/payload.go`), built-in `time.Time` mapping.
   Independently usable via the Go API; no discovery/CLI change.
2. **Stage 1 — built-ins** (`feat(schema)`): `internal/mappingconfig`,
   embedded `builtin_mappings.yaml`, the D9 table. Zero dependencies, zero
   effect on modules not using the libraries.
3. **Stage 2 — config path** (`feat(discovery)`): `asyncgo.yaml` parsing,
   whitelist validation, harness emission of `MapSchemaJSON`, warnings folded
   into harness errors output.
4. **Stage 3 — alias edges** (`feat(discovery)`): `aliases.go` scan and
   `MapAlias` emission into the harness.
5. **Stage 4 — documentation** (`docs`): README section on third-party types,
   godoc contracts on the mapping functions, entry in this doc's index flipped
   to Accepted once reviewed.
6. **Stage 5 — deferred**: `--config` flag override, kind-based mapping
   hooks, config sections beyond `mappings`, opt-out of alias inheritance,
   mapping an *existing* component by `$ref` instead of an inline schema.

## Testing plan

- **Stage 0** (`schema/mapping_test.go`, `schema/derive_test.go`):
  - `should_map_exact_type_inline`
  - `should_map_pointer_to_mapped_schema`
  - `should_map_transitive_slice_and_map`
  - `should_map_inside_an_anonymous_struct`
  - `should_win_over_schema_provider`
  - `should_inline_mapped_named_struct_without_hoisting`
  - `should_hoist_registered_mapped_type_in_finalize`
  - `should_resolve_combinator_ref_to_registered_mapped_type`
  - `should_replace_mapping_on_reregistration`
  - `should_panic_on_a_nil_schema` / `should_panic_on_an_unnamed_type`
  - `should_derive_time_as_builtin_mapping`
  - `should_override_builtin_mapping`
  - `should_derive_byte_slice_unmapped`
  - `should_clone_mapped_schema_per_consultation`
  - `should_report_mapping_warnings` (registered, never consulted)
  - `should_exclude_builtins_from_mapping_warnings`
  - `should_warn_about_an_unused_override_of_a_builtin`
  - `should_follow_alias_chain_to_mapping`
  - `should_prefer_alias_own_mapping_over_inherited`
  - `should_follow_transitive_alias_chain`
  - `should_stop_alias_chain_at_missing_edge`
  - `should_stop_on_a_cyclic_alias_chain`
  - `should_prefer_a_provider_over_an_inherited_mapping`
  - `should_hoist_alias_mapping_for_combinator_ref`
- **Stage 0, payload re-derivation** (`schema/payload_test.go`, `asyncgo_test.go`):
  - `should_rederive_a_payload_with_the_mapping_registered_later`
  - `should_inline_a_payload_whose_whole_type_is_mapped`
  - `should_drop_a_component_a_mapping_made_obsolete`
  - `should_preserve_payload_examples_on_a_hoisted_payload`
  - `should_preserve_payload_examples_on_an_inline_payload`
  - `should_rewrite_the_payload_node_in_place`
  - `should_consume_the_deferred_derivation`
  - `TestFinalizeRedrivesPayloadWithDeclaredMapping` — the same order a catalog
    goes through, from `Spec` to `Finalize`
- **Stage 1** (`schema/mapping_test.go`, `internal/mappingconfig`):
  - `should_load_builtin_mappings_at_init`
  - `should_map_gofrs_uuid_to_string_format_uuid`
  - `should_map_google_uuid_to_string_format_uuid`
  - `should_map_ulid_with_crockford_pattern`
  - `should_map_decimal_to_string`
  - `should_map_big_int_to_integer` — through the registry, since asyncgo does
    not import the libraries; `should_map_big_int_through_derivation` covers a
    real stdlib type end to end
  - `should_reject_unknown_keyword_with_allowed_list`
  - `should_reject_semi_qualified_mapping_name`
  - `should_reject_nested_object` / `should_reject_a_value_that_is_not_a_schema`
  - `should_accept_every_whitelisted_keyword`
- **Stage 2** (`internal/discovery`):
  - `should_load_mappings_from_asyncgo_yaml`
  - `should_reject_an_invalid_config` (message names the file)
  - `should_return_no_mappings_without_a_config_file`
  - `should_emit_mappings_and_aliases_before_register`
  - `should_report_mapping_warnings` (harness output → stderr, never fatal)
- **Stage 3** (`internal/discovery`):
  - `should_emit_alias_edge_for_defined_type_declaration`
  - `should_emit_alias_edge_for_pointer_base_declaration`
  - `should_emit_alias_edge_for_alias_of_alias`
  - `should_not_emit_edge_for_go_type_alias`
  - `should_not_emit_edge_for_predeclared_base`
  - `should_dedupe_and_sort_alias_edges`
- **End to end** (`test/data/mappings`, the `mappings` integration fixture): a
  projected `asyncgo.yaml` mapping, a derived alias, an alias of an alias, a Go
  type alias, a named pointer, and the `math/big.Int` built-in in one payload,
  asserted by the golden comparison and by `asyncapi validate` in a container.
- **Race**: `go test ./... -race` per AGENT.md.

## Open / deferred

- Should `MappingWarnings` be an error under a strict flag (`check --strict`)
  rather than a warning?
- Whether `MapSchema` should accept a value alongside the function form, once
  real usage shows which dominates.
- Opt-out of alias inheritance for a single alias (a sentinel "no mapping"
  registration), if a real case appears.
- Config-driven mapping of combinator directives or tag behavior, when a
  second config section materializes.
- Growing the built-in set (e.g. `rs/xid`, `segmentio/ksuid`): data-only
  changes to `builtin_mappings.yaml`, each requiring a verified-marshal note
  per library version.
