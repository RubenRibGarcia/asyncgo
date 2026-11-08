# Design: Verbose logging for the asyncgo CLI

- **Status**: Accepted
- **Created**: 2026-11-08
- **Status updated**: 2026-11-08
- **Scope**: `internal/cli/`, `internal/discovery/`, `cmd/asyncgo/`, new `internal/logging/`

## Summary

`asyncgo generate` and `asyncgo check` report one line on success and an error
on failure; everything between the two flags being parsed and the artifact
being written is a black box. Add a repeatable `-v` counter to the root command
so a run narrates what it did: `-v` prints the stage story (config resolved,
packages loaded, catalogs found, harness compiled, documents merged, bytes
encoded), `-vv` adds the internals (per-package reachability, per-catalog
origin, each mapping/ref/alias/description hit, harness build output) and an
end-of-run timing recap.

Output is **stage-tagged prose** on stderr — one record per line, no
`key=value` debris — with typed attributes carried alongside each record so a
machine-readable handler stays a drop-in later. Problems (check drift, unused
mappings, merge collisions) render as **compiler-style findings**, one located
`path: section.key: problem` line each. Verbose output never changes the
artifact, the exit codes, or the wording of an error.

One correctness-adjacent behaviour change comes with it: `Merge` stops dropping
duplicate keys in silence, and a failing harness keeps its generated program on
disk so the thing that failed can be read.

## Background

### The pipeline a single run walks

`generate` and `check` share one path; `check` adds a byte comparison at the
end. Every stage below is a place a user can be confused and get no signal.

| # | Stage | Code | Signal today |
| --- | --- | --- | --- |
| 1 | `--format` → encoder | `cli/generate.go` `resolveFormat` | error only |
| 2 | `dir` → absolute path | `cli/root.go` `resolveDir` | error only |
| 3 | mapping config read + parsed | `discovery/build.go` `loadMappingConfig` | error only |
| 4 | `packages.Load(./...)` + type-check | `discovery/discover.go` `load` | error only |
| 5 | reachable-from-main filter | `discover.go` `reachableFromMain` | none |
| 6 | catalog scan (`*asyncgo.SpecResult` vars) | `discover.go` `scanCatalogs` | count, at the end |
| 7 | combinator refs (`oneOf`/`anyOf`/`allOf`) | `combinators.go` | error only |
| 8 | alias edges | `aliases.go` | none |
| 9 | harness: mkdir, write `main.go`, `go run` | `materialize.go` `Materialize` | **none** |
| 10 | per-catalog validation | harness → `materialize.go` | failures only |
| 11 | `Finalize` hoisting, mapping warnings | harness | warnings on stderr |
| 12 | merge, first-wins map union | `merge.go` `Merge` | **silent drops** |
| 13 | doc-comment descriptions | `descriptions.go` | none |
| 14 | encode yaml/json | `cli/generate.go` | none |
| 15 | atomic write / byte compare | `writeFileAtomic`, `cli/check.go` | one line |

### Why the current output is insufficient

**Two of the slowest stages are the least observable.** Stage 4 type-checks the
module; stage 9 shells out to `go run` and *compiles* it. A cold cache, a hung
toolchain, a stale `go.sum` and a genuinely slow disk all present identically
as silence. Nothing distinguishes "working" from "stuck".

**The harness's stderr is discarded on success.** `materialize.go:137-140`
buffers the subprocess's stderr and only includes it in the returned error.
A compiler warning, a `go vet` note, or a package `init()` writing to stderr on
a run that otherwise succeeds is lost. The generated program is deleted by the
deferred `removeHarnessDir` *before* the caller can look at it, including on
failure — so the one artifact that explains a broken catalog is unreadable
precisely when it matters.

**Merge drops duplicates silently.** `merge.go:61-71` unions maps first-wins.
Two catalogs each declaring `channels.orderCreated` produce a document
containing one of them and no indication the other was discarded. This is not
noise: the merged document is wrong for one of the two catalogs, and today
nothing says so.

**Problems have no common shape.** A check failure is one prose sentence; a
mapping warning is another (`asyncgo: schema mapping for X was never used`);
a merge collision would be a third. All three are "something at path X is wrong
in way Y", and none of them can be located or parsed.

**`discovery` writes to the process's stderr directly.** `materialize.go:123`
and `:146` bypass `cmd.ErrOrStderr()`. `cli_test.go`'s `execute` helper captures
stderr through `root.SetErr`, so those two lines are invisible to the test
suite, and a caller that redirects a command's error stream gets no mapping
warnings. Any logging design has to fix this to be testable.

