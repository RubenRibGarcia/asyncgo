package asyncgo

import "github.com/RubenRibGarcia/asyncgo/spec"

// The following methods attach protocol-specific bindings to servers, channels,
// operations, and messages. Each protocol gets a typed helper (compile-checked)
// plus a generic Binding escape hatch for protocols not yet modeled.

// --- server bindings ---------------------------------------------------------

func (s *server) setBinding(proto string, v any) *server {
	if s.s.Bindings == nil {
		s.s.Bindings = spec.ServerBindings{}
	}
	s.s.Bindings[proto] = v
	return s
}

func (s *server) Kafka(b spec.KafkaServerBinding) *server {
	return s.setBinding(spec.ProtocolKafka, &b)
}

func (s *server) AMQP(
	b spec.AMQPServerBinding,
) *server {
	return s.setBinding(spec.ProtocolAMQP, &b)
}

func (s *server) NATS(
	b spec.NATSServerBinding,
) *server {
	return s.setBinding(spec.ProtocolNATS, &b)
}

func (s *server) MQTT(
	b spec.MQTTServerBinding,
) *server {
	return s.setBinding(spec.ProtocolMQTT, &b)
}

// Binding attaches a protocol-specific server binding under the given protocol key.
func (s *server) Binding(proto string, v any) *server { return s.setBinding(proto, v) }

// --- channel bindings --------------------------------------------------------

func (c *channel) setBinding(proto string, v any) *channel {
	if c.s.Bindings == nil {
		c.s.Bindings = spec.ChannelBindings{}
	}
	c.s.Bindings[proto] = v
	return c
}

func (c *channel) Kafka(b spec.KafkaChannelBinding) *channel {
	return c.setBinding(spec.ProtocolKafka, &b)
}

func (c *channel) AMQP(b spec.AMQPChannelBinding) *channel {
	return c.setBinding(spec.ProtocolAMQP, &b)
}

func (c *channel) NATS(b spec.NATSChannelBinding) *channel {
	return c.setBinding(spec.ProtocolNATS, &b)
}

func (c *channel) MQTT(b spec.MQTTChannelBinding) *channel {
	return c.setBinding(spec.ProtocolMQTT, &b)
}

// Binding attaches a protocol-specific channel binding under the given protocol key.
func (c *channel) Binding(proto string, v any) *channel { return c.setBinding(proto, v) }

// --- operation bindings ------------------------------------------------------

func (o *operation) setBinding(proto string, v any) *operation {
	if o.bindings == nil {
		o.bindings = spec.OperationBindings{}
	}
	o.bindings[proto] = v
	return o
}

func (o *operation) Kafka(b spec.KafkaOperationBinding) *operation {
	return o.setBinding(spec.ProtocolKafka, &b)
}

func (o *operation) AMQP(b spec.AMQPOperationBinding) *operation {
	return o.setBinding(spec.ProtocolAMQP, &b)
}

func (o *operation) NATS(b spec.NATSOperationBinding) *operation {
	return o.setBinding(spec.ProtocolNATS, &b)
}

func (o *operation) MQTT(b spec.MQTTOperationBinding) *operation {
	return o.setBinding(spec.ProtocolMQTT, &b)
}

// Binding attaches a protocol-specific operation binding under the given protocol key.
func (o *operation) Binding(proto string, v any) *operation { return o.setBinding(proto, v) }

// --- message bindings --------------------------------------------------------

func (m *message) setBinding(proto string, v any) *message {
	if m.bindings == nil {
		m.bindings = spec.MessageBindings{}
	}
	m.bindings[proto] = v
	return m
}

func (m *message) Kafka(b spec.KafkaMessageBinding) *message {
	return m.setBinding(spec.ProtocolKafka, &b)
}

func (m *message) AMQP(b spec.AMQPMessageBinding) *message {
	return m.setBinding(spec.ProtocolAMQP, &b)
}

func (m *message) NATS(b spec.NATSMessageBinding) *message {
	return m.setBinding(spec.ProtocolNATS, &b)
}

func (m *message) MQTT(b spec.MQTTMessageBinding) *message {
	return m.setBinding(spec.ProtocolMQTT, &b)
}

// Binding attaches a protocol-specific message binding under the given protocol key.
func (m *message) Binding(proto string, v any) *message { return m.setBinding(proto, v) }

// --- operation trait bindings --------------------------------------------------

func (t *operationTrait) setBinding(proto string, v any) *operationTrait {
	if t.t.Bindings == nil {
		t.t.Bindings = spec.OperationBindings{}
	}
	t.t.Bindings[proto] = v
	return t
}

func (t *operationTrait) Kafka(b spec.KafkaOperationBinding) *operationTrait {
	return t.setBinding(spec.ProtocolKafka, &b)
}

func (t *operationTrait) AMQP(b spec.AMQPOperationBinding) *operationTrait {
	return t.setBinding(spec.ProtocolAMQP, &b)
}

func (t *operationTrait) NATS(b spec.NATSOperationBinding) *operationTrait {
	return t.setBinding(spec.ProtocolNATS, &b)
}

func (t *operationTrait) MQTT(b spec.MQTTOperationBinding) *operationTrait {
	return t.setBinding(spec.ProtocolMQTT, &b)
}

// Binding attaches a protocol-specific operation trait binding under the given
// protocol key.
func (t *operationTrait) Binding(proto string, v any) *operationTrait {
	return t.setBinding(proto, v)
}

// --- message trait bindings ----------------------------------------------------

func (t *messageTrait) setBinding(proto string, v any) *messageTrait {
	if t.t.Bindings == nil {
		t.t.Bindings = spec.MessageBindings{}
	}
	t.t.Bindings[proto] = v
	return t
}

func (t *messageTrait) Kafka(b spec.KafkaMessageBinding) *messageTrait {
	return t.setBinding(spec.ProtocolKafka, &b)
}

func (t *messageTrait) AMQP(b spec.AMQPMessageBinding) *messageTrait {
	return t.setBinding(spec.ProtocolAMQP, &b)
}

func (t *messageTrait) NATS(b spec.NATSMessageBinding) *messageTrait {
	return t.setBinding(spec.ProtocolNATS, &b)
}

func (t *messageTrait) MQTT(b spec.MQTTMessageBinding) *messageTrait {
	return t.setBinding(spec.ProtocolMQTT, &b)
}

// Binding attaches a protocol-specific message trait binding under the given
// protocol key.
func (t *messageTrait) Binding(proto string, v any) *messageTrait {
	return t.setBinding(proto, v)
}
