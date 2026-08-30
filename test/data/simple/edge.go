package simple

// EdgeCases exercises description-extraction and combinator-walk branches
// (unexported fields, plain json tags, ignored fields) without being referenced
// by any message payload, so it never appears in the generated asyncapi.yaml.
type EdgeCases struct {
	untagged string
	// Plain uses a json tag with no options.
	Plain   string `json:"plain"`
	Ignored string `json:"-"`
}
