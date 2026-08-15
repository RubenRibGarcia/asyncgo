package discovery

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractDescriptionsSimple(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("..", "..", "examples", "simple"))
	require.NoError(t, err)

	pkgs, err := load(dir)
	require.NoError(t, err)
	desc := extractDescriptions(pkgs, reachableFromMain(pkgs))

	const key = "github.com/RubenRibGarcia/asyncgo/examples/simple.OrderPlaced"
	require.Contains(t, desc, key)
	assert.Equal(t, "Optional note from the customer", desc[key]["note"])
}

func TestExtractDescriptionsEmbedded(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("..", "..", "examples", "embedded"))
	require.NoError(t, err)

	pkgs, err := load(dir)
	require.NoError(t, err)
	desc := extractDescriptions(pkgs, reachableFromMain(pkgs))

	const key = "github.com/RubenRibGarcia/asyncgo/examples/embedded.OrderPlaced"
	require.Contains(t, desc, key)
	// Embedded BaseSchema.ID is flattened into OrderPlaced under "id".
	assert.Equal(t, "Unique identifier for the order", desc[key]["id"])
	assert.Equal(t, "Optional note from the customer", desc[key]["note"])
}
