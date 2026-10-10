package spec

// Schema is an AsyncAPI 3.1.0 Schema Object: a superset of JSON Schema Draft
// 07, plus the three AsyncAPI-specific keywords discriminator, externalDocs,
// and deprecated. The keyword set below is the whole Draft-07 vocabulary a
// document can use; docs/asyncapi-3.1.0-coverage.md §4.3 lists the few
// authoring-only gaps that remain.
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

	// Identification. SchemaURI is the `$schema` keyword — a URI such as
	// "http://json-schema.org/draft-07/schema#", not a version number. ID and
	// SchemaURI rebase and declare the dialect respectively; AsyncAPI requires
	// `$ref` to follow Reference Object behaviour rather than JSON Schema's, so
	// prefer Ref and leave these to hand-authored documents.
	Ref       string `json:"$ref,omitempty"    yaml:"$ref,omitempty"`
	ID        string `json:"$id,omitempty"     yaml:"$id,omitempty"`
	SchemaURI string `json:"$schema,omitempty" yaml:"$schema,omitempty"`

	// Annotations.
	Type         string        `json:"type,omitempty"         yaml:"type,omitempty"`
	Title        string        `json:"title,omitempty"        yaml:"title,omitempty"`
	Description  string        `json:"description,omitempty"  yaml:"description,omitempty"`
	Format       string        `json:"format,omitempty"       yaml:"format,omitempty"`
	Comment      string        `json:"$comment,omitempty"     yaml:"$comment,omitempty"`
	ExternalDocs *ExternalDocs `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
	Deprecated   bool          `json:"deprecated,omitempty"   yaml:"deprecated,omitempty"`

	Properties           map[string]*Schema `json:"properties,omitempty"           yaml:"properties,omitempty"`
	Required             []string           `json:"required,omitempty"             yaml:"required,omitempty"`
	Items                *Schema            `json:"items,omitempty"                yaml:"items,omitempty"`
	AdditionalProperties *Schema            `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`

	// AdditionalItems applies only when Items is the tuple (array) form of
	// Draft-07's `items`, which this model does not express. It is emitted
	// verbatim and otherwise has no effect.
	AdditionalItems *Schema `json:"additionalItems,omitempty" yaml:"additionalItems,omitempty"`

	PatternProperties map[string]*Schema `json:"patternProperties,omitempty" yaml:"patternProperties,omitempty"`
	PropertyNames     *Schema            `json:"propertyNames,omitempty"     yaml:"propertyNames,omitempty"`

	// Dependencies maps a property name to the schema its presence requires.
	// Draft-07 also allows a shorthand array of property names here
	// (`creditCard: [billingAddress]`), which is sugar for the equivalent
	// schema form (`creditCard: {required: [billingAddress]}`) and is
	// deliberately not modeled.
	Dependencies map[string]*Schema `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`

	Contains *Schema `json:"contains,omitempty" yaml:"contains,omitempty"`

	// Values. Examples is the Draft-07 array — the only example keyword 3.1.0
	// defines. The 2.x singular `example` was removed as a spec deviation.
	Enum     []any `json:"enum,omitempty"     yaml:"enum,omitempty"`
	Const    any   `json:"const,omitempty"    yaml:"const,omitempty"`
	Examples []any `json:"examples,omitempty" yaml:"examples,omitempty"`
	Default  any   `json:"default,omitempty"  yaml:"default,omitempty"`

	Definitions map[string]*Schema `json:"definitions,omitempty" yaml:"definitions,omitempty"`
	OneOf       []*Schema          `json:"oneOf,omitempty"       yaml:"oneOf,omitempty"`
	AllOf       []*Schema          `json:"allOf,omitempty"       yaml:"allOf,omitempty"`
	AnyOf       []*Schema          `json:"anyOf,omitempty"       yaml:"anyOf,omitempty"`
	Not         *Schema            `json:"not,omitempty"         yaml:"not,omitempty"`
	If          *Schema            `json:"if,omitempty"          yaml:"if,omitempty"`
	Then        *Schema            `json:"then,omitempty"        yaml:"then,omitempty"`
	Else        *Schema            `json:"else,omitempty"        yaml:"else,omitempty"`

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

	// Array constraints.
	MinItems    *uint64 `json:"minItems,omitempty"    yaml:"minItems,omitempty"`
	MaxItems    *uint64 `json:"maxItems,omitempty"    yaml:"maxItems,omitempty"`
	UniqueItems bool    `json:"uniqueItems,omitempty" yaml:"uniqueItems,omitempty"`

	// Object constraints.
	MinProperties *uint64 `json:"minProperties,omitempty" yaml:"minProperties,omitempty"`
	MaxProperties *uint64 `json:"maxProperties,omitempty" yaml:"maxProperties,omitempty"`

	// Access.
	ReadOnly  bool `json:"readOnly,omitempty"  yaml:"readOnly,omitempty"`
	WriteOnly bool `json:"writeOnly,omitempty" yaml:"writeOnly,omitempty"`

	// Discriminator is the AsyncAPI-specific polymorphism keyword: the name of
	// a schema property used to differentiate between schemas that inherit this
	// one. It is a plain string in AsyncAPI 3.1.0 — not the OpenAPI
	// Discriminator Object — and the named property must also be in Required.
	Discriminator string `json:"discriminator,omitempty" yaml:"discriminator,omitempty"`
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
