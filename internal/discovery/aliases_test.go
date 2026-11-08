package discovery

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// aliasFixturePkg is the fixture package the alias corpus lives in: it declares a
// derived alias, an alias of an alias, a named pointer, a Go type alias, a type
// derived from a predeclared one, and one derived from an array.
const aliasFixturePkg = "github.com/RubenRibGarcia/asyncgo/test/data/mappings"

func TestCollectAliasEdges(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("..", "..", "test", "data", "mappings"))
	require.NoError(t, err)
	pkgs, err := load(t.Context(), dir)
	require.NoError(t, err)

	edges := collectAliasEdges(pkgs, reachableFromMain(pkgs))

	t.Run("should_emit_alias_edge_for_defined_type_declaration", func(t *testing.T) {
		assert.Contains(t, edges, aliasEdge{
			Alias: aliasFixturePkg + ".OrderID",
			Base:  aliasFixturePkg + ".UUID",
		})
	})

	t.Run("should_emit_alias_edge_for_pointer_base_declaration", func(t *testing.T) {
		assert.Contains(t, edges, aliasEdge{
			Alias: aliasFixturePkg + ".OrderRef",
			Base:  aliasFixturePkg + ".UUID",
		})
	})

	t.Run("should_emit_alias_edge_for_alias_of_alias", func(t *testing.T) {
		// One edge per declaration, so an intermediate alias can carry its own
		// mapping and the types derived from it inherit that rather than the base's.
		assert.Contains(t, edges, aliasEdge{
			Alias: aliasFixturePkg + ".LineID",
			Base:  aliasFixturePkg + ".OrderID",
		})
	})

	t.Run("should_not_emit_edge_for_go_type_alias", func(t *testing.T) {
		// `type UUIDAlias = UUID` introduces no new type: reflection already sees
		// UUID, so an edge would be dead weight.
		for _, edge := range edges {
			assert.NotEqual(t, aliasFixturePkg+".UUIDAlias", edge.Alias)
		}
	})

	t.Run("should_not_emit_edge_for_predeclared_base", func(t *testing.T) {
		// `type Quantity int` has no mapping to inherit, and `type UUID [16]byte`
		// is derived from an unnamed type rather than a defined one.
		for _, edge := range edges {
			assert.NotEqual(t, aliasFixturePkg+".Quantity", edge.Alias)
			assert.NotEqual(t, aliasFixturePkg+".UUID", edge.Alias)
		}
	})

	t.Run("should_dedupe_and_sort_alias_edges", func(t *testing.T) {
		// A duplicate declaration does not compile, so uniqueness is a property of
		// the output rather than a state a test can construct; sorting is what makes
		// the emitted harness deterministic.
		assert.Equal(t, []aliasEdge{
			{Alias: aliasFixturePkg + ".LineID", Base: aliasFixturePkg + ".OrderID"},
			{Alias: aliasFixturePkg + ".OrderID", Base: aliasFixturePkg + ".UUID"},
			{Alias: aliasFixturePkg + ".OrderRef", Base: aliasFixturePkg + ".UUID"},
		}, edges)
	})
}
