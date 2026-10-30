package spec

// --- HTTP bindings (binding version 0.3.0) ----------------------------------
//
// The server and channel objects are reserved: they are aliases in
// binding_reserved.go.

// HTTPOperationBinding describes HTTP-specific operation information. Query is
// the Schema Object describing the query string.
type HTTPOperationBinding struct {
	Method         string  `json:"method,omitempty"         yaml:"method,omitempty"`
	Query          *Schema `json:"query,omitempty"          yaml:"query,omitempty"`
	BindingVersion string  `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}

// HTTPMessageBinding describes HTTP-specific message information. Headers is
// the Schema Object describing the HTTP headers; StatusCode is the documented
// response status code.
type HTTPMessageBinding struct {
	Headers        *Schema `json:"headers,omitempty"        yaml:"headers,omitempty"`
	StatusCode     int     `json:"statusCode,omitempty"     yaml:"statusCode,omitempty"`
	BindingVersion string  `json:"bindingVersion,omitempty" yaml:"bindingVersion,omitempty"`
}