**`check` cannot say what changed.** `cli/check.go` compares bytes and reports
`<path>: document is out of date`. In CI the only recourse is to run `generate`
and read `git diff`. See D9.

## Goals / Non-goals

**Goals**

1. `-v` and `-vv` narrate a run on stderr; default output is byte-identical to
   today (no noise for anyone who did not ask).
2. Verbose output is deterministic and assertable: sorted, fixed stamp format,
   stable prefixes, so tests pin the exact lines.
3. The two slow stages log their **start** as well as their duration, so a hang
   is attributable.
4. Merge collisions and unused mappings are reported as located findings, not
   swallowed.
5. Every record carries named attributes, so a JSON handler later is a swap and
   not a refactor.
6. Off by default at zero meaningful cost: no work in the derivation and
   traversal loops when verbosity is zero.
7. No behavior change beyond reporting: same exit codes (`0/1/2/3`), same
   artifact bytes, same error text.

**Non-goals (v1)**

- Ship a machine-readable rendering (`--log-format=json|logfmt`). The record's
  field names are frozen now (D3) so it is cheap later.
- `--log-file=path` (full verbosity stream to a file, readable terminal).
  Deferred; it is a second writer over the same handler.
- GitHub Actions annotations (`::error file=…`). Deferred; D11's finding shape
  is what makes it a renderer rather than a rewrite.
- `ASYNCGO_VERBOSE` / config-file verbosity. One flag surface, chosen per
  invocation.
- Per-stage *profiling* (CPU/mem). Durations only.
- A `--keep-harness` flag: D7 keeps the directory only on failure and names it.
- Logging inside the harness process itself (it runs under `go run`; its own
  stderr is the channel it already has, and D7 passes it through).

## Design decisions

### D1 — Verbosity is a repeatable count on the root command

`-v` / `--verbose` is a `PersistentFlags().CountVarP` counter on the root
command, so `generate` and `check` inherit it and `version` ignores it. The
count maps to a level:

| count | level | content |
| --- | --- | --- |
| 0 | `Quiet` | today's behaviour, unchanged |
| 1 | `Verbose` | the stage story: one line per stage, start + summary |
| ≥2 | `Debug` | plus internals, harness passthrough, and the timing recap |

Counts above 2 clamp to `Debug` rather than erroring: `-vvv` from a shell alias
should not fail a build.

**Rationale**: a single boolean cannot express the separation between "the
story" (what a user wants when confused) and "the internals" (what a maintainer
wants when debugging asyncgo itself), and both were requested. A named
`--log-level` was rejected as a heavier surface for a three-command CLI that
forces a `warn`/`error` taxonomy nobody needs; an env var alone was rejected
because it is invisible in `--help` and cannot be set per-invocation.

### D2 — Verbose output goes to stderr

Everything verbose — including the `-vv` harness passthrough, findings and the
recap — is written to the command's error stream via Cobra's
`cmd.ErrOrStderr()`. Stdout keeps carrying only the artifact messages:
`wrote <path> (N catalog(s))` and `<path> is up to date`.

**Rationale**: it keeps the two streams semantically separate (result vs.
narration) and makes the logging stream redirectable on its own. It also means
`check`'s existing contract, which CI scripts parse, is untouched.

### D3 — Stage-tagged prose, with typed attributes carried but not printed

A new `internal/logging` package wraps `log/slog`:

```go
package logging

// Level is the CLI verbosity, chosen by the -v count.
type Level int

const (
    Quiet   Level = iota // warnings and errors only; the zero value
    Verbose              // -v: the stage story
    Debug                // -vv: internals, harness passthrough, recap
)

// New returns a logger writing to w at level. A nil *slog.Logger is treated as
// Quiet everywhere it is consumed, so callers need no branch.
func New(w io.Writer, level Level) *slog.Logger

// Tag returns a logger whose records are tagged with the stage name, without the
// timing Step adds.
func Tag(log *slog.Logger, name string) *slog.Logger

// Step marks a stage: it returns a tagged logger and a close func that logs the
// summary with the measured duration and records it for the recap. The attrs are
// slog key/value pairs for the machine rendering.
func Step(log *slog.Logger, name string) (*slog.Logger, func(summary string, attrs ...any))
```

Every record has exactly three parts:

| part | example | printed? |
| --- | --- | --- |
| level | `info`, `warn`, `error` | only as color, never as text |
| `stage` attr | `load`, `harness`, `merge` | yes, as the tag after the prefix |
| `msg` | `42 packages, 1 main package in 1.2s` | yes, verbatim |
| named attrs | `packages=42`, `duration_ms=1204` | **no** — machine-only |

The text handler renders one line per record:

