# Design: Conformance validation against the official 3.1.0 schema

- **Status**: Accepted
- **Created**: 2026-11-01
- **Status updated**: 2026-11-01
- **Scope**: `internal/validation/` (new), `internal/cli/`, `test/integration/`,
  `go.mod`, `README.md`, `AGENT.md`, `docs/`

## Summary

`asyncgo check` compares bytes: it proves the committed artifact matches what the
generator would emit, never that the document is valid AsyncAPI. The only real
validation in the repository is the containerized `asyncapi validate` inside
`test/integration`, which users cannot run against their own module and which —
as D4 shows — does not look at most of the document.

This design adds `asyncgo validate [dir]`, an offline command that builds the
document from the module's catalogs and validates it against a pinned, vendored
copy of the official AsyncAPI 3.1.0 JSON Schema, reporting every violation by its
RFC 6901 JSON Pointer.

## Background

### What exists today

| Layer | What it proves | Where |
| --- | --- | --- |
| `SpecResult.ValidationErrors()` + `builder.validateServerRefs` | catalog-level rules (`info`, `server.host`, dangling `server` `$ref`s) | `doc.go`, `internal/discovery` |
| `asyncgo check` | byte equality with the committed `asyncapi.yaml` | `internal/cli/check.go` |
| `test/integration` | golden, JSON/YAML codec equivalence, `asyncapi validate` in a container | `test/integration/asyncgo_generate_test.go` |

`docs/asyncapi-3.1.0-coverage.md` §6 lists "spec-conformance of the final
document (schema validation)" as ❌ and §8 B14 tracks it.

### Constraints

1. **No run-time fetch.** The issue is explicit: the schema must be vendored or
   pinned. A CLI that reaches the network cannot validate an air-gapped build,
   and the repository already pins the validator CLI image for the same reason.
2. **The schema is a bundle, not a fragment.** At
   `asyncapi/spec-json-schemas@v6.11.1`, `schemas/3.1.0.json` is self-contained:
   112 sub-schemas — every protocol binding included — live inside `definitions`
   with matching `$id`s, and all 178 unique `$ref`s resolve internally. One
   416 KB file, no companion documents, no loader mapping. (The one dangling
   ref, `path/to/user-create.avsc#/UserCreate`, sits inside an `examples` value
   and is never compiled.)
3. **The library's own keys do not match the spec's key pattern.** All 19
   `components.<field>` maps in the schema use
   `patternProperties: {"^[\\w\\d\\.\\-_]+$": …}` with **no**
   `additionalProperties`. `asyncgo` keys hoisted schemas `pkgPath.TypeName`
   (AGENT.md, "Fully-qualified schema names"), which contains `/`, so every
   hoisted entry matches nothing and is silently unvalidated — by the official
   schema *and* therefore by the containerized CLI too. Measured: a catalog whose
   `SchemaProvider` returns `&spec.Schema{Type: "bogus"}` produces a document the
   official schema reports as valid.
4. **Validation is defined over JSON.** JSON Schema constrains the JSON data
   model, not YAML, so the validator must see the JSON encoding regardless of
   what `generate --format` would write.
5. **Error trees are noisy.** Flattening `ValidationError.Causes` to leaves
   includes the failed branch of every `oneOf: [Reference, Object]` union:
   `*kind.Required{Missing: ["$ref"]}` ("this node is not a Reference Object")
   and `*kind.Reference`. Reporting those verbatim would bury the real cause.

### Spec facts

| Claim | Evidence |
| --- | --- |
| The published 3.1.0 schema is draft-07 | `"$schema": "http://json-schema.org/draft-07/schema"` in `schemas/3.1.0.json` |
| Component keys are restricted to `^[\w\d\.\-_]+$` | `definitions/…/3.1.0/components.json` → each `properties.<field>.patternProperties` |
| Bindings are inlined into `definitions` | 112 `definitions` entries, each carrying the matching `$id`; e.g. `bindings/kafka/0.5.0/message.json` |
| mqtt5 and mercure bindings are absent from the bundle | no `bindings/mqtt5/**` or `bindings/mercure/**` definition; `test/data/bindings-unlisted` fails with 5 `additionalProperties` violations |

