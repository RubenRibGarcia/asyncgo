# 0007. Share the struct-tag grammar between the two passes

- Status: accepted
- Deciders: asyncgo maintainers
- Created: 2026-11-08
- Status updated: 2026-11-08

## Context and Problem Statement

asyncgo reads the same struct tags in two places, in two different processes. The
reflection pass (`schema/`) turns a `reflect.Type` into a Schema Object: field
names from `json` tags, keywords from `asyncapi` directives. The static pass
(`internal/discovery/`) walks `go/ast` over the same source to collect field doc
comments and to learn which combinator types must be registered — it cannot use
reflection, because the CLI never imports the user's packages and reflection
cannot see comments.

Both must agree on three grammars: the `json` field-name rules, the
`asyncapi` directive split (`key` or `key=value`, whitespace-separated), and the
combinator value syntax (`oneOf=A|B`). They were implemented twice and coupled
only by comments saying "mirrors schema.X", so nothing failed when they diverged.

They already had. `schema.jsonName` falls back to the Go field name for a tag that
names nothing (`json:",omitempty"`), matching `encoding/json`.
`discovery.jsonFieldName` returned the empty string for the same tag, and its
caller reads empty as "skip this field" — the signal `json:"-"` uses. A field
declared `Note string \`json:",omitempty"\`` therefore derived into the schema
correctly but silently lost its doc comment from the emitted document. No test
failed; the golden file simply had no description.

## Decision Drivers

- Two hand-written parsers of one grammar will diverge; the only question is when.
- The divergence that already happened loses author-written data silently.
- `schema` is public API; adding grammar helpers there widens the surface the DSL's
  users would be tempted to depend on.
- The two passes need different answers from the same syntax (one sets keywords, the
  other only collects names), so sharing semantics would force the static pass to
  model keywords it has no use for.
- The passes run in different processes, so they cannot share a runtime object;
  only code and tests can be shared.

## Considered Options

- **Extract the syntax into `internal/taggrammar`**, keep semantics in the two
  passes, and pin them together with a conformance test.
- Export the helpers from `schema` and have `internal/discovery` import them.
- Leave the duplication and rely on the "mirrors" comments plus review.
- Merge the two passes into one.

## Decision Outcome

Chosen option: "Extract the syntax into `internal/taggrammar`", because it removes
the drift without widening the public API, and it lets each pass keep the
semantics it actually needs.

`internal/taggrammar` returns syntax only: `Parse(tag) []Directive`,
`Flag(ds, flag) bool`, `Value(ds, key) (string, bool)` and
`JSONName(tag) (string, bool)`. It knows nothing about `required`, `enum=` or
which types a combinator names. `schema` keeps `applyTag`, its combinator helper
and the field loop; `internal/discovery` keeps `resolveCombinatorName` and the
description collector.

The static pass adopts the reflection pass's `json` semantics, which is the fix
for the defect above: `json:",omitempty"` now falls back to the Go field name in
both passes. That is the one place this change can move an emitted document.

## Consequences

- **Good**: one parser per grammar; a `json:"-"` or `oneOf=` change lands once.
- **Good**: the `json:",omitempty"` description loss is fixed, not merely frozen.
- **Good**: `schema`'s exported surface is unchanged.
- **Bad**: one more `internal` package and an indirection in two hot-ish loops
  (parsing is per field, and the result is reused across the three combinator
  lookups exactly as before).
- **Neutral**: a third reader of the grammar (a linter, say) now has an obvious
  home instead of growing a third copy.
- The conformance test over a shared tag corpus is what makes the extraction worth
  its cost: it is the drift guard, not the shared code itself.

## More Information

- Design doc: [`docs/designdoc/shared-tag-grammar.md`](../designdoc/shared-tag-grammar.md)
- Tracking issue: [#47](https://github.com/RubenRibGarcia/asyncgo/issues/47), phase 5 item 5.11
