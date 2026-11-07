// Package schema derives AsyncAPI 3.1.0 Schema Objects (a superset of JSON
// Schema Draft 07) from Go types via reflection.
//
// Derivation rules (per the locked design decisions):
//
//   - Named struct types are always hoisted into the provided defs map under
//     their fully-qualified name ("pkgPath.TypeName") and referenced via $ref
//     into "#/components/schemas/...".
//   - Anonymous embedded structs are flattened, matching encoding/json, unless
//     tagged asyncgo:"allOf", in which case the embedded type is composed via
//     allOf instead (opt-in). An embedded struct carrying an explicit name in
//     its json tag (for example a Base field tagged json:"base") is a regular
//     named property instead — encoding/json does not inline it, and neither
//     does derivation.
//   - All fields are optional unless tagged asyncgo:"required".
//   - json struct tags drive field names; "-" skips a field.
//   - asyncapi struct tags carry "required", "enum=a|b|...", "examples=...",
//     "format=...", "const=...", "discriminator=...", the numeric bounds
//     ("minItems=", "maxItems=", "minProperties=", "maxProperties="), and the
//     flags "readOnly", "writeOnly", "uniqueItems", "deprecated".
//     Combinator members are named as "oneOf=A|B": a bare name resolves against
//     the declaring struct's package, and a name containing "/" is a
//     fully-qualified pkgPath.TypeName. A semi-qualified name (pkg.Type, with no
//     import path) is neither, so it emits a dangling $ref — use the full path.
//     internal/discovery applies the same rule when it collects the types to
//     hoist, and rejects a name the generated harness could not emit.
//     Directives are separated by whitespace, so a value may contain a comma.
//     Descriptions are read from the field's doc comment by
//     the generator's discovery pass (internal/discovery), since reflection
//     cannot see comments.
//   - A named type (struct or non-struct) implementing spec.SchemaProvider
//     overrides derivation: its AsyncAPISchema() result is hoisted under the
//     type's fully-qualified name as-is. Non-struct kinds are hoisted only when
//     they carry a provider, so a plain `type ID string` stays inline.
//   - Byte slices are base64 strings by kind, so a named `type Raw []byte`
//     derives the same way []byte does. Only the exact time.Time maps to
//     date-time: a type defined from time.Time is an ordinary named struct
//     whose fields are all unexported, so it derives as an empty object unless
//     it implements spec.SchemaProvider.
package schema

import (
	"encoding/json"
	"reflect"
	"strings"
	"time"

	"github.com/RubenRibGarcia/asyncgo/internal/jsonpointer"
	"github.com/RubenRibGarcia/asyncgo/spec"
)

var (
	timeType           = reflect.TypeFor[time.Time]()
	rawMessageType     = reflect.TypeFor[json.RawMessage]()
	schemaProviderType = reflect.TypeFor[spec.SchemaProvider]()
)

// Name returns the fully-qualified schema name for a named type:
// "pkgPath.TypeName" (e.g. "github.com/acme/app/orders.OrderPlaced").
// Name returns the fully-qualified schema name for a named type:
// "pkgPath.TypeName" (e.g. "github.com/acme/app/orders.OrderPlaced"). It
// returns "" for an unnamed type: an anonymous struct, slice, map or interface
// has no stable name, so it cannot key a component schema. That is the same
// predicate isNamed enforces internally — only named types are ever hoisted.
func Name(t reflect.Type) string {
	if !isNamed(t) {
		return ""
	}
	return t.PkgPath() + "." + t.Name()
}

// Ref returns the $ref for a hoisted named type, or "" for an unnamed type.
// An empty result means "this type cannot be referenced": an unnamed type is
// always derived inline, so a non-empty $ref built from it would dangle.
func Ref(t reflect.Type) string {
	name := Name(t)
	if name == "" {
		return ""
	}
	return "#/components/schemas/" + jsonpointer.Escape(name)
}

// RefByName returns the $ref for a hoisted schema identified by its
// fully-qualified name ("pkgPath.TypeName"), for combinator directives that
// resolve names as strings rather than reflect.Type.
func RefByName(fqn string) string {
	return "#/components/schemas/" + jsonpointer.Escape(fqn)
}

