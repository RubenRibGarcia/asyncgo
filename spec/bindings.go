package spec

// Protocol identifiers used as keys in the *Bindings maps.
const (
	ProtocolKafka = "kafka"
	ProtocolAMQP  = "amqp"
	ProtocolNATS  = "nats"
	ProtocolMQTT  = "mqtt"
	ProtocolHTTP  = "http"
)

// The *Bindings types are protocol-keyed maps. Values are typed binding
// structs (see below); using a map keeps the model extensible to new protocols
// without changing the core object model.
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

// --- Kafka bindings ---------------------------------------------------------

// KafkaServerBinding describes Kafka-specific server information.
type KafkaServerBinding struct {
	SchemaRegistryURL    string `json:"schemaRegistryUrl,omitempty"    yaml:"schemaRegistryUrl,omitempty"`
	SchemaRegistryVendor string `json:"schemaRegistryVendor,omitempty" yaml:"schemaRegistryVendor,omitempty"`
	BindingVersion       string `json:"bindingVersion,omitempty"       yaml:"bindingVersion,omitempty"`
}

// KafkaChannelBinding describes Kafka-specific channel information.
type KafkaChannelBinding struct {
	Topic              string         `json:"topic,omitempty"              yaml:"topic,omitempty"`
	Partitions         int            `json:"partitions,omitempty"         yaml:"partitions,omitempty"`
	Replicas           int            `json:"replicas,omitempty"           yaml:"replicas,omitempty"`
	TopicConfiguration map[string]any `json:"topicConfiguration,omitempty" yaml:"topicConfiguration,omitempty"`
	BindingVersion     string         `json:"bindingVersion,omitempty"     yaml:"bindingVersion,omitempty"`
}

// KafkaOperationBinding describes Kafka-specific operation information.
type KafkaOperationBinding struct {
	GroupID        string `json:"groupId,omitempty"        yaml:"groupId,omitempty"`
	ClientID       string `json:"clientId,omitempty"       yaml:"clientId,omitempty"`
	BindingVersion string `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}

// KafkaMessageBinding describes Kafka-specific message information.
type KafkaMessageBinding struct {
	Key              string `json:"key,omitempty"              yaml:"key,omitempty"`
	SchemaIDLocation string `json:"schemaIdLocation,omitempty" yaml:"schemaIdLocation,omitempty"`
	BindingVersion   string `json:"bindingVersion,omitempty"   yaml:"bindingVersion,omitempty"`
}

// --- AMQP (RabbitMQ) bindings -------------------------------------------------

// AMQPServerBinding describes AMQP-specific server information.
type AMQPServerBinding struct {
	BindingVersion string `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}

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

// AMQPOperationBinding describes AMQP-specific operation information.
type AMQPOperationBinding struct {
	Expiration     int      `json:"expiration,omitempty"     yaml:"expiration,omitempty"`
	UserID         string   `json:"userId,omitempty"         yaml:"userId,omitempty"`
	CC             []string `json:"cc,omitempty"             yaml:"cc,omitempty"`
	Priority       int      `json:"priority,omitempty"       yaml:"priority,omitempty"`
	DeliveryMode   int      `json:"deliveryMode,omitempty"   yaml:"deliveryMode,omitempty"`
	Mandatory      bool     `json:"mandatory,omitempty"      yaml:"mandatory,omitempty"`
	BCC            []string `json:"bcc,omitempty"            yaml:"bcc,omitempty"`
	ReplyTo        string   `json:"replyTo,omitempty"        yaml:"replyTo,omitempty"`
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

// --- NATS bindings ------------------------------------------------------------

// NATSServerBinding describes NATS-specific server information.
type NATSServerBinding struct {
	BindingVersion string `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}

// NATSChannelBinding describes NATS-specific channel information.
type NATSChannelBinding struct {
	BindingVersion string `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}

// NATSOperationBinding describes NATS-specific operation information.
type NATSOperationBinding struct {
	Queue          string `json:"queue,omitempty"          yaml:"queue,omitempty"`
	BindingVersion string `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}

// NATSMessageBinding describes NATS-specific message information.
type NATSMessageBinding struct {
	BindingVersion string `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}

// --- MQTT bindings ------------------------------------------------------------

// MQTTServerBinding describes MQTT-specific server information.
type MQTTServerBinding struct {
	ClientID              string        `json:"clientId,omitempty"              yaml:"clientId,omitempty"`
	CleanSession          bool          `json:"cleanSession,omitempty"          yaml:"cleanSession,omitempty"`
	KeepAlive             int           `json:"keepAlive,omitempty"             yaml:"keepAlive,omitempty"`
	SessionExpiryInterval int           `json:"sessionExpiryInterval,omitempty" yaml:"sessionExpiryInterval,omitempty"`
	MaximumPacketSize     int           `json:"maximumPacketSize,omitempty"     yaml:"maximumPacketSize,omitempty"`
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

// MQTTChannelBinding describes MQTT-specific channel information.
type MQTTChannelBinding struct {
	BindingVersion string `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}

// MQTTOperationBinding describes MQTT-specific operation information.
type MQTTOperationBinding struct {
	QoS                   int    `json:"qos,omitempty"                   yaml:"qos,omitempty"`
	Retain                bool   `json:"retain,omitempty"                yaml:"retain,omitempty"`
	MessageExpiryInterval int    `json:"messageExpiryInterval,omitempty" yaml:"messageExpiryInterval,omitempty"`
	BindingVersion        string `json:"bindingVersion,omitempty"        yaml:"bindingVersion,omitempty"`
}

// MQTTMessageBinding describes MQTT-specific message information.
type MQTTMessageBinding struct {
	BindingVersion string `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}
