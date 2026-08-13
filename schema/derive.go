// Package schema derives JSON Schemas from Go types via reflection.
//
// Derivation rules (per the locked design decisions):
//
//   - Named struct types are always hoisted into the provided defs map under
//     their fully-qualified name ("pkgPath.TypeName") and referenced via $ref
//     into "#/components/schemas/...".
//   - All fields are optional unless tagged asyncgo:"required".
//   - json struct tags drive field names; "-" skips a field.
//   - asyncgo struct tags carry "required", "description=...", "enum=a|b|...",
//     "example=...", and "format=...".
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

func fillObject(s *spec.Schema, t reflect.Type, defs map[string]*spec.Schema) {
	if s.Properties == nil {
		s.Properties = map[string]*spec.Schema{}
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" { // unexported
			continue
		}

		// Anonymous embedded structs are flattened, matching encoding/json.
		if f.Anonymous {
			ft := f.Type
			for ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				fillObject(s, ft, defs)
				continue
			}
		}

		name, skip := jsonName(f)
		if skip {
			continue
		}

		prop := FromType(f.Type, defs)
		if tag := f.Tag.Get("asyncgo"); tag != "" {
			applyTag(prop, tag)
			if hasFlag(tag, "required") {
				s.Required = append(s.Required, name)
			}
		}
		s.Properties[name] = prop
	}
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
