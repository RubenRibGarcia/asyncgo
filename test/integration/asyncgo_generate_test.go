package integration

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/RubenRibGarcia/asyncgo/internal/discovery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	// asyncapiCLIImage pins the AsyncAPI CLI that validates the generated
	// documents. The tag is deliberate: tracking `latest` would let a new CLI
	// release break this test without any change in this repository.
	asyncapiCLIImage = "asyncapi/cli:6.1.0"

	// asyncapiCLIDocumentPath is where the generated document is placed inside
	// the validator container. The image's WORKDIR is /app.
	asyncapiCLIDocumentPath = "/app/asyncapi.generated.yaml"
)

// fixtures are the discovery fixtures under test/data, each carrying a
// committed asyncapi.yaml.
var fixtures = []string{"simple", "allof", "oneof", "anyof", "provider"}

// TestAsyncGoGenerate regenerates each test fixture's document, asserts it
// matches the committed asyncapi.yaml, and validates it with the real
// `asyncapi validate` command from the AsyncAPI CLI running in a container.
//
// The golden comparison catches drift against the committed artifacts; the CLI
// validation catches a document that is internally consistent but not valid
// AsyncAPI — a failure mode the golden comparison cannot see, since a document
// regenerated wrongly is compared against the wrongly regenerated artifact.
// When the generator output changes, regenerate with:
//
//	go run ./cmd/asyncgo generate ./test/data/<name>
func TestAsyncGoGenerate(t *testing.T) {
	for _, name := range fixtures {
		t.Run(name, func(t *testing.T) {
			dir, err := filepath.Abs(filepath.Join("..", "data", name))
			require.NoError(t, err)

			doc, _, err := discovery.Build(dir)
			require.NoError(t, err)

			got, err := doc.YAML()
			require.NoError(t, err)

			want, err := os.ReadFile(filepath.Join(dir, "asyncapi.yaml"))
			require.NoError(t, err)

			assert.Equal(
				t,
				string(want),
				string(got),
				"generated document differs from committed fixture; regenerate with `go run ./cmd/asyncgo generate ./test/data/%s`",
				name,
			)

			validateDocument(t, got)
		})
	}
}

// validateDocument validates a freshly generated document — not the committed
// artifact — by handing it to `asyncapi validate` in a container.
//
// wait.ForExit returns as soon as the container stops, whatever its exit status:
// it never inspects the exit code, so the CLI's status is asserted here
// explicitly, with the CLI's own diagnostics attached to the failure.
func validateDocument(t *testing.T, doc []byte) {
	t.Helper()

	ctx := t.Context()

	// t.TempDir keeps the document out of the fixture tree: nothing under
	// test/data is written, so a failing assertion cannot leave debris behind.
	hostPath := filepath.Join(t.TempDir(), "asyncapi.generated.yaml")
	require.NoError(t, os.WriteFile(hostPath, doc, 0o644))

	container, err := testcontainers.Run(ctx, asyncapiCLIImage,
		testcontainers.WithFiles(testcontainers.ContainerFile{
			HostFilePath:      hostPath,
			ContainerFilePath: asyncapiCLIDocumentPath,
			FileMode:          0o644,
		}),
		testcontainers.WithCmdArgs("validate", asyncapiCLIDocumentPath),
		testcontainers.WithWaitStrategy(wait.ForExit()),
	)
	require.NoError(t, err)
	testcontainers.CleanupContainer(t, container)

	state, err := container.State(ctx)
	require.NoError(t, err)

	logs, err := container.Logs(ctx)
	require.NoError(t, err)
	defer func() { _ = logs.Close() }()

	diagnostics, err := io.ReadAll(logs)
	require.NoError(t, err)

	assert.Zero(
		t,
		state.ExitCode,
		"asyncapi validate rejected the generated document (exit code %d):\n%s",
		state.ExitCode,
		diagnostics,
	)
}
