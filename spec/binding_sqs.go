package spec

// --- SQS bindings (binding version 0.3.0) -----------------------------------
//
// The server and message objects are reserved: they are aliases in
// binding_reserved.go.

// SQSChannelBinding describes SQS-specific channel information. Queue is
// required by the binding specification.
type SQSChannelBinding struct {
	Queue           *SQSQueue `json:"queue"                     yaml:"queue"`
	DeadLetterQueue *SQSQueue `json:"deadLetterQueue,omitempty" yaml:"deadLetterQueue,omitempty"`
	BindingVersion  string    `json:"bindingVersion,omitempty"  yaml:"bindingVersion,omitempty"`
}

// SQSOperationBinding describes SQS-specific operation information. Queues is
// required by the binding specification.
type SQSOperationBinding struct {
	Queues         []SQSQueue `json:"queues"                   yaml:"queues"`
	BindingVersion string     `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}

// SQSQueue describes an SQS queue. Name and FifoQueue are required by the
// binding specification, so neither carries omitempty.
type SQSQueue struct {
	Name                   string            `json:"name"                             yaml:"name"`
	FifoQueue              bool              `json:"fifoQueue"                        yaml:"fifoQueue"`
	DeduplicationScope     string            `json:"deduplicationScope,omitempty"     yaml:"deduplicationScope,omitempty"`
	FifoThroughputLimit    string            `json:"fifoThroughputLimit,omitempty"    yaml:"fifoThroughputLimit,omitempty"`
	DeliveryDelay          int               `json:"deliveryDelay,omitempty"          yaml:"deliveryDelay,omitempty"`
	VisibilityTimeout      int               `json:"visibilityTimeout,omitempty"      yaml:"visibilityTimeout,omitempty"`
	ReceiveMessageWaitTime int               `json:"receiveMessageWaitTime,omitempty" yaml:"receiveMessageWaitTime,omitempty"`
	MessageRetentionPeriod int               `json:"messageRetentionPeriod,omitempty" yaml:"messageRetentionPeriod,omitempty"`
	RedrivePolicy          *SQSRedrivePolicy `json:"redrivePolicy,omitempty"          yaml:"redrivePolicy,omitempty"`
	Policy                 *SQSPolicy        `json:"policy,omitempty"                 yaml:"policy,omitempty"`
	Tags                   map[string]string `json:"tags,omitempty"                   yaml:"tags,omitempty"`
}

// SQSRedrivePolicy describes the dead-letter queue of an SQS queue.
type SQSRedrivePolicy struct {
	DeadLetterQueue *SQSIdentifier `json:"deadLetterQueue"           yaml:"deadLetterQueue"`
	MaxReceiveCount int            `json:"maxReceiveCount,omitempty" yaml:"maxReceiveCount,omitempty"`
}

// SQSIdentifier identifies an SQS queue by ARN or by name.
type SQSIdentifier struct {
	ARN  string `json:"arn,omitempty"  yaml:"arn,omitempty"`
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
}

// SQSPolicy describes the access policy of an SQS queue.
type SQSPolicy struct {
	Statements []SQSStatement `json:"statements" yaml:"statements"`
}

// SQSStatement is one statement of an SQS access policy. Principal, Action, and
// Resource are string | []string, and Principal may also be a map, so they are
// `any`.
type SQSStatement struct {
	Effect    string         `json:"effect"              yaml:"effect"`
	Principal any            `json:"principal"           yaml:"principal"`
	Action    any            `json:"action"              yaml:"action"`
	Resource  any            `json:"resource,omitempty"  yaml:"resource,omitempty"`
	Condition map[string]any `json:"condition,omitempty" yaml:"condition,omitempty"`
}
