package asyncgo

import "github.com/RubenRibGarcia/asyncgo/spec"

// --- AMQP 0-9-1 bindings -----------------------------------------------------

// AMQP attaches an AMQP server binding.
func (s *server) AMQP(b spec.AMQPServerBinding) *server {
	return s.setBinding(spec.ProtocolAMQP, &b)
}

// AMQP attaches an AMQP channel binding.
func (c *channel) AMQP(b spec.AMQPChannelBinding) *channel {
	return c.setBinding(spec.ProtocolAMQP, &b)
}

// AMQP attaches an AMQP operation binding.
func (o *operation) AMQP(b spec.AMQPOperationBinding) *operation {
	return o.setBinding(spec.ProtocolAMQP, &b)
}

// AMQP attaches an AMQP message binding.
func (m *message) AMQP(b spec.AMQPMessageBinding) *message {
	return m.setBinding(spec.ProtocolAMQP, &b)
}

// AMQP attaches an AMQP operation trait binding.
func (t *operationTrait) AMQP(b spec.AMQPOperationBinding) *operationTrait {
	return t.setBinding(spec.ProtocolAMQP, &b)
}

// AMQP attaches an AMQP message trait binding.
func (t *messageTrait) AMQP(b spec.AMQPMessageBinding) *messageTrait {
	return t.setBinding(spec.ProtocolAMQP, &b)
}

// --- AMQP 1.0 bindings -------------------------------------------------------

// AMQP1 attaches an AMQP 1.0 server binding.
func (s *server) AMQP1(b spec.AMQP1ServerBinding) *server {
	return s.setBinding(spec.ProtocolAMQP1, &b)
}

// AMQP1 attaches an AMQP 1.0 channel binding.
func (c *channel) AMQP1(b spec.AMQP1ChannelBinding) *channel {
	return c.setBinding(spec.ProtocolAMQP1, &b)
}

// AMQP1 attaches an AMQP 1.0 operation binding.
func (o *operation) AMQP1(b spec.AMQP1OperationBinding) *operation {
	return o.setBinding(spec.ProtocolAMQP1, &b)
}

// AMQP1 attaches an AMQP 1.0 message binding.
func (m *message) AMQP1(b spec.AMQP1MessageBinding) *message {
	return m.setBinding(spec.ProtocolAMQP1, &b)
}

// AMQP1 attaches an AMQP 1.0 operation trait binding.
func (t *operationTrait) AMQP1(b spec.AMQP1OperationBinding) *operationTrait {
	return t.setBinding(spec.ProtocolAMQP1, &b)
}

// AMQP1 attaches an AMQP 1.0 message trait binding.
func (t *messageTrait) AMQP1(b spec.AMQP1MessageBinding) *messageTrait {
	return t.setBinding(spec.ProtocolAMQP1, &b)
}
