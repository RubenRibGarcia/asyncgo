package schema

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// nilableInner is the hoisted struct that nullable pointer fields reference.
type nilableInner struct {
	Name string `json:"name"`
}

// unconstrainedProvider is a SchemaProvider that says "any instance will do".
// It is hoisted as an empty component, which is the one shape where a $ref hides
// that its target admits null.
type unconstrainedProvider struct {
	Raw string `json:"raw"`
}

func (unconstrainedProvider) AsyncAPISchema() *spec.Schema { return &spec.Schema{} }

// nilableRef is a named pointer type: nullability follows the declared type of
// the position, not a '*' written at the field.
type nilableRef *nilableInner

// nilableHolder places every nilable kind in a field position, plus the
// non-nilable and already-null-accepting controls that must stay untouched.
type nilableHolder struct {
	Str      *string            `json:"str"`
	Deep     **string           `json:"deep"`
	Inner    *nilableInner      `json:"inner"`
	InnerPtr []*nilableInner    `json:"inner_ptrs"`
	StrPtrs  []*string          `json:"str_ptrs"`
	MapPtrs  map[string]*string `json:"map_ptrs"`
	PtrSlice *[]string          `json:"ptr_slice"`
	PtrMap   *map[string]int    `json:"ptr_map"`
	Strs     []string           `json:"strs"`
	Bytes    []byte             `json:"bytes"`
	Attrs    map[string]int     `json:"attrs"`
	AnyMap   map[string]any     `json:"any_map"`
	AnySlice []any              `json:"any_slice"`
	When     *time.Time         `json:"when"`
	Ref      nilableRef         `json:"ref"`
	Plain    time.Time          `json:"plain"`
	Omit     *string            `json:"omit,omitempty"`
	Any      any                `json:"any"`
	Raw      json.RawMessage    `json:"raw"`
	Ch       chan int           `json:"ch"`
	Fn       func()             `json:"fn"`
	Req      *string            `json:"req"            asyncapi:"required"`
	Bare     string             `json:"bare"`
}

// requireNullBranch asserts s is a nullable position — `oneOf: [payload,
// {type: "null"}]` — and returns its payload branch.
func requireNullBranch(t *testing.T, s *spec.Schema) *spec.Schema {
	t.Helper()
	require.NotNil(t, s)
	require.Len(t, s.OneOf, 2, "expected a nullable position")
	require.NotNil(t, s.OneOf[1])
	assert.Equal(t, "null", s.OneOf[1].Type)
	return s.OneOf[0]
}

