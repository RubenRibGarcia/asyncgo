// Package schema derives JSON Schemas from Go types via reflection.
//
// Derivation rules (per the locked design decisions):
//
//   - Named struct types are always hoisted into the provided defs map under
//     their fully-qualified name ("pkgPath.TypeName") and referenced via $ref
//     into "#/components/schemas/...".
//   - Anonymous embedded structs are flattened, matching encoding/json, unless
//     tagged asyncgo:"allOf", in which case the embedded type is composed via
//     allOf instead (opt-in).
//   - All fields are optional unless tagged asyncgo:"required".
//   - json struct tags drive field names; "-" skips a field.
//   - asyncapi struct tags carry "required", "enum=a|b|...", "example=...",
//     and "format=...". Descriptions are read from the field's doc comment by
//     the generator's discovery pass (internal/discovery), since reflection
//     cannot see comments.
package schema

import (
	"encoding/json"
	"reflect"
	"strings"
	"time"

	"github.com/RubenRibGarcia/asyncgo/spec"
)

var (
	timeType       = reflect.TypeFor[time.Time]()
	rawMessageType = reflect.TypeFor[json.RawMessage]()
	byteSliceType  = reflect.TypeFor[[]byte]()
)

// Name returns the fully-qualified schema name for a named type:
// "pkgPath.TypeName" (e.g. "github.com/acme/app/orders.OrderPlaced").
func Name(t reflect.Type) string { return t.PkgPath() + "." + t.Name() }

// Ref returns the $ref for a hoisted named type.
func Ref(t reflect.Type) string {
	return "#/components/schemas/" + escapePointer(Name(t))
}

// RefByName returns the $ref for a hoisted schema identified by its
// fully-qualified name ("pkgPath.TypeName"), for combinator directives that
// resolve names as strings rather than reflect.Type.
func RefByName(fqn string) string {
	return "#/components/schemas/" + escapePointer(fqn)
}

// escapePointer applies JSON Pointer escaping (RFC 6901): "~" -> "~0", "/" -> "~1".
func escapePointer(s string) string {
	s = strings.ReplaceAll(s, "~", "~0")
	s = strings.ReplaceAll(s, "/", "~1")
	return s
}

// FromType derives a JSON Schema for t. Named struct types are hoisted into
// defs (keyed by their fully-qualified name) and returned as a $ref; all other
// types are inlined.
func FromType(t reflect.Type, defs map[string]*spec.Schema) *spec.Schema {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	switch t {
	case timeType:
		return &spec.Schema{Type: "string", Format: "date-time"}
	case byteSliceType:
		return &spec.Schema{Type: "string", Format: "byte"}
	case rawMessageType:
		return &spec.Schema{}
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
		if t.Name() != "" && t.PkgPath() != "" {
			return hoistStruct(t, defs)
		}
		return inlineStruct(t, defs)
	default:
		// interface, chan, func, complex, uintptr, etc. -> unconstrained.
		return &spec.Schema{}
	}
}

// hoistStruct registers t in defs (once) and returns a $ref to it. The schema
// is pre-registered before filling so recursive types terminate.
func hoistStruct(t reflect.Type, defs map[string]*spec.Schema) *spec.Schema {
	name := Name(t)
	if _, ok := defs[name]; ok {
		return spec.Ref(Ref(t))
	}
	s := &spec.Schema{Type: "object"}
	defs[name] = s
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

		if f.Anonymous {
			if ft, ok := structOf(f.Type); ok {
				if !flattening && hasFlag(f.Tag.Get("asyncapi"), "allOf") {
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
		if tag := f.Tag.Get("asyncapi"); tag != "" {
			applyTag(prop, tag)
			if hasFlag(tag, "required") {
				own.Required = append(own.Required, name)
			}
			if names, ok := combinatorNames(tag, "oneOf"); ok {
				prop.OneOf = refs(names, t)
			}
			if names, ok := combinatorNames(tag, "anyOf"); ok {
				prop.AnyOf = refs(names, t)
			}
			if names, ok := combinatorNames(tag, "allOf"); ok {
				prop.AllOf = refs(names, t)
			}
		}
		own.Properties[name] = prop
	}
}

// structOf dereferences pointers and returns the struct type, or ok=false when
// t is not a struct (possibly behind pointers).
func structOf(t reflect.Type) (reflect.Type, bool) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
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
// back to the Go field name. A "-" tag skips the field.
func jsonName(f reflect.StructField) (string, bool) {
	tag := f.Tag.Get("json")
	if tag == "" {
		return f.Name, false
	}
	if tag == "-" {
		return "", true
	}
	if before, _, ok := strings.Cut(tag, ","); ok {
		return before, false
	}
	return tag, false
}
