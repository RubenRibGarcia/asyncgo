package discovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGoldenExample regenerates each example's document and asserts it matches
// the committed asyncapi.yaml. This locks the artifacts against drift; when the
// generator output changes, regenerate with:
//
//	go run ./cmd/asyncgo generate ./examples/<name>
func TestGoldenExample(t *testing.T) {
	for _, name := range []string{"simple", "allof", "oneof", "anyof"} {
		t.Run(name, func(t *testing.T) {
			dir, err := filepath.Abs(filepath.Join("..", "..", "examples", name))
			require.NoError(t, err)

			doc, _, err := Build(dir)
			require.NoError(t, err)
			got, err := doc.YAML()
			require.NoError(t, err)

			want, err := os.ReadFile(filepath.Join(dir, "asyncapi.yaml"))
			require.NoError(t, err)
			assert.Equal(
				t,
				string(want),
				string(got),
				"generated document differs from committed example; regenerate with `go run ./cmd/asyncgo generate ./examples/%s`",
				name,
			)
		})
	}
}
