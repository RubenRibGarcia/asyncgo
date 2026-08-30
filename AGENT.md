# asyncgo — AGENT.md

## Project Summary

A Go library that **generates** an
[AsyncAPI v3.1.0](https://www.asyncapi.com/docs/reference/specification/v3.1.0)
specification document from Go code — the *code → spec* direction that most Go
AsyncAPI tooling (which goes *spec → code*) leaves unserved.

`asyncgo` is a **documentation generator**, not a messaging framework. It does
not route messaging; it derives a committed `asyncapi.yaml` from two touchpoints:

1. **Structs** (data contracts) — message payload schemas are derived via
   reflection.
2. **A typed, declarative catalog** (topology) — channels, operations, servers,
   and bindings declared once through a fluent DSL.

The CLI (`asyncgo generate` / `asyncgo check`) discovers catalogs reachable from
`main`, materializes them, and emits (or verifies) the committed artifact.

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      User code                          │
│   structs (schema source) + asyncgo.Spec(...) catalog   │
└───────────────────────┬─────────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────────┐
│               asyncgo (fluent DSL, root pkg)            │
│  Spec, Info, Server, Channel, Operation, MessageOf,     │
│  Kafka/AMQP/NATS/MQTT binding helpers.                  │
└───────┬─────────────────────────────────┬───────────────┘
        │ builds                         │ references structs
┌───────▼───────────────┐   ┌─────────────▼───────────────┐
│   spec (object model) │   │ schema (struct → JSON Schema)│
│  AsyncAPI 3.1.0 types │   │ FromType(reflect.Type)       │
│  + codecs   │   │ fully-qualified, hoisted     │
└───────┬───────────────┘   └─────────────────────────────┘
        │
┌───────▼─────────────────────────────────────────────────┐
│         internal/discovery + cmd/asyncgo                │
│  1. go/packages: find *spec.AsyncAPI vars reachable     │
│     from main (static, no execution)                    │
│  2. Generate a harness that imports catalog packages    │
│     and prints them as YAML (runs init only)            │
│  3. Merge catalogs       │
│  4. Emit YAML (generate) or diff (check)                │
└─────────────────────────────────────────────────────────┘
```

### Key Design Decisions

1. **Hybrid approach** — schemas derive from real structs via reflection (no
   duplication); topology is declared once in a typed, compiler-checked catalog
   (no magic-string comments).

2. **Fully-qualified schema names** — hoisted schemas are keyed
   `pkgPath.TypeName` (e.g. `example.com/orders/orders.OrderPlaced`); `$ref`
   escapes `/`→`~1`, `~`→`~0` per RFC 6901.

3. **Optional by default** — a field is required only when tagged
   `asyncapi:"required"`. Field descriptions are read from the field's doc
   comment by the discovery pass, not from the tag.

4. **Always hoist** — named struct types go into `components.schemas` and are
   referenced via `$ref`; only anonymous inline types are inlined.

5. **Static discovery + harness materialization** — catalogs are located
   statically via `go/packages`; their values are materialized by running a
   generated harness (which executes only the catalog packages' `init`, never
   `main`). This avoids executing user code while still giving `schema/` a real
   `reflect.Type`.

## Public API

| Area | Symbols |
| ---- | ------- |
| DSL | `asyncgo.Spec`, `Info`, `DefaultContentType`, `Servers`, `Server`, `Channels`, `Channel`, `Operation`, `MessageOf`, `Kafka`/`AMQP`/`NATS`/`MQTT`/`Binding` helpers |
| Spec model | `spec.AsyncAPI`, `Info`, `Server`, `Channel`, `Operation`, `Message`, `Schema`, `Components`, `*Bindings`, protocol binding structs |
| Schema | `schema.FromType(reflect.Type, defs)`, `schema.Name`, `schema.Ref` |
| CLI | `asyncgo generate [dir]`, `asyncgo check [dir]` |

## Directory Structure

```
asyncgo/
├── AGENT.md                    # This file — project plan & conventions
├── README.md                   # User-facing docs
├── go.mod / go.sum
├── go.work                     # multi-module workspace (root + test/data/* + tools)
├── .goreleaser.yaml            # release build: cross-compile + version ldflags
├── .github/workflows/          # CI (test.yaml) + release (release.yml)
├── spec/                       # Typed AsyncAPI 3.1.0 object model + codecs
├── schema/                     # struct → JSON Schema (the "data contract" half)
├── cmd/asyncgo/                # CLI: generate | check | version
├── docs/                       # development process (see "Development Workflow")
│   ├── adr/                    #   Architecture Decision Records (MADR format)
│   ├── designdoc/              #   design docs (proposal → review → ADR)
│   └── templates/              #   adr/design-doc templates
├── internal/discovery/         # catalog discovery + materialization (not public API)
├── test/data/                  # discovery test fixtures (simple, allof, oneof, anyof, provider — each its own Go module)
└── test/integration/           # end-to-end golden test against the test/data/ fixtures
```

The root package (top-level `.go` files) is the fluent DSL. See
[Public API](#public-api) for the symbols each area exposes.

## Development Workflow

Major features and significant changes follow a documented design process. The
process exists to give the repo a durable, in-repo memory of *why* decisions
were made, and is defined in two canonical files:

- **[docs/designdoc/README.md](docs/designdoc/README.md)** — design docs. A
  major feature is proposed *before* it is built from
  `docs/templates/design-doc-template.md`, then reviewed and accepted.
- **[docs/adr/README.md](docs/adr/README.md)** — Architecture Decision Records
  (MADR format). Accepted decisions are distilled into a numbered
  `docs/adr/NNNN-kebab-case-title.md`.

When a change warrants it, the workflow is:

1. **Propose** — copy `docs/templates/design-doc-template.md` into
   `docs/designdoc/`, fill it in with `Status: Proposed`.
2. **Review & accept** — discuss and revise, then flip the design doc to
   `Status: Accepted`.
3. **Record** — distill the accepted decisions into a new ADR under
   `docs/adr/` (from `docs/templates/adr-template.md`), linking back to the
   design doc.
4. **Implement** — build the change with the design doc and ADR as the
   contract.

This process applies to major features and big changes. Small, routine changes
(e.g. bug fixes, minor refactors) don't require a design doc or ADR — use
judgment, and refer to the canonical files above when unsure.

## Release

Releases follow [semantic versioning](https://semver.org/). A release is made
manually by pushing a new git tag following semver:

1. Tag the commit to release (`git tag vX.Y.Z`) and push it
   (`git push origin vX.Y.Z`).
2. Pushing the `v*` tag triggers the `release` workflow
   (`.github/workflows/release.yml`), which runs
   [goreleaser](https://goreleaser.com) (`.goreleaser.yaml`).
3. GoReleaser builds the `asyncgo` binary for linux/darwin/windows ×
   amd64/arm64, stamps the version via `-ldflags -X main.version=vX.Y.Z`, and
   creates the GitHub release with the archives, checksums, and source tarball.
   The workflow then attests the checksums and digests.

`asyncgo version` reports the stamped version (falling back to the module
version for `go install ...@vX.Y.Z` installs, and `devel` for local builds).
The version is the git tag — never stored in source.

## Conventions

- **Go version**: 1.25+
- **Dependencies**: `github.com/goccy/go-yaml` for YAML,
  `github.com/stretchr/testify` for test assertions,
  `golang.org/x/tools/go/packages` for discovery, standard library for JSON.
  Avoid heavy frameworks. No code generation — the DSL and model are hand-written.
- **Testing**: `make test` must pass. The `test/integration` golden test is
  end-to-end: it runs the generator against the `test/data/` fixtures
  (`simple`, `allof`, `oneof`, `anyof`, `provider`) and asserts each committed
  `asyncapi.yaml` is reproduced exactly.

  **Table-driven tests** — when a single test function covers multiple cases,
  use a table-driven test with `t.Run` subtests:

  ```go
  tt := []struct {
      name string
      in   any
      want string
  }{
      {name: "should_return_string_for_string", in: "x", want: "string"},
  }
  for _, tc := range tt {
      t.Run(tc.name, func(t *testing.T) {
          // ...
      })
  }
  ```

  **Subtest names** — use `snake_case` and start with `should_`:
  - `should_return_X` for success paths
  - `should_return_error` / `should_return_nil` for error and boundary cases

  **Assertions** — use `testify/assert` and `testify/require` for all
  assertions. Never call `t.Error`/`t.Errorf`/`t.Fatal`/`t.Fatalf` directly.

  - `assert.*` for soft failures (continue on failure)
  - `require.*` for hard failures (stop immediately)

  ```go
  doc, err := buildDocument()
  require.NoError(t, err)
  require.NotNil(t, doc)

  assert.Equal(t, "kafka", server.Protocol)
  assert.Equal(t, 3, len(channels))
  assert.Contains(t, channels, "order-placed")
  assert.Len(t, schemas, 5)
  assert.Empty(t, tags)
  assert.Error(t, err)
  ```

- **Lint & format**: `make lint` must pass. It runs `golangci-lint` (pinned in the
  `tools` module and invoked via `go tool golangci-lint`), executing both
  `golangci-lint run` (linter) and `golangci-lint fmt` (formatter).
- **Error handling**: wrap errors with context using `fmt.Errorf("...: %w", err)`.
- **Naming**:
  - Files: `snake_case.go`
  - Packages: single word, lowercase
  - Types: PascalCase
  - Receiver: single letter (e.g. `func (c *channel) apply(b *builder)`)
- **No panics in library code**: all errors are returned. The generated harness
  is the one exception — it may `panic` on unrecoverable write errors since it
  has no caller to return to.
- **Module layout**: the root package is the public DSL (`asyncgo`); `spec` and
  `schema` are public subpackages; `internal/discovery` and `cmd/asyncgo` are
  not part of the public API. `test/data/*` are separate modules joined via
  `go.work`, holding discovery test fixtures.

## Commit Conventions

All commits must follow the
[Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) spec.

### Format

```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

### Types

| Type       | When to use                                                   |
| ---------- | ------------------------------------------------------------- |
| `feat`     | New feature or public API                                      |
| `fix`      | Bug fix                                                       |
| `docs`     | Documentation only (README, AGENT.md, godoc comments)         |
| `style`    | Formatting, gofmt — no logic change                           |
| `refactor` | Code restructuring without changing behavior                  |
| `test`     | Adding or updating tests (no production code change)          |
| `chore`    | Maintenance: deps updates, `.gitignore`, tooling config       |
| `ci`       | CI/CD pipeline changes                                        |
| `perf`     | Performance improvement                                       |
| `revert`   | Revert a previous commit                                      |

### Scopes (project-specific)

| Scope      | Applies to                                                |
| ---------- | --------------------------------------------------------- |
| `spec`     | `spec/` — object model, codecs, bindings                |
| `schema`   | `schema/` — struct → JSON Schema reflection               |
| `dsl`      | root package — fluent DSL (`doc.go`, `message.go`, `bindings.go`) |
| `cmd`      | `cmd/asyncgo/` CLI                                        |
| `internal` | `internal/discovery/`                                     |
| `test`     | `test/data/` — discovery test fixtures                        |
| `docs`     | Project-level docs (README, AGENT.md, docs/)              |
| `deps`     | Dependency changes (`go.mod`, `go.sum`)                   |

### Examples

```
feat(dsl): add NATS and MQTT binding helpers
fix(spec): preserve integer bindings through materialization
refactor(internal): extract catalog materialization into discovery
test(schema): add table-driven tests for scalar derivation
chore(deps): switch yaml.v3 to goccy/go-yaml
docs: document schema derivation rules in README
```

### Breaking changes

Append `!` after the type/scope or add a `BREAKING CHANGE:` footer:

```
feat(dsl)!: remove deprecated MessageOf alias

BREAKING CHANGE: MessageOf now requires an explicit name argument.
```
