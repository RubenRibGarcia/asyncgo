package asyncgo

import "github.com/RubenRibGarcia/asyncgo/spec"

// --- Solace bindings ---------------------------------------------------------

// Solace attaches a Solace server binding.
func (s *server) Solace(b spec.SolaceServerBinding) *server {
	return s.setBinding(spec.ProtocolSolace, &b)
}

// Solace attaches a Solace channel binding.
func (c *channel) Solace(b spec.SolaceChannelBinding) *channel {
	return c.setBinding(spec.ProtocolSolace, &b)
}

// Solace attaches a Solace operation binding.
func (o *operation) Solace(b spec.SolaceOperationBinding) *operation {
	return o.setBinding(spec.ProtocolSolace, &b)
}

// Solace attaches a Solace message binding.
func (m *message) Solace(b spec.SolaceMessageBinding) *message {
	return m.setBinding(spec.ProtocolSolace, &b)
}

// Solace attaches a Solace operation trait binding.
func (t *operationTrait) Solace(b spec.SolaceOperationBinding) *operationTrait {
	return t.setBinding(spec.ProtocolSolace, &b)
}

// Solace attaches a Solace message trait binding.
func (t *messageTrait) Solace(b spec.SolaceMessageBinding) *messageTrait {
	return t.setBinding(spec.ProtocolSolace, &b)
}
