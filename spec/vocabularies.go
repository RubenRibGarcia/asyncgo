package spec

// Enum-like vocabularies the model carries as strings.
//
// Every one of these is a closed set in the 3.1.0 specification, so naming the
// values keeps a typo ("kafka" for a protocol, "Fifo" for an SNS ordering type)
// out of a document that no compiler would otherwise check. The values are the
// wire strings, not a derived casing: an SNS ordering type is "FIFO", not "fifo".
//
// The groups are deliberately string constants rather than a named string type:
// a typed field would reject the escape hatch values the *Bindings maps exist to
// carry, and would break decoding of a document that a later spec revision
// extends.

// SecurityScheme.Type values.
const (
	SecuritySchemeTypeUserPassword  = "userPassword"
	SecuritySchemeTypeAPIKey        = "apiKey"
	SecuritySchemeTypeHTTP          = "http"
	SecuritySchemeTypeOAuth2        = "oauth2"
	SecuritySchemeTypeOpenIDConnect = "openIdConnect"
)

// AMQPChannelBinding.Is values: what the routing key of an AMQP channel means.
const (
	AMQPChannelBindingIsQueue      = "queue"
	AMQPChannelBindingIsRoutingKey = "routingKey"
)

// PulsarChannelBinding.Persistence values.
const (
	PulsarChannelBindingPersistencePersistent    = "persistent"
	PulsarChannelBindingPersistenceNonPersistent = "non-persistent"
)

// IBMMQChannelBinding.DestinationType values.
const (
	IBMMQDestinationTypeTopic = "topic"
	IBMMQDestinationTypeQueue = "queue"
)

// SNSOrdering.Type values, from the specification's SNS binding schema.
const (
	SNSOrderingTypeStandard = "standard"
	SNSOrderingTypeFIFO     = "FIFO"
)

// ROS2OperationBinding.Role values, from the specification's ROS 2 operation
// binding schema.
const (
	ROS2RolePublisher     = "publisher"
	ROS2RoleSubscriber    = "subscriber"
	ROS2RoleActionClient  = "action_client"
	ROS2RoleActionServer  = "action_server"
	ROS2RoleServiceClient = "service_client"
	ROS2RoleServiceServer = "service_server"
)
