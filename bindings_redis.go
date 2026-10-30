package asyncgo

import "github.com/RubenRibGarcia/asyncgo/spec"

// --- Redis bindings ----------------------------------------------------------

// Redis attaches a Redis server binding.
func (s *server) Redis(b spec.RedisServerBinding) *server {
	return s.setBinding(spec.ProtocolRedis, &b)
}

// Redis attaches a Redis channel binding.
func (c *channel) Redis(b spec.RedisChannelBinding) *channel {
	return c.setBinding(spec.ProtocolRedis, &b)
}

// Redis attaches a Redis operation binding.
func (o *operation) Redis(b spec.RedisOperationBinding) *operation {
	return o.setBinding(spec.ProtocolRedis, &b)
}

// Redis attaches a Redis message binding.
func (m *message) Redis(b spec.RedisMessageBinding) *message {
	return m.setBinding(spec.ProtocolRedis, &b)
}

// Redis attaches a Redis operation trait binding.
func (t *operationTrait) Redis(b spec.RedisOperationBinding) *operationTrait {
	return t.setBinding(spec.ProtocolRedis, &b)
}

// Redis attaches a Redis message trait binding.
func (t *messageTrait) Redis(b spec.RedisMessageBinding) *messageTrait {
	return t.setBinding(spec.ProtocolRedis, &b)
}
