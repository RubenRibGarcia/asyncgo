package schema

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Address struct {
	Street string `json:"street" asyncapi:"required"`
	City   string `json:"city"`
}

type Order struct {
	ID      string         `json:"id"             asyncapi:"required"`
	Amount  float64        `json:"amount"`
	Note    string         `json:"note,omitempty"`
	Address Address        `json:"address"`
	Tags    []string       `json:"tags"`
	Meta    map[string]int `json:"meta"`
	Skip    string         `json:"-"`
}

const (
	orderKey   = "github.com/RubenRibGarcia/asyncgo/schema.Order"
	addressKey = "github.com/RubenRibGarcia/asyncgo/schema.Address"
)

func TestFromTypeHoistsNamedStructs(t *testing.T) {
	defs := map[string]*spec.Schema{}
	s := FromType(reflect.TypeFor[Order](), defs)

	require.NotEmpty(t, s.Ref)
	assert.Equal(t, Ref(reflect.TypeFor[Order]()), s.Ref)

	require.Contains(t, defs, orderKey)
	obj := defs[orderKey]

	// Optional-by-default: only "id" is required.
	assert.Equal(t, []string{"id"}, obj.Required)

	// Nested Address is hoisted and referenced.
	require.Contains(t, obj.Properties, "address")
	addrProp := obj.Properties["address"]
	require.NotNil(t, addrProp)
	assert.NotEmpty(t, addrProp.Ref)
	assert.Contains(t, defs, addressKey)

	// Note has no description: the reflection path cannot read field comments,
	// so descriptions are injected later by the discovery pass.
	require.Contains(t, obj.Properties, "note")
	assert.Empty(t, obj.Properties["note"].Description)

	// json:"-" and unexported fields are skipped.
	assert.NotContains(t, obj.Properties, "Skip")
	assert.NotContains(t, obj.Properties, "hidden")
}

func TestFromTypeInlineAnonymousStruct(t *testing.T) {
	defs := map[string]*spec.Schema{}
	typ := reflect.TypeOf(struct {
		Name string `json:"name"`
	}{})
	s := FromType(typ, defs)

	// Anonymous structs are inlined, not hoisted to a $ref.
	assert.Empty(t, s.Ref)
	assert.Equal(t, "object", s.Type)
	require.Contains(t, s.Properties, "name")
	assert.Equal(t, "string", s.Properties["name"].Type)
}

func TestFromTypeUnexportedFieldSkipped(t *testing.T) {
	// Touch the unexported field so the `unused` linter considers it used;
	// derivation must still skip it.
	_ = withUnexported{}.hidden

	defs := map[string]*spec.Schema{}
	s := FromType(reflect.TypeFor[withUnexported](), defs)

	obj := defs["github.com/RubenRibGarcia/asyncgo/schema.withUnexported"]
	require.NotNil(t, obj)
	assert.Contains(t, obj.Properties, "public")
	assert.NotContains(t, obj.Properties, "hidden")
	assert.NotEmpty(t, s.Ref)
}

func TestFromTypeScalars(t *testing.T) {
	defs := map[string]*spec.Schema{}
	tests := []struct {
		name   string
		v      any
		typ    string
		format string
	}{
		{name: "should_return_string", v: "x", typ: "string"},
		{name: "should_return_boolean", v: true, typ: "boolean"},
		{name: "should_return_integer_for_int", v: int(1), typ: "integer"},
		{name: "should_return_integer_for_int64", v: int64(1), typ: "integer"},
		{name: "should_return_double_for_float64", v: float64(1), typ: "number", format: "double"},
		{name: "should_return_float_for_float32", v: float32(1), typ: "number", format: "float"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := FromType(reflect.TypeOf(tc.v), defs)
			assert.Equal(t, tc.typ, s.Type)
			assert.Equal(t, tc.format, s.Format)
		})
	}
}

