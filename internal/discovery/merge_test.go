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