// TestNilableFieldsAreNullable walks every nilable kind through a field
// position, so each one proves the derived schema accepts the null encoding/json
// produces for a nil value and the null it accepts on unmarshal.
func TestNilableFieldsAreNullable(t *testing.T) {
	defs := map[string]*spec.Schema{}
	FromType(reflect.TypeFor[nilableHolder](), defs)
	obj := defs[Name(reflect.TypeFor[nilableHolder]())]
	require.NotNil(t, obj)

	tests := []struct {
		name   string
		field  string
		verify func(t *testing.T, payload *spec.Schema)
	}{
		{
			name:  "should_wrap_a_pointer_scalar",
			field: "str",
			verify: func(t *testing.T, payload *spec.Schema) {
				assert.Equal(t, "string", payload.Type)
			},
		},
		{
			name:  "should_wrap_a_pointer_once_however_deep",
			field: "deep",
			verify: func(t *testing.T, payload *spec.Schema) {
				assert.Equal(t, "string", payload.Type)
				assert.Empty(t, payload.OneOf)
			},
		},
		{
			name:  "should_wrap_a_pointer_to_a_hoisted_struct",
			field: "inner",
			verify: func(t *testing.T, payload *spec.Schema) {
				assert.Equal(t, Ref(reflect.TypeFor[nilableInner]()), payload.Ref)
			},
		},
		{
			name:  "should_wrap_a_slice",
			field: "strs",
			verify: func(t *testing.T, payload *spec.Schema) {
				assert.Equal(t, "array", payload.Type)
				require.NotNil(t, payload.Items)
				assert.Equal(t, "string", payload.Items.Type)
				assert.Empty(t, payload.Items.OneOf, "a string element cannot be nil")
			},
		},
		{
			name:  "should_wrap_a_byte_slice",
			field: "bytes",
			verify: func(t *testing.T, payload *spec.Schema) {
				assert.Equal(t, "string", payload.Type)
				assert.Equal(t, "byte", payload.Format)
				assert.Equal(t, "base64", payload.ContentEncoding)
			},
		},
		{
			name:  "should_wrap_a_map",
			field: "attrs",
			verify: func(t *testing.T, payload *spec.Schema) {
				assert.Equal(t, "object", payload.Type)
				require.NotNil(t, payload.AdditionalProperties)
				assert.Equal(t, "integer", payload.AdditionalProperties.Type)
			},
		},
		{
			name:  "should_wrap_a_pointer_slice_element",
			field: "str_ptrs",
			verify: func(t *testing.T, payload *spec.Schema) {
				assert.Equal(t, "array", payload.Type)
				element := requireNullBranch(t, payload.Items)
				assert.Equal(t, "string", element.Type)
			},
		},
		{
			name:  "should_wrap_a_pointer_struct_element",
			field: "inner_ptrs",
			verify: func(t *testing.T, payload *spec.Schema) {
				assert.Equal(t, "array", payload.Type)
				element := requireNullBranch(t, payload.Items)
				assert.Equal(t, Ref(reflect.TypeFor[nilableInner]()), element.Ref)
			},
		},
		{
			name:  "should_wrap_a_pointer_map_value",
			field: "map_ptrs",
			verify: func(t *testing.T, payload *spec.Schema) {
				assert.Equal(t, "object", payload.Type)
				value := requireNullBranch(t, payload.AdditionalProperties)
				assert.Equal(t, "string", value.Type)
			},
		},
		{
			name:  "should_wrap_a_pointer_to_a_slice",
			field: "ptr_slice",
			verify: func(t *testing.T, payload *spec.Schema) {
				assert.Equal(t, "array", payload.Type)
				require.NotNil(t, payload.Items)
				assert.Equal(t, "string", payload.Items.Type)
			},
		},
		{
			name:  "should_wrap_a_pointer_to_a_map",
			field: "ptr_map",
			verify: func(t *testing.T, payload *spec.Schema) {
				assert.Equal(t, "object", payload.Type)
				require.NotNil(t, payload.AdditionalProperties)
				assert.Equal(t, "integer", payload.AdditionalProperties.Type)
			},
		},
		{
			name:  "should_keep_a_mapped_shape_on_the_payload_branch",
			field: "when",
			verify: func(t *testing.T, payload *spec.Schema) {
				assert.Equal(t, "string", payload.Type)
				assert.Equal(t, "date-time", payload.Format)
			},
		},
		{
			name:  "should_wrap_a_named_pointer_type",
			field: "ref",
			verify: func(t *testing.T, payload *spec.Schema) {
				assert.Equal(t, Ref(reflect.TypeFor[nilableInner]()), payload.Ref)
			},
		},
		{
			name:  "should_wrap_despite_omitempty",
			field: "omit",
			verify: func(t *testing.T, payload *spec.Schema) {
				assert.Equal(t, "string", payload.Type)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prop, ok := obj.Properties[tc.field]
			require.True(t, ok, "no property %q", tc.field)
			tc.verify(t, requireNullBranch(t, prop))
		})
	}

	t.Run("should_keep_a_required_pointer_in_required", func(t *testing.T) {
		// required is presence, nullable is value: a nil pointer still marshals to
		// null when the key is there, so both hold at once.
		assert.Contains(t, obj.Required, "req")
		assert.Equal(t, "string", requireNullBranch(t, obj.Properties["req"]).Type)
	})
}

// TestNilableLeavesNullAdmittingPositionsAlone pins the guard that makes the
// wrapper safe: an unconstrained schema already accepts null, and
// `oneOf: [{}, {type: "null"}]` would match every instance against both branches,
// leaving the position accepting nothing at all.
func TestNilableLeavesNullAdmittingPositionsAlone(t *testing.T) {
	defs := map[string]*spec.Schema{}
	FromType(reflect.TypeFor[nilableHolder](), defs)
	obj := defs[Name(reflect.TypeFor[nilableHolder]())]
	require.NotNil(t, obj)

	tests := []struct {
		name  string
		field string
	}{
		{name: "should_not_wrap_an_interface", field: "any"},
		{name: "should_not_wrap_json_raw_message", field: "raw"},
		{name: "should_not_wrap_a_chan", field: "ch"},
		{name: "should_not_wrap_a_func", field: "fn"},
		{name: "should_not_wrap_a_non_nilable_scalar", field: "bare"},
		{name: "should_not_wrap_a_non_nilable_struct", field: "plain"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prop, ok := obj.Properties[tc.field]
			require.True(t, ok, "no property %q", tc.field)
			assert.Empty(t, prop.OneOf, "position already admits null")
		})
	}

	t.Run("should_wrap_a_map_whose_values_admit_anything", func(t *testing.T) {
		// The map itself is nilable even though its values are unconstrained.
		payload := requireNullBranch(t, obj.Properties["any_map"])
		assert.Equal(t, "object", payload.Type)
		require.NotNil(t, payload.AdditionalProperties)
		assert.Empty(t, payload.AdditionalProperties.OneOf)
	})

	t.Run("should_wrap_a_slice_whose_elements_admit_anything", func(t *testing.T) {
		payload := requireNullBranch(t, obj.Properties["any_slice"])
		assert.Equal(t, "array", payload.Type)
		require.NotNil(t, payload.Items)
		assert.Empty(t, payload.Items.OneOf)
	})
}

