package spec

// Schema is an AsyncAPI 3.1.0 Schema Object (a superset of JSON Schema Draft
// 07). Only the keywords the generator emits — plus the ones a hand-authored
// catalog is likely to need — are modeled; the type is straightforward to
// extend.
//
// The same struct models the AsyncAPI 3.1.0 Multi Format Schema Object through
// SchemaFormat and Schema. It is deliberately one struct rather than a union:
// the discovery harness round-trips every document through YAML, and a node that
// is not a concrete struct decodes into a map and is re-encoded with sorted
// keys, which would rewrite committed golden files.
type Schema struct {
	// Multi Format Schema Object fields. SchemaFormat is the format identifier
	// (emitted verbatim, e.g. "application/vnd.apache.avro;version=1.9.0");
	// Schema is its opaque body. Both are REQUIRED together, and a node that
	// sets either must not also carry the JSON Schema keywords below.
	SchemaFormat string `json:"schemaFormat,omitempty" yaml:"schemaFormat,omitempty"`
	Schema       any    `json:"schema,omitempty"       yaml:"schema,omitempty"`

	Ref         string `json:"$ref,omitempty"        yaml:"$ref,omitempty"`
	Type        string `json:"type,omitempty"        yaml:"type,omitempty"`
	Title       string `json:"title,omitempty"       yaml:"title,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Format      string `json:"format,omitempty"      yaml:"format,omitempty"`

	Properties           map[string]*Schema `json:"properties,omitempty"           yaml:"properties,omitempty"`
	Required             []string           `json:"required,omitempty"             yaml:"required,omitempty"`
	Items                *Schema            `json:"items,omitempty"                yaml:"items,omitempty"`
	AdditionalProperties *Schema            `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`

	Enum    []any `json:"enum,omitempty"    yaml:"enum,omitempty"`
	Example any   `json:"example,omitempty" yaml:"example,omitempty"`
	Default any   `json:"default,omitempty" yaml:"default,omitempty"`

	Definitions map[string]*Schema `json:"definitions,omitempty" yaml:"definitions,omitempty"`
	OneOf       []*Schema          `json:"oneOf,omitempty"       yaml:"oneOf,omitempty"`
	AllOf       []*Schema          `json:"allOf,omitempty"       yaml:"allOf,omitempty"`
	AnyOf       []*Schema          `json:"anyOf,omitempty"       yaml:"anyOf,omitempty"`
	Not         *Schema            `json:"not,omitempty"         yaml:"not,omitempty"`

	// String constraints.
	MinLength *uint64 `json:"minLength,omitempty" yaml:"minLength,omitempty"`
	MaxLength *uint64 `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`
	Pattern   string  `json:"pattern,omitempty"   yaml:"pattern,omitempty"`

	// Numeric constraints.
	Minimum          *float64 `json:"minimum,omitempty"          yaml:"minimum,omitempty"`
	Maximum          *float64 `json:"maximum,omitempty"          yaml:"maximum,omitempty"`
	ExclusiveMinimum *float64 `json:"exclusiveMinimum,omitempty" yaml:"exclusiveMinimum,omitempty"`
	ExclusiveMaximum *float64 `json:"exclusiveMaximum,omitempty" yaml:"exclusiveMaximum,omitempty"`
	MultipleOf       *float64 `json:"multipleOf,omitempty"       yaml:"multipleOf,omitempty"`
}

// Ref returns a schema that is a JSON Reference to the given pointer.
func Ref(pointer string) *Schema { return &Schema{Ref: pointer} }

// MultiFormat returns a Multi Format Schema Object: a schema in a non-JSON-Schema
// format. format is emitted verbatim as schemaFormat and body is carried opaquely
// as schema — neither is parsed, validated, or rewritten.
//
// Use it wherever a *Schema is accepted (a message payload or headers, a message
// trait's headers, or a component declared with asyncgo.Schema).
func MultiFormat(format string, body any) *Schema {
	return &Schema{SchemaFormat: format, Schema: body}
}
