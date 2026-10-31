package spec

// Protocol identifiers used as keys in the *Bindings maps.
const (
	ProtocolKafka        = "kafka"
	ProtocolAMQP         = "amqp"
	ProtocolNATS         = "nats"
	ProtocolMQTT         = "mqtt"
	ProtocolHTTP         = "http"
	ProtocolWS           = "ws"
	ProtocolAMQP1        = "amqp1"
	ProtocolMQTT5        = "mqtt5"
	ProtocolAnypointMQ   = "anypointmq"
	ProtocolJMS          = "jms"
	ProtocolSNS          = "sns"
	ProtocolSolace       = "solace"
	ProtocolSQS          = "sqs"
	ProtocolStomp        = "stomp"
	ProtocolRedis        = "redis"
	ProtocolMercure      = "mercure"
	ProtocolIBMMQ        = "ibmmq"
	ProtocolGooglePubSub = "googlepubsub"
	ProtocolPulsar       = "pulsar"
	ProtocolROS2         = "ros2"
)

// ProtocolKeys lists every protocol key the model supports, in the order the
// bindings coverage table presents them.
//
// mqtt5 and mercure are listed even though the published 3.1.0 binding JSON
// Schemas omit them: the specification text defines both protocols, and a
// document using either is rejected only because the schemas lag the prose.
var ProtocolKeys = []string{
	ProtocolKafka,
	ProtocolAMQP,
	ProtocolNATS,
	ProtocolMQTT,
	ProtocolHTTP,
	ProtocolWS,
	ProtocolAMQP1,
	ProtocolMQTT5,
	ProtocolAnypointMQ,
	ProtocolJMS,
	ProtocolSNS,
	ProtocolSolace,
	ProtocolSQS,
	ProtocolStomp,
	ProtocolRedis,
	ProtocolMercure,
	ProtocolIBMMQ,
	ProtocolGooglePubSub,
	ProtocolPulsar,
	ProtocolROS2,
}

// The *Bindings types are protocol-keyed maps. Values are typed binding
// structs (see the protocol sections below); using a map keeps the model
// extensible to new protocols without changing the core object model.
type (
	// ServerBindings holds protocol-specific server bindings, keyed by protocol.
	ServerBindings map[string]any
	// ChannelBindings holds protocol-specific channel bindings, keyed by protocol.
	ChannelBindings map[string]any
	// OperationBindings holds protocol-specific operation bindings, keyed by protocol.
	OperationBindings map[string]any
	// MessageBindings holds protocol-specific message bindings, keyed by protocol.
	MessageBindings map[string]any
)

// --- Reserved bindings ------------------------------------------------------
//
// Every object below shares one shape: a binding object the specification
// defines but reserves for future use, carrying only bindingVersion.

// ReservedBinding is the shape of a binding object the specification defines
// but reserves for future use: it carries only bindingVersion. Every protocol
// object whose JSON schema is empty ({} — "MUST NOT contain any properties")
// aliases this type, so DSL signatures stay protocol-specific without dozens of
// byte-identical structs. The protocol sections below hold only the objects
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

// --- AMQP 0-9-1 bindings (binding version 0.3.0) -----------------------------
//
// The server object is reserved: AMQPServerBinding is an alias in the
// reserved bindings section.

// AMQPChannelBinding describes AMQP-specific channel information.
type AMQPChannelBinding struct {
	Is             string        `json:"is,omitempty"             yaml:"is,omitempty"` // "queue" | "routingKey"
	Exchange       *AMQPExchange `json:"exchange,omitempty"       yaml:"exchange,omitempty"`
	Queue          *AMQPQueue    `json:"queue,omitempty"          yaml:"queue,omitempty"`
	BindingVersion string        `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}

// AMQPExchange describes an AMQP exchange.
type AMQPExchange struct {
	Name       string `json:"name,omitempty"       yaml:"name,omitempty"`
	Type       string `json:"type,omitempty"       yaml:"type,omitempty"`
	Durable    bool   `json:"durable,omitempty"    yaml:"durable,omitempty"`
	AutoDelete bool   `json:"autoDelete,omitempty" yaml:"autoDelete,omitempty"`
	VHost      string `json:"vhost,omitempty"      yaml:"vhost,omitempty"`
}

// AMQPQueue describes an AMQP queue.
type AMQPQueue struct {
	Name       string `json:"name,omitempty"       yaml:"name,omitempty"`
	Durable    bool   `json:"durable,omitempty"    yaml:"durable,omitempty"`
	Exclusive  bool   `json:"exclusive,omitempty"  yaml:"exclusive,omitempty"`
	AutoDelete bool   `json:"autoDelete,omitempty" yaml:"autoDelete,omitempty"`
	VHost      string `json:"vhost,omitempty"      yaml:"vhost,omitempty"`
}

// AMQPOperationBinding describes AMQP-specific operation information. The
// binding version 0.3.0 defines no replyTo field; it was removed along with the
// 2.x shape.
type AMQPOperationBinding struct {
	Expiration     int      `json:"expiration,omitempty"     yaml:"expiration,omitempty"`
	UserID         string   `json:"userId,omitempty"         yaml:"userId,omitempty"`
	CC             []string `json:"cc,omitempty"             yaml:"cc,omitempty"`
	Priority       int      `json:"priority,omitempty"       yaml:"priority,omitempty"`
	DeliveryMode   int      `json:"deliveryMode,omitempty"   yaml:"deliveryMode,omitempty"`
	Mandatory      bool     `json:"mandatory,omitempty"      yaml:"mandatory,omitempty"`
	BCC            []string `json:"bcc,omitempty"            yaml:"bcc,omitempty"`
	Timestamp      bool     `json:"timestamp,omitempty"      yaml:"timestamp,omitempty"`
	Ack            bool     `json:"ack,omitempty"            yaml:"ack,omitempty"`
	BindingVersion string   `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}

