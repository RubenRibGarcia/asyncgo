package discovery

import (
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// copyFixtureModule copies a test/data fixture into a fresh directory and rewrites
// its replace directive to the absolute repo root. The copy sits outside the
// workspace, so `go run` inside it resolves the library the way a user's module
// does — the same preparation the CLI tests use, kept local so a discovery test
// can edit the fixture's main package without touching the committed one.
func copyFixtureModule(t *testing.T, name string) string {
	t.Helper()

	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	dst := t.TempDir()
	require.NoError(t, os.CopyFS(dst, os.DirFS(filepath.Join(repoRoot, "test", "data", name))))

	goMod, err := os.ReadFile(filepath.Join(dst, "go.mod"))
	require.NoError(t, err)
	replaced := strings.ReplaceAll(string(goMod), "=> ../../../", "=> "+repoRoot)
	require.NoError(t, os.WriteFile(filepath.Join(dst, "go.mod"), []byte(replaced), 0o644))

	return dst
}

// TestBuildIgnoresMainImportGraph pins the discovery scope rule: a document
// describes the module, not one of its binaries, so a catalog reaches the
// document whether or not a main package imports it.
//
// The simple fixture's cmd/ imports the catalog, which is the imported case. The
// second half replaces that main with one importing nothing, which is the case
// that used to fail with "no AsyncAPI catalogs (*asyncgo.SpecResult vars)
// reachable from main" — and the two documents must be identical, since a main
// package has no say in what the module's topology is.
func TestBuildIgnoresMainImportGraph(t *testing.T) {
	t.Setenv("GOWORK", "off")

	withMain := copyFixtureModule(t, "simple")
	want, wantN, err := Build(t.Context(), withMain, Options{})
	require.NoError(t, err)
	require.Equal(t, 1, wantN)

	// A binary that imports nothing: under reachability it would hide the catalog.
	withoutImport := copyFixtureModule(t, "simple")
	require.NoError(t, os.WriteFile(
		filepath.Join(withoutImport, "cmd", "main.go"),
		[]byte("package main\n\nfunc main() {}\n"),
		0o644,
	))

	got, gotN, err := Build(t.Context(), withoutImport, Options{})
	require.NoError(t, err)
	assert.Equal(t, 1, gotN, "a main package that imports nothing must not hide a catalog")

	wantYAML, err := want.YAML()
	require.NoError(t, err)
	gotYAML, err := got.YAML()
	require.NoError(t, err)
	assert.Equal(t, string(wantYAML), string(gotYAML),
		"the module's document must not depend on what a main package imports")
}

func TestFindMaterializeMerge(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("..", "..", "test", "data", "simple"))
	require.NoError(t, err)

	cats, err := Find(t.Context(), dir)
	require.NoError(t, err)
	require.Len(t, cats, 1)
	assert.Equal(t, "github.com/RubenRibGarcia/asyncgo/test/data/simple", cats[0].PkgPath)
	assert.Equal(t, "Catalog", cats[0].VarName)

	docs, err := Materialize(t.Context(), dir, Plan{Catalogs: cats}, Options{})
	require.NoError(t, err)
	require.Len(t, docs, 1)

	merged, collisions := Merge(pairCatalogs(cats, docs)...)
	assert.Empty(t, collisions, "one catalog cannot collide with itself")
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
	_, err := load(t.Context(), filepath.Join(t.TempDir(), "does-not-exist"))
	require.Error(t, err)
}

func TestFindErrors(t *testing.T) {
	_, err := Find(t.Context(), filepath.Join(t.TempDir(), "does-not-exist"))
	require.Error(t, err)
}

func TestIsSpecResult(t *testing.T) {
	t.Run("should_report_true_for_pointer_to_spec_result", func(t *testing.T) {
		dir, err := filepath.Abs(filepath.Join("..", "..", "test", "data", "simple"))
		require.NoError(t, err)
		pkgs, err := load(t.Context(), dir)
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