func TestFromTypeCollections(t *testing.T) {
	defs := map[string]*spec.Schema{}
	tests := []struct {
		name   string
		v      any
		verify func(*testing.T, *spec.Schema)
	}{
		{
			name: "should_return_array_schema_for_slice",
			v:    []string{},
			verify: func(t *testing.T, s *spec.Schema) {
				assert.Equal(t, "array", s.Type)
				require.NotNil(t, s.Items)
				assert.Equal(t, "string", s.Items.Type)
			},
		},
		{
			name: "should_return_array_schema_for_array",
			v:    [3]string{},
			verify: func(t *testing.T, s *spec.Schema) {
				assert.Equal(t, "array", s.Type)
				require.NotNil(t, s.Items)
				assert.Equal(t, "string", s.Items.Type)
			},
		},
		{
			name: "should_return_object_schema_for_map",
			v:    map[string]int{},
			verify: func(t *testing.T, s *spec.Schema) {
				assert.Equal(t, "object", s.Type)
				require.NotNil(t, s.AdditionalProperties)
				assert.Equal(t, "integer", s.AdditionalProperties.Type)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.verify(t, FromType(reflect.TypeOf(tc.v), defs))
		})
	}
}

func TestFromTypeSpecials(t *testing.T) {
	defs := map[string]*spec.Schema{}
	tests := []struct {
		name                string
		typ                 reflect.Type
		wantType            string
		wantFormat          string
		wantContentEncoding string
	}{
		{
			name:       "should_return_date_time_for_time",
			typ:        reflect.TypeFor[time.Time](),
			wantType:   "string",
			wantFormat: "date-time",
		},
		{
			name:                "should_return_byte_string_for_byte_slice",
			typ:                 reflect.TypeFor[[]byte](),
			wantType:            "string",
			wantFormat:          "byte",
			wantContentEncoding: "base64",
		},
		{
			name:     "should_return_unconstrained_for_interface",
			typ:      reflect.TypeFor[any](),
			wantType: "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := FromType(tc.typ, defs)
			assert.Equal(t, tc.wantType, s.Type)
			assert.Equal(t, tc.wantFormat, s.Format)
			assert.Equal(t, tc.wantContentEncoding, s.ContentEncoding)
			assert.Empty(t, s.ContentMediaType, "a Go type carries no media type")
		})
	}
}

func TestFromTypeRawMessage(t *testing.T) {
	defs := map[string]*spec.Schema{}
	s := FromType(reflect.TypeFor[json.RawMessage](), defs)
	assert.Empty(t, s.Type) // unconstrained
}

func TestRecursiveStructTerminates(t *testing.T) {
	defs := map[string]*spec.Schema{}
	s := FromType(reflect.TypeFor[Node](), defs)
	require.NotEmpty(t, s.Ref)
	assert.Contains(t, defs, "github.com/RubenRibGarcia/asyncgo/schema.Node")
}

func TestAllOfEmbedding(t *testing.T) {
	const (
		composedKey = "github.com/RubenRibGarcia/asyncgo/schema.AllOfComposed"
		flatKey     = "github.com/RubenRibGarcia/asyncgo/schema.FlatDefault"
		topKey      = "github.com/RubenRibGarcia/asyncgo/schema.TopFlatten"
		baseKey     = "github.com/RubenRibGarcia/asyncgo/schema.AllOfBase"
		wrapperKey  = "github.com/RubenRibGarcia/asyncgo/schema.RecursiveAllOfWrapper"
	)
	tests := []struct {
		name   string
		typ    reflect.Type
		verify func(t *testing.T, defs map[string]*spec.Schema)
	}{
		{
			name: "should_compose_embedded_struct_with_allOf",
			typ:  reflect.TypeFor[AllOfComposed](),
			verify: func(t *testing.T, defs map[string]*spec.Schema) {
				composed := defs[composedKey]
				require.NotNil(t, composed)
				require.Len(t, composed.AllOf, 2)
				assert.Equal(t, Ref(reflect.TypeFor[AllOfBase]()), composed.AllOf[0].Ref)
				own := composed.AllOf[1]
				require.NotNil(t, own)
				assert.Equal(t, "object", own.Type)
				assert.Contains(t, own.Properties, "amount")
			},
		},
		{
			name: "should_flatten_embedded_struct_by_default",
			typ:  reflect.TypeFor[FlatDefault](),
			verify: func(t *testing.T, defs map[string]*spec.Schema) {
				obj := defs[flatKey]
				require.NotNil(t, obj)
				assert.Empty(t, obj.AllOf)
				assert.Contains(t, obj.Properties, "id")
				assert.Contains(t, obj.Properties, "note")
				assert.Equal(t, []string{"id"}, obj.Required)
			},
		},
		{
			name: "should_keep_required_local_per_allOf_member",
			typ:  reflect.TypeFor[AllOfComposed](),
			verify: func(t *testing.T, defs map[string]*spec.Schema) {
				composed := defs[composedKey]
				require.NotNil(t, composed)
				assert.Empty(t, composed.Required)
				require.Len(t, composed.AllOf, 2)
				assert.Equal(t, []string{"amount"}, composed.AllOf[1].Required)
				base := defs[baseKey]
				require.NotNil(t, base)
				assert.Equal(t, []string{"id"}, base.Required)
			},
		},
		{
			name: "should_flatten_marked_base_fully_when_unmarked",
			typ:  reflect.TypeFor[TopFlatten](),
			verify: func(t *testing.T, defs map[string]*spec.Schema) {
				obj := defs[topKey]
				require.NotNil(t, obj)
				assert.Empty(t, obj.AllOf)
				assert.Contains(t, obj.Properties, "id")
				assert.Contains(t, obj.Properties, "extra")
				assert.Contains(t, obj.Properties, "own")
			},
		},
		{
			name: "should_terminate_on_recursive_allOf",
			typ:  reflect.TypeFor[RecursiveAllOfWrapper](),
			verify: func(t *testing.T, defs map[string]*spec.Schema) {
				wrapper := defs[wrapperKey]
				require.NotNil(t, wrapper)
				require.Len(t, wrapper.AllOf, 2)
				assert.Equal(t, Ref(reflect.TypeFor[RecursiveAllOfNode]()), wrapper.AllOf[0].Ref)
				assert.Contains(
					t,
					defs,
					"github.com/RubenRibGarcia/asyncgo/schema.RecursiveAllOfNode",
				)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defs := map[string]*spec.Schema{}
			FromType(tc.typ, defs)
			tc.verify(t, defs)
		})
	}
}