// AMQPMessageBinding describes AMQP-specific message information.
type AMQPMessageBinding struct {
	ContentEncoding string `json:"contentEncoding,omitempty" yaml:"contentEncoding,omitempty"`
	MessageType     string `json:"messageType,omitempty"     yaml:"messageType,omitempty"`
	BindingVersion  string `json:"bindingVersion,omitempty"  yaml:"bindingVersion,omitempty"`
}

// --- Anypoint MQ bindings (binding version 0.0.1) ----------------------------
//
// The server and operation objects are reserved: they are aliases in the
// reserved bindings section.

// AnypointMQChannelBinding describes Anypoint MQ-specific channel information.
type AnypointMQChannelBinding struct {
	Destination     string `json:"destination,omitempty"     yaml:"destination,omitempty"`
	DestinationType string `json:"destinationType,omitempty" yaml:"destinationType,omitempty"`
	BindingVersion  string `json:"bindingVersion,omitempty"  yaml:"bindingVersion,omitempty"`
}

// AnypointMQMessageBinding describes Anypoint MQ-specific message information.
// Headers is Schema Object | Reference Object, so it is `any`.
type AnypointMQMessageBinding struct {
	Headers        any    `json:"headers,omitempty"        yaml:"headers,omitempty"`
	BindingVersion string `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}

// --- Google Cloud Pub/Sub bindings (binding version 0.2.0) -------------------
//
// The server and operation objects are reserved: they are aliases in the
// reserved bindings section.

// GooglePubSubChannelBinding describes Google Cloud Pub/Sub-specific channel
// information. SchemaSettings is required by the binding specification.
type GooglePubSubChannelBinding struct {
	Labels                   map[string]string                 `json:"labels,omitempty"                   yaml:"labels,omitempty"`
	MessageRetentionDuration string                            `json:"messageRetentionDuration,omitempty" yaml:"messageRetentionDuration,omitempty"`
	MessageStoragePolicy     *GooglePubSubMessageStoragePolicy `json:"messageStoragePolicy,omitempty"     yaml:"messageStoragePolicy,omitempty"`
	SchemaSettings           *GooglePubSubSchemaSettings       `json:"schemaSettings"                     yaml:"schemaSettings"`
	BindingVersion           string                            `json:"bindingVersion,omitempty"           yaml:"bindingVersion,omitempty"`
}

// GooglePubSubMessageStoragePolicy describes where messages may be stored.
type GooglePubSubMessageStoragePolicy struct {
	AllowedPersistenceRegions []string `json:"allowedPersistenceRegions,omitempty" yaml:"allowedPersistenceRegions,omitempty"`
}

// GooglePubSubSchemaSettings describes the schema a topic is validated against.
// Encoding and Name are required by the binding specification.
type GooglePubSubSchemaSettings struct {
	Encoding        string `json:"encoding"                  yaml:"encoding"`
	FirstRevisionID string `json:"firstRevisionId,omitempty" yaml:"firstRevisionId,omitempty"`
	LastRevisionID  string `json:"lastRevisionId,omitempty"  yaml:"lastRevisionId,omitempty"`
	Name            string `json:"name"                      yaml:"name"`
}

// GooglePubSubMessageBinding describes Google Cloud Pub/Sub-specific message
// information.
type GooglePubSubMessageBinding struct {
	Attributes     map[string]string   `json:"attributes,omitempty"     yaml:"attributes,omitempty"`
	OrderingKey    string              `json:"orderingKey,omitempty"    yaml:"orderingKey,omitempty"`
	Schema         *GooglePubSubSchema `json:"schema,omitempty"         yaml:"schema,omitempty"`
	BindingVersion string              `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}

// GooglePubSubSchema references the schema a message is validated against. Name
// is required by the binding specification.
type GooglePubSubSchema struct {
	Name string `json:"name" yaml:"name"`
}

