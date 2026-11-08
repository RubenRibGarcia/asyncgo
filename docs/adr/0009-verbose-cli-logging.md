# 0009. Narrate CLI runs with a repeatable verbose flag

- Status: accepted
- Deciders: asyncgo maintainers
- Created: 2026-11-08
- Status updated: 2026-11-08

## Context and Problem Statement

`asyncgo generate` and `asyncgo check` report one line on success and an error on
failure. Between parsing the flags and writing the artifact, two stages dominate
the wall clock — `packages.Load` type-checks the module, and the harness stage
shells out to `go run`, which *compiles* it — and neither says anything while it
runs. A cold build, a hung toolchain and a slow disk are indistinguishable from
each other, and the generated harness program is deleted before a user can read
it, including on the failure path that most needs it.

Three problems are reported in three unrelated prose shapes (a check failure, an
unused-mapping warning, and — today — nothing at all for a merge collision), and
none of them says *where* the problem is. Meanwhile `Merge` silently drops the
losing document's keys on a collision, so a merged document can be wrong for one
catalog with no signal anywhere.

The output surface also has to stay boring: `check`'s exit codes (3 = stale,
1 = broken, 2 = usage) and its one-line success message are a CI contract, and
the artifact bytes must not depend on how loudly the command talks.

## Decision Drivers

- A confused user must be able to see *which* stage is running, not just that
  something is slow.
- Default output must be unchanged: verbosity is opt-in, per invocation.
- Verbose output must be deterministic and assertable in tests (this project
  pins behaviour with tests, not with prose).
- Findings (drift, unused mappings, collisions) are located problems and should
  be machine-actionable, not sentences.
- No new dependency, and no measurable cost when verbosity is off: the
  derivation traversals already walk thousands of types.
- A machine-readable rendering will be wanted later without a second pass over
  every call site.

## Considered Options

- **Stage-tagged prose with machine-only attributes**, rendering
  `asyncgo: +1.204s load: 42 packages in 1.2s`, plus compiler-style findings and
  a `-v`/`-vv` counter.
- Prose with trailing `key=value` attributes (a hand-shaped logfmt).
- An aligned stage gutter (`asyncgo [ load ] 42 packages`).
- An indented stage tree (stage headers with indented children).
- A single end-of-run summary block, with no per-stage lines.
- A named `--log-level=info|debug` flag, or verbosity via an env var only.
- Ship a JSON or logfmt rendering in v1.

## Decision Outcome

Chosen option: "Stage-tagged prose with machine-only attributes", because it is
the only shape that keeps one prefixed record per line (so lines stay greppable,
pipeable and orderable) while making the stage scannable and giving every record
a stable field vocabulary.

Concretely:

- **`-v` is a repeatable count on the root command** (`CountVarP`), inherited by
  both subcommands: 0 = today's output, 1 = the stage story, ≥2 = internals plus
  an end-of-run timing recap. Counts above 2 clamp rather than error.
- **All narration goes to stderr**, so stdout keeps carrying only
  `wrote …` / `is up to date`. Exit codes and error wording are untouched.
- **Two payloads per record, one per rendering.** Every record carries a
  human sentence (`msg`) and typed attributes (`packages`, `duration_ms`,
  `section`, `kept`, `dropped`, …). The text handler prints the sentence and
  **ignores the attributes**, so no line ever grows `key=value` debris; a future
  JSON handler consumes the attributes and needs no call-site changes. The
  vocabulary is frozen now and is the contract that future format will use.
- **`log/slog` with a custom handler**, in a new `internal/logging` package.
  Stdlib, level filtering for free, and a format swap is a handler swap. The
  custom handler exists because `slog.TextHandler` insists on
  `time=… level=INFO msg=…`, which is wrong for CLI output.
- **The logger travels in `discovery.Options{Config, Log}`**, not in the
  context, so "which functions can log?" is answerable at the call site and the
  dependency is injectable in tests. Stage 0 threads it through `Build` and
  `Materialize` and converts the two direct `os.Stderr` writes in
  `internal/discovery` into records — those bypass `cmd.ErrOrStderr()` today and
  are invisible to the test suite.
- **Slow stages log twice**: an opening `…` line and a summary with its
  duration, so a hang is attributable to the stage that is running.
- **Problems are one type.** `logging.Finding{Path, Where, What, Why}` renders
  compiler-style (`asyncapi.yaml: channels.orderCreated: missing (declared by
  catalog.Admin)`) and carries check drift, unused mappings and merge collisions
  alike. `Merge` therefore gains labelled documents and returns the collisions
  it resolved instead of swallowing them.
- **A failed harness is kept on disk and named in the error.** The generated
  program explains a catalog that will not compile; deleting it on failure is
  backwards. The `_` directory prefix already makes the `go` tool ignore it, so
  a leftover cannot poison a later run.
- **Three decorations, owned by the handler**: color only on a real terminal
  (honoring `NO_COLOR` and `TERM=dumb`, no `x/term` dependency), relative
  `+s.mmm` stamps, and a timing recap at `-vv` only.

Rejected:

- Trailing `key=value` was rejected as unreadable debris, and an aligned gutter
  as a ceremonial prefix on a three-command CLI.
- The indented tree was rejected because its children stop being prefixed
  records: tailing and stream interleaving both get worse.
- The end-of-run block alone was rejected because it removes the only signal a
  hang gives you — which stage is running now.
- `--log-level` was rejected as a heavier flag surface that imposes a
  `warn`/`error` taxonomy nobody asked for; an env var alone is invisible in
  `--help` and cannot be set per invocation.
- Shipping JSON/logfmt in v1 was deferred, not rejected: with the field
  vocabulary frozen, it is a handler.

### Consequences

- Good, because the two expensive stages become attributable, and their cost is
  visible as a duration instead of a silent pause.
- Good, because merge collisions and unused mappings are now reported with a
  location, and a future `--log-format=github` annotations renderer consumes the
  same `Finding` slice rather than re-deriving anything.
- Good, because default output is byte-identical and `check`'s exit-code
  contract is untouched.
- Good, because the record vocabulary exists before anything consumes it, so the
  machine format is additive.
- Bad, because a call site that has both a sentence and attributes says the same
  thing twice; the duplication is deliberate and visible in review, but it is
  still duplication.
- Bad, because `Merge`'s signature changes (`[]Document` → collisions), which
  pushes label bookkeeping into `Build` and churn into the merge tests.
- Bad, because a failed harness now leaves a directory behind until the user
  deletes it — accepted because the message names it, and because the
  alternative is losing the artifact that explains the failure.
- Bad, because `-vv` per-package logging is unbounded on a large module. Bounded
  verbosity would need a sampling decision nobody has asked for.
- Neutral, because the human line wording is not part of the contract until the
  design is accepted; the attribute vocabulary is.

## More Information

- Design doc:
  [`docs/designdoc/cli-verbose-logging.md`](../designdoc/cli-verbose-logging.md)
- Related: [ADR 0003](0003-adopt-cobra-for-cli.md) — the command tree this flag
  hangs off, and where `--help`/`--version` already come from
- Related: [ADR 0008](0008-external-type-mappings.md) — the unused-mapping
  warning this ADR promotes to a located finding
