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
// structs (see binding_*.go); using a map keeps the model extensible to new
// protocols without changing the core object model.
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