## Goals / non-goals

**Goals**

1. `asyncgo validate [dir]` exits non-zero on an invalid document and prints one
   line per violation, each carrying the JSON Pointer of the offending node.
2. Validation works offline, from the pinned schema, with no Docker and no
   network.
3. The validator catches what the official schema would catch **if** the
   library's component keys matched the spec's pattern (D4).
4. `generate` and `check` are unaffected.

**Non-goals (v1)**

- Validating a committed artifact (`asyncapi.yaml`/`asyncapi.json`) rather than
  the in-memory document.
- Folding validation into `check` or `generate --validate`.
- Fixing the component-key deviation itself (keys are `pkgPath.TypeName`); that
  would rename every emitted key and is breaking.
- Validating against a schema other than 3.1.0, or writing schemas the 3.1.0
  bundle omits (mqtt5, mercure).

## Design decisions

### D1 — A new `asyncgo validate [dir]` subcommand

`validate` is a sibling of `generate` and `check`, with the same
`MaximumNArgs(1)` directory argument and the same `resolveDir` /
`discovery.Build` preamble.

**Rationale**: `generate` writes, `check` diffs, `validate` judges — three
independent verbs. Folding validation into `generate` would couple writing to
validating and make it impossible to validate without re-emitting; folding it
into `check` would change existing semantics for every user. `check` therefore
stays byte-only, which also keeps the drift gate cheap in CI.

### D2 — Validate the freshly built document, not the committed artifact

`Validate` consumes `*spec.AsyncAPI` from `discovery.Build`, exactly as
`generate` does.

**Rationale**: the failure the issue describes is "a structurally invalid
document is reported as out of date" — the mistake lives in the catalog, and
reporting it before anything is written names the real problem. Validating the
artifact would additionally require a prior `generate` and would conflate "you
have not regenerated" with "your document is invalid".

### D3 — Vendor one self-contained schema file, embedded with `go:embed`

`internal/validation/asyncapi-3.1.0.json`, from
`asyncapi/spec-json-schemas@v6.11.1`, embedded with `//go:embed`; its SHA-256 is
pinned in code and in `PROVENANCE.md`, and a unit test fails if the bytes drift.

**Rationale**: 113 separate binding documents were the alternative; they buy
nothing, since the released `3.1.0.json` is already the bundle (constraint 2).
`go:embed` keeps the binary self-contained (air-gapped installs, `go run`, and
`goreleaser` archives all work unchanged), and the digest pin makes an
accidental or malicious edit to the vendored bytes a test failure rather than a
silent validator change. `PROVENANCE.md` records the tag, digest, retrieval date,
and the exact `curl` used, so re-vendoring is a reviewable commit.

### D4 — Relax component key patterns in memory before compiling

At load time, for each `components.<field>` map that declares a single
`patternProperties` and no `additionalProperties`, copy that pattern's value into
`additionalProperties` on the same map (19 maps). The schema *file* on disk is
never modified.

**Rationale**: without this, constraint 3 makes the command report invalid
hoisted schemas as valid — precisely the bug class the issue exists to catch.
The alternatives were worse:

- *Leave it faithful.* The command would be correct against the published schema
  and useless for `asyncgo`'s own output.
- *Compile each `components` entry separately* against the value schema the
  pattern points to. Same coverage, ~40 lines of extra code, an explicit 19-entry
  table, and pointer re-prefixing; and it cannot `Compile` a sub-schema by URL
  without registering all 112 `$id`s first (one of which collides with the
  built-in draft-07 metaschema), so it needs more moving parts to reach the same
  result.
- *Fix the keys.* Renaming hoisted schemas to satisfy `^[\w\d\.\-_]+$` rewrites
  every `$ref` in every committed fixture. Breaking, and out of scope.

