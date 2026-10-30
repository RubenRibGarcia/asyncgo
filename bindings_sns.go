package asyncgo

import "github.com/RubenRibGarcia/asyncgo/spec"

// --- SNS bindings ------------------------------------------------------------

// SNS attaches an SNS server binding.
func (s *server) SNS(b spec.SNSServerBinding) *server {
	return s.setBinding(spec.ProtocolSNS, &b)
}

// SNS attaches an SNS channel binding.
func (c *channel) SNS(b spec.SNSChannelBinding) *channel {
	return c.setBinding(spec.ProtocolSNS, &b)
}

// SNS attaches an SNS operation binding.
func (o *operation) SNS(b spec.SNSOperationBinding) *operation {
	return o.setBinding(spec.ProtocolSNS, &b)
}

// SNS attaches an SNS message binding.
func (m *message) SNS(b spec.SNSMessageBinding) *message {
	return m.setBinding(spec.ProtocolSNS, &b)
}

// SNS attaches an SNS operation trait binding.
func (t *operationTrait) SNS(b spec.SNSOperationBinding) *operationTrait {
	return t.setBinding(spec.ProtocolSNS, &b)
}

// SNS attaches an SNS message trait binding.
func (t *messageTrait) SNS(b spec.SNSMessageBinding) *messageTrait {
	return t.setBinding(spec.ProtocolSNS, &b)
}
