package spec

// --- Pulsar bindings (binding version 0.1.0) --------------------------------
//
// The specification defines only server and channel objects for Pulsar: the
// channel and operation bindings objects do not list a pulsar key, so there is
// no PulsarOperationBinding or PulsarMessageBinding to model.

// PulsarServerBinding describes Pulsar-specific server information.
type PulsarServerBinding struct {
	Tenant         string `json:"tenant,omitempty"         yaml:"tenant,omitempty"`
	BindingVersion string `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}

// PulsarChannelBinding describes Pulsar-specific channel information. Namespace
// and Persistence are required by the binding specification.
type PulsarChannelBinding struct {
	Namespace      string           `json:"namespace"                 yaml:"namespace"`
	Persistence    string           `json:"persistence"               yaml:"persistence"` // "persistent" | "non-persistent"
	Compaction     int              `json:"compaction,omitempty"      yaml:"compaction,omitempty"`
	GeoReplication []string         `json:"geo-replication,omitempty" yaml:"geo-replication,omitempty"`
	Retention      *PulsarRetention `json:"retention,omitempty"       yaml:"retention,omitempty"`
	TTL            int              `json:"ttl,omitempty"             yaml:"ttl,omitempty"`
	Deduplication  bool             `json:"deduplication,omitempty"   yaml:"deduplication,omitempty"`
	BindingVersion string           `json:"bindingVersion,omitempty"  yaml:"bindingVersion,omitempty"`
}

// PulsarRetention describes the retention policy of a Pulsar channel.
type PulsarRetention struct {
	Time int `json:"time,omitempty" yaml:"time,omitempty"`
	Size int `json:"size,omitempty" yaml:"size,omitempty"`
}
