// Package schema derives AsyncAPI 3.1.0 Schema Objects (a superset of JSON
// Schema Draft 07) from Go types via reflection.
//
// The dialect is Draft 07, not 2019-09 or 2020-12: the specification's Schema
// Object pins it with "$schema": "http://json-schema.org/draft-07/schema#" and
// descends from that metaschema. The difference is not cosmetic — Draft 07
// ignores the keywords beside a $ref, which is why an example or a constraint
// has to be written onto the referenced component rather than next to the
// reference (see message.PayloadExamples for the one place that bites).
//
// Derivation rules (per the locked design decisions):
//
//   - Named struct types are always hoisted into the provided defs map under
//     the components key derived from their fully-qualified name
//     ("pkgPath.TypeName" sanitized by ComponentKey, e.g.
//     "github.com_acme_orders.OrderPlaced") and referenced via $ref into
//     "#/components/schemas/...".
//   - Anonymous embedded structs are flattened, matching encoding/json, unless
//     tagged asyncapi:"allOf", in which case the embedded type is composed via
//     allOf instead (opt-in). An embedded struct carrying an explicit name in
//     its json tag (for example a Base field tagged json:"base") is a regular
//     named property instead — encoding/json does not inline it, and neither
//     does derivation.
//   - All fields are optional unless tagged asyncgo:"required".
//   - A nilable position — a pointer, slice, map, interface, chan or func —
//     derives a nullable schema, because encoding/json accepts null for all six
//     on unmarshal and emits it for a nil value whenever the key is present. The
//     shape is `oneOf: [<schema>, {type: "null"}]`: Draft-07 ignores keywords
//     beside $ref, so a pointer to a hoisted struct could not carry the null
//     branch any other way. A position whose schema already admits null — the
//     unconstrained {} of an interface, chan or func, or a provider/mapping that
//     returns one — is left alone, because `oneOf: [{}, {type: "null"}]` matches
//     every instance against two branches and so validates nothing.
//     Nullability belongs to the position, not to the type: a components.schemas
//     entry is never nullable, since `*T` and `T` share one component and the
//     wrapper is emitted at each use site instead. omitempty does not change
//     this — it governs only what asyncgo writes, not what a consumer may send.
//     Struct tags apply to the payload branch, never beside the oneOf, so an
//     `enum=` cannot silently exclude null. The combinator directives are the
//     one shape that is folded in rather than wrapped: `oneOf=` and `anyOf=`
//     take the null branch as a member of their own list, since a null instance
//     matches only that member and exactly-one (or any-of) still holds. `allOf=`
//     is wrapped, because every member must pass and a null member would leave
//     the position accepting nothing.
//   - json struct tags drive field names; "-" skips a field.
//   - asyncapi struct tags carry "required", "enum=a,b,...", "examples=...",
//     "format=...", "const=...", "discriminator=...", the numeric bounds
//     ("minItems=", "maxItems=", "minProperties=", "maxProperties="), and the
//     flags "readOnly", "writeOnly", "uniqueItems", "deprecated".
//     Combinator members are named as "oneOf=A,B": a bare name resolves against
//     the declaring struct's package, and a name containing "/" is a
//     fully-qualified pkgPath.TypeName. A semi-qualified name (pkg.Type, with no
//     import path) is neither, so this pass would emit a dangling $ref — the
//     name is a generate-time error instead, because internal/discovery applies
//     the same rule when it collects the types to hoist and rejects a name the
//     generated harness could not emit.
//     Directives are separated by ";" and a list directive's members by ",", so a
//     scalar value may contain a comma.
//     Descriptions are read from the field's doc comment by
//     the generator's discovery pass (internal/discovery), since reflection
//     cannot see comments.
//   - A named type (struct or non-struct) implementing spec.SchemaProvider
//     overrides derivation: its AsyncAPISchema() result is hoisted under the
//     type's components key as-is. Non-struct kinds are hoisted only when
//     they carry a provider, so a plain `type ID string` stays inline.
//   - Byte slices are base64 strings by kind, so a named `type Raw []byte`
//     derives the same way []byte does.
//   - Declared mappings override derivation by exact type name: a mapping
//     registered with MapSchema/MapSchemaByName (or picked up from asyncgo.yaml,
//     or shipped built-in for a third-party type) replaces both the provider path
//     and kind derivation for that type, and the mapped schema is inlined at every
//     use site. time.Time derives date-time through exactly this path. A derived
//     alias (`type OrderID uuid.UUID`) inherits the base type's mapping unless it
//     declares its own, and a type's own provider outranks an inherited mapping.
package schema

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/RubenRibGarcia/asyncgo/internal/jsonpointer"
	"github.com/RubenRibGarcia/asyncgo/internal/taggrammar"
	"github.com/RubenRibGarcia/asyncgo/spec"
)

