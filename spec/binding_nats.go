package spec

// --- NATS bindings (binding version 0.1.0) ----------------------------------
//
// The server, channel, and message objects are reserved: they are aliases in
// binding_reserved.go. Only the operation object defines a field.

// NATSOperationBinding describes NATS-specific operation information.
type NATSOperationBinding struct {
	Queue          string `json:"queue,omitempty"          yaml:"queue,omitempty"`
	BindingVersion string `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}
