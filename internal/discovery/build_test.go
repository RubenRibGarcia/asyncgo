package discovery

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuild(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("..", "..", "test", "data", "simple"))
	require.NoError(t, err)

	doc, n, err := Build(t.Context(), dir, "")
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	assert.Equal(t, "Orders Service", doc.Info.Title)
	require.Contains(t, doc.Channels, "order-placed")
	require.Contains(
		t,
		doc.Components.Schemas,
		"github.com/RubenRibGarcia/asyncgo/test/data/simple.OrderPlaced",
	)
}

func TestBuildNoCatalogs(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("..", "..", "spec"))
	require.NoError(t, err)

	doc, n, err := Build(t.Context(), dir, "")
	require.Error(t, err)
	assert.Nil(t, doc)
	assert.Zero(t, n)
	assert.Contains(t, err.Error(), "no AsyncAPI catalogs")
}

func TestBuildBadDir(t *testing.T) {
	_, _, err := Build(t.Context(), filepath.Join(t.TempDir(), "does-not-exist"), "")
	require.Error(t, err)
}

func TestBuildRejectsInvalidConfig(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, configFileName),
		[]byte("mappings:\n  big.Int:\n    type: integer\n"),
		0o644,
	))

	doc, n, err := Build(t.Context(), dir, "")

	require.Error(t, err)
	assert.Nil(t, doc)
	assert.Zero(t, n)
	assert.Contains(t, err.Error(), configFileName)
	assert.Contains(t, err.Error(), "not a fully-qualified type name")
}

// writeConfig writes a mapping config file in dir and returns its path.
func writeConfig(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
	return path
}

// TestLoadMappingConfig pins the rule Build applies to the config file: where it
// looks, which absences are allowed, and that an explicit path replaces the
// default rather than being merged with it.
func TestLoadMappingConfig(t *testing.T) {
	const valid = "mappings:\n  math/big.Int:\n    type: integer\n"

	t.Run("should_return_no_mappings_without_a_config_file", func(t *testing.T) {
		mappings, err := loadMappingConfig(t.TempDir(), "")

		require.NoError(t, err)
		assert.Empty(t, mappings)
	})

	t.Run("should_load_mappings_from_the_default_asyncgo_yaml", func(t *testing.T) {
		dir := t.TempDir()
		writeConfig(t, dir, configFileName, valid)

		mappings, err := loadMappingConfig(dir, "")

		require.NoError(t, err)
		require.Len(t, mappings, 1)
		assert.Equal(t, "math/big.Int", mappings[0].FQN)
		assert.JSONEq(t, `{"type":"integer"}`, mappings[0].Schema)
	})

	t.Run("should_load_mappings_from_an_explicit_path", func(t *testing.T) {
		// The explicit path is used verbatim, so a config outside the module works
		// and the error names the path the user typed rather than a joined one.
		path := writeConfig(t, t.TempDir(), "team.yaml", valid)

		mappings, err := loadMappingConfig(t.TempDir(), path)

		require.NoError(t, err)
		require.Len(t, mappings, 1)
		assert.Equal(t, "math/big.Int", mappings[0].FQN)
	})

	t.Run("should_use_the_explicit_path_instead_of_the_default", func(t *testing.T) {
		dir := t.TempDir()
		writeConfig(t, dir, configFileName, "mappings:\n  math/big.Int:\n    type: string\n")
		path := writeConfig(t, t.TempDir(), "team.yaml", valid)

		mappings, err := loadMappingConfig(dir, path)

		require.NoError(t, err)
		// One mapping set shapes a run: the default file is not read at all.
		require.Len(t, mappings, 1)
		assert.JSONEq(t, `{"type":"integer"}`, mappings[0].Schema)
	})

	t.Run("should_reject_an_invalid_default_config", func(t *testing.T) {
		dir := t.TempDir()
		writeConfig(t, dir, configFileName, "mappings:\n  big.Int:\n    type: integer\n")

		mappings, err := loadMappingConfig(dir, "")

		require.Error(t, err)
		assert.Nil(t, mappings)
		// The message names the file, so a project with both a module config and a
		// fixture config knows which one failed.
		assert.Contains(t, err.Error(), configFileName)
		assert.Contains(t, err.Error(), "not a fully-qualified type name")
	})

	t.Run("should_reject_a_missing_explicit_config", func(t *testing.T) {
		// A named path must exist: a typo would otherwise silently generate a
		// document with no mappings.
		dir := t.TempDir()
		writeConfig(t, dir, configFileName, valid)
		missing := filepath.Join(dir, "typo.yaml")

		mappings, err := loadMappingConfig(dir, missing)

		require.Error(t, err)
		assert.Nil(t, mappings)
		assert.True(t, strings.HasPrefix(err.Error(), "reading "+missing), err.Error())
	})

	t.Run("should_report_an_unreadable_default_config", func(t *testing.T) {
		// A directory in the file's place is a read error that exists on every
		// platform, unlike a permission bit.
		dir := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(dir, configFileName), 0o755))

		_, err := loadMappingConfig(dir, "")

		require.Error(t, err)
		// The default file is reported by its joined path, so the message says which
		// module's config could not be read.
		assert.Contains(t, err.Error(), "reading ")
		assert.Contains(t, err.Error(), configFileName)
	})
}