func TestCombinatorDirectives(t *testing.T) {
	const key = "github.com/RubenRibGarcia/asyncgo/schema.CombinatorHolder"
	tests := []struct {
		name     string
		field    string
		accessor func(*spec.Schema) []*spec.Schema
	}{
		{
			name:  "should_emit_oneOf_refs",
			field: "one",
			accessor: func(s *spec.Schema) []*spec.Schema {
				return s.OneOf
			},
		},
		{
			name:  "should_emit_anyOf_refs",
			field: "any",
			accessor: func(s *spec.Schema) []*spec.Schema {
				return s.AnyOf
			},
		},
		{
			name:  "should_emit_allOf_refs",
			field: "all",
			accessor: func(s *spec.Schema) []*spec.Schema {
				return s.AllOf
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defs := map[string]*spec.Schema{}
			FromType(reflect.TypeFor[CombinatorHolder](), defs)
			obj := defs[key]
			require.NotNil(t, obj)
			prop := obj.Properties[tc.field]
			require.NotNil(t, prop)
			got := tc.accessor(prop)
			require.Len(t, got, 2)
			assert.Equal(
				t,
				RefByName("github.com/RubenRibGarcia/asyncgo/schema.UnionMemberA"),
				got[0].Ref,
			)
			assert.Equal(
				t,
				RefByName("github.com/RubenRibGarcia/asyncgo/schema.UnionMemberB"),
				got[1].Ref,
			)
		})
	}
}

func TestCombinatorNameResolution(t *testing.T) {
	t.Run("should_resolve_same_package_short_name", func(t *testing.T) {
		defs := map[string]*spec.Schema{}
		FromType(reflect.TypeFor[UnionHolder](), defs)
		obj := defs["github.com/RubenRibGarcia/asyncgo/schema.UnionHolder"]
		require.NotNil(t, obj)
		prop := obj.Properties["data"]
		require.NotNil(t, prop)
		require.Len(t, prop.OneOf, 2)
		assert.Equal(
			t,
			RefByName("github.com/RubenRibGarcia/asyncgo/schema.UnionMemberA"),
			prop.OneOf[0].Ref,
		)
	})

	t.Run("should_pass_through_fully_qualified_name", func(t *testing.T) {
		defs := map[string]*spec.Schema{}
		FromType(reflect.TypeFor[FQNUnionHolder](), defs)
		obj := defs["github.com/RubenRibGarcia/asyncgo/schema.FQNUnionHolder"]
		require.NotNil(t, obj)
		prop := obj.Properties["data"]
		require.NotNil(t, prop)
		require.Len(t, prop.OneOf, 1)
		assert.Equal(t, RefByName("github.com/acme/orders.OrderPlaced"), prop.OneOf[0].Ref)
	})
}