// --- HTTP bindings (binding version 0.3.0) -----------------------------------
//
// The server and channel objects are reserved: they are aliases in the
// reserved bindings section.

// HTTPOperationBinding describes HTTP-specific operation information. Query is
// the Schema Object describing the query string.
type HTTPOperationBinding struct {
	Method         string  `json:"method,omitempty"         yaml:"method,omitempty"`
	Query          *Schema `json:"query,omitempty"          yaml:"query,omitempty"`
	BindingVersion string  `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}

// HTTPMessageBinding describes HTTP-specific message information. Headers is
// the Schema Object describing the HTTP headers; StatusCode is the documented
// response status code.
type HTTPMessageBinding struct {
	Headers        *Schema `json:"headers,omitempty"        yaml:"headers,omitempty"`
	StatusCode     int     `json:"statusCode,omitempty"     yaml:"statusCode,omitempty"`
	BindingVersion string  `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}

// --- IBM MQ bindings (binding version 0.1.0) ---------------------------------
//
// The operation object is reserved: IBMMQOperationBinding is an alias in the
// reserved bindings section.

// IBMMQServerBinding describes IBM MQ-specific server information.
type IBMMQServerBinding struct {
	GroupID              string `json:"groupId,omitempty"              yaml:"groupId,omitempty"`
	CCDTQueueManagerName string `json:"ccdtQueueManagerName,omitempty" yaml:"ccdtQueueManagerName,omitempty"`
	CipherSpec           string `json:"cipherSpec,omitempty"           yaml:"cipherSpec,omitempty"`
	MultiEndpointServer  bool   `json:"multiEndpointServer,omitempty"  yaml:"multiEndpointServer,omitempty"`
	HeartBeatInterval    int    `json:"heartBeatInterval,omitempty"    yaml:"heartBeatInterval,omitempty"`
	BindingVersion       string `json:"bindingVersion,omitempty"       yaml:"bindingVersion,omitempty"`
}

// IBMMQChannelBinding describes IBM MQ-specific channel information.
type IBMMQChannelBinding struct {
	DestinationType string      `json:"destinationType,omitempty" yaml:"destinationType,omitempty"` // "topic" | "queue"
	Queue           *IBMMQQueue `json:"queue,omitempty"           yaml:"queue,omitempty"`
	Topic           *IBMMQTopic `json:"topic,omitempty"           yaml:"topic,omitempty"`
	MaxMsgLength    int         `json:"maxMsgLength,omitempty"    yaml:"maxMsgLength,omitempty"`
	BindingVersion  string      `json:"bindingVersion,omitempty"  yaml:"bindingVersion,omitempty"`
}

// IBMMQQueue describes an IBM MQ queue. ObjectName is required by the binding
// specification.
type IBMMQQueue struct {
	ObjectName    string `json:"objectName"              yaml:"objectName"`
	IsPartitioned bool   `json:"isPartitioned,omitempty" yaml:"isPartitioned,omitempty"`
	Exclusive     bool   `json:"exclusive,omitempty"     yaml:"exclusive,omitempty"`
}

// IBMMQTopic describes an IBM MQ topic.
type IBMMQTopic struct {
	String           string `json:"string,omitempty"           yaml:"string,omitempty"`
	ObjectName       string `json:"objectName,omitempty"       yaml:"objectName,omitempty"`
	DurablePermitted bool   `json:"durablePermitted,omitempty" yaml:"durablePermitted,omitempty"`
	LastMsgRetained  bool   `json:"lastMsgRetained,omitempty"  yaml:"lastMsgRetained,omitempty"`
}

// IBMMQMessageBinding describes IBM MQ-specific message information. Type is
// "string", "jms", or "binary"; Headers names the JMS header that carries the
// queue-manager routing information.
type IBMMQMessageBinding struct {
	Type           string `json:"type,omitempty"           yaml:"type,omitempty"`
	Headers        string `json:"headers,omitempty"        yaml:"headers,omitempty"`
	Description    string `json:"description,omitempty"    yaml:"description,omitempty"`
	Expiry         int    `json:"expiry,omitempty"         yaml:"expiry,omitempty"`
	BindingVersion string `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}

// --- JMS bindings (binding version 0.0.1) ------------------------------------
//
// The operation object is reserved: JMSOperationBinding is an alias in the
// reserved bindings section.

// JMSServerBinding describes JMS-specific server information.
type JMSServerBinding struct {
	JMSConnectionFactory string        `json:"jmsConnectionFactory"     yaml:"jmsConnectionFactory"`
	Properties           []JMSProperty `json:"properties,omitempty"     yaml:"properties,omitempty"`
	ClientID             string        `json:"clientID,omitempty"       yaml:"clientID,omitempty"`
	BindingVersion       string        `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}

// JMSProperty is one JMS connection-factory property.
type JMSProperty struct {
	Name  string `json:"name"  yaml:"name"`
	Value any    `json:"value" yaml:"value"`
}

