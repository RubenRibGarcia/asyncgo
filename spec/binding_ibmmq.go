package spec

// --- IBM MQ bindings (binding version 0.1.0) --------------------------------
//
// The operation object is reserved: IBMMQOperationBinding is an alias in
// binding_reserved.go.

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
