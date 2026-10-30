package asyncgo

import "github.com/RubenRibGarcia/asyncgo/spec"

// --- JMS bindings ------------------------------------------------------------

// JMS attaches a JMS server binding.
func (s *server) JMS(b spec.JMSServerBinding) *server {
	return s.setBinding(spec.ProtocolJMS, &b)
}

// JMS attaches a JMS channel binding.
func (c *channel) JMS(b spec.JMSChannelBinding) *channel {
	return c.setBinding(spec.ProtocolJMS, &b)
}

// JMS attaches a JMS operation binding.
func (o *operation) JMS(b spec.JMSOperationBinding) *operation {
	return o.setBinding(spec.ProtocolJMS, &b)
}

// JMS attaches a JMS message binding.
func (m *message) JMS(b spec.JMSMessageBinding) *message {
	return m.setBinding(spec.ProtocolJMS, &b)
}

// JMS attaches a JMS operation trait binding.
func (t *operationTrait) JMS(b spec.JMSOperationBinding) *operationTrait {
	return t.setBinding(spec.ProtocolJMS, &b)
}

// JMS attaches a JMS message trait binding.
func (t *messageTrait) JMS(b spec.JMSMessageBinding) *messageTrait {
	return t.setBinding(spec.ProtocolJMS, &b)
}
