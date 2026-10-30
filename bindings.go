package asyncgo

import "github.com/RubenRibGarcia/asyncgo/spec"

// The following setters attach protocol-specific bindings to servers, channels,
// operations, messages, and the two trait builders. Each protocol has a typed
// helper in its own bindings_<protocol>.go file (compile-checked), plus a
// generic Binding escape hatch for protocols the library does not model.

// --- server bindings ---------------------------------------------------------

func (s *server) setBinding(proto string, v any) *server {
	if s.s.Bindings == nil {
		s.s.Bindings = spec.ServerBindings{}
	}
	s.s.Bindings[proto] = v
	return s
}

// Binding attaches a protocol-specific server binding under the given protocol key.
func (s *server) Binding(proto string, v any) *server { return s.setBinding(proto, v) }

// --- channel bindings --------------------------------------------------------

func (c *channel) setBinding(proto string, v any) *channel {
	if c.s.Bindings == nil {
		c.s.Bindings = spec.ChannelBindings{}
	}
	c.s.Bindings[proto] = v
	return c
}

// Binding attaches a protocol-specific channel binding under the given protocol key.
func (c *channel) Binding(proto string, v any) *channel { return c.setBinding(proto, v) }

// --- operation bindings ------------------------------------------------------

func (o *operation) setBinding(proto string, v any) *operation {
	if o.bindings == nil {
		o.bindings = spec.OperationBindings{}
	}
	o.bindings[proto] = v
	return o
}

// Binding attaches a protocol-specific operation binding under the given protocol key.
func (o *operation) Binding(proto string, v any) *operation { return o.setBinding(proto, v) }

// --- message bindings --------------------------------------------------------

func (m *message) setBinding(proto string, v any) *message {
	if m.bindings == nil {
		m.bindings = spec.MessageBindings{}
	}
	m.bindings[proto] = v
	return m
}

// Binding attaches a protocol-specific message binding under the given protocol key.
func (m *message) Binding(proto string, v any) *message { return m.setBinding(proto, v) }

// --- operation trait bindings ------------------------------------------------

func (t *operationTrait) setBinding(proto string, v any) *operationTrait {
	if t.t.Bindings == nil {
		t.t.Bindings = spec.OperationBindings{}
	}
	t.t.Bindings[proto] = v
	return t
}

// Binding attaches a protocol-specific operation trait binding under the given
// protocol key.
func (t *operationTrait) Binding(proto string, v any) *operationTrait {
	return t.setBinding(proto, v)
}

// --- message trait bindings --------------------------------------------------

func (t *messageTrait) setBinding(proto string, v any) *messageTrait {
	if t.t.Bindings == nil {
		t.t.Bindings = spec.MessageBindings{}
	}
	t.t.Bindings[proto] = v
	return t
}

// Binding attaches a protocol-specific message trait binding under the given
// protocol key.
func (t *messageTrait) Binding(proto string, v any) *messageTrait {
	return t.setBinding(proto, v)
}
