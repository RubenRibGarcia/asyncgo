package spec

// SchemaProvider is implemented by a named struct type that customizes its
// wire representation (e.g. via MarshalJSON). AsyncAPISchema returns the
// schema asyncgo emits in place of reflection-derived derivation.
//
// It is invoked on a zero value, so it must not depend on receiver state; it
// must be pure, deterministic, and must not panic. It may be called more than
// once per generation.
type SchemaProvider interface {
	AsyncAPISchema() *Schema
}
