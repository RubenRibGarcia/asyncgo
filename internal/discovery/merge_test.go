package discovery

import (
	"testing"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeSkipsNilDocuments(t *testing.T) {
	out := Merge(nil)
	require.NotNil(t, out)
	assert.Equal(t, spec.Version, out.AsyncAPI)
	assert.Empty(t, out.Info.Title)
	assert.Nil(t, out.Components)

	doc := spec.New()
	doc.Info = spec.Info{Title: "T", Version: "1.0.0"}
	out = Merge(nil, doc, nil)
	assert.Equal(t, "T", out.Info.Title)
}

func TestMergeNilComponents(t *testing.T) {
	doc := spec.New()
	doc.Info = spec.Info{Title: "T", Version: "1.0.0"}

	out := Merge(doc)
	require.NotNil(t, out)
	assert.Equal(t, "T", out.Info.Title)
	assert.Nil(t, out.Components)
}

func TestMergeSecuritySchemes(t *testing.T) {
	first := spec.New()
	first.Info = spec.Info{Title: "T", Version: "1.0.0"}
	first.Components = &spec.Components{
		SecuritySchemes: map[string]*spec.SecurityScheme{
			"oauth": {Type: "oauth2"},
		},
	}

	second := spec.New()
	second.Info = spec.Info{Title: "U", Version: "1.0.0"}
	second.Components = &spec.Components{
		SecuritySchemes: map[string]*spec.SecurityScheme{
			"basic": {Type: "http", Scheme: "basic"},
			"oauth": {Type: "http", Scheme: "basic"},
		},
	}

	out := Merge(first, second)
	require.NotNil(t, out.Components)
	assert.Len(t, out.Components.SecuritySchemes, 2)
	assert.Equal(t, "oauth2", out.Components.SecuritySchemes["oauth"].Type, "first occurrence wins")
	assert.Equal(t, "basic", out.Components.SecuritySchemes["basic"].Scheme)
}
