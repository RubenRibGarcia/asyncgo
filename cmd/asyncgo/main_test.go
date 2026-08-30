package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveVersion(t *testing.T) {
	t.Run("should_return_injected_version", func(t *testing.T) {
		old := version
		defer func() { version = old }()
		version = "v1.2.3"
		assert.Equal(t, "v1.2.3", resolveVersion())
	})

	t.Run("should_fall_back_to_devel_when_untagged", func(t *testing.T) {
		old := version
		defer func() { version = old }()
		version = "devel"
		// A `go test` binary carries "(devel)" as its main module version.
		assert.Equal(t, "devel", resolveVersion())
	})
}

func TestRunVersion(t *testing.T) {
	old := version
	defer func() { version = old }()
	version = "v1.2.3"

	out := captureStdout(t, func() {
		require.NoError(t, run([]string{"version"}))
	})
	assert.Equal(t, "asyncgo v1.2.3\n", out)
}

func TestRunVersionWithCommit(t *testing.T) {
	oldVersion, oldCommit, oldDate := version, commit, date
	defer func() { version, commit, date = oldVersion, oldCommit, oldDate }()
	version, commit, date = "v1.2.3", "abc1234", "2026-08-18T00:00:00Z"

	out := captureStdout(t, func() {
		require.NoError(t, run([]string{"version"}))
	})
	assert.Equal(t, "asyncgo v1.2.3 (commit abc1234, built 2026-08-18T00:00:00Z)\n", out)
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	defer func() { os.Stdout = old }()

	fn()

	require.NoError(t, w.Close())
	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)
	return buf.String()
}

func TestRunUsageError(t *testing.T) {
	err := run(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "usage:")
}

func TestRunUnknownCommand(t *testing.T) {
	err := run([]string{"bogus"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")
}

func TestResolveDir(t *testing.T) {
	t.Run("should_default_to_current_directory", func(t *testing.T) {
		got, err := resolveDir(nil)
		require.NoError(t, err)
		want, err := filepath.Abs(".")
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("should_absolutize_explicit_path", func(t *testing.T) {
		got, err := resolveDir([]string{"sub"})
		require.NoError(t, err)
		want, err := filepath.Abs("sub")
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})
}

func TestRunGenerateAndCheck(t *testing.T) {
	t.Setenv("GOWORK", "off")
	dir := copySimpleFixture(t)

	out := captureStdout(t, func() {
		require.NoError(t, run([]string{"generate", dir}))
	})
	assert.Contains(t, out, "wrote ")
	assert.Contains(t, out, "(1 catalog(s))")

	out = captureStdout(t, func() {
		require.NoError(t, run([]string{"check", dir}))
	})
	assert.Contains(t, out, "is up to date")
}

func TestGenerateNoCatalogs(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("..", "..", "spec"))
	require.NoError(t, err)

	err = generate([]string{dir})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no AsyncAPI catalogs")
}

func TestCheckOutOfDate(t *testing.T) {
	t.Setenv("GOWORK", "off")
	dir := copySimpleFixture(t)

	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "asyncapi.yaml"),
		[]byte("asyncapi: 0.0.0\n"),
		0o644,
	))

	err := check([]string{dir})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of date")
}

func TestCheckMissing(t *testing.T) {
	t.Setenv("GOWORK", "off")
	dir := copySimpleFixture(t)

	err := check([]string{dir})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestCheckNoCatalogs(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("..", "..", "spec"))
	require.NoError(t, err)

	err = check([]string{dir})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no AsyncAPI catalogs")
}

func TestGenerateWriteError(t *testing.T) {
	t.Setenv("GOWORK", "off")
	dir := copySimpleFixture(t)
	// A directory in place of asyncapi.yaml forces os.WriteFile to fail.
	require.NoError(t, os.Mkdir(filepath.Join(dir, "asyncapi.yaml"), 0o755))

	err := generate([]string{dir})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "writing ")
}

func TestCheckReadError(t *testing.T) {
	t.Setenv("GOWORK", "off")
	dir := copySimpleFixture(t)
	// A directory in place of asyncapi.yaml makes os.ReadFile fail with a
	// non-IsNotExist error.
	require.NoError(t, os.Mkdir(filepath.Join(dir, "asyncapi.yaml"), 0o755))

	err := check([]string{dir})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading ")
}

// copySimpleFixture copies the simple test fixture into a fresh temp directory,
// rewrites its replace directive to the absolute repo root, and removes the
// committed asyncapi.yaml so generate/check operate on a clean slate without
// racing the integration golden test over the shared fixture files.
func copySimpleFixture(t *testing.T) string {
	t.Helper()

	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	src := filepath.Join(repoRoot, "test", "data", "simple")

	dst := t.TempDir()
	require.NoError(t, os.CopyFS(dst, os.DirFS(src)))

	goMod, err := os.ReadFile(filepath.Join(dst, "go.mod"))
	require.NoError(t, err)
	replaced := strings.ReplaceAll(string(goMod), "=> ../../../", "=> "+repoRoot)
	require.NoError(t, os.WriteFile(filepath.Join(dst, "go.mod"), []byte(replaced), 0o644))

	require.NoError(t, os.Remove(filepath.Join(dst, "asyncapi.yaml")))

	return dst
}
