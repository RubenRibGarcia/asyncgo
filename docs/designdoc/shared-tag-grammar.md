# Design: Shared struct-tag grammar

- **Status**: Proposed
- **Created**: 2026-11-08
- **Status updated**: 2026-11-08
- **Scope**: `internal/discovery/`, `schema/`, `internal/taggrammar/` (new), `docs/`

## Summary

Two passes read the same struct tags with two hand-written parsers: the
reflection pass in `schema/` derives the document from `reflect.StructField`, and
the static pass in `internal/discovery/` reads the same tags from `go/ast` to
collect descriptions and to know which types to register. They are coupled by
three "mirrors" comments and nothing else, and they have already drifted in a way
that loses data. This design extracts the grammar into one package that both
import, and adds a conformance test over a shared tag corpus so the drift cannot
recur.

## Background

The two passes exist because reflection cannot see what the compiler keeps:

| Pass | Input | Sees | Answers |
| --- | --- | --- | --- |
| reflection (`schema/`) | `reflect.Type` | values, tags | what schema does this type derive to |
| static (`internal/discovery/`) | `go/ast` + `go/types` | doc comments, tags, imports | which types to hoist, what descriptions to attach |

Both must agree on three tag grammars:

1. **`json`** — the field name, `-` for skip, options after a comma.
2. **`asyncapi` directives** — whitespace-separated `key` or `key=value` parts.
3. **Combinator values** — `oneOf=A|B`, where `|` separates member names.

Today (after the phase 3 `parseTag` refactor) `schema/tags.go` owns the canonical
version of 2 and 3, and `schema/derive.go` owns 1. The static pass re-implements
all three:

| Grammar | Canonical | Duplicate | Coupling |
| --- | --- | --- | --- |
| `json` name | `schema.jsonName` | `discovery.jsonFieldName` (`descriptions.go:176`) | comment: "mirrors schema.jsonName" |
| combinator | `schema.combinatorNames` | `discovery.combinatorNamesInTag` (`combinators.go:125`) | comment: "mirrors schema.combinatorNames" |
| `asyncapi` split | `schema.parseTag` | — (the static pass only looks for the three combinator keys) | — |

**The duplication has already cost a defect.** `schema.jsonName` falls back to the
Go field name when a `json` tag names nothing (`json:",omitempty"`), matching
`encoding/json`. `discovery.jsonFieldName` returns the empty string for the same
tag, and its caller treats empty as "skip this field" — the same signal as
`json:"-"`. So a field declared `Note string \`json:",omitempty"\`` derives into
the schema correctly but silently loses its doc comment, and the emitted document
drops a description that the author wrote. Nothing fails; the golden file just
has no description. That is the class of bug a shared grammar prevents.

## Goals / Non-goals

**Goals**

1. One implementation of each of the three grammars, imported by both passes.
2. A conformance test that runs one corpus through both passes and asserts
   identical parses, so a future divergence fails CI instead of dropping a field.
3. No behaviour change in the reflection pass; the static pass converges on the
   reflection pass's semantics (including the `json:",omitempty"` fallback).

**Non-goals (v1)**

- Unifying the two *passes*. They read different inputs for different reasons;
  only the grammar is shared.
- Parsing tag grammar for anything other than `json`, `asyncapi` and the
  combinator keys — no `yaml` tag handling, no validation of a directive's value
  (a non-numeric `minItems=abc` stays silently ignored, per D6 of
  `docs/designdoc/schema-keywords.md`).
- Changing which directives exist, or their semantics.

## Design decisions

### D1 — A standalone `internal/taggrammar` package, not exports from `schema`

`schema` is a public package; exporting `ParseTag`, `Directive` and friends would
add API surface that the DSL's users have no reason to depend on, and this repo
has kept its public surface deliberately small. `internal/taggrammar` is
importable by both `schema` and `internal/discovery` (Go's internal rule resolves
on the path prefix, and both live under the same module).

**Rationale**: it keeps the extraction invisible to users of the library and
leaves `schema`'s API exactly as it is.

### D2 — The grammar package parses; the packages that know about tags decide

`taggrammar` returns syntax, not semantics: [`Directive{Key, Value, HasValue}`]
from a tag string, and a field name from a `json` tag. It does not know that
`required` is a flag, that `enum=` splits on `|`, or that `oneOf=` names types —
those stay where they are today (`schema.applyTag`, `schema.combinatorNames`,
`discovery.resolveCombinatorName`).

**Rationale**: the two passes need different questions answered from the same
syntax. Sharing syntax is what removes the drift; sharing semantics would
force the static pass to model keywords it has no use for.

### D3 — The static pass adopts the reflection pass's `json` semantics

`jsonFieldName` goes away in favour of the shared name parser, so
`json:",omitempty"` falls back to the Go field name instead of being treated as
`json:"-"`. This changes emitted documents: a fixture with such a tag gains a
description it currently loses.

**Rationale**: the reflection pass matches `encoding/json`, which is the
authority for what a field is called; the static pass's behaviour is simply
wrong. It is a fix, not a preference, and it is called out in its own commit with
its own golden delta.

### D4 — Conformance is a test over a corpus, not a shared code path with two adapters

The conformance test feeds a corpus of tag strings through
`taggrammar`-backed reflection parsing and through the discovery-side path, and
asserts the two agree. It is a test rather than a runtime assertion because the
passes run in different processes: the static pass runs in the CLI, the
reflection pass in the generated harness.

**Rationale**: the harness/CLI split (see `docs/designdoc/multi-format-schemas.md`)
makes a single shared runtime object impossible without moving Tag parsing into
the harness. A corpus test is the cheapest thing that actually catches drift.