func TestFinalizeHoistsRegisteredTypes(t *testing.T) {
	t.Run("should_hoist_referenced_types_on_finalize", func(t *testing.T) {
		Register(UnionMemberA{}, UnionMemberB{})
		doc := spec.New()
		Finalize(doc)

		require.NotNil(t, doc.Components)
		require.Contains(
			t,
			doc.Components.Schemas,
			"github.com/RubenRibGarcia/asyncgo/schema.UnionMemberA",
		)
		require.Contains(
			t,
			doc.Components.Schemas,
			"github.com/RubenRibGarcia/asyncgo/schema.UnionMemberB",
		)
		a := doc.Components.Schemas["github.com/RubenRibGarcia/asyncgo/schema.UnionMemberA"]
		require.NotNil(t, a)
		assert.Contains(t, a.Properties, "a")
		assert.Equal(t, []string{"a"}, a.Required)
	})
}

func TestJSONNamePlainTag(t *testing.T) {
	name, skip := jsonName(reflect.StructField{Name: "Plain", Tag: `json:"plain"`})
	assert.False(t, skip)
	assert.Equal(t, "plain", name)
}

func TestStructOf(t *testing.T) {
	tests := []struct {
		name string
		typ  reflect.Type
		want reflect.Type
		ok   bool
	}{
		{
			name: "should_return_struct",
			typ:  reflect.TypeFor[Order](),
			want: reflect.TypeFor[Order](),
			ok:   true,
		},
		{
			name: "should_dereference_pointer_to_struct",
			typ:  reflect.TypeFor[*Order](),
			want: reflect.TypeFor[Order](),
			ok:   true,
		},
		{
			name: "should_return_false_for_non_struct",
			typ:  reflect.TypeFor[int](),
			ok:   false,
		},
		{
			name: "should_return_false_for_pointer_to_non_struct",
			typ:  reflect.TypeFor[*int](),
			ok:   false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := structOf(tc.typ)
			assert.Equal(t, tc.ok, ok)
			if tc.ok {
				assert.Equal(t, tc.want, got)
			}
		})
	}
}

func TestRefEscapesSlashes(t *testing.T) {
	typ := reflect.TypeFor[Order]()
	// Compute the expectation from the type's own package path so the test
	// survives module renames: JSON Pointer escaping must turn "/" into "~1".
	want := "#/components/schemas/" + strings.ReplaceAll(
		typ.PkgPath(),
		"/",
		"~1",
	) + "." + typ.Name()
	assert.Equal(t, want, Ref(typ))
}

// withUnexported has an unexported field that must be skipped during derivation.
type withUnexported struct {
	Public string `json:"public"`
	hidden string
}

// Node is a self-referential type used to verify cycle termination.
type Node struct {
	Value    string `json:"value"`
	Children []Node `json:"children"`
}

// AllOfBase is the shared base composed via allOf in the embedding tests.
type AllOfBase struct {
	ID string `json:"id" asyncapi:"required"`
}

// AllOfComposed opts into allOf composition of AllOfBase.
type AllOfComposed struct {
	AllOfBase `        asyncapi:"allOf"`
	Amount    float64 `asyncapi:"required" json:"amount"`
}

// FlatDefault embeds AllOfBase without a marker, so it is flattened.
type FlatDefault struct {
	AllOfBase
	Note string `json:"note"`
}

// MidMarked marks its own embedded base as allOf.
type MidMarked struct {
	AllOfBase `       asyncapi:"allOf"`
	Extra     string `                 json:"extra"`
}

// TopFlatten embeds MidMarked unmarked: MidMarked's allOf marker is ignored
// and everything is flattened into TopFlatten.
type TopFlatten struct {
	MidMarked
	Own string `json:"own"`
}

// RecursiveAllOfNode is a recursive type composed via allOf by the wrapper.
type RecursiveAllOfNode struct {
	Value string              `json:"value"`
	Next  *RecursiveAllOfNode `json:"next"`
}

// RecursiveAllOfWrapper allOf-composes a recursive type.
type RecursiveAllOfWrapper struct {
	RecursiveAllOfNode `       asyncapi:"allOf"`
	Label              string `                 json:"label"`
}

// UnionMemberA is one arm of a tag-driven union.
type UnionMemberA struct {
	A string `json:"a" asyncapi:"required"`
}

// UnionMemberB is another arm of a tag-driven union.
type UnionMemberB struct {
	B string `json:"b"`
}

