package discovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGoldenExample regenerates the example's document and asserts it matches
// the committed asyncapi.yaml. This locks the artifact against drift; when the
// generator output changes, regenerate with:
//
//	go run ./cmd/asyncgo generate ./examples/orders
func TestGoldenExample(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("..", "..", "examples", "orders"))
	require.NoError(t, err)

	cats, err := Find(dir)
	require.NoError(t, err)
	docs, err := Materialize(dir, cats)
	require.NoError(t, err)
	doc := Merge(docs...)
	got, err := doc.YAML()
	require.NoError(t, err)

	want, err := os.ReadFile(filepath.Join(dir, "asyncapi.yaml"))
	require.NoError(t, err)
	assert.Equal(
		t,
		string(want),
		string(got),
		"generated document differs from committed example; regenerate with `go run ./cmd/asyncgo generate ./examples/orders`",
	)
}
