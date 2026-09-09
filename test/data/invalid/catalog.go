// Package invalid is a discovery fixture that intentionally declares a catalog
// with two validation violations, so the generator exercises the validation
// error path end to end:
//
//   - a server missing its required host
//   - a channel referencing a server never declared via Servers(...)
package invalid

import "github.com/RubenRibGarcia/asyncgo"

// Catalog is an invalid catalog: the server has an empty host (required by the
// AsyncAPI 3.1.0 specification) and the channel references an undeclared
// server, producing a dangling $ref.
var Catalog = asyncgo.Spec(
	asyncgo.Info("Orders Service", "1.0.0"),
	asyncgo.Servers(
		asyncgo.Server("prod", "kafka", ""),
	),
	asyncgo.Channels(
		asyncgo.Channel("order-placed").
			Servers(asyncgo.Server("staging", "kafka", "broker-staging:9092")),
	),
)