The relaxation only widens which keys are *reachable*; the value is still
validated by the official sub-schema. The deviation is recorded in the ADR and
in the coverage doc, because it means `asyncgo validate` can report violations the
AsyncAPI CLI does not.

### D5 — Report leaf causes by JSON Pointer, dropping union noise

Flatten `ValidationError.Causes` to leaves, drop `*kind.Reference` wrappers, drop a
`*kind.Required{Missing: ["$ref"]}` leaf when the same node also reports another
violation (at the same pointer or deeper), dedupe by pointer, sort, and render
`<pointer>: <message>`.

**Rationale**: the acceptance criterion is a pointer per violation, and
`InstanceLocation` already carries the token path — escaping through
`internal/jsonpointer` turns it into an RFC 6901 pointer, so a channel keyed
`orders/v1` reports `/channels/orders~1v1/bindings/kafka/topic` rather than an
ambiguous path. Constraint 5 explains the two dropped kinds: they are artifacts
of `oneOf: [Reference, Object]`, and the sibling branch reports the real cause.
The sibling *deeper* qualifier matters: the wrapper chain puts the artifact at
the union node while the real cause sits deeper, so a purely local (same-pointer)
rule would keep the artifact. A missing-`$ref` leaf with nothing else at that
node is kept — it is then a real Reference with no target.

Deduping by pointer keeps the report one line per offending node (the draft-07
metaschema emits two leaves for one bad `type`; `kind.Required` already lists all
missing properties in one message), and sorting makes the output deterministic
enough to assert on.

### D6 — `--ignore-missing-bindings` for bindings the published schema omits

The flag suppresses exactly `*kind.AdditionalProperties` violations located at
`<node>/bindings` whose offending property names are all `mqtt5` or `mercure`.
Nothing else is suppressed.

**Rationale**: those two protocols define bindings in their own repositories and
are absent from the 3.1.0 bundle, so a correct document using them fails
validation (5 violations on `test/data/bindings-unlisted`). Silently suppressing
them would hide genuine mistakes in those bindings, and per-project config is
overkill; an explicit flag makes the omission a visible, reviewable decision at
the call site (`asyncgo validate --ignore-missing-bindings`).

### D7 — `santhosh-tekuri/jsonschema/v6`

**Rationale**: draft-07 support (required), no network resolution unless a
loader asks for it, `InstanceLocation []string` on every `ValidationError` (the
whole point of D5), and one indirect dependency (`golang.org/x/text`).
`xeipuuv/gojsonschema` is effectively unmaintained; a hand-written draft-07
validator for a 112-definition schema is not a reasonable maintenance burden; and
shelling out to the containerized CLI cannot work offline, cannot report
pointers the CLI does not surface, and would make a runtime Docker dependency.

## Detailed design

### 1. `internal/validation/validation.go`

```go
package validation

//go:embed asyncapi-3.1.0.json
var schemaJSON []byte

const (
	// schemaRootURL is the $id of the embedded document; AddResource and
	// Compile must agree with it.
	schemaRootURL = "http://asyncapi.com/definitions/3.1.0/asyncapi.json"
	componentsURL = "http://asyncapi.com/definitions/3.1.0/components.json"

	// pinnedSHA256 is the digest of the vendored file (see PROVENANCE.md).
	pinnedSHA256 = "51d3274899ad2875f25c18fd1aef4d5512f0a97be785d519740bde55a4162f61"
)

// Options controls which violations are reported.
type Options struct {
	// IgnoreMissingBindings suppresses violations for protocol binding keys
	// the published 3.1.0 schema omits (mqtt5 and mercure).
	IgnoreMissingBindings bool
}

// Violation is one schema violation, cited by the JSON Pointer of the
// offending node.
type Violation struct {
	Pointer string // RFC 6901 pointer; "/" is the whole document
	Message string
}

// Violations is the report returned for an invalid document.
type Violations []Violation

func (v Violations) Error() string // "document is not valid AsyncAPI 3.1.0:\n  /a/b: <msg>"

// Validate checks doc against the pinned AsyncAPI 3.1.0 JSON Schema.
func Validate(doc *spec.AsyncAPI, opts Options) error
```