var (
	rawMessageType     = reflect.TypeFor[json.RawMessage]()
	schemaProviderType = reflect.TypeFor[spec.SchemaProvider]()
)

// Name returns the schema name a named type is keyed by in the document:
// the type's fully-qualified name (pkgPath.TypeName) with the import path
// sanitized into a components key (ComponentKey), so
// "github.com/acme/app/orders.OrderPlaced" becomes
// "github.com_acme_app_orders.OrderPlaced". The specification's key pattern is
// why: it admits no "/", and a key that misses it is not a key a components map
// may carry.
//
// It returns "" for an unnamed type: an anonymous struct, slice, map or
// interface has no stable name, so it cannot key a component schema. That is the
// same predicate isNamed enforces internally — only named types are ever
// hoisted.
func Name(t reflect.Type) string {
	if !isNamed(t) {
		return ""
	}
	fqn := t.PkgPath() + "." + t.Name()
	key := ComponentKey(fqn)
	claimComponentKey(key, fqn)
	return key
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
// resolve names as strings rather than reflect.Type. The name is sanitized to
// the components key the schema is hoisted under (ComponentKey), so a directive
// keeps naming the type the way Go does.
func RefByName(fqn string) string {
	return "#/components/schemas/" + jsonpointer.Escape(ComponentKey(fqn))
}

// FromType derives a JSON Schema for t. Named struct types, and named
// non-struct types implementing spec.SchemaProvider, are hoisted into defs
// (keyed by Name) and returned as a $ref; all other types are inlined.
//
// The returned schema describes a non-nil instance of t: nullability is a
// property of the position a type occupies, so FromType never wraps its own
// result, while the positions it fills — a struct field, `items`, and
// `additionalProperties` — do (see the package comment). A caller placing the
// result somewhere else inlines a schema that a nil value cannot satisfy.
//
// A nil t yields an unconstrained schema rather than a panic. defs must be
// non-nil whenever t is hoistable (a named struct or provider); FromType panics
// on a hoistable type with a nil defs map.
func FromType(t reflect.Type, defs map[string]*spec.Schema) *spec.Schema {
	if t == nil {
		return &spec.Schema{}
	}
	t = deref(t)

	// A declared mapping outranks everything below it: the document's author is
	// overriding both the type's own suggested default (its provider) and its Go
	// kind. The returned node is already a copy, so use sites cannot share it.
	if m := exactMapping(t); m != nil {
		return m
	}

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

	// An inherited mapping — the aliases of D8 — ranks below a provider, so a
	// mapped struct alias consults it inside hoistStruct, after that provider
	// check. Every other kind has no provider left to consult and resolves here.
	if t.Kind() != reflect.Struct {
		if m := mappingFor(t); m != nil {
			return m
		}
	}

	switch t {
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
		return &spec.Schema{
			Type:  "array",
			Items: nullable(t.Elem(), FromType(t.Elem(), defs), defs),
		}
	case reflect.Map:
		return &spec.Schema{
			Type:                 "object",
			AdditionalProperties: nullable(t.Elem(), FromType(t.Elem(), defs), defs),
		}
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
//
// Because it runs on entry, a pointer is indistinguishable from its element for
// the rest of derivation. So nullability cannot be decided from the type once
// derivation has started: each caller that places a schema at a value position
// passes the type it was given, still undereferenced, to nullable.
func deref(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

// nullSchema is the Draft-07 schema of JSON null, the second branch of a
// nullable position. It is never handed out as-is: nullNode copies it, so no two
// positions share one node.
var nullSchema = &spec.Schema{Type: "null"}

// nullNode returns a fresh copy of the null schema, so a node can be placed at
// one position without every other position aliasing it.
func nullNode() *spec.Schema {
	null := *nullSchema
	return &null
}

// nilableKind reports whether a value of type t can marshal to JSON null. All
// six kinds are also accepted as null by encoding/json on unmarshal, so each is
// nullable regardless of whether asyncgo itself would emit the null.
func nilableKind(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Map,
		reflect.Interface, reflect.Chan, reflect.Func:
		return true
	default:
		return false
	}
}

// nullable adds the null branch to s when the declared type of the position is
// nilable, yielding `oneOf: [s, {type: "null"}]`. s is returned unchanged
// otherwise, and when it already admits null: an unconstrained schema accepts
// every instance, so wrapping it would make exactly-one fail everywhere and
// leave the position accepting nothing at all.
//
// t must be the type as declared at the position. FromType dereferences on
// entry, so by the time a schema exists the pointer layer is gone.
func nullable(t reflect.Type, s *spec.Schema, defs map[string]*spec.Schema) *spec.Schema {
	if t == nil || !nilableKind(t) || !constrains(s, defs) {
		return s
	}
	return &spec.Schema{OneOf: []*spec.Schema{s, nullNode()}}
}

// nullableField adds the null branch to a field's schema in the shape the field
// allows. merged reports that a oneOf/anyOf directive built the combinator that
// s already carries: the null branch then joins that member list, which is
// equivalent — a null instance matches only the null member, so exactly-one and
// any-of still hold — and one nesting level shorter. Otherwise the wrapper
// nullable builds is used.
//
// allOf is deliberately not mergeable and is passed here with merged=false: every
// member of an allOf must pass, so appending {type: "null"} would require the
// instance to satisfy the named types AND be null, which nothing does. The
// wrapper is the only form that means "the conjunction, or null".
func nullableField(
	t reflect.Type,
	s *spec.Schema,
	defs map[string]*spec.Schema,
	merged bool,
) *spec.Schema {
	if !merged || t == nil || !nilableKind(t) {
		return nullable(t, s, defs)
	}
	if s.OneOf == nil && s.AnyOf == nil {
		return nullable(t, s, defs)
	}
	if s.OneOf != nil {
		s.OneOf = append(s.OneOf, nullNode())
	}
	if s.AnyOf != nil {
		s.AnyOf = append(s.AnyOf, nullNode())
	}
	return s
}

// constrains reports whether s accepts a strict subset of instances, which is
// what makes a null branch meaningful. The annotations are folded away first and
// the empty struct is the test, so the list to maintain is the set of keywords
// that restrict nothing — a newly modelled constraint counts as a constraint by
// default. That is the safe direction to be wrong in: the worst case is an
// unnecessary null branch, never an unsatisfiable one.
//
// A $ref constrains exactly when its target does, so the target is resolved one
// level through defs. Skipping that check would break the nilable case it is
// there to protect: a provider that returns an unconstrained schema is hoisted
// as an empty component, and `oneOf: [$ref, {type: "null"}]` would then match a
// null instance against both branches and reject it. Only one level is followed
// because a ref-to-ref is the only shape left, and treating it as a constraint
// is the conservative answer.
func constrains(s *spec.Schema, defs map[string]*spec.Schema) bool {
	if s == nil {
		return false
	}
	if name, ok := componentName(s.Ref); ok && defs != nil {
		target, ok := defs[name]
		if !ok || target == nil || target.Ref != "" {
			return true
		}
		return constrains(target, nil)
	}
	annotated := *s
	annotated.ID = ""
	annotated.SchemaURI = ""
	annotated.Title = ""
	annotated.Description = ""
	annotated.Comment = ""
	annotated.ExternalDocs = nil
	annotated.Examples = nil
	annotated.Default = nil
	annotated.ReadOnly = false
	annotated.WriteOnly = false
	annotated.Deprecated = false
	annotated.Discriminator = ""
	return !reflect.DeepEqual(annotated, spec.Schema{})
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

	// A mapping inherited through the alias chain (D8) ranks below the provider
	// just checked and above field derivation: `type Money2 SomeMappedStruct`
	// derives SomeMappedStruct's mapping under its own name.
	if m := mappingFor(t); m != nil {
		defs[Name(t)] = m
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
	// A struct that only composes embeds contributes no member of its own, and an
	// empty {type: object, properties: {}} among the allOf members constrains
	// nothing — every instance satisfies it. Omit it rather than emit noise.
	if len(own.Properties) == 0 {
		s.AllOf = embedded
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
				if !flattening &&
					taggrammar.Flag(taggrammar.Parse(f.Tag.Get("asyncapi")), "allOf") {
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
		// mergeNull records that a oneOf/anyOf directive built the combinator, which
		// lets the null branch join its member list instead of wrapping it.
		mergeNull := false
		if ds := taggrammar.Parse(f.Tag.Get("asyncapi")); len(ds) > 0 {
			applyTag(prop, ds)
			if taggrammar.Flag(ds, "required") {
				own.Required = append(own.Required, name)
			}
			if names, ok := combinatorNames(ds, "oneOf"); ok {
				prop.OneOf = refs(names, t)
				mergeNull = true
			}
			if names, ok := combinatorNames(ds, "anyOf"); ok {
				prop.AnyOf = refs(names, t)
				mergeNull = true
			}
			if names, ok := combinatorNames(ds, "allOf"); ok {
				prop.AllOf = refs(names, t)
			}
		}
		// The null branch is added last, so every keyword the tag supplied lands on
		// the payload branch. Beside a combinator they would apply to the whole
		// instance: `enum=a,b` would then exclude null while appearing to constrain
		// the value.
		own.Properties[name] = nullableField(f.Type, prop, defs, mergeNull)
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
	name, skip := taggrammar.JSONName(f.Tag.Get("json"))
	switch {
	case skip:
		return "", true
	case name == "":
		// The tag named no field (json:",omitempty"), so the Go name stands.
		return f.Name, false
	default:
		return name, false
	}
}

// hasExplicitJSONName reports whether the "json" tag names the field, which
// makes encoding/json treat an anonymous embedded field as a regular field
// rather than inlining it. Options alone (json:",omitempty") do not name it, and
// "-" does: it is a directive, not an inline request.
func hasExplicitJSONName(f reflect.StructField) bool {
	name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
	return name != ""
}

// ComponentKeyPattern is the pattern the AsyncAPI 3.1.0 specification applies to
// every key of a components fixed field — schemas, messages, securitySchemes and
// the rest of componentsObject — where it is expressed as patternProperties. The
// root package reuses this constant for the keys an author writes, so the two
// enforcement points cannot disagree.
const ComponentKeyPattern = `^[\w\d.\-_]+$`

// componentKeyPatternRE is the compiled form of ComponentKeyPattern. Derivation
// filters characters with componentKeyRune rather than a regexp, since Name runs
// on every type in a document; the pattern is kept compiled so the conformance
// test can hold the two to the same answer.
var componentKeyPatternRE = regexp.MustCompile(ComponentKeyPattern)

// ComponentKey returns the key a components fixed field uses for a fully
// qualified type name (pkgPath.TypeName, the form Name produces).
//
// An import path is not a valid components key: the specification's pattern
// admits letters, digits, dots, hyphens and underscores, and pkgPath carries
// "/". The key therefore keeps the pkgPath.TypeName shape and replaces every
// character the pattern rejects with "_":
//
//	github.com/acme/orders.OrderPlaced  ->  github.com_acme_orders.OrderPlaced
//
// The dot before the type name survives — it is admitted by the pattern — so the
// key stays readable and stays distinct per package and type. The transform is
// idempotent: an already conforming name comes back unchanged.
//
// It is lossy. Two import paths that differ only by "_" against "/" sanitize to
// one key (a/b_c.T and a_b/c.T both give a_b_c.T), so the second type would
// overwrite the first in the components map. Name records that case — it is the
// point where a key comes from a real Go type — and KeyCollisionErrors reports
// it instead of letting it pass silently.
func ComponentKey(fqn string) string {
	if !strings.ContainsFunc(fqn, func(r rune) bool { return !componentKeyRune(r) }) {
		return fqn
	}
	var b strings.Builder
	b.Grow(len(fqn) + 8)
	for _, r := range fqn {
		if componentKeyRune(r) {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('_')
	}
	return b.String()
}

// componentKeyRune reports whether r may appear in a components key. It is
// ComponentKeyPattern applied per rune, and every non-ASCII rune — a Unicode
// letter in an import path or a type name — is rejected: the pattern is ASCII.
func componentKeyRune(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return true
	case r == '.', r == '-', r == '_':
		return true
	default:
		return false
	}
}

// KeyCollision is one components key produced by two different fully qualified
// type names. The key can hold one schema, so the second name's type is not
// representable in the document.
type KeyCollision struct {
	Key    string // the shared components key
	First  string // the fully qualified name that claimed the key first
	Second string // the fully qualified name that also produced it
}

var (
	// componentKeyOwner maps a components key to the first fully qualified name
	// that produced it. It is what makes a collision detectable at all: the
	// transform drops the "/" that told the two names apart.
	componentKeyOwner = map[string]string{}

	// componentKeyCollisions collects the collisions detected since the last
	// drain, and collided marks the keys already recorded, so a derivation that
	// revisits a type does not append the same pair again.
	componentKeyCollisions []KeyCollision
	collided               = map[string]bool{}
)

// claimComponentKey records that fqn produced key, and notes a collision the
// first time a second name produces it. Only Name calls it: a name that reaches
// ComponentKey as a string — a combinator directive, a mapping, a $ref lookup —
// may already be the key, so it cannot be told apart from a different type's
// name and would report a collision against itself.
func claimComponentKey(key, fqn string) {
	if key == "" {
		return
	}
	owner, claimed := componentKeyOwner[key]
	if !claimed {
		componentKeyOwner[key] = fqn
		return
	}
	if owner == fqn || collided[key] {
		return
	}
	collided[key] = true
	componentKeyCollisions = append(
		componentKeyCollisions,
		KeyCollision{Key: key, First: owner, Second: fqn},
	)
}

// KeyCollisionErrors returns one error per components key that two different
// fully qualified type names produced, and clears the list so the same collision
// is reported once per drain window.
//
// It is drained rather than merely read because the report has to reach a caller
// that owns an error channel: builder.Spec joins it into a catalog's errors and
// Finalize returns it, so a document that would silently drop a schema fails
// instead. Detection is process-wide — Name claims the key where the derivation
// happens, which is the only place that knows a key came from a Go type.
func KeyCollisionErrors() []error {
	if len(componentKeyCollisions) == 0 {
		return nil
	}
	collisions := componentKeyCollisions
	componentKeyCollisions = nil
	clear(collided)
	errs := make([]error, 0, len(collisions))
	for _, c := range collisions {
		errs = append(errs, fmt.Errorf(
			"components key %q is claimed by both %s and %s",
			c.Key,
			c.First,
			c.Second,
		))
	}
	return errs
}

// collisionError is KeyCollisionErrors joined into a single error, for the
// callers whose signature carries one error rather than a list.
func collisionError() error {
	return errors.Join(KeyCollisionErrors()...)
}
