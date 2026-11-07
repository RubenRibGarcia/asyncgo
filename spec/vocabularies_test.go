package spec

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestVocabularyConstantValues pins the wire strings. The point of naming these
// enum values is that the exact spelling is load-bearing — an SNS ordering type
// is "FIFO", not "fifo" — so a rename that still compiles has to fail here.
func TestVocabularyConstantValues(t *testing.T) {
	tests := []struct {
		name string
		want string
		got  string
	}{
		{"security_scheme_user_password", "userPassword", SecuritySchemeTypeUserPassword},
		{"security_scheme_api_key", "apiKey", SecuritySchemeTypeAPIKey},
		{"security_scheme_http", "http", SecuritySchemeTypeHTTP},
		{"security_scheme_oauth2", "oauth2", SecuritySchemeTypeOAuth2},
		{"security_scheme_openid_connect", "openIdConnect", SecuritySchemeTypeOpenIDConnect},
		{"amqp_is_queue", "queue", AMQPChannelBindingIsQueue},
		{"amqp_is_routing_key", "routingKey", AMQPChannelBindingIsRoutingKey},
		{"pulsar_persistent", "persistent", PulsarChannelBindingPersistencePersistent},
		{"pulsar_non_persistent", "non-persistent", PulsarChannelBindingPersistenceNonPersistent},
		{"ibmmq_destination_topic", "topic", IBMMQDestinationTypeTopic},
		{"ibmmq_destination_queue", "queue", IBMMQDestinationTypeQueue},
		{"sns_ordering_standard", "standard", SNSOrderingTypeStandard},
		{"sns_ordering_fifo", "FIFO", SNSOrderingTypeFIFO},
		{"ros2_role_publisher", "publisher", ROS2RolePublisher},
		{"ros2_role_subscriber", "subscriber", ROS2RoleSubscriber},
		{"ros2_role_action_client", "action_client", ROS2RoleActionClient},
		{"ros2_role_action_server", "action_server", ROS2RoleActionServer},
		{"ros2_role_service_client", "service_client", ROS2RoleServiceClient},
		{"ros2_role_service_server", "service_server", ROS2RoleServiceServer},
	}
	for _, tc := range tests {
		t.Run("should_keep_the_wire_value_"+tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.got)
		})
	}
}
