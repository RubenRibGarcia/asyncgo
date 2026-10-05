# 0005. Extend the Schema Object type to support Multi Format Schema Objects

- Status: accepted
- Deciders: asyncgo maintainers
- Created: 2026-10-05
- Status updated: 2026-10-05

## Context and Problem Statement

Payloads and headers are `*spec.Schema` only, with no `schemaFormat`, so Avro and
Protobuf payloads — the norm for Kafka, the library's flagship protocol — cannot
be described. AsyncAPI 3.1.0 defines the Multi Format Schema Object
(`schemaFormat` plus an opaque `schema` body) and accepts it wherever a schema is
accepted: `Message.payload`, `Message.headers`, `MessageTrait.headers`, and
`components.schemas` map values. The library needs a Go representation for that
union.

The decisive constraint is how a catalog becomes a document: a generated harness
marshals it to YAML, and `internal/discovery/materialize.go` unmarshals it back
into `*spec.AsyncAPI` before the final encode. Any representation whose node is
not a concrete struct decodes into `map[string]any` and re-marshals with sorted
keys, which rewrites every committed golden and breaks the `asyncgo check`
byte-equality drift gate.

## Decision Drivers

- Byte-stable output: the eight committed goldens and the `asyncgo check` gate
  must not move.
- Accepted anywhere a schema is accepted today, with no field-type change to
  `Message`, `MessageTrait`, or `Components`.
- `components.schemas` must become reachable from the DSL, so one Avro schema is
  shared by `$ref` rather than repeated inline on every message.
- `schemaFormat` is emitted verbatim: the library generates documents rather than
  validating formats, and the spec permits custom values.
- Contain the invalid states that a unified struct makes representable.

## Considered Options

- Extend `spec.Schema` with `SchemaFormat`/`Schema` and add a `spec.MultiFormat`
  constructor.
- Nominal `spec.MultiFormatSchema` behind a sealed union interface on
  `Message.Payload`/`Headers`, `MessageTrait.Headers`, and `Components.Schemas`.
- Nominal `spec.MultiFormatSchema` with those fields typed `any`.

## Decision Outcome

Chosen option: "Extend `spec.Schema`", because keeping the schema node a struct
is the only representation that survives the harness round-trip byte-for-byte
while requiring no field-type change at any schema location.

- The sealed-union option was rejected because an interface field has no concrete
  decode target, so it reorders keys exactly like `any` and would additionally
  need custom `MarshalJSON`/`MarshalYAML`/`UnmarshalYAML` at four locations — a
  breaking public API change for no functional gain.
- The `any` option was rejected because it is precisely the shape that a
  marshal → unmarshal → marshal probe shows breaking every golden.

### Consequences

- Good, because a multi-format schema is accepted in `payload`, `headers`, and
  `components.schemas` with no change to those types, and every existing
  generated document stays byte-identical.
- Good, because `schemaFormat` serializes in declaration order ahead of `schema`,
  and the opaque body is never parsed, validated, or rewritten.
- Good, because `Schema(name, *spec.Schema)` + `Schemas(...)` make
  `components.schemas` reachable from the DSL, so one declaration is `$ref`'d
  from several messages.
- Good, because `MessageFrom(name, *spec.Schema)` gives a payload with no Go type
  to reflect a first-class constructor rather than a silent override.
- Bad, because one struct makes invalid states representable — `schemaFormat`
  beside JSON Schema keywords, or either field alone — which needs a
  declaration-time validation post-pass.
- Bad, because inside the opaque body, map keys are emitted sorted after
  materialization (`fields, name, type`). Cosmetic for Avro JSON, and identical
  to how `Schema.Properties` keys already behave.
- Bad, because external `$ref` bodies are emitted verbatim but not bundled, so a
  fixture using one cannot be resolved by the spec validator.

## More Information

- Design doc: docs/designdoc/multi-format-schemas.md
- Issue: <https://github.com/RubenRibGarcia/asyncgo/issues/15>
- Related: [0002](0002-custom-schema-providers.md) — the other "declare it when it
  cannot be derived" surface.
- Implementation note (2026-10-05): the fixture shipped as a single
  `test/data/multiformat` module covering Avro, OpenAPI 3.0.0, RAML 1.0, and
  Protobuf, not the standalone `test/data/avro` the design doc planned; that doc
  carries the amendment. The pinned `asyncapi/cli` validates Avro, OpenAPI, and
  RAML bodies that are mappings, resolves a string-valued `schema` as a
  reference, and has no Protobuf parser — so the combined fixture is excluded
  from the CLI check by a self-policing entry in `test/integration`, and Protobuf
  stays emitted-but-unvalidated rather than silently untested.