// JMSChannelBinding describes JMS-specific channel information.
type JMSChannelBinding struct {
	Destination     string `json:"destination,omitempty"     yaml:"destination,omitempty"`
	DestinationType string `json:"destinationType,omitempty" yaml:"destinationType,omitempty"`
	BindingVersion  string `json:"bindingVersion,omitempty"  yaml:"bindingVersion,omitempty"`
}

// JMSMessageBinding describes JMS-specific message information. Headers is the
// Schema Object describing the JMS headers.
type JMSMessageBinding struct {
	Headers        *Schema `json:"headers,omitempty"        yaml:"headers,omitempty"`
	BindingVersion string  `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}

// --- Kafka bindings (binding version 0.5.0) ----------------------------------

// KafkaServerBinding describes Kafka-specific server information.
type KafkaServerBinding struct {
	SchemaRegistryURL    string `json:"schemaRegistryUrl,omitempty"    yaml:"schemaRegistryUrl,omitempty"`
	SchemaRegistryVendor string `json:"schemaRegistryVendor,omitempty" yaml:"schemaRegistryVendor,omitempty"`
	BindingVersion       string `json:"bindingVersion,omitempty"       yaml:"bindingVersion,omitempty"`
}

// KafkaChannelBinding describes Kafka-specific channel information.
type KafkaChannelBinding struct {
	Topic              string                   `json:"topic,omitempty"              yaml:"topic,omitempty"`
	Partitions         int                      `json:"partitions,omitempty"         yaml:"partitions,omitempty"`
	Replicas           int                      `json:"replicas,omitempty"           yaml:"replicas,omitempty"`
	TopicConfiguration *KafkaTopicConfiguration `json:"topicConfiguration,omitempty" yaml:"topicConfiguration,omitempty"`
	BindingVersion     string                   `json:"bindingVersion,omitempty"     yaml:"bindingVersion,omitempty"`
}

// KafkaTopicConfiguration describes the topic-level configuration of a Kafka
// channel. The keys are Kafka's own dotted configuration names.
type KafkaTopicConfiguration struct {
	CleanupPolicy                     []string `json:"cleanup.policy,omitempty"                        yaml:"cleanup.policy,omitempty"`
	RetentionMs                       int      `json:"retention.ms,omitempty"                          yaml:"retention.ms,omitempty"`
	RetentionBytes                    int      `json:"retention.bytes,omitempty"                       yaml:"retention.bytes,omitempty"`
	DeleteRetentionMs                 int      `json:"delete.retention.ms,omitempty"                   yaml:"delete.retention.ms,omitempty"`
	MaxMessageBytes                   int      `json:"max.message.bytes,omitempty"                     yaml:"max.message.bytes,omitempty"`
	ConfluentKeySchemaValidation      bool     `json:"confluent.key.schema.validation,omitempty"       yaml:"confluent.key.schema.validation,omitempty"`
	ConfluentKeySubjectNameStrategy   string   `json:"confluent.key.subject.name.strategy,omitempty"   yaml:"confluent.key.subject.name.strategy,omitempty"`
	ConfluentValueSchemaValidation    bool     `json:"confluent.value.schema.validation,omitempty"     yaml:"confluent.value.schema.validation,omitempty"`
	ConfluentValueSubjectNameStrategy string   `json:"confluent.value.subject.name.strategy,omitempty" yaml:"confluent.value.subject.name.strategy,omitempty"`
}

// KafkaOperationBinding describes Kafka-specific operation information. GroupID
// and ClientID are Schema Objects describing how the value is derived.
type KafkaOperationBinding struct {
	GroupID        *Schema `json:"groupId,omitempty"        yaml:"groupId,omitempty"`
	ClientID       *Schema `json:"clientId,omitempty"       yaml:"clientId,omitempty"`
	BindingVersion string  `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}

// KafkaMessageBinding describes Kafka-specific message information. Key is
// Schema Object | Reference Object: a schema describing the message key, or a
// reference to one. It is `any` because the union's reference side has to be
// expressible.
type KafkaMessageBinding struct {
	Key                     any    `json:"key,omitempty"                     yaml:"key,omitempty"`
	SchemaIDLocation        string `json:"schemaIdLocation,omitempty"        yaml:"schemaIdLocation,omitempty"`
	SchemaIDPayloadEncoding string `json:"schemaIdPayloadEncoding,omitempty" yaml:"schemaIdPayloadEncoding,omitempty"`
	SchemaLookupStrategy    string `json:"schemaLookupStrategy,omitempty"    yaml:"schemaLookupStrategy,omitempty"`
	BindingVersion          string `json:"bindingVersion,omitempty"          yaml:"bindingVersion,omitempty"`
}

// --- MQTT bindings (binding version 0.2.0) -----------------------------------
//
// The channel object is reserved: MQTTChannelBinding is an alias in the
// reserved bindings section.

