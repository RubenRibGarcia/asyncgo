# Architecture Decision Records

An Architecture Decision Record (ADR) captures a significant architectural
decision, the context that led to it, and its consequences. It is a durable,
in-repo memory of *why* the code looks the way it does — reviewed decisions,
not lost in review threads or chat.

ADRs are written in [MADR](https://adr.github.io/madr/) format. Use the template
at `docs/templates/adr-template.md`.

## Naming

Each ADR is a file named `NNNN-kebab-case-title.md` in this directory, numbered
sequentially from `0001`.

```text
docs/adr/
├── README.md               # this file
├── 0001-some-decision.md   # first decision
├── 0002-another-decision.md
└── ...
```

The next number is the highest existing number plus one. Never reuse a number.

## Workflow

A design doc precedes an ADR for any non-trivial change:

1. **Write the design doc** — create `docs/designdoc/<kebab-case-title>.md` from
   `docs/templates/design-doc-template.md`, with `Status: Proposed`.
2. **Review and accept** — discuss, revise, and reach agreement. Flip the
   design doc to `Status: Accepted`.
3. **Record the ADR** — distill the accepted design's decisions into a new ADR
   here, with `Status: accepted`, linking back to the design doc under
   "More Information".

For a small, self-contained decision that doesn't warrant a full design doc, an
ADR can be written directly — the design-doc step is optional, the ADR is not.

## Status

| Status       | Meaning                                                       |
| ------------ | ------------------------------------------------------------- |
| `proposed`   | Draft under discussion, not yet binding                       |
| `accepted`   | The decision is in force                                      |
| `deprecated` | No longer in force, but not yet superseded (state the reason) |
| `superseded` | Replaced by a later ADR (link it under "More Information")    |

## Index

| # | Title | Status | Created | Status updated |
| - | --------------- | ------ | ------- | -------------- |
| [0001](0001-schema-composition-from-go-structs.md) | Derive allOf / oneOf / anyOf composition from Go structs | accepted | 2026-08-18 | 2026-08-18 |
| [0002](0002-custom-schema-providers.md) | Custom schema providers for types with custom (de)serialization | accepted | 2026-08-29 | 2026-08-29 |
