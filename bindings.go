package asyncgo

import "github.com/RubenRibGarcia/asyncgo/spec"

// The following setters attach protocol-specific bindings to servers, channels,
// operations, messages, and the two trait builders. Each protocol has a typed
// helper in its own protocol section below (compile-checked), plus a generic
// Binding escape hatch for protocols the library does not model.

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

// --- AMQP 0-9-1 bindings -----------------------------------------------------

// AMQP attaches an AMQP server binding.
func (s *server) AMQP(b spec.AMQPServerBinding) *server {
	return s.setBinding(spec.ProtocolAMQP, &b)
}

// AMQP attaches an AMQP channel binding.
func (c *channel) AMQP(b spec.AMQPChannelBinding) *channel {
	return c.setBinding(spec.ProtocolAMQP, &b)
}

// AMQP attaches an AMQP operation binding.
func (o *operation) AMQP(b spec.AMQPOperationBinding) *operation {
	return o.setBinding(spec.ProtocolAMQP, &b)
}

// AMQP attaches an AMQP message binding.
func (m *message) AMQP(b spec.AMQPMessageBinding) *message {
	return m.setBinding(spec.ProtocolAMQP, &b)
}

// AMQP attaches an AMQP operation trait binding.
func (t *operationTrait) AMQP(b spec.AMQPOperationBinding) *operationTrait {
	return t.setBinding(spec.ProtocolAMQP, &b)
}

// AMQP attaches an AMQP message trait binding.
func (t *messageTrait) AMQP(b spec.AMQPMessageBinding) *messageTrait {
	return t.setBinding(spec.ProtocolAMQP, &b)
}

// --- AMQP 1.0 bindings -------------------------------------------------------

// AMQP1 attaches an AMQP 1.0 server binding.
func (s *server) AMQP1(b spec.AMQP1ServerBinding) *server {
	return s.setBinding(spec.ProtocolAMQP1, &b)
}

// AMQP1 attaches an AMQP 1.0 channel binding.
func (c *channel) AMQP1(b spec.AMQP1ChannelBinding) *channel {
	return c.setBinding(spec.ProtocolAMQP1, &b)
}

// AMQP1 attaches an AMQP 1.0 operation binding.
func (o *operation) AMQP1(b spec.AMQP1OperationBinding) *operation {
	return o.setBinding(spec.ProtocolAMQP1, &b)
}

// AMQP1 attaches an AMQP 1.0 message binding.
func (m *message) AMQP1(b spec.AMQP1MessageBinding) *message {
	return m.setBinding(spec.ProtocolAMQP1, &b)
}

// AMQP1 attaches an AMQP 1.0 operation trait binding.
func (t *operationTrait) AMQP1(b spec.AMQP1OperationBinding) *operationTrait {
	return t.setBinding(spec.ProtocolAMQP1, &b)
}

// AMQP1 attaches an AMQP 1.0 message trait binding.
func (t *messageTrait) AMQP1(b spec.AMQP1MessageBinding) *messageTrait {
	return t.setBinding(spec.ProtocolAMQP1, &b)
}

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

// --- IBM MQ bindings ---------------------------------------------------------

// IBMMQ attaches an IBM MQ server binding.
func (s *server) IBMMQ(b spec.IBMMQServerBinding) *server {
	return s.setBinding(spec.ProtocolIBMMQ, &b)
}

// IBMMQ attaches an IBM MQ channel binding.
func (c *channel) IBMMQ(b spec.IBMMQChannelBinding) *channel {
	return c.setBinding(spec.ProtocolIBMMQ, &b)
}

// IBMMQ attaches an IBM MQ operation binding.
func (o *operation) IBMMQ(b spec.IBMMQOperationBinding) *operation {
	return o.setBinding(spec.ProtocolIBMMQ, &b)
}

// IBMMQ attaches an IBM MQ message binding.
func (m *message) IBMMQ(b spec.IBMMQMessageBinding) *message {
	return m.setBinding(spec.ProtocolIBMMQ, &b)
}

// IBMMQ attaches an IBM MQ operation trait binding.
func (t *operationTrait) IBMMQ(b spec.IBMMQOperationBinding) *operationTrait {
	return t.setBinding(spec.ProtocolIBMMQ, &b)
}

// IBMMQ attaches an IBM MQ message trait binding.
func (t *messageTrait) IBMMQ(b spec.IBMMQMessageBinding) *messageTrait {
	return t.setBinding(spec.ProtocolIBMMQ, &b)
}

// --- JMS bindings ------------------------------------------------------------

// JMS attaches a JMS server binding.
func (s *server) JMS(b spec.JMSServerBinding) *server {
	return s.setBinding(spec.ProtocolJMS, &b)
}

// JMS attaches a JMS channel binding.
func (c *channel) JMS(b spec.JMSChannelBinding) *channel {
	return c.setBinding(spec.ProtocolJMS, &b)
}

// JMS attaches a JMS operation binding.
func (o *operation) JMS(b spec.JMSOperationBinding) *operation {
	return o.setBinding(spec.ProtocolJMS, &b)
}

// JMS attaches a JMS message binding.
func (m *message) JMS(b spec.JMSMessageBinding) *message {
	return m.setBinding(spec.ProtocolJMS, &b)
}

// JMS attaches a JMS operation trait binding.
func (t *operationTrait) JMS(b spec.JMSOperationBinding) *operationTrait {
	return t.setBinding(spec.ProtocolJMS, &b)
}

// JMS attaches a JMS message trait binding.
func (t *messageTrait) JMS(b spec.JMSMessageBinding) *messageTrait {
	return t.setBinding(spec.ProtocolJMS, &b)
}

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

