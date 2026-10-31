package spec

// SchemaProvider is implemented by a named type that customizes its wire
// representation (e.g. via MarshalJSON). AsyncAPISchema returns the schema
// asyncgo emits in place of reflection-derived derivation.
//
// The returned schema is hoisted under the type's fully-qualified name
// ("pkgPath.TypeName") and referenced via $ref from every use site, so a named
// struct and a named scalar or collection are treated alike: a type such as
// `type OrderID string` implements this interface to carry a reusable
// constraint instead of being inlined. Returning nil means "derive as usual";
// a non-nil &Schema{} is an explicitly unconstrained schema, not a no-op.
//
// It is invoked on a zero value, so it must not depend on receiver state; it
// must be pure, deterministic, and must not panic. It may be called more than
// once per generation.
type SchemaProvider interface {
	AsyncAPISchema() *Schema
}