// UnionHolder references two same-package types by short name.
type UnionHolder struct {
	Data any `json:"data" asyncapi:"oneOf=UnionMemberA|UnionMemberB"`
}

// FQNUnionHolder references a type by fully-qualified name.
type FQNUnionHolder struct {
	Data any `json:"data" asyncapi:"oneOf=github.com/acme/orders.OrderPlaced"`
}

// CombinatorHolder exercises oneOf, anyOf, and allOf directives side by side.
type CombinatorHolder struct {
	One any `json:"one" asyncapi:"oneOf=UnionMemberA|UnionMemberB"`
	Any any `json:"any" asyncapi:"anyOf=UnionMemberA|UnionMemberB"`
	All any `json:"all" asyncapi:"allOf=UnionMemberA|UnionMemberB"`
}

// CustomString is a SchemaProvider on the value receiver.
type CustomString struct {
	Raw string `json:"raw"`
}

func (CustomString) AsyncAPISchema() *spec.Schema {
	return &spec.Schema{Type: "string", Format: "uuid"}
}

// CustomPointer is a SchemaProvider on the pointer receiver.
type CustomPointer struct {
	Raw string `json:"raw"`
}

func (*CustomPointer) AsyncAPISchema() *spec.Schema {
	return &spec.Schema{Type: "integer"}
}

// NilProvider returns nil, so reflection derivation must be used.
type NilProvider struct {
	Value string `json:"value" asyncapi:"required"`
}

func (NilProvider) AsyncAPISchema() *spec.Schema { return nil }

// EmptyProvider returns an empty (non-nil) schema → unconstrained.
type EmptyProvider struct {
	Value string `json:"value"`
}

func (EmptyProvider) AsyncAPISchema() *spec.Schema { return &spec.Schema{} }

// PlainStruct is a non-provider control for fall-back behavior.
type PlainStruct struct {
	Value string `json:"value" asyncapi:"required"`
}

func TestSchemaProvider(t *testing.T) {
	const (
		customStringKey  = "github.com/RubenRibGarcia/asyncgo/schema.CustomString"
		customPointerKey = "github.com/RubenRibGarcia/asyncgo/schema.CustomPointer"
		nilProviderKey   = "github.com/RubenRibGarcia/asyncgo/schema.NilProvider"
		emptyProviderKey = "github.com/RubenRibGarcia/asyncgo/schema.EmptyProvider"
		plainKey         = "github.com/RubenRibGarcia/asyncgo/schema.PlainStruct"
	)

	tests := []struct {
		name   string
		typ    reflect.Type
		verify func(*testing.T, *spec.Schema, map[string]*spec.Schema)
	}{
		{
			name: "should_use_custom_schema_for_provider_struct",
			typ:  reflect.TypeFor[CustomString](),
			verify: func(t *testing.T, s *spec.Schema, defs map[string]*spec.Schema) {
				require.NotEmpty(t, s.Ref)
				assert.Equal(t, Ref(reflect.TypeFor[CustomString]()), s.Ref)
				custom := defs[customStringKey]
				require.NotNil(t, custom)
				assert.Equal(t, "string", custom.Type)
				assert.Equal(t, "uuid", custom.Format)
			},
		},
		{
			name: "should_hoist_custom_schema_under_fqn",
			typ:  reflect.TypeFor[CustomString](),
			verify: func(t *testing.T, _ *spec.Schema, defs map[string]*spec.Schema) {
				require.Contains(t, defs, customStringKey)
			},
		},
		{
			name: "should_detect_pointer_receiver_provider",
			typ:  reflect.TypeFor[CustomPointer](),
			verify: func(t *testing.T, _ *spec.Schema, defs map[string]*spec.Schema) {
				custom := defs[customPointerKey]
				require.NotNil(t, custom)
				assert.Equal(t, "integer", custom.Type)
			},
		},
		{
			name: "should_fall_back_to_reflection_on_nil_schema",
			typ:  reflect.TypeFor[NilProvider](),
			verify: func(t *testing.T, _ *spec.Schema, defs map[string]*spec.Schema) {
				obj := defs[nilProviderKey]
				require.NotNil(t, obj)
				assert.Contains(t, obj.Properties, "value")
				assert.Equal(t, []string{"value"}, obj.Required)
			},
		},
		{
			name: "should_fall_back_to_reflection_when_not_provider",
			typ:  reflect.TypeFor[PlainStruct](),
			verify: func(t *testing.T, _ *spec.Schema, defs map[string]*spec.Schema) {
				obj := defs[plainKey]
				require.NotNil(t, obj)
				assert.Contains(t, obj.Properties, "value")
				assert.Equal(t, []string{"value"}, obj.Required)
			},
		},
		{
			name: "should_hoist_empty_schema_as_unconstrained",
			typ:  reflect.TypeFor[EmptyProvider](),
			verify: func(t *testing.T, _ *spec.Schema, defs map[string]*spec.Schema) {
				custom := defs[emptyProviderKey]
				require.NotNil(t, custom)
				assert.Empty(t, custom.Type)
				assert.Empty(t, custom.Properties)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defs := map[string]*spec.Schema{}
			s := FromType(tc.typ, defs)
			tc.verify(t, s, defs)
		})
	}
}

