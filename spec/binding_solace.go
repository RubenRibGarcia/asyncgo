package spec

// --- Solace bindings (binding version 0.4.0) --------------------------------
//
// The channel and message objects are reserved: they are aliases in
// binding_reserved.go.

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