// MQTTServerBinding describes MQTT-specific server information.
//
// SessionExpiryInterval and MaximumPacketSize are Schema Object | Reference
// Object | integer in the binding specification, so they are `any`.
type MQTTServerBinding struct {
	ClientID              string        `json:"clientId,omitempty"              yaml:"clientId,omitempty"`
	CleanSession          bool          `json:"cleanSession,omitempty"          yaml:"cleanSession,omitempty"`
	KeepAlive             int           `json:"keepAlive,omitempty"             yaml:"keepAlive,omitempty"`
	SessionExpiryInterval any           `json:"sessionExpiryInterval,omitempty" yaml:"sessionExpiryInterval,omitempty"`
	MaximumPacketSize     any           `json:"maximumPacketSize,omitempty"     yaml:"maximumPacketSize,omitempty"`
	LastWill              *MQTTLastWill `json:"lastWill,omitempty"              yaml:"lastWill,omitempty"`
	BindingVersion        string        `json:"bindingVersion,omitempty"        yaml:"bindingVersion,omitempty"`
}

// MQTTLastWill describes the MQTT last-will message.
type MQTTLastWill struct {
	Topic   string `json:"topic,omitempty"   yaml:"topic,omitempty"`
	QoS     int    `json:"qos,omitempty"     yaml:"qos,omitempty"`
	Message string `json:"message,omitempty" yaml:"message,omitempty"`
	Retain  bool   `json:"retain,omitempty"  yaml:"retain,omitempty"`
}

// MQTTOperationBinding describes MQTT-specific operation information.
// MessageExpiryInterval is Schema Object | Reference Object | integer.
type MQTTOperationBinding struct {
	QoS                   int    `json:"qos,omitempty"                   yaml:"qos,omitempty"`
	Retain                bool   `json:"retain,omitempty"                yaml:"retain,omitempty"`
	MessageExpiryInterval any    `json:"messageExpiryInterval,omitempty" yaml:"messageExpiryInterval,omitempty"`
	BindingVersion        string `json:"bindingVersion,omitempty"        yaml:"bindingVersion,omitempty"`
}

// MQTTMessageBinding describes MQTT-specific message information.
//
// CorrelationData is Schema Object | Reference Object, and ResponseTopic is
// string | Schema Object | Reference Object, so both are `any`.
type MQTTMessageBinding struct {
	PayloadFormatIndicator int    `json:"payloadFormatIndicator,omitempty" yaml:"payloadFormatIndicator,omitempty"`
	CorrelationData        any    `json:"correlationData,omitempty"        yaml:"correlationData,omitempty"`
	ContentType            string `json:"contentType,omitempty"            yaml:"contentType,omitempty"`
	ResponseTopic          any    `json:"responseTopic,omitempty"          yaml:"responseTopic,omitempty"`
	BindingVersion         string `json:"bindingVersion,omitempty"         yaml:"bindingVersion,omitempty"`
}

// --- MQTT 5 bindings (binding version 0.2.0) ---------------------------------

// MQTT5ServerBinding describes MQTT 5-specific server information.
// SessionExpiryInterval is Schema Object | Reference Object | integer, so it is
// `any`.
//
// The MQTT 5 binding is deprecated upstream in favour of the version-agnostic
// MQTT binding (spec.ProtocolMQTT), and it is not part of the published 3.1.0
// binding JSON Schemas: the specification defines only a server object for it,
// and a document using the key fails schema validation.
type MQTT5ServerBinding struct {
	SessionExpiryInterval any    `json:"sessionExpiryInterval,omitempty" yaml:"sessionExpiryInterval,omitempty"`
	BindingVersion        string `json:"bindingVersion,omitempty"        yaml:"bindingVersion,omitempty"`
}

// --- NATS bindings (binding version 0.1.0) -----------------------------------
//
// The server, channel, and message objects are reserved: they are aliases in the
// reserved bindings section. Only the operation object defines a field.

// NATSOperationBinding describes NATS-specific operation information.
type NATSOperationBinding struct {
	Queue          string `json:"queue,omitempty"          yaml:"queue,omitempty"`
	BindingVersion string `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}

// --- Pulsar bindings (binding version 0.1.0) ---------------------------------
//
// The specification defines only server and channel objects for Pulsar: the
// channel and operation bindings objects do not list a pulsar key, so there is
// no PulsarOperationBinding or PulsarMessageBinding to model.

// PulsarServerBinding describes Pulsar-specific server information.
type PulsarServerBinding struct {
	Tenant         string `json:"tenant,omitempty"         yaml:"tenant,omitempty"`
	BindingVersion string `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}