func TestSchemaProviderFinalize(t *testing.T) {
	t.Run("should_honor_provider_via_finalize", func(t *testing.T) {
		Register(CustomString{})
		doc := spec.New()
		Finalize(doc)

		require.NotNil(t, doc.Components)
		require.Contains(
			t,
			doc.Components.Schemas,
			"github.com/RubenRibGarcia/asyncgo/schema.CustomString",
		)
		custom := doc.Components.Schemas["github.com/RubenRibGarcia/asyncgo/schema.CustomString"]
		require.NotNil(t, custom)
		assert.Equal(t, "string", custom.Type)
		assert.Equal(t, "uuid", custom.Format)
	})
}

// Named non-struct providers (coverage doc §B16): a named scalar or collection
// implementing spec.SchemaProvider is hoisted under its fully-qualified name
// exactly like a named struct, so its constraint is reusable from every use site.

// OrderID is a named string provider: the wire value is a constrained string.
type OrderID string

func (OrderID) AsyncAPISchema() *spec.Schema {
	minLen := uint64(10)
	return &spec.Schema{
		Type:      "string",
		Pattern:   `^ord_[0-9]{6}$`,
		MinLength: &minLen,
		Examples:  []any{"ord_000123"},
	}
}

// Tick is a named integer whose wire form is a duration string. The provider is
// on the pointer receiver, so detection must probe reflect.PointerTo.
type Tick int64

func (*Tick) AsyncAPISchema() *spec.Schema {
	return &spec.Schema{Type: "string", Format: "duration"}
}

// Tags is a named slice provider.
type Tags []string

func (Tags) AsyncAPISchema() *spec.Schema {
	return &spec.Schema{
		Type:        "array",
		Items:       &spec.Schema{Type: "string"},
		UniqueItems: true,
	}
}

// Labels is a named map provider.
type Labels map[string]string

func (Labels) AsyncAPISchema() *spec.Schema {
	return &spec.Schema{
		Type:                 "object",
		AdditionalProperties: &spec.Schema{Type: "string"},
	}
}

// NilStringProvider returns nil, so a non-struct must still derive inline.
type NilStringProvider string

func (NilStringProvider) AsyncAPISchema() *spec.Schema { return nil }

// PlainID is a named non-struct control that does not implement the provider.
type PlainID string

// Providerish is an interface whose method set includes AsyncAPISchema. Its zero
// value is a nil interface, so the provider probe must skip the kind rather than
// assert nil and panic.
type Providerish interface {
	AsyncAPISchema() *spec.Schema
}

// NamedProviderHolder reaches the same non-struct provider twice, so the hoist
// must happen once and every use site must be a $ref.
type NamedProviderHolder struct {
	Primary   OrderID `json:"primary"   asyncapi:"required"`
	Secondary OrderID `json:"secondary"`
}

