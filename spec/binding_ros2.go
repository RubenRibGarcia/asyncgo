package spec

// --- ROS 2 bindings (binding version 0.1.0) ---------------------------------
//
// The specification defines only server and operation objects for ROS 2: the
// channel and message bindings objects do not list a ros2 key, so there is no
// ROS2ChannelBinding or ROS2MessageBinding to model.

// ROS2ServerBinding describes ROS 2-specific server information.
type ROS2ServerBinding struct {
	RMWImplementation string `json:"rmwImplementation,omitempty" yaml:"rmwImplementation,omitempty"`
	DomainID          int    `json:"domainId,omitempty"          yaml:"domainId,omitempty"`
	BindingVersion    string `json:"bindingVersion,omitempty"    yaml:"bindingVersion,omitempty"`
}

// ROS2OperationBinding describes ROS 2-specific operation information. Role and
// Node are required by the binding specification.
type ROS2OperationBinding struct {
	Role           string           `json:"role"                     yaml:"role"`
	Node           string           `json:"node"                     yaml:"node"`
	QoSPolicies    *ROS2QoSPolicies `json:"qosPolicies,omitempty"    yaml:"qosPolicies,omitempty"`
	BindingVersion string           `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}

// ROS2QoSPolicies describes the quality-of-service policies of a ROS 2
// operation.
type ROS2QoSPolicies struct {
	Deadline      int    `json:"deadline,omitempty"      yaml:"deadline,omitempty"`
	Durability    string `json:"durability,omitempty"    yaml:"durability,omitempty"`
	History       string `json:"history,omitempty"       yaml:"history,omitempty"`
	LeaseDuration int    `json:"leaseDuration,omitempty" yaml:"leaseDuration,omitempty"`
	Lifespan      int    `json:"lifespan,omitempty"      yaml:"lifespan,omitempty"`
	Liveliness    string `json:"liveliness,omitempty"    yaml:"liveliness,omitempty"`
	Reliability   string `json:"reliability,omitempty"   yaml:"reliability,omitempty"`
}
