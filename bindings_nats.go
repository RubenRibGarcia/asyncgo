package asyncgo

import "github.com/RubenRibGarcia/asyncgo/spec"

// --- NATS bindings -----------------------------------------------------------

// NATS attaches a NATS server binding.
func (s *server) NATS(b spec.NATSServerBinding) *server {
	return s.setBinding(spec.ProtocolNATS, &b)
}

// NATS attaches a NATS channel binding.
func (c *channel) NATS(b spec.NATSChannelBinding) *channel {
	return c.setBinding(spec.ProtocolNATS, &b)
}

// NATS attaches a NATS operation binding.
func (o *operation) NATS(b spec.NATSOperationBinding) *operation {
	return o.setBinding(spec.ProtocolNATS, &b)
}

// NATS attaches a NATS message binding.
func (m *message) NATS(b spec.NATSMessageBinding) *message {
	return m.setBinding(spec.ProtocolNATS, &b)
}

// NATS attaches a NATS operation trait binding.
func (t *operationTrait) NATS(b spec.NATSOperationBinding) *operationTrait {
	return t.setBinding(spec.ProtocolNATS, &b)
}

// NATS attaches a NATS message trait binding.
func (t *messageTrait) NATS(b spec.NATSMessageBinding) *messageTrait {
	return t.setBinding(spec.ProtocolNATS, &b)
}
