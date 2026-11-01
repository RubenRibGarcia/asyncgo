// Package invalid is a discovery fixture that intentionally declares a catalog
// with one violation per validation rule, so the generator exercises the
// validation error path end to end:
//
//   - a server missing its required host
//   - a channel referencing a server never declared via Servers(...)
//   - a security scheme name outside the components key pattern
//   - a channel address expression with no declared parameter
//   - a second Send on a channel, whose map key would silently overwrite
//   - an operation carrying a message the channel does not have
package invalid

import (
	"github.com/RubenRibGarcia/asyncgo"
	"github.com/RubenRibGarcia/asyncgo/spec"
)

// OrderPlaced is the payload of the fixture messages.
type OrderPlaced struct {
	Amount float64
}

// Catalog is an invalid catalog. The server has an empty host and the
// order-placed channel references an undeclared server, producing a dangling
// $ref; the remaining channels and components trip the key-pattern, address
// expression, duplicate-action and operation-message rules.
//
// The last two need a comment each: "orders" repeats an action on one channel,
// and "orders-shipped" is declared twice with different actions, so the second
// declaration's message is written into the channel struct the first
// declaration displaced.
var Catalog = asyncgo.Spec(
	asyncgo.Info("Orders Service", "1.0.0"),
	asyncgo.Servers(
		asyncgo.Server("prod", "kafka", ""),
	),
	asyncgo.SecuritySchemes(
		asyncgo.SecurityScheme("oauth basic", spec.SecurityScheme{Type: "http"}),
	),
	asyncgo.Channels(
		asyncgo.Channel("order-placed").
			Servers(asyncgo.Server("staging", "kafka", "broker-staging:9092")),
		asyncgo.Channel("orders/{orderId}"),
		asyncgo.Channel("orders").
			Send(asyncgo.Operation().Message(
				asyncgo.MessageOf(OrderPlaced{}).Name("OrderPlaced"),
			)).
			Send(asyncgo.Operation().Message(
				asyncgo.MessageOf(OrderPlaced{}).Name("OrderPlaced"),
			)),
		asyncgo.Channel("orders-shipped").Send(asyncgo.Operation().Message(
			asyncgo.MessageOf(OrderPlaced{}).Name("OrderPlaced"),
		)),
		asyncgo.Channel("orders-shipped").Receive(asyncgo.Operation().Message(
			asyncgo.MessageOf(OrderPlaced{}).Name("OrderShipped"),
		)),
	),
)
