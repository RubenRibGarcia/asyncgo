package spec

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// bindingValues lists every typed binding struct with all of its fields set, so
// the assertions below see every tag. Structs whose fields are all zero-valued
// by design (the reserved objects) appear once, because they share one type.
var bindingValues = []struct {
	name  string
	value any
}{
	{
		name: "server/kafka",
		value: KafkaServerBinding{
			SchemaRegistryURL:    "http://registry",
			SchemaRegistryVendor: "confluent",
			BindingVersion:       "0.5.0",
		},
	},
	{name: "server/amqp", value: AMQPServerBinding{BindingVersion: "0.3.0"}},
	{name: "server/nats", value: NATSServerBinding{BindingVersion: "0.1.0"}},
	{
		name: "server/mqtt",
		value: MQTTServerBinding{
			ClientID:              "client",
			CleanSession:          true,
			KeepAlive:             60,
			SessionExpiryInterval: 30,
			MaximumPacketSize:     1024,
			LastWill:              &MQTTLastWill{Topic: "t", QoS: 1, Message: "m", Retain: true},
			BindingVersion:        "0.2.0",
		},
	},
	{name: "server/http", value: HTTPServerBinding{BindingVersion: "0.3.0"}},
	{name: "server/ws", value: WSServerBinding{BindingVersion: "0.1.0"}},
	{name: "server/amqp1", value: AMQP1ServerBinding{BindingVersion: "0.1.0"}},
	{
		name:  "server/mqtt5",
		value: MQTT5ServerBinding{SessionExpiryInterval: 30, BindingVersion: "0.2.0"},
	},
	{name: "server/anypointmq", value: AnypointMQServerBinding{BindingVersion: "0.0.1"}},
	{
		name: "server/jms",
		value: JMSServerBinding{
			JMSConnectionFactory: "cf",
			Properties:           []JMSProperty{{Name: "p", Value: "v"}},
			ClientID:             "client",
			BindingVersion:       "0.0.1",
		},
	},
	{name: "server/sns", value: SNSServerBinding{BindingVersion: "0.2.0"}},
	{
		name:  "server/solace",
		value: SolaceServerBinding{MsgVPN: "vpn", ClientName: "client", BindingVersion: "0.4.0"},
	},
	{name: "server/sqs", value: SQSServerBinding{BindingVersion: "0.3.0"}},
	{name: "server/stomp", value: StompServerBinding{BindingVersion: "0.1.0"}},
	{name: "server/redis", value: RedisServerBinding{BindingVersion: "0.1.0"}},
	{name: "server/mercure", value: MercureServerBinding{BindingVersion: "0.1.0"}},
	{
		name: "server/ibmmq",
		value: IBMMQServerBinding{
			GroupID:              "g",
			CCDTQueueManagerName: "qm",
			CipherSpec:           "ANY_TLS12",
			MultiEndpointServer:  true,
			HeartBeatInterval:    300,
			BindingVersion:       "0.1.0",
		},
	},
	{name: "server/googlepubsub", value: GooglePubSubServerBinding{BindingVersion: "0.2.0"}},
	{name: "server/pulsar", value: PulsarServerBinding{Tenant: "tenant", BindingVersion: "0.1.0"}},
	{
		name: "server/ros2",
		value: ROS2ServerBinding{
			RMWImplementation: "rmw_fastrtps_cpp",
			DomainID:          1,
			BindingVersion:    "0.1.0",
		},
	},
	{
		name: "channel/kafka",
		value: KafkaChannelBinding{
			Topic:      "orders",
			Partitions: 3,
			Replicas:   1,
			TopicConfiguration: &KafkaTopicConfiguration{
				CleanupPolicy: []string{"delete"},
				RetentionMs:   1000,
			},
			BindingVersion: "0.5.0",
		},
	},
	{
		name: "channel/amqp",
		value: AMQPChannelBinding{
			Is: "queue",
			Exchange: &AMQPExchange{
				Name:       "e",
				Type:       "topic",
				Durable:    true,
				AutoDelete: false,
				VHost:      "/",
			},
			Queue: &AMQPQueue{
				Name:       "q",
				Durable:    true,
				Exclusive:  false,
				AutoDelete: false,
				VHost:      "/",
			},
			BindingVersion: "0.3.0",
		},
	},
	{name: "channel/nats", value: NATSChannelBinding{BindingVersion: "0.1.0"}},
	{name: "channel/mqtt", value: MQTTChannelBinding{BindingVersion: "0.2.0"}},
	{name: "channel/http", value: HTTPChannelBinding{BindingVersion: "0.3.0"}},
	{
		name: "channel/ws",
		value: WSChannelBinding{
			Method:         "GET",
			Query:          &Schema{Type: "object"},
			Headers:        &Schema{Type: "object"},
			BindingVersion: "0.1.0",
		},
	},
	{name: "channel/amqp1", value: AMQP1ChannelBinding{BindingVersion: "0.1.0"}},
	{
		name: "channel/anypointmq",
		value: AnypointMQChannelBinding{
			Destination:     "d",
			DestinationType: "queue",
			BindingVersion:  "0.0.1",
		},
	},
	{
		name: "channel/jms",
		value: JMSChannelBinding{
			Destination:     "d",
			DestinationType: "fifo-queue",
			BindingVersion:  "0.0.1",
		},
	},
	{
		name: "channel/sns",
		value: SNSChannelBinding{
			Name:     "orders",
			Ordering: &SNSOrdering{Type: "FIFO", ContentBasedDeduplication: true},
			Policy: &SNSPolicy{
				Statements: []SNSStatement{
					{Effect: "Allow", Principal: "*", Action: "SNS:Publish"},
				},
			},
			Tags:           map[string]any{"env": "prod"},
			BindingVersion: "0.2.0",
		},
	},
	{name: "channel/solace", value: SolaceChannelBinding{BindingVersion: "0.4.0"}},
	{
		name: "channel/sqs",
		value: SQSChannelBinding{
			Queue:           &SQSQueue{Name: "orders", FifoQueue: false},
			DeadLetterQueue: &SQSQueue{Name: "dlq", FifoQueue: false},
			BindingVersion:  "0.3.0",
		},
	},
	{name: "channel/stomp", value: StompChannelBinding{BindingVersion: "0.1.0"}},
	{name: "channel/redis", value: RedisChannelBinding{BindingVersion: "0.1.0"}},
	{name: "channel/mercure", value: MercureChannelBinding{BindingVersion: "0.1.0"}},
	{
		name: "channel/ibmmq",
		value: IBMMQChannelBinding{
			DestinationType: "queue",
			Queue:           &IBMMQQueue{ObjectName: "Q", IsPartitioned: false, Exclusive: false},
			Topic: &IBMMQTopic{
				String:           "s",
				ObjectName:       "T",
				DurablePermitted: true,
				LastMsgRetained:  false,
			},
			MaxMsgLength:   1024,
			BindingVersion: "0.1.0",
		},
	},
	{
		name: "channel/googlepubsub",
		value: GooglePubSubChannelBinding{
			Labels:                   map[string]string{"env": "prod"},
			MessageRetentionDuration: "86400s",
			MessageStoragePolicy: &GooglePubSubMessageStoragePolicy{
				AllowedPersistenceRegions: []string{"europe-west1"},
			},
			SchemaSettings: &GooglePubSubSchemaSettings{
				Encoding:        "JSON",
				FirstRevisionID: "1",
				LastRevisionID:  "2",
				Name:            "schema",
			},
			BindingVersion: "0.2.0",
		},
	},
	{
		name: "channel/pulsar",
		value: PulsarChannelBinding{
			Namespace:      "ns",
			Persistence:    "persistent",
			Compaction:     1000,
			GeoReplication: []string{"us-west"},
			Retention:      &PulsarRetention{Time: 60, Size: 1000},
			TTL:            3600,
			Deduplication:  true,
			BindingVersion: "0.1.0",
		},
	},
	{
		name: "operation/kafka",
		value: KafkaOperationBinding{
			GroupID:        &Schema{Type: "string"},
			ClientID:       &Schema{Type: "string"},
			BindingVersion: "0.5.0",
		},
	},
	{
		name: "operation/amqp",
		value: AMQPOperationBinding{
			Expiration:     30,
			UserID:         "guest",
			CC:             []string{"cc"},
			Priority:       1,
			DeliveryMode:   2,
			Mandatory:      true,
			BCC:            []string{"bcc"},
			Timestamp:      true,
			Ack:            false,
			BindingVersion: "0.3.0",
		},
	},
	{
		name:  "operation/nats",
		value: NATSOperationBinding{Queue: "workers", BindingVersion: "0.1.0"},
	},
	{
		name: "operation/mqtt",
		value: MQTTOperationBinding{
			QoS:                   1,
			Retain:                true,
			MessageExpiryInterval: 60,
			BindingVersion:        "0.2.0",
		},
	},
	{
		name: "operation/http",
		value: HTTPOperationBinding{
			Method:         "POST",
			Query:          &Schema{Type: "object"},
			BindingVersion: "0.3.0",
		},
	},
	{name: "operation/ws", value: WSOperationBinding{BindingVersion: "0.1.0"}},
	{name: "operation/amqp1", value: AMQP1OperationBinding{BindingVersion: "0.1.0"}},
	{name: "operation/anypointmq", value: AnypointMQOperationBinding{BindingVersion: "0.0.1"}},
	{name: "operation/jms", value: JMSOperationBinding{BindingVersion: "0.0.1"}},
	{
		name: "operation/sns",
		value: SNSOperationBinding{
			Topic: &SNSIdentifier{Name: "orders"},
			Consumers: []SNSConsumer{
				{Protocol: "sqs", Endpoint: &SNSIdentifier{ARN: "arn"}, RawMessageDelivery: true},
			},
			DeliveryPolicy: &SNSDeliveryPolicy{
				MinDelayTarget: 20,
				MaxDelayTarget: 20,
				NumRetries:     3,
			},
			BindingVersion: "0.2.0",
		},
	},
	{
		name: "operation/solace",
		value: SolaceOperationBinding{
			Destinations: []SolaceDestination{
				{
					DestinationType: "queue",
					Queue:           &SolaceQueue{Name: "q", TopicSubscriptions: []string{"a/*"}},
				},
			},
			TimeToLive:     5000,
			Priority:       120,
			DMQEligible:    true,
			BindingVersion: "0.4.0",
		},
	},
	{
		name: "operation/sqs",
		value: SQSOperationBinding{
			Queues:         []SQSQueue{{Name: "q", FifoQueue: true}},
			BindingVersion: "0.3.0",
		},
	},
	{name: "operation/stomp", value: StompOperationBinding{BindingVersion: "0.1.0"}},
	{name: "operation/redis", value: RedisOperationBinding{BindingVersion: "0.1.0"}},
	{name: "operation/mercure", value: MercureOperationBinding{BindingVersion: "0.1.0"}},
	{name: "operation/ibmmq", value: IBMMQOperationBinding{BindingVersion: "0.1.0"}},
	{name: "operation/googlepubsub", value: GooglePubSubOperationBinding{BindingVersion: "0.2.0"}},
	{
		name: "operation/ros2",
		value: ROS2OperationBinding{
			Role:           "service_server",
			Node:           "/turtlesim",
			QoSPolicies:    &ROS2QoSPolicies{Reliability: "reliable"},
			BindingVersion: "0.1.0",
		},
	},
	{
		name: "message/kafka",
		value: KafkaMessageBinding{
			Key:                     &Schema{Type: "string"},
			SchemaIDLocation:        "payload",
			SchemaIDPayloadEncoding: "4",
			SchemaLookupStrategy:    "TopicNameStrategy",
			BindingVersion:          "0.5.0",
		},
	},
	{
		name: "message/amqp",
		value: AMQPMessageBinding{
			ContentEncoding: "gzip",
			MessageType:     "order",
			BindingVersion:  "0.3.0",
		},
	},
	{name: "message/nats", value: NATSMessageBinding{BindingVersion: "0.1.0"}},
	{
		name: "message/mqtt",
		value: MQTTMessageBinding{
			PayloadFormatIndicator: 1,
			CorrelationData:        &Schema{Type: "string"},
			ContentType:            "application/json",
			ResponseTopic:          "reply",
			BindingVersion:         "0.2.0",
		},
	},
	{
		name: "message/http",
		value: HTTPMessageBinding{
			Headers:        &Schema{Type: "object"},
			StatusCode:     201,
			BindingVersion: "0.3.0",
		},
	},
	{name: "message/ws", value: WSMessageBinding{BindingVersion: "0.1.0"}},
	{name: "message/amqp1", value: AMQP1MessageBinding{BindingVersion: "0.1.0"}},
	{
		name:  "message/anypointmq",
		value: AnypointMQMessageBinding{Headers: &Schema{Type: "object"}, BindingVersion: "0.0.1"},
	},
	{
		name:  "message/jms",
		value: JMSMessageBinding{Headers: &Schema{Type: "object"}, BindingVersion: "0.0.1"},
	},
	{name: "message/sns", value: SNSMessageBinding{BindingVersion: "0.2.0"}},
	{name: "message/solace", value: SolaceMessageBinding{BindingVersion: "0.4.0"}},
	{name: "message/sqs", value: SQSMessageBinding{BindingVersion: "0.3.0"}},
	{name: "message/stomp", value: StompMessageBinding{BindingVersion: "0.1.0"}},
	{name: "message/redis", value: RedisMessageBinding{BindingVersion: "0.1.0"}},
	{name: "message/mercure", value: MercureMessageBinding{BindingVersion: "0.1.0"}},
	{
		name: "message/ibmmq",
		value: IBMMQMessageBinding{
			Type:           "string",
			Headers:        "h",
			Description:    "d",
			Expiry:         60,
			BindingVersion: "0.1.0",
		},
	},
	{
		name: "message/googlepubsub",
		value: GooglePubSubMessageBinding{
			Attributes:     map[string]string{"k": "v"},
			OrderingKey:    "k",
			Schema:         &GooglePubSubSchema{Name: "s"},
			BindingVersion: "0.2.0",
		},
	},
}

