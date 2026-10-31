// Package unlisted is a discovery fixture for the two protocol keys the
// AsyncAPI specification text defines but the published 3.1.0 binding JSON
// Schemas omit: mqtt5 and mercure. The typed bindings round-trip through the
// committed asyncapi.yaml, but `asyncapi validate` rejects the document, so the
// integration test asserts the rejection instead of the acceptance.
package unlisted

import (
	"github.com/RubenRibGarcia/asyncgo"
	"github.com/RubenRibGarcia/asyncgo/spec"
)

// Event is the payload the message carries; the bindings are what this fixture
// exercises.
type Event struct {
	ID string `json:"id" asyncapi:"required"`
}

var (
	mqtt5Server = asyncgo.Server("mqtt5", "mqtt5", "broker.mqtt:1883").
			MQTT5(spec.MQTT5ServerBinding{SessionExpiryInterval: 30, BindingVersion: "0.2.0"})
	mercureServer = asyncgo.Server("mercure", "mercure", "hub.example.com").
			Mercure(spec.MercureServerBinding{BindingVersion: "0.1.0"})
)

// Catalog is the AsyncAPI description used to generate asyncapi.yaml.
var Catalog = asyncgo.Spec(
	asyncgo.Info("Unlisted bindings", "1.0.0").
		Description("mqtt5 and mercure, which the 3.1.0 binding schemas omit"),

	asyncgo.DefaultContentType("application/json"),

	asyncgo.Servers(mqtt5Server, mercureServer),

	asyncgo.Channels(
		asyncgo.Channel("orders-mqtt5").
			Servers(mqtt5Server).
			Send(asyncgo.Operation().
				Message(asyncgo.MessageOf(Event{}).Name("MQTT5Event"))),

		asyncgo.Channel("orders-mercure").
			Servers(mercureServer).
			Mercure(spec.MercureChannelBinding{BindingVersion: "0.1.0"}).
			Send(asyncgo.Operation().
				Mercure(spec.MercureOperationBinding{BindingVersion: "0.1.0"}).
				Message(asyncgo.MessageOf(Event{}).
					Name("MercureEvent").
					Mercure(spec.MercureMessageBinding{BindingVersion: "0.1.0"}))),
	),
)