// PulsarChannelBinding describes Pulsar-specific channel information. Namespace
// and Persistence are required by the binding specification.
type PulsarChannelBinding struct {
	Namespace      string           `json:"namespace"                 yaml:"namespace"`
	Persistence    string           `json:"persistence"               yaml:"persistence"` // "persistent" | "non-persistent"
	Compaction     int              `json:"compaction,omitempty"      yaml:"compaction,omitempty"`
	GeoReplication []string         `json:"geo-replication,omitempty" yaml:"geo-replication,omitempty"`
	Retention      *PulsarRetention `json:"retention,omitempty"       yaml:"retention,omitempty"`
	TTL            int              `json:"ttl,omitempty"             yaml:"ttl,omitempty"`
	Deduplication  bool             `json:"deduplication,omitempty"   yaml:"deduplication,omitempty"`
	BindingVersion string           `json:"bindingVersion,omitempty"  yaml:"bindingVersion,omitempty"`
}

// PulsarRetention describes the retention policy of a Pulsar channel.
type PulsarRetention struct {
	Time int `json:"time,omitempty" yaml:"time,omitempty"`
	Size int `json:"size,omitempty" yaml:"size,omitempty"`
}

// --- ROS 2 bindings (binding version 0.1.0) ----------------------------------
//
// The specification defines only server and operation objects for ROS 2: the
// channel and message bindings objects do not list a ros2 key, so there is no
// ROS2ChannelBinding or ROS2MessageBinding to model.

// ROS2ServerBinding describes ROS 2-specific server information.
type ROS2ServerBinding struct {
	RMWImplementation string `json:"rmwImplementation,omitempty" yaml:"rmwImplementation,omitempty"`
	DomainID          int    `json:"domainId,omitempty"          yaml:"domainId,omitempty"`
	BindingVersion    string `json:"bindingVersion,omitempty"    yaml:"bindingVersion,omitempty"`
}

// ROS2OperationBinding describes ROS 2-specific operation information. Role and
// Node are required by the binding specification.
type ROS2OperationBinding struct {
	Role           string           `json:"role"                     yaml:"role"`
	Node           string           `json:"node"                     yaml:"node"`
	QoSPolicies    *ROS2QoSPolicies `json:"qosPolicies,omitempty"    yaml:"qosPolicies,omitempty"`
	BindingVersion string           `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}

// ROS2QoSPolicies describes the quality-of-service policies of a ROS 2
// operation.
type ROS2QoSPolicies struct {
	Deadline      int    `json:"deadline,omitempty"      yaml:"deadline,omitempty"`
	Durability    string `json:"durability,omitempty"    yaml:"durability,omitempty"`
	History       string `json:"history,omitempty"       yaml:"history,omitempty"`
	LeaseDuration int    `json:"leaseDuration,omitempty" yaml:"leaseDuration,omitempty"`
	Lifespan      int    `json:"lifespan,omitempty"      yaml:"lifespan,omitempty"`
	Liveliness    string `json:"liveliness,omitempty"    yaml:"liveliness,omitempty"`
	Reliability   string `json:"reliability,omitempty"   yaml:"reliability,omitempty"`
}

// --- SNS bindings (binding version 0.2.0) ------------------------------------
//
// The server and message objects are reserved: they are aliases in the
// reserved bindings section.

// SNSChannelBinding describes SNS-specific channel information. Name is
// required by the binding specification.
type SNSChannelBinding struct {
	Name           string         `json:"name"                     yaml:"name"`
	Ordering       *SNSOrdering   `json:"ordering,omitempty"       yaml:"ordering,omitempty"`
	Policy         *SNSPolicy     `json:"policy,omitempty"         yaml:"policy,omitempty"`
	Tags           map[string]any `json:"tags,omitempty"           yaml:"tags,omitempty"`
	BindingVersion string         `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}

// SNSOrdering describes the ordering of an SNS FIFO topic.
type SNSOrdering struct {
	Type                      string `json:"type"                                yaml:"type"`
	ContentBasedDeduplication bool   `json:"contentBasedDeduplication,omitempty" yaml:"contentBasedDeduplication,omitempty"`
}

// SNSPolicy describes the access policy of an SNS topic.
type SNSPolicy struct {
	Statements []SNSStatement `json:"statements" yaml:"statements"`
}

// SNSStatement is one statement of an SNS access policy. Principal, Action, and
// Resource are string | []string, and Principal may also be a map, so they are
// `any`.
type SNSStatement struct {
	Effect    string         `json:"effect"              yaml:"effect"`
	Principal any            `json:"principal"           yaml:"principal"`
	Action    any            `json:"action"              yaml:"action"`
	Resource  any            `json:"resource,omitempty"  yaml:"resource,omitempty"`
	Condition map[string]any `json:"condition,omitempty" yaml:"condition,omitempty"`
}

