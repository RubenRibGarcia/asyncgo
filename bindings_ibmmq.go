package asyncgo

import "github.com/RubenRibGarcia/asyncgo/spec"

// --- IBM MQ bindings ---------------------------------------------------------

// IBMMQ attaches an IBM MQ server binding.
func (s *server) IBMMQ(b spec.IBMMQServerBinding) *server {
	return s.setBinding(spec.ProtocolIBMMQ, &b)
}

// IBMMQ attaches an IBM MQ channel binding.
func (c *channel) IBMMQ(b spec.IBMMQChannelBinding) *channel {
	return c.setBinding(spec.ProtocolIBMMQ, &b)
}

// IBMMQ attaches an IBM MQ operation binding.
func (o *operation) IBMMQ(b spec.IBMMQOperationBinding) *operation {
	return o.setBinding(spec.ProtocolIBMMQ, &b)
}

// IBMMQ attaches an IBM MQ message binding.
func (m *message) IBMMQ(b spec.IBMMQMessageBinding) *message {
	return m.setBinding(spec.ProtocolIBMMQ, &b)
}

// IBMMQ attaches an IBM MQ operation trait binding.
func (t *operationTrait) IBMMQ(b spec.IBMMQOperationBinding) *operationTrait {
	return t.setBinding(spec.ProtocolIBMMQ, &b)
}

// IBMMQ attaches an IBM MQ message trait binding.
func (t *messageTrait) IBMMQ(b spec.IBMMQMessageBinding) *messageTrait {
	return t.setBinding(spec.ProtocolIBMMQ, &b)
}
