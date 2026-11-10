# 0011. Emit one document per catalog when the destination is a directory

- Status: accepted
- Deciders: asyncgo maintainers
- Created: 2026-11-10
- Status updated: 2026-11-10

## Context and Problem Statement

`generate` and `check` each resolve to exactly one file: `resolveOutput` maps an
empty `-o` to `<dir>/asyncapi.<format>`, a trailing-separator `-o` to
`<dir>/asyncapi.<format>`, and anything else to that exact path. Underneath,
`discovery.Build` always merges every catalog in the module into a single
document, so the single-file rule is not a CLI convenience — it is the only
output the data model can produce.

That constraint is wrong for a module that carries more than one catalog:

- **Merging is lossy.** `Merge` is first-wins, including `Info`, which is taken
  whole from the first catalog whose title is non-empty
  (`internal/discovery/merge.go`). A second catalog's title, version and
  description are dropped, so the merged document is *incorrect* for every
  catalog but the first. Collisions are reported, but a report does not make the
  document right.
- **`-o` cannot express intent.** `-o docs/orders.yaml` on a module with three
  catalogs silently writes the lossy merge of all three. The flag names a
  destination; nothing in the interface can name *which catalogs* belong in it.
- **`check` cannot attribute.** It reproduces `generate`'s resolution and
  byte-compares one merged document against one file. Given several catalogs
  there is no catalog↔artifact correspondence to verify, so a per-catalog
  regression is invisible until someone regenerates and reads the diff.

[ADR 0010](0010-catalog-scope-is-the-module.md) removed the import-graph gate and
noted the consequence it accepted rather than solved: "a module that carries
sample, example or alternate catalogs now merges all of them". Its Option D —
explicit scope, package patterns or globs in `asyncgo.yaml` — lost on scope, and
the ADR left it open "if collision reports on multi-catalog modules prove too
blunt".

## Decision Drivers

- A multi-catalog module must be able to commit one document per catalog, each
  correct on its own terms (its own `Info`, no dropped keys).
- `generate` and `check` must not be able to drift: the artifact set `check`
  verifies has to be the set `generate` writes, derived from the same source.
- Backward compatibility: a module with one catalog keeps writing
  `<dir>/asyncapi.yaml`, with no flag and no config change.
- No new silent loss. A conflict, a missing artifact or an artifact nothing
  claims must be reported, not skipped.
- No new configuration surface when a naming convention plus one DSL call
  suffices.
- Keep CI's failure contract: one "regenerate" exit code, not a matrix of new
  ones.

## Considered Options

- **A — Destination kind selects the model.** A file destination keeps today's
  merged document; a directory destination (trailing separator) writes one
  document per catalog into it, named from the catalog.
- **B — Repeatable `--catalog` selector.** Narrow the merge from the command
  line, keeping `-o` a single-file destination.
- **C — Declarative `documents:` list in `asyncgo.yaml`.** Each entry names a
  catalog set and a destination; both commands walk the list.
- **D — Guardrail only.** Keep the merge, but make `-o <file>` an error when
  discovery finds more than one catalog.
- **E — Directory convention with names derived only from the declaring
  variable.** As A, with no way to choose the filename.

## Decision Outcome

Chosen option: **A**, because the two destinations are already distinct in
`resolveOutput`, so the model can be selected by an argument the user already
passes — while each catalog gains its own document, its own `Info` and its own
filename, with no new flag and no new config file.

The rule:

