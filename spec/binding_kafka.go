package spec

// --- Kafka bindings (binding version 0.5.0) ---------------------------------

// KafkaServerBinding describes Kafka-specific server information.
type KafkaServerBinding struct {
	SchemaRegistryURL    string `json:"schemaRegistryUrl,omitempty"    yaml:"schemaRegistryUrl,omitempty"`
	SchemaRegistryVendor string `json:"schemaRegistryVendor,omitempty" yaml:"schemaRegistryVendor,omitempty"`
	BindingVersion       string `json:"bindingVersion,omitempty"       yaml:"bindingVersion,omitempty"`
}

// KafkaChannelBinding describes Kafka-specific channel information.
type KafkaChannelBinding struct {
	Topic              string                   `json:"topic,omitempty"              yaml:"topic,omitempty"`
	Partitions         int                      `json:"partitions,omitempty"         yaml:"partitions,omitempty"`
	Replicas           int                      `json:"replicas,omitempty"           yaml:"replicas,omitempty"`
	TopicConfiguration *KafkaTopicConfiguration `json:"topicConfiguration,omitempty" yaml:"topicConfiguration,omitempty"`
	BindingVersion     string                   `json:"bindingVersion,omitempty"     yaml:"bindingVersion,omitempty"`
}

// KafkaTopicConfiguration describes the topic-level configuration of a Kafka
// channel. The keys are Kafka's own dotted configuration names.
type KafkaTopicConfiguration struct {
	CleanupPolicy                     []string `json:"cleanup.policy,omitempty"                        yaml:"cleanup.policy,omitempty"`
	RetentionMs                       int      `json:"retention.ms,omitempty"                          yaml:"retention.ms,omitempty"`
	RetentionBytes                    int      `json:"retention.bytes,omitempty"                       yaml:"retention.bytes,omitempty"`
	DeleteRetentionMs                 int      `json:"delete.retention.ms,omitempty"                   yaml:"delete.retention.ms,omitempty"`
	MaxMessageBytes                   int      `json:"max.message.bytes,omitempty"                     yaml:"max.message.bytes,omitempty"`
	ConfluentKeySchemaValidation      bool     `json:"confluent.key.schema.validation,omitempty"       yaml:"confluent.key.schema.validation,omitempty"`
	ConfluentKeySubjectNameStrategy   string   `json:"confluent.key.subject.name.strategy,omitempty"   yaml:"confluent.key.subject.name.strategy,omitempty"`
	ConfluentValueSchemaValidation    bool     `json:"confluent.value.schema.validation,omitempty"     yaml:"confluent.value.schema.validation,omitempty"`
	ConfluentValueSubjectNameStrategy string   `json:"confluent.value.subject.name.strategy,omitempty" yaml:"confluent.value.subject.name.strategy,omitempty"`
}

// KafkaOperationBinding describes Kafka-specific operation information. GroupID
// and ClientID are Schema Objects describing how the value is derived.
type KafkaOperationBinding struct {
	GroupID        *Schema `json:"groupId,omitempty"        yaml:"groupId,omitempty"`
	ClientID       *Schema `json:"clientId,omitempty"       yaml:"clientId,omitempty"`
	BindingVersion string  `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}

// KafkaMessageBinding describes Kafka-specific message information. Key is
// Schema Object | Reference Object: a schema describing the message key, or a
// reference to one. It is `any` because the union's reference side has to be
// expressible.
type KafkaMessageBinding struct {
	Key                     any    `json:"key,omitempty"                     yaml:"key,omitempty"`
	SchemaIDLocation        string `json:"schemaIdLocation,omitempty"        yaml:"schemaIdLocation,omitempty"`
	SchemaIDPayloadEncoding string `json:"schemaIdPayloadEncoding,omitempty" yaml:"schemaIdPayloadEncoding,omitempty"`
	SchemaLookupStrategy    string `json:"schemaLookupStrategy,omitempty"    yaml:"schemaLookupStrategy,omitempty"`
	BindingVersion          string `json:"bindingVersion,omitempty"          yaml:"bindingVersion,omitempty"`
}