```text
asyncgo: +1.204s load: 42 packages, 1 main package in 1.2s
```

and **deliberately ignores named attributes**. Attributes exist so a future
JSON handler has a stable vocabulary (`stage`, `msg`, `duration_ms`,
`packages`, `mains`, `catalogs`, `errors`, `format`, `bytes`, `path`,
`section`, `name`, `kept`, `dropped`, `where`) without changing a single call
site; the stage summaries already attach them, so the vocabulary is real code,
not a plan. Interpolating them into the text would
reproduce the `key=value` debris this format was chosen to avoid, so the rule
is: **the prose is the human truth, the attributes are the machine truth, and
each rendering uses exactly one of them.** A call site that has something to
say says it twice, explicitly, which also makes the duplication visible in
review.

**Rationale**: `slog` is stdlib (Go 1.26), gives level filtering and
`With`/`WithAttrs` for free, and keeps a future JSON mode to "swap the
handler". A bespoke logger would re-implement that filtering; a third-party
logger would add a dependency the project does not need. A custom handler is
required either way: `slog.TextHandler` insists on
`time=… level=INFO msg=…`, which is wrong for human-facing CLI output. The
stage tag was chosen over trailing attrs (unreadable debris), an aligned gutter
(ceremonial prefix) and an indented tree (children stop being prefixed
records, which breaks tailing and stream interleaving).

### D4 — A logger travels in `discovery.Options`, not the context

`Build` and `Materialize` gain an options value:

```go
// Options carries the run-time configuration and diagnostics a build needs.
type Options struct {
    Config string         // schema-mapping config path; empty auto-discovers
    Log    *slog.Logger   // nil means quiet
}

func Build(ctx context.Context, dir string, opts Options) (*spec.AsyncAPI, int, error)
func Materialize(ctx context.Context, dir string, plan Plan, opts Options) ([]*spec.AsyncAPI, error)
```

`Build`'s current third-parameter `configPath` folds into `opts.Config`, so the
signature does not grow a fourth positional argument. `Materialize` reuses the
same type: only `Log` is meaningful to it, and a future `KeepHarness` knob
belongs there too.

**Rationale**: an explicit field is discoverable at the call site, has no global
state (the discovery tests run packages in parallel), and is trivially testable
by handing in a logger over a `bytes.Buffer`. A `context` value was rejected
because it hides the dependency and makes "which functions can log?" a
question nobody can answer without grepping. A package-level variable was
rejected for racy global state.

Consequence: `internal/cli/root.go:100` and
`test/integration/asyncgo_generate_test.go:91` are the only two non-test
call sites that change, plus the `Materialize` callers in
`discovery/*_test.go`.

### D5 — Stage lines are emitted at start, summarised at end, with durations

Each slow stage opens a line and closes it with the result:

```text
asyncgo: +0.012s load: loading packages in /repo...
asyncgo: +1.216s load: 42 packages, 1 main package in 1.2s
```

The opening line is what makes a hang attributable: the last `...` line names
the stage that is running. `logging.Stage` (D3) owns the pairing so a stage
cannot close with the wrong elapsed time.