// TestNilableLooksThroughARefToAnUnconstrainedComponent covers the one shape
// where the guard cannot see emptiness directly: a provider that returns {} is
// hoisted as an empty component, and the use site only holds a $ref to it.
func TestNilableLooksThroughARefToAnUnconstrainedComponent(t *testing.T) {
	type holder struct {
		Ptr *unconstrainedProvider `json:"ptr"`
	}

	defs := map[string]*spec.Schema{}
	FromType(reflect.TypeFor[holder](), defs)

	component := defs[Name(reflect.TypeFor[unconstrainedProvider]())]
	require.NotNil(t, component)
	assert.True(t, !constrains(component, defs), "the component accepts anything")

	obj := defs[Name(reflect.TypeFor[holder]())]
	require.NotNil(t, obj)
	prop := obj.Properties["ptr"]
	require.NotNil(t, prop)
	assert.Equal(t, Ref(reflect.TypeFor[unconstrainedProvider]()), prop.Ref)
	assert.Empty(t, prop.OneOf, "wrapping a ref to {} would reject null")
}

// TestNilableRootAndComponentsStayBare pins the boundary: nullability belongs to
// a value position, so the schema of a type itself — the FromType result and
// every components.schemas entry — is never nullable. *T and T share one
// component, so a nullable component would poison every non-nilable use of T.
func TestNilableRootAndComponentsStayBare(t *testing.T) {
	defs := map[string]*spec.Schema{}
	s := FromType(reflect.TypeFor[*nilableInner](), defs)

	require.NotEmpty(t, s.Ref)
	assert.Equal(t, Ref(reflect.TypeFor[nilableInner]()), s.Ref)
	assert.Empty(t, s.OneOf)

	component := defs[Name(reflect.TypeFor[nilableInner]())]
	require.NotNil(t, component)
	assert.Equal(t, "object", component.Type)
	assert.Empty(t, component.OneOf)
}

// TestNilableTagKeywordsLandOnThePayloadBranch pins D6: a keyword written beside
// the oneOf would apply to the whole instance, so `enum=` would silently exclude
// the null the wrapper just added.
func TestNilableTagKeywordsLandOnThePayloadBranch(t *testing.T) {
	type tagged struct {
		Note  *string `json:"note"  asyncapi:"enum=draft,sent;examples=draft"`
		Items *[]int  `json:"items" asyncapi:"minItems=1;maxItems=4;uniqueItems"`
	}

	defs := map[string]*spec.Schema{}
	FromType(reflect.TypeFor[tagged](), defs)
	obj := defs[Name(reflect.TypeFor[tagged]())]
	require.NotNil(t, obj)

	note := obj.Properties["note"]
	require.NotNil(t, note)
	assert.Empty(t, note.Enum, "enum would exclude null on the wrapper")
	assert.Empty(t, note.Examples)
	payload := requireNullBranch(t, note)
	assert.Equal(t, []any{"draft", "sent"}, payload.Enum)
	assert.Equal(t, []any{"draft"}, payload.Examples)

	items := requireNullBranch(t, obj.Properties["items"])
	require.NotNil(t, items.MinItems)
	require.NotNil(t, items.MaxItems)
	assert.Equal(t, uint64(1), *items.MinItems)
	assert.Equal(t, uint64(4), *items.MaxItems)
	assert.True(t, items.UniqueItems)
}

// TestNilableCombinatorDirectiveNests pins D7: the null branch wraps whatever the
// directive produced rather than being merged into its member list. Merging is
// wrong for allOf — every branch must pass, so a null member would make the
// position unsatisfiable — and nesting keeps the three directives uniform.
func TestNilableCombinatorDirectiveNests(t *testing.T) {
	type union struct {
		Data any `json:"data" asyncapi:"oneOf=nilableInner"`
	}

	defs := map[string]*spec.Schema{}
	FromType(reflect.TypeFor[union](), defs)
	obj := defs[Name(reflect.TypeFor[union]())]
	require.NotNil(t, obj)

	prop := obj.Properties["data"]
	require.NotNil(t, prop)
	require.Len(t, prop.OneOf, 2)
	assert.Equal(t, "null", prop.OneOf[1].Type)

	members := prop.OneOf[0].OneOf
	require.Len(t, members, 1)
	assert.Equal(t, Ref(reflect.TypeFor[nilableInner]()), members[0].Ref)
}

// TestNilableNeverSharesTheNullNode guards against handing one mutable node to
// every nullable position: a document that edited one would edit them all.
func TestNilableNeverSharesTheNullNode(t *testing.T) {
	defs := map[string]*spec.Schema{}
	FromType(reflect.TypeFor[nilableHolder](), defs)
	obj := defs[Name(reflect.TypeFor[nilableHolder]())]
	require.NotNil(t, obj)

	a := obj.Properties["str"].OneOf[1]
	b := obj.Properties["omit"].OneOf[1]
	require.NotNil(t, a)
	require.NotNil(t, b)
	assert.NotSame(t, a, b)
	assert.NotSame(t, a, nullSchema)
}
