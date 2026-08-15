package discovery

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindMaterializeMerge(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("..", "..", "examples", "simple"))
	require.NoError(t, err)

	cats, err := Find(dir)
	require.NoError(t, err)
	require.Len(t, cats, 1)
	assert.Equal(t, "github.com/RubenRibGarcia/asyncgo/examples/simple", cats[0].PkgPath)
	assert.Equal(t, "Catalog", cats[0].VarName)

	docs, err := Materialize(dir, cats)
	require.NoError(t, err)
	require.Len(t, docs, 1)

	merged := Merge(docs...)
	assert.Equal(t, "Orders Service", merged.Info.Title)

	require.Contains(t, merged.Channels, "order-placed")
	ch := merged.Channels["order-placed"]
	require.Contains(t, ch.Messages, "OrderPlaced")

	// Schema is hoisted under the fully-qualified type name.
	const key = "github.com/RubenRibGarcia/asyncgo/examples/simple.OrderPlaced"
	require.Contains(t, merged.Components.Schemas, key)
	assert.Len(t, merged.Components.Schemas[key].Required, 2)
}
