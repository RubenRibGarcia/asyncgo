package asyncgo

import "github.com/RubenRibGarcia/asyncgo/spec"

// --- Anypoint MQ bindings ----------------------------------------------------

// AnypointMQ attaches an Anypoint MQ server binding.
func (s *server) AnypointMQ(b spec.AnypointMQServerBinding) *server {
	return s.setBinding(spec.ProtocolAnypointMQ, &b)
}

// AnypointMQ attaches an Anypoint MQ channel binding.
func (c *channel) AnypointMQ(b spec.AnypointMQChannelBinding) *channel {
	return c.setBinding(spec.ProtocolAnypointMQ, &b)
}

// AnypointMQ attaches an Anypoint MQ operation binding.
func (o *operation) AnypointMQ(b spec.AnypointMQOperationBinding) *operation {
	return o.setBinding(spec.ProtocolAnypointMQ, &b)
}

// AnypointMQ attaches an Anypoint MQ message binding.
func (m *message) AnypointMQ(b spec.AnypointMQMessageBinding) *message {
	return m.setBinding(spec.ProtocolAnypointMQ, &b)
}

// AnypointMQ attaches an Anypoint MQ operation trait binding.
func (t *operationTrait) AnypointMQ(b spec.AnypointMQOperationBinding) *operationTrait {
	return t.setBinding(spec.ProtocolAnypointMQ, &b)
}

// AnypointMQ attaches an Anypoint MQ message trait binding.
func (t *messageTrait) AnypointMQ(b spec.AnypointMQMessageBinding) *messageTrait {
	return t.setBinding(spec.ProtocolAnypointMQ, &b)
}
