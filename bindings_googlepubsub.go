package asyncgo

import "github.com/RubenRibGarcia/asyncgo/spec"

// --- Google Cloud Pub/Sub bindings -------------------------------------------

// GooglePubSub attaches a Google Cloud Pub/Sub server binding.
func (s *server) GooglePubSub(b spec.GooglePubSubServerBinding) *server {
	return s.setBinding(spec.ProtocolGooglePubSub, &b)
}

// GooglePubSub attaches a Google Cloud Pub/Sub channel binding.
func (c *channel) GooglePubSub(b spec.GooglePubSubChannelBinding) *channel {
	return c.setBinding(spec.ProtocolGooglePubSub, &b)
}

// GooglePubSub attaches a Google Cloud Pub/Sub operation binding.
func (o *operation) GooglePubSub(b spec.GooglePubSubOperationBinding) *operation {
	return o.setBinding(spec.ProtocolGooglePubSub, &b)
}

// GooglePubSub attaches a Google Cloud Pub/Sub message binding.
func (m *message) GooglePubSub(b spec.GooglePubSubMessageBinding) *message {
	return m.setBinding(spec.ProtocolGooglePubSub, &b)
}

// GooglePubSub attaches a Google Cloud Pub/Sub operation trait binding.
func (t *operationTrait) GooglePubSub(b spec.GooglePubSubOperationBinding) *operationTrait {
	return t.setBinding(spec.ProtocolGooglePubSub, &b)
}

// GooglePubSub attaches a Google Cloud Pub/Sub message trait binding.
func (t *messageTrait) GooglePubSub(b spec.GooglePubSubMessageBinding) *messageTrait {
	return t.setBinding(spec.ProtocolGooglePubSub, &b)
}
