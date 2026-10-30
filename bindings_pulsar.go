package asyncgo

import "github.com/RubenRibGarcia/asyncgo/spec"

// --- Pulsar bindings ---------------------------------------------------------
//
// The specification defines only server and channel objects for Pulsar, so
// there is no operation or message helper.

// Pulsar attaches a Pulsar server binding.
func (s *server) Pulsar(b spec.PulsarServerBinding) *server {
	return s.setBinding(spec.ProtocolPulsar, &b)
}

// Pulsar attaches a Pulsar channel binding.
func (c *channel) Pulsar(b spec.PulsarChannelBinding) *channel {
	return c.setBinding(spec.ProtocolPulsar, &b)
}
