# Vendored AsyncAPI 3.1.0 JSON Schema — provenance

`asyncapi-3.1.0.json` in this directory is a verbatim copy of the published
AsyncAPI 3.1.0 JSON Schema. It is embedded into the binary with `//go:embed`
(`validation.go`) and is **never fetched at run time**.

| | |
| --- | --- |
| Upstream repository | <https://github.com/asyncapi/spec-json-schemas> |
| Upstream path | `schemas/3.1.0.json` |
| Pinned tag | `v6.11.1` |
| Pinned commit | `e609fc2341007395d75df5756fc6fccf662c2087` (the commit tag `v6.11.1` points at) |
| Downloaded from | <https://raw.githubusercontent.com/asyncapi/spec-json-schemas/v6.11.1/schemas/3.1.0.json> |
| Retrieved | 2026-11-01 |
| Size | 416137 bytes |
| SHA-256 | `51d3274899ad2875f25c18fd1aef4d5512f0a97be785d519740bde55a4162f61` |
| `$id` | `http://asyncapi.com/definitions/3.1.0/asyncapi.json` |
| `$schema` | `http://json-schema.org/draft-07/schema` |

The digest is also pinned in code as `pinnedSHA256` and asserted by
`TestValidate/should_embed_the_pinned_schema`, so an edit to the vendored bytes
fails the build rather than silently changing what the validator accepts.

## Why a single file

The released `schemas/3.1.0.json` is already a **self-contained bundle**: all 112
sub-schemas — including every protocol binding, e.g.
`http://asyncapi.com/bindings/kafka/0.5.0/message.json` — live inside
`definitions`, each carrying the matching `$id`, and every `$ref` in the document
resolves internally. No companion documents and no custom loader are needed, so
the validator compiles it offline with `AddResource` + `Compile`.

Two properties worth knowing before editing this file:

- The only dangling reference, `path/to/user-create.avsc#/UserCreate`, appears
  inside an `examples` value (three binding examples and one
  `components.schemas` example). It is never compiled; a validator that resolves
  references eagerly would choke on it.
- `definitions` also contains `http://json-schema.org/draft-07/schema`, which
  collides with the validator's built-in draft-07 metaschema. `internal/validation`
  therefore registers only the root document — never the individual definitions.

## Re-vendoring

Upgrading is a deliberate commit. From the repository root:

```bash
curl -sSfL -o internal/validation/asyncapi-3.1.0.json \
  https://raw.githubusercontent.com/asyncapi/spec-json-schemas/v6.11.1/schemas/3.1.0.json
sha256sum internal/validation/asyncapi-3.1.0.json
```

Then:

1. Update `pinnedSHA256` in `internal/validation/validation.go`.
2. Update this file (tag, commit, retrieval date, size, digest).
3. Re-check the `components.<field>` key patterns — the in-memory relaxation in
   `relaxComponentKeys` assumes each map declares a single `patternProperties`
   and no `additionalProperties`. If upstream changes that shape, revisit
   [ADR 0007](../../docs/adr/0007-vendored-schema-validation.md).
4. Re-check whether mqtt5/mercure bindings have been added upstream; if so,
   remove them from `omittedBindings` and drop the `--ignore-missing-bindings`
   guard in `test/integration`.
5. Run `make test` — the digest test, the unit tests, and the integration
   assertion all exercise the new bytes.
