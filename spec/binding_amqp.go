package spec

// --- AMQP 0-9-1 bindings (binding version 0.3.0) ----------------------------
//
// The server object is reserved: AMQPServerBinding is an alias in
// binding_reserved.go.

// AMQPChannelBinding describes AMQP-specific channel information.
type AMQPChannelBinding struct {
	Is             string        `json:"is,omitempty"             yaml:"is,omitempty"` // "queue" | "routingKey"
	Exchange       *AMQPExchange `json:"exchange,omitempty"       yaml:"exchange,omitempty"`
	Queue          *AMQPQueue    `json:"queue,omitempty"          yaml:"queue,omitempty"`
	BindingVersion string        `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}

// AMQPExchange describes an AMQP exchange.
type AMQPExchange struct {
	Name       string `json:"name,omitempty"       yaml:"name,omitempty"`
	Type       string `json:"type,omitempty"       yaml:"type,omitempty"`
	Durable    bool   `json:"durable,omitempty"    yaml:"durable,omitempty"`
	AutoDelete bool   `json:"autoDelete,omitempty" yaml:"autoDelete,omitempty"`
	VHost      string `json:"vhost,omitempty"      yaml:"vhost,omitempty"`
}

// AMQPQueue describes an AMQP queue.
type AMQPQueue struct {
	Name       string `json:"name,omitempty"       yaml:"name,omitempty"`
	Durable    bool   `json:"durable,omitempty"    yaml:"durable,omitempty"`
	Exclusive  bool   `json:"exclusive,omitempty"  yaml:"exclusive,omitempty"`
	AutoDelete bool   `json:"autoDelete,omitempty" yaml:"autoDelete,omitempty"`
	VHost      string `json:"vhost,omitempty"      yaml:"vhost,omitempty"`
}

// AMQPOperationBinding describes AMQP-specific operation information. The
// binding version 0.3.0 defines no replyTo field; it was removed along with the
// 2.x shape.
type AMQPOperationBinding struct {
	Expiration     int      `json:"expiration,omitempty"     yaml:"expiration,omitempty"`
	UserID         string   `json:"userId,omitempty"         yaml:"userId,omitempty"`
	CC             []string `json:"cc,omitempty"             yaml:"cc,omitempty"`
	Priority       int      `json:"priority,omitempty"       yaml:"priority,omitempty"`
	DeliveryMode   int      `json:"deliveryMode,omitempty"   yaml:"deliveryMode,omitempty"`
	Mandatory      bool     `json:"mandatory,omitempty"      yaml:"mandatory,omitempty"`
	BCC            []string `json:"bcc,omitempty"            yaml:"bcc,omitempty"`
	Timestamp      bool     `json:"timestamp,omitempty"      yaml:"timestamp,omitempty"`
	Ack            bool     `json:"ack,omitempty"            yaml:"ack,omitempty"`
	BindingVersion string   `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}

// AMQPMessageBinding describes AMQP-specific message information.
type AMQPMessageBinding struct {
	ContentEncoding string `json:"contentEncoding,omitempty" yaml:"contentEncoding,omitempty"`
	MessageType     string `json:"messageType,omitempty"     yaml:"messageType,omitempty"`
	BindingVersion  string `json:"bindingVersion,omitempty"  yaml:"bindingVersion,omitempty"`
}
