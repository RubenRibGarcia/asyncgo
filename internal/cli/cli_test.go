package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// execute builds a fresh command tree, runs it with args, and captures stdout
// and stderr separately.
func execute(t *testing.T, bi BuildInfo, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	root := newRootCmd(bi)
	var out, errOut bytes.Buffer
	root.SetArgs(args)
	root.SetOut(&out)
	root.SetErr(&errOut)
	err = root.Execute()
	return out.String(), errOut.String(), err
}

func TestResolveVersion(t *testing.T) {
	t.Run("should_return_injected_version", func(t *testing.T) {
		assert.Equal(t, "v1.2.3", resolveVersion("v1.2.3"))
	})

	t.Run("should_fall_back_to_devel_when_untagged", func(t *testing.T) {
		// A `go test` binary carries "(devel)" as its main module version.
		assert.Equal(t, "devel", resolveVersion("devel"))
	})

	t.Run("should_fall_back_to_devel_when_empty", func(t *testing.T) {
		assert.Equal(t, "devel", resolveVersion(""))
	})
}

func TestFormatVersion(t *testing.T) {
	t.Run("should_return_bare_version", func(t *testing.T) {
		assert.Equal(t, "v1.2.3", formatVersion(BuildInfo{Version: "v1.2.3"}))
	})

	t.Run("should_append_commit_and_date", func(t *testing.T) {
		got := formatVersion(BuildInfo{
			Version: "v1.2.3",
			Commit:  "abc1234",
			Date:    "2026-08-18T00:00:00Z",
		})
		assert.Equal(t, "v1.2.3 (commit abc1234, built 2026-08-18T00:00:00Z)", got)
	})
}

func TestExecute(t *testing.T) {
	out := captureStdout(t, func() {
		require.NoError(t, Execute([]string{"version"}, BuildInfo{Version: "v1.2.3"}))
	})
	assert.Equal(t, "asyncgo v1.2.3\n", out)
}

func TestVersionCommand(t *testing.T) {
	out, _, err := execute(t, BuildInfo{Version: "v1.2.3"}, "version")
	require.NoError(t, err)
	assert.Equal(t, "asyncgo v1.2.3\n", out)
}

func TestVersionCommandWithCommit(t *testing.T) {
	out, _, err := execute(t, BuildInfo{
		Version: "v1.2.3",
		Commit:  "abc1234",
		Date:    "2026-08-18T00:00:00Z",
	}, "version")
	require.NoError(t, err)
	assert.Equal(t, "asyncgo v1.2.3 (commit abc1234, built 2026-08-18T00:00:00Z)\n", out)
}

func TestVersionFlag(t *testing.T) {
	out, _, err := execute(t, BuildInfo{Version: "v1.2.3"}, "--version")
	require.NoError(t, err)
	assert.Equal(t, "asyncgo v1.2.3\n", out)
}

func TestNoArgsShowsHelp(t *testing.T) {
	stdout, _, err := execute(t, BuildInfo{})
	require.NoError(t, err)
	assert.Contains(t, stdout, "Usage:")
	assert.Contains(t, stdout, "generate")
	assert.Contains(t, stdout, "check")
}

func TestUnknownCommand(t *testing.T) {
	_, _, err := execute(t, BuildInfo{}, "bogus")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")
}