**Rationale**: a summary-only log leaves a user staring at a terminal with no
idea which of two multi-second stages is running; a start-only log never tells
them how expensive it was. Durations come from `time.Since` in the caller, so
no clock abstraction is needed — tests assert the prose after the stamp
(D12 defines the stamp's exact shape and how tests tolerate it).

### D6 — `Merge` reports collisions instead of swallowing them

`Merge` takes labelled documents and returns the collisions it resolved:

```go
// Document pairs a catalog's identity with the document it materialized.
type Document struct {
    Catalog Catalog
    Doc     *spec.AsyncAPI
}

// Collision is a map key two or more catalogs defined; the first one won.
type Collision struct {
    Section string // "channels", "components.schemas", ...
    Name    string
    Kept    Catalog
    Dropped []Catalog
}

func Merge(docs ...Document) (*spec.AsyncAPI, []Collision)
```

Collisions are sorted by section then name (map iteration order is random) and
`Build` reports each as a finding (D11):

```text
asyncgo: merge: 3 channels, 12 schemas in 40ms
  channels.orderCreated: duplicate, kept from catalog.Admin (dropped from catalog.Orders)
```

**Rationale**: the merged document is genuinely lossy for one of the catalogs,
and a user cannot discover that from the artifact. Reporting belongs at `-v`
rather than only `-vv` because it is a correctness fact, not an internal
detail. Labelling documents is unavoidable anyway: without a source, a
collision message cannot say *what* was dropped in favour of *what*.

### D7 — The harness's stderr is passed through, and a failed harness is kept

- The subprocess's stderr is teed: `cmd.Stderr = io.MultiWriter(&buf, logWriter)`
  when `Debug` is enabled, so compiler/vet/`init()` output appears live instead
  of being swallowed. At `Verbose` it stays buffered and appears only in the
  error, as today.
- `Debug` logs the harness directory path before running it.
- On a **non-zero exit from the harness**, the directory is *kept* and its path
  is appended to the returned error:

  ```text
  asyncgo: materializing catalogs: exit status 1
  <harness stderr>
  harness program kept for inspection: /repo/_asyncgo-harness-1234567/main.go
  ```

**Rationale**: the generated `main.go` is the only artifact that explains a
catalog that will not compile, and today it is deleted on the failure path,
which is exactly when it is needed. Keeping it is safe: the `_`-prefix already
makes the `go` tool ignore the directory when `load`'s `./...` scan runs, so a
leftover cannot break a subsequent run. The trade-off — leftovers accumulate if
a user retries a broken build repeatedly — is accepted and named in the error
message, which tells the user what to delete.

### D8 — `discovery`'s direct `os.Stderr` writes become log records

`removeHarnessDir` (`materialize.go:123`) and `reportMappingWarnings`
(`:146`) stop taking an `io.Writer` / using `os.Stderr` and instead log through
`Options.Log`. The removal failure is a `warn` record; the mapping warning
becomes a finding (D11) with `asyncgo.yaml` as its location, since that is the
file the user has to edit:

```text
asyncgo: config: 3 mappings (auto-discovered)
  asyncgo.yaml: pkg.Type: mapping never used, no derived type matched that name
```

Both are emitted at `Verbose` (not gated behind `Debug`) because they were
already printed on success today, and demoting them would be a regression.

**Rationale**: this is the fix for the untestable `os.Stderr` bypass described
in Background. It also gives the two messages a home that is already gated by
verbosity, so the "always printed" semantics get an explicit decision rather
than an accident of `os.Stderr`.

### D9 — `check` explains what differs, with a byte-offset fallback

When the comparison fails, `check` decodes the on-disk artifact with the same
format's decoder and diffs it against the freshly generated document:

1. If the artifact does not decode (hand-edited invalid YAML/JSON), fall back
   to today's message plus the first differing byte offset and a window of
   context; the decode failure itself is logged at `Debug`.
2. Otherwise emit one finding per differing section key, sorted, capped at 10
   per section with an `and N more` finding at the end (D11).
3. The error itself is unchanged (`<path>: document is out of date; run
   asyncgo generate`), so `errors.Is(err, ErrOutOfDate)` and exit code 3 keep
   working; the findings go to stderr at `Verbose`.

```text
asyncgo: check: /repo/asyncapi.yaml is out of date
  /repo/asyncapi.yaml: channels.orderCreated: missing
  /repo/asyncapi.yaml: channels.orderUpdated: stale
  /repo/asyncapi.yaml: components.schemas.OrderV2: differs
```

**Rationale**: this is the single highest-value item for CI, where the current
message forces a full regenerate-and-read-diff loop. Putting the findings
behind `-v` rather than in the error keeps the default output and the exit-code
contract exactly as they are, and keeps the message short for the common
"someone forgot to regenerate" case. A generic structural diff engine is out of
scope: the AsyncAPI document is a known shape, so per-section key comparison is
enough and much cheaper than fidelity to arbitrary YAML.

### D10 — Nothing is formatted unless the level is enabled

Every call site that builds a message inside a loop (`walkStructTypes`
callbacks in `combinators.go` / `descriptions.go`, the per-catalog loop in
`materialize.go`) is guarded by `if log.Enabled(ctx, slog.LevelDebug)` — or,
where a single-line `Debug` is inside a hot traversal, hoisted so the
per-iteration branch is one map lookup. `Options.Log` may be nil; the helper
that consumes it normalises nil to a discard logger.

**Rationale**: `packages.Load` dominates the run today and adding per-struct
string formatting to a traversal over thousands of types would be a real
regression for the default, non-verbose case. `slog`'s `Enabled` makes the
guard cheap and explicit.

### D11 — Problems render as located findings, compiler-style

One shape for all three kinds of problem — check drift, unused mappings, merge
collisions:

```go
// Finding is one located problem: where it is, what it is, and why. It is
// rendered compiler-style — "path: where: what (why)" — so a terminal, an
// editor, and a CI annotation renderer can each consume it.
type Finding struct {
    Path  string // file the user must edit: asyncapi.yaml, asyncgo.yaml; empty for the generated doc
    Where string // path inside the document: channels.orderCreated, components.schemas.OrderV2
    What  string // missing, stale, differs, duplicate, mapping never used
    Why   string // optional parenthetical: declared by catalog.Admin, dropped from catalog.Orders
}

// ReportFindings writes findings at Verbose as indented "path: where: what (why)"
// lines, sorted by Path then Where.
func ReportFindings(log *slog.Logger, findings []Finding)
```

Rendering rules: two-space indent under the stage line that produced them, no
`asyncgo:` prefix per finding (they belong to the record above them), sorted by
`Path` then `Where`, and never wrapped.

**Rationale**: three prose sentences for three located problems was the status
quo, and none of them could be acted on mechanically. A single struct makes the
finding a data type rather than a string, which is what makes a future
`--log-format=github` (`::error file=…,title=…::`) a renderer of the same slice
instead of a second analysis pass. Keeping findings indented rather than
prefixed preserves the D3 rule that each line is one prefixed record — findings
are the deliberate exception, because they are subordinate detail of a record,
and they always appear as a contiguous block.

### D12 — Three decorations: color on a TTY, relative stamps, a timing recap

All three are rendering concerns owned by `internal/logging`, so no call site
knows about them.

**Color, only on a terminal.** Enabled when the writer is a real character
device (`f, ok := w.(*os.File); ok && f.Stat().Mode()&os.ModeCharDevice != 0`),
`NO_COLOR` is unset, and `TERM != "dumb"`. No new dependency (`golang.org/x/term`
is not needed for this check). The `asyncgo:` prefix is bold, the stage tag and
stamp dim, `warn` records yellow, `error` records and their findings red.
Because the test harness writes to a `bytes.Buffer`, tests are colorless by
construction — no `NO_COLOR` bookkeeping in the suite.

**Relative stamps.** Every prefixed record carries elapsed time since the
command started, as `+<seconds>.<millis>s` with exactly three decimals
(`+0.000s`, `+1.204s`, `+12.345s`, `+123.456s`), placed between the prefix and
the stage tag:

```text
asyncgo: +3.412s harness: go run in 3.4s
```

The stamp answers "when", the prose answers "how long" — deliberately
redundant, and the recap answers "where did the time go".

**End-of-run recap, at `Debug` only.** One aligned table listing the stages that
ran, their durations and their summary counts, printed after the last stage and
before the artifact message on stdout:

```text
asyncgo: recap
  config     0.4ms  3 mappings
  load      1.204s  42 packages, 2 catalogs
  harness   3.412s  2 catalogs valid
  merge      40ms   3 channels, 12 schemas, 1 collision
  encode      1ms   12.3 KiB
  total     4.669s
```

Stage names are left-padded to the longest name, durations right-aligned to the
longest duration string, so the columns hold for a given run and never depend
on terminal width. The recap is skipped when a run fails before reaching a
stage, and lists only the stages that actually ran.

**Rationale**: color is the cheapest way to make a 15-line run scannable and
costs nothing off a terminal; a stamp makes the 3.4s harness visible in
scrollback instead of requiring arithmetic on the `in 3.4s` prose; the recap is
what a user actually wants after a slow run, and it is additive, so D5's
hang attribution is unaffected. The recap sits at `Debug` because at `Verbose`
it would restate the five lines directly above it.

## Detailed design

### 1. `internal/logging/logging.go` (new)

```go
package logging

// New returns a logger that writes CLI lines to w at level. Quiet returns a
// logger whose Enabled is always false; callers never branch on nil.
func New(w io.Writer, level Level) *slog.Logger

// Verbosity maps a -v count to a Level, clamping anything above Debug.
func Verbosity(count int) Level

// Stage returns a logger with the stage attribute set plus a func that logs the
// stage's summary line with its elapsed time. The closing func is idempotent.
func Stage(log *slog.Logger, name string) (*slog.Logger, func())

// handler implements slog.Handler: filters by level, prefixes "asyncgo: ",
// renders the stamp and stage tag, and ignores attrs (D3).
type handler struct {
    mu     *sync.Mutex
    w      io.Writer
    level  Level
    color  bool
    start  time.Time
    stage  string
    attrs  []slog.Attr // carried for the machine rendering only
}
```

- One `Write` per record, under a mutex, so concurrent loggers over the same
  `bytes.Buffer` in tests never interleave mid-line.
- `recap.go` in the same package holds the timing table (`Stage` appends to a
  shared `[]stageTiming`), and `findings.go` holds `Finding` +
  `ReportFindings` (D11).

### 2. `internal/cli/root.go`

```go
func newRootCmd(bi BuildInfo) *cobra.Command {
    var verbosity int
    root := &cobra.Command{ /* unchanged */ }
    root.PersistentFlags().CountVarP(&verbosity, "verbose", "v",
        "log what the command is doing to stderr (repeat for more detail)")
    root.PersistentPreRun = func(cmd *cobra.Command, _ []string) {
        level := logging.Verbosity(verbosity)
        logger := logging.New(cmd.ErrOrStderr(), level)
        cmd.SetContext(logging.WithLogger(cmd.Context(), logger))
    }
    ...
}
```

`logging.WithLogger`/`FromContext` exist **only** so the `RunE` bodies can
build `discovery.Options{Log: logging.FromContext(cmd.Context())}` without
threading a second parameter through `buildDoc`; `Build` itself never reads the
context for a logger (D4). `buildDoc`'s signature becomes:

```go
func buildDoc(cmd *cobra.Command, dir string, opts discovery.Options, verb string) (*spec.AsyncAPI, int, error)
```

which also removes the awkward `configPath` string parameter. The recap prints
in `PersistentPostRun` (so it appears for `check` too, before the command's own
stdout line is flushed).

### 3. `internal/discovery/build.go`

`Build` logs, in order:

| level | stage tag | line |
| --- | --- | --- |
| Verbose | `config` | `loading <path> (explicit)` / `loading <path> (auto-discovered)` / `none found` |
| Verbose | `config` | `3 mappings` + a finding per never-used mapping (D8) |
| Verbose | `load` | `loading packages in <dir>...` |
| Verbose | `load` | `42 packages, 1 main package in 1.2s` |
| Verbose | `load` | `5 packages unreachable from main skipped` |
| Verbose | `catalogs` | `2 found` |
| Debug | `catalogs` | one line per catalog with its declaring file |
| Debug | `refs` | `4 combinator refs from 2 packages, 1 alias edge` |
| Verbose | `harness` | `materializing 2 catalogs...` |
| Verbose | `harness` | `2 catalogs valid, 0 errors in 3.4s` |
| Verbose | `merge` | `3 channels, 1 server, 12 schemas in 40ms` |
| Verbose | `merge` | a finding per collision (D6) |
| Debug | `descriptions` | `applied 37 field descriptions across 9 schemas` |
| Verbose | `encode` | `yaml, 12.3 KiB in 1ms` |

The zero-catalog error path stays an error, but its preceding lines already
told the user what was loaded and why nothing was reachable.

### 4. `internal/discovery/materialize.go`

- `Plan` is unchanged. `Materialize` gains `opts Options`.
- Debug: `harness: directory <path>` before writing, then
  `harness: wrote main.go (3.1 KiB, 6 imports)`.
- Debug: `harness: go run ./_asyncgo-harness-… (compiles the module)` before
  `cmd.Run()`.
- Debug: stderr tee (D7); Verbose: buffered, error-only as today.
- D8 rewires the two diagnostics; `removeHarnessDir` gains a `keep bool` so the
  failure path can retain the directory.
- `reportMappingWarnings(w io.Writer, …)` becomes
  `mappingFindings(warnings []string) []logging.Finding`; its tests assert the
  slice, which is stronger than asserting formatted text.

### 5. `internal/discovery/merge.go`

`Merge` takes `[]Document` and returns `[]Collision` as in D6. The internal
`mergeMap[V]` gains `section string`, `kept Catalog`, and a `*[]Collision`
sink; each `components.*` call passes its own section name
(`components.schemas`, …). `Info`/`DefaultContentType` first-wins behaviour is
unchanged, but a *conflicting* later `Info` (non-empty Title that lost) is
reported as a collision too, since the comment in `merge.go:9-18` already warns
that this silently drops the other catalogs' Info.

`merge_test.go` and `build_test.go` update to the new signature; the existing
`should_take_info_from_the_first_document_with_a_title` case gains a collision
assertion.

### 6. `internal/cli/check.go`

Extract the comparison into a helper so the fallback and the section diff are
testable without a command:

```go
// diffArtifact reports how the on-disk artifact differs from the generated
// document as located findings. It returns ok=false with no findings when the
// artifact does not decode, so the caller falls back to a byte window.
func diffArtifact(artifact, want []byte, format string) (ok bool, findings []logging.Finding)
```

`RunE` calls it, reports the findings, and returns the unchanged `ErrOutOfDate`
wrap.

## Example (before / after)

`before` — a real confusing case: a second catalog declares a channel the first
one already declared, and the harness takes four seconds.

```console
$ asyncgo generate
wrote /repo/asyncapi.yaml (2 catalog(s))
```

`after`, `-v`:

```console
$ asyncgo generate -v
asyncgo: +0.000s config: loading /repo/asyncgo.yaml (auto-discovered)
asyncgo: +0.001s config: 3 mappings
asyncgo: +0.001s load: loading packages in /repo...
asyncgo: +1.216s load: 42 packages, 1 main package in 1.2s
asyncgo: +1.217s load: 5 packages unreachable from main skipped
asyncgo: +1.218s catalogs: 2 found
asyncgo: +1.219s harness: materializing 2 catalogs...
asyncgo: +4.631s harness: 2 catalogs valid, 0 errors in 3.4s
asyncgo: +4.632s merge: 3 channels, 1 server, 12 schemas in 40ms
  channels.orderCreated: duplicate, kept from catalog.Admin (dropped from catalog.Orders)
asyncgo: +4.673s encode: yaml, 12.3 KiB in 1ms
wrote /repo/asyncapi.yaml (2 catalog(s))
```

`after`, `-vv` adds the internals and closes with the recap:

```console
asyncgo: +1.218s catalogs: catalog.Admin.AdminSpec declared in catalog/admin.go
asyncgo: +1.218s catalogs: catalog.Orders.OrdersSpec declared in catalog/orders.go
asyncgo: +1.219s refs: 4 combinator refs from 2 packages, 1 alias edge
asyncgo: +1.219s harness: directory /repo/_asyncgo-harness-2739442
asyncgo: +1.220s harness: go run ./_asyncgo-harness-2739442 (compiles the module)
asyncgo: +4.673s descriptions: applied 37 field descriptions across 9 schemas
asyncgo: recap
  config     0.4ms  3 mappings
  load      1.204s  42 packages, 2 catalogs
  harness   3.412s  2 catalogs, 0 with errors
  merge      40ms   3 channels, 12 schemas, 1 collision
  encode      1ms   12.3 KiB
  total     4.669s
wrote /repo/asyncapi.yaml (2 catalog(s))
```

`check` with a drifted artifact:

```console
$ asyncgo check -v
asyncgo: +0.000s load: loading packages in /repo...
asyncgo: +1.204s load: 42 packages, 1 main package in 1.2s
...
asyncgo: +4.669s check: /repo/asyncapi.yaml is out of date
  /repo/asyncapi.yaml: channels.orderCreated: missing
  /repo/asyncapi.yaml: channels.orderUpdated: stale
asyncgo: /repo/asyncapi.yaml: document is out of date; run `asyncgo generate`
$ echo $?
3
```

## Edge cases

- **Module with no `main` package** — `reachableFromMain` treats all packages as
  reachable; the log says `0 main packages, considering all 42`.
- **Zero catalogs** — the error is unchanged; the preceding `-v` lines show
  what loaded and how many packages were skipped, which is the actual diagnosis.
- **`-v` on a huge module** — `Debug` per-package lines are unbounded by
  design; `Verbose` is one line per stage plus one per catalog, which stays
  readable at any size. Documented in `--help` as "repeat for more detail".
- **Harness failure** — directory kept and named (D7); a later successful run
  does not remove an older leftover, which the message tells the user to delete.
- **Unparsable artifact in `check`** — byte-offset fallback (D9 step 1), plus a
  `Debug` line naming the decode error, so "someone committed a hand-edit" is
  distinguishable from "someone forgot to regenerate".
- **Concurrent tests** — `Quiet` loggers are cheap and stateless; tests that
  assert log lines build their own logger over a `bytes.Buffer` rather than
  touching a global.
- **Color leaking into a file** — impossible by construction: the check is on
  the `*os.File` behind the writer, and a buffer or pipe fails it. `NO_COLOR`
  and `TERM=dumb` are honored on top.
- **Stamps in golden tests** — assertions strip the `^asyncgo: \+\d+\.\d{3}s `
  prefix with a helper, then compare the remainder; the stamp's *format* is
  pinned once by a dedicated test.
- **Recap with a failing stage** — the failing stage's row is omitted (it has no
  duration) and `total` still prints, so a run that died in the harness shows
  the earlier stages' costs.
