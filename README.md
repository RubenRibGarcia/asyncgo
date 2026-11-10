# asyncgo

Generate an [AsyncAPI 3.1.0](https://www.asyncapi.com/docs/reference/specification/v3.1.0)
specification document from Go code — the *code → spec* direction that most Go
AsyncAPI tooling (which goes *spec → code*) leaves unserved.

`asyncgo` is a **documentation generator**, not a messaging framework. It does not
route your actual messaging; it derives a committed AsyncAPI document
(`asyncapi.yaml`, or `asyncapi.json` with `generate --format json`) from two
touchpoints in your code: a struct that is the data contract, and a typed catalog
that is the topology.

**📖 Documentation: <https://rubenribgarcia.github.io/asyncgo/>**

## Quickstart

```bash
go install github.com/RubenRibGarcia/asyncgo/cmd/asyncgo@latest
```

Declare a payload struct and a catalog — a package-level variable in your
module:

```go
package orders

import "github.com/RubenRibGarcia/asyncgo"

type OrderPlaced struct {
	OrderID string  `json:"order_id" asyncapi:"required"`
	Amount  float64 `json:"amount"   asyncapi:"required"`
	// Optional note from the customer
	Note string `json:"note"`
}

// Catalog is the AsyncAPI description of the orders service.
var Catalog = asyncgo.Spec(
	asyncgo.Info("Orders Service", "1.0.0"),
	asyncgo.Servers(asyncgo.Server("prod", "kafka", "broker.example.com:9092")),
	asyncgo.Channels(
		asyncgo.Channel("order-placed").
			Send(asyncgo.Operation().
				Message(asyncgo.MessageOf(OrderPlaced{}).Name("OrderPlaced"))),
	),
)
```

Then write the document, and commit it:

```bash
asyncgo generate .          # writes asyncapi.yaml
asyncgo check .             # exit 3 when the committed document is stale
```

## Documentation

The full documentation lives on the project site, page by page:

| | |
| --- | --- |
| [Getting started](https://rubenribgarcia.github.io/asyncgo/getting-started/) | Install, declare a catalog, generate, wire `check` into CI |
| [Concepts](https://rubenribgarcia.github.io/asyncgo/concepts/) | The two touchpoints, message identity, parameters, servers |
| [Schema derivation](https://rubenribgarcia.github.io/asyncgo/schema/) | Struct → schema rules, `allOf`/`oneOf`/`anyOf` composition |
| [Custom types](https://rubenribgarcia.github.io/asyncgo/custom-types/) | `SchemaProvider`, declared mappings, built-in third-party entries |
| [Protocol bindings](https://rubenribgarcia.github.io/asyncgo/bindings/) | The typed binding helper for each of the 20 protocols |
| [Features](https://rubenribgarcia.github.io/asyncgo/features/) | Security, request/reply, traits, multi-format (Avro, Protobuf) |
| [CLI](https://rubenribgarcia.github.io/asyncgo/cli/) | Flags, exit codes, verbose logging, configuration |
| [Reference](https://rubenribgarcia.github.io/asyncgo/reference/) | Curated index of the public API, with pkg.go.dev links |
| [Project](https://rubenribgarcia.github.io/asyncgo/project/) | ADRs, design docs, and the AsyncAPI 3.1.0 coverage assessment |

API reference: <https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo>

## Contributing

The repository keeps its reasoning in-tree: significant changes follow a
design-doc → ADR workflow described in
[`AGENT.md`](AGENT.md) and published at
<https://rubenribgarcia.github.io/asyncgo/project/>. Repository layout, public-API
map, testing and coverage rules, and commit conventions are all in `AGENT.md`.

## License

MIT — see [LICENSE](LICENSE).
