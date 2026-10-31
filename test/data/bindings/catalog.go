// Package bindings is a discovery fixture for the typed protocol bindings: one
// server, channel, operation, and message per protocol the published AsyncAPI
// 3.1.0 binding JSON Schemas accept, each carrying the typed binding struct
// rather than a hand-written value. mqtt5 and mercure live in the
// bindings-unlisted fixture instead, because those protocol keys are not part of
// the published schemas.
package bindings

import (
	"github.com/RubenRibGarcia/asyncgo"
	"github.com/RubenRibGarcia/asyncgo/spec"
)

// Event is the payload every message in this fixture carries. The bindings, not
// the payload, are what the fixture exercises.
type Event struct {
	ID string `json:"id" asyncapi:"required"`
}

// The servers below each carry that protocol's typed server binding. Protocols
// whose server binding object is reserved set only bindingVersion.
var (
	kafkaServer = asyncgo.Server("kafka", "kafka", "broker.kafka:9092").
			Kafka(spec.KafkaServerBinding{
			SchemaRegistryURL:    "https://registry.example.com",
			SchemaRegistryVendor: "confluent",
			BindingVersion:       "0.5.0",
		})
	amqpServer = asyncgo.Server("amqp", "amqp", "broker.amqp:5672").
			AMQP(spec.AMQPServerBinding{BindingVersion: "0.3.0"})
	natsServer = asyncgo.Server("nats", "nats", "broker.nats:4222").
			NATS(spec.NATSServerBinding{BindingVersion: "0.1.0"})
	mqttServer = asyncgo.Server("mqtt", "mqtt", "broker.mqtt:1883").
			MQTT(spec.MQTTServerBinding{
			ClientID:              "orders-service",
			CleanSession:          true,
			KeepAlive:             60,
			SessionExpiryInterval: 120,
			MaximumPacketSize:     1024,
			LastWill: &spec.MQTTLastWill{
				Topic:   "orders/last-will",
				QoS:     1,
				Message: "offline",
				Retain:  true,
			},
			BindingVersion: "0.2.0",
		})
	httpServer = asyncgo.Server("http", "http", "api.example.com").
			HTTP(spec.HTTPServerBinding{BindingVersion: "0.3.0"})
	wsServer = asyncgo.Server("ws", "ws", "api.example.com/socket").
			WS(spec.WSServerBinding{BindingVersion: "0.1.0"})
	amqp1Server = asyncgo.Server("amqp1", "amqp1", "broker.amqp1:5672").
			AMQP1(spec.AMQP1ServerBinding{BindingVersion: "0.1.0"})
	anypointMQServer = asyncgo.Server("anypointmq", "anypointmq", "mq.example.com").
				AnypointMQ(spec.AnypointMQServerBinding{BindingVersion: "0.0.1"})
	jmsServer = asyncgo.Server("jms", "jms", "broker.jms:61616").
			JMS(spec.JMSServerBinding{
			JMSConnectionFactory: "ConnectionFactory",
			Properties:           []spec.JMSProperty{{Name: "provider", Value: "ActiveMQ"}},
			ClientID:             "orders-service",
			BindingVersion:       "0.0.1",
		})
	snsServer = asyncgo.Server("sns", "sns", "sns.eu-west-1.amazonaws.com").
			SNS(spec.SNSServerBinding{BindingVersion: "0.1.0"})
	solaceServer = asyncgo.Server("solace", "solace", "broker.solace:55555").
			Solace(spec.SolaceServerBinding{MsgVPN: "orders-vpn", ClientName: "orders-service", BindingVersion: "0.4.0"})
	sqsServer = asyncgo.Server("sqs", "sqs", "sqs.eu-west-1.amazonaws.com").
			SQS(spec.SQSServerBinding{BindingVersion: "0.2.0"})
	stompServer = asyncgo.Server("stomp", "stomp", "broker.stomp:61613").
			Stomp(spec.StompServerBinding{BindingVersion: "0.1.0"})
	redisServer = asyncgo.Server("redis", "redis", "broker.redis:6379").
			Redis(spec.RedisServerBinding{BindingVersion: "0.1.0"})
	ibmmqServer = asyncgo.Server("ibmmq", "ibmmq", "broker.ibmmq:1414").
			IBMMQ(spec.IBMMQServerBinding{
			GroupID:              "orders",
			CCDTQueueManagerName: "QM1",
			CipherSpec:           "ANY_TLS12",
			MultiEndpointServer:  true,
			HeartBeatInterval:    300,
			BindingVersion:       "0.1.0",
		})
	googlePubSubServer = asyncgo.Server("googlepubsub", "googlepubsub", "pubsub.googleapis.com").
				GooglePubSub(spec.GooglePubSubServerBinding{BindingVersion: "0.2.0"})
	pulsarServer = asyncgo.Server("pulsar", "pulsar", "broker.pulsar:6650").
			Pulsar(spec.PulsarServerBinding{Tenant: "orders", BindingVersion: "0.1.0"})
	ros2Server = asyncgo.Server("ros2", "ros2", "ros2.example.com").
			ROS2(spec.ROS2ServerBinding{RMWImplementation: "rmw_fastrtps_cpp", DomainID: 7, BindingVersion: "0.1.0"})
)