## Detailed design

### 1. `internal/taggrammar` (new)

```go
// Directive is one whitespace-separated part of a struct tag, split at the first
// "=" into a key and its value. HasValue separates a value directive (even one
// with an empty value, as in "format=") from a bare flag (readOnly, allOf).
type Directive struct {
	Key      string
	Value    string
	HasValue bool
}

// Parse splits a struct tag into its directives.
func Parse(tag string) []Directive

// Flag reports whether the tag carries flag as a bare directive.
func Flag(ds []Directive, flag string) bool

// Value returns the value of the first directive with the given key, or
// ok=false when the key is absent or carries no value.
func Value(ds []Directive, key string) (string, bool)

// JSONName returns the field name a `json` tag declares, ok=false when the tag
// does not name one (so the caller falls back to the Go field name), and
// skip=true for "-".
func JSONName(tag string) (name string, skip bool, ok bool)
```

This is `schema/tags.go`'s `directive`, `parseTag`, `hasFlag` and `combinatorNames`
moved, plus `schema/derive.go`'s `jsonName` reshaped. `combinatorNames`'s "split
on `|`, absent when empty" rule stays in `schema` because only the reflection
pass and the combinator collector use it — both call `taggrammar.Value` and then
split.

### 2. `schema/` as a thin caller

`tags.go` keeps `applyTag` (the switch over directive keys) and delegates to
`taggrammar.Parse`; `derive.go`'s `fillFields` calls `taggrammar.Parse` once per
field (as today) and passes the result to `applyTag`, `taggrammar.Flag` and the
combinator helper. `jsonName` is replaced by `taggrammar.JSONName`.

### 3. `internal/discovery/` as a thin caller

- `descriptions.go`: `jsonFieldName` → `taggrammar.JSONName`, keeping the
  `json:"-"` skip.
- `combinators.go`: `combinatorNamesInTag` → `taggrammar.Parse` +
  `taggrammar.Value(key)`, then split on `|` — the same two steps `schema` takes,
  so the two cannot disagree about whitespace or the `=` split.

## Example (before / after)

```go
type Order struct {
	// Optional note from the customer.
	Note string `json:",omitempty"`
}
```

Before — the two passes disagree, and the description is lost:

```yaml
# reflection pass: property name "Note", description missing (static pass skipped it)
Note:
  type: string
```

After — one grammar, one name, description attached:

```yaml
note:
  type: string
  description: Optional note from the customer.
```

(The name also becomes `note`: `encoding/json` honours the field name, and both
passes now agree that the tag declared no name.)

## Edge cases

- `json:"-"` — skip, in both passes; `JSONName` returns `skip=true`.
- `json:",omitempty"` — fall back to the Go field name, in both passes. This is
  the divergence D3 fixes.
- `asyncapi:""` and `asyncapi:"   "` — no directives; `Parse` returns nil.
- `format=` (empty value) — a value directive with an empty value; `Value`
  reports `ok=true`, `""`. `applyTag` decides what an empty value means.
- `allOf` versus `allOf=A` — the bare form is the embed marker (`Flag`), the
  valued form is a combinator (`Value`). Phase 3's `hasFlag` already draws this
  line; it moves unchanged.
- A directive with several `=` (`examples=a=b`) — split at the first, value
  `a=b`, as today.
- Legacy comma-separated tags (`required,readOnly`) — one unknown directive,
  ignored, as today (pinned by `TestApplyTag`).

## Rollout plan

1. **Stage 0 — design** (`docs`): this document. No code.
2. **Stage 1 — ADR** (`docs`): record the accepted decisions.
3. **Stage 2 — extraction, no behaviour change** (`refactor(schema)`): add
   `internal/taggrammar`, make `schema` its first caller. `internal/discovery` is
   untouched, so no golden moves and no pass can disagree yet in a new way.
4. **Stage 3 — static pass adopts it** (`fix(internal)`): `descriptions.go` and
   `combinators.go` call `taggrammar`; `jsonFieldName` and
   `combinatorNamesInTag` are deleted. Any golden delta here is the D3 fix and is
   described in that commit.
5. **Stage 4 — conformance test** (`test(internal)`): the corpus test that pins
   the two passes together.

Each stage is independently mergeable; only stage 3 can move a golden file.

## Testing plan

- **Stage 2**: `schema`'s existing tests must pass unchanged — the extraction
  moves code, it does not change it. Plus `taggrammar`'s own table tests, moved
  from `schema/tags_test.go`:
  - `should_split_each_part_at_its_first_equals`
  - `should_return_nil_for_an_empty_tag`
- **Stage 3**: `should_keep_a_description_for_a_field_with_only_json_options`
  (the D3 fix), and the existing `internal/discovery` tests unchanged.
- **Stage 4**: `should_parse_the_same_corpus_through_both_passes` — one table of
  tag strings, both parsers, identical results; and
  `should_report_the_divergent_tag` so a failure names the offending string.
- **Race**: `go test ./... -race` per AGENT.md.

## Open / deferred

- **`yaml` tags.** `spec/bindings_test.go` asserts that `json` and `yaml` tags on
  binding structs agree. That is a model invariant, not derivation grammar, and
  it is not part of this extraction.
- **Directive validation.** Unknown directives and malformed values stay silently
  ignored (D6 of the schema-keywords design). Making them loud is follow-up work
  tracked there, not here.
- **A third reader.** If a linter or a generator ever needs the same grammar, it
  should import `internal/taggrammar` rather than grow a fourth copy — which is
  the reason the package exists rather than two exported helpers in `schema`.