// SNSOperationBinding describes SNS-specific operation information. Consumers
// is required by the binding specification.
type SNSOperationBinding struct {
	Topic          *SNSIdentifier     `json:"topic,omitempty"          yaml:"topic,omitempty"`
	Consumers      []SNSConsumer      `json:"consumers"                yaml:"consumers"`
	DeliveryPolicy *SNSDeliveryPolicy `json:"deliveryPolicy,omitempty" yaml:"deliveryPolicy,omitempty"`
	BindingVersion string             `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}

// SNSIdentifier identifies an SNS topic or consumer endpoint. Exactly one of
// the fields is meaningful per endpoint kind.
type SNSIdentifier struct {
	URL   string `json:"url,omitempty"   yaml:"url,omitempty"`
	Email string `json:"email,omitempty" yaml:"email,omitempty"`
	Phone string `json:"phone,omitempty" yaml:"phone,omitempty"`
	ARN   string `json:"arn,omitempty"   yaml:"arn,omitempty"`
	Name  string `json:"name,omitempty"  yaml:"name,omitempty"`
}

// SNSConsumer describes one subscriber of an SNS topic. Protocol, Endpoint, and
// RawMessageDelivery are required by the binding specification.
type SNSConsumer struct {
	Protocol           string             `json:"protocol"                    yaml:"protocol"`
	Endpoint           *SNSIdentifier     `json:"endpoint"                    yaml:"endpoint"`
	FilterPolicy       map[string]any     `json:"filterPolicy,omitempty"      yaml:"filterPolicy,omitempty"`
	FilterPolicyScope  string             `json:"filterPolicyScope,omitempty" yaml:"filterPolicyScope,omitempty"`
	RawMessageDelivery bool               `json:"rawMessageDelivery"          yaml:"rawMessageDelivery"`
	RedrivePolicy      *SNSRedrivePolicy  `json:"redrivePolicy,omitempty"     yaml:"redrivePolicy,omitempty"`
	DeliveryPolicy     *SNSDeliveryPolicy `json:"deliveryPolicy,omitempty"    yaml:"deliveryPolicy,omitempty"`
	DisplayName        string             `json:"displayName,omitempty"       yaml:"displayName,omitempty"`
}

// SNSRedrivePolicy describes the dead-letter queue of an SNS consumer.
type SNSRedrivePolicy struct {
	DeadLetterQueue *SNSIdentifier `json:"deadLetterQueue"           yaml:"deadLetterQueue"`
	MaxReceiveCount int            `json:"maxReceiveCount,omitempty" yaml:"maxReceiveCount,omitempty"`
}

// SNSDeliveryPolicy describes the retry policy for HTTP consumers.
type SNSDeliveryPolicy struct {
	MinDelayTarget       int    `json:"minDelayTarget,omitempty"       yaml:"minDelayTarget,omitempty"`
	MaxDelayTarget       int    `json:"maxDelayTarget,omitempty"       yaml:"maxDelayTarget,omitempty"`
	NumRetries           int    `json:"numRetries,omitempty"           yaml:"numRetries,omitempty"`
	NumNoDelayRetries    int    `json:"numNoDelayRetries,omitempty"    yaml:"numNoDelayRetries,omitempty"`
	NumMinDelayRetries   int    `json:"numMinDelayRetries,omitempty"   yaml:"numMinDelayRetries,omitempty"`
	NumMaxDelayRetries   int    `json:"numMaxDelayRetries,omitempty"   yaml:"numMaxDelayRetries,omitempty"`
	BackoffFunction      string `json:"backoffFunction,omitempty"      yaml:"backoffFunction,omitempty"`
	MaxReceivesPerSecond int    `json:"maxReceivesPerSecond,omitempty" yaml:"maxReceivesPerSecond,omitempty"`
}

// --- Solace bindings (binding version 0.4.0) ---------------------------------
//
// The channel and message objects are reserved: they are aliases in the
// reserved bindings section.

// SolaceServerBinding describes Solace-specific server information.
type SolaceServerBinding struct {
	MsgVPN         string `json:"msgVpn,omitempty"         yaml:"msgVpn,omitempty"`
	ClientName     string `json:"clientName,omitempty"     yaml:"clientName,omitempty"`
	BindingVersion string `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}

