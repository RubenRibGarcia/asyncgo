package spec

// --- MQTT bindings (binding version 0.2.0) ----------------------------------
//
// The channel object is reserved: MQTTChannelBinding is an alias in
// binding_reserved.go.

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