func TestFromTypeHoistsNamedNonStructProviders(t *testing.T) {
	tests := []struct {
		name   string
		typ    reflect.Type
		verify func(*testing.T, *spec.Schema, map[string]*spec.Schema)
	}{
		{
			name: "should_hoist_named_string_provider",
			typ:  reflect.TypeFor[OrderID](),
			verify: func(t *testing.T, s *spec.Schema, defs map[string]*spec.Schema) {
				t.Helper()
				require.NotEmpty(t, s.Ref)
				assert.Equal(t, Ref(reflect.TypeFor[OrderID]()), s.Ref)

				custom := defs[Name(reflect.TypeFor[OrderID]())]
				require.NotNil(t, custom)
				assert.Equal(t, "string", custom.Type)
				assert.Equal(t, `^ord_[0-9]{6}$`, custom.Pattern)
				require.NotNil(t, custom.MinLength)
				assert.Equal(t, uint64(10), *custom.MinLength)
			},
		},
		{
			name: "should_hoist_named_slice_provider",
			typ:  reflect.TypeFor[Tags](),
			verify: func(t *testing.T, s *spec.Schema, defs map[string]*spec.Schema) {
				t.Helper()
				assert.Equal(t, Ref(reflect.TypeFor[Tags]()), s.Ref)

				custom := defs[Name(reflect.TypeFor[Tags]())]
				require.NotNil(t, custom)
				assert.Equal(t, "array", custom.Type)
				assert.True(t, custom.UniqueItems)
				require.NotNil(t, custom.Items)
				assert.Equal(t, "string", custom.Items.Type)
			},
		},
		{
			name: "should_hoist_named_map_provider",
			typ:  reflect.TypeFor[Labels](),
			verify: func(t *testing.T, s *spec.Schema, defs map[string]*spec.Schema) {
				t.Helper()
				assert.Equal(t, Ref(reflect.TypeFor[Labels]()), s.Ref)

				custom := defs[Name(reflect.TypeFor[Labels]())]
				require.NotNil(t, custom)
				require.NotNil(t, custom.AdditionalProperties)
				assert.Equal(t, "string", custom.AdditionalProperties.Type)
			},
		},
		{
			name: "should_detect_pointer_receiver_on_named_non_struct",
			typ:  reflect.TypeFor[Tick](),
			verify: func(t *testing.T, s *spec.Schema, defs map[string]*spec.Schema) {
				t.Helper()
				assert.Equal(t, Ref(reflect.TypeFor[Tick]()), s.Ref)

				custom := defs[Name(reflect.TypeFor[Tick]())]
				require.NotNil(t, custom)
				assert.Equal(t, "string", custom.Type)
				assert.Equal(t, "duration", custom.Format)
			},
		},
		{
			name: "should_fall_back_to_inline_on_nil_provider",
			typ:  reflect.TypeFor[NilStringProvider](),
			verify: func(t *testing.T, s *spec.Schema, defs map[string]*spec.Schema) {
				t.Helper()
				assert.Empty(t, s.Ref)
				assert.Equal(t, "string", s.Type)
				assert.NotContains(t, defs, Name(reflect.TypeFor[NilStringProvider]()))
			},
		},
		{
			name: "should_leave_named_non_struct_inline_without_provider",
			typ:  reflect.TypeFor[PlainID](),
			verify: func(t *testing.T, s *spec.Schema, defs map[string]*spec.Schema) {
				t.Helper()
				assert.Empty(t, s.Ref)
				assert.Equal(t, "string", s.Type)
				assert.NotContains(t, defs, Name(reflect.TypeFor[PlainID]()))
			},
		},
		{
			name: "should_not_panic_on_interface_with_provider_method_set",
			typ:  reflect.TypeFor[Providerish](),
			verify: func(t *testing.T, s *spec.Schema, defs map[string]*spec.Schema) {
				t.Helper()
				assert.Empty(t, s.Type)
				assert.Empty(t, s.Ref)
				assert.NotContains(t, defs, Name(reflect.TypeFor[Providerish]()))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defs := map[string]*spec.Schema{}
			s := FromType(tc.typ, defs)
			tc.verify(t, s, defs)
		})
	}
}

func TestFromTypeReferencesNamedNonStructProviderFromField(t *testing.T) {
	defs := map[string]*spec.Schema{}
	s := FromType(reflect.TypeFor[NamedProviderHolder](), defs)

	assert.Equal(t, Ref(reflect.TypeFor[NamedProviderHolder]()), s.Ref)

	// Hoisted once, however many use sites reach it.
	assert.Len(t, defs, 2)
	require.Contains(t, defs, Name(reflect.TypeFor[OrderID]()))
	assert.Equal(t, "string", defs[Name(reflect.TypeFor[OrderID]())].Type)

	obj := defs[Name(reflect.TypeFor[NamedProviderHolder]())]
	require.NotNil(t, obj)
	require.Contains(t, obj.Properties, "primary")
	require.Contains(t, obj.Properties, "secondary")
	assert.Equal(t, Ref(reflect.TypeFor[OrderID]()), obj.Properties["primary"].Ref)
	assert.Equal(t, Ref(reflect.TypeFor[OrderID]()), obj.Properties["secondary"].Ref)
	assert.Equal(t, []string{"primary"}, obj.Required)
}