// SolaceOperationBinding describes Solace-specific operation information.
//
// Destinations is untyped upstream (an array of arbitrary objects); it is
// modelled as []SolaceDestination, a stricter shape the schema still accepts.
type SolaceOperationBinding struct {
	Destinations   []SolaceDestination `json:"destinations,omitempty"   yaml:"destinations,omitempty"`
	TimeToLive     int                 `json:"timeToLive,omitempty"     yaml:"timeToLive,omitempty"`
	Priority       int                 `json:"priority,omitempty"       yaml:"priority,omitempty"`
	DMQEligible    bool                `json:"dmqEligible,omitempty"    yaml:"dmqEligible,omitempty"`
	BindingVersion string              `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}

// SolaceDestination is one queue or topic an operation targets.
type SolaceDestination struct {
	DestinationType string       `json:"destinationType,omitempty" yaml:"destinationType,omitempty"` // "queue" | "topic"
	Queue           *SolaceQueue `json:"queue,omitempty"           yaml:"queue,omitempty"`
	Topic           *SolaceTopic `json:"topic,omitempty"           yaml:"topic,omitempty"`
}

// SolaceQueue describes the queue a Solace operation targets.
type SolaceQueue struct {
	Name               string   `json:"name,omitempty"               yaml:"name,omitempty"`
	TopicSubscriptions []string `json:"topicSubscriptions,omitempty" yaml:"topicSubscriptions,omitempty"`
}

// SolaceTopic describes the topic a Solace operation targets.
type SolaceTopic struct {
	Name               string   `json:"name,omitempty"               yaml:"name,omitempty"`
	TopicSubscriptions []string `json:"topicSubscriptions,omitempty" yaml:"topicSubscriptions,omitempty"`
}

// --- SQS bindings (binding version 0.3.0) ------------------------------------
//
// The server and message objects are reserved: they are aliases in the
// reserved bindings section.

// SQSChannelBinding describes SQS-specific channel information. Queue is
// required by the binding specification.
type SQSChannelBinding struct {
	Queue           *SQSQueue `json:"queue"                     yaml:"queue"`
	DeadLetterQueue *SQSQueue `json:"deadLetterQueue,omitempty" yaml:"deadLetterQueue,omitempty"`
	BindingVersion  string    `json:"bindingVersion,omitempty"  yaml:"bindingVersion,omitempty"`
}

// SQSOperationBinding describes SQS-specific operation information. Queues is
// required by the binding specification.
type SQSOperationBinding struct {
	Queues         []SQSQueue `json:"queues"                   yaml:"queues"`
	BindingVersion string     `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}

// SQSQueue describes an SQS queue. Name and FifoQueue are required by the
// binding specification, so neither carries omitempty.
type SQSQueue struct {
	Name                   string            `json:"name"                             yaml:"name"`
	FifoQueue              bool              `json:"fifoQueue"                        yaml:"fifoQueue"`
	DeduplicationScope     string            `json:"deduplicationScope,omitempty"     yaml:"deduplicationScope,omitempty"`
	FifoThroughputLimit    string            `json:"fifoThroughputLimit,omitempty"    yaml:"fifoThroughputLimit,omitempty"`
	DeliveryDelay          int               `json:"deliveryDelay,omitempty"          yaml:"deliveryDelay,omitempty"`
	VisibilityTimeout      int               `json:"visibilityTimeout,omitempty"      yaml:"visibilityTimeout,omitempty"`
	ReceiveMessageWaitTime int               `json:"receiveMessageWaitTime,omitempty" yaml:"receiveMessageWaitTime,omitempty"`
	MessageRetentionPeriod int               `json:"messageRetentionPeriod,omitempty" yaml:"messageRetentionPeriod,omitempty"`
	RedrivePolicy          *SQSRedrivePolicy `json:"redrivePolicy,omitempty"          yaml:"redrivePolicy,omitempty"`
	Policy                 *SQSPolicy        `json:"policy,omitempty"                 yaml:"policy,omitempty"`
	Tags                   map[string]string `json:"tags,omitempty"                   yaml:"tags,omitempty"`
}

// SQSRedrivePolicy describes the dead-letter queue of an SQS queue.
type SQSRedrivePolicy struct {
	DeadLetterQueue *SQSIdentifier `json:"deadLetterQueue"           yaml:"deadLetterQueue"`
	MaxReceiveCount int            `json:"maxReceiveCount,omitempty" yaml:"maxReceiveCount,omitempty"`
}

// SQSIdentifier identifies an SQS queue by ARN or by name.
type SQSIdentifier struct {
	ARN  string `json:"arn,omitempty"  yaml:"arn,omitempty"`
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
}

// SQSPolicy describes the access policy of an SQS queue.
type SQSPolicy struct {
	Statements []SQSStatement `json:"statements" yaml:"statements"`
}

// SQSStatement is one statement of an SQS access policy. Principal, Action, and
// Resource are string | []string, and Principal may also be a map, so they are
// `any`.
type SQSStatement struct {
	Effect    string         `json:"effect"              yaml:"effect"`
	Principal any            `json:"principal"           yaml:"principal"`
	Action    any            `json:"action"              yaml:"action"`
	Resource  any            `json:"resource,omitempty"  yaml:"resource,omitempty"`
	Condition map[string]any `json:"condition,omitempty" yaml:"condition,omitempty"`
}

// --- WebSockets bindings (binding version 0.1.0) -----------------------------
//
// The server, operation, and message objects are reserved: they are aliases in the
// reserved bindings section. Only the channel object defines fields.

// WSChannelBinding describes WebSockets-specific channel information. Query and
// Headers are Schema Object | Reference Object, so they are `any`.
type WSChannelBinding struct {
	Method         string `json:"method,omitempty"         yaml:"method,omitempty"`
	Query          any    `json:"query,omitempty"          yaml:"query,omitempty"`
	Headers        any    `json:"headers,omitempty"        yaml:"headers,omitempty"`
	BindingVersion string `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}
