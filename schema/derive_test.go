package schema

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Address struct {
	Street string `json:"street" asyncgo:"required"`
	City   string `json:"city"`
}

type Order struct {
	ID      string         `json:"id"             asyncgo:"required"`
	Amount  float64        `json:"amount"`
	Note    string         `json:"note,omitempty" asyncgo:"description=Optional note"`
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
	s := FromType(reflect.TypeOf(Order{}), defs)

	require.NotEmpty(t, s.Ref)
	assert.Equal(t, Ref(reflect.TypeOf(Order{})), s.Ref)

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

	// Note description from asyncgo tag.
	require.Contains(t, obj.Properties, "note")
	assert.Equal(t, "Optional note", obj.Properties["note"].Description)

	// json:"-" and unexported fields are skipped.
	assert.NotContains(t, obj.Properties, "Skip")
	assert.NotContains(t, obj.Properties, "hidden")
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
		name       string
		typ        reflect.Type
		wantType   string
		wantFormat string
	}{
		{
			name:       "should_return_date_time_for_time",
			typ:        reflect.TypeOf(time.Time{}),
			wantType:   "string",
			wantFormat: "date-time",
		},
		{
			name:       "should_return_byte_string_for_byte_slice",
			typ:        reflect.TypeOf([]byte{}),
			wantType:   "string",
			wantFormat: "byte",
		},
		{
			name:     "should_return_unconstrained_for_interface",
			typ:      reflect.TypeOf((*any)(nil)).Elem(),
			wantType: "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := FromType(tc.typ, defs)
			assert.Equal(t, tc.wantType, s.Type)
			assert.Equal(t, tc.wantFormat, s.Format)
		})
	}
}

func TestRecursiveStructTerminates(t *testing.T) {
	defs := map[string]*spec.Schema{}
	s := FromType(reflect.TypeOf(Node{}), defs)
	require.NotEmpty(t, s.Ref)
	assert.Contains(t, defs, "github.com/RubenRibGarcia/asyncgo/schema.Node")
}

func TestRefEscapesSlashes(t *testing.T) {
	typ := reflect.TypeOf(Order{})
	// Compute the expectation from the type's own package path so the test
	// survives module renames: JSON Pointer escaping must turn "/" into "~1".
	want := "#/components/schemas/" + strings.ReplaceAll(
		typ.PkgPath(),
		"/",
		"~1",
	) + "." + typ.Name()
	assert.Equal(t, want, Ref(typ))
}

func TestEscapePointer(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "should_escape_slash", in: "a/b", want: "a~1b"},
		{name: "should_escape_tilde", in: "a~b", want: "a~0b"},
		{name: "should_escape_slash_and_tilde", in: "a/b~c", want: "a~1b~0c"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, escapePointer(tc.in))
		})
	}
}

// Node is a self-referential type used to verify cycle termination.
type Node struct {
	Value    string `json:"value"`
	Children []Node `json:"children"`
}
