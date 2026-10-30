package asyncgo

import "github.com/RubenRibGarcia/asyncgo/spec"

// --- STOMP bindings ----------------------------------------------------------

// Stomp attaches a STOMP server binding.
func (s *server) Stomp(b spec.StompServerBinding) *server {
	return s.setBinding(spec.ProtocolStomp, &b)
}

// Stomp attaches a STOMP channel binding.
func (c *channel) Stomp(b spec.StompChannelBinding) *channel {
	return c.setBinding(spec.ProtocolStomp, &b)
}

// Stomp attaches a STOMP operation binding.
func (o *operation) Stomp(b spec.StompOperationBinding) *operation {
	return o.setBinding(spec.ProtocolStomp, &b)
}

// Stomp attaches a STOMP message binding.
func (m *message) Stomp(b spec.StompMessageBinding) *message {
	return m.setBinding(spec.ProtocolStomp, &b)
}

// Stomp attaches a STOMP operation trait binding.
func (t *operationTrait) Stomp(b spec.StompOperationBinding) *operationTrait {
	return t.setBinding(spec.ProtocolStomp, &b)
}

// Stomp attaches a STOMP message trait binding.
func (t *messageTrait) Stomp(b spec.StompMessageBinding) *messageTrait {
	return t.setBinding(spec.ProtocolStomp, &b)
}
