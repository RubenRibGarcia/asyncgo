package asyncgo

import "github.com/RubenRibGarcia/asyncgo/spec"

// --- Mercure bindings --------------------------------------------------------

// Mercure attaches a Mercure server binding.
func (s *server) Mercure(b spec.MercureServerBinding) *server {
	return s.setBinding(spec.ProtocolMercure, &b)
}

// Mercure attaches a Mercure channel binding.
func (c *channel) Mercure(b spec.MercureChannelBinding) *channel {
	return c.setBinding(spec.ProtocolMercure, &b)
}

// Mercure attaches a Mercure operation binding.
func (o *operation) Mercure(b spec.MercureOperationBinding) *operation {
	return o.setBinding(spec.ProtocolMercure, &b)
}

// Mercure attaches a Mercure message binding.
func (m *message) Mercure(b spec.MercureMessageBinding) *message {
	return m.setBinding(spec.ProtocolMercure, &b)
}

// Mercure attaches a Mercure operation trait binding.
func (t *operationTrait) Mercure(b spec.MercureOperationBinding) *operationTrait {
	return t.setBinding(spec.ProtocolMercure, &b)
}

// Mercure attaches a Mercure message trait binding.
func (t *messageTrait) Mercure(b spec.MercureMessageBinding) *messageTrait {
	return t.setBinding(spec.ProtocolMercure, &b)
}
