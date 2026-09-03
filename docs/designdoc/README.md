# Design Docs

A design doc proposes a non-trivial change *before* it is built: what changes,
why, and how — with the alternatives and trade-offs made explicit. It is the
input to review. Once a design is accepted, its decisions are distilled into an
[ADR](../adr/README.md).

Design docs are written from `docs/templates/design-doc-template.md`.

## Naming

Each design doc is a file named `<kebab-case-title>.md` in this directory, named
after the feature or area it covers. No sequential numbering — that is what
ADRs are for.

```text
docs/designdoc/
├── README.md
├── schema-composition.md
└── ...
```

## Workflow

1. **Write the design doc** — copy `docs/templates/design-doc-template.md` into
   this directory and fill it in, with `Status: Proposed`.
2. **Review and accept** — discuss, revise, and reach agreement, then flip it
   to `Status: Accepted`.
3. **Record the ADR** — distill the accepted decisions into a new ADR in
   `docs/adr/` (see [docs/adr/README.md](../adr/README.md)).

## Status

| Status       | Meaning                             |
| ------------ | ----------------------------------- |
| `Proposed`   | Under review, not yet agreed        |
| `Accepted`   | Reviewed and agreed; ready to build |
| `Superseded` | Replaced by a later design doc      |

## Index

| Title | Status | Created | Status updated | ADR |
| ------------------------------------------- | -------- | ---------- | -------------- | --- |
| [schema-composition](schema-composition.md) | Accepted | 2026-08-15 | 2026-08-18 | [0001](../adr/0001-schema-composition-from-go-structs.md) |
| [custom-schema-providers](custom-schema-providers.md) | Accepted | 2026-08-29 | 2026-08-29 | [0002](../adr/0002-custom-schema-providers.md) |
| [fluent-dsl-validation](fluent-dsl-validation.md) | Accepted | 2026-09-01 | 2026-09-03 | [0004](../adr/0004-fluent-dsl-validation.md) |
