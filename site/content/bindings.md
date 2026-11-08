# Protocol bindings

Every protocol AsyncAPI 3.1.0 defines has a typed helper on the server, channel,
operation, and message builders — `Kafka`, `AMQP`, `NATS`, `MQTT`, `HTTP`, `WS`,
`AMQP1`, `MQTT5`, `AnypointMQ`, `JMS`, `SNS`, `Solace`, `SQS`, `Stomp`, `Redis`,
`Mercure`, `IBMMQ`, `GooglePubSub`, `Pulsar`, `ROS2` — plus the same helper on the
matching trait builder.

Each takes the protocol's binding struct, so a misspelled field is a compile error
instead of a document the validator rejects:

```go
asyncgo.Channel("order-placed").
	Kafka(spec.KafkaChannelBinding{Topic: "order-placed", Partitions: 3}).
	Send(asyncgo.Operation().
		Kafka(spec.KafkaOperationBinding{GroupID: &spec.Schema{Type: "string"}}).
		Message(asyncgo.MessageOf(OrderPlaced{}).
			Kafka(spec.KafkaMessageBinding{Key: &spec.Schema{Type: "string"}})))
```

[`spec.Protocols()`](https://pkg.go.dev/github.com/RubenRibGarcia/asyncgo/spec#Protocols)
returns all 20 keys as a fresh slice.

## Where a helper exists

A helper exists only where the specification defines the binding object:

| Protocol | Missing binding levels |
| --- | --- |
| Pulsar | operation, message |
| ROS 2 | channel, message |
| MQTT 5 | all but server |

Objects whose field set is reserved (`stomp`, `redis`, `amqp1`, and the empty level
of several other protocols) are `spec.ReservedBinding` aliases carrying
`bindingVersion` only.

## Union-typed fields

Some fields are `any` because the specification types them as unions:

- `Schema Object | Reference Object` — Kafka `key`, MQTT `correlationData`, WS
  `query`/`headers`, …
- `scalar | Schema Object | Reference Object` — MQTT
  `sessionExpiryInterval`/`responseTopic`, …

## Validator gaps

`mqtt5` and `mercure` are modelled, but they are absent from the published 3.1.0
binding JSON Schemas, so a document that sets them fails `asyncapi validate`.

For a protocol added to the specification later, the untyped escape hatch stays
available on all six builders: `Binding(proto string, v any)`.
