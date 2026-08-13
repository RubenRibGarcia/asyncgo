package schema

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/RubenRibGarcia/asyncgo/spec"
)

type Address struct {
	Street string `json:"street" asyncgo:"required"`
	City   string `json:"city"`
}

type Order struct {
	ID      string         `json:"id" asyncgo:"required"`
	Amount  float64        `json:"amount"`
	Note    string         `json:"note,omitempty" asyncgo:"description=Optional note"`
	Address Address        `json:"address"`
	Tags    []string       `json:"tags"`
	Meta    map[string]int `json:"meta"`
	Skip    string         `json:"-"`
	hidden  string
}

const (
	orderKey   = "github.com/RubenRibGarcia/asyncgo/schema.Order"
	addressKey = "github.com/RubenRibGarcia/asyncgo/schema.Address"
)

func TestFromTypeHoistsNamedStructs(t *testing.T) {
	defs := map[string]*spec.Schema{}
	s := FromType(reflect.TypeOf(Order{}), defs)

	if s.Ref == "" {
		t.Fatalf("expected $ref for named struct, got %+v", s)
	}
	if want := Ref(reflect.TypeOf(Order{})); s.Ref != want {
		t.Errorf("unexpected ref: got %s, want %s", s.Ref, want)
	}

	obj, ok := defs[orderKey]
	if !ok {
		t.Fatalf("expected defs to contain %q, got keys %v", orderKey, keys(defs))
	}

	// Optional-by-default: only "id" is required.
	if len(obj.Required) != 1 || obj.Required[0] != "id" {
		t.Errorf("expected required=[id], got %v", obj.Required)
	}

	// Nested Address is hoisted and referenced.
	addrProp := obj.Properties["address"]
	if addrProp == nil || addrProp.Ref == "" {
		t.Errorf("expected address property to be a $ref, got %+v", addrProp)
	}
	if _, ok := defs[addressKey]; !ok {
		t.Errorf("expected Address to be hoisted")
	}

	// Note description from asyncgo tag.
	if got := obj.Properties["note"].Description; got != "Optional note" {
		t.Errorf("expected note description, got %q", got)
	}

	// json:"-" and unexported fields are skipped.
	if _, ok := obj.Properties["Skip"]; ok {
		t.Errorf("expected json:\"-\" field to be skipped")
	}
	if _, ok := obj.Properties["hidden"]; ok {
		t.Errorf("expected unexported field to be skipped")
	}
}

func TestFromTypeScalars(t *testing.T) {
	defs := map[string]*spec.Schema{}
	cases := []struct {
		v      any
		typ    string
		format string
	}{
		{"x", "string", ""},
		{true, "boolean", ""},
		{int(1), "integer", ""},
		{int64(1), "integer", ""},
		{float64(1), "number", "double"},
		{float32(1), "number", "float"},
	}
	for _, c := range cases {
		s := FromType(reflect.TypeOf(c.v), defs)
		if s.Type != c.typ {
			t.Errorf("%T: expected type %q, got %q", c.v, c.typ, s.Type)
		}
		if s.Format != c.format {
			t.Errorf("%T: expected format %q, got %q", c.v, c.format, s.Format)
		}
	}
}

func TestFromTypeCollections(t *testing.T) {
	defs := map[string]*spec.Schema{}

	arr := FromType(reflect.TypeOf([]string{}), defs)
	if arr.Type != "array" || arr.Items == nil || arr.Items.Type != "string" {
		t.Errorf("unexpected array schema: %+v", arr)
	}

	m := FromType(reflect.TypeOf(map[string]int{}), defs)
	if m.Type != "object" || m.AdditionalProperties == nil || m.AdditionalProperties.Type != "integer" {
		t.Errorf("unexpected map schema: %+v", m)
	}
}

func TestFromTypeSpecials(t *testing.T) {
	defs := map[string]*spec.Schema{}

	tt := FromType(reflect.TypeOf(time.Time{}), defs)
	if tt.Type != "string" || tt.Format != "date-time" {
		t.Errorf("unexpected time schema: %+v", tt)
	}

	b := FromType(reflect.TypeOf([]byte("x")), defs)
	if b.Type != "string" || b.Format != "byte" {
		t.Errorf("unexpected []byte schema: %+v", b)
	}

	any := FromType(reflect.TypeOf((*any)(nil)).Elem(), defs)
	if any.Type != "" && any.Ref != "" {
		t.Errorf("expected unconstrained schema for any, got %+v", any)
	}
}

func TestRecursiveStructTerminates(t *testing.T) {
	defs := map[string]*spec.Schema{}
	s := FromType(reflect.TypeOf(Node{}), defs)
	if s.Ref == "" {
		t.Fatalf("expected $ref for recursive named struct")
	}
	if _, ok := defs["github.com/RubenRibGarcia/asyncgo/schema.Node"]; !ok {
		t.Errorf("expected Node to be hoisted")
	}
}

func TestRefEscapesSlashes(t *testing.T) {
	typ := reflect.TypeOf(Order{})
	got := Ref(typ)
	// Compute the expectation from the type's own package path so the test
	// survives module renames: JSON Pointer escaping must turn "/" into "~1".
	want := "#/components/schemas/" + strings.ReplaceAll(typ.PkgPath(), "/", "~1") + "." + typ.Name()
	if got != want {
		t.Errorf("unexpected ref escaping:\n got %s\nwant %s", got, want)
	}
}

func TestEscapePointer(t *testing.T) {
	cases := map[string]string{
		"a/b":   "a~1b",
		"a~b":   "a~0b",
		"a/b~c": "a~1b~0c",
	}
	for in, want := range cases {
		if got := escapePointer(in); got != want {
			t.Errorf("escapePointer(%q) = %q, want %q", in, got, want)
		}
	}
}

// Node is a self-referential type used to verify cycle termination.
type Node struct {
	Value    string `json:"value"`
	Children []Node `json:"children"`
}

func keys(m map[string]*spec.Schema) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
