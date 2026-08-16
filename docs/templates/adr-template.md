# <NNNN>. <short title in sentence case>

<!--
How to use this template (MADR format)
1. Copy this file to docs/adr/NNNN-kebab-case-title.md, where NNNN is the next
   sequential number (see docs/adr/README.md for the numbering rule).
2. Replace every <placeholder> and delete this comment block.
3. Link the source design doc under "More Information" when one exists.
4. Status is lowercase: proposed, accepted, deprecated, or superseded.
   A newly created ADR is "accepted" once the decision is final.
-->

- Status: <proposed | accepted | deprecated | superseded>
- Deciders: <who made the decision — names or roles>
- Created: <YYYY-MM-DD>
- Status updated: <YYYY-MM-DD>

## Context and Problem Statement

<Describe the problem the project faces, and the forces at play: technical,
organizational, or external constraints. State the problem in a way that makes
clear why a decision is needed.>

## Decision Drivers

<!-- The forces that influenced the decision. Keep short; one per bullet. -->

- <driver, e.g. backward compatibility>
- <driver, e.g. deterministic output>
- <driver, e.g. minimal user boilerplate>

## Considered Options

- <Option A — short name>
- <Option B — short name>
- <Option C — short name>

## Decision Outcome

Chosen option: "<Option A>", because <the decisive reason in one sentence>.

<!-- Optional: summarize the rejected options and why they lost:

* Option B was rejected because ...
* Option C was rejected because ...
-->

### Consequences

<!-- What becomes easier or harder because of this decision. -->

- Good, because <positive effect>
- Good, because <positive effect>
- Bad, because <negative effect or trade-off>
- Bad, because <negative effect or trade-off>

## More Information

<!-- Optional: links to the source design doc, issue tracker, or related ADRs. -->

- Design doc: docs/designdoc/<kebab-case-title>.md
- Supersedes: NNNN (if applicable)
