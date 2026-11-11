package discovery

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
)

// tagPackages builds the minimum a static pass needs: one parsed file, its
// FileSet for positions, and a non-main package so it is scannable.
func tagPackages(t *testing.T, src string) []*packages.Package {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "orders.go", src, 0)
	require.NoError(t, err)
	return []*packages.Package{{
		Name:    "orders",
		PkgPath: "example.com/orders",
		Fset:    fset,
		Syntax:  []*ast.File{file},
	}}
}

func TestCheckDirectives(t *testing.T) {
	tests := []struct {
		name string
		src  string
		// want is one substring per expected error; nil means the corpus is clean.
		want []string
	}{
		{
			name: "should_accept_a_legal_corpus",
			src: "package orders\n\ntype Order struct {\n" +
				"\tID    string `json:\"id\"     asyncapi:\"required format=uuid\"`\n" +
				"\tNote  string `json:\"note\"   asyncapi:\"deprecated examples=none\"`\n" +
				"\tItems []string `json:\"items\" asyncapi:\"minItems=1 uniqueItems\"`\n" +
				"}\n",
		},
		{
			// Only the asyncapi tag is ours; json options belong to encoding/json.
			name: "should_ignore_json_tag_options_entirely",
			src: "package orders\n\ntype Order struct {\n" +
				"\tID string `json:\"id,omitempty,string\"`\n" +
				"}\n",
		},
		{
			name: "should_ignore_a_field_with_no_tag",
			src:  "package orders\n\ntype Order struct {\n\tID string\n}\n",
		},
		{
			// A misspelled directive used to vanish; now it names the file and line.
			name: "should_report_a_typo_with_its_position",
			src: "package orders\n\ntype Order struct {\n" +
				"\tID string `json:\"id\" asyncapi:\"requird\"`\n" +
				"}\n",
			want: []string{
				`orders.go:4:`,
				`asyncapi:"requird": unknown directive "requird"; did you mean "required"?`,
			},
		},
		{
			name: "should_report_every_offending_field",
			src: "package orders\n\ntype Order struct {\n" +
				"\tID   string `json:\"id\"   asyncapi:\"requird\"`\n" +
				"\tNote string `json:\"note\" asyncapi:\"readOnly=true\"`\n" +
				"}\n",
			want: []string{
				`orders.go:4:`,
				`asyncapi:"requird": unknown directive "requird"`,
				`orders.go:5:`,
				`asyncapi:"readOnly=true": "readOnly" is a flag and takes no value`,
			},
		},
		{
			// The key is checked whatever the field's fate: a tag on a field the
			// document will not carry is still a tag somebody meant to write.
			name: "should_check_a_tag_on_a_skipped_field",
			src: "package orders\n\ntype Order struct {\n" +
				"\tHidden string `json:\"-\" asyncapi:\"requird\"`\n" +
				"}\n",
			want: []string{`asyncapi:"requird": unknown directive "requird"`},
		},
		{
			name: "should_name_the_whitespace_rule_for_the_legacy_comma_form",
			src: "package orders\n\ntype Order struct {\n" +
				"\tID string `json:\"id\" asyncapi:\"required,format=uuid\"`\n" +
				"}\n",
			want: []string{`directives are separated by spaces, not commas`},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := checkDirectives(tagPackages(t, tc.src))
			if len(tc.want) == 0 {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			for _, want := range tc.want {
				assert.Contains(t, err.Error(), want)
			}
		})
	}
}

// TestCheckDirectivesReportsEachViolationOnce pins the aggregation: one line per
// violation, so a catalog with many stale tags is fixed in one run rather than
// one run per tag.
func TestCheckDirectivesReportsEachViolationOnce(t *testing.T) {
	src := "package orders\n\ntype Order struct {\n" +
		"\tA string `asyncapi:\"requird\"`\n" +
		"\tB string `asyncapi:\"example=x\"`\n" +
		"\tC string `asyncapi:\"zzz\"`\n" +
		"}\n"

	err := checkDirectives(tagPackages(t, src))
	require.Error(t, err)
	assert.Len(t, strings.Split(strings.TrimSpace(err.Error()), "\n"), 3,
		"want one line per violation")
}