// NamedProviderUnion names a non-struct provider from a combinator directive, so
// the harness must register a type a composite literal cannot express.
type NamedProviderUnion struct {
	ID any `json:"id" asyncapi:"oneOf=github.com/RubenRibGarcia/asyncgo/schema.OrderID"`
}

func TestSchemaProviderNonStructFinalize(t *testing.T) {
	t.Run("should_hoist_registered_non_struct_provider", func(t *testing.T) {
		Register(OrderID(""))
		doc := spec.New()
		Finalize(doc)

		require.NotNil(t, doc.Components)
		custom := doc.Components.Schemas[Name(reflect.TypeFor[OrderID]())]
		require.NotNil(t, custom)
		assert.Equal(t, "string", custom.Type)
		assert.Equal(t, `^ord_[0-9]{6}$`, custom.Pattern)
	})

	t.Run("should_resolve_combinator_ref_to_non_struct_provider", func(t *testing.T) {
		Register(NamedProviderUnion{}, OrderID(""))
		doc := spec.New()
		Finalize(doc)

		require.NotNil(t, doc.Components)
		holder := doc.Components.Schemas[Name(reflect.TypeFor[NamedProviderUnion]())]
		require.NotNil(t, holder)

		oneOf := holder.Properties["id"].OneOf
		require.Len(t, oneOf, 1)
		assert.Equal(t, Ref(reflect.TypeFor[OrderID]()), oneOf[0].Ref)

		// The $ref resolves: the member schema exists in components.
		assert.NotNil(t, doc.Components.Schemas[Name(reflect.TypeFor[OrderID]())])
	})
}

// keywordTagged covers the Schema keywords that are derivable from a struct tag
// (docs/designdoc/schema-keywords.md, D6). The keywords that need a schema or an
// object — if/then/else, contains, propertyNames, patternProperties,
// dependencies, additionalItems, $id, $schema, externalDocs — are deliberately
// absent: they are authoring-only. So are minLength/maxLength/pattern, which
// stay reachable only through spec.SchemaProvider (coverage doc §4.3).
type keywordTagged struct {
	Kind  string   `json:"kind"  asyncapi:"required const=OrderPlaced discriminator=kind"`
	Items []string `json:"items" asyncapi:"minItems=1 maxItems=10 uniqueItems examples=sku-1 examples=sku-2"`
	Note  string   `json:"note"  asyncapi:"deprecated writeOnly"`
	ID    string   `json:"id"    asyncapi:"readOnly"`
	Meta  struct{} `json:"meta"  asyncapi:"minProperties=1 maxProperties=8"`
}

// TestFromTypeSchemaKeywordsDerivation walks the whole reflection path —
// FromType -> fillFields -> applyTag — for every derivable keyword, so the
// directives are proven reachable from a real struct and not only from a
// hand-built spec.Schema.
func TestFromTypeSchemaKeywordsDerivation(t *testing.T) {
	defs := map[string]*spec.Schema{}
	s := FromType(reflect.TypeFor[keywordTagged](), defs)

	key := "github.com/RubenRibGarcia/asyncgo/schema.keywordTagged"
	require.Contains(t, defs, key)
	require.NotEmpty(t, s.Ref)

	obj := defs[key]
	require.Equal(t, []string{"kind"}, obj.Required)

	kind := obj.Properties["kind"]
	require.Equal(t, "OrderPlaced", kind.Const)
	assert.Equal(t, "kind", kind.Discriminator)

	items := obj.Properties["items"]
	require.NotNil(t, items.MinItems)
	require.NotNil(t, items.MaxItems)
	assert.Equal(t, uint64(1), *items.MinItems)
	assert.Equal(t, uint64(10), *items.MaxItems)
	assert.True(t, items.UniqueItems)
	assert.Equal(t, []any{"sku-1", "sku-2"}, items.Examples)

	note := obj.Properties["note"]
	assert.True(t, note.Deprecated)
	assert.True(t, note.WriteOnly)

	assert.True(t, obj.Properties["id"].ReadOnly)

	meta := obj.Properties["meta"]
	require.NotNil(t, meta.MinProperties)
	require.NotNil(t, meta.MaxProperties)
	assert.Equal(t, uint64(1), *meta.MinProperties)
	assert.Equal(t, uint64(8), *meta.MaxProperties)
}
