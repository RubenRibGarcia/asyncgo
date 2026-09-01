# Design: Fluent DSL validation with a Result type

- **Status**: Proposed
- **Created**: 2026-09-01
- **Status updated**: 2026-09-01
- **Scope**: root DSL (`asyncgo`: `doc.go`, `message.go`, `bindings.go`), `internal/discovery/`, `test/data/`, `docs/`

## Summary

Give the fluent DSL a first-class way to report that a catalog was assembled
incorrectly. `Item.apply` gains an `error` return, and `Spec` returns a new
`Result` value that bundles the assembled `*spec.AsyncAPI` with the validation
errors found. The discovery harness materializes `Result` and reports failures
as a per-catalog report, so `asyncgo generate` / `asyncgo check` fail with a
clear, contextualized message at generation time.

Two fixes land together:

- `server.host` is REQUIRED by AsyncAPI 3.1.0 but was modeled as optional
  (`Server(name, protocol).Host(h)`); it becomes a required constructor
  argument, `Server(name, protocol, host)`.
- Validation failures (including an empty `host` passed to the constructor) are
  returned from `apply` and rendered per catalog, listing the specific
  violations found.

## Background

### Current shape of the DSL

```go
type Item interface { apply(b *builder) }          // doc.go
func Spec(items ...Item) *spec.AsyncAPI            // doc.go
```

`apply` mutates a `builder` and returns nothing; `Spec` returns only the
document. There is no channel for a builder to say "this item was invalid".

### How catalogs reach the CLI

Discovery locates **exported package-level variables of type `*spec.AsyncAPI`**
(`internal/discovery/discover.go`, `isAsyncAPI`), then a generated harness
imports those packages and serializes the already-built values to YAML
(`internal/discovery/materialize.go`). The harness is a separate `go run`
process; its stdout is the only transport back to `discovery.Build`.

Two consequences constrain every decision below:

1. A package-level `var` holds **one** value, so a `(*spec.AsyncAPI, error)`
   return from `Spec` cannot be assigned to a single catalog variable.
2. Errors produced while building a catalog must be carried **inside** the
   catalog value itself (or rebuilt by the harness) — there is no other path to
   the caller.

### Required fields in AsyncAPI 3.1.0

| Object | Required fields |
| ------ | --------------- |
| Info | `title`, `version` |
| Server | `host`, `protocol` |

`Info(title, version)` and `Server(name, protocol)` already capture `title`/
`version`/`protocol` as constructor arguments; `host` is the odd one out — it
lives only behind an optional fluent setter. Making `host` a constructor
argument closes that gap at compile time, but an empty string can still be
passed, so a runtime check remains necessary. The DSL also lets the other
required constructor arguments be empty strings, which the type system cannot
prevent.

## Goals / Non-goals

**Goals**

1. `Spec` can report validation errors rather than dropping them on the floor.
2. Errors surface at generation time as a per-catalog report that names the
   catalog and the specific violations found in it.
3. `server.host` is enforced as required — via a required constructor argument
   (compile-time) plus an empty-string check at `apply` (runtime).
4. The mechanism is uniform and extensible — adding a future validation is one
   `apply` returning an error, with no discovery/harness changes.
5. Fluent chaining ergonomics are preserved (`Server(...).Variable(...)`).

**Non-goals (v1)**

- Full AsyncAPI 3.1.0 schema validation. Only the required-field checks above.
- Cross-item reference integrity (e.g. `Channel.Servers(sv)` pointing at a
  server never declared via `Servers(...)`).
- `MessageOf(nil)` nil-type guard (currently a latent panic in `schema.FromType`).
- Duplicate server-name / channel-address detection within a single `Spec`.
- Dedicated CLI rendering (color, no `Error:` prefix) — v1 prints the returned
  error via Cobra.

## Design decisions

### D1 — `Item.apply` returns `error`; `Spec` returns `*Result`

```go
type Result struct {
    Doc *spec.AsyncAPI
    Err error
}

type Item interface { apply(b *builder) error }

func Spec(items ...Item) *Result
```

`Spec` accumulates per-item errors with `errors.Join` into `Err`, and also keeps
the individual errors available via `Result.ValidationErrors()` so the harness
can render the per-catalog report:

