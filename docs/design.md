# asyncgo — Design Sketch (Hybrid Approach)

> Status: **Approved.** All decisions below are locked.
> Companion: `docs/brainstorm.md` (the rationale that led here).

## Locked decisions

### High-level (from brainstorm)

1. **Artifact** — the AsyncAPI doc is a **committed artifact** (produced by a
   `go generate` / CLI run, checked into the repo).
2. **Transport scope** — **multi-transport** from day one (Kafka, RabbitMQ/AMQP,
   NATS, …), delivered via the hybrid approach.
3. **Fundamental nature** — asyncgo is a **documentation generator**, not a
   messaging framework.
4. **Discovery boundary** — everything **reachable from `main`** (load the main
   package and its dependency graph; union catalogs found along the way).

### Sub-decisions (locked)

| # | Decision | Resolution |
| --- | --- | --- |
| 1 | Schema naming | **Fully-qualified names** — `pkgPath.TypeName` (e.g. `orders.OrderPlaced`) as the `components.schemas` key; `$ref` escapes `/`→`~1`, `~`→`~0` per JSON Pointer |
| 2 | Required semantics | **All fields optional by default**; `asyncgo:"required"` opts a field in |
| 3 | `$ref` granularity | **Always hoist** named struct types to `components.schemas` and reference them; only anonymous/inline types are inlined |
| 4 | Topology source | **Support both** a pure-Go catalog *and* YAML-fragment merge |

## Package layout

```
asyncgo/                            # module: github.com/RubenRibGarcia/asyncgo (root = package asyncgo)
├── doc.go                         # public fluent DSL: Spec(), Info(), Server(), Channel(), Operation()
├── message.go                     #   MessageOf(T{})
├── bindings.go                    #   Kafka(...), AMQP(...), NATS(...), MQTT(...), Binding(...) helpers
├── spec/                          # Typed AsyncAPI v3.1.0 object model + codecs
│   ├── spec.go                    #   AsyncAPI, Info, Server, Channel, Operation, Message
│   ├── schema.go                  #   JSON Schema types + $defs/$ref
│   ├── bindings.go                #   Kafka, AMQP, NATS, MQTT bindings (+ extensible *Bindings maps)
│   ├── merge.go                   #   Overlay (deep-merge a YAML fragment over a document)
│   └── encode.go                  #   YAML/JSON marshal
├── schema/                        # struct -> JSON Schema (the "data contract" half)
│   ├── derive.go                  #   FromType(reflect.Type) -> spec.Schema
│   └── tags.go                    #   asyncgo struct-tag parsing
├── cmd/asyncgo/
│   └── main.go                    # CLI: asyncgo generate | check
└── internal/discovery/            # find + materialize catalogs reachable from main
    ├── discover.go                #   go/packages: locate *spec.AsyncAPI vars reachable from main
    ├── materialize.go             #   run a generated harness to materialize catalogs
    └── merge.go                   #   merge multiple catalogs into one document
```

Three layers, three different difficulty classes:

1. **`spec/`** — commodity work: plain data model + marshal. No opinions, no magic.
2. **`schema/`** — the reflection half (Approach 1's strong suit):
   `struct → JSON Schema`.
3. **root `asyncgo` package + `internal/discovery`** — the topology half (Approach 1's
   weak suit), solved with a *typed declarative catalog* instead of magic-string
   comments.

## Developer-facing API — exactly two touchpoints

### 1. The struct (data contract) — reflection, optional tags

```go
type OrderPlaced struct {
    OrderID string  `json:"order_id" asyncgo:"required,description=Unique order id"`
    Amount  float64 `json:"amount"   asyncgo:"required"`
    Note    string  `json:"note,omitempty"` // optional (omitempty => not required)
}
```

Nothing else needed. `json` tags drive field names; `asyncgo` tags add only what
reflection can't see (`required`, `description`, `enum`, `example`, `format`).

### 2. The catalog (topology) — typed, declarative, one per package or app

```go
// catalog.go  (any package reachable from main)
package orders

import "github.com/RubenRibGarcia/asyncgo"

var Catalog = asyncgo.Spec(
    asyncgo.Info("Orders Service", "1.0.0").
        Description("Order lifecycle events"),

    asyncgo.DefaultContentType("application/json"),

    asyncgo.Servers(
        asyncgo.Server("prod", "kafka").Host("broker:9092"),
    ),

    asyncgo.Channels(
        asyncgo.Channel("order-placed").
            Description("Emitted when an order is placed").
            Send(asyncgo.Operation().
                Message(asyncgo.MessageOf(OrderPlaced{}).
                    Name("OrderPlaced"))).
            Kafka(asyncgo.Kafka{Topic: "order-placed", PartitionKey: "orderID"}),

        asyncgo.Channel("order-cancelled").
            Receive(asyncgo.Operation().
                Message(asyncgo.MessageOf(OrderCancelled{}))).
            AMQP(asyncgo.AMQP{Exchange: "orders", RoutingKey: "cancelled"}),
    ),
)
```

What this buys over both pure approaches:

- **`Send` / `Receive`** → `operations[].action`, a typed enum — no magic strings
  for direction.
- **`Kafka{...}` / `AMQP{...}`** → per-protocol bindings, **typed and
  compiler-checked** — multi-transport satisfied *without a runtime wrapper*.
- **`MessageOf(OrderPlaced{})`** → the catalog *references* the real struct
  rather than duplicating the payload shape. Schemas stay single-sourced from the
  struct; the catalog cannot drift from the data contract.
- The catalog is **not** routing actual messaging — it's a declaration.
  Non-invasive like Approach 1, typed like Approach 2.

## CLI / workflow

```bash
go generate ./...            # //go:generate asyncgo generate
asyncgo generate             # walk main + deps -> write asyncapi.yaml (committed)
asyncgo check                # re-derive + diff vs committed; non-zero on drift (CI gate)
```

`asyncgo check` is the drift guard: re-derive from code and diff against the
committed `asyncapi.yaml`; non-zero exit fails CI.

## Key mechanism — static discovery + harness materialization

Two stages:

1. **Discovery (static, no execution).** `internal/discovery` uses
   `golang.org/x/tools/go/packages` (`NeedSyntax | NeedTypes | NeedTypesInfo`) to
   load the module's packages, walks imports from each `main` package to compute
   the reachable set, and locates exported package-level vars of type
   `*spec.AsyncAPI`.
2. **Materialization (runs a generated harness).** A schema cannot be derived
   from `go/types` alone — deriving the JSON Schema needs a real `reflect.Type`
   for the user's structs, which only exists once their code is compiled. So the
   generator emits a small `main` that imports the catalog packages and prints
   each catalog value as JSON, then runs it with `go run`.

   This executes only the catalog packages' `init` functions — **never the
   `main` package** (which is not importable). The catalog values themselves are
   already-built DSL results, so nothing beyond `init` runs. This is the
   pragmatic tradeoff versus a full AST interpreter: the latter would require a
   second schema engine over `go/types`, duplicating `schema/`.

## Build order / roadmap

1. `spec/` — object model + bindings + codecs (commodity, tested first).
2. `schema/` — struct → JSON Schema reflection (sub-decisions #1–3).
3. `asyncgo/` — fluent DSL.
4. `cmd/asyncgo` + `internal/discovery` — generator + `check` (sub-decision #4).
5. Example program + `go generate` wiring + end-to-end golden test.

> Status: **implemented** — all five steps are done as of this revision.
