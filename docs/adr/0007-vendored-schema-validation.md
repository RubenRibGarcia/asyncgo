# 0007. Validate documents against a vendored AsyncAPI 3.1.0 JSON Schema

- Status: accepted
- Deciders: asyncgo maintainers
- Created: 2026-11-01
- Status updated: 2026-11-01

## Context and Problem Statement

`asyncgo check` compares bytes against the committed artifact, so a structurally
invalid document is reported as "out of date", never as "invalid". The only real
validation in the repository is the containerized `asyncapi validate` inside
`test/integration` — unavailable to users, and blind to most of the document
(see below). Issue [#25](https://github.com/RubenRibGarcia/asyncgo/issues/25) /
coverage item §8 B14 asks for a user-facing validation.

Three constraints decide this record:

- The schema must be pinned or vendored, never fetched at run time.
- Validation is defined over the JSON data model, not YAML.
- **The published schema does not look at `asyncgo`'s `components`.** All 19
  `components.<field>` maps use `patternProperties: {"^[\\w\\d\\.\\-_]+$": …}`
  with no `additionalProperties`, while `asyncgo` keys hoisted schemas
  `pkgPath.TypeName` — which contains `/`, matches nothing, and is therefore
  never validated. Measured: a catalog whose `SchemaProvider` returns
  `&spec.Schema{Type: "bogus"}` yields a document the official schema calls
  valid; the same schema under the key `X` is rejected at
  `/components/schemas/X/type`.

## Decision Drivers

- Offline operation: a CLI that needs the network (or Docker) cannot validate an
  air-gapped build, and the repo already pins its test-only validator image for
  the same reason.
- A violation report precise enough to act on: the JSON Pointer of the offending
  node, one line per node.
- Faithfulness to the official schema, with the smallest possible, documented
  deviation where the schema would otherwise be blind to our own output.
- No change to `generate` or `check`, whose semantics users depend on.
- Self-contained binary (goreleaser archives, `go run`, `go install`) — no
  data files to ship alongside.
- Minimal dependency surface: one validator, no framework.

## Considered Options

- **Command surface** — a new `asyncgo validate [dir]`; a `generate --validate`
  flag; both.
- **Validator** — `santhosh-tekuri/jsonschema/v6`; shelling out to the pinned
  AsyncAPI CLI container; a hand-written draft-07 subset.
- **Schema layout** — one released bundle file (`schemas/3.1.0.json`, 112
  inlined `definitions`); 113 separately vendored documents plus a URL-to-file
  loader.
- **Validation target** — the freshly built in-memory document; the committed
  artifact; both.
- **Component-key blind spot** — relax the patterns in memory; validate each
  `components` entry in a second pass; report only what the schema sees.
- **mqtt5 / mercure** — report the violations; add an explicit opt-out flag;
  silently suppress.

## Decision Outcome

Chosen options: a **new `asyncgo validate [dir]` subcommand**, using
**`santhosh-tekuri/jsonschema/v6`**, over **one vendored bundle** embedded with
`go:embed` and **pinned by SHA-256**, validating the **freshly built in-memory
document**, with the **component key patterns relaxed in memory**, and an
**`--ignore-missing-bindings`** flag for the protocols the bundle omits.

- `generate --validate` was rejected because it couples writing to validating and
  makes validation impossible without re-emitting; doing both commands was
  rejected as two surfaces to test for one behavior. Folding validation into
  `check` was rejected outright: it would change an existing drift gate's
  semantics and output for every user, and validation needs the JSON encoding
  while `check` verifies YAML bytes.
- The containerized CLI was rejected as a runtime dependency (Docker, offline
  builds) and because it cannot supply the JSON Pointers this feature exists to
  report. A hand-written draft-07 validator was rejected as a large, silently
  incomplete reimplementation.
- Vendoring 113 documents was rejected because the released `3.1.0.json` is
  already the bundle: all 112 sub-schemas live in `definitions` with matching
  `$id`s and every `$ref` resolves internally. The single file plus `go:embed`
  keeps the binary self-contained; the digest pin makes any drift a test
  failure, and `PROVENANCE.md` makes re-vendoring a reviewable commit.
- Validating the committed artifact was rejected because the mistake lives in the
  catalog, not the file: reporting it before anything is written names the real
  problem, and it avoids conflating "not regenerated" with "invalid".
- Validating `components` entries in a second pass was rejected as more code for
  the same coverage: the compiler cannot `Compile` a sub-schema by URL without
  registering all 112 `$id`s first, one of which collides with the built-in
  draft-07 metaschema. Copying each pattern's value into `additionalProperties`
  on the same map — 19 maps, in memory only — reaches the same result through
  the normal single-pass validation.
- Reporting only what the schema sees was rejected because it would leave the
  command reporting invalid hoisted schemas as valid — exactly the failure mode
  the issue is about.
- Silently suppressing mqtt5/mercure was rejected because it would hide genuine
  mistakes in those bindings; the flag makes the omission explicit at the call
  site.

### Consequences

- Good, because validation is offline, deterministic, and reports one line per
  offending node as an RFC 6901 pointer (`/channels/orders~1v1/address` for a
  key containing `/`), which is directly actionable.
- Good, because the binary stays self-contained and the vendored bytes are
  pinned: an edited schema fails a test, and an upgrade is a deliberate commit
  touching the file, the digest constant, and `PROVENANCE.md`.
- Good, because `generate` and `check` are untouched, and the DSL's own
  `CatalogErrors` remain the first, more useful report when a catalog is wrong.
- Good, because the integration test now asserts schema validity in-process for
  every fixture, so the package is exercised end to end and `bindings-unlisted`'s
  exclusion is guarded (it must still fail without the flag).
- Bad, because the validator runs against a schema that has been modified in
  memory: `asyncgo validate` can therefore report violations the AsyncAPI CLI
  (and the published schema applied verbatim) will not. This is deliberate and
  documented — the alternative is a validator blind to our own component keys.
- Bad, because the component-key deviation it compensates for remains: the spec's
  `^[\w\d\.\-_]+$` forbids the `/` in `pkgPath.TypeName`. Fixing it renames every
  hoisted key and every `$ref`, so it is follow-up work, not part of this change.
- Bad, because draft-07 `format` stays an annotation (v6 does not assert it by
  default), so `format: byte` on a non-base64 string is still accepted — matching
  the CLI's observed behavior, but not a proof of format correctness.
- Bad, because the binary grows by ~416 KB (compressed in release archives) and
  each process compiles the schema once (~milliseconds, cached with
  `sync.OnceValues`).
- Bad, because documents using mqtt5 or mercure bindings fail unless the flag is
  passed — 5 violations on `test/data/bindings-unlisted`, which the integration
  test documents and guards.

## More Information

- Design doc: [docs/designdoc/schema-validation.md](../designdoc/schema-validation.md)
- Issue: <https://github.com/RubenRibGarcia/asyncgo/issues/25>
- Coverage: [§8 B14](../asyncapi-3.1.0-coverage.md#b14), [§6](../asyncapi-3.1.0-coverage.md#6-validation)
- Related: [0003](0003-adopt-cobra-for-cli.md) — the command tree this extends;
  [0002](0002-custom-schema-providers.md) — `SchemaProvider`, the surface through
  which the component-key blind spot is reachable.
