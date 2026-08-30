package discovery

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuild(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("..", "..", "test", "data", "simple"))
	require.NoError(t, err)

	doc, n, err := Build(dir)
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	assert.Equal(t, "Orders Service", doc.Info.Title)
	require.Contains(t, doc.Channels, "order-placed")
	require.Contains(
		t,
		doc.Components.Schemas,
		"github.com/RubenRibGarcia/asyncgo/test/data/simple.OrderPlaced",
	)
}

func TestBuildNoCatalogs(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("..", "..", "spec"))
	require.NoError(t, err)

	doc, n, err := Build(dir)
	require.Error(t, err)
	assert.Nil(t, doc)
	assert.Zero(t, n)
	assert.Contains(t, err.Error(), "no AsyncAPI catalogs")
}

func TestBuildBadDir(t *testing.T) {
	_, _, err := Build(filepath.Join(t.TempDir(), "does-not-exist"))
	require.Error(t, err)
}