// FromType derives a JSON Schema for t. Named struct types, and named
// non-struct types implementing spec.SchemaProvider, are hoisted into defs
// (keyed by their fully-qualified name) and returned as a $ref; all other types
// are inlined.
//
// A nil t yields an unconstrained schema rather than a panic. defs must be
// non-nil whenever t is hoistable (a named struct or provider); FromType panics
// on a hoistable type with a nil defs map.
func FromType(t reflect.Type, defs map[string]*spec.Schema) *spec.Schema {
	if t == nil {
		return &spec.Schema{}
	}
	t = deref(t)

	// A named non-struct provider opts into the same hoisting named structs get,
	// so a scalar or collection can carry a reusable constraint. Struct
	// providers are intercepted in hoistStruct instead. Interface kinds are
	// skipped because their zero value is a nil interface, and asSchemaProvider
	// asserts that zero value to spec.SchemaProvider.
	if isNamed(t) && t.Kind() != reflect.Struct && t.Kind() != reflect.Interface {
		if ref := hoistedRef(t, defs); ref != "" {
			return spec.Ref(ref)
		}
		if custom := providerSchema(t); custom != nil {
			defs[Name(t)] = custom
			return spec.Ref(Ref(t))
		}
	}

	switch t {
	case timeType:
		return &spec.Schema{Type: "string", Format: "date-time"}
	case rawMessageType:
		return &spec.Schema{}
	}

	// encoding/json marshals any byte slice as base64, so the Draft-07 content
	// annotation says so. The check is kind-based because a named type defined
	// from []byte (type Raw []byte) marshals the same way, while an array
	// ([N]byte) does not. Format "byte" stays: AsyncAPI's own Data Type Formats
	// table defines byte as base64-encoded characters, and it is an annotation,
	// so emitting both asserts nothing contradictory.
	if t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8 {
		return &spec.Schema{Type: "string", Format: "byte", ContentEncoding: "base64"}
	}

	switch t.Kind() {
	case reflect.String:
		return &spec.Schema{Type: "string"}
	case reflect.Bool:
		return &spec.Schema{Type: "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &spec.Schema{Type: "integer"}
	case reflect.Float32:
		return &spec.Schema{Type: "number", Format: "float"}
	case reflect.Float64:
		return &spec.Schema{Type: "number", Format: "double"}
	case reflect.Slice, reflect.Array:
		return &spec.Schema{Type: "array", Items: FromType(t.Elem(), defs)}
	case reflect.Map:
		return &spec.Schema{Type: "object", AdditionalProperties: FromType(t.Elem(), defs)}
	case reflect.Struct:
		if isNamed(t) {
			return hoistStruct(t, defs)
		}
		return inlineStruct(t, defs)
	default:
		// interface, chan, func, complex, uintptr, etc. -> unconstrained.
		return &spec.Schema{}
	}
}

// isNamed reports whether t is a defined type with a stable fully-qualified
// name (anonymous types have neither, and cannot be hoisted).
func isNamed(t reflect.Type) bool { return t.Name() != "" && t.PkgPath() != "" }

// deref strips any pointer layers from t. Derivation looks at the element type
// only — a pointer and its element derive the same schema — so FromType, structOf
// and Register each start from this instead of repeating the loop.
func deref(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

// hoistedRef returns the $ref to t when defs already holds its schema, and ""
// otherwise. Both hoisting paths consult it before deriving, so a type reached
// twice derives once.
func hoistedRef(t reflect.Type, defs map[string]*spec.Schema) string {
	if _, ok := defs[Name(t)]; ok {
		return Ref(t)
	}
	return ""
}

// providerSchema returns the schema t declares through spec.SchemaProvider, or
// nil when t has no provider or the provider returns nil.
//
// A nil result always means "derive as usual" — fillObject for a struct, the
// kind-based schema for anything else — so both hoisting paths agree on what an
// opt-out looks like.
func providerSchema(t reflect.Type) *spec.Schema {
	p, ok := asSchemaProvider(t)
	if !ok {
		return nil
	}
	return p.AsyncAPISchema()
}

// asSchemaProvider detects a SchemaProvider on t or *t, mirroring
// encoding/json's value-then-addressable rule. Pointers are dereferenced before
// this is called, so t is never a pointer, but it may be of any kind.
func asSchemaProvider(t reflect.Type) (spec.SchemaProvider, bool) {
	if t.Implements(schemaProviderType) {
		return reflect.New(t).Elem().Interface().(spec.SchemaProvider), true
	}
	if reflect.PointerTo(t).Implements(schemaProviderType) {
		return reflect.New(t).Interface().(spec.SchemaProvider), true
	}
	return nil, false
}

// hoistStruct registers t in defs (once) and returns a $ref to it. The schema is
// pre-registered before filling so recursive types terminate.
func hoistStruct(t reflect.Type, defs map[string]*spec.Schema) *spec.Schema {
	if ref := hoistedRef(t, defs); ref != "" {
		return spec.Ref(ref)
	}

	s := &spec.Schema{Type: "object"}
	defs[Name(t)] = s // pre-register for recursion termination

	// A provider replaces the derived schema wholesale, so the placeholder is
	// discarded rather than filled.
	if custom := providerSchema(t); custom != nil {
		defs[Name(t)] = custom
		return spec.Ref(Ref(t))
	}

	fillObject(s, t, defs)
	return spec.Ref(Ref(t))
}

func inlineStruct(t reflect.Type, defs map[string]*spec.Schema) *spec.Schema {
	s := &spec.Schema{Type: "object"}
	fillObject(s, t, defs)
	return s
}

// fillObject derives the fields of t into s. Named embeds marked
// `asyncgo:"allOf"` are composed via allOf; unmarked embeds are flattened
// (matching encoding/json), and a flattened base is flattened fully — allOf
// markers on its nested embeds are ignored.
func fillObject(s *spec.Schema, t reflect.Type, defs map[string]*spec.Schema) {
	own := &spec.Schema{Type: "object", Properties: map[string]*spec.Schema{}}
	var embedded []*spec.Schema
	fillFields(own, &embedded, t, defs, false)

	if len(embedded) == 0 {
		s.Properties = own.Properties
		s.Required = own.Required
		return
	}
	s.AllOf = append(embedded, own)
}

// fillFields collects the fields of t into own, appending any `allOf`-marked
// embedded struct refs to embedded. When flattening is true the fields are
// being inlined into an enclosing struct, so allOf markers are ignored and
// everything is flattened; markers are honored only at the level where the
// type is derived standalone.
func fillFields(
	own *spec.Schema,
	embedded *[]*spec.Schema,
	t reflect.Type,
	defs map[string]*spec.Schema,
	flattening bool,
) {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" { // unexported
			continue
		}

		if f.Anonymous && !hasExplicitJSONName(f) {
			if ft, ok := structOf(f.Type); ok {
				if !flattening && hasFlag(parseTag(f.Tag.Get("asyncapi")), "allOf") {
					*embedded = append(*embedded, FromType(f.Type, defs))
					continue
				}
				fillFields(own, embedded, ft, defs, true)
				continue
			}
		}

		name, skip := jsonName(f)
		if skip {
			continue
		}

		prop := FromType(f.Type, defs)
		if ds := parseTag(f.Tag.Get("asyncapi")); len(ds) > 0 {
			applyTag(prop, ds)
			if hasFlag(ds, "required") {
				own.Required = append(own.Required, name)
			}
			if names, ok := combinatorNames(ds, "oneOf"); ok {
				prop.OneOf = refs(names, t)
			}
			if names, ok := combinatorNames(ds, "anyOf"); ok {
				prop.AnyOf = refs(names, t)
			}
			if names, ok := combinatorNames(ds, "allOf"); ok {
				prop.AllOf = refs(names, t)
			}
		}
		own.Properties[name] = prop
	}
}

