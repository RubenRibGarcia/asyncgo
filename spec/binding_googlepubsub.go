package spec

// --- Google Cloud Pub/Sub bindings (binding version 0.2.0) ------------------
//
// The server and operation objects are reserved: they are aliases in
// binding_reserved.go.

// GooglePubSubChannelBinding describes Google Cloud Pub/Sub-specific channel
// information. SchemaSettings is required by the binding specification.
type GooglePubSubChannelBinding struct {
	Labels                   map[string]string                 `json:"labels,omitempty"                   yaml:"labels,omitempty"`
	MessageRetentionDuration string                            `json:"messageRetentionDuration,omitempty" yaml:"messageRetentionDuration,omitempty"`
	MessageStoragePolicy     *GooglePubSubMessageStoragePolicy `json:"messageStoragePolicy,omitempty"     yaml:"messageStoragePolicy,omitempty"`
	SchemaSettings           *GooglePubSubSchemaSettings       `json:"schemaSettings"                     yaml:"schemaSettings"`
	BindingVersion           string                            `json:"bindingVersion,omitempty"           yaml:"bindingVersion,omitempty"`
}

// GooglePubSubMessageStoragePolicy describes where messages may be stored.
type GooglePubSubMessageStoragePolicy struct {
	AllowedPersistenceRegions []string `json:"allowedPersistenceRegions,omitempty" yaml:"allowedPersistenceRegions,omitempty"`
}

// GooglePubSubSchemaSettings describes the schema a topic is validated against.
// Encoding and Name are required by the binding specification.
type GooglePubSubSchemaSettings struct {
	Encoding        string `json:"encoding"                  yaml:"encoding"`
	FirstRevisionID string `json:"firstRevisionId,omitempty" yaml:"firstRevisionId,omitempty"`
	LastRevisionID  string `json:"lastRevisionId,omitempty"  yaml:"lastRevisionId,omitempty"`
	Name            string `json:"name"                      yaml:"name"`
}

// GooglePubSubMessageBinding describes Google Cloud Pub/Sub-specific message
// information.
type GooglePubSubMessageBinding struct {
	Attributes     map[string]string   `json:"attributes,omitempty"     yaml:"attributes,omitempty"`
	OrderingKey    string              `json:"orderingKey,omitempty"    yaml:"orderingKey,omitempty"`
	Schema         *GooglePubSubSchema `json:"schema,omitempty"         yaml:"schema,omitempty"`
	BindingVersion string              `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}

// GooglePubSubSchema references the schema a message is validated against. Name
// is required by the binding specification.
type GooglePubSubSchema struct {
	Name string `json:"name" yaml:"name"`
}
