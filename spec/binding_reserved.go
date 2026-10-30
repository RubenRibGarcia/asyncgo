package spec

// ReservedBinding is the shape of a binding object the specification defines
// but reserves for future use: it carries only bindingVersion. Every protocol
// object whose JSON schema is empty ({} — "MUST NOT contain any properties")
// aliases this type, so DSL signatures stay protocol-specific without dozens of
// byte-identical structs. The per-protocol binding files hold only the objects
// that define fields.
type ReservedBinding struct {
	BindingVersion string `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}

type (
	// AMQPServerBinding describes AMQP-specific server information. The binding
	// object is reserved.
	AMQPServerBinding = ReservedBinding

	// AMQP1ServerBinding describes AMQP 1.0-specific server information. The
	// binding object is reserved.
	AMQP1ServerBinding = ReservedBinding
	// AMQP1ChannelBinding describes AMQP 1.0-specific channel information. The
	// binding object is reserved.
	AMQP1ChannelBinding = ReservedBinding
	// AMQP1OperationBinding describes AMQP 1.0-specific operation information.
	// The binding object is reserved.
	AMQP1OperationBinding = ReservedBinding
	// AMQP1MessageBinding describes AMQP 1.0-specific message information. The
	// binding object is reserved.
	AMQP1MessageBinding = ReservedBinding

	// AnypointMQServerBinding describes Anypoint MQ-specific server information.
	// The binding object is reserved.
	AnypointMQServerBinding = ReservedBinding
	// AnypointMQOperationBinding describes Anypoint MQ-specific operation
	// information. The binding object is reserved.
	AnypointMQOperationBinding = ReservedBinding

	// GooglePubSubServerBinding describes Google Cloud Pub/Sub-specific server
	// information. The binding object is reserved.
	GooglePubSubServerBinding = ReservedBinding
	// GooglePubSubOperationBinding describes Google Cloud Pub/Sub-specific
	// operation information. The binding object is reserved.
	GooglePubSubOperationBinding = ReservedBinding

	// HTTPServerBinding describes HTTP-specific server information. The binding
	// object is reserved.
	HTTPServerBinding = ReservedBinding
	// HTTPChannelBinding describes HTTP-specific channel information. The
	// binding object is reserved.
	HTTPChannelBinding = ReservedBinding

	// IBMMQOperationBinding describes IBM MQ-specific operation information. The
	// binding object is reserved.
	IBMMQOperationBinding = ReservedBinding

	// JMSOperationBinding describes JMS-specific operation information. The
	// binding object is reserved.
	JMSOperationBinding = ReservedBinding

	// MQTTChannelBinding describes MQTT-specific channel information. The
	// binding object is reserved.
	MQTTChannelBinding = ReservedBinding

	// NATSServerBinding describes NATS-specific server information. The binding
	// object is reserved.
	NATSServerBinding = ReservedBinding
	// NATSChannelBinding describes NATS-specific channel information. The
	// binding object is reserved.
	NATSChannelBinding = ReservedBinding
	// NATSMessageBinding describes NATS-specific message information. The
	// binding object is reserved.
	NATSMessageBinding = ReservedBinding

	// SNSServerBinding describes SNS-specific server information. The binding
	// object is reserved.
	SNSServerBinding = ReservedBinding
	// SNSMessageBinding describes SNS-specific message information. The binding
	// object is reserved.
	SNSMessageBinding = ReservedBinding

	// SolaceChannelBinding describes Solace-specific channel information. The
	// binding object is reserved.
	SolaceChannelBinding = ReservedBinding
	// SolaceMessageBinding describes Solace-specific message information. The
	// binding object is reserved.
	SolaceMessageBinding = ReservedBinding

	// SQSServerBinding describes SQS-specific server information. The binding
	// object is reserved.
	SQSServerBinding = ReservedBinding
	// SQSMessageBinding describes SQS-specific message information. The binding
	// object is reserved.
	SQSMessageBinding = ReservedBinding

	// WSServerBinding describes WebSockets-specific server information. The
	// binding object is reserved.
	WSServerBinding = ReservedBinding
	// WSOperationBinding describes WebSockets-specific operation information.
	// The binding object is reserved.
	WSOperationBinding = ReservedBinding
	// WSMessageBinding describes WebSockets-specific message information. The
	// binding object is reserved.
	WSMessageBinding = ReservedBinding

	// StompServerBinding describes STOMP-specific server information. The
	// binding object is reserved.
	StompServerBinding = ReservedBinding
	// StompChannelBinding describes STOMP-specific channel information. The
	// binding object is reserved.
	StompChannelBinding = ReservedBinding
	// StompOperationBinding describes STOMP-specific operation information. The
	// binding object is reserved.
	StompOperationBinding = ReservedBinding
	// StompMessageBinding describes STOMP-specific message information. The
	// binding object is reserved.
	StompMessageBinding = ReservedBinding

	// RedisServerBinding describes Redis-specific server information. The
	// binding object is reserved.
	RedisServerBinding = ReservedBinding
	// RedisChannelBinding describes Redis-specific channel information. The
	// binding object is reserved.
	RedisChannelBinding = ReservedBinding
	// RedisOperationBinding describes Redis-specific operation information. The
	// binding object is reserved.
	RedisOperationBinding = ReservedBinding
	// RedisMessageBinding describes Redis-specific message information. The
	// binding object is reserved.
	RedisMessageBinding = ReservedBinding

	// MercureServerBinding describes Mercure-specific server information. The
	// binding object is reserved. Mercure is not part of the published 3.1.0
	// binding JSON Schemas, so a document using it fails schema validation.
	MercureServerBinding = ReservedBinding
	// MercureChannelBinding describes Mercure-specific channel information. The
	// binding object is reserved. Mercure is not part of the published 3.1.0
	// binding JSON Schemas.
	MercureChannelBinding = ReservedBinding
	// MercureOperationBinding describes Mercure-specific operation information.
	// The binding object is reserved. Mercure is not part of the published 3.1.0
	// binding JSON Schemas.
	MercureOperationBinding = ReservedBinding
	// MercureMessageBinding describes Mercure-specific message information. The
	// binding object is reserved. Mercure is not part of the published 3.1.0
	// binding JSON Schemas.
	MercureMessageBinding = ReservedBinding
)
