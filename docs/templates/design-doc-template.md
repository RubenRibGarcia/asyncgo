# Design: <short title>

<!--
How to use this template
1. Copy this file to docs/designdoc/<kebab-case-title>.md.
2. Replace every <placeholder> and delete this comment block.
3. Keep "Status: Proposed" until the design is reviewed and accepted,
   then flip it to "Accepted". Once accepted, distill the decisions into
   an ADR (see docs/adr/README.md).
-->

- **Status**: Proposed
- **Created**: <YYYY-MM-DD>
- **Status updated**: <YYYY-MM-DD>
- **Scope**: `<packages/dirs touched, e.g. schema/, internal/discovery/>`
- **Raw notes**: `<optional docs/scratch/<brainstorm file>, gitignored — else delete this line>`

## Summary

<Two or three sentences: what is being changed and why.>

## Background

<Constraints and context that every decision below depends on. Include the
architecture facts, precedents, or dialect/spec details that constrain the
solution. Use a table or diagram where it clarifies the split between passes,
layers, or tools.>

## Goals / Non-goals

**Goals**

1. <Measurable goal 1>
2. <Measurable goal 2>

**Non-goals (v1)**

- <Explicitly deferred work, so reviewers don't scope-creep>

## Design decisions

### D1 — <one-line decision>

<The decision, stated concretely.>

**Rationale**: <Why this over the alternatives.>

### D2 — <one-line decision>

<The decision.>

**Rationale**: <Why.>

<!-- Repeat Dn as needed. Each decision should be independently justifiable. -->

## Detailed design

### 1. <area, e.g. schema/tags.go>

<Concrete changes, signatures, and code/schema snippets. Reference exact files
and symbols.>

### 2. <area, e.g. registry + finalization>

<...>

<!-- Repeat per area. Keep snippets minimal but exact. -->

## Example (before / after)

<Input code/schema, then the expected output. Abridged is fine; annotate where
the interesting transformation happens.>

## Edge cases

- <Edge case> — <how it is handled>
- ...

## Rollout plan

1. **Stage 0 — <name>** (`<conventional-commit scope>`): <increment, no behavior change>
2. **Stage 1 — <name>** (`<scope>`): <increment>
3. **Stage 2 — <name>** (`<scope>`): <increment>
4. **Stage 3 — deferred**: <future work>

<!-- Each stage should be independently mergeable and testable. -->

## Testing plan

- **Stage N**: <test strategy — table-driven cases, golden tests, race check>
  - `should_<behavior>`
  - `should_<behavior>`
- **Race**: `go test ./... -race` per AGENT.md.

<!-- Subtest names follow AGENT.md: snake_case, start with should_. -->

## Open / deferred

- <Follow-up work and why it's out of scope for this design.>
