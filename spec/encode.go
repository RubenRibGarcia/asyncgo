package spec

import (
	"encoding/json"

	"github.com/goccy/go-yaml"
)

// New returns an empty AsyncAPI document with the spec version set.
func New() *AsyncAPI {
	return &AsyncAPI{AsyncAPI: Version}
}

// YAML serializes the document to YAML.
func (a *AsyncAPI) YAML() ([]byte, error) { return yaml.Marshal(a) }

// JSON serializes the document to JSON.
func (a *AsyncAPI) JSON() ([]byte, error) { return json.Marshal(a) }