// TestBindingEncodings guards the two ways a binding struct can silently emit a
// wrong document: a field whose yaml tag disagrees with its json tag (the two
// codecs are independent), and a tag colliding with another field's.
func TestBindingEncodings(t *testing.T) {
	for _, tc := range bindingValues {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("should_tag_json_and_yaml_identically", func(t *testing.T) {
				typ := reflect.TypeOf(tc.value)
				seen := make(map[string]bool, typ.NumField())
				for field := range typ.Fields() {
					jsonName, _, _ := strings.Cut(field.Tag.Get("json"), ",")
					yamlName, _, _ := strings.Cut(field.Tag.Get("yaml"), ",")
					require.NotEmpty(t, jsonName, "field %s has no json tag", field.Name)
					assert.Equal(
						t,
						jsonName,
						yamlName,
						"field %s: json and yaml tags must match",
						field.Name,
					)
					require.False(t, seen[jsonName], "duplicate json tag %q", jsonName)
					seen[jsonName] = true
				}
			})

			t.Run("should_encode_yaml_and_json_to_the_same_object", func(t *testing.T) {
				jsonOut, err := json.Marshal(tc.value)
				require.NoError(t, err)

				yamlOut, err := yaml.Marshal(tc.value)
				require.NoError(t, err)

				var fromJSON, fromYAML map[string]any
				require.NoError(t, json.Unmarshal(jsonOut, &fromJSON))
				require.NoError(t, yaml.Unmarshal(yamlOut, &fromYAML))

				assert.JSONEq(t, mustMarshal(t, fromJSON), mustMarshal(t, fromYAML))
			})
		})
	}
}

// TestReservedBindingReservedShape pins the reserved-object encoding: one key.
func TestReservedBindingReservedShape(t *testing.T) {
	out, err := json.Marshal(StompServerBinding{BindingVersion: "0.1.0"})
	require.NoError(t, err)
	assert.JSONEq(t, `{"bindingVersion":"0.1.0"}`, string(out))
}

func mustMarshal(t *testing.T, v any) string {
	t.Helper()
	out, err := json.Marshal(v)
	require.NoError(t, err)
	return string(out)
}
