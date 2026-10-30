package asyncgo

import "github.com/RubenRibGarcia/asyncgo/spec"

// --- HTTP bindings -----------------------------------------------------------

// HTTP attaches an HTTP server binding.
func (s *server) HTTP(b spec.HTTPServerBinding) *server {
	return s.setBinding(spec.ProtocolHTTP, &b)
}

// HTTP attaches an HTTP channel binding.
func (c *channel) HTTP(b spec.HTTPChannelBinding) *channel {
	return c.setBinding(spec.ProtocolHTTP, &b)
}

// HTTP attaches an HTTP operation binding.
func (o *operation) HTTP(b spec.HTTPOperationBinding) *operation {
	return o.setBinding(spec.ProtocolHTTP, &b)
}

// HTTP attaches an HTTP message binding.
func (m *message) HTTP(b spec.HTTPMessageBinding) *message {
	return m.setBinding(spec.ProtocolHTTP, &b)
}

// HTTP attaches an HTTP operation trait binding.
func (t *operationTrait) HTTP(b spec.HTTPOperationBinding) *operationTrait {
	return t.setBinding(spec.ProtocolHTTP, &b)
}

// HTTP attaches an HTTP message trait binding.
func (t *messageTrait) HTTP(b spec.HTTPMessageBinding) *messageTrait {
	return t.setBinding(spec.ProtocolHTTP, &b)
}

// --- WebSockets bindings -----------------------------------------------------

// WS attaches a WebSockets server binding.
func (s *server) WS(b spec.WSServerBinding) *server {
	return s.setBinding(spec.ProtocolWS, &b)
}

// WS attaches a WebSockets channel binding.
func (c *channel) WS(b spec.WSChannelBinding) *channel {
	return c.setBinding(spec.ProtocolWS, &b)
}

// WS attaches a WebSockets operation binding.
func (o *operation) WS(b spec.WSOperationBinding) *operation {
	return o.setBinding(spec.ProtocolWS, &b)
}

// WS attaches a WebSockets message binding.
func (m *message) WS(b spec.WSMessageBinding) *message {
	return m.setBinding(spec.ProtocolWS, &b)
}

// WS attaches a WebSockets operation trait binding.
func (t *operationTrait) WS(b spec.WSOperationBinding) *operationTrait {
	return t.setBinding(spec.ProtocolWS, &b)
}

// WS attaches a WebSockets message trait binding.
func (t *messageTrait) WS(b spec.WSMessageBinding) *messageTrait {
	return t.setBinding(spec.ProtocolWS, &b)
}
