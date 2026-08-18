package spec

// Schema is a JSON Schema (Draft 07, the dialect used by AsyncAPI 3.x).
// Only the keywords the generator emits — plus the ones a hand-authored catalog
// is likely to need — are modeled; the type is straightforward to extend.
type Schema struct {
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
