package discovery

import (
	"go/ast"
	"testing"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
)

func TestJSONFieldName(t *testing.T) {
	tests := []struct {
		name     string
		field    *ast.Field
		fallback string
		want     string
	}{
		{
			name:     "should_return_fallback_when_no_tag",
			field:    &ast.Field{},
			fallback: "fallback",
			want:     "fallback",
		},
		{
			name:     "should_return_fallback_when_unquotable_tag",
			field:    &ast.Field{Tag: &ast.BasicLit{Value: "not quoted"}},
			fallback: "fallback",
			want:     "fallback",
		},
		{
			name:     "should_return_fallback_when_empty_json_tag",
			field:    &ast.Field{Tag: &ast.BasicLit{Value: "`json:\"\"`"}},
			fallback: "fallback",
			want:     "fallback",
		},
		{
			name:     "should_skip_when_dash",
			field:    &ast.Field{Tag: &ast.BasicLit{Value: "`json:\"-\"`"}},
			fallback: "fallback",
			want:     "",
		},
		{
			name:     "should_strip_options",
			field:    &ast.Field{Tag: &ast.BasicLit{Value: "`json:\"name,omitempty\"`"}},
			fallback: "fallback",
			want:     "name",
		},
		{
			name:     "should_return_plain_name",
			field:    &ast.Field{Tag: &ast.BasicLit{Value: "`json:\"plain\"`"}},
			fallback: "fallback",
			want:     "plain",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, jsonFieldName(tc.field, tc.fallback))
		})
	}
}

func TestFieldDoc(t *testing.T) {
	tests := []struct {
		name  string
		field *ast.Field
		want  string
	}{
		{
			name: "should_return_doc_comment",
			field: &ast.Field{
				Doc: &ast.CommentGroup{List: []*ast.Comment{{Text: "// the doc comment"}}},
			},
			want: "the doc comment",
		},
		{
			name: "should_return_line_comment_when_no_doc",
			field: &ast.Field{
				Comment: &ast.CommentGroup{List: []*ast.Comment{{Text: "// trailing"}}},
			},
			want: "trailing",
		},
		{
			name:  "should_return_empty_when_no_comment",
			field: &ast.Field{},
			want:  "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, fieldDoc(tc.field))
		})
	}
}

func TestApplyDescriptions(t *testing.T) {
	t.Run("should_be_noop_on_nil_document", func(t *testing.T) {
		assert.NotPanics(t, func() {
			applyDescriptions(nil, descriptions{"k": {"f": "d"}})
		})
	})

	t.Run("should_be_noop_when_no_components", func(t *testing.T) {
		doc := spec.New()
		applyDescriptions(doc, descriptions{"k": {"f": "d"}})
		assert.Nil(t, doc.Components)
	})

	t.Run("should_apply_description_to_matching_schema", func(t *testing.T) {
		doc := spec.New()
		doc.Components = &spec.Components{
			Schemas: map[string]*spec.Schema{
				"k": {Properties: map[string]*spec.Schema{"f": {Type: "string"}}},
			},
		}
		applyDescriptions(doc, descriptions{"k": {"f": "the description"}})
		assert.Equal(
			t,
			"the description",
			doc.Components.Schemas["k"].Properties["f"].Description,
		)
	})

	t.Run("should_skip_missing_schema", func(t *testing.T) {
		doc := spec.New()
		doc.Components = &spec.Components{Schemas: map[string]*spec.Schema{}}
		assert.NotPanics(t, func() {
			applyDescriptions(doc, descriptions{"unknown": {"f": "d"}})
		})
	})
}

func TestApplySchemaDescriptions(t *testing.T) {
	t.Run("should_be_noop_on_nil_schema", func(t *testing.T) {
		assert.NotPanics(t, func() {
			applySchemaDescriptions(nil, map[string]string{"f": "d"})
		})
	})

	t.Run("should_apply_to_properties", func(t *testing.T) {
		s := &spec.Schema{Properties: map[string]*spec.Schema{"f": {Type: "string"}}}
		applySchemaDescriptions(s, map[string]string{"f": "d"})
		assert.Equal(t, "d", s.Properties["f"].Description)
	})

	t.Run("should_recurse_into_allOf_members", func(t *testing.T) {
		s := &spec.Schema{
			AllOf: []*spec.Schema{
				{Properties: map[string]*spec.Schema{"own": {Type: "string"}}},
			},
		}
		applySchemaDescriptions(s, map[string]string{"own": "d"})
		assert.Equal(t, "d", s.AllOf[0].Properties["own"].Description)
	})
}
