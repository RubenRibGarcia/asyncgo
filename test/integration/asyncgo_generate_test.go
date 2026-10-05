package integration

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/RubenRibGarcia/asyncgo/internal/discovery"
	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/goccy/go-yaml"
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

	// asyncapiCLIDirectory is where a generated document is placed inside the
	// validator container. The image's WORKDIR is /app.
	asyncapiCLIDirectory = "/app"

	// The document names double as the container filename, and the AsyncAPI CLI
	// picks its parser from the extension: a JSON document has to be named .json
	// for `asyncapi validate` to read it as JSON rather than as YAML.
	yamlDocumentName = "asyncapi.generated.yaml"
	jsonDocumentName = "asyncapi.generated.json"
)

// fixtures are the discovery fixtures under test/data, each carrying a
// committed asyncapi.yaml.
var fixtures = []string{
	"simple",
	"allof",
	"oneof",
	"anyof",
	"provider",
	"security",
	"reply",
	"traits",
	"avro",
}

// TestAsyncGoGenerate regenerates each test fixture's document, asserts it
// matches the committed asyncapi.yaml, asserts the JSON encoding denotes the
// same document, and validates both encodings with the real `asyncapi validate`
// command from the AsyncAPI CLI running in a container.
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

			validateDocument(t, got, yamlDocumentName)

			// The JSON artifact must denote the same document as the committed
			// YAML: the codecs are independent implementations over a model whose
			// bindings and examples are any-valued. Both sides are canonicalized
			// through encoding/json first, because a direct comparison of the
			// decoded documents is a false negative — decoding YAML yields
			// uint64(3) where decoding JSON yields float64(3) for the same
			// any-valued binding number.
			gotJSON, err := doc.JSON()
			require.NoError(t, err)

			var fromYAML spec.AsyncAPI
			require.NoError(t, yaml.Unmarshal(want, &fromYAML))

			canonicalYAML, err := json.Marshal(&fromYAML)
			require.NoError(t, err)

			assert.JSONEq(
				t,
				string(canonicalYAML),
				string(gotJSON),
				"generated JSON encodes a different document than the committed asyncapi.yaml",
			)

			validateDocument(t, gotJSON, jsonDocumentName)
		})
	}
}

// validateDocument validates a freshly generated document — not the committed
// artifact — by handing it to `asyncapi validate` in a container. The document
// name selects the parser, since the CLI reads the format from the extension.
//
// wait.ForExit returns as soon as the container stops, whatever its exit status:
// it never inspects the exit code, so the CLI's status is asserted here
// explicitly, with the CLI's own diagnostics attached to the failure.
func validateDocument(t *testing.T, doc []byte, documentName string) {
	t.Helper()

	ctx := t.Context()
	containerPath := asyncapiCLIDirectory + "/" + documentName

	// t.TempDir keeps the document out of the fixture tree: nothing under
	// test/data is written, so a failing assertion cannot leave debris behind.
	hostPath := filepath.Join(t.TempDir(), documentName)
	require.NoError(t, os.WriteFile(hostPath, doc, 0o644))

	container, err := testcontainers.Run(ctx, asyncapiCLIImage,
		testcontainers.WithFiles(testcontainers.ContainerFile{
			HostFilePath:      hostPath,
			ContainerFilePath: containerPath,
			FileMode:          0o644,
		}),
		testcontainers.WithCmdArgs("validate", containerPath),
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
		"asyncapi validate rejected the generated %s (exit code %d):\n%s",
		documentName,
		state.ExitCode,
		diagnostics,
	)
}