// structOf dereferences pointers and returns the struct type, or ok=false when
// t is not a struct (possibly behind pointers).
func structOf(t reflect.Type) (reflect.Type, bool) {
	t = deref(t)
	if t.Kind() != reflect.Struct {
		return nil, false
	}
	return t, true
}

// refs resolves combinator directive type names to $refs. A name containing
// "/" is treated as a fully-qualified name (pkgPath.TypeName) and passed
// through; otherwise it is resolved against the declaring struct's package.
func refs(names []string, declaring reflect.Type) []*spec.Schema {
	out := make([]*spec.Schema, 0, len(names))
	for _, name := range names {
		fqn := name
		if !strings.Contains(name, "/") {
			fqn = declaring.PkgPath() + "." + name
		}
		out = append(out, spec.Ref(RefByName(fqn)))
	}
	return out
}

// jsonName resolves the JSON field name from the "json" struct tag, falling
// back to the Go field name. A "-" tag skips the field, and an empty name
// (json:",omitempty") falls back too, matching encoding/json.
func jsonName(f reflect.StructField) (string, bool) {
	tag := f.Tag.Get("json")
	if tag == "" {
		return f.Name, false
	}
	if tag == "-" {
		return "", true
	}
	if before, _, ok := strings.Cut(tag, ","); ok {
		if before == "" {
			return f.Name, false
		}
		return before, false
	}
	return tag, false
}

// hasExplicitJSONName reports whether the "json" tag names the field, which
// makes encoding/json treat an anonymous embedded field as a regular field
// rather than inlining it. Options alone (json:",omitempty") do not name it, and
// "-" does: it is a directive, not an inline request.
func hasExplicitJSONName(f reflect.StructField) bool {
	name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
	return name != ""
}
