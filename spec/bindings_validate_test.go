package spec

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerBindingsValidate(t *testing.T) {
	tests := []struct {
		name    string
		binding any
		wantErr string
	}{
		{name: "should_accept_the_modelled_binding_object", binding: KafkaServerBinding{}},
		{
			name:    "should_accept_a_pointer_to_the_modelled_binding_object",
			binding: &KafkaServerBinding{},
		},
		{
			name:    "should_reject_another_protocols_binding_object",
			binding: KafkaChannelBinding{},
			wantErr: "bindings.kafka: want spec.KafkaServerBinding, got spec.KafkaChannelBinding",
		},
		{
			name:    "should_reject_a_binding_object_for_another_position",
			binding: KafkaOperationBinding{},
			wantErr: "bindings.kafka: want spec.KafkaServerBinding, got spec.KafkaOperationBinding",
		},
		{
			name:    "should_reject_an_arbitrary_value",
			binding: "kafka",
			wantErr: "bindings.kafka: want spec.KafkaServerBinding, got string",
		},
		{
			name:    "should_reject_a_nil_value",
			binding: nil,
			wantErr: "bindings.kafka: is nil; want spec.KafkaServerBinding",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := ServerBindings{ProtocolKafka: tc.binding}

			errs := b.Validate()
			if tc.wantErr == "" {
				assert.Empty(t, errs)
				return
			}
			require.Len(t, errs, 1)
			assert.EqualError(t, errs[0], tc.wantErr)
		})
	}
}

// TestValidateLeavesUnmodelledEntriesAlone pins the escape hatch: the untyped
// Binding helper is how a protocol added to the specification before asyncgo
// models it is expressed, so a key outside the table has no expected type and is
// not reported. A modelled protocol with no binding object at that position is
// likewise unchecked.
func TestValidateLeavesUnmodelledEntriesAlone(t *testing.T) {
	tests := []struct {
		name  string
		check func() []error
	}{
		{
			name: "should_accept_an_unknown_protocol_key_with_an_arbitrary_value",
			check: func() []error {
				return ServerBindings{"some-new-protocol": "value"}.Validate()
			},
		},
		{
			name: "should_accept_an_unknown_protocol_key_with_a_foreign_struct",
			check: func() []error {
				return ServerBindings{"some-new-protocol": struct{ X int }{}}.Validate()
			},
		},
		{
			name: "should_accept_a_modelled_protocol_without_a_binding_object_here",
			check: func() []error {
				// mqtt5 defines only a server binding.
				return ChannelBindings{ProtocolMQTT5: ReservedBinding{}}.Validate()
			},
		},
		{
			name: "should_accept_an_empty_map",
			check: func() []error {
				return MessageBindings(nil).Validate()
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Empty(t, tc.check())
		})
	}
}

func TestChannelOperationAndMessageBindingsValidate(t *testing.T) {
	tests := []struct {
		name    string
		check   func() []error
		wantErr string
	}{
		{
			name:  "should_accept_a_channel_binding",
			check: func() []error { return ChannelBindings{ProtocolMQTT: MQTTChannelBinding{}}.Validate() },
		},
		{
			name: "should_reject_a_server_binding_in_a_channel",
			check: func() []error {
				return ChannelBindings{ProtocolMQTT: MQTTServerBinding{}}.Validate()
			},
			// MQTTChannelBinding is an alias of ReservedBinding, so the expected type
			// in the message is the underlying one.
			wantErr: "bindings.mqtt: want spec.ReservedBinding, got spec.MQTTServerBinding",
		},
		{
			name:  "should_accept_an_operation_binding",
			check: func() []error { return OperationBindings{ProtocolKafka: KafkaOperationBinding{}}.Validate() },
		},
		{
			name:    "should_reject_an_operation_trait_receiving_a_channel_binding",
			check:   func() []error { return OperationBindings{ProtocolKafka: KafkaChannelBinding{}}.Validate() },
			wantErr: "bindings.kafka: want spec.KafkaOperationBinding, got spec.KafkaChannelBinding",
		},
		{
			name:  "should_accept_a_message_binding",
			check: func() []error { return MessageBindings{ProtocolKafka: KafkaMessageBinding{}}.Validate() },
		},
		{
			name:    "should_reject_a_message_binding_in_a_server",
			check:   func() []error { return ServerBindings{ProtocolKafka: KafkaMessageBinding{}}.Validate() },
			wantErr: "bindings.kafka: want spec.KafkaServerBinding, got spec.KafkaMessageBinding",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := tc.check()
			if tc.wantErr == "" {
				assert.Empty(t, errs)
				return
			}
			require.Len(t, errs, 1)
			assert.EqualError(t, errs[0], tc.wantErr)
		})
	}
}

func TestValidateReportsEveryFailureInKeyOrder(t *testing.T) {
	b := ServerBindings{
		ProtocolKafka: KafkaChannelBinding{},
		ProtocolMQTT:  KafkaServerBinding{},
	}

	errs := b.Validate()

	require.Len(t, errs, 2)
	assert.EqualError(
		t,
		errs[0],
		"bindings.kafka: want spec.KafkaServerBinding, got spec.KafkaChannelBinding",
	)
	assert.EqualError(
		t,
		errs[1],
		"bindings.mqtt: want spec.MQTTServerBinding, got spec.KafkaServerBinding",
	)
}