// Catalog is the AsyncAPI description used to generate asyncapi.yaml. Every
// protocol the published 3.1.0 binding schemas accept contributes one channel
// with an operation and a message; pulsar has no operation or message binding
// object and ros2 has no channel or message binding object, so those levels carry
// no binding for them.
var Catalog = asyncgo.Spec(
	asyncgo.Info("Bindings", "1.0.0").
		Description("Every protocol binding object AsyncAPI 3.1.0 defines"),

	asyncgo.DefaultContentType("application/json"),

	asyncgo.Servers(
		kafkaServer,
		amqpServer,
		natsServer,
		mqttServer,
		httpServer,
		wsServer,
		amqp1Server,
		anypointMQServer,
		jmsServer,
		snsServer,
		solaceServer,
		sqsServer,
		stompServer,
		redisServer,
		ibmmqServer,
		googlePubSubServer,
		pulsarServer,
		ros2Server,
	),

	asyncgo.Channels(
		asyncgo.Channel("orders-kafka").
			Servers(kafkaServer).
			Kafka(spec.KafkaChannelBinding{
				Topic:      "orders",
				Partitions: 3,
				Replicas:   1,
				TopicConfiguration: &spec.KafkaTopicConfiguration{
					CleanupPolicy:   []string{"delete"},
					RetentionMs:     604800000,
					MaxMessageBytes: 1048576,
				},
				BindingVersion: "0.5.0",
			}).
			Send(asyncgo.Operation().
				Kafka(spec.KafkaOperationBinding{
					GroupID:        &spec.Schema{Type: "string"},
					ClientID:       &spec.Schema{Type: "string"},
					BindingVersion: "0.5.0",
				}).
				Message(asyncgo.MessageOf(Event{}).
					Name("KafkaEvent").
					Kafka(spec.KafkaMessageBinding{
						Key:                     &spec.Schema{Type: "string"},
						SchemaIDLocation:        "payload",
						SchemaIDPayloadEncoding: "4",
						SchemaLookupStrategy:    "TopicNameStrategy",
						BindingVersion:          "0.5.0",
					}))),

		asyncgo.Channel("orders-amqp").
			Servers(amqpServer).
			AMQP(spec.AMQPChannelBinding{
				Is: "routingKey",
				Exchange: &spec.AMQPExchange{
					Name:    "orders",
					Type:    "topic",
					Durable: true,
					VHost:   "/",
				},
				BindingVersion: "0.3.0",
			}).
			Send(asyncgo.Operation().
				AMQP(spec.AMQPOperationBinding{
					Expiration:     60000,
					UserID:         "orders-service",
					CC:             []string{"audit"},
					Priority:       1,
					DeliveryMode:   2,
					Mandatory:      true,
					BCC:            []string{"archive"},
					Timestamp:      true,
					Ack:            true,
					BindingVersion: "0.3.0",
				}).
				Message(asyncgo.MessageOf(Event{}).
					Name("AMQPEvent").
					AMQP(spec.AMQPMessageBinding{
						ContentEncoding: "gzip",
						MessageType:     "order",
						BindingVersion:  "0.3.0",
					}))),

		asyncgo.Channel("orders-nats").
			Servers(natsServer).
			NATS(spec.NATSChannelBinding{BindingVersion: "0.1.0"}).
			Send(asyncgo.Operation().
				NATS(spec.NATSOperationBinding{Queue: "workers", BindingVersion: "0.1.0"}).
				Message(asyncgo.MessageOf(Event{}).
					Name("NATSEvent").
					NATS(spec.NATSMessageBinding{BindingVersion: "0.1.0"}))),

		asyncgo.Channel("orders-mqtt").
			Servers(mqttServer).
			MQTT(spec.MQTTChannelBinding{BindingVersion: "0.2.0"}).
			Send(asyncgo.Operation().
				MQTT(spec.MQTTOperationBinding{
					QoS:                   1,
					Retain:                true,
					MessageExpiryInterval: 60,
					BindingVersion:        "0.2.0",
				}).
				Message(asyncgo.MessageOf(Event{}).
					Name("MQTTEvent").
					MQTT(spec.MQTTMessageBinding{
						PayloadFormatIndicator: 1,
						CorrelationData:        &spec.Schema{Type: "string"},
						ContentType:            "application/json",
						ResponseTopic:          "orders/reply",
						BindingVersion:         "0.2.0",
					}))),

		asyncgo.Channel("orders-http").
			Servers(httpServer).
			HTTP(spec.HTTPChannelBinding{BindingVersion: "0.3.0"}).
			Send(asyncgo.Operation().
				HTTP(spec.HTTPOperationBinding{
					Method:         "POST",
					Query:          &spec.Schema{Type: "object"},
					BindingVersion: "0.3.0",
				}).
				Message(asyncgo.MessageOf(Event{}).
					Name("HTTPEvent").
					HTTP(spec.HTTPMessageBinding{
						Headers:        &spec.Schema{Type: "object"},
						StatusCode:     201,
						BindingVersion: "0.3.0",
					}))),

		asyncgo.Channel("orders-ws").
			Servers(wsServer).
			WS(spec.WSChannelBinding{
				Method:         "GET",
				Query:          &spec.Schema{Type: "object"},
				Headers:        &spec.Schema{Type: "object"},
				BindingVersion: "0.1.0",
			}).
			Send(asyncgo.Operation().
				WS(spec.WSOperationBinding{BindingVersion: "0.1.0"}).
				Message(asyncgo.MessageOf(Event{}).
					Name("WSEvent").
					WS(spec.WSMessageBinding{BindingVersion: "0.1.0"}))),

		asyncgo.Channel("orders-amqp1").
			Servers(amqp1Server).
			AMQP1(spec.AMQP1ChannelBinding{BindingVersion: "0.1.0"}).
			Send(asyncgo.Operation().
				AMQP1(spec.AMQP1OperationBinding{BindingVersion: "0.1.0"}).
				Message(asyncgo.MessageOf(Event{}).
					Name("AMQP1Event").
					AMQP1(spec.AMQP1MessageBinding{BindingVersion: "0.1.0"}))),

		asyncgo.Channel("orders-anypointmq").
			Servers(anypointMQServer).
			AnypointMQ(spec.AnypointMQChannelBinding{
				Destination:     "orders",
				DestinationType: "queue",
				BindingVersion:  "0.0.1",
			}).
			Send(asyncgo.Operation().
				AnypointMQ(spec.AnypointMQOperationBinding{BindingVersion: "0.0.1"}).
				Message(asyncgo.MessageOf(Event{}).
					Name("AnypointMQEvent").
					AnypointMQ(spec.AnypointMQMessageBinding{
						Headers:        &spec.Schema{Type: "object"},
						BindingVersion: "0.0.1",
					}))),

		asyncgo.Channel("orders-jms").
			Servers(jmsServer).
			JMS(spec.JMSChannelBinding{
				Destination:     "orders",
				DestinationType: "queue",
				BindingVersion:  "0.0.1",
			}).
			Send(asyncgo.Operation().
				JMS(spec.JMSOperationBinding{BindingVersion: "0.0.1"}).
				Message(asyncgo.MessageOf(Event{}).
					Name("JMSEvent").
					JMS(spec.JMSMessageBinding{
						Headers:        &spec.Schema{Type: "object"},
						BindingVersion: "0.0.1",
					}))),

		asyncgo.Channel("orders-sns").
			Servers(snsServer).
			SNS(spec.SNSChannelBinding{
				Name:           "orders",
				Ordering:       &spec.SNSOrdering{Type: "standard"},
				Tags:           map[string]any{"env": "test"},
				BindingVersion: "0.1.0",
			}).
			Send(asyncgo.Operation().
				SNS(spec.SNSOperationBinding{
					Topic: &spec.SNSIdentifier{Name: "orders"},
					Consumers: []spec.SNSConsumer{
						{
							Protocol: "sqs",
							Endpoint: &spec.SNSIdentifier{
								ARN: "arn:aws:sqs:eu-west-1:123456789012:orders",
							},
							RawMessageDelivery: true,
						},
					},
					DeliveryPolicy: &spec.SNSDeliveryPolicy{
						MinDelayTarget: 20,
						MaxDelayTarget: 20,
						NumRetries:     3,
					},
					BindingVersion: "0.1.0",
				}).
				Message(asyncgo.MessageOf(Event{}).
					Name("SNSEvent").
					SNS(spec.SNSMessageBinding{BindingVersion: "0.1.0"}))),

		asyncgo.Channel("orders-solace").
			Servers(solaceServer).
			Solace(spec.SolaceChannelBinding{BindingVersion: "0.4.0"}).
			Send(asyncgo.Operation().
				Solace(spec.SolaceOperationBinding{
					Destinations: []spec.SolaceDestination{
						{
							DestinationType: "queue",
							Queue: &spec.SolaceQueue{
								Name:               "orders",
								TopicSubscriptions: []string{"orders/*"},
							},
						},
					},
					TimeToLive:     5000,
					Priority:       100,
					DMQEligible:    true,
					BindingVersion: "0.4.0",
				}).
				Message(asyncgo.MessageOf(Event{}).
					Name("SolaceEvent").
					Solace(spec.SolaceMessageBinding{BindingVersion: "0.4.0"}))),

		asyncgo.Channel("orders-sqs").
			Servers(sqsServer).
			SQS(spec.SQSChannelBinding{
				Queue: &spec.SQSQueue{
					Name:              "orders",
					FifoQueue:         false,
					VisibilityTimeout: 30,
				},
				BindingVersion: "0.2.0",
			}).
			Send(asyncgo.Operation().
				SQS(spec.SQSOperationBinding{
					Queues:         []spec.SQSQueue{{Name: "orders", FifoQueue: false}},
					BindingVersion: "0.2.0",
				}).
				Message(asyncgo.MessageOf(Event{}).
					Name("SQSEvent").
					SQS(spec.SQSMessageBinding{BindingVersion: "0.2.0"}))),

		asyncgo.Channel("orders-stomp").
			Servers(stompServer).
			Stomp(spec.StompChannelBinding{BindingVersion: "0.1.0"}).
			Send(asyncgo.Operation().
				Stomp(spec.StompOperationBinding{BindingVersion: "0.1.0"}).
				Message(asyncgo.MessageOf(Event{}).
					Name("StompEvent").
					Stomp(spec.StompMessageBinding{BindingVersion: "0.1.0"}))),

		asyncgo.Channel("orders-redis").
			Servers(redisServer).
			Redis(spec.RedisChannelBinding{BindingVersion: "0.1.0"}).
			Send(asyncgo.Operation().
				Redis(spec.RedisOperationBinding{BindingVersion: "0.1.0"}).
				Message(asyncgo.MessageOf(Event{}).
					Name("RedisEvent").
					Redis(spec.RedisMessageBinding{BindingVersion: "0.1.0"}))),

		asyncgo.Channel("orders-ibmmq").
			Servers(ibmmqServer).
			IBMMQ(spec.IBMMQChannelBinding{
				DestinationType: "queue",
				Queue: &spec.IBMMQQueue{
					ObjectName:    "ORDERS.QUEUE",
					IsPartitioned: false,
					Exclusive:     false,
				},
				MaxMsgLength:   1048576,
				BindingVersion: "0.1.0",
			}).
			Send(asyncgo.Operation().
				IBMMQ(spec.IBMMQOperationBinding{BindingVersion: "0.1.0"}).
				Message(asyncgo.MessageOf(Event{}).
					Name("IBMMQEvent").
					// The schema's oneOf only permits headers when type is binary.
					IBMMQ(spec.IBMMQMessageBinding{
						Type:           "binary",
						Headers:        "JMSDestination",
						Description:    "IBM MQ message",
						Expiry:         60,
						BindingVersion: "0.1.0",
					}))),

		asyncgo.Channel("orders-googlepubsub").
			Servers(googlePubSubServer).
			GooglePubSub(spec.GooglePubSubChannelBinding{
				Labels:                   map[string]string{"env": "test"},
				MessageRetentionDuration: "86400s",
				MessageStoragePolicy: &spec.GooglePubSubMessageStoragePolicy{
					AllowedPersistenceRegions: []string{"europe-west1"},
				},
				SchemaSettings: &spec.GooglePubSubSchemaSettings{
					Encoding: "JSON",
					Name:     "projects/orders/schemas/orders",
				},
				BindingVersion: "0.2.0",
			}).
			Send(asyncgo.Operation().
				GooglePubSub(spec.GooglePubSubOperationBinding{BindingVersion: "0.2.0"}).
				Message(asyncgo.MessageOf(Event{}).
					Name("GooglePubSubEvent").
					GooglePubSub(spec.GooglePubSubMessageBinding{
						Attributes:  map[string]string{"source": "orders"},
						OrderingKey: "customer",
						Schema: &spec.GooglePubSubSchema{
							Name: "projects/orders/schemas/orders",
						},
						BindingVersion: "0.2.0",
					}))),

		// Pulsar has no operation or message binding object in 3.1.0.
		asyncgo.Channel("orders-pulsar").
			Servers(pulsarServer).
			Pulsar(spec.PulsarChannelBinding{
				Namespace:      "orders",
				Persistence:    "persistent",
				GeoReplication: []string{"eu-west-1"},
				Retention:      &spec.PulsarRetention{Time: 60, Size: 1000},
				TTL:            3600,
				BindingVersion: "0.1.0",
			}).
			Send(asyncgo.Operation().
				Message(asyncgo.MessageOf(Event{}).Name("PulsarEvent"))),

		// ROS 2 has no channel or message binding object in 3.1.0.
		asyncgo.Channel("orders-ros2").
			Servers(ros2Server).
			Send(asyncgo.Operation().
				ROS2(spec.ROS2OperationBinding{
					Role: "subscriber",
					Node: "/orders",
					QoSPolicies: &spec.ROS2QoSPolicies{
						Reliability: "best_effort",
						Durability:  "volatile",
					},
					BindingVersion: "0.1.0",
				}).
				Message(asyncgo.MessageOf(Event{}).Name("ROS2Event"))),
	),
)
