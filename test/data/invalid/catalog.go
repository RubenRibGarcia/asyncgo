// Package invalid is a discovery fixture that intentionally declares a catalog
// with a server missing its required host, so the generator exercises the
// validation error path end to end.
package invalid

import "github.com/RubenRibGarcia/asyncgo"

// Catalog is an invalid catalog: the server has an empty host, which the
// AsyncAPI 3.1.0 specification requires.
var Catalog = asyncgo.Spec(
	asyncgo.Info("Orders Service", "1.0.0"),
	asyncgo.Servers(
		asyncgo.Server("prod", "kafka", ""),
	),
)
