package spec

// --- SNS bindings (binding version 0.2.0) -----------------------------------
//
// The server and message objects are reserved: they are aliases in
// binding_reserved.go.

// SNSChannelBinding describes SNS-specific channel information. Name is
// required by the binding specification.
type SNSChannelBinding struct {
	Name           string         `json:"name"                     yaml:"name"`
	Ordering       *SNSOrdering   `json:"ordering,omitempty"       yaml:"ordering,omitempty"`
	Policy         *SNSPolicy     `json:"policy,omitempty"         yaml:"policy,omitempty"`
	Tags           map[string]any `json:"tags,omitempty"           yaml:"tags,omitempty"`
	BindingVersion string         `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}

// SNSOrdering describes the ordering of an SNS FIFO topic.
type SNSOrdering struct {
	Type                      string `json:"type"                                yaml:"type"`
	ContentBasedDeduplication bool   `json:"contentBasedDeduplication,omitempty" yaml:"contentBasedDeduplication,omitempty"`
}

// SNSPolicy describes the access policy of an SNS topic.
type SNSPolicy struct {
	Statements []SNSStatement `json:"statements" yaml:"statements"`
}

// SNSStatement is one statement of an SNS access policy. Principal, Action, and
// Resource are string | []string, and Principal may also be a map, so they are
// `any`.
type SNSStatement struct {
	Effect    string         `json:"effect"              yaml:"effect"`
	Principal any            `json:"principal"           yaml:"principal"`
	Action    any            `json:"action"              yaml:"action"`
	Resource  any            `json:"resource,omitempty"  yaml:"resource,omitempty"`
	Condition map[string]any `json:"condition,omitempty" yaml:"condition,omitempty"`
}

// SNSOperationBinding describes SNS-specific operation information. Consumers
// is required by the binding specification.
type SNSOperationBinding struct {
	Topic          *SNSIdentifier     `json:"topic,omitempty"          yaml:"topic,omitempty"`
	Consumers      []SNSConsumer      `json:"consumers"                yaml:"consumers"`
	DeliveryPolicy *SNSDeliveryPolicy `json:"deliveryPolicy,omitempty" yaml:"deliveryPolicy,omitempty"`
	BindingVersion string             `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}

// SNSIdentifier identifies an SNS topic or consumer endpoint. Exactly one of
// the fields is meaningful per endpoint kind.
type SNSIdentifier struct {
	URL   string `json:"url,omitempty"   yaml:"url,omitempty"`
	Email string `json:"email,omitempty" yaml:"email,omitempty"`
	Phone string `json:"phone,omitempty" yaml:"phone,omitempty"`
	ARN   string `json:"arn,omitempty"   yaml:"arn,omitempty"`
	Name  string `json:"name,omitempty"  yaml:"name,omitempty"`
}

// SNSConsumer describes one subscriber of an SNS topic. Protocol, Endpoint, and
// RawMessageDelivery are required by the binding specification.
type SNSConsumer struct {
	Protocol           string             `json:"protocol"                    yaml:"protocol"`
	Endpoint           *SNSIdentifier     `json:"endpoint"                    yaml:"endpoint"`
	FilterPolicy       map[string]any     `json:"filterPolicy,omitempty"      yaml:"filterPolicy,omitempty"`
	FilterPolicyScope  string             `json:"filterPolicyScope,omitempty" yaml:"filterPolicyScope,omitempty"`
	RawMessageDelivery bool               `json:"rawMessageDelivery"          yaml:"rawMessageDelivery"`
	RedrivePolicy      *SNSRedrivePolicy  `json:"redrivePolicy,omitempty"     yaml:"redrivePolicy,omitempty"`
	DeliveryPolicy     *SNSDeliveryPolicy `json:"deliveryPolicy,omitempty"    yaml:"deliveryPolicy,omitempty"`
	DisplayName        string             `json:"displayName,omitempty"       yaml:"displayName,omitempty"`
}

// SNSRedrivePolicy describes the dead-letter queue of an SNS consumer.
type SNSRedrivePolicy struct {
	DeadLetterQueue *SNSIdentifier `json:"deadLetterQueue"           yaml:"deadLetterQueue"`
	MaxReceiveCount int            `json:"maxReceiveCount,omitempty" yaml:"maxReceiveCount,omitempty"`
}

// SNSDeliveryPolicy describes the retry policy for HTTP consumers.
type SNSDeliveryPolicy struct {
	MinDelayTarget       int    `json:"minDelayTarget,omitempty"       yaml:"minDelayTarget,omitempty"`
	MaxDelayTarget       int    `json:"maxDelayTarget,omitempty"       yaml:"maxDelayTarget,omitempty"`
	NumRetries           int    `json:"numRetries,omitempty"           yaml:"numRetries,omitempty"`
	NumNoDelayRetries    int    `json:"numNoDelayRetries,omitempty"    yaml:"numNoDelayRetries,omitempty"`
	NumMinDelayRetries   int    `json:"numMinDelayRetries,omitempty"   yaml:"numMinDelayRetries,omitempty"`
	NumMaxDelayRetries   int    `json:"numMaxDelayRetries,omitempty"   yaml:"numMaxDelayRetries,omitempty"`
	BackoffFunction      string `json:"backoffFunction,omitempty"      yaml:"backoffFunction,omitempty"`
	MaxReceivesPerSecond int    `json:"maxReceivesPerSecond,omitempty" yaml:"maxReceivesPerSecond,omitempty"`
}
