package spec

// --- JMS bindings (binding version 0.0.1) -----------------------------------
//
// The operation object is reserved: JMSOperationBinding is an alias in
// binding_reserved.go.

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
