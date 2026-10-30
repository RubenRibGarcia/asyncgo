package asyncgo

import "github.com/RubenRibGarcia/asyncgo/spec"

// --- Kafka bindings ----------------------------------------------------------

// Kafka attaches a Kafka server binding.
func (s *server) Kafka(b spec.KafkaServerBinding) *server {
	return s.setBinding(spec.ProtocolKafka, &b)
}

// Kafka attaches a Kafka channel binding.
func (c *channel) Kafka(b spec.KafkaChannelBinding) *channel {
	return c.setBinding(spec.ProtocolKafka, &b)
}

// Kafka attaches a Kafka operation binding.
func (o *operation) Kafka(b spec.KafkaOperationBinding) *operation {
	return o.setBinding(spec.ProtocolKafka, &b)
}

// Kafka attaches a Kafka message binding.
func (m *message) Kafka(b spec.KafkaMessageBinding) *message {
	return m.setBinding(spec.ProtocolKafka, &b)
}

// Kafka attaches a Kafka operation trait binding.
func (t *operationTrait) Kafka(b spec.KafkaOperationBinding) *operationTrait {
	return t.setBinding(spec.ProtocolKafka, &b)
}

// Kafka attaches a Kafka message trait binding.
func (t *messageTrait) Kafka(b spec.KafkaMessageBinding) *messageTrait {
	return t.setBinding(spec.ProtocolKafka, &b)
}
