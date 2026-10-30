package spec

// --- MQTT 5 bindings (binding version 0.2.0) --------------------------------

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
