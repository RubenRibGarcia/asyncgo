# 0006. Model the full Draft-07 and AsyncAPI Schema keyword set

- Status: accepted
- Deciders: asyncgo maintainers
- Created: 2026-10-10
- Status updated: 2026-10-10

## Context and Problem Statement

`spec.Schema` is documented as "a superset of JSON Schema Draft 07", but it
models only part of that vocabulary plus none of the three AsyncAPI-specific
Schema keywords. `docs/asyncapi-3.1.0-coverage.md` §4.2–§4.3 catalogues the gap
and §8 B7 tracks it: `discriminator`, `externalDocs`, `deprecated`, `const`,
`if`/`then`/`else`, `contains`, `propertyNames`, `patternProperties`,
`minItems`, `maxItems`, `uniqueItems`, `minProperties`, `maxProperties`,
`readOnly`, `writeOnly`, `$id`, `$schema` — and, once the spec's own bullet list
is cross-checked, also `examples`, `additionalItems`, `$comment`, and
`dependencies`.

The decisive constraints:

- **The schema node must stay a concrete struct.** The generated harness marshals
  a catalog to YAML and `internal/discovery/materialize.go` unmarshals it back
  before the final encode. A node that decodes to `map[string]any` re-marshals
  with sorted keys, which rewrites committed goldens and breaks the `asyncgo
  check` byte-equality gate. This is the same constraint that produced
  [0005](0005-multi-format-schema-objects.md).
- **Most of the new keywords are value-shaped, but one is not.** Draft-07 types
  `dependencies` as `Schema | [string]` per key — the only union in the set.

## Decision Drivers

- Byte-stable output: no committed golden may move, and `asyncgo check` must stay
  a usable drift gate.
- Spec fidelity without silent narrowing: a keyword that is modelled should mean
  what the spec says it means.
- Reachability, not just representation: a field only `spec.Schema` can carry is
  unreachable for a user writing a catalog from Go types.
- Deterministic output: one hoisted `components.schemas` entry per payload type is
  shared across channels, so nothing may depend on item order.
- No breaking change to the existing public API.

## Considered Options

- **Model plus derivation.** Add the fields, make the mechanically derivable ones
  settable from `asyncapi` struct tags, and add a Message-DSL builder for the
  Draft-07 `examples` array.
- **Model only.** Add the fields and stop there — no tag directives, no DSL.
- **`dependencies` as `map[string]any`** (faithful union, untyped).
- **`dependencies` as a named union type with custom codecs** (faithful union,
  typed).
- **`dependencies` as `map[string]*Schema`** (schema form only).

## Decision Outcome

Chosen option: "Model plus derivation", with `dependencies` as
`map[string]*Schema`.

- **Model only** was rejected because it leaves every new keyword unreachable from
  the library's differentiating feature — deriving a catalog from real Go types —
  and would have closed the coverage gap on paper only.
- **`map[string]any` for `dependencies`** was rejected after measurement, not
  assumption: a schema-valued entry stored in an `any` decodes to a map and
  re-encodes with sorted keys, so `type: object` and `properties:` swap order
  between the first and second marshal. That is exactly the byte-instability the
  one-struct design exists to prevent. The `[string]` form it would have
  preserved is sugar with an exact schema-form equivalent
  (`creditCard: [billingAddress]` ≡ `creditCard: {required: [billingAddress]}`),
  so modelling the schema form loses no expressiveness.
- **A named union type with custom codecs** was rejected as disproportionate: a
  hand-written `Marshal`/`Unmarshal` pair for both JSON and YAML, for a keyword
  Draft 2019-09 has already deprecated in favour of
  `dependentRequired`/`dependentSchemas`.

### Consequences

- Good, because all 23 keywords round-trip through both codecs and the YAML
  round-trip stays byte-stable, so no committed golden moves.
- Good, because the derivable subset is reachable from Go struct tags, and
  `Message.PayloadExamples` writes the Draft-07 `examples` array onto the
  *referenced* `components.schemas` entry — the only place it can take effect,
  since JSON Schema does not apply keywords beside a `$ref`.
- Good, because `schemaChildren` was extended in the same change, so the Multi
  Format invariant is still enforced *inside* `if`/`contains`/
  `patternProperties`/`dependencies` rather than silently stopping at the new
  keywords.
- Bad, because `example` (singular) and `examples` (plural) now both exist.
  `example` is not a 3.1.0 Schema keyword; it is kept working and marked
  deprecated rather than removed, which is a documented inconsistency (§7) and
  leaves two ways to express one idea.
- Bad, because `additionalItems` is modelled but inert: it applies only to the
  tuple form of `items`, which this model cannot express.
- Bad, because a malformed tag value (`minItems=abc`) is ignored rather than
  reported — `FromType`/`fillFields`/`applyTag` have no error channel, and
  threading one through would dwarf the feature.
- Bad, because a `PayloadExamples` call against a `$ref` whose target cannot be
  resolved is a build error, so a typo'd hand-written ref is louder than before.

## More Information

- Design doc: docs/designdoc/schema-keywords.md
- Issue: <https://github.com/RubenRibGarcia/asyncgo/issues/18>
- Related: [0001](0001-schema-composition-from-go-structs.md) — established the
  tag-directive vocabulary (`allOf`) and the hoisting model this extends;
  [0005](0005-multi-format-schema-objects.md) — the one-struct decision and its
  byte-stability constraint.
- Coverage: [§8 B7](../asyncapi-3.1.0-coverage.md#b7).