func TestBadFlag(t *testing.T) {
	_, _, err := execute(t, BuildInfo{}, "generate", "-x")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown shorthand flag")
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

func TestResolveOutput(t *testing.T) {
	dir := filepath.Join("some", "dir")

	tests := []struct {
		name        string
		output      string
		defaultName string
		want        string
	}{
		{
			name:        "should_default_to_dir_asyncapi_yaml",
			defaultName: "asyncapi.yaml",
			want:        filepath.Join(dir, "asyncapi.yaml"),
		},
		{
			name:        "should_default_to_dir_asyncapi_json",
			defaultName: "asyncapi.json",
			want:        filepath.Join(dir, "asyncapi.json"),
		},
		{
			name:        "should_join_default_filename_for_trailing_separator",
			output:      "out/",
			defaultName: "asyncapi.yaml",
			want:        filepath.Join("out", "asyncapi.yaml"),
		},
		{
			name:        "should_join_json_default_filename_for_trailing_separator",
			output:      "out/",
			defaultName: "asyncapi.json",
			want:        filepath.Join("out", "asyncapi.json"),
		},
		{
			name:        "should_absolutize_explicit_file_whatever_its_extension",
			output:      "custom.yaml",
			defaultName: "asyncapi.json",
			want:        absPath(t, "custom.yaml"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveOutput(dir, tc.output, tc.defaultName)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestResolveFormat(t *testing.T) {
	tests := []struct {
		name   string
		format string
		want   string
		errMsg string
	}{
		{name: "should_accept_yaml", format: "yaml", want: "asyncapi.yaml"},
		{name: "should_accept_json", format: "json", want: "asyncapi.json"},
		{
			name:   "should_reject_an_unknown_format",
			format: "xml",
			errMsg: `unknown format "xml": want "yaml" or "json"`,
		},
		{
			// `yml` is a common alias for YAML but is deliberately not accepted.
			name:   "should_reject_the_yml_alias",
			format: "yml",
			errMsg: `unknown format "yml": want "yaml" or "json"`,
		},
		{
			name:   "should_reject_an_empty_format",
			format: "",
			errMsg: `unknown format "": want "yaml" or "json"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveFormat(tc.format)
			if tc.errMsg != "" {
				require.EqualError(t, err, tc.errMsg)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, got.defaultName)
			assert.NotNil(t, got.encode)
		})
	}
}

func absPath(t *testing.T, p string) string {
	t.Helper()
	abs, err := filepath.Abs(p)
	require.NoError(t, err)
	return abs
}

// captureStdout runs fn with os.Stdout redirected to a pipe and returns what
// was written. It is used to exercise Execute, which writes to os.Stdout.
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

func TestGenerateAndCheck(t *testing.T) {
	t.Setenv("GOWORK", "off")
	dir := copySimpleFixture(t)

	out, _, err := execute(t, BuildInfo{}, "generate", dir)
	require.NoError(t, err)
	assert.Contains(t, out, "wrote ")
	assert.Contains(t, out, "(1 catalog(s))")

	out, _, err = execute(t, BuildInfo{}, "check", dir)
	require.NoError(t, err)
	assert.Contains(t, out, "is up to date")
}

func TestGenerateOutputFlag(t *testing.T) {
	t.Setenv("GOWORK", "off")
	dir := copySimpleFixture(t)
	dst := filepath.Join(t.TempDir(), "spec.yaml")

	out, _, err := execute(t, BuildInfo{}, "generate", "-o", dst, dir)
	require.NoError(t, err)
	assert.Contains(t, out, "wrote "+dst)
	require.FileExists(t, dst)
	require.NoFileExists(t, filepath.Join(dir, "asyncapi.yaml"))
}

func TestGenerateOutputDirFlag(t *testing.T) {
	t.Setenv("GOWORK", "off")
	dir := copySimpleFixture(t)
	outDir := t.TempDir()

	out, _, err := execute(t, BuildInfo{}, "generate", "-o", outDir+string(os.PathSeparator), dir)
	require.NoError(t, err)
	dst := filepath.Join(outDir, "asyncapi.yaml")
	assert.Contains(t, out, "wrote "+dst)
	require.FileExists(t, dst)
	require.NoFileExists(t, filepath.Join(dir, "asyncapi.yaml"))
}

func TestGenerateJSONFormat(t *testing.T) {
	t.Setenv("GOWORK", "off")
	dir := copySimpleFixture(t)

	out, _, err := execute(t, BuildInfo{}, "generate", dir, "--format", "json")
	require.NoError(t, err)

	dst := filepath.Join(dir, "asyncapi.json")
	assert.Contains(t, out, "wrote "+dst)
	require.FileExists(t, dst)
	require.NoFileExists(t, filepath.Join(dir, "asyncapi.yaml"))

	content, err := os.ReadFile(dst)
	require.NoError(t, err)

	// The artifact shape: parseable, 2-space indented, newline-terminated.
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(content, &decoded))
	assert.Equal(t, "3.1.0", decoded["asyncapi"])
	assert.Contains(t, string(content), "\n  \"asyncapi\": \"3.1.0\"")
	assert.Equal(t, byte('\n'), content[len(content)-1], "output must be newline-terminated")
}

func TestGenerateJSONFormatOutputDir(t *testing.T) {
	t.Setenv("GOWORK", "off")
	dir := copySimpleFixture(t)
	outDir := t.TempDir()

	out, _, err := execute(
		t,
		BuildInfo{},
		"generate",
		dir,
		"-o",
		outDir+string(os.PathSeparator),
		"--format",
		"json",
	)
	require.NoError(t, err)

	dst := filepath.Join(outDir, "asyncapi.json")
	assert.Contains(t, out, "wrote "+dst)
	require.FileExists(t, dst)
	require.NoFileExists(t, filepath.Join(dir, "asyncapi.json"))
	require.NoFileExists(t, filepath.Join(dir, "asyncapi.yaml"))
}

func TestGenerateJSONFormatExplicitPath(t *testing.T) {
	t.Setenv("GOWORK", "off")
	dir := copySimpleFixture(t)
	// An explicit path wins whatever its extension: --format picks the encoding,
	// never rewrites a path the user typed.
	dst := filepath.Join(t.TempDir(), "custom.yaml")

	out, _, err := execute(t, BuildInfo{}, "generate", dir, "-o", dst, "--format", "json")
	require.NoError(t, err)
	assert.Contains(t, out, "wrote "+dst)

	content, err := os.ReadFile(dst)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(content, &decoded))
	assert.Equal(t, "3.1.0", decoded["asyncapi"])
}

func TestGenerateFormatYAMLExplicit(t *testing.T) {
	t.Setenv("GOWORK", "off")

	withoutFlag := copySimpleFixture(t)
	_, _, err := execute(t, BuildInfo{}, "generate", withoutFlag)
	require.NoError(t, err)

	withFlag := copySimpleFixture(t)
	_, _, err = execute(t, BuildInfo{}, "generate", withFlag, "--format", "yaml")
	require.NoError(t, err)

	// An explicit --format yaml is indistinguishable from an omitted flag.
	require.FileExists(t, filepath.Join(withFlag, "asyncapi.yaml"))
	require.NoFileExists(t, filepath.Join(withFlag, "asyncapi.json"))

	want, err := os.ReadFile(filepath.Join(withoutFlag, "asyncapi.yaml"))
	require.NoError(t, err)
	got, err := os.ReadFile(filepath.Join(withFlag, "asyncapi.yaml"))
	require.NoError(t, err)

	assert.Equal(t, string(want), string(got))
}

func TestGenerateBadFormat(t *testing.T) {
	t.Setenv("GOWORK", "off")

	t.Run("should_reject_an_unknown_format_and_write_nothing", func(t *testing.T) {
		dir := copySimpleFixture(t)

		_, _, err := execute(t, BuildInfo{}, "generate", dir, "--format", "xml")
		require.EqualError(t, err, `unknown format "xml": want "yaml" or "json"`)

		require.NoFileExists(t, filepath.Join(dir, "asyncapi.yaml"))
		require.NoFileExists(t, filepath.Join(dir, "asyncapi.json"))
	})

	t.Run("should_reject_an_unknown_format_before_resolving_the_directory", func(t *testing.T) {
		// Fail-fast: the format is validated before the directory is resolved or
		// the discovery harness runs, so a missing directory is not the error.
		missing := filepath.Join(t.TempDir(), "missing")

		_, _, err := execute(t, BuildInfo{}, "generate", missing, "--format", "xml")
		require.EqualError(t, err, `unknown format "xml": want "yaml" or "json"`)
		require.NoDirExists(t, missing)
	})
}

func TestGenerateNoCatalogs(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("..", "..", "spec"))
	require.NoError(t, err)

	_, _, err = execute(t, BuildInfo{}, "generate", dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no AsyncAPI catalogs")
}

func TestGenerateValidationError(t *testing.T) {
	t.Setenv("GOWORK", "off")
	dir := copyFixture(t, "invalid")

	_, _, err := execute(t, BuildInfo{}, "generate", dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid AsyncAPI catalog(s): 1")
	assert.Contains(t, err.Error(), "test/data/invalid.Catalog:")
	assert.Contains(t, err.Error(), "server.prod.host: is required")
	assert.Contains(t, err.Error(), `channel.order-placed: references unknown server "staging"`)
}

func TestCheckOutOfDate(t *testing.T) {
	t.Setenv("GOWORK", "off")
	dir := copySimpleFixture(t)

	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "asyncapi.yaml"),
		[]byte("asyncapi: 0.0.0\n"),
		0o644,
	))

	_, _, err := execute(t, BuildInfo{}, "check", dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of date")
}

func TestCheckMissing(t *testing.T) {
	t.Setenv("GOWORK", "off")
	dir := copySimpleFixture(t)

	_, _, err := execute(t, BuildInfo{}, "check", dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestCheckNoCatalogs(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("..", "..", "spec"))
	require.NoError(t, err)

	_, _, err = execute(t, BuildInfo{}, "check", dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no AsyncAPI catalogs")
}

func TestGenerateWriteError(t *testing.T) {
	t.Setenv("GOWORK", "off")
	dir := copySimpleFixture(t)
	// A directory in place of asyncapi.yaml forces os.WriteFile to fail.
	require.NoError(t, os.Mkdir(filepath.Join(dir, "asyncapi.yaml"), 0o755))

	_, _, err := execute(t, BuildInfo{}, "generate", dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "writing ")
}

func TestCheckReadError(t *testing.T) {
	t.Setenv("GOWORK", "off")
	dir := copySimpleFixture(t)
	// A directory in place of asyncapi.yaml makes os.ReadFile fail with a
	// non-IsNotExist error.
	require.NoError(t, os.Mkdir(filepath.Join(dir, "asyncapi.yaml"), 0o755))

	_, _, err := execute(t, BuildInfo{}, "check", dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading ")
}

// copySimpleFixture copies the simple test fixture into a fresh temp directory.
func copySimpleFixture(t *testing.T) string {
	return copyFixture(t, "simple")
}

// copyFixture copies the named test fixture into a fresh temp directory,
// rewrites its replace directive to the absolute repo root, and removes the
// committed asyncapi.yaml (when present) so generate/check operate on a clean
// slate without racing the integration golden test over the shared files.
func copyFixture(t *testing.T, name string) string {
	t.Helper()

	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	src := filepath.Join(repoRoot, "test", "data", name)

	dst := t.TempDir()
	require.NoError(t, os.CopyFS(dst, os.DirFS(src)))

	goMod, err := os.ReadFile(filepath.Join(dst, "go.mod"))
	require.NoError(t, err)
	replaced := strings.ReplaceAll(string(goMod), "=> ../../../", "=> "+repoRoot)
	require.NoError(t, os.WriteFile(filepath.Join(dst, "go.mod"), []byte(replaced), 0o644))

	if _, err := os.Stat(filepath.Join(dst, "asyncapi.yaml")); err == nil {
		require.NoError(t, os.Remove(filepath.Join(dst, "asyncapi.yaml")))
	}

	return dst
}
