package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGoldenExample regenerates the example's document and asserts it matches
// the committed asyncapi.yaml. This locks the artifact against drift; when the
// generator output changes, regenerate with:
//
//	go run ./cmd/asyncgo generate ./examples/orders
func TestGoldenExample(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("..", "..", "examples", "orders"))
	if err != nil {
		t.Fatal(err)
	}

	cats, err := Find(dir)
	if err != nil {
		t.Fatal(err)
	}
	docs, err := Materialize(dir, cats)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := ApplyFragment(dir, Merge(docs...))
	if err != nil {
		t.Fatal(err)
	}
	got, err := doc.YAML()
	if err != nil {
		t.Fatal(err)
	}

	want, err := os.ReadFile(filepath.Join(dir, "asyncapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Errorf("generated document differs from committed example; regenerate with `go run ./cmd/asyncgo generate ./examples/orders`\n--- generated ---\n%s\n--- committed ---\n%s", got, want)
	}
}
