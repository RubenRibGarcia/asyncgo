package asyncgo

import (
	"testing"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The tables below cover every (protocol, receiver) pair the specification
// defines a binding object for. A protocol whose binding object is reserved
// still has a helper, so its row is present too; a protocol with no object at
// a level (mqtt5 below the server, pulsar operation/message, ros2
// channel/message) has no helper and no row.

func TestServerBindings(t *testing.T) {
	tt := []struct {
		name  string
		proto string
		apply func(*server) *server
		want  any
	}{
		{
			name:  "should_set_kafka_binding",
			proto: spec.ProtocolKafka,
			apply: func(s *server) *server {
				return s.Kafka(
					spec.KafkaServerBinding{
						SchemaRegistryURL:    "http://registry",
						SchemaRegistryVendor: "confluent",
						BindingVersion:       "0.5.0",
					},
				)
			},
			want: &spec.KafkaServerBinding{
				SchemaRegistryURL:    "http://registry",
				SchemaRegistryVendor: "confluent",
				BindingVersion:       "0.5.0",
			},
		},
		{
			name:  "should_set_amqp_binding",
			proto: spec.ProtocolAMQP,
			apply: func(s *server) *server { return s.AMQP(spec.AMQPServerBinding{BindingVersion: "0.3.0"}) },
			want:  &spec.AMQPServerBinding{BindingVersion: "0.3.0"},
		},
		{
			name:  "should_set_nats_binding",
			proto: spec.ProtocolNATS,
			apply: func(s *server) *server { return s.NATS(spec.NATSServerBinding{BindingVersion: "0.1.0"}) },
			want:  &spec.NATSServerBinding{BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_mqtt_binding",
			proto: spec.ProtocolMQTT,
			apply: func(s *server) *server {
				return s.MQTT(
					spec.MQTTServerBinding{
						ClientID:              "client",
						CleanSession:          true,
						KeepAlive:             60,
						SessionExpiryInterval: 30,
						MaximumPacketSize:     1024,
						LastWill: &spec.MQTTLastWill{
							Topic:   "t",
							QoS:     1,
							Message: "m",
							Retain:  true,
						},
						BindingVersion: "0.2.0",
					},
				)
			},
			want: &spec.MQTTServerBinding{
				ClientID:              "client",
				CleanSession:          true,
				KeepAlive:             60,
				SessionExpiryInterval: 30,
				MaximumPacketSize:     1024,
				LastWill: &spec.MQTTLastWill{
					Topic:   "t",
					QoS:     1,
					Message: "m",
					Retain:  true,
				},
				BindingVersion: "0.2.0",
			},
		},
		{
			name:  "should_set_http_binding",
			proto: spec.ProtocolHTTP,
			apply: func(s *server) *server { return s.HTTP(spec.HTTPServerBinding{BindingVersion: "0.3.0"}) },
			want:  &spec.HTTPServerBinding{BindingVersion: "0.3.0"},
		},
		{
			name:  "should_set_ws_binding",
			proto: spec.ProtocolWS,
			apply: func(s *server) *server { return s.WS(spec.WSServerBinding{BindingVersion: "0.1.0"}) },
			want:  &spec.WSServerBinding{BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_amqp1_binding",
			proto: spec.ProtocolAMQP1,
			apply: func(s *server) *server { return s.AMQP1(spec.AMQP1ServerBinding{BindingVersion: "0.1.0"}) },
			want:  &spec.AMQP1ServerBinding{BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_mqtt5_binding",
			proto: spec.ProtocolMQTT5,
			apply: func(s *server) *server {
				return s.MQTT5(
					spec.MQTT5ServerBinding{SessionExpiryInterval: 30, BindingVersion: "0.2.0"},
				)
			},
			want: &spec.MQTT5ServerBinding{SessionExpiryInterval: 30, BindingVersion: "0.2.0"},
		},
		{
			name:  "should_set_anypoint_mq_binding",
			proto: spec.ProtocolAnypointMQ,
			apply: func(s *server) *server { return s.AnypointMQ(spec.AnypointMQServerBinding{BindingVersion: "0.0.1"}) },
			want:  &spec.AnypointMQServerBinding{BindingVersion: "0.0.1"},
		},
		{
			name:  "should_set_jms_binding",
			proto: spec.ProtocolJMS,
			apply: func(s *server) *server {
				return s.JMS(
					spec.JMSServerBinding{
						JMSConnectionFactory: "cf",
						Properties:           []spec.JMSProperty{{Name: "p", Value: "v"}},
						ClientID:             "client",
						BindingVersion:       "0.0.1",
					},
				)
			},
			want: &spec.JMSServerBinding{
				JMSConnectionFactory: "cf",
				Properties:           []spec.JMSProperty{{Name: "p", Value: "v"}},
				ClientID:             "client",
				BindingVersion:       "0.0.1",
			},
		},
		{
			name:  "should_set_sns_binding",
			proto: spec.ProtocolSNS,
			apply: func(s *server) *server { return s.SNS(spec.SNSServerBinding{BindingVersion: "0.2.0"}) },
			want:  &spec.SNSServerBinding{BindingVersion: "0.2.0"},
		},
		{
			name:  "should_set_solace_binding",
			proto: spec.ProtocolSolace,
			apply: func(s *server) *server {
				return s.Solace(
					spec.SolaceServerBinding{
						MsgVPN:         "vpn",
						ClientName:     "client",
						BindingVersion: "0.4.0",
					},
				)
			},
			want: &spec.SolaceServerBinding{
				MsgVPN:         "vpn",
				ClientName:     "client",
				BindingVersion: "0.4.0",
			},
		},
		{
			name:  "should_set_sqs_binding",
			proto: spec.ProtocolSQS,
			apply: func(s *server) *server { return s.SQS(spec.SQSServerBinding{BindingVersion: "0.3.0"}) },
			want:  &spec.SQSServerBinding{BindingVersion: "0.3.0"},
		},
		{
			name:  "should_set_stomp_binding",
			proto: spec.ProtocolStomp,
			apply: func(s *server) *server { return s.Stomp(spec.StompServerBinding{BindingVersion: "0.1.0"}) },
			want:  &spec.StompServerBinding{BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_redis_binding",
			proto: spec.ProtocolRedis,
			apply: func(s *server) *server { return s.Redis(spec.RedisServerBinding{BindingVersion: "0.1.0"}) },
			want:  &spec.RedisServerBinding{BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_mercure_binding",
			proto: spec.ProtocolMercure,
			apply: func(s *server) *server { return s.Mercure(spec.MercureServerBinding{BindingVersion: "0.1.0"}) },
			want:  &spec.MercureServerBinding{BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_ibmmq_binding",
			proto: spec.ProtocolIBMMQ,
			apply: func(s *server) *server {
				return s.IBMMQ(
					spec.IBMMQServerBinding{
						GroupID:              "g",
						CCDTQueueManagerName: "qm",
						CipherSpec:           "ANY_TLS12",
						MultiEndpointServer:  true,
						HeartBeatInterval:    300,
						BindingVersion:       "0.1.0",
					},
				)
			},
			want: &spec.IBMMQServerBinding{
				GroupID:              "g",
				CCDTQueueManagerName: "qm",
				CipherSpec:           "ANY_TLS12",
				MultiEndpointServer:  true,
				HeartBeatInterval:    300,
				BindingVersion:       "0.1.0",
			},
		},
		{
			name:  "should_set_google_pub_sub_binding",
			proto: spec.ProtocolGooglePubSub,
			apply: func(s *server) *server {
				return s.GooglePubSub(spec.GooglePubSubServerBinding{BindingVersion: "0.2.0"})
			},
			want: &spec.GooglePubSubServerBinding{BindingVersion: "0.2.0"},
		},
		{
			name:  "should_set_pulsar_binding",
			proto: spec.ProtocolPulsar,
			apply: func(s *server) *server {
				return s.Pulsar(spec.PulsarServerBinding{Tenant: "tenant", BindingVersion: "0.1.0"})
			},
			want: &spec.PulsarServerBinding{Tenant: "tenant", BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_ros2_binding",
			proto: spec.ProtocolROS2,
			apply: func(s *server) *server {
				return s.ROS2(
					spec.ROS2ServerBinding{
						RMWImplementation: "rmw_fastrtps_cpp",
						DomainID:          1,
						BindingVersion:    "0.1.0",
					},
				)
			},
			want: &spec.ROS2ServerBinding{
				RMWImplementation: "rmw_fastrtps_cpp",
				DomainID:          1,
				BindingVersion:    "0.1.0",
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			s := tc.apply(Server("prod", tc.proto, "broker:9092"))
			require.Contains(t, s.s.Bindings, tc.proto)
			assert.Equal(t, tc.want, s.s.Bindings[tc.proto])
		})
	}

	t.Run("should_set_generic_binding", func(t *testing.T) {
		s := Server("prod", "kafka", "broker:9092").Binding("custom", "value")
		assert.Equal(t, "value", s.s.Bindings["custom"])
	})

	t.Run("should_initialize_bindings_when_nil", func(t *testing.T) {
		s := Server("prod", "kafka", "broker:9092")
		assert.Nil(t, s.s.Bindings)
		s.Kafka(
			spec.KafkaServerBinding{
				SchemaRegistryURL:    "http://registry",
				SchemaRegistryVendor: "confluent",
				BindingVersion:       "0.5.0",
			},
		)
		assert.NotNil(t, s.s.Bindings)
	})
}

func TestChannelBindings(t *testing.T) {
	tt := []struct {
		name  string
		proto string
		apply func(*channel) *channel
		want  any
	}{
		{
			name:  "should_set_kafka_binding",
			proto: spec.ProtocolKafka,
			apply: func(c *channel) *channel {
				return c.Kafka(
					spec.KafkaChannelBinding{
						Topic:      "orders",
						Partitions: 3,
						Replicas:   1,
						TopicConfiguration: &spec.KafkaTopicConfiguration{
							CleanupPolicy: []string{"delete"},
							RetentionMs:   1000,
						},
						BindingVersion: "0.5.0",
					},
				)
			},
			want: &spec.KafkaChannelBinding{
				Topic:      "orders",
				Partitions: 3,
				Replicas:   1,
				TopicConfiguration: &spec.KafkaTopicConfiguration{
					CleanupPolicy: []string{"delete"},
					RetentionMs:   1000,
				},
				BindingVersion: "0.5.0",
			},
		},
		{
			name:  "should_set_amqp_binding",
			proto: spec.ProtocolAMQP,
			apply: func(c *channel) *channel {
				return c.AMQP(
					spec.AMQPChannelBinding{
						Is: "queue",
						Exchange: &spec.AMQPExchange{
							Name:       "e",
							Type:       "topic",
							Durable:    true,
							AutoDelete: false,
							VHost:      "/",
						},
						Queue: &spec.AMQPQueue{
							Name:       "q",
							Durable:    true,
							Exclusive:  false,
							AutoDelete: false,
							VHost:      "/",
						},
						BindingVersion: "0.3.0",
					},
				)
			},
			want: &spec.AMQPChannelBinding{
				Is: "queue",
				Exchange: &spec.AMQPExchange{
					Name:       "e",
					Type:       "topic",
					Durable:    true,
					AutoDelete: false,
					VHost:      "/",
				},
				Queue: &spec.AMQPQueue{
					Name:       "q",
					Durable:    true,
					Exclusive:  false,
					AutoDelete: false,
					VHost:      "/",
				},
				BindingVersion: "0.3.0",
			},
		},
		{
			name:  "should_set_nats_binding",
			proto: spec.ProtocolNATS,
			apply: func(c *channel) *channel { return c.NATS(spec.NATSChannelBinding{BindingVersion: "0.1.0"}) },
			want:  &spec.NATSChannelBinding{BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_mqtt_binding",
			proto: spec.ProtocolMQTT,
			apply: func(c *channel) *channel { return c.MQTT(spec.MQTTChannelBinding{BindingVersion: "0.2.0"}) },
			want:  &spec.MQTTChannelBinding{BindingVersion: "0.2.0"},
		},
		{
			name:  "should_set_http_binding",
			proto: spec.ProtocolHTTP,
			apply: func(c *channel) *channel { return c.HTTP(spec.HTTPChannelBinding{BindingVersion: "0.3.0"}) },
			want:  &spec.HTTPChannelBinding{BindingVersion: "0.3.0"},
		},
		{
			name:  "should_set_ws_binding",
			proto: spec.ProtocolWS,
			apply: func(c *channel) *channel {
				return c.WS(
					spec.WSChannelBinding{
						Method:         "GET",
						Query:          &spec.Schema{Type: "object"},
						Headers:        &spec.Schema{Type: "object"},
						BindingVersion: "0.1.0",
					},
				)
			},
			want: &spec.WSChannelBinding{
				Method:         "GET",
				Query:          &spec.Schema{Type: "object"},
				Headers:        &spec.Schema{Type: "object"},
				BindingVersion: "0.1.0",
			},
		},
		{
			name:  "should_set_amqp1_binding",
			proto: spec.ProtocolAMQP1,
			apply: func(c *channel) *channel { return c.AMQP1(spec.AMQP1ChannelBinding{BindingVersion: "0.1.0"}) },
			want:  &spec.AMQP1ChannelBinding{BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_anypoint_mq_binding",
			proto: spec.ProtocolAnypointMQ,
			apply: func(c *channel) *channel {
				return c.AnypointMQ(
					spec.AnypointMQChannelBinding{
						Destination:     "d",
						DestinationType: "queue",
						BindingVersion:  "0.0.1",
					},
				)
			},
			want: &spec.AnypointMQChannelBinding{
				Destination:     "d",
				DestinationType: "queue",
				BindingVersion:  "0.0.1",
			},
		},
		{
			name:  "should_set_jms_binding",
			proto: spec.ProtocolJMS,
			apply: func(c *channel) *channel {
				return c.JMS(
					spec.JMSChannelBinding{
						Destination:     "d",
						DestinationType: "fifo-queue",
						BindingVersion:  "0.0.1",
					},
				)
			},
			want: &spec.JMSChannelBinding{
				Destination:     "d",
				DestinationType: "fifo-queue",
				BindingVersion:  "0.0.1",
			},
		},
		{
			name:  "should_set_sns_binding",
			proto: spec.ProtocolSNS,
			apply: func(c *channel) *channel {
				return c.SNS(
					spec.SNSChannelBinding{
						Name: "orders",
						Ordering: &spec.SNSOrdering{
							Type:                      "FIFO",
							ContentBasedDeduplication: true,
						},
						Policy: &spec.SNSPolicy{
							Statements: []spec.SNSStatement{
								{Effect: "Allow", Principal: "*", Action: "SNS:Publish"},
							},
						},
						Tags:           map[string]any{"env": "prod"},
						BindingVersion: "0.2.0",
					},
				)
			},
			want: &spec.SNSChannelBinding{
				Name:     "orders",
				Ordering: &spec.SNSOrdering{Type: "FIFO", ContentBasedDeduplication: true},
				Policy: &spec.SNSPolicy{
					Statements: []spec.SNSStatement{
						{Effect: "Allow", Principal: "*", Action: "SNS:Publish"},
					},
				},
				Tags:           map[string]any{"env": "prod"},
				BindingVersion: "0.2.0",
			},
		},
		{
			name:  "should_set_solace_binding",
			proto: spec.ProtocolSolace,
			apply: func(c *channel) *channel { return c.Solace(spec.SolaceChannelBinding{BindingVersion: "0.4.0"}) },
			want:  &spec.SolaceChannelBinding{BindingVersion: "0.4.0"},
		},
		{
			name:  "should_set_sqs_binding",
			proto: spec.ProtocolSQS,
			apply: func(c *channel) *channel {
				return c.SQS(
					spec.SQSChannelBinding{
						Queue:           &spec.SQSQueue{Name: "orders", FifoQueue: false},
						DeadLetterQueue: &spec.SQSQueue{Name: "dlq", FifoQueue: false},
						BindingVersion:  "0.3.0",
					},
				)
			},
			want: &spec.SQSChannelBinding{
				Queue:           &spec.SQSQueue{Name: "orders", FifoQueue: false},
				DeadLetterQueue: &spec.SQSQueue{Name: "dlq", FifoQueue: false},
				BindingVersion:  "0.3.0",
			},
		},
		{
			name:  "should_set_stomp_binding",
			proto: spec.ProtocolStomp,
			apply: func(c *channel) *channel { return c.Stomp(spec.StompChannelBinding{BindingVersion: "0.1.0"}) },
			want:  &spec.StompChannelBinding{BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_redis_binding",
			proto: spec.ProtocolRedis,
			apply: func(c *channel) *channel { return c.Redis(spec.RedisChannelBinding{BindingVersion: "0.1.0"}) },
			want:  &spec.RedisChannelBinding{BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_mercure_binding",
			proto: spec.ProtocolMercure,
			apply: func(c *channel) *channel { return c.Mercure(spec.MercureChannelBinding{BindingVersion: "0.1.0"}) },
			want:  &spec.MercureChannelBinding{BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_ibmmq_binding",
			proto: spec.ProtocolIBMMQ,
			apply: func(c *channel) *channel {
				return c.IBMMQ(
					spec.IBMMQChannelBinding{
						DestinationType: "queue",
						Queue: &spec.IBMMQQueue{
							ObjectName:    "Q",
							IsPartitioned: false,
							Exclusive:     false,
						},
						Topic: &spec.IBMMQTopic{
							String:           "s",
							ObjectName:       "T",
							DurablePermitted: true,
							LastMsgRetained:  false,
						},
						MaxMsgLength:   1024,
						BindingVersion: "0.1.0",
					},
				)
			},
			want: &spec.IBMMQChannelBinding{
				DestinationType: "queue",
				Queue: &spec.IBMMQQueue{
					ObjectName:    "Q",
					IsPartitioned: false,
					Exclusive:     false,
				},
				Topic: &spec.IBMMQTopic{
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
			name:  "should_set_google_pub_sub_binding",
			proto: spec.ProtocolGooglePubSub,
			apply: func(c *channel) *channel {
				return c.GooglePubSub(
					spec.GooglePubSubChannelBinding{
						Labels:                   map[string]string{"env": "prod"},
						MessageRetentionDuration: "86400s",
						MessageStoragePolicy: &spec.GooglePubSubMessageStoragePolicy{
							AllowedPersistenceRegions: []string{"europe-west1"},
						},
						SchemaSettings: &spec.GooglePubSubSchemaSettings{
							Encoding:        "JSON",
							FirstRevisionID: "1",
							LastRevisionID:  "2",
							Name:            "schema",
						},
						BindingVersion: "0.2.0",
					},
				)
			},
			want: &spec.GooglePubSubChannelBinding{
				Labels:                   map[string]string{"env": "prod"},
				MessageRetentionDuration: "86400s",
				MessageStoragePolicy: &spec.GooglePubSubMessageStoragePolicy{
					AllowedPersistenceRegions: []string{"europe-west1"},
				},
				SchemaSettings: &spec.GooglePubSubSchemaSettings{
					Encoding:        "JSON",
					FirstRevisionID: "1",
					LastRevisionID:  "2",
					Name:            "schema",
				},
				BindingVersion: "0.2.0",
			},
		},
		{
			name:  "should_set_pulsar_binding",
			proto: spec.ProtocolPulsar,
			apply: func(c *channel) *channel {
				return c.Pulsar(
					spec.PulsarChannelBinding{
						Namespace:      "ns",
						Persistence:    "persistent",
						Compaction:     1000,
						GeoReplication: []string{"us-west"},
						Retention:      &spec.PulsarRetention{Time: 60, Size: 1000},
						TTL:            3600,
						Deduplication:  true,
						BindingVersion: "0.1.0",
					},
				)
			},
			want: &spec.PulsarChannelBinding{
				Namespace:      "ns",
				Persistence:    "persistent",
				Compaction:     1000,
				GeoReplication: []string{"us-west"},
				Retention:      &spec.PulsarRetention{Time: 60, Size: 1000},
				TTL:            3600,
				Deduplication:  true,
				BindingVersion: "0.1.0",
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			c := tc.apply(Channel("channel-" + tc.proto))
			require.Contains(t, c.s.Bindings, tc.proto)
			assert.Equal(t, tc.want, c.s.Bindings[tc.proto])
		})
	}

	t.Run("should_set_generic_binding", func(t *testing.T) {
		c := Channel("channel-generic").Binding("custom", "value")
		assert.Equal(t, "value", c.s.Bindings["custom"])
	})

	t.Run("should_initialize_bindings_when_nil", func(t *testing.T) {
		c := Channel("channel-generic")
		assert.Nil(t, c.s.Bindings)
		c.Kafka(
			spec.KafkaChannelBinding{
				Topic:      "orders",
				Partitions: 3,
				Replicas:   1,
				TopicConfiguration: &spec.KafkaTopicConfiguration{
					CleanupPolicy: []string{"delete"},
					RetentionMs:   1000,
				},
				BindingVersion: "0.5.0",
			},
		)
		assert.NotNil(t, c.s.Bindings)
	})
}

func TestOperationBindings(t *testing.T) {
	tt := []struct {
		name  string
		proto string
		apply func(*operation) *operation
		want  any
	}{
		{
			name:  "should_set_kafka_binding",
			proto: spec.ProtocolKafka,
			apply: func(o *operation) *operation {
				return o.Kafka(
					spec.KafkaOperationBinding{
						GroupID:        &spec.Schema{Type: "string"},
						ClientID:       &spec.Schema{Type: "string"},
						BindingVersion: "0.5.0",
					},
				)
			},
			want: &spec.KafkaOperationBinding{
				GroupID:        &spec.Schema{Type: "string"},
				ClientID:       &spec.Schema{Type: "string"},
				BindingVersion: "0.5.0",
			},
		},
		{
			name:  "should_set_amqp_binding",
			proto: spec.ProtocolAMQP,
			apply: func(o *operation) *operation {
				return o.AMQP(
					spec.AMQPOperationBinding{
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
				)
			},
			want: &spec.AMQPOperationBinding{
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
			name:  "should_set_nats_binding",
			proto: spec.ProtocolNATS,
			apply: func(o *operation) *operation {
				return o.NATS(spec.NATSOperationBinding{Queue: "workers", BindingVersion: "0.1.0"})
			},
			want: &spec.NATSOperationBinding{Queue: "workers", BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_mqtt_binding",
			proto: spec.ProtocolMQTT,
			apply: func(o *operation) *operation {
				return o.MQTT(
					spec.MQTTOperationBinding{
						QoS:                   1,
						Retain:                true,
						MessageExpiryInterval: 60,
						BindingVersion:        "0.2.0",
					},
				)
			},
			want: &spec.MQTTOperationBinding{
				QoS:                   1,
				Retain:                true,
				MessageExpiryInterval: 60,
				BindingVersion:        "0.2.0",
			},
		},
		{
			name:  "should_set_http_binding",
			proto: spec.ProtocolHTTP,
			apply: func(o *operation) *operation {
				return o.HTTP(
					spec.HTTPOperationBinding{
						Method:         "POST",
						Query:          &spec.Schema{Type: "object"},
						BindingVersion: "0.3.0",
					},
				)
			},
			want: &spec.HTTPOperationBinding{
				Method:         "POST",
				Query:          &spec.Schema{Type: "object"},
				BindingVersion: "0.3.0",
			},
		},
		{
			name:  "should_set_ws_binding",
			proto: spec.ProtocolWS,
			apply: func(o *operation) *operation { return o.WS(spec.WSOperationBinding{BindingVersion: "0.1.0"}) },
			want:  &spec.WSOperationBinding{BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_amqp1_binding",
			proto: spec.ProtocolAMQP1,
			apply: func(o *operation) *operation { return o.AMQP1(spec.AMQP1OperationBinding{BindingVersion: "0.1.0"}) },
			want:  &spec.AMQP1OperationBinding{BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_anypoint_mq_binding",
			proto: spec.ProtocolAnypointMQ,
			apply: func(o *operation) *operation {
				return o.AnypointMQ(spec.AnypointMQOperationBinding{BindingVersion: "0.0.1"})
			},
			want: &spec.AnypointMQOperationBinding{BindingVersion: "0.0.1"},
		},
		{
			name:  "should_set_jms_binding",
			proto: spec.ProtocolJMS,
			apply: func(o *operation) *operation { return o.JMS(spec.JMSOperationBinding{BindingVersion: "0.0.1"}) },
			want:  &spec.JMSOperationBinding{BindingVersion: "0.0.1"},
		},
		{
			name:  "should_set_sns_binding",
			proto: spec.ProtocolSNS,
			apply: func(o *operation) *operation {
				return o.SNS(
					spec.SNSOperationBinding{
						Topic: &spec.SNSIdentifier{Name: "orders"},
						Consumers: []spec.SNSConsumer{
							{
								Protocol:           "sqs",
								Endpoint:           &spec.SNSIdentifier{ARN: "arn"},
								RawMessageDelivery: true,
							},
						},
						DeliveryPolicy: &spec.SNSDeliveryPolicy{
							MinDelayTarget: 20,
							MaxDelayTarget: 20,
							NumRetries:     3,
						},
						BindingVersion: "0.2.0",
					},
				)
			},
			want: &spec.SNSOperationBinding{
				Topic: &spec.SNSIdentifier{Name: "orders"},
				Consumers: []spec.SNSConsumer{
					{
						Protocol:           "sqs",
						Endpoint:           &spec.SNSIdentifier{ARN: "arn"},
						RawMessageDelivery: true,
					},
				},
				DeliveryPolicy: &spec.SNSDeliveryPolicy{
					MinDelayTarget: 20,
					MaxDelayTarget: 20,
					NumRetries:     3,
				},
				BindingVersion: "0.2.0",
			},
		},
		{
			name:  "should_set_solace_binding",
			proto: spec.ProtocolSolace,
			apply: func(o *operation) *operation {
				return o.Solace(
					spec.SolaceOperationBinding{
						Destinations: []spec.SolaceDestination{
							{
								DestinationType: "queue",
								Queue: &spec.SolaceQueue{
									Name:               "q",
									TopicSubscriptions: []string{"a/*"},
								},
							},
						},
						TimeToLive:     5000,
						Priority:       120,
						DMQEligible:    true,
						BindingVersion: "0.4.0",
					},
				)
			},
			want: &spec.SolaceOperationBinding{
				Destinations: []spec.SolaceDestination{
					{
						DestinationType: "queue",
						Queue: &spec.SolaceQueue{
							Name:               "q",
							TopicSubscriptions: []string{"a/*"},
						},
					},
				},
				TimeToLive:     5000,
				Priority:       120,
				DMQEligible:    true,
				BindingVersion: "0.4.0",
			},
		},
		{
			name:  "should_set_sqs_binding",
			proto: spec.ProtocolSQS,
			apply: func(o *operation) *operation {
				return o.SQS(
					spec.SQSOperationBinding{
						Queues:         []spec.SQSQueue{{Name: "q", FifoQueue: true}},
						BindingVersion: "0.3.0",
					},
				)
			},
			want: &spec.SQSOperationBinding{
				Queues:         []spec.SQSQueue{{Name: "q", FifoQueue: true}},
				BindingVersion: "0.3.0",
			},
		},
		{
			name:  "should_set_stomp_binding",
			proto: spec.ProtocolStomp,
			apply: func(o *operation) *operation { return o.Stomp(spec.StompOperationBinding{BindingVersion: "0.1.0"}) },
			want:  &spec.StompOperationBinding{BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_redis_binding",
			proto: spec.ProtocolRedis,
			apply: func(o *operation) *operation { return o.Redis(spec.RedisOperationBinding{BindingVersion: "0.1.0"}) },
			want:  &spec.RedisOperationBinding{BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_mercure_binding",
			proto: spec.ProtocolMercure,
			apply: func(o *operation) *operation { return o.Mercure(spec.MercureOperationBinding{BindingVersion: "0.1.0"}) },
			want:  &spec.MercureOperationBinding{BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_ibmmq_binding",
			proto: spec.ProtocolIBMMQ,
			apply: func(o *operation) *operation { return o.IBMMQ(spec.IBMMQOperationBinding{BindingVersion: "0.1.0"}) },
			want:  &spec.IBMMQOperationBinding{BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_google_pub_sub_binding",
			proto: spec.ProtocolGooglePubSub,
			apply: func(o *operation) *operation {
				return o.GooglePubSub(spec.GooglePubSubOperationBinding{BindingVersion: "0.2.0"})
			},
			want: &spec.GooglePubSubOperationBinding{BindingVersion: "0.2.0"},
		},
		{
			name:  "should_set_ros2_binding",
			proto: spec.ProtocolROS2,
			apply: func(o *operation) *operation {
				return o.ROS2(
					spec.ROS2OperationBinding{
						Role:           "service_server",
						Node:           "/turtlesim",
						QoSPolicies:    &spec.ROS2QoSPolicies{Reliability: "reliable"},
						BindingVersion: "0.1.0",
					},
				)
			},
			want: &spec.ROS2OperationBinding{
				Role:           "service_server",
				Node:           "/turtlesim",
				QoSPolicies:    &spec.ROS2QoSPolicies{Reliability: "reliable"},
				BindingVersion: "0.1.0",
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			o := tc.apply(Operation())
			require.Contains(t, o.bindings, tc.proto)
			assert.Equal(t, tc.want, o.bindings[tc.proto])
		})
	}

	t.Run("should_set_generic_binding", func(t *testing.T) {
		o := Operation().Binding("custom", "value")
		assert.Equal(t, "value", o.bindings["custom"])
	})

	t.Run("should_initialize_bindings_when_nil", func(t *testing.T) {
		o := Operation()
		assert.Nil(t, o.bindings)
		o.Kafka(
			spec.KafkaOperationBinding{
				GroupID:        &spec.Schema{Type: "string"},
				ClientID:       &spec.Schema{Type: "string"},
				BindingVersion: "0.5.0",
			},
		)
		assert.NotNil(t, o.bindings)
	})
}

func TestMessageBindings(t *testing.T) {
	tt := []struct {
		name  string
		proto string
		apply func(*message) *message
		want  any
	}{
		{
			name:  "should_set_kafka_binding",
			proto: spec.ProtocolKafka,
			apply: func(m *message) *message {
				return m.Kafka(
					spec.KafkaMessageBinding{
						Key:                     &spec.Schema{Type: "string"},
						SchemaIDLocation:        "payload",
						SchemaIDPayloadEncoding: "4",
						SchemaLookupStrategy:    "TopicNameStrategy",
						BindingVersion:          "0.5.0",
					},
				)
			},
			want: &spec.KafkaMessageBinding{
				Key:                     &spec.Schema{Type: "string"},
				SchemaIDLocation:        "payload",
				SchemaIDPayloadEncoding: "4",
				SchemaLookupStrategy:    "TopicNameStrategy",
				BindingVersion:          "0.5.0",
			},
		},
		{
			name:  "should_set_amqp_binding",
			proto: spec.ProtocolAMQP,
			apply: func(m *message) *message {
				return m.AMQP(
					spec.AMQPMessageBinding{
						ContentEncoding: "gzip",
						MessageType:     "order",
						BindingVersion:  "0.3.0",
					},
				)
			},
			want: &spec.AMQPMessageBinding{
				ContentEncoding: "gzip",
				MessageType:     "order",
				BindingVersion:  "0.3.0",
			},
		},
		{
			name:  "should_set_nats_binding",
			proto: spec.ProtocolNATS,
			apply: func(m *message) *message { return m.NATS(spec.NATSMessageBinding{BindingVersion: "0.1.0"}) },
			want:  &spec.NATSMessageBinding{BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_mqtt_binding",
			proto: spec.ProtocolMQTT,
			apply: func(m *message) *message {
				return m.MQTT(
					spec.MQTTMessageBinding{
						PayloadFormatIndicator: 1,
						CorrelationData:        &spec.Schema{Type: "string"},
						ContentType:            "application/json",
						ResponseTopic:          "reply",
						BindingVersion:         "0.2.0",
					},
				)
			},
			want: &spec.MQTTMessageBinding{
				PayloadFormatIndicator: 1,
				CorrelationData:        &spec.Schema{Type: "string"},
				ContentType:            "application/json",
				ResponseTopic:          "reply",
				BindingVersion:         "0.2.0",
			},
		},
		{
			name:  "should_set_http_binding",
			proto: spec.ProtocolHTTP,
			apply: func(m *message) *message {
				return m.HTTP(
					spec.HTTPMessageBinding{
						Headers:        &spec.Schema{Type: "object"},
						StatusCode:     201,
						BindingVersion: "0.3.0",
					},
				)
			},
			want: &spec.HTTPMessageBinding{
				Headers:        &spec.Schema{Type: "object"},
				StatusCode:     201,
				BindingVersion: "0.3.0",
			},
		},
		{
			name:  "should_set_ws_binding",
			proto: spec.ProtocolWS,
			apply: func(m *message) *message { return m.WS(spec.WSMessageBinding{BindingVersion: "0.1.0"}) },
			want:  &spec.WSMessageBinding{BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_amqp1_binding",
			proto: spec.ProtocolAMQP1,
			apply: func(m *message) *message { return m.AMQP1(spec.AMQP1MessageBinding{BindingVersion: "0.1.0"}) },
			want:  &spec.AMQP1MessageBinding{BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_anypoint_mq_binding",
			proto: spec.ProtocolAnypointMQ,
			apply: func(m *message) *message {
				return m.AnypointMQ(
					spec.AnypointMQMessageBinding{
						Headers:        &spec.Schema{Type: "object"},
						BindingVersion: "0.0.1",
					},
				)
			},
			want: &spec.AnypointMQMessageBinding{
				Headers:        &spec.Schema{Type: "object"},
				BindingVersion: "0.0.1",
			},
		},
		{
			name:  "should_set_jms_binding",
			proto: spec.ProtocolJMS,
			apply: func(m *message) *message {
				return m.JMS(
					spec.JMSMessageBinding{
						Headers:        &spec.Schema{Type: "object"},
						BindingVersion: "0.0.1",
					},
				)
			},
			want: &spec.JMSMessageBinding{
				Headers:        &spec.Schema{Type: "object"},
				BindingVersion: "0.0.1",
			},
		},
		{
			name:  "should_set_sns_binding",
			proto: spec.ProtocolSNS,
			apply: func(m *message) *message { return m.SNS(spec.SNSMessageBinding{BindingVersion: "0.2.0"}) },
			want:  &spec.SNSMessageBinding{BindingVersion: "0.2.0"},
		},
		{
			name:  "should_set_solace_binding",
			proto: spec.ProtocolSolace,
			apply: func(m *message) *message { return m.Solace(spec.SolaceMessageBinding{BindingVersion: "0.4.0"}) },
			want:  &spec.SolaceMessageBinding{BindingVersion: "0.4.0"},
		},
		{
			name:  "should_set_sqs_binding",
			proto: spec.ProtocolSQS,
			apply: func(m *message) *message { return m.SQS(spec.SQSMessageBinding{BindingVersion: "0.3.0"}) },
			want:  &spec.SQSMessageBinding{BindingVersion: "0.3.0"},
		},
		{
			name:  "should_set_stomp_binding",
			proto: spec.ProtocolStomp,
			apply: func(m *message) *message { return m.Stomp(spec.StompMessageBinding{BindingVersion: "0.1.0"}) },
			want:  &spec.StompMessageBinding{BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_redis_binding",
			proto: spec.ProtocolRedis,
			apply: func(m *message) *message { return m.Redis(spec.RedisMessageBinding{BindingVersion: "0.1.0"}) },
			want:  &spec.RedisMessageBinding{BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_mercure_binding",
			proto: spec.ProtocolMercure,
			apply: func(m *message) *message { return m.Mercure(spec.MercureMessageBinding{BindingVersion: "0.1.0"}) },
			want:  &spec.MercureMessageBinding{BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_ibmmq_binding",
			proto: spec.ProtocolIBMMQ,
			apply: func(m *message) *message {
				return m.IBMMQ(
					spec.IBMMQMessageBinding{
						Type:           "string",
						Headers:        "h",
						Description:    "d",
						Expiry:         60,
						BindingVersion: "0.1.0",
					},
				)
			},
			want: &spec.IBMMQMessageBinding{
				Type:           "string",
				Headers:        "h",
				Description:    "d",
				Expiry:         60,
				BindingVersion: "0.1.0",
			},
		},
		{
			name:  "should_set_google_pub_sub_binding",
			proto: spec.ProtocolGooglePubSub,
			apply: func(m *message) *message {
				return m.GooglePubSub(
					spec.GooglePubSubMessageBinding{
						Attributes:     map[string]string{"k": "v"},
						OrderingKey:    "k",
						Schema:         &spec.GooglePubSubSchema{Name: "s"},
						BindingVersion: "0.2.0",
					},
				)
			},
			want: &spec.GooglePubSubMessageBinding{
				Attributes:     map[string]string{"k": "v"},
				OrderingKey:    "k",
				Schema:         &spec.GooglePubSubSchema{Name: "s"},
				BindingVersion: "0.2.0",
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			m := tc.apply(MessageOf(OrderPlaced{}))
			require.Contains(t, m.bindings, tc.proto)
			assert.Equal(t, tc.want, m.bindings[tc.proto])
		})
	}

	t.Run("should_set_generic_binding", func(t *testing.T) {
		m := MessageOf(OrderPlaced{}).Binding("custom", "value")
		assert.Equal(t, "value", m.bindings["custom"])
	})

	t.Run("should_initialize_bindings_when_nil", func(t *testing.T) {
		m := MessageOf(OrderPlaced{})
		assert.Nil(t, m.bindings)
		m.Kafka(
			spec.KafkaMessageBinding{
				Key:                     &spec.Schema{Type: "string"},
				SchemaIDLocation:        "payload",
				SchemaIDPayloadEncoding: "4",
				SchemaLookupStrategy:    "TopicNameStrategy",
				BindingVersion:          "0.5.0",
			},
		)
		assert.NotNil(t, m.bindings)
	})
}

func TestOperationTraitBindings(t *testing.T) {
	tt := []struct {
		name  string
		proto string
		apply func(*operationTrait) *operationTrait
		want  any
	}{
		{
			name:  "should_set_kafka_binding",
			proto: spec.ProtocolKafka,
			apply: func(tr *operationTrait) *operationTrait {
				return tr.Kafka(
					spec.KafkaOperationBinding{
						GroupID:        &spec.Schema{Type: "string"},
						ClientID:       &spec.Schema{Type: "string"},
						BindingVersion: "0.5.0",
					},
				)
			},
			want: &spec.KafkaOperationBinding{
				GroupID:        &spec.Schema{Type: "string"},
				ClientID:       &spec.Schema{Type: "string"},
				BindingVersion: "0.5.0",
			},
		},
		{
			name:  "should_set_amqp_binding",
			proto: spec.ProtocolAMQP,
			apply: func(tr *operationTrait) *operationTrait {
				return tr.AMQP(
					spec.AMQPOperationBinding{
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
				)
			},
			want: &spec.AMQPOperationBinding{
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
			name:  "should_set_nats_binding",
			proto: spec.ProtocolNATS,
			apply: func(tr *operationTrait) *operationTrait {
				return tr.NATS(spec.NATSOperationBinding{Queue: "workers", BindingVersion: "0.1.0"})
			},
			want: &spec.NATSOperationBinding{Queue: "workers", BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_mqtt_binding",
			proto: spec.ProtocolMQTT,
			apply: func(tr *operationTrait) *operationTrait {
				return tr.MQTT(
					spec.MQTTOperationBinding{
						QoS:                   1,
						Retain:                true,
						MessageExpiryInterval: 60,
						BindingVersion:        "0.2.0",
					},
				)
			},
			want: &spec.MQTTOperationBinding{
				QoS:                   1,
				Retain:                true,
				MessageExpiryInterval: 60,
				BindingVersion:        "0.2.0",
			},
		},
		{
			name:  "should_set_http_binding",
			proto: spec.ProtocolHTTP,
			apply: func(tr *operationTrait) *operationTrait {
				return tr.HTTP(
					spec.HTTPOperationBinding{
						Method:         "POST",
						Query:          &spec.Schema{Type: "object"},
						BindingVersion: "0.3.0",
					},
				)
			},
			want: &spec.HTTPOperationBinding{
				Method:         "POST",
				Query:          &spec.Schema{Type: "object"},
				BindingVersion: "0.3.0",
			},
		},
		{
			name:  "should_set_ws_binding",
			proto: spec.ProtocolWS,
			apply: func(tr *operationTrait) *operationTrait {
				return tr.WS(spec.WSOperationBinding{BindingVersion: "0.1.0"})
			},
			want: &spec.WSOperationBinding{BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_amqp1_binding",
			proto: spec.ProtocolAMQP1,
			apply: func(tr *operationTrait) *operationTrait {
				return tr.AMQP1(spec.AMQP1OperationBinding{BindingVersion: "0.1.0"})
			},
			want: &spec.AMQP1OperationBinding{BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_anypoint_mq_binding",
			proto: spec.ProtocolAnypointMQ,
			apply: func(tr *operationTrait) *operationTrait {
				return tr.AnypointMQ(spec.AnypointMQOperationBinding{BindingVersion: "0.0.1"})
			},
			want: &spec.AnypointMQOperationBinding{BindingVersion: "0.0.1"},
		},
		{
			name:  "should_set_jms_binding",
			proto: spec.ProtocolJMS,
			apply: func(tr *operationTrait) *operationTrait {
				return tr.JMS(spec.JMSOperationBinding{BindingVersion: "0.0.1"})
			},
			want: &spec.JMSOperationBinding{BindingVersion: "0.0.1"},
		},
		{
			name:  "should_set_sns_binding",
			proto: spec.ProtocolSNS,
			apply: func(tr *operationTrait) *operationTrait {
				return tr.SNS(
					spec.SNSOperationBinding{
						Topic: &spec.SNSIdentifier{Name: "orders"},
						Consumers: []spec.SNSConsumer{
							{
								Protocol:           "sqs",
								Endpoint:           &spec.SNSIdentifier{ARN: "arn"},
								RawMessageDelivery: true,
							},
						},
						DeliveryPolicy: &spec.SNSDeliveryPolicy{
							MinDelayTarget: 20,
							MaxDelayTarget: 20,
							NumRetries:     3,
						},
						BindingVersion: "0.2.0",
					},
				)
			},
			want: &spec.SNSOperationBinding{
				Topic: &spec.SNSIdentifier{Name: "orders"},
				Consumers: []spec.SNSConsumer{
					{
						Protocol:           "sqs",
						Endpoint:           &spec.SNSIdentifier{ARN: "arn"},
						RawMessageDelivery: true,
					},
				},
				DeliveryPolicy: &spec.SNSDeliveryPolicy{
					MinDelayTarget: 20,
					MaxDelayTarget: 20,
					NumRetries:     3,
				},
				BindingVersion: "0.2.0",
			},
		},
		{
			name:  "should_set_solace_binding",
			proto: spec.ProtocolSolace,
			apply: func(tr *operationTrait) *operationTrait {
				return tr.Solace(
					spec.SolaceOperationBinding{
						Destinations: []spec.SolaceDestination{
							{
								DestinationType: "queue",
								Queue: &spec.SolaceQueue{
									Name:               "q",
									TopicSubscriptions: []string{"a/*"},
								},
							},
						},
						TimeToLive:     5000,
						Priority:       120,
						DMQEligible:    true,
						BindingVersion: "0.4.0",
					},
				)
			},
			want: &spec.SolaceOperationBinding{
				Destinations: []spec.SolaceDestination{
					{
						DestinationType: "queue",
						Queue: &spec.SolaceQueue{
							Name:               "q",
							TopicSubscriptions: []string{"a/*"},
						},
					},
				},
				TimeToLive:     5000,
				Priority:       120,
				DMQEligible:    true,
				BindingVersion: "0.4.0",
			},
		},
		{
			name:  "should_set_sqs_binding",
			proto: spec.ProtocolSQS,
			apply: func(tr *operationTrait) *operationTrait {
				return tr.SQS(
					spec.SQSOperationBinding{
						Queues:         []spec.SQSQueue{{Name: "q", FifoQueue: true}},
						BindingVersion: "0.3.0",
					},
				)
			},
			want: &spec.SQSOperationBinding{
				Queues:         []spec.SQSQueue{{Name: "q", FifoQueue: true}},
				BindingVersion: "0.3.0",
			},
		},
		{
			name:  "should_set_stomp_binding",
			proto: spec.ProtocolStomp,
			apply: func(tr *operationTrait) *operationTrait {
				return tr.Stomp(spec.StompOperationBinding{BindingVersion: "0.1.0"})
			},
			want: &spec.StompOperationBinding{BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_redis_binding",
			proto: spec.ProtocolRedis,
			apply: func(tr *operationTrait) *operationTrait {
				return tr.Redis(spec.RedisOperationBinding{BindingVersion: "0.1.0"})
			},
			want: &spec.RedisOperationBinding{BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_mercure_binding",
			proto: spec.ProtocolMercure,
			apply: func(tr *operationTrait) *operationTrait {
				return tr.Mercure(spec.MercureOperationBinding{BindingVersion: "0.1.0"})
			},
			want: &spec.MercureOperationBinding{BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_ibmmq_binding",
			proto: spec.ProtocolIBMMQ,
			apply: func(tr *operationTrait) *operationTrait {
				return tr.IBMMQ(spec.IBMMQOperationBinding{BindingVersion: "0.1.0"})
			},
			want: &spec.IBMMQOperationBinding{BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_google_pub_sub_binding",
			proto: spec.ProtocolGooglePubSub,
			apply: func(tr *operationTrait) *operationTrait {
				return tr.GooglePubSub(spec.GooglePubSubOperationBinding{BindingVersion: "0.2.0"})
			},
			want: &spec.GooglePubSubOperationBinding{BindingVersion: "0.2.0"},
		},
		{
			name:  "should_set_ros2_binding",
			proto: spec.ProtocolROS2,
			apply: func(tr *operationTrait) *operationTrait {
				return tr.ROS2(
					spec.ROS2OperationBinding{
						Role:           "service_server",
						Node:           "/turtlesim",
						QoSPolicies:    &spec.ROS2QoSPolicies{Reliability: "reliable"},
						BindingVersion: "0.1.0",
					},
				)
			},
			want: &spec.ROS2OperationBinding{
				Role:           "service_server",
				Node:           "/turtlesim",
				QoSPolicies:    &spec.ROS2QoSPolicies{Reliability: "reliable"},
				BindingVersion: "0.1.0",
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			tr := tc.apply(OperationTrait("Trait"))
			require.Contains(t, tr.t.Bindings, tc.proto)
			assert.Equal(t, tc.want, tr.t.Bindings[tc.proto])
		})
	}

	t.Run("should_set_generic_binding", func(t *testing.T) {
		tr := OperationTrait("Trait").Binding("custom", "value")
		assert.Equal(t, "value", tr.t.Bindings["custom"])
	})

	t.Run("should_initialize_bindings_when_nil", func(t *testing.T) {
		tr := OperationTrait("Trait")
		assert.Nil(t, tr.t.Bindings)
		tr.Kafka(
			spec.KafkaOperationBinding{
				GroupID:        &spec.Schema{Type: "string"},
				ClientID:       &spec.Schema{Type: "string"},
				BindingVersion: "0.5.0",
			},
		)
		assert.NotNil(t, tr.t.Bindings)
	})
}

func TestMessageTraitBindings(t *testing.T) {
	tt := []struct {
		name  string
		proto string
		apply func(*messageTrait) *messageTrait
		want  any
	}{
		{
			name:  "should_set_kafka_binding",
			proto: spec.ProtocolKafka,
			apply: func(tr *messageTrait) *messageTrait {
				return tr.Kafka(
					spec.KafkaMessageBinding{
						Key:                     &spec.Schema{Type: "string"},
						SchemaIDLocation:        "payload",
						SchemaIDPayloadEncoding: "4",
						SchemaLookupStrategy:    "TopicNameStrategy",
						BindingVersion:          "0.5.0",
					},
				)
			},
			want: &spec.KafkaMessageBinding{
				Key:                     &spec.Schema{Type: "string"},
				SchemaIDLocation:        "payload",
				SchemaIDPayloadEncoding: "4",
				SchemaLookupStrategy:    "TopicNameStrategy",
				BindingVersion:          "0.5.0",
			},
		},
		{
			name:  "should_set_amqp_binding",
			proto: spec.ProtocolAMQP,
			apply: func(tr *messageTrait) *messageTrait {
				return tr.AMQP(
					spec.AMQPMessageBinding{
						ContentEncoding: "gzip",
						MessageType:     "order",
						BindingVersion:  "0.3.0",
					},
				)
			},
			want: &spec.AMQPMessageBinding{
				ContentEncoding: "gzip",
				MessageType:     "order",
				BindingVersion:  "0.3.0",
			},
		},
		{
			name:  "should_set_nats_binding",
			proto: spec.ProtocolNATS,
			apply: func(tr *messageTrait) *messageTrait { return tr.NATS(spec.NATSMessageBinding{BindingVersion: "0.1.0"}) },
			want:  &spec.NATSMessageBinding{BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_mqtt_binding",
			proto: spec.ProtocolMQTT,
			apply: func(tr *messageTrait) *messageTrait {
				return tr.MQTT(
					spec.MQTTMessageBinding{
						PayloadFormatIndicator: 1,
						CorrelationData:        &spec.Schema{Type: "string"},
						ContentType:            "application/json",
						ResponseTopic:          "reply",
						BindingVersion:         "0.2.0",
					},
				)
			},
			want: &spec.MQTTMessageBinding{
				PayloadFormatIndicator: 1,
				CorrelationData:        &spec.Schema{Type: "string"},
				ContentType:            "application/json",
				ResponseTopic:          "reply",
				BindingVersion:         "0.2.0",
			},
		},
		{
			name:  "should_set_http_binding",
			proto: spec.ProtocolHTTP,
			apply: func(tr *messageTrait) *messageTrait {
				return tr.HTTP(
					spec.HTTPMessageBinding{
						Headers:        &spec.Schema{Type: "object"},
						StatusCode:     201,
						BindingVersion: "0.3.0",
					},
				)
			},
			want: &spec.HTTPMessageBinding{
				Headers:        &spec.Schema{Type: "object"},
				StatusCode:     201,
				BindingVersion: "0.3.0",
			},
		},
		{
			name:  "should_set_ws_binding",
			proto: spec.ProtocolWS,
			apply: func(tr *messageTrait) *messageTrait { return tr.WS(spec.WSMessageBinding{BindingVersion: "0.1.0"}) },
			want:  &spec.WSMessageBinding{BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_amqp1_binding",
			proto: spec.ProtocolAMQP1,
			apply: func(tr *messageTrait) *messageTrait {
				return tr.AMQP1(spec.AMQP1MessageBinding{BindingVersion: "0.1.0"})
			},
			want: &spec.AMQP1MessageBinding{BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_anypoint_mq_binding",
			proto: spec.ProtocolAnypointMQ,
			apply: func(tr *messageTrait) *messageTrait {
				return tr.AnypointMQ(
					spec.AnypointMQMessageBinding{
						Headers:        &spec.Schema{Type: "object"},
						BindingVersion: "0.0.1",
					},
				)
			},
			want: &spec.AnypointMQMessageBinding{
				Headers:        &spec.Schema{Type: "object"},
				BindingVersion: "0.0.1",
			},
		},
		{
			name:  "should_set_jms_binding",
			proto: spec.ProtocolJMS,
			apply: func(tr *messageTrait) *messageTrait {
				return tr.JMS(
					spec.JMSMessageBinding{
						Headers:        &spec.Schema{Type: "object"},
						BindingVersion: "0.0.1",
					},
				)
			},
			want: &spec.JMSMessageBinding{
				Headers:        &spec.Schema{Type: "object"},
				BindingVersion: "0.0.1",
			},
		},
		{
			name:  "should_set_sns_binding",
			proto: spec.ProtocolSNS,
			apply: func(tr *messageTrait) *messageTrait { return tr.SNS(spec.SNSMessageBinding{BindingVersion: "0.2.0"}) },
			want:  &spec.SNSMessageBinding{BindingVersion: "0.2.0"},
		},
		{
			name:  "should_set_solace_binding",
			proto: spec.ProtocolSolace,
			apply: func(tr *messageTrait) *messageTrait {
				return tr.Solace(spec.SolaceMessageBinding{BindingVersion: "0.4.0"})
			},
			want: &spec.SolaceMessageBinding{BindingVersion: "0.4.0"},
		},
		{
			name:  "should_set_sqs_binding",
			proto: spec.ProtocolSQS,
			apply: func(tr *messageTrait) *messageTrait { return tr.SQS(spec.SQSMessageBinding{BindingVersion: "0.3.0"}) },
			want:  &spec.SQSMessageBinding{BindingVersion: "0.3.0"},
		},
		{
			name:  "should_set_stomp_binding",
			proto: spec.ProtocolStomp,
			apply: func(tr *messageTrait) *messageTrait {
				return tr.Stomp(spec.StompMessageBinding{BindingVersion: "0.1.0"})
			},
			want: &spec.StompMessageBinding{BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_redis_binding",
			proto: spec.ProtocolRedis,
			apply: func(tr *messageTrait) *messageTrait {
				return tr.Redis(spec.RedisMessageBinding{BindingVersion: "0.1.0"})
			},
			want: &spec.RedisMessageBinding{BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_mercure_binding",
			proto: spec.ProtocolMercure,
			apply: func(tr *messageTrait) *messageTrait {
				return tr.Mercure(spec.MercureMessageBinding{BindingVersion: "0.1.0"})
			},
			want: &spec.MercureMessageBinding{BindingVersion: "0.1.0"},
		},
		{
			name:  "should_set_ibmmq_binding",
			proto: spec.ProtocolIBMMQ,
			apply: func(tr *messageTrait) *messageTrait {
				return tr.IBMMQ(
					spec.IBMMQMessageBinding{
						Type:           "string",
						Headers:        "h",
						Description:    "d",
						Expiry:         60,
						BindingVersion: "0.1.0",
					},
				)
			},
			want: &spec.IBMMQMessageBinding{
				Type:           "string",
				Headers:        "h",
				Description:    "d",
				Expiry:         60,
				BindingVersion: "0.1.0",
			},
		},
		{
			name:  "should_set_google_pub_sub_binding",
			proto: spec.ProtocolGooglePubSub,
			apply: func(tr *messageTrait) *messageTrait {
				return tr.GooglePubSub(
					spec.GooglePubSubMessageBinding{
						Attributes:     map[string]string{"k": "v"},
						OrderingKey:    "k",
						Schema:         &spec.GooglePubSubSchema{Name: "s"},
						BindingVersion: "0.2.0",
					},
				)
			},
			want: &spec.GooglePubSubMessageBinding{
				Attributes:     map[string]string{"k": "v"},
				OrderingKey:    "k",
				Schema:         &spec.GooglePubSubSchema{Name: "s"},
				BindingVersion: "0.2.0",
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			tr := tc.apply(MessageTrait("Trait"))
			require.Contains(t, tr.t.Bindings, tc.proto)
			assert.Equal(t, tc.want, tr.t.Bindings[tc.proto])
		})
	}

	t.Run("should_set_generic_binding", func(t *testing.T) {
		tr := MessageTrait("Trait").Binding("custom", "value")
		assert.Equal(t, "value", tr.t.Bindings["custom"])
	})

	t.Run("should_initialize_bindings_when_nil", func(t *testing.T) {
		tr := MessageTrait("Trait")
		assert.Nil(t, tr.t.Bindings)
		tr.Kafka(
			spec.KafkaMessageBinding{
				Key:                     &spec.Schema{Type: "string"},
				SchemaIDLocation:        "payload",
				SchemaIDPayloadEncoding: "4",
				SchemaLookupStrategy:    "TopicNameStrategy",
				BindingVersion:          "0.5.0",
			},
		)
		assert.NotNil(t, tr.t.Bindings)
	})
}

func TestProtocolKeys(t *testing.T) {
	t.Run("should_list_twenty_unique_protocols", func(t *testing.T) {
		assert.Len(t, spec.ProtocolKeys, 20)
		seen := make(map[string]bool, len(spec.ProtocolKeys))
		for _, proto := range spec.ProtocolKeys {
			require.False(t, seen[proto], "duplicate protocol key %q", proto)
			seen[proto] = true
		}
	})

	t.Run("should_match_the_protocol_constants", func(t *testing.T) {
		constants := []string{
			spec.ProtocolAMQP,
			spec.ProtocolAMQP1,
			spec.ProtocolAnypointMQ,
			spec.ProtocolGooglePubSub,
			spec.ProtocolHTTP,
			spec.ProtocolIBMMQ,
			spec.ProtocolJMS,
			spec.ProtocolKafka,
			spec.ProtocolMQTT,
			spec.ProtocolMQTT5,
			spec.ProtocolMercure,
			spec.ProtocolNATS,
			spec.ProtocolPulsar,
			spec.ProtocolROS2,
			spec.ProtocolRedis,
			spec.ProtocolSNS,
			spec.ProtocolSQS,
			spec.ProtocolSolace,
			spec.ProtocolStomp,
			spec.ProtocolWS,
		}
		assert.ElementsMatch(t, constants, spec.ProtocolKeys)
	})
}