Internals, each small and testable on its own:

- `compiledSchema = sync.OnceValues(compile)` — `compile` unmarshals
  `schemaJSON` into `map[string]any`, calls `relaxComponentKeys(root)`, then
  `jsonschema.NewCompiler().AddResource(schemaRootURL, root)` +
  `Compile(schemaRootURL)`.
- `relaxComponentKeys(root map[string]any)` — D4, documented in place with the
  reason and the pointer to this design doc.
- `instanceOf(doc *spec.AsyncAPI) (any, error)` — `doc.JSON()` then
  `json.Unmarshal` into `any`. Plain unmarshal (no `UseNumber`): the integration
  test canonicalizes the same way, and draft-07's integer check accepts a whole
  float.
- `violations(err error, opts Options) Violations` — D5 + D6.
- `pointer(location []string) string` — `"/"` for empty, otherwise
  `"/" + strings.Join(escaped, "/")` using `jsonpointer.Escape`.
- `message(k kind.ErrorKind) string` — `k.LocalizedString(printer)` with a
  package-level `message.NewPrinter(language.English)`.

Failure modes: a schema that will not compile is a bug, not user input, so it is
returned wrapped (`compiling AsyncAPI 3.1.0 schema: %w`) rather than panicking —
consistent with AGENT.md's "no panics in library code".

### 2. `internal/cli/validate.go`

Mirrors `check.go`: `Use: "validate [dir]"`, `Args: cobra.MaximumNArgs(1)`,
`resolveDir` → `discovery.Build` → `validation.Validate`; `CatalogErrors` are
returned unchanged (same `errors.As` block as `generate.go`/`check.go`).
Success prints `<dir> produces a valid AsyncAPI 3.1.0 document`; violations are
returned as the error, so `main` prefixes them with `asyncgo: `. The flag:

```go
cmd.Flags().BoolVar(&ignoreMissing, "ignore-missing-bindings", false,
	"ignore binding keys the published 3.1.0 schema omits (mqtt5, mercure)")
```

`root.AddCommand(newGenerateCmd(), newCheckCmd(), newValidateCmd(), newVersionCmd(bi))`.

### 3. `test/integration/asyncgo_generate_test.go`

A fourth assertion per fixture, on the freshly generated document:

- default fixtures — `assert.Empty(t, validation.Validate(doc, validation.Options{}))`;
- `bindings-unlisted` — assert it *does* report violations **and** that
  `Options{IgnoreMissingBindings: true}` reports none. This is the same
  "assert the exclusion is still needed" guard `cliUnsupported` uses: if the
  published schema ever gains mqtt5/mercure, the first half fails and the
  exclusion is removed deliberately.

The module may import `asyncgo/internal/validation` because its path is nested
under the repository's module path (AGENT.md, "Module layout").

## Example (before / after)

Catalog whose provider returns an invalid schema:

```go
type Bogus struct{ X string }

func (Bogus) AsyncAPISchema() *spec.Schema { return &spec.Schema{Type: "bogus"} }

var Catalog = asyncgo.Spec(
	asyncgo.Info("Bogus Service", "1.0.0"),
	asyncgo.Channels(
		asyncgo.Channel("orders/v1").
			Send(asyncgo.Operation().Message(asyncgo.MessageOf(Bogus{}).Name("Bogus"))),
	),
)
```

Before (and, today, with the containerized CLI too):

```console
$ asyncgo validate .
/…/vcli produces a valid AsyncAPI 3.1.0 document
```

After:

```console
$ asyncgo validate .
asyncgo: document is not valid AsyncAPI 3.1.0:
  /components/schemas/example.com~1vcli.Bogus/type: value must be one of 'array', 'boolean', 'integer', 'null', 'number', 'object', 'string'
$ echo $?
1
```

