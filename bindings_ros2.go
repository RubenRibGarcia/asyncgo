package asyncgo

import "github.com/RubenRibGarcia/asyncgo/spec"

// --- ROS 2 bindings ----------------------------------------------------------
//
// The specification defines only server and operation objects for ROS 2, so
// there is no channel or message helper.

// ROS2 attaches a ROS 2 server binding.
func (s *server) ROS2(b spec.ROS2ServerBinding) *server {
	return s.setBinding(spec.ProtocolROS2, &b)
}

// ROS2 attaches a ROS 2 operation binding.
func (o *operation) ROS2(b spec.ROS2OperationBinding) *operation {
	return o.setBinding(spec.ProtocolROS2, &b)
}

// ROS2 attaches a ROS 2 operation trait binding.
func (t *operationTrait) ROS2(b spec.ROS2OperationBinding) *operationTrait {
	return t.setBinding(spec.ProtocolROS2, &b)
}