- **`version`** — inherits `-v` and ignores it; no logger is constructed since
  nothing logs.
- **Determinism** — catalogs, refs, aliases, collisions, findings and diff keys
  are all sorted before logging; no map iteration reaches the writer. Stamps are
  the only non-deterministic part of a line, and every assertion strips them.
- **Stream separation** — a `Verbose`/`Debug` line never goes to
  `cmd.OutOrStdout()`; only `wrote …` / `is up to date` do.

## Rollout plan

1. **Stage 0 — plumb a writer, change no output** (`internal/discovery`):
   introduce `internal/logging` (Quiet only), convert `removeHarnessDir` and
   `reportMappingWarnings` to log through `Options.Log`, thread `Options`
   through `Build`/`Materialize` and the two call sites. Output is
   byte-identical; the `os.Stderr` bypass is gone and the two existing messages
   become assertable.
2. **Stage 1 — the flag, the record shape, the stage story**
   (`internal/cli`, `internal/logging`): `-v` counter, `Verbosity`, the handler
   with stage tag and stamp, color-on-TTY, `buildDoc` signature, `Verbose`
   lines for the stages that are already instrumented. Golden tests for
   `generate -v` and `check -v` on `test/data/simple`.
3. **Stage 2 — findings, internals, harness** (`internal/discovery`):
   `Finding`/`ReportFindings`, `Merge`'s `Document`/`Collision` signature change,
   the `Debug` per-package/per-catalog/mapping/ref/alias/description lines,
   stderr tee, keep-failed-harness, and the recap (`Stage` timing accumulation).
