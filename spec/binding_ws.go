package spec

// --- WebSockets bindings (binding version 0.1.0) ----------------------------
//
// The server, operation, and message objects are reserved: they are aliases in
// binding_reserved.go. Only the channel object defines fields.

// WSChannelBinding describes WebSockets-specific channel information. Query and
// Headers are Schema Object | Reference Object, so they are `any`.
type WSChannelBinding struct {
	Method         string `json:"method,omitempty"         yaml:"method,omitempty"`
	Query          any    `json:"query,omitempty"          yaml:"query,omitempty"`
	Headers        any    `json:"headers,omitempty"        yaml:"headers,omitempty"`
	BindingVersion string `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}
