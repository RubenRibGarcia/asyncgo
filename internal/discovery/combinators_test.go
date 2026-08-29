package discovery

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveCombinatorName(t *testing.T) {
	t.Run("should_resolve_short_name_against_declaring_package", func(t *testing.T) {
		assert.Equal(
			t,
			combinatorRef{ImportPath: "example.com/orders", TypeName: "OrderPlaced"},
			resolveCombinatorName("OrderPlaced", "example.com/orders"),
		)
	})

	t.Run("should_pass_through_fully_qualified_name", func(t *testing.T) {
		assert.Equal(
			t,
			combinatorRef{ImportPath: "example.com/orders", TypeName: "OrderPlaced"},
			resolveCombinatorName("example.com/orders.OrderPlaced", "unused"),
		)
	})
}

func TestCollectCombinatorRefs(t *testing.T) {
	t.Run("should_collect_union_members_from_example", func(t *testing.T) {
		dir, err := filepath.Abs(filepath.Join("..", "..", "test", "data", "oneof"))
		require.NoError(t, err)

		pkgs, err := load(dir)
		require.NoError(t, err)

		refs := collectCombinatorRefs(pkgs, reachableFromMain(pkgs))

		const pkg = "github.com/RubenRibGarcia/asyncgo/test/data/oneof"
		require.Contains(t, refs, combinatorRef{ImportPath: pkg, TypeName: "OrderPlaced"})
		require.Contains(t, refs, combinatorRef{ImportPath: pkg, TypeName: "OrderCancelled"})
	})
}
