# 0010. Scope catalog discovery to the module, not the main import graph

- Status: accepted
- Deciders: asyncgo maintainers
- Created: 2026-11-10
- Status updated: 2026-11-10

## Context and Problem Statement

Discovery located catalogs (`*asyncgo.SpecResult` package variables) only in
packages reachable from a `main` package by following import edges inside the
loaded module, with a fallback: a module with no `main` package had every package
considered reachable.

The rule was documented (`getting-started.md`, `cli.md`, `AGENT.md`) as a hard
requirement — "the catalog must be reachable from `main`" — but the
implementation, the tests and the fixtures disagreed with it:

- Fifteen of the sixteen `test/data` fixtures have no `main` package at all and
  were generated through the fallback. The one that has one,
  `test/data/simple/cmd/main.go`, existed *only* to exercise the reachability
  walk.
- No test asserted that an unreachable catalog is skipped. The helper appeared in
  tests solely to build the set the other walkers consumed.
- Reachability gated four traversals, not just the catalog scan: combinator
  references, alias edges and field descriptions were all narrowed by the same
  import walk.

As a rule it was also weaker than it looked. A catalog is topology — a statement
about the module, not about one binary — and the walk was a union over *all*
mains, so it never disambiguated a module that carried several catalogs (dev and
prod variants, an example package). What it did do was fail non-monotonically:
adding a `cmd/` directory to a module that previously had none could empty the
document ("no AsyncAPI catalogs ... reachable from main") or change it, with no
other clue. A library module — the primary audience for a code→spec generator —
had no main at all, so the constraint it documented was vacuous.

## Decision Drivers

- A document describes the **module**; no single binary should decide its
  contents.
- No silent loss: a catalog the generator declines to include must say why.
- Predictability: the same module must not generate different documents because a
  `main` package was added, moved or deleted.
- Keep the harness buildable: `package main` is a program and cannot be imported
  by the generated harness.
- Keep `generate`'s default output quiet; verbosity stays opt-in
  ([ADR 0009](0009-verbose-cli-logging.md)).

## Considered Options

- **A — Remove the gate.** Scope becomes every package under `./...` in the
  module, minus `package main`. `reachableFromMain` and the `reachable` parameter
  threaded through four walkers disappear.
- **B — Remove the gate, keep the signal.** As A, plus a warning when a module has
  a `main` package that imports no catalog: "… is still generated".
- **C — Keep the gate as an opt-in.** Default to the whole module, restore the old
  filter behind `--from-main` or `discover.fromMain` in `asyncgo.yaml`.
- **D — Replace the gate with explicit scope.** Package patterns (`--packages
  ./catalog/...`) or include/exclude globs in `asyncgo.yaml`.

## Decision Outcome

Chosen option: **B**, because it removes the unpredictability of an import-graph
scope while keeping the one piece of information the walk was actually good for:
telling the user their catalog is not wired into any binary.

The `main`-reachability walk survives as `importedByMain`, an advisory input with
no effect on what is scanned. It returns nothing for a module without a `main`
package, which is why the warning also checks for the existence of a main before
reading absence from it.

- Option A lost only the diagnosis; it is otherwise the same change, and B adds
  one warning record.
- Option C lost because two discovery modes double the test matrix and the
  documentation surface to serve a rule that no fixture and no test relied on.
- Option D lost because it solves a real but different problem — a repo wanting
  many documents — and would have widened the config surface in a change whose
  point is to narrow behaviour. It remains available if collision reports on
  multi-catalog modules prove too blunt.

### Consequences

- Good: the document no longer depends on the import graph, so adding, moving or
  removing a `main` package cannot change or empty it.
- Good: library modules are covered by the same rule as binaries, and
  `package main` is excluded explicitly rather than by accident of
  unimportability.
- Good: the rule is testable — `TestBuildIgnoresMainImportGraph` compares a module
  whose `main` imports the catalog against a copy whose `main` imports nothing and
  requires byte-identical YAML.
- Bad: a module that carries sample, example or alternate catalogs now merges all
  of them. Their keys are first-wins and every dropped key is reported as a
  collision, so the loss is loud, but it is a merge where the old walk silently
  picked one side.
- Bad: the four traversals now walk every module package, so `_test`-only helper
  packages are scanned for combinator refs and alias edges. The harness imports any
  package a combinator ref names, so a sample package with a build-tag-only
  compile error can now fail the harness build.
- Neutral: a catalog declared in `package main` is not generated. It was
  unreachable-by-import before (except as the main itself) and it could never have
  been materialized; the exclusion is now explicit in `scannable`.

## More Information

- Supersedes the "restricted to packages reachable from a main package" rule in
  `internal/discovery/discover.go` and the "must be reachable from `main`" note in
  the docs.
- Implementation: `internal/discovery/discover.go` (`scannable`,
  `importedByMain`), `internal/discovery/build.go`
  (`warnCatalogsOutsideMain`).
- Related: [ADR 0009](0009-verbose-cli-logging.md) — the warning is a default
  output line, not a `-v` detail.
