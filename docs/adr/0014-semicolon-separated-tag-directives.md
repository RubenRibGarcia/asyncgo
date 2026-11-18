# 0014. Separate asyncapi tag directives with `;` and list members with `,`

- Status: accepted
- Deciders: asyncgo maintainers
- Created: 2026-11-18
- Status updated: 2026-11-18

## Context and Problem Statement

The `asyncapi` struct tag started out as a whitespace-separated directive list —
`asyncapi:"required format=uuid"` — and its list directives (`enum`, `oneOf`,
`anyOf`, `allOf`) split their value on `|` — `enum=draft|sent`. Neither character
is what Go authors expect:

- `encoding/json` separates a tag's options with a comma, and so do the tags of
  most schema-from-Go libraries. Whitespace is the outlier.
- `|` appears in no common struct-tag vocabulary, reads as a shell pipe, and is
  easy to mistype as `/` or `l`.
- A tag is a single Go string literal and `gofmt` does not normalize its
  contents, so a space-separated grammar makes alignment padding load-bearing:
  `asyncapi:"required  format=uuid"` is one directive whose key contains spaces,
  and the mistake surfaces only when the static pass names the key.

The grammar is read by two passes in two processes (ADR 0007): reflection in
`schema/` and `go/ast` in `internal/discovery/`. Whatever the separators are,
both must agree, and a tag spelling that a reader of one pass would guess wrong
is a defect waiting to be written.

The operative question is which two characters to use, given that the separator
for directives and the separator for a list directive's members cannot be the
same character without an escape mechanism, a positional rule, or key
repetition.

## Decision Drivers

- A tag must be unambiguous by inspection: no rule about which directive may
  come last, no quoting, no escaping.
- The grammar must fit in one sentence a user can remember after reading the
  reference page once.
- A scalar value should lose as little expressiveness as possible. `examples=`
  exists to carry real-world example values, which contain commas (`1,000`) and
  pipes.
- The two passes must share one implementation of each separator (ADR 0007).
- An old-spelling tag must not silently derive a different document.

## Considered Options

- `,` between directives, list members by repeating the key (`enum=a,enum=b`).
- `,` between directives, list members after the same `,` with the list
  directive required to be last (`required,enum=a,b`).
- `;` between directives, `,` between list members (`required;enum=a,b`).
- `,` between directives, `|` kept for list members (`required,enum=a|b`).
- Keep whitespace and `|`.

## Decision Outcome

Chosen option: "`;` between directives, `,` between list members", because it is
the only syntax that gives comma-separated lists their conventional spelling
while keeping every directive position free and every scalar value verbatim.

```go
type OrderPlaced struct {
	OrderID string `json:"order_id" asyncapi:"required;format=uuid"`
	Sku     string `json:"sku"      asyncapi:"required;enum=sku-1,sku-2"`
	Data    any    `json:"data"     asyncapi:"oneOf=OrderPlaced,OrderCancelled"`
	Note    string `json:"note"     asyncapi:"examples=1,000;deprecated"`
}
```

- Directives are separated by `;`. The first `=` in a directive splits key from
  value, so a value may contain further `=`.
- The value of a list directive (`enum=`, `oneOf=`, `anyOf=`, `allOf=`) is split
  on `,`. An empty member (`enum=a,`) is an error.
- A scalar value is taken verbatim, so `examples=1,000` and `examples=a|b` are
  single values. Neither a scalar value nor a list member may contain `;`, which
  is the one character the grammar reserves.
- A space is not allowed anywhere in the tag. With `;` as the separator there is
  no reason to pad, and rejecting it keeps the old whitespace form from being
  read as one long directive name.
- Both separators live in `internal/taggrammar` (`DirectiveSeparator`,
  `ListSeparator`, `Parse`, `SplitList`, `List`), so the reflection pass and the
  static pass cannot read a list two ways.

* Option "`,` + repeated key" was rejected because `enum=a,enum=b` reads like a
  mistake to anyone who has not read the reference, and a repeated key conflicts
  with the first-match rule `taggrammar.Value` uses for every other directive.
* Option "`,` + list last" was rejected because the ordering rule is invisible
  at the call site: `required,enum=a,b,format=uuid` puts `format=uuid` in the
  enum, and a comma inside a scalar value becomes inexpressible.
* Option "`,` + `|`" was rejected because keeping `|` gives up the reason to
  change at all: it is not a Go struct-tag separator, and the tag then teaches
  two different list conventions for no gain.
* Keeping whitespace was rejected because the padding is invisible in review and
  the resulting error ("unknown directive `required format`") names a key nobody
  wrote.

### Consequences

- Good, because the two levels of the grammar are different characters, so no
  position is special and no escaping exists.
- Good, because `,` in a scalar value and `|` in a scalar value keep their
  literal meaning, which `examples=` needs.
- Good, because both passes split lists through `taggrammar.SplitList`, so the
  drift class ADR 0007 guards against cannot reappear at this level.
- Bad, because a tag now uses two separators instead of one, and `;` is rarer in
  Go tags than `,`.
- Bad, because `;` can no longer appear in a value, so an `examples=` entry
  containing a semicolon is not expressible.
- Bad, because the change is breaking: tags in the old form stop deriving what
  they used to. The module is pre-1.0 (`v0.1.0`), and `schema.CheckTag` reports
  the old forms rather than accepting them silently: a space names the rule, a
  comma in a directive key names `;`, a `|` inside a list names `,`, and a list
  member spelled as a directive (`enum=a,format=uuid`) says the directive was
  swallowed.
- Neutral, because the derived document is unchanged: only the tag spelling
  moves, and the committed golden files stay identical.

## More Information

- Amends: [0007](0007-shared-tag-grammar.md), whose shared-package decision
  stands; only its stated separators are superseded.
- Design doc: [shared-tag-grammar](../designdoc/shared-tag-grammar.md) (written
  for ADR 0007, kept as the record of that decision).
- Reference page: `site/content/reference/struct-tags.md`.
