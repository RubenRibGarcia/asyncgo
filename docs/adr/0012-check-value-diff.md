# 0012. Explain a stale artifact with a value diff

- Status: accepted
- Deciders: asyncgo maintainers
- Created: 2026-11-12
- Status updated: 2026-11-12

## Context and Problem Statement

`asyncgo check` decides staleness by comparing the committed artifact with the
bytes `generate` would write. When the two differ it reports the one thing that
is cheap to say — that they are out of date — and, at `-v`, one finding per
section key that is `stale` or `missing` (ADR 0009, D9).

That covers a key that appeared or disappeared. It does not cover a key whose
*value* changed: a channel address, a `$ref`, a binding, an enum. Those are
exactly the edits a schema or catalog change produces most often, and today they
surface as `asyncgo: … document is out of date` with nothing behind it, or at
best as a finding that names a section rather than the value. The user's next
step is to regenerate and read the diff by hand.

ADR 0009 D9 put a structural diff out of scope. This ADR revisits that: a
`--diff` flag that walks the two documents and reports value changes is small,
dependency-free, and turns the check failure into the explanation the user
actually needs.

## Decision Drivers

- A stale artifact should say *what changed*, not only *that* something did.
- Default output and the exit-code contract (3 = stale, 1 = broken, 2 = usage)
  must not change — ADR 0009 made `check` a CI contract.
- One implementation for both encodings: YAML, JSON, and the mixed case where
  `-o` names a YAML file while `--format json` picks the encoding.
- No new dependency, and no cost on the quiet path.
- Deterministic, assertable output, like every other CLI surface in this repo.

## Considered Options

- A unified text diff (`---`/`+++`, `@@` hunks, `-`/`+` lines).
- A structured value diff over the generically decoded documents.
- A structured diff over the typed `spec.AsyncAPI` model.
- Shelling out to `git diff --no-index`.

## Decision Outcome

Chosen option: "A structured value diff over the generically decoded documents",
because the document is a mapping tree, so walking keys and values reports a
rename as one removal plus one addition instead of a hundred changed lines — and
it does so without a diff engine.

Concretely:

- **`check --diff` prints one entry per changed key** in the shape findings
  already use — `<path>: <where>: <added|removed|changed> (<values>)` — through a
  new `logging.ReportDiff`, which is the *ungated* counterpart of
  `ReportFindings`: `--diff` is the user asking for this detail, so it prints at
  default verbosity, while findings stay `-v` narration.
- **`--diff` replaces the `-v` section findings** for a stale artifact rather
  than printing both: the value diff is the superset (it also sees an empty
  section, which key comparison misses), and two reports of one change is noise.
  Without `--diff`, `-v` behaves exactly as before.
- **Both sides are decoded generically**, not into `spec.AsyncAPI`, by a new
  `internal/docdiff` package. The typed model cannot see keys it does not carry
  (bindings are arbitrary JSON, `additionalProperties` is a free-form map), and
  the walk must compare *values*, not presence.
- **Decoding tolerates the other encoding.** `--format` picks the encoding and
  `-o` picks the file, so `-o spec.yaml --format json` is a documented
  invocation; the artifact is decoded with the run's format first and the other
  one on failure, so neither is silently reported as unparsable. The typed
  findings path gets the same tolerance, which removes a byte-offset fallback
  that used to fire for a perfectly readable YAML artifact.
- **The walk descends only when both sides have the key.** A removed or added
  subtree is one entry whose value *is* the subtree, and sequences compare as
  values (a reordered `enum` is a change). This is what keeps a rename at two
  entries instead of one per leaf.
- **Numbers compare by value, not by Go type.** goccy/go-yaml yields `uint64`
  for `3` and `int64` for `-7`; encoding/json yields `float64` for both. The same
  number therefore arrives as different types depending on the encoding, and a
  stray "everything numeric changed" report would make the flag useless. JSON is
  read with `UseNumber` so a 64-bit integer keeps its digits.
- **Values render as compact JSON** — format-neutral, so a value reads the same
  whichever encoding produced it, and a quoted string stays distinguishable from
  a number. A value too large for one line is summarised (`{5 keys}`,
  `[12 items]`) rather than cut mid-token: a truncated JSON object is no longer
  JSON, and the path already names the subtree.
- **Two fallbacks keep the flag honest.** An artifact that does not parse
  reports the first differing byte offset. An artifact that decodes to the same
  document — reindented, reordered, different line endings, or carrying YAML
  anchors — reports `values match, encoding differs` instead of printing nothing
  on a failing check.
- **The report is capped at 50 changes per artifact**, with an
  `and N more change(s)` line. Higher than the 10-per-section findings cap
  because the flag is a targeted request, and lower than "unbounded" because a
  regenerated document can legitimately differ in hundreds of keys.
- **Directory mode diffs only the stale members.** A missing artifact and an
  orphan have no second document to compare against, so they keep their single
  finding and the set keeps its one verdict.

Rejected:

- The unified text diff was rejected as noise: reproducing hundreds of
  `-`/`+` lines for a key rename explains less than one `added`/`removed` pair,
  and it needs a line-diff algorithm the project would own and test.
- The typed-model diff was rejected because it cannot see arbitrary binding and
  schema keys, which is where a fair share of real drift lives.
- Shelling out to `git diff --no-index` was rejected because it needs temp files,
  an external binary, and different behaviour on Windows — for a comparison the
  program already has both sides of.

### Consequences

- Good, because a failing `check` now names the keys and values that differ, in
  either encoding, without a regenerate-and-read loop.
- Good, because the default output, the error text, exit code 3, and the `-v`
  findings are unchanged; the flag is additive.
- Good, because one walker covers YAML, JSON and the mixed pair, and its numeric
  comparison is exact rather than float-truncating.
- Good, because the encoding-tolerant decode also fixes the mixed case on the
  existing findings path.
- Bad, because `--diff` and `-v` are two ways to explain the same failure with
  different granularity; the interaction (diff wins) has to stay documented.
- Bad, because generic decoding means the diff sees whatever the decoders
  produce — a YAML value the encoder quotes differently (an exponent such as
  `1e3`, which goccy reads as a *string*) is reported as a value change rather
  than an encoding change. That is honest but surprising the first time.
- Bad, because `internal/docdiff` is a new package with a valuation rule set
  (numbers, sequences, nested mappings) that has to be kept in step with the
  encoders' behaviour.

## More Information

- Amends ADR 0009, D9 ("a generic structural diff engine is out of scope"): the
  scope is now a key/value walk, not a structural or textual diff engine.
- Related: [ADR 0009](0009-verbose-cli-logging.md) — the findings shape and the
  verbosity contract this flag extends
- Related: [ADR 0011](0011-per-catalog-documents.md) — the directory artifact set
  the flag walks per document