4. **Stage 3 — `check` diff** (`internal/cli`): `diffArtifact`, section/key
   comparison, byte-offset fallback.
5. **Stage 4 — deferred**: `--log-format=json` (attribute vocabulary is already
   frozen), `--log-format=github` (findings already shaped), `--log-file`,
   `ASYNCGO_VERBOSE`.

## Testing plan

- **Stage 0** (`internal/discovery`): `materialize_test.go` asserts the mapping
  findings and the harness-removal record through an injected logger over a
  `bytes.Buffer`.
  - `should_report_unused_mapping_as_a_finding`
  - `should_report_harness_removal_failure`
- **Stage 1** (`internal/cli`, `internal/logging`): golden tests over
  `test/data/simple`, with the stamp-stripping helper.
  - `should_print_no_verbose_lines_without_the_flag`
  - `should_tag_each_line_with_its_stage`
  - `should_format_the_stamp_with_three_decimals`
  - `should_colour_output_only_when_the_writer_is_a_terminal`
  - `should_honour_no_color`
  - `should_keep_stdout_limited_to_the_artifact_message`
  - `should_clamp_verbosity_above_debug`
- **Stage 2** (`internal/discovery`, `internal/logging`):
  - `should_report_each_collision_with_both_catalogs`
  - `should_sort_collisions_by_section_then_name`
  - `should_render_findings_as_path_where_what`
  - `should_print_the_recap_only_at_debug`
  - `should_right_align_recap_durations`
  - `should_pass_harness_stderr_through_at_debug`
  - `should_keep_the_harness_directory_when_the_harness_fails`
  - `should_not_format_debug_messages_at_verbose` (a counting writer proves the
    line was filtered before formatting)
