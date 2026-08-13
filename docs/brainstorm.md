# asyncgo — Brainstorm & Initial Decisions

> Status: **Decisions locked** — proceeding to package/API sketch.
> Date: (session)

## The idea

`asyncgo` is a Go library that **generates an AsyncAPI specification document
(v3.1.0)** for a given Go program. It answers the under-served *code → spec*
direction — existing Go tooling (`asyncapi-codegen`, `go-asyncapi`, the official
AsyncAPI Generator) is almost entirely *spec → code*.

Reference: <https://www.asyncapi.com/docs/reference/specification/v3.1.0>

## The two candidate approaches

### Approach 1 — annotation / comment-driven

Developer writes structured comments (markers) near producer/consumer code; the
generator parses those markers plus the Go structs to build the document.

**Pros**

- Non-invasive — keeps using `sarama`, `confluent-kafka-go`, `amqp091-go`, etc.
  directly; zero migration of existing code.
- Co-located with usage, so more likely to be maintained.
- Fits Go culture (go-swagger, kubebuilder markers); AST tooling exists.
- Incremental adoption — annotate one topic today, grow later.
- No loss of tech power — every Kafka/RabbitMQ feature stays available.

**Cons**

- Invented DSL with no compiler help; a marker typo silently drops fields.
- The publish/subscribe call is an *expression in a function body*, not a
  *declaration* — markers attach cleanly only to declarations, so they end up on
  an invented wrapper function one step removed from the real call.
- Silent staleness — change the topic string in the body and the comment doesn't
  move; `go vet` won't notice.
- Discovery boundary ("the program") is fuzzy; linking a marker to a semantic
  fact ("this is a Kafka producer with consumer group X") is heuristic.
- Tech-specific bindings degrade into free-text marker fields with no validation.

### Approach 2 — abstraction-driven (a transport wrapper)

Developer routes messaging through library interfaces, so the library inherently
knows the topology and can dump it.

**Pros**

- Compiler-checked and honest — the doc derives from what code *actually does*;
  no heuristics, no drift.
- Captures the topology half beautifully (send vs receive, content type,
  serialization, correlation IDs, headers).
- Natural home for per-protocol bindings (typed, validated).
- Central registry → trivial doc generation, no AST scanning.
- Bonus value beyond docs: retries, observability, error wrapping, DLQs.

**Cons**

- Adoption cost / lock-in — existing code must be rewritten to use the wrapper.
- Abstraction over messaging is genuinely hard: Kafka (partitions, offsets,
  consumer groups, ordering) and RabbitMQ (exchanges, queues, routing keys, QoS)
  don't map to one clean interface without losing power.
- Escape hatches break the guarantee — drop to the raw SDK "just this once" and
  that usage silently vanishes from the docs.
- Permanent maintenance liability — N transport wrappers through upstream churn.

## Key insight — an AsyncAPI doc has two halves of very different nature

1. **Data contracts** (`components.schemas`, `message.payload`) — pure data
   shapes. A Go `struct` is already ~90% of this; reflection derives it trivially.
2. **Topology + protocol semantics** (`channels`, `operations`, `servers`,
   *bindings*) — runtime behavior, **not visible in any struct**. This half is
   where the two approaches actually diverge, and where comments are weakest and
   abstraction is strongest.

## Prior art

| Approach | Precedent | What it proves |
| --- | --- | --- |
| 1 — annotations | **go-swagger** `swagger:model`/`swagger:route` | Comments work well for *data contracts*, weaker for *operations* |
| 2 — abstraction | **Watermill**, **gocloud.dev/pubsub** | Pub/sub abstraction works; AsyncAPI Generator even ships a Watermill template (but *spec → code*, the inverse of our goal) |

## Recommendation — hybrid, biased toward non-invasiveness

Split along the two halves:

1. **Schemas → pure reflection.** `struct` → `components.schemas`, zero markers
   required. Optional tags only for what reflection can't know (required vs
   optional, descriptions, enums, examples).
2. **Topology → a thin, *declarative* registry** (not a runtime transport
   wrapper). The developer *declares* channels/operations once in a typed,
   compiler-checked way, and the doc references real structs for payloads. This
   keeps Approach 1's non-invasiveness while fixing its worst failure mode
   (magic-string markers that can't be validated).
3. **A linter, not just a generator.** Any doc generator can also be a doc
   linter — `asyncgo check` re-derives and diffs against the committed artifact,
   failing CI on drift. This is the real fix for staleness.

Approach 2 is deferred: offer a convenience transport wrapper later only if
adoption demands it, and steal Watermill's interface shape rather than inventing
a new one.

## Decisions (locked)

1. **Artifact** — the AsyncAPI doc is a **committed artifact** (produced by a
   `go generate` / CLI run, checked into the repo).
2. **Transport scope** — **multi-transport** from the start (Kafka, RabbitMQ/AMQP,
   NATS, …). Approach: hybrid.
3. **Fundamental nature** — asyncgo is **a documentation generator**, not a
   messaging framework.
4. **Discovery boundary** — everything **reachable from `main`** (load the main
   package and its dependency graph; union catalogs found along the way).

## Next steps

- Sketch concrete package layout + API shape (see `docs/design.md`).
- Resolve open sub-decisions: schema naming/collisions, required-vs-optional
  default, `$ref` granularity, pure-Go catalog vs YAML-fragment merge.
