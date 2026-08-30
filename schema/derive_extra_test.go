package schema

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromTypeRawMessage(t *testing.T) {
	defs := map[string]*spec.Schema{}
	s := FromType(reflect.TypeFor[json.RawMessage](), defs)
	assert.Empty(t, s.Type) // unconstrained
}

func TestFromTypeArray(t *testing.T) {
	defs := map[string]*spec.Schema{}
	s := FromType(reflect.TypeFor[[3]string](), defs)
	assert.Equal(t, "array", s.Type)
	require.NotNil(t, s.Items)
	assert.Equal(t, "string", s.Items.Type)
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

// withUnexported has an unexported field that must be skipped during derivation.
type withUnexported struct {
	Public string `json:"public"`
	hidden string
}
