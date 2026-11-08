# Project

`asyncgo` keeps its reasoning in the repository, not in review threads. This
section publishes those records: why the code looks the way it does, the designs
behind each major feature, and how much of AsyncAPI 3.1.0 the library can actually
express today.

## AsyncAPI 3.1.0 coverage

A point-in-time, field-by-field assessment of how much of the specification
`asyncgo` can model, express through its DSL, and emit — object by object, with
the gaps and the backlog spelled out.

[Read the coverage assessment](asyncapi-3.1.0-coverage.md){ .md-button }

## Architecture Decision Records

An ADR captures a significant architectural decision, the context that led to it,
and its consequences. Each is written in [MADR](https://adr.github.io/madr/) format,
numbered sequentially, and never renumbered or reused.

The decisions are listed in the sidebar under **Architecture Decision Records**.
Start with the [overview and index](adr/README.md) for the status vocabulary and
the workflow.

## Design documents

A design doc proposes a non-trivial change *before* it is built: what changes,
why, and how — with the alternatives and trade-offs made explicit. Once a design
is accepted, its decisions are distilled into an ADR that links back to it.

The documents are listed in the sidebar under **Design docs**; the
[overview](designdoc/README.md) covers naming and the review process.

## The workflow that ties them together

1. **Propose** — copy `docs/templates/design-doc-template.md` into `docs/designdoc/`
   and fill it in with `Status: Proposed`.
2. **Review and accept** — discuss and revise, then flip the design doc to
   `Status: Accepted`.
3. **Record the ADR** — distill the accepted decisions into
   `docs/adr/NNNN-kebab-case-title.md`, with `Status: accepted`, linking back to
   the design doc under "More Information".
4. **Implement** — build the change with the design doc and ADR as the reference.

Small, self-contained changes (bug fixes, minor refactors) need neither; a
self-contained decision that does not warrant a design doc can go straight to an
ADR — the design-doc step is optional, the ADR is not.

## Contributing

See [`AGENT.md`](https://github.com/RubenRibGarcia/asyncgo/blob/master/AGENT.md)
for the repository layout, the public-API map, testing and coverage rules, and the
commit conventions, and the
[PR template](https://github.com/RubenRibGarcia/asyncgo/blob/master/.github/PULL_REQUEST_TEMPLATE.md)
for what a pull request is expected to contain.
