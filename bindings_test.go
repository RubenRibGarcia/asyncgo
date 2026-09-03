package asyncgo

import (
	"testing"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerBindings(t *testing.T) {
	t.Run("should_set_kafka_binding", func(t *testing.T) {
		s := Server(
			"prod",
			"kafka",
			"broker:9092",
		).Kafka(spec.KafkaServerBinding{SchemaRegistryURL: "http://r"})
		b, ok := s.s.Bindings[spec.ProtocolKafka].(*spec.KafkaServerBinding)
		require.True(t, ok)
		assert.Equal(t, "http://r", b.SchemaRegistryURL)
	})

	t.Run("should_set_amqp_binding", func(t *testing.T) {
		s := Server(
			"prod",
			"amqp",
			"broker:9092",
		).AMQP(spec.AMQPServerBinding{BindingVersion: "0.3.0"})
		b, ok := s.s.Bindings[spec.ProtocolAMQP].(*spec.AMQPServerBinding)
		require.True(t, ok)
		assert.Equal(t, "0.3.0", b.BindingVersion)
	})

	t.Run("should_set_nats_binding", func(t *testing.T) {
		s := Server(
			"prod",
			"nats",
			"broker:9092",
		).NATS(spec.NATSServerBinding{BindingVersion: "0.1.0"})
		_, ok := s.s.Bindings[spec.ProtocolNATS].(*spec.NATSServerBinding)
		assert.True(t, ok)
	})

	t.Run("should_set_mqtt_binding", func(t *testing.T) {
		s := Server("prod", "mqtt", "broker:9092").MQTT(spec.MQTTServerBinding{ClientID: "c"})
		b, ok := s.s.Bindings[spec.ProtocolMQTT].(*spec.MQTTServerBinding)
		require.True(t, ok)
		assert.Equal(t, "c", b.ClientID)
	})

	t.Run("should_set_generic_binding", func(t *testing.T) {
		s := Server("prod", "kafka", "broker:9092").Binding("custom", "value")
		assert.Equal(t, "value", s.s.Bindings["custom"])
	})

	t.Run("should_initialize_bindings_when_nil", func(t *testing.T) {
		s := Server("prod", "kafka", "broker:9092")
		assert.Nil(t, s.s.Bindings)
		s.Kafka(spec.KafkaServerBinding{})
		assert.NotNil(t, s.s.Bindings)
	})
}

func TestChannelBindings(t *testing.T) {
	t.Run("should_set_kafka_binding", func(t *testing.T) {
		c := Channel("topic").Kafka(spec.KafkaChannelBinding{Topic: "t"})
		b, ok := c.s.Bindings[spec.ProtocolKafka].(*spec.KafkaChannelBinding)
		require.True(t, ok)
		assert.Equal(t, "t", b.Topic)
	})

	t.Run("should_set_amqp_binding", func(t *testing.T) {
		c := Channel("q").AMQP(spec.AMQPChannelBinding{Is: "queue"})
		b, ok := c.s.Bindings[spec.ProtocolAMQP].(*spec.AMQPChannelBinding)
		require.True(t, ok)
		assert.Equal(t, "queue", b.Is)
	})

	t.Run("should_set_nats_binding", func(t *testing.T) {
		c := Channel("subj").NATS(spec.NATSChannelBinding{BindingVersion: "0.1.0"})
		_, ok := c.s.Bindings[spec.ProtocolNATS].(*spec.NATSChannelBinding)
		assert.True(t, ok)
	})

	t.Run("should_set_mqtt_binding", func(t *testing.T) {
		c := Channel("topic").MQTT(spec.MQTTChannelBinding{BindingVersion: "0.2.0"})
		_, ok := c.s.Bindings[spec.ProtocolMQTT].(*spec.MQTTChannelBinding)
		assert.True(t, ok)
	})

	t.Run("should_set_generic_binding", func(t *testing.T) {
		c := Channel("topic").Binding("custom", "value")
		assert.Equal(t, "value", c.s.Bindings["custom"])
	})

	t.Run("should_initialize_bindings_when_nil", func(t *testing.T) {
		c := Channel("topic")
		assert.Nil(t, c.s.Bindings)
		c.Kafka(spec.KafkaChannelBinding{})
		assert.NotNil(t, c.s.Bindings)
	})
}

func TestOperationBindings(t *testing.T) {
	t.Run("should_set_kafka_binding", func(t *testing.T) {
		o := Operation().Kafka(spec.KafkaOperationBinding{GroupID: "g"})
		b, ok := o.bindings[spec.ProtocolKafka].(*spec.KafkaOperationBinding)
		require.True(t, ok)
		assert.Equal(t, "g", b.GroupID)
	})

	t.Run("should_set_amqp_binding", func(t *testing.T) {
		o := Operation().AMQP(spec.AMQPOperationBinding{Expiration: 30})
		b, ok := o.bindings[spec.ProtocolAMQP].(*spec.AMQPOperationBinding)
		require.True(t, ok)
		assert.Equal(t, 30, b.Expiration)
	})

	t.Run("should_set_nats_binding", func(t *testing.T) {
		o := Operation().NATS(spec.NATSOperationBinding{Queue: "q"})
		b, ok := o.bindings[spec.ProtocolNATS].(*spec.NATSOperationBinding)
		require.True(t, ok)
		assert.Equal(t, "q", b.Queue)
	})

	t.Run("should_set_mqtt_binding", func(t *testing.T) {
		o := Operation().MQTT(spec.MQTTOperationBinding{QoS: 1})
		b, ok := o.bindings[spec.ProtocolMQTT].(*spec.MQTTOperationBinding)
		require.True(t, ok)
		assert.Equal(t, 1, b.QoS)
	})

	t.Run("should_set_generic_binding", func(t *testing.T) {
		o := Operation().Binding("custom", "value")
		assert.Equal(t, "value", o.bindings["custom"])
	})

	t.Run("should_initialize_bindings_when_nil", func(t *testing.T) {
		o := Operation()
		assert.Nil(t, o.bindings)
		o.Kafka(spec.KafkaOperationBinding{})
		assert.NotNil(t, o.bindings)
	})
}

func TestMessageBindings(t *testing.T) {
	t.Run("should_set_kafka_binding", func(t *testing.T) {
		m := MessageOf(OrderPlaced{}).Kafka(spec.KafkaMessageBinding{Key: "k"})
		b, ok := m.bindings[spec.ProtocolKafka].(*spec.KafkaMessageBinding)
		require.True(t, ok)
		assert.Equal(t, "k", b.Key)
	})

	t.Run("should_set_amqp_binding", func(t *testing.T) {
		m := MessageOf(OrderPlaced{}).AMQP(spec.AMQPMessageBinding{ContentEncoding: "gzip"})
		b, ok := m.bindings[spec.ProtocolAMQP].(*spec.AMQPMessageBinding)
		require.True(t, ok)
		assert.Equal(t, "gzip", b.ContentEncoding)
	})

	t.Run("should_set_nats_binding", func(t *testing.T) {
		m := MessageOf(OrderPlaced{}).NATS(spec.NATSMessageBinding{BindingVersion: "0.1.0"})
		_, ok := m.bindings[spec.ProtocolNATS].(*spec.NATSMessageBinding)
		assert.True(t, ok)
	})

	t.Run("should_set_mqtt_binding", func(t *testing.T) {
		m := MessageOf(OrderPlaced{}).MQTT(spec.MQTTMessageBinding{BindingVersion: "0.2.0"})
		_, ok := m.bindings[spec.ProtocolMQTT].(*spec.MQTTMessageBinding)
		assert.True(t, ok)
	})

	t.Run("should_set_generic_binding", func(t *testing.T) {
		m := MessageOf(OrderPlaced{}).Binding("custom", "value")
		assert.Equal(t, "value", m.bindings["custom"])
	})

	t.Run("should_initialize_bindings_when_nil", func(t *testing.T) {
		m := MessageOf(OrderPlaced{})
		assert.Nil(t, m.bindings)
		m.Kafka(spec.KafkaMessageBinding{})
		assert.NotNil(t, m.bindings)
	})
}
