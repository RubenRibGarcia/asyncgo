package asyncgo

import "github.com/RubenRibGarcia/asyncgo/spec"

// --- MQTT bindings -----------------------------------------------------------

// MQTT attaches an MQTT server binding.
func (s *server) MQTT(b spec.MQTTServerBinding) *server {
	return s.setBinding(spec.ProtocolMQTT, &b)
}

// MQTT attaches an MQTT channel binding.
func (c *channel) MQTT(b spec.MQTTChannelBinding) *channel {
	return c.setBinding(spec.ProtocolMQTT, &b)
}

// MQTT attaches an MQTT operation binding.
func (o *operation) MQTT(b spec.MQTTOperationBinding) *operation {
	return o.setBinding(spec.ProtocolMQTT, &b)
}

// MQTT attaches an MQTT message binding.
func (m *message) MQTT(b spec.MQTTMessageBinding) *message {
	return m.setBinding(spec.ProtocolMQTT, &b)
}

// MQTT attaches an MQTT operation trait binding.
func (t *operationTrait) MQTT(b spec.MQTTOperationBinding) *operationTrait {
	return t.setBinding(spec.ProtocolMQTT, &b)
}

// MQTT attaches an MQTT message trait binding.
func (t *messageTrait) MQTT(b spec.MQTTMessageBinding) *messageTrait {
	return t.setBinding(spec.ProtocolMQTT, &b)
}

// --- MQTT 5 bindings ---------------------------------------------------------

// MQTT5 attaches an MQTT 5 server binding. The MQTT 5 binding is deprecated
// upstream in favour of MQTT; the specification defines no channel, operation,
// or message object for it.
func (s *server) MQTT5(b spec.MQTT5ServerBinding) *server {
	return s.setBinding(spec.ProtocolMQTT5, &b)
}
