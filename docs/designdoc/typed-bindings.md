# Design: Typed bindings for all AsyncAPI 3.1.0 protocols

- **Status**: Proposed
- **Created**: 2026-10-25
- **Status updated**: 2026-10-25
- **Scope**: `spec/`, root DSL package, `test/data/bindings*`, `test/integration`, `docs/`
- **Issue**: [#23](https://github.com/RubenRibGarcia/asyncgo/issues/23) (coverage backlog B12)

## Summary

Only `kafka`, `amqp`, `nats`, and `mqtt` have typed binding structs; the other protocol keys must be
hand-written `any` values through `Binding(proto string, v any)`. This design models every binding object
that AsyncAPI 3.1.0 defines, for every protocol, and gives each one a typed DSL helper — while keeping the
untyped escape hatch for protocols added later.

## Background

The binding objects are four protocol-keyed maps (`spec.ServerBindings` … `spec.MessageBindings`, each
`map[string]any`). Emitting a typed struct through a map is already how the four shipped protocols work, so
no codec change is needed: `encoding/json` and `goccy/go-yaml` marshal the struct behind the `any`.

Two authoritative sources disagree in places, and the disagreement decides the design:

1. **`github.com/asyncapi/bindings`** — one prose README per protocol, listing fixed fields as tables.
2. **`@asyncapi/specs` (6.11.1, bundled in the pinned `asyncapi/cli:6.1.0`)** — the machine form used by
   `asyncapi validate`: `definitions/3.1.0/{server,channel,operation,message}BindingsObject.json` enumerate
   the allowed protocol keys per level with `additionalProperties: false`, and
   `bindings/<proto>/<version>/<level>.json` give the field schemas.

**The schemas win**, for one reason: `test/integration` validates every generated document with the pinned
CLI, so a field set taken from a README that the schema does not know produces a fixture that cannot pass.
Evidence extracted from the image lives in `.tmp/specs/report.txt` (regenerate with
`.tmp/specs/extract.py`).

Facts that follow from the schemas and that a README-only design would get wrong:

- 18 protocol keys exist, not 20: `mqtt5` and `mercure` are absent from all four binding objects and
  `additionalProperties: false` makes them invalid.
- `pulsar` has no operation/message object; `ros2` has no channel/message object.
- `amqp` 0.3.0 has no `replyTo` (the model ships one today).
- `kafka.groupId`/`clientId` are Schema refs, not `Schema | Reference | boolean`.
- Required fields exist (`sns.channel.name`, `sqs.queue.name/fifoQueue`, `pulsar.channel.namespace`, …), so
  `omitempty` on those is a correctness bug, not a cosmetic one.

## Goals / Non-goals

**Goals**

1. Every binding object defined by a 3.1.0 protocol has a struct in `spec`, with the field set and
   JSON/YAML tag spelling from its JSON schema.
2. Every (protocol, level) pair the specification defines has a typed DSL helper on the corresponding core
   receiver, and — for operation and message — on the corresponding trait builder.
3. `spec.ProtocolKeys` lists all protocol keys the model supports, so coverage is assertable.
4. A committed golden fixture proves the typed bindings round-trip through YAML and JSON, and the pinned
   CLI validates the schema-known subset.

**Non-goals**

- Fixing the `omitempty` zero-value hazard for *optional* fields (B8 stays open); only schema-required fields
  drop `omitempty`.
- Modelling objects the 3.1.0 schemas reject (pulsar operation/message, ros2 channel/message).
- Validating binding content at catalog build time; the CLI remains the validator.

## Design decisions

### D1 — One file per package, one section per protocol

`spec/bindings.go` holds everything binding-related for the model: the protocol constants, `ProtocolKeys`, the
four map types, `ReservedBinding` with its aliases, and then one `// --- <protocol> bindings` section per
protocol with the structs that protocol defines. `bindings.go` in the root package mirrors it: the six
`setBinding` helpers, the six generic `Binding` methods, and one section per protocol with its DSL helpers.
Protocol sections are ordered alphabetically in both files.

**Rationale**: a protocol is a contiguous block, not an independent unit — every protocol PR is a few additions
at most. Forty stub files (structs and helpers) buy a per-protocol diff show at the cost of a 40-entry
directory, a package clause and import block each, and no way to see the API surface in one place. A single
file keeps the section order obvious and makes the whole binding surface reviewable in one scroll. The cost —
two protocol PRs conflicting in the same file — is accepted: the sections sit apart, so git resolves the hunks
routinely.

### D2 — `any` only where the schema is a union, `omitempty` only where the field is optional

A union field (`Schema | Reference`, `string | Schema | Reference`, `integer | Schema | Reference`) is `any`
with a doc comment, following `KafkaMessageBinding.Key`. A schema-required field carries no `omitempty`.

**Rationale**: `any` keeps the reference side expressible; dropping `omitempty` on required fields is what
makes the emitted document valid. Optional zero values are still dropped — that is B8's problem, not a
regression introduced here.

### D3 — Unconstrained objects share one type

Objects the schema declares as `{}` (amqp1/stomp/redis/mercure at all four levels, and the empty level of
nats, mqtt, http, ws, anypointmq, jms, sns, solace, sqs, ibmmq, googlepubsub) are modelled as
`spec.ReservedBinding{BindingVersion string}`, with a per-protocol type alias
(`type StompChannelBinding = ReservedBinding`) so DSL signatures stay protocol-specific.

**Rationale**: 40+ byte-identical structs add nothing; aliases keep the API readable and names greppable.

### D4 — Two fixtures, because validity and coverage disagree

`test/data/bindings` carries the 18 schema-known protocols and is validated by the CLI.
`test/data/bindings-unlisted` carries `mqtt5` + `mercure`, is registered in `cliUnsupported`, and the test
asserts the CLI *still rejects* it — the same discipline the `multiformat` entry uses. Both fixtures assert
byte-equality and JSON equivalence, so both prove the YAML/JSON round-trip.

**Rationale**: one combined fixture would either lose CLI validation for 18 protocols or have to pretend the
document is valid.

### D5 — The DSL surface follows the specification, not the library

A helper exists exactly where the specification defines the object: no `Pulsar` operation/message helper, no
`ROS2` channel/message helper. Trait helpers exist for operation and message levels only.

**Rationale**: a helper that can only produce an invalid document is worse than none, and the generic
`Binding(proto, v any)` still covers hand-written values.

## Detailed design

### `spec/bindings.go`

```go
const (
    ProtocolKafka         = "kafka"
    ProtocolAMQP          = "amqp"
    ProtocolNATS          = "nats"
    ProtocolMQTT          = "mqtt"
    ProtocolHTTP          = "http"
    ProtocolWS            = "ws"
    ProtocolAMQP1         = "amqp1"
    ProtocolMQTT5         = "mqtt5"     // not in the 3.1.0 binding schemas
    ProtocolAnypointMQ    = "anypointmq"
    ProtocolJMS           = "jms"
    ProtocolSNS           = "sns"
    ProtocolSolace        = "solace"
    ProtocolSQS           = "sqs"
    ProtocolStomp         = "stomp"
    ProtocolRedis         = "redis"
    ProtocolMercure       = "mercure"   // not in the 3.1.0 binding schemas
    ProtocolIBMMQ         = "ibmmq"
    ProtocolGooglePubSub  = "googlepubsub"
    ProtocolPulsar        = "pulsar"
    ProtocolROS2          = "ros2"
)

// ProtocolKeys lists every protocol key the model supports, in the order the
// specification presents them.
var ProtocolKeys = []string{ProtocolKafka, /* … */ ProtocolROS2}
```

### `spec/bindings.go` — one protocol section (shape of every protocol section)

```go
// --- SQS bindings (binding version 0.3.0) -----------------------------------

// SQSChannelBinding describes SQS-specific channel information.
type SQSChannelBinding struct {
    Queue          *SQSQueue `json:"queue"                       yaml:"queue"`
    DeadLetterQueue *SQSQueue `json:"deadLetterQueue,omitempty"  yaml:"deadLetterQueue,omitempty"`
    BindingVersion string    `json:"bindingVersion,omitempty"     yaml:"bindingVersion,omitempty"`
}
```

### `bindings.go` — one protocol section (shape of every DSL section)

```go
// --- SQS bindings ------------------------------------------------------------

func (c *channel) SQS(b spec.SQSChannelBinding) *channel {
    return c.setBinding(spec.ProtocolSQS, &b)
}
```

## Edge cases

- **Binding object reserved but key allowed** (nats server, ws operation) — `ReservedBinding`, helper still
  emitted; the CLI accepts `{}`/`bindingVersion` here because the schema is `{}`.
- **Key not in the 3.1.0 schema** (mqtt5, mercure) — modelled, documented, isolated in
  `test/data/bindings-unlisted`.
- **Required field with a zero value** — `omitempty` removed, so `fifoQueue: false` and `name: ""` serialize.
- **`solace.destinations`** — untyped upstream (`[]any`); modelled as `[]SolaceDestination` (a stricter
  shape the schema still accepts).
- **`kafka.topicConfiguration`** — was `map[string]any`; becomes `*KafkaTopicConfiguration`. Breaking for
  callers that passed maps, correct against the schema.

## Rollout plan

1. **Stage 0 — preflight** (done): schema extraction + CLI spike.
2. **Stage 1 — split** (`spec`, `dsl`): move the four shipped protocols into their files, add constants and
   `ProtocolKeys`. No behaviour change.
3. **Stage 2 — corrections** (`spec`): Kafka message fields, MQTT union fields + message fields, AMQP
   `replyTo` removal.
4. **Stage 3 — reserved** (`spec`): `ReservedBinding` + aliases.
5. **Stage 4 — remaining protocols** (`spec`, `dsl`), one protocol per commit.
6. **Stage 5 — tests + fixtures** (`test`): table tests, encode test, two fixtures.
7. **Stage 6 — docs** (`docs`): coverage doc, README, AGENT.md.

## Testing plan

- Table-driven `bindings_test.go`: one row per (protocol, level) per receiver; asserts key, stored type, and a
  non-zero field.
- `TestEncodeAllBindingProtocols`: every struct with all fields set, asserting exact JSON key spelling and
  YAML/JSON equivalence.
- `TestProtocolKeys`: 20 unique keys, each with a constant.
- `test/data/bindings` (18 protocols, CLI-validated) and `test/data/bindings-unlisted` (mqtt5 + mercure,
  `cliUnsupported`).

## Open / deferred

- B8 (`omitempty` drops optional zero values) — untouched, still tracked.
- `bindingVersion` enums per protocol are not validated by the library; the CLI catches them.
- The ADR (0007) is written once this design is accepted.