With the omitted bindings:

```console
$ asyncgo validate ./test/data/bindings-unlisted
asyncgo: document is not valid AsyncAPI 3.1.0:
  /channels/orders-mercure/bindings: additional properties 'mercure' not allowed
  …
$ asyncgo validate ./test/data/bindings-unlisted --ignore-missing-bindings
/…/bindings-unlisted produces a valid AsyncAPI 3.1.0 document
```

## Edge cases

- No catalogs in the module — `discovery.Build`'s "no AsyncAPI catalogs" error,
  unchanged from `generate`/`check`.
- Catalog violates a DSL rule — `CatalogErrors` is reported instead of a schema
  report; the DSL's messages are the more useful ones.
- Component key containing `~` or `/` — escaped into the pointer
  (`~0`/`~1`), so the pointer is unambiguous.
- Root-level violation — pointer `/`.
- A node failing one `oneOf` branch — the `$ref`-branch artifact is dropped; the
  failing branch's real cause is reported. A Reference that really has no target
  is still reported when nothing else explains the node.
- **A missing required property is not reachable from the typed model.** Every
  required property is either serialized unconditionally (`host`, `protocol`,
  `action`, …) or serialized even when empty (`$ref: ""`, which draft-07 accepts
  because `format` is an annotation). `validate` therefore reports wrong types,
  unknown binding keys and invalid nested schemas — not absent properties. Only
  hand-authored documents could produce those, and v1 does not validate them.
- One bad keyword producing two metaschema leaves — reported once.
- Unknown binding key that is not mqtt5/mercure — still a violation with
  `--ignore-missing-bindings`.
- Corrupt or tampered vendored file — the digest test fails before anything else
  is trusted.

## Rollout plan

1. **Stage 0 — schema vendored** (`internal`): schema file, `PROVENANCE.md`,
   digest pin, dependency. No behavior change.
2. **Stage 1 — validator** (`internal`): `internal/validation` + unit tests,
   including the F1/D4 and D5 guards.
3. **Stage 2 — command** (`internal`): `asyncgo validate` + CLI tests + README.
4. **Stage 3 — pipeline** (`test`): fourth integration assertion; coverage doc,
   AGENT.md and the ADR.

## Testing plan

- `internal/validation` — table-driven, `should_*` names:
  - `should_accept_a_minimal_document`
  - `should_accept_a_reference_with_an_empty_target` (required-property boundary)
  - `should_report_a_bogus_schema_type_under_a_qualified_key` (D4)
  - `should_escape_json_pointer_tokens`
  - `should_omit_reference_branch_noise` (D5)
  - `should_report_each_pointer_once`
  - `should_ignore_missing_binding_keys_only_when_asked` (D6)
  - `should_not_ignore_other_binding_keys`
  - `should_report_nothing_beyond_the_suppressed_bindings`
  - `should_return_violations_as_error`
  - `should_reject_a_nil_document`
  - `should_embed_the_pinned_schema`
- `internal/cli` — `TestValidate`, `TestValidateSchemaViolation`,
  `TestValidateCatalogErrors`, `TestValidateNoCatalogs`,
  `TestValidateIgnoreMissingBindings`, help listing.
- `test/integration` — the fourth assertion, per fixture.
- **Race**: `make test` (runs `-race`).

## Open / deferred

- **Component keys vs. the spec pattern** — `pkgPath.TypeName` violates
  `^[\w\d\.\-_]+$`. Fixing it renames every hoisted key and is breaking; tracked
  as follow-up work, not here. Until then D4 is the compensating measure.
- **Validating a committed artifact** — `validate --file <path>` (and the
  `--format` interaction from B15) is deferred until there is a request for it.
- **Format assertions** — draft-07 treats `format` as an annotation; enabling
  assertion would diverge from the AsyncAPI CLI's observed behavior.
- **Upgrading the schema** — a new `spec-json-schemas` tag is a deliberate
  commit: update the file, the digest constant, and `PROVENANCE.md`.
