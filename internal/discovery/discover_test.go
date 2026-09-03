package discovery

import (
	"go/types"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindMaterializeMerge(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("..", "..", "test", "data", "simple"))
	require.NoError(t, err)

	cats, err := Find(dir)
	require.NoError(t, err)
	require.Len(t, cats, 1)
	assert.Equal(t, "github.com/RubenRibGarcia/asyncgo/test/data/simple", cats[0].PkgPath)
	assert.Equal(t, "Catalog", cats[0].VarName)

	docs, err := Materialize(dir, cats, nil)
	require.NoError(t, err)
	require.Len(t, docs, 1)

	merged := Merge(docs...)
	assert.Equal(t, "Orders Service", merged.Info.Title)

	require.Contains(t, merged.Channels, "order-placed")
	ch := merged.Channels["order-placed"]
	require.Contains(t, ch.Messages, "OrderPlaced")

	// Schema is hoisted under the fully-qualified type name.
	const key = "github.com/RubenRibGarcia/asyncgo/test/data/simple.OrderPlaced"
	require.Contains(t, merged.Components.Schemas, key)
	assert.Len(t, merged.Components.Schemas[key].Required, 2)
}

func TestLoadErrors(t *testing.T) {
	_, err := load(filepath.Join(t.TempDir(), "does-not-exist"))
	require.Error(t, err)
}

func TestFindErrors(t *testing.T) {
	_, err := Find(filepath.Join(t.TempDir(), "does-not-exist"))
	require.Error(t, err)
}

func TestIsSpecResult(t *testing.T) {
	t.Run("should_report_true_for_pointer_to_spec_result", func(t *testing.T) {
		dir, err := filepath.Abs(filepath.Join("..", "..", "test", "data", "simple"))
		require.NoError(t, err)
		pkgs, err := load(dir)
		require.NoError(t, err)

		var catType types.Type
		for _, p := range pkgs {
			if obj := p.Types.Scope().Lookup("Catalog"); obj != nil {
				catType = obj.Type()
				break
			}
		}
		require.NotNil(t, catType)
		assert.True(t, isSpecResult(catType))
	})

	t.Run("should_report_false_for_non_pointer", func(t *testing.T) {
		assert.False(t, isSpecResult(types.Typ[types.String]))
	})

	t.Run("should_report_false_for_pointer_to_non_named", func(t *testing.T) {
		assert.False(t, isSpecResult(types.NewPointer(types.Typ[types.String])))
	})
}
