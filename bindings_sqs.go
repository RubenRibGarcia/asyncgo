package asyncgo

import "github.com/RubenRibGarcia/asyncgo/spec"

// --- SQS bindings ------------------------------------------------------------

// SQS attaches an SQS server binding.
func (s *server) SQS(b spec.SQSServerBinding) *server {
	return s.setBinding(spec.ProtocolSQS, &b)
}

// SQS attaches an SQS channel binding.
func (c *channel) SQS(b spec.SQSChannelBinding) *channel {
	return c.setBinding(spec.ProtocolSQS, &b)
}

// SQS attaches an SQS operation binding.
func (o *operation) SQS(b spec.SQSOperationBinding) *operation {
	return o.setBinding(spec.ProtocolSQS, &b)
}

// SQS attaches an SQS message binding.
func (m *message) SQS(b spec.SQSMessageBinding) *message {
	return m.setBinding(spec.ProtocolSQS, &b)
}

// SQS attaches an SQS operation trait binding.
func (t *operationTrait) SQS(b spec.SQSOperationBinding) *operationTrait {
	return t.setBinding(spec.ProtocolSQS, &b)
}

// SQS attaches an SQS message trait binding.
func (t *messageTrait) SQS(b spec.SQSMessageBinding) *messageTrait {
	return t.setBinding(spec.ProtocolSQS, &b)
}
