package spec

import (
	"fmt"
	"reflect"
	"sort"
)

// bindingObjects names the binding object each document position accepts for one
// protocol. A nil field means the specification defines no binding object for
// that position (mqtt5 has only a server binding, Pulsar no operation or message
// binding, ROS2 no channel or message binding), so only the key is checked there.
type bindingObjects struct {
	server    reflect.Type
	channel   reflect.Type
	operation reflect.Type
	message   reflect.Type
}

// bindingObjectTypes maps each modelled protocol key to the binding object type
// accepted at each position.
//
// The operation and message columns also govern the two trait positions: the
// specification defines an operation trait's bindings as operation bindings and a
// message trait's as message bindings, which is why the DSL reuses the same
// structs there and the table needs no separate columns.
//
// Many of these names are aliases of ReservedBinding (an object the specification
// defines but reserves, carrying only bindingVersion), so they are the same
// reflect.Type. A type mismatch there reports the underlying spec.ReservedBinding,
// not the protocol-specific alias.
var bindingObjectTypes = map[string]bindingObjects{
	ProtocolAMQP: {
		server:    reflect.TypeFor[AMQPServerBinding](),
		channel:   reflect.TypeFor[AMQPChannelBinding](),
		operation: reflect.TypeFor[AMQPOperationBinding](),
		message:   reflect.TypeFor[AMQPMessageBinding](),
	},
	ProtocolAMQP1: {
		server:    reflect.TypeFor[AMQP1ServerBinding](),
		channel:   reflect.TypeFor[AMQP1ChannelBinding](),
		operation: reflect.TypeFor[AMQP1OperationBinding](),
		message:   reflect.TypeFor[AMQP1MessageBinding](),
	},
	ProtocolAnypointMQ: {
		server:    reflect.TypeFor[AnypointMQServerBinding](),
		channel:   reflect.TypeFor[AnypointMQChannelBinding](),
		operation: reflect.TypeFor[AnypointMQOperationBinding](),
		message:   reflect.TypeFor[AnypointMQMessageBinding](),
	},
	ProtocolGooglePubSub: {
		server:    reflect.TypeFor[GooglePubSubServerBinding](),
		channel:   reflect.TypeFor[GooglePubSubChannelBinding](),
		operation: reflect.TypeFor[GooglePubSubOperationBinding](),
		message:   reflect.TypeFor[GooglePubSubMessageBinding](),
	},
	ProtocolHTTP: {
		server:    reflect.TypeFor[HTTPServerBinding](),
		channel:   reflect.TypeFor[HTTPChannelBinding](),
		operation: reflect.TypeFor[HTTPOperationBinding](),
		message:   reflect.TypeFor[HTTPMessageBinding](),
	},
	ProtocolIBMMQ: {
		server:    reflect.TypeFor[IBMMQServerBinding](),
		channel:   reflect.TypeFor[IBMMQChannelBinding](),
		operation: reflect.TypeFor[IBMMQOperationBinding](),
		message:   reflect.TypeFor[IBMMQMessageBinding](),
	},
	ProtocolJMS: {
		server:    reflect.TypeFor[JMSServerBinding](),
		channel:   reflect.TypeFor[JMSChannelBinding](),
		operation: reflect.TypeFor[JMSOperationBinding](),
		message:   reflect.TypeFor[JMSMessageBinding](),
	},
	ProtocolKafka: {
		server:    reflect.TypeFor[KafkaServerBinding](),
		channel:   reflect.TypeFor[KafkaChannelBinding](),
		operation: reflect.TypeFor[KafkaOperationBinding](),
		message:   reflect.TypeFor[KafkaMessageBinding](),
	},
	ProtocolMQTT: {
		server:    reflect.TypeFor[MQTTServerBinding](),
		channel:   reflect.TypeFor[MQTTChannelBinding](),
		operation: reflect.TypeFor[MQTTOperationBinding](),
		message:   reflect.TypeFor[MQTTMessageBinding](),
	},
	ProtocolMQTT5: {
		server:    reflect.TypeFor[MQTT5ServerBinding](),
		channel:   nil,
		operation: nil,
		message:   nil,
	},
	ProtocolMercure: {
		server:    reflect.TypeFor[MercureServerBinding](),
		channel:   reflect.TypeFor[MercureChannelBinding](),
		operation: reflect.TypeFor[MercureOperationBinding](),
		message:   reflect.TypeFor[MercureMessageBinding](),
	},
	ProtocolNATS: {
		server:    reflect.TypeFor[NATSServerBinding](),
		channel:   reflect.TypeFor[NATSChannelBinding](),
		operation: reflect.TypeFor[NATSOperationBinding](),
		message:   reflect.TypeFor[NATSMessageBinding](),
	},
	ProtocolPulsar: {
		server:    reflect.TypeFor[PulsarServerBinding](),
		channel:   reflect.TypeFor[PulsarChannelBinding](),
		operation: nil,
		message:   nil,
	},
	ProtocolROS2: {
		server:    reflect.TypeFor[ROS2ServerBinding](),
		channel:   nil,
		operation: reflect.TypeFor[ROS2OperationBinding](),
		message:   nil,
	},
	ProtocolRedis: {
		server:    reflect.TypeFor[RedisServerBinding](),
		channel:   reflect.TypeFor[RedisChannelBinding](),
		operation: reflect.TypeFor[RedisOperationBinding](),
		message:   reflect.TypeFor[RedisMessageBinding](),
	},
	ProtocolSNS: {
		server:    reflect.TypeFor[SNSServerBinding](),
		channel:   reflect.TypeFor[SNSChannelBinding](),
		operation: reflect.TypeFor[SNSOperationBinding](),
		message:   reflect.TypeFor[SNSMessageBinding](),
	},
	ProtocolSQS: {
		server:    reflect.TypeFor[SQSServerBinding](),
		channel:   reflect.TypeFor[SQSChannelBinding](),
		operation: reflect.TypeFor[SQSOperationBinding](),
		message:   reflect.TypeFor[SQSMessageBinding](),
	},
	ProtocolSolace: {
		server:    reflect.TypeFor[SolaceServerBinding](),
		channel:   reflect.TypeFor[SolaceChannelBinding](),
		operation: reflect.TypeFor[SolaceOperationBinding](),
		message:   reflect.TypeFor[SolaceMessageBinding](),
	},
	ProtocolStomp: {
		server:    reflect.TypeFor[StompServerBinding](),
		channel:   reflect.TypeFor[StompChannelBinding](),
		operation: reflect.TypeFor[StompOperationBinding](),
		message:   reflect.TypeFor[StompMessageBinding](),
	},
	ProtocolWS: {
		server:    reflect.TypeFor[WSServerBinding](),
		channel:   reflect.TypeFor[WSChannelBinding](),
		operation: reflect.TypeFor[WSOperationBinding](),
		message:   reflect.TypeFor[WSMessageBinding](),
	},
}

