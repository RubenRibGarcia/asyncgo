package discovery

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RubenRibGarcia/asyncgo/internal/logging"
	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuild(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("..", "..", "test", "data", "simple"))
	require.NoError(t, err)

	doc, n, err := Build(t.Context(), dir, Options{})
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

	doc, n, err := Build(t.Context(), dir, Options{})
	require.Error(t, err)
	assert.Nil(t, doc)
	assert.Zero(t, n)
	assert.Contains(t, err.Error(), "no AsyncAPI catalogs")
}

func TestBuildBadDir(t *testing.T) {
	_, _, err := Build(t.Context(), filepath.Join(t.TempDir(), "does-not-exist"), Options{})
	require.Error(t, err)
}

// TestWarnCatalogsOutsideMain pins the one advisory the widened scope owes the
// user: a catalog nothing imports is still generated, and saying so rules out the
// "my catalog vanished" reading the old reachability filter produced. A module
// with no main package gets no warning — every catalog would trip it, and no
// binary exists to import anything.
func TestWarnCatalogsOutsideMain(t *testing.T) {
	const pkg = "example.com/mod/catalog"
	cat := Catalog{PkgPath: pkg, VarName: "Catalog"}

	tests := []struct {
		name     string
		imported map[string]bool
		hasMain  bool
		want     string
	}{
		{
			name:     "should_warn_when_no_main_package_imports_the_catalog",
			imported: map[string]bool{"example.com/mod/other": true},
			hasMain:  true,
			want:     pkg + ".Catalog is not imported by any main package",
		},
		{
			name:     "should_stay_quiet_when_a_main_package_imports_the_catalog",
			imported: map[string]bool{pkg: true},
			hasMain:  true,
		},
		{
			name:     "should_stay_quiet_when_the_module_has_no_main_package",
			imported: map[string]bool{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			warnCatalogsOutsideMain(
				logging.New(&buf, logging.Quiet),
				[]Catalog{cat},
				tt.imported,
				tt.hasMain,
			)

			if tt.want == "" {
				assert.Empty(t, buf.String())
				return
			}
			assert.Contains(t, buf.String(), tt.want)
		})
	}
}

func TestBuildRejectsInvalidConfig(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, configFileName),
		[]byte("mappings:\n  big.Int:\n    type: integer\n"),
		0o644,
	))

	doc, n, err := Build(t.Context(), dir, Options{})

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
		mappings, err := loadMappingConfig(t.TempDir(), "", logging.New(io.Discard, logging.Quiet))

		require.NoError(t, err)
		assert.Empty(t, mappings)
	})

	t.Run("should_load_mappings_from_the_default_asyncgo_yaml", func(t *testing.T) {
		dir := t.TempDir()
		writeConfig(t, dir, configFileName, valid)

		mappings, err := loadMappingConfig(dir, "", logging.New(io.Discard, logging.Quiet))

		require.NoError(t, err)
		require.Len(t, mappings, 1)
		assert.Equal(t, "math/big.Int", mappings[0].FQN)
		assert.JSONEq(t, `{"type":"integer"}`, mappings[0].Schema)
	})

	t.Run("should_load_mappings_from_an_explicit_path", func(t *testing.T) {
		// The explicit path is used verbatim, so a config outside the module works
		// and the error names the path the user typed rather than a joined one.
		path := writeConfig(t, t.TempDir(), "team.yaml", valid)

		mappings, err := loadMappingConfig(
			t.TempDir(),
			path,
			logging.New(io.Discard, logging.Quiet),
		)

		require.NoError(t, err)
		require.Len(t, mappings, 1)
		assert.Equal(t, "math/big.Int", mappings[0].FQN)
	})

	t.Run("should_use_the_explicit_path_instead_of_the_default", func(t *testing.T) {
		dir := t.TempDir()
		writeConfig(t, dir, configFileName, "mappings:\n  math/big.Int:\n    type: string\n")
		path := writeConfig(t, t.TempDir(), "team.yaml", valid)

		mappings, err := loadMappingConfig(dir, path, logging.New(io.Discard, logging.Quiet))

		require.NoError(t, err)
		// One mapping set shapes a run: the default file is not read at all.
		require.Len(t, mappings, 1)
		assert.JSONEq(t, `{"type":"integer"}`, mappings[0].Schema)
	})

	t.Run("should_reject_an_invalid_default_config", func(t *testing.T) {
		dir := t.TempDir()
		writeConfig(t, dir, configFileName, "mappings:\n  big.Int:\n    type: integer\n")

		mappings, err := loadMappingConfig(dir, "", logging.New(io.Discard, logging.Quiet))

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

		mappings, err := loadMappingConfig(dir, missing, logging.New(io.Discard, logging.Quiet))

		require.Error(t, err)
		assert.Nil(t, mappings)
		assert.True(t, strings.HasPrefix(err.Error(), "reading "+missing), err.Error())
	})

	t.Run("should_report_an_unreadable_default_config", func(t *testing.T) {
		// A directory in the file's place is a read error that exists on every
		// platform, unlike a permission bit.
		dir := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(dir, configFileName), 0o755))

		_, err := loadMappingConfig(dir, "", logging.New(io.Discard, logging.Quiet))

		require.Error(t, err)
		// The default file is reported by its joined path, so the message says which
		// module's config could not be read.
		assert.Contains(t, err.Error(), "reading ")
		assert.Contains(t, err.Error(), configFileName)
	})
}

// TestBuildReportsDebugDetail pins the debug-level story: the per-catalog origin,
// the combinator and alias counts, and the description count. They are the lines
// a maintainer reads when the document is not what they expected.
func TestBuildReportsDebugDetail(t *testing.T) {
	t.Run("should_report_catalogs_refs_and_descriptions", func(t *testing.T) {
		dir, err := filepath.Abs(filepath.Join("..", "..", "test", "data", "simple"))
		require.NoError(t, err)
		var log bytes.Buffer

		_, _, err = Build(
			t.Context(),
			dir,
			Options{Log: logging.New(&log, logging.Debug)},
		)
		require.NoError(t, err)

		got := log.String()
		assert.Contains(
			t,
			got,
			"catalogs: github.com/RubenRibGarcia/asyncgo/test/data/simple.Catalog",
		)
		assert.Contains(t, got, "declared in catalog.go", "the file is shown relative to dir")
		assert.Contains(t, got, "refs: ")
		assert.Contains(t, got, "descriptions: ")
	})

	t.Run("should_stay_quiet_without_a_logger", func(t *testing.T) {
		dir, err := filepath.Abs(filepath.Join("..", "..", "test", "data", "simple"))
		require.NoError(t, err)

		_, _, err = Build(t.Context(), dir, Options{})

		require.NoError(t, err, "a nil logger must not break the build")
	})
}

func TestSchemaCount(t *testing.T) {
	t.Run("should_be_zero_without_components", func(t *testing.T) {
		assert.Zero(t, schemaCount(spec.New()))
	})

	t.Run("should_count_hoisted_schemas", func(t *testing.T) {
		doc := spec.New()
		doc.Components = &spec.Components{Schemas: map[string]*spec.Schema{"a": {}, "b": {}}}

		assert.Equal(t, 2, schemaCount(doc))
	})
}