| Destination | Document model |
| --- | --- |
| omitted, or a file path | one merged document (today's behaviour, unchanged) |
| a directory (trailing separator) | one document per catalog, inside it |

### Decisions the option required

1. **A catalog carries its own document name.**
   `asyncgo.Spec(...)` gains `.Named("orders")`, which records the name on the
   `SpecResult` (`SpecResult.Name`) and returns the same value, so the chain
   stays intact. A method, not an `Item`, because the name labels an
   already-built catalog and a bare `Name` item among `Info`/`Servers`/`Channels`
   would read as if it named a component. The name is code, not configuration,
   so `check` can compute the expected artifact set from the same materialization
   it already performs.

2. **An unnamed catalog's name is derived, and never from the catalog count.**
   The default stem is `<declaring package basename>.<variable name>` —
   `orders.Catalog` for `var Catalog` in package `orders`. Deriving it from the
   count would rename an existing artifact the moment a second catalog appeared
   (`asyncapi.yaml` → `orders.Catalog.asyncapi.yaml`), silently breaking the
   committed file and `check` with it. A collision between two derived stems
   (packages with the same basename) is an error naming both catalogs, and the
   fix is an explicit `.Named()`.

3. **Per-catalog filenames are self-identifying:** `<stem>.asyncapi.<format>`,
   e.g. `docs/orders.asyncapi.yaml`. The `asyncapi` infix is what makes
   **orphan detection** possible without a manifest: `check` adopts files
   matching `*.asyncapi.<format>` and nothing else, so an unrelated
   `docs/architecture.yaml` is never mistaken for a stale artifact, and an
   artifact whose catalog was deleted is never silently unverified.

4. **The name is validated where the document is built.** It must match the
   grammar the specification already applies to component keys
   (`^[\w\d.\-_]+$`, `componentKeyPattern`), which doubles as the filename
   safety rule: no separator, no traversal, no empty stem. A violation is a
   catalog validation error, reported through the existing per-catalog report.

5. **`check` reports one verdict over the whole expected set.** Three finding
   classes share exit code 3: a file that differs (*stale*), an expected file
   that is absent (*missing*, e.g. a catalog added since the last generate), and
   a `*.asyncapi.<format>` file no catalog claims (*orphan*, e.g. a deleted or
   renamed catalog). The message says "delete it" for an orphan and "run
   `asyncgo generate`" otherwise, but CI keeps branching on one code. A missing
   artifact stays exit 1 in *file* mode, unchanged: there the caller named the
   one path, so its absence is a setup error rather than a statement about the
   committed set.

6. **The artifact set is written all-or-nothing.** `generate` validates names,
   rejects duplicates, and encodes every document *before* writing the first
   file. `writeFileAtomic` guarantees one file is never half-written; without
   this step a failure on the third of five documents would leave a directory
   that is neither the old set nor the new one.

### Why the other options lost

- **B (`--catalog`)** solves selection but not naming: every invocation still
  needs a shell loop and an explicit `-o` per catalog, so the artifact set lives
  in CI scripts instead of the repository. It also leaves `-o <file>` able to
  merge an arbitrary subset lossily, which is the footgun A removes.
- **C (`documents:` in `asyncgo.yaml`)** is the most explicit, but it makes the
  document set a second source of truth next to the catalogs themselves: a
  renamed catalog has to be fixed in two places, and `check` verifies whatever
  the config happens to declare rather than what the module actually contains.
  It is the right shape only if catalogs ever need to be *grouped* arbitrarily;
  `.Named()` covers the naming need at a fraction of the surface.
- **D (guardrail)** is honest but inert: it prevents the lossy write without
  making the correct output reachable.
- **E** drops the point of the change — a filename derived from a Go identifier
  is the naming problem, not the solution to it.

### Consequences

- Good, because each catalog gets a document with its own `Info`, so multi-team
  module documents stop being "first one wins".
- Good, because generate and check read the same rule off the same argument:
  parity is structural, not a documented convention that can rot.
- Good, because a module with one catalog is untouched — same default path, same
  bytes, no flag, no migration.
- Good, because orphan artifacts become visible, which the merged-single-file
  model could never express.
- Bad, because a trailing path separator now carries the whole document model.
  `-o docs` and `-o docs/` do different things, and a shell typo is silent. The
  mitigation is output, not a flag: directory mode prints one `wrote` line per
  file, so the shape of the run is visible at default verbosity.
- Bad, because files silently change name when a catalog gains a `.Named()`, and
  `check` then reports the old file as an orphan. The rename is loud (exit 3,
  named file) but it is a manual two-step: rename the artifact, regenerate.
- Bad, because `asyncgo.yaml` still only holds mappings. A future "group these
  catalogs into one document" requirement reopens Option C, and this ADR does
  not pre-empt it.
- Neutral, because `.Named()` is public API: it is additive, but `SpecResult`
  gains a field and the harness envelope gains `name`, so `internal/discovery`
  and `internal/cli` carry it end to end.

## More Information

- Supersedes: the single-document assumption in `internal/cli/root.go`
  (`resolveOutput`) and `internal/discovery/build.go` (`Build`), documented as
  "A module with several catalogs merges them into one document" in
  `site/content/cli.md`.
- Related: [ADR 0010](0010-catalog-scope-is-the-module.md) — the multi-catalog
  merge this ADR makes avoidable; its Option D is option C here.
- Related: [ADR 0009](0009-verbose-cli-logging.md) — directory mode reports the
  written set through the existing `write` stage rather than adding output.
- Implementation: `builder.go` (`SpecResult.Name`, `Named`),
  `internal/discovery/materialize.go` (harness envelope `name`),
  `internal/discovery/build.go` (`BuildAll`),
  `internal/discovery/merge.go` (`Build` as `Merge(BuildAll(...))`),
  `internal/cli/generate.go` / `internal/cli/check.go` (destination mode).