// Validate reports the server bindings whose value is not the binding object the
// specification defines for that protocol.
//
// A key this package does not model is left alone: the untyped Binding escape
// hatch is how a protocol added to the specification before asyncgo models it is
// expressed, and there is no expected type to compare against. A modelled
// protocol with no binding object at this position (mqtt5) is likewise unchecked.
//
// The bindings map stays map[string]any because the decode path needs any; this
// walk is how the authored side recovers the typing it erases.
func (b ServerBindings) Validate() []error {
	return validateBindings(b, func(o bindingObjects) reflect.Type { return o.server })
}

// Validate reports the channel bindings whose value is not the binding object the
// specification defines for that protocol. See ServerBindings.Validate for the
// treatment of unmodelled protocols.
func (b ChannelBindings) Validate() []error {
	return validateBindings(b, func(o bindingObjects) reflect.Type { return o.channel })
}

// Validate reports the operation (and operation trait) bindings whose value is
// not the binding object the specification defines for that protocol. See
// ServerBindings.Validate for the treatment of unmodelled protocols.
func (b OperationBindings) Validate() []error {
	return validateBindings(b, func(o bindingObjects) reflect.Type { return o.operation })
}

// Validate reports the message (and message trait) bindings whose value is not
// the binding object the specification defines for that protocol. See
// ServerBindings.Validate for the treatment of unmodelled protocols.
func (b MessageBindings) Validate() []error {
	return validateBindings(b, func(o bindingObjects) reflect.Type { return o.message })
}

// validateBindings walks m in sorted key order so the joined message is stable,
// and reports every entry whose value does not match want(protocol).
func validateBindings(m map[string]any, want func(bindingObjects) reflect.Type) []error {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var errs []error
	for _, key := range keys {
		objects, modelled := bindingObjectTypes[key]
		if !modelled {
			continue
		}
		expected := want(objects)
		if expected == nil {
			continue // the specification defines no binding object here
		}
		if err := checkBindingValue(expected, m[key]); err != nil {
			errs = append(errs, fmt.Errorf("bindings.%s: %w", key, err))
		}
	}
	return errs
}

// checkBindingValue reports whether v is want or *want.
func checkBindingValue(want reflect.Type, v any) error {
	if v == nil {
		return fmt.Errorf("is nil; want %s", want)
	}
	t := reflect.TypeOf(v)
	if t == want || (t.Kind() == reflect.Pointer && t.Elem() == want) {
		return nil
	}
	return fmt.Errorf("want %s, got %s", want, t)
}
