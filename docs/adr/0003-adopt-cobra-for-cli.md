# 0003. Adopt Cobra for the CLI command structure

- Status: accepted
- Deciders: asyncgo maintainers
- Created: 2026-08-31
- Status updated: 2026-08-31

## Context and Problem Statement

The CLI (`cmd/asyncgo`) is a single `package main` file that dispatches
subcommands by hand — a `switch` on `args[0]` — and parses flags with a
stdlib `flag.FlagSet` per subcommand. Today that is only three subcommands
(`generate`, `check`, `version`) and one flag (`-o/--output`), so it is
manageable.

As the CLI grows (more subcommands, persistent flags, `--help`/`--version`,
shell completion, generated docs), the manual dispatch and per-subcommand flag
parsing become duplicated, error-prone boilerplate. The project needs a command
structure that scales without reworking the dispatch on every addition.

## Decision Drivers

- Easy expansion: add subcommands and flags with minimal, uniform boilerplate.
- Standard tooling: prefer the de-facto standard library over a bespoke frame.
- Preserve the release pipeline: goreleaser stamps `-X main.version/commit/date`.
- Testability: commands must remain directly unit-testable.
- No behavior change to the actual generation/checking logic (only the CLI shell).

## Considered Options

- Adopt Cobra, with the command tree in an `internal/cli` package (thin `main`).
- Adopt Cobra, keeping `package main` split across files.
- Adopt `urfave/cli` (the main alternative framework).
- Keep the hand-rolled dispatch and grow it incrementally.

## Decision Outcome

Chosen option: "Adopt Cobra, with the command tree in an `internal/cli`
package", because it provides the standard, extensible command structure while
keeping `main` a thin shim that preserves the release pipeline.

- The `package main` split was rejected because it keeps the CLI coupled to
  `main` (not importable/testable as a package) and offers no growth advantage
  over the `internal/cli` layout.
- `urfave/cli` was rejected because Cobra is the more widely adopted,
  better-documented de-facto standard, with completion and docs generation we
  may want later.
- The hand-rolled option was rejected because manual dispatch and
  per-subcommand flag parsing do not scale.

### Consequences

- Good, because new subcommands and flags are added as small `*cobra.Command`
  values with `AddCommand` and `StringVarP`, not new `switch` arms and
  `flag.FlagSet`s.
- Good, because build metadata stays in `main` and is injected into the CLI
  (a `BuildInfo` value), so goreleaser's `-X main.version/commit/date` is
  unchanged.
- Good, because `--help`, `--version`, and usage text come for free; the manual
  `-o`/`--output` alias collapses to one `Flags().StringVarP(..., "output",
  "o", ...)`.
- Good, because commands are testable via `SetArgs`/`SetOut`/`SetErr`, and the
  pure logic (`resolveDir`, `resolveOutput`, `resolveVersion`,
  `discovery.Build`) remains directly unit-testable.
- Bad, because it adds a dependency (`spf13/cobra` plus `spf13/pflag`, and
  `inconshreveable/mousetrap` on Windows) that must be vendored — relaxing the
  "avoid heavy frameworks" convention.
- Bad, because a few CLI edge behaviors adopt Cobra's cleaner defaults (no-args
  prints help; bad-flag errors report `unknown flag`) rather than reproducing
  the old hand-written `usage:` strings; the affected tests are updated instead
  of adding custom usage plumbing.
- Bad, because the test suite migrates from `package main` white-box tests to
  the `internal/cli` package.

## More Information

- No design doc: this is a small, internal change to the CLI shell, not a
  change to the library's capabilities (see docs/adr/README.md — the
  design-doc step is optional).