// --- NATS bindings -----------------------------------------------------------

// NATS attaches a NATS server binding.
func (s *server) NATS(b spec.NATSServerBinding) *server {
	return s.setBinding(spec.ProtocolNATS, &b)
}

// NATS attaches a NATS channel binding.
func (c *channel) NATS(b spec.NATSChannelBinding) *channel {
	return c.setBinding(spec.ProtocolNATS, &b)
}

// NATS attaches a NATS operation binding.
func (o *operation) NATS(b spec.NATSOperationBinding) *operation {
	return o.setBinding(spec.ProtocolNATS, &b)
}

// NATS attaches a NATS message binding.
func (m *message) NATS(b spec.NATSMessageBinding) *message {
	return m.setBinding(spec.ProtocolNATS, &b)
}

// NATS attaches a NATS operation trait binding.
func (t *operationTrait) NATS(b spec.NATSOperationBinding) *operationTrait {
	return t.setBinding(spec.ProtocolNATS, &b)
}

// NATS attaches a NATS message trait binding.
func (t *messageTrait) NATS(b spec.NATSMessageBinding) *messageTrait {
	return t.setBinding(spec.ProtocolNATS, &b)
}

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

// --- ROS 2 bindings ----------------------------------------------------------
//
// The specification defines only server and operation objects for ROS 2, so
// there is no channel or message helper.

// ROS2 attaches a ROS 2 server binding.
func (s *server) ROS2(b spec.ROS2ServerBinding) *server {
	return s.setBinding(spec.ProtocolROS2, &b)
}

// ROS2 attaches a ROS 2 operation binding.
func (o *operation) ROS2(b spec.ROS2OperationBinding) *operation {
	return o.setBinding(spec.ProtocolROS2, &b)
}

// ROS2 attaches a ROS 2 operation trait binding.
func (t *operationTrait) ROS2(b spec.ROS2OperationBinding) *operationTrait {
	return t.setBinding(spec.ProtocolROS2, &b)
}

// --- SNS bindings ------------------------------------------------------------

// SNS attaches an SNS server binding.
func (s *server) SNS(b spec.SNSServerBinding) *server {
	return s.setBinding(spec.ProtocolSNS, &b)
}

// SNS attaches an SNS channel binding.
func (c *channel) SNS(b spec.SNSChannelBinding) *channel {
	return c.setBinding(spec.ProtocolSNS, &b)
}

// SNS attaches an SNS operation binding.
func (o *operation) SNS(b spec.SNSOperationBinding) *operation {
	return o.setBinding(spec.ProtocolSNS, &b)
}

// SNS attaches an SNS message binding.
func (m *message) SNS(b spec.SNSMessageBinding) *message {
	return m.setBinding(spec.ProtocolSNS, &b)
}

// SNS attaches an SNS operation trait binding.
func (t *operationTrait) SNS(b spec.SNSOperationBinding) *operationTrait {
	return t.setBinding(spec.ProtocolSNS, &b)
}

// SNS attaches an SNS message trait binding.
func (t *messageTrait) SNS(b spec.SNSMessageBinding) *messageTrait {
	return t.setBinding(spec.ProtocolSNS, &b)
}

// --- Solace bindings ---------------------------------------------------------

// Solace attaches a Solace server binding.
func (s *server) Solace(b spec.SolaceServerBinding) *server {
	return s.setBinding(spec.ProtocolSolace, &b)
}

// Solace attaches a Solace channel binding.
func (c *channel) Solace(b spec.SolaceChannelBinding) *channel {
	return c.setBinding(spec.ProtocolSolace, &b)
}

// Solace attaches a Solace operation binding.
func (o *operation) Solace(b spec.SolaceOperationBinding) *operation {
	return o.setBinding(spec.ProtocolSolace, &b)
}

// Solace attaches a Solace message binding.
func (m *message) Solace(b spec.SolaceMessageBinding) *message {
	return m.setBinding(spec.ProtocolSolace, &b)
}

// Solace attaches a Solace operation trait binding.
func (t *operationTrait) Solace(b spec.SolaceOperationBinding) *operationTrait {
	return t.setBinding(spec.ProtocolSolace, &b)
}

// Solace attaches a Solace message trait binding.
func (t *messageTrait) Solace(b spec.SolaceMessageBinding) *messageTrait {
	return t.setBinding(spec.ProtocolSolace, &b)
}

// --- SQS bindings ------------------------------------------------------------

// SQS attaches an SQS server binding.
func (s *server) SQS(b spec.SQSServerBinding) *server {
	return s.setBinding(spec.ProtocolSQS, &b)
}

// SQS attaches an SQS channel binding.
func (c *channel) SQS(b spec.SQSChannelBinding) *channel {
	return c.setBinding(spec.ProtocolSQS, &b)
}

// SQS attaches an SQS operation binding.
func (o *operation) SQS(b spec.SQSOperationBinding) *operation {
	return o.setBinding(spec.ProtocolSQS, &b)
}

// SQS attaches an SQS message binding.
func (m *message) SQS(b spec.SQSMessageBinding) *message {
	return m.setBinding(spec.ProtocolSQS, &b)
}

// SQS attaches an SQS operation trait binding.
func (t *operationTrait) SQS(b spec.SQSOperationBinding) *operationTrait {
	return t.setBinding(spec.ProtocolSQS, &b)
}

// SQS attaches an SQS message trait binding.
func (t *messageTrait) SQS(b spec.SQSMessageBinding) *messageTrait {
	return t.setBinding(spec.ProtocolSQS, &b)
}

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
