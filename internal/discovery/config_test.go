package discovery

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadMappingConfig(t *testing.T) {
	t.Run("should_return_no_mappings_without_a_config_file", func(t *testing.T) {
		mappings, err := loadMappingConfig(t.TempDir())

		require.NoError(t, err)
		assert.Empty(t, mappings)
	})

	t.Run("should_load_mappings_from_asyncgo_yaml", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(
			filepath.Join(dir, configFileName),
			[]byte("mappings:\n  math/big.Int:\n    type: integer\n"),
			0o644,
		))

		mappings, err := loadMappingConfig(dir)

		require.NoError(t, err)
		require.Len(t, mappings, 1)
		assert.Equal(t, "math/big.Int", mappings[0].FQN)
		assert.JSONEq(t, `{"type":"integer"}`, mappings[0].Schema)
	})

	t.Run("should_reject_an_invalid_config", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(
			filepath.Join(dir, configFileName),
			[]byte("mappings:\n  big.Int:\n    type: integer\n"),
			0o644,
		))

		mappings, err := loadMappingConfig(dir)

		require.Error(t, err)
		assert.Nil(t, mappings)
		// The message names the file, so a project with two config files (a module
		// root and a fixture) knows which one failed.
		assert.True(t, strings.HasPrefix(err.Error(), configFileName+": "), err.Error())
		assert.Contains(t, err.Error(), "not a fully-qualified type name")
	})

	t.Run("should_report_an_unreadable_config", func(t *testing.T) {
		// A directory in the file's place is a read error that exists on every
		// platform, unlike a permission bit.
		dir := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(dir, configFileName), 0o755))

		_, err := loadMappingConfig(dir)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "reading "+configFileName)
	})
}
