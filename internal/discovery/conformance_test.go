package discovery

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/RubenRibGarcia/asyncgo/schema"
	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The three types below are the corpus, not fixtures in the test/data sense: one
// declaration that both passes read. Every branch of the json grammar and one
// combinator directive appear here, so a divergence in either grammar fails
// TestTagGrammarConformance.
type conformanceMemberA struct{ A string }

type conformanceMemberB struct{ B string }

type conformanceFixture struct {
	Renamed     string `json:"renamed"`
	OptionsOnly string `json:",omitempty"`
	Defaulted   string
	Hidden      string `json:"-"`
	United      string `json:"united"     asyncapi:"oneOf=conformanceMemberA|conformanceMemberB"`
}

// TestTagGrammarConformance is the drift guard ADR 0007 promised: it runs the same
// tag corpus through both passes and requires them to agree. The grammar is shared
// today, so this passes by construction — its job is to fail the day someone
// re-implements a local split in either pass, which is exactly how the
// json:",omitempty" defect appeared.
//
// staticNames and staticMembers come from internal/discovery's go/ast path;
// derived comes from schema's reflection path.
func TestTagGrammarConformance(t *testing.T) {
	staticNames, staticMembers := conformanceStaticView(t)

	defs := map[string]*spec.Schema{}
	schema.FromType(reflect.TypeFor[conformanceFixture](), defs)
	obj := defs[schema.Name(reflect.TypeFor[conformanceFixture]())]
	require.NotNil(t, obj)

	t.Run("should_agree_on_the_json_field_names", func(t *testing.T) {
		derived := make([]string, 0, len(obj.Properties))
		for name := range obj.Properties {
			derived = append(derived, name)
		}
		sort.Strings(derived)

		want := append([]string(nil), staticNames...)
		sort.Strings(want)

		assert.Equal(t, want, derived)
		// The shape the two passes used to disagree on: a tag that names no field
		// falls back to the Go name instead of reading as json:"-".
		assert.Contains(t, derived, "OptionsOnly")
		assert.NotContains(t, derived, "Hidden")
	})

	t.Run("should_agree_on_the_combinator_members", func(t *testing.T) {
		require.Contains(t, obj.Properties, "united")

		// The reflection pass resolves each member name to a $ref. Naming the types
		// here is also what keeps them from being dead declarations: the tag
		// directive refers to them only as text.
		want := []string{
			schema.Ref(reflect.TypeFor[conformanceMemberA]()),
			schema.Ref(reflect.TypeFor[conformanceMemberB]()),
		}
		got := make([]string, 0, len(obj.Properties["united"].OneOf))
		for _, ref := range obj.Properties["united"].OneOf {
			got = append(got, ref.Ref)
		}
		assert.Equal(t, want, got)

		// The static pass resolved the same names out of the tag text.
		require.Len(t, staticMembers, len(want))
		for i, member := range staticMembers {
			assert.True(
				t,
				strings.HasSuffix(want[i], "."+member),
				"want %q should name %q",
				want[i],
				member,
			)
		}
	})
}

// conformanceStaticView reads the fixture's tags out of this file's AST, through
// the same helpers the CLI uses.
func conformanceStaticView(t *testing.T) (names, members []string) {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "conformance_test.go", nil, 0)
	require.NoError(t, err)

	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok || ts.Name.Name != "conformanceFixture" {
			return true
		}
		st, ok := ts.Type.(*ast.StructType)
		require.True(t, ok, "conformanceFixture must be a struct literal")
		found = true

		for _, field := range st.Fields.List {
			if len(field.Names) == 0 {
				continue // embedded; the corpus declares none
			}
			name := jsonFieldName(field, field.Names[0].Name)
			if name == "" {
				continue // json:"-"
			}
			names = append(names, name)

			if field.Tag == nil {
				continue
			}
			raw, err := strconv.Unquote(field.Tag.Value)
			require.NoError(t, err)
			if got, ok := combinatorNamesInTag(
				reflect.StructTag(raw).Get("asyncapi"),
				"oneOf",
			); ok {
				members = append(members, got...)
			}
		}
		return false
	})
	require.True(t, found, "conformanceFixture not found in this file's AST")

	return names, members
}
