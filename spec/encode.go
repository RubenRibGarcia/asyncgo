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

// JSON serializes the document to compact JSON.
func (a *AsyncAPI) JSON() ([]byte, error) { return json.Marshal(a) }

// JSONIndent serializes the document to 2-space-indented JSON terminated by a
// newline: the shape used for committed artifacts, so a regenerated document
// diffs cleanly in review. Output is deterministic (struct fields in
// declaration order, map keys sorted), which lets callers compare it byte for
// byte. Use JSON for the compact form.
func (a *AsyncAPI) JSONIndent() ([]byte, error) {
	out, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}