```go
// ValidationErrors returns the individual validation errors that Err joins,
// or nil when the catalog is valid. The generator harness uses this to render
// the per-catalog report.
func (r *Result) ValidationErrors() []error { return r.errs }
```

`Result.Doc` is always populated (the document is assembled best-effort, but is
only consumed when `Err == nil`).

**Rationale**: a catalog must remain a single package-level `var` so discovery
can keep finding it by type. The alternatives all trade that away or leak the
error:

- **`Spec` returns `(*spec.AsyncAPI, error)`** — impossible to assign to one
  `var`; forces the catalog to become a *function*
  (`func Catalog() (*spec.AsyncAPI, error)`), which changes the discovery
  contract from "value" to "call", or two exported vars (`Catalog`,
  `CatalogErr`) with fragile name-pairing and a leaked symbol. The function
  form is idiomatic but is a larger conceptual shift; `Result` keeps the
  existing "catalog is a value" model (AGENT.md decision #5).
- **A separate `Validate(doc) error` pass** — clean, but the user explicitly
  wants errors returned "when applying the item", and a separate pass can't
  see construction-time context that `apply` has (it re-derives it from the
  flattened document).
- **Attach errors to `spec.AsyncAPI`** — pollutes the plain data model (and
  its YAML/JSON codecs) with build-state; `spec` is documented as having "no
  opinions about how documents are produced".

`ValidationErrors()` (rather than only the joined `Err`) is what lets the
harness report *which* violations a specific catalog has, instead of one opaque
string.

### D2 — `Result` lives in the root `asyncgo` package, with exported fields

`Result` is a DSL artifact, not part of the object model, so it belongs next to
`Spec`, not in `spec`. Exported `Doc`/`Err` fields and the `ValidationErrors()`
method (rather than unexported state) keep the harness — which reads them from
a generated `main` — trivial.

**Rationale**: `spec`'s package contract is "a plain data model"; an error
field violates that. The root package is already the home of the DSL and is
importable by `internal/discovery` with no cycle (root imports `spec`/`schema`,
never `internal/*`).

### D3 — Validate at `apply` time, and accumulate — do not short-circuit

Each `apply` joins all of *its* violations, and container items (`serversItem`,
`channelsItem`) join all of *their* children's errors. A single invalid catalog
therefore reports every violation in one `generate` run.

**Rationale**: fail-fast at the first item would hide later errors and force
multiple round-trips; joining is the same pattern `errors.Join` exists for.

### D4 — `host` is a required `Server` constructor argument; the `Host` setter is removed

```go
func Server(name, protocol, host string) *server
```

`Server(name, protocol).Host(h)` becomes `Server(name, protocol, host)`, and
the `Host` method is deleted. `server.apply` still validates that the three
constructor inputs are non-empty (empty strings remain possible), so
`Server("prod", "kafka", "")` fails at generation time rather than silently.

**Rationale**: making `host` a constructor argument turns omission into a
compile error — the strongest guarantee available — and is consistent with
`protocol`, which is already a constructor argument. A fluent setter for a
REQUIRED field is exactly the shape that produced this bug. The runtime
empty-string check is still necessary because Go cannot enforce non-empty
strings at compile time.

### D5 — Harness transports a structured per-catalog envelope; `Materialize` returns a formatted error

The generated harness emits one outcome per catalog (identity + document +
errors), not a flat list:

```go
type harnessOutput struct {
    Catalogs []catalogOutcome `yaml:"catalogs"`
}

type catalogOutcome struct {
    PkgPath string         `yaml:"pkgPath"`
    VarName string         `yaml:"varName"`
    Doc     *spec.AsyncAPI `yaml:"doc,omitempty"`
    Errors  []string       `yaml:"errors,omitempty"`
}
```

`Materialize` decodes the envelope; if any outcome has `Errors`, it returns a
`CatalogErrors` error (D7) and no documents. Successful docs are `Finalize`d
and returned only when every catalog is valid.

**Rationale**: stdout is the only structured transport back from the harness.
A structured envelope keeps the per-catalog grouping (`pkgPath`/`varName`/
`errors`) intact across the process boundary instead of flattening to one
string, which lets the report be rendered on the Go side (not inside generated
code). Parsing stderr (the current failure path) intermixes `go run` noise and
can't distinguish per-catalog errors.

### D6 — Discovery detects `*asyncgo.Result` (replaces `isAsyncAPI`)

`discover.go` computes the identity from the real type and checks for a pointer
to it:

```go
var resultType = reflect.TypeFor[asyncgo.Result]()

func isResult(t types.Type) bool {
    ptr, ok := t.(*types.Pointer)
    if !ok { return false }
    named, ok := ptr.Elem().(*types.Named)
    if !ok { return false }
    obj := named.Obj()
    return obj.Pkg() != nil && obj.Pkg().Path() == resultType.PkgPath() &&
        obj.Name() == resultType.Name()
}
```

`scanFile` still walks `token.VAR` declarations — only the type predicate
changes. This adds a root-package import to `internal/discovery`.

**Rationale**: the catalog's type changes, so the detector must change with it;
deriving the identity via `reflect.TypeFor` (not hard-coded strings) mirrors
the existing `isAsyncAPI` and stays correct if the module path changes.

### D7 — Validation failures are rendered as a per-catalog report

`internal/discovery` defines a typed error that carries the grouped violations:

```go
type CatalogErrors []CatalogError

type CatalogError struct {
    PkgPath string
    VarName string
    Errors  []string
}

func (e CatalogErrors) Error() string {
    var b strings.Builder
    fmt.Fprintf(&b, "invalid AsyncAPI catalog(s): %d\n", len(e))
    for _, c := range e {
        fmt.Fprintf(&b, "\n%s.%s:\n", c.PkgPath, c.VarName)
        for _, msg := range c.Errors {
            fmt.Fprintf(&b, "  - %s\n", msg)
        }
    }
    return strings.TrimRight(b.String(), "\n")
}
```

`Materialize` returns this as the `error`; `Build` and the CLI propagate it, so
the user sees:

```
invalid AsyncAPI catalog(s): 1

example.com/app.Catalog:
  - server "prod": host is required
```

**Rationale**: both dimensions of the report — *which catalog* and *which
violations* — are shown, grouped and without `go run` stderr noise. A typed
error (not a pre-formatted string) keeps the door open for the CLI to render a
dedicated report later; v1 simply prints the returned error.

## Detailed design

### 1. Root DSL — `doc.go`

Add `"errors"` and `"fmt"` imports, the `Result` type, the new `Item` shape,
and the new `Spec`:

```go
// Result is the outcome of building a catalog: the assembled document plus any
// validation errors encountered while applying its items. Doc is always
// non-nil; Err is nil when the catalog is valid.
type Result struct {
    Doc  *spec.AsyncAPI
    Err  error
    errs []error
}

// ValidationErrors returns the individual validation errors that Err joins, or
// nil when the catalog is valid. The generator harness uses this to render the
// per-catalog report.
func (r *Result) ValidationErrors() []error { return r.errs }

func Spec(items ...Item) *Result {
    b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
    var errs []error
    for _, it := range items {
        if err := it.apply(b); err != nil {
            errs = append(errs, err)
        }
    }
    if len(b.defs) > 0 {
        c := b.components()
        maps.Copy(c.Schemas, b.defs)
    }
    return &Result{Doc: b.doc, Err: errors.Join(errs...), errs: errs}
}
```

Every `apply` changes signature. Non-validating leaves return `nil`
(`contentTypeItem`); containers join children:

```go
func (s serversItem) apply(b *builder) error {
    if b.doc.Servers == nil {
        b.doc.Servers = map[string]*spec.Server{}
    }
    var errs []error
    for _, sv := range s {
        if err := sv.apply(b); err != nil {
            errs = append(errs, err)
        }
    }
    return errors.Join(errs...)
}
```

### 2. Root DSL — `Server` constructor + validations

`Server` gains the required `host` argument; the `Host` setter is removed:

```go
func Server(name, protocol, host string) *server {
    return &server{name: name, s: spec.Server{Protocol: protocol, Host: host}}
}
```

Error convention: `<kind> "<name>": <field> is required` (name omitted where
the kind has no name, e.g. `info`).

```go
func (i *infoBuilder) apply(b *builder) error {
    var errs []error
    if i.info.Title == "" {
        errs = append(errs, fmt.Errorf("info: title is required"))
    }
    if i.info.Version == "" {
        errs = append(errs, fmt.Errorf("info: version is required"))
    }
    b.doc.Info = i.info
    return errors.Join(errs...)
}

func (s *server) apply(b *builder) error {
    var errs []error
    if s.name == "" {
        errs = append(errs, fmt.Errorf("server: name is required"))
    }
    if s.s.Protocol == "" {
        errs = append(errs, fmt.Errorf("server %q: protocol is required", s.name))
    }
    if s.s.Host == "" {
        errs = append(errs, fmt.Errorf("server %q: host is required", s.name))
    }
    b.doc.Servers[s.name] = &s.s
    return errors.Join(errs...)
}
```

`channel.apply` adds a single check — `channel: address is required` when
`c.address == ""` — returning it alongside the existing build logic.

### 3. Discovery — `internal/discovery/discover.go`

- Replace the `asyncAPIPkgPath`/`asyncAPIName` vars and `isAsyncAPI` with
  `resultType` + `isResult` (D6).
- `scanFile`'s var walk is unchanged; the `isAsyncAPI(obj.Type())` call becomes
  `isResult(obj.Type())`.
- Update the package doc comment: variables of type `*asyncgo.Result`.

### 4. Harness — `internal/discovery/materialize.go`

`materialize.go` declares the envelope types for decoding; the generated
harness embeds structurally identical types (same `yaml:` tags) and emits one
outcome per catalog:

```go
// generated main, per catalog (unrolled):
o := catalogOutcome{PkgPath: "example.com/app", VarName: "Catalog"}
if errs := pkg0.Catalog.ValidationErrors(); len(errs) > 0 {
    for _, e := range errs {
        o.Errors = append(o.Errors, e.Error())
    }
} else {
    o.Doc = pkg0.Catalog.Doc
}
out.Catalogs = append(out.Catalogs, o)

// after all catalogs:
for _, c := range out.Catalogs {
    if c.Doc != nil { schema.Finalize(c.Doc) }
}
outBytes, err := yaml.Marshal(out)
```

`Materialize` decodes the envelope and folds invalid outcomes into
`CatalogErrors`:

```go
var out harnessOutput
if err := yaml.Unmarshal(stdout.Bytes(), &out); err != nil {
    return nil, fmt.Errorf("decoding harness output: %v", err)
}
var docs []*spec.AsyncAPI
var catErrs CatalogErrors
for _, c := range out.Catalogs {
    if len(c.Errors) > 0 {
        catErrs = append(catErrs, CatalogError{
            PkgPath: c.PkgPath, VarName: c.VarName, Errors: c.Errors,
        })
    } else {
        docs = append(docs, c.Doc)
    }
}
if len(catErrs) > 0 {
    return nil, catErrs
}
return docs, nil
```

`Build` propagates `CatalogErrors` unchanged (validation errors must not be
obscured by a `"materializing catalogs:"` wrapper); transport/compile errors
keep their existing `%w` context. The CLI's Cobra layer prints the returned
error, so the report reaches the user as-is (Cobra prefixes `Error:`).

## Example (before / after)

Before — the bug is silent, and `host` is an optional setter:

```go
var Catalog = asyncgo.Spec(
    asyncgo.Info("Orders Service", "1.0.0"),
    asyncgo.Servers(asyncgo.Server("prod", "kafka")), // host forgotten
)
```

```console
$ asyncgo generate .
wrote asyncapi.yaml (1 catalog(s))   # emits servers.prod WITHOUT host
```

After — `host` is a constructor argument, so forgetting it is a compile error,
and passing an empty value is reported at generation time:

```go
var Catalog = asyncgo.Spec(
    asyncgo.Info("Orders Service", "1.0.0"),
    asyncgo.Servers(asyncgo.Server("prod", "kafka", "broker:9092")),
)
```

```go
// the only way to produce an invalid host now is an empty string:
asyncgo.Server("prod", "kafka", "")
```

```console
$ asyncgo generate .
Error: invalid AsyncAPI catalog(s): 1

example.com/app.Catalog:
  - server "prod": host is required
```

## Edge cases

- **Host omitted** — now a compile error (`Server` requires three arguments);
  no runtime path can omit it.
- **Empty host / protocol / name** — still possible at runtime (empty strings);
  each is reported by `server.apply`.
- **Multiple violations, one catalog** — all listed under that catalog
  (`server "prod": host is required` and `server "prod": protocol is required`).
- **Multiple catalogs** — each invalid catalog gets its own
  `<pkgPath>.<VarName>:` block in the report; valid catalogs are absent from it.
- **Empty server name** — `Server("", "kafka", ...)` now errors; previously it
  wrote a `""` map key and would also emit a broken `#/servers/` `$ref` from
  `Channel.Servers`.
- **Document built on error** — the partially-assembled `Doc` is discarded
  because `Materialize` returns an error; it never reaches `Merge`/encode.
- **`errors.Join` with all-nil** — returns `nil`, so valid catalogs carry
  `Err == nil` and an empty `ValidationErrors()`; the harness emits `Doc` only.
- **Raw `*spec.AsyncAPI` vars** — hand-built `var X = &spec.AsyncAPI{...}`
  catalogs are no longer discovered (only `*asyncgo.Result` is). This is an
  accepted consequence of D6 for v1; the documented way is `Spec`.
- **No catalogs** — `Materialize` still returns `(nil, nil)` early when
  `len(cats) == 0`; unchanged.

## Rollout plan

1. **Stage 0 — `host` into the `Server` constructor** (`feat(dsl)!`): change
   `Server(name, protocol)` → `Server(name, protocol, host string)`; delete
   `Host`. Update fixtures, README, and tests. Breaking, but no behavior change
   for valid catalogs (they inline the host they already passed via `.Host`).
2. **Stage 1 — `Result` plumbing** (`feat(dsl)!` + `feat(internal)`): introduce
   `Result` + `ValidationErrors()`, change `Item.apply` → `error` and
   `Spec` → `*Result` with every `apply` returning `nil`; update `discover.go`
   (`isResult`) and `materialize.go` (structured envelope, `CatalogErrors`).
   Behaviorally neutral — golden YAML is byte-identical and the error path never
   fires. Breaking because the public `Spec` signature changes.
3. **Stage 2 — validations + report** (`feat(dsl)` + `feat(internal)`): add the
   `info`, `server`, and `channel` required-field checks; exercise the
   `CatalogErrors` report end-to-end with a `test/data/invalid` fixture.
4. **Stage 3 — deferred**: cross-item reference integrity, `MessageOf(nil)`
   guard, duplicate-name detection, dedicated CLI rendering.

## Testing plan

- **Stage 0**: fixtures compile with the new `Server(name, protocol, host)`
  signature; integration golden test and `cli_test.go` stay green.
- **Stage 1**: existing tests updated mechanically (`asyncgo_test.go` unwraps
  `res.Doc` + `require.NoError(t, res.Err)`; `doc_test.go` handles `apply`'s
  return; `discover_test.go` renames `TestIsAsyncAPI`→`TestIsResult`;
  `materialize_test.go`'s `TestHarness` asserts the envelope). Golden YAML
  unchanged.
- **Stage 2** (`doc_test.go`, table-driven):
  - `should_return_error_when_server_host_is_empty`
  - `should_return_error_when_server_protocol_is_empty`
  - `should_return_error_when_server_name_is_empty`
  - `should_return_error_when_info_title_is_missing`
  - `should_return_error_when_info_version_is_missing`
  - `should_join_multiple_validation_errors` (one invalid `Spec`, all reported)
  - `should_return_nil_error_for_valid_catalog`
  - `should_report_errors_grouped_by_catalog` (`CatalogErrors.Error()` format)
  - New `test/data/invalid` fixture + a CLI test
    `should_fail_generate_when_server_host_is_empty` asserting the report text.
- **Race**: `go test ./... -race` per AGENT.md.

## Open / deferred

- Dedicated CLI rendering: detect `CatalogErrors` and print the report without
  Cobra's `Error:` prefix (and optionally colorized); v1 relies on `Error()`.
- Cross-item reference integrity and the other v1 non-goals.
- Preserving discovery of raw `*spec.AsyncAPI` vars (accepted as dropped in D6).