- **Stage 3** (`internal/cli`):
  - `should_report_stale_and_missing_channel_names`
  - `should_fall_back_to_a_byte_offset_when_the_artifact_does_not_decode`
  - `should_keep_exit_code_three_when_verbose`
- **Cross-cutting**: a test asserting the artifact bytes and the exit code are
  identical with and without `-vv`, and that no verbose line reaches stdout.
- **Race**: `go test ./... -race`, per `AGENT.md`.

## Open / deferred

- **Stderr tee interleaving** — line-atomic passthrough of a subprocess's stderr
  interleaved with our own lines cannot be guaranteed; accepted (the harness
  writes one block at the end in practice).
- **`Info` field-wise merge** — `merge.go:9-18` notes that filling gaps rather
  than taking the first `Info` whole is a possible future change. Out of scope;
  D6 only makes the current drop visible.
- **Finding severities** — `Finding` has no `Severity` field; check drift and
  collisions are warnings, an unused mapping arguably is too. If
  `--log-format=github` arrives it will need one (`::warning` vs `::error`), and
  adding the field then is a one-line change with no call-site churn.
- **Provenance in check findings** — a drift finding says
  `channels.orderCreated: missing` but cannot say *which catalog* declares it,
  because the merged document it is compared against carries no per-key source.
  The information exists inside `Merge` (it is what makes a collision message
  nameable) and would need exposing from `Build`; deferred as a nicety until
  somebody asks for it in CI output.
- **Recap as a data structure** — the recap is currently printed by the handler;
  a future `--log-format=json` run would want the recap as a record. Deferred
  with the JSON handler.
- **Harness-level logging** — the harness could log registration counts from
  inside the subprocess. Deferred: everything it would say is already derivable
  from the plan in `Materialize`.
- **Verbose output stability as an API** — CI may come to depend on these lines.
  Treated as unstable until this design is `Accepted`; afterwards, changes are
  documented in the ADR rather than silently reworded. The attribute vocabulary
  is the exception: it is frozen now (D3) and is what the future JSON contract
  will be built from.
