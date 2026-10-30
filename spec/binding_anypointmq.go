package spec

// --- Anypoint MQ bindings (binding version 0.0.1) ---------------------------
//
// The server and operation objects are reserved: they are aliases in
// binding_reserved.go.

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
