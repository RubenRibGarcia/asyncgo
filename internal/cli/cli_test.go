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

// repoRootDir resolves a directory under the repository, for the tests that run
// against a committed fixture rather than a copy of it.
func repoRootDir(t *testing.T, name string) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("..", "..", name))
	require.NoError(t, err)
	return dir
}

// TestGenerateArtifact covers where each flag combination puts the artifact and
// what the artifact is. Every case runs generate once and asserts the same two
// things — the wrote line names the file, the file exists — so they are rows of
// one table; a row adds only the assertions its own flags call for.
func TestGenerateArtifact(t *testing.T) {
	t.Setenv("GOWORK", "off")

	tests := []struct {
		name string
		// args builds the generate arguments and names the artifact they must
		// produce. Both come from one closure because an explicit -o needs a
		// temp dir that the expectation has to name too.
		args func(t *testing.T, dir string) (args []string, want string)
		// verify adds what the flags make worth checking beyond existence.
		verify func(t *testing.T, dir, dst, stdout string)
	}{
		{
			name: "should_write_the_default_yaml_into_the_fixture",
			args: func(_ *testing.T, dir string) ([]string, string) {
				return []string{"generate", dir}, filepath.Join(dir, "asyncapi.yaml")
			},
		},
		{
			name: "should_write_the_file_named_by_the_output_flag",
			args: func(t *testing.T, dir string) ([]string, string) {
				dst := filepath.Join(t.TempDir(), "spec.yaml")
				return []string{"generate", "-o", dst, dir}, dst
			},
		},
		{
			name: "should_treat_a_trailing_separator_as_a_directory",
			args: func(t *testing.T, dir string) ([]string, string) {
				outDir := t.TempDir()
				args := []string{"generate", "-o", outDir + string(os.PathSeparator), dir}
				return args, filepath.Join(outDir, "asyncapi.yaml")
			},
		},
		{
			name: "should_encode_json_for_the_format_flag",
			args: func(_ *testing.T, dir string) ([]string, string) {
				return []string{"generate", dir, "--format", "json"}, filepath.Join(
					dir,
					"asyncapi.json",
				)
			},
			verify: func(t *testing.T, dir, dst, _ string) {
				t.Helper()
				// The artifact shape: parseable, 2-space indented, newline-terminated.
				content, err := os.ReadFile(dst)
				require.NoError(t, err)

				var decoded map[string]any
				require.NoError(t, json.Unmarshal(content, &decoded))
				assert.Equal(t, "3.1.0", decoded["asyncapi"])
				assert.Contains(t, string(content), "\n  \"asyncapi\": \"3.1.0\"")
				assert.Equal(
					t,
					byte('\n'),
					content[len(content)-1],
					"output must be newline-terminated",
				)
				require.NoFileExists(t, filepath.Join(dir, "asyncapi.yaml"))
			},
		},
		{
			name: "should_honour_an_output_directory_with_json",
			args: func(t *testing.T, dir string) ([]string, string) {
				outDir := t.TempDir()
				args := []string{
					"generate",
					dir,
					"-o",
					outDir + string(os.PathSeparator),
					"--format",
					"json",
				}
				return args, filepath.Join(outDir, "asyncapi.json")
			},
		},
		{
			// An explicit --format yaml is indistinguishable from an omitted flag.
			name: "should_treat_explicit_yaml_as_the_default",
			args: func(_ *testing.T, dir string) ([]string, string) {
				return []string{"generate", dir, "--format", "yaml"}, filepath.Join(
					dir,
					"asyncapi.yaml",
				)
			},
			verify: func(t *testing.T, _, dst, _ string) {
				t.Helper()
				baseline := copySimpleFixture(t)
				_, _, err := execute(t, BuildInfo{}, "generate", baseline)
				require.NoError(t, err)

				want, err := os.ReadFile(filepath.Join(baseline, "asyncapi.yaml"))
				require.NoError(t, err)
				got, err := os.ReadFile(dst)
				require.NoError(t, err)

				assert.Equal(t, string(want), string(got))
				require.NoFileExists(t, filepath.Join(filepath.Dir(dst), "asyncapi.json"))
			},
		},
		{
			name: "should_keep_an_explicit_path_whatever_its_extension",
			args: func(t *testing.T, dir string) ([]string, string) {
				// An explicit path wins whatever its extension: --format picks the
				// encoding, never rewrites a path the user typed.
				dst := filepath.Join(t.TempDir(), "custom.yaml")
				return []string{"generate", dir, "-o", dst, "--format", "json"}, dst
			},
			verify: func(t *testing.T, _ string, dst, _ string) {
				t.Helper()
				content, err := os.ReadFile(dst)
				require.NoError(t, err)

				var decoded map[string]any
				require.NoError(t, json.Unmarshal(content, &decoded))
				assert.Equal(t, "3.1.0", decoded["asyncapi"])
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := copySimpleFixture(t)
			args, dst := tc.args(t, dir)

			stdout, _, err := execute(t, BuildInfo{}, args...)
			require.NoError(t, err)
			assert.Contains(t, stdout, "wrote "+dst)
			require.FileExists(t, dst)

			// An artifact written outside the fixture leaves both defaults
			// untouched; a row that writes into the fixture says so in verify.
			if filepath.Dir(dst) != dir {
				require.NoFileExists(t, filepath.Join(dir, "asyncapi.yaml"))
				require.NoFileExists(t, filepath.Join(dir, "asyncapi.json"))
			}
			if tc.verify != nil {
				tc.verify(t, dir, dst, stdout)
			}
		})
	}
}

// fixtureArgs returns command arguments that run against a fresh copy of the
// simple fixture, with the workspace disabled: the copy carries its own go.mod,
// so the go.work file must not pull it back into the repository module.
func fixtureArgs(t *testing.T, command string, flags ...string) []string {
	t.Helper()
	t.Setenv("GOWORK", "off")
	return append([]string{command, copySimpleFixture(t)}, flags...)
}

// TestCommandErrors covers the generate and check failure paths: rows that each
// need a differently prepared directory and each assert one message, which is
// the shape a table is for.
func TestCommandErrors(t *testing.T) {
	tests := []struct {
		name string
		// setup prepares the directory to run against and returns the command
		// arguments, command included.
		setup func(t *testing.T) []string
		// wantErr is a substring of the error the command must report.
		wantErr string
		// verify adds assertions about what the failed run left behind.
		verify func(t *testing.T, args []string)
	}{
		{
			name: "should_report_no_catalogs_on_generate",
			setup: func(t *testing.T) []string {
				return []string{"generate", repoRootDir(t, "spec")}
			},
			wantErr: "no AsyncAPI catalogs",
		},
		{
			name: "should_report_no_catalogs_on_check",
			setup: func(t *testing.T) []string {
				return []string{"check", repoRootDir(t, "spec")}
			},
			wantErr: "no AsyncAPI catalogs",
		},
		{
			name: "should_reject_an_unknown_format_and_write_nothing",
			setup: func(t *testing.T) []string {
				return fixtureArgs(t, "generate", "--format", "xml")
			},
			wantErr: `unknown format "xml": want "yaml" or "json"`,
			verify: func(t *testing.T, args []string) {
				t.Helper()
				dir := args[1]
				require.NoFileExists(t, filepath.Join(dir, "asyncapi.yaml"))
				require.NoFileExists(t, filepath.Join(dir, "asyncapi.json"))
			},
		},
		{
			name: "should_reject_an_unknown_format_before_resolving_the_directory",
			setup: func(t *testing.T) []string {
				// Fail-fast: the format is validated before the directory is
				// resolved or the discovery harness runs, so a missing directory
				// is not the error.
				missing := filepath.Join(t.TempDir(), "missing")
				return []string{"generate", missing, "--format", "xml"}
			},
			wantErr: `unknown format "xml": want "yaml" or "json"`,
			verify: func(t *testing.T, args []string) {
				t.Helper()
				require.NoDirExists(t, args[1])
			},
		},
		{
			name: "should_reject_an_unknown_format_on_check",
			setup: func(t *testing.T) []string {
				return fixtureArgs(t, "check", "--format", "xml")
			},
			wantErr: `unknown format "xml"`,
		},
		{
			// A directory in place of asyncapi.yaml forces os.WriteFile to fail.
			name: "should_report_a_write_error",
			setup: func(t *testing.T) []string {
				t.Setenv("GOWORK", "off")
				dir := copySimpleFixture(t)
				require.NoError(t, os.Mkdir(filepath.Join(dir, "asyncapi.yaml"), 0o755))
				return []string{"generate", dir}
			},
			wantErr: "writing ",
		},
		{
			// A directory in place of asyncapi.yaml makes os.ReadFile fail with a
			// non-IsNotExist error.
			name: "should_report_a_read_error",
			setup: func(t *testing.T) []string {
				t.Setenv("GOWORK", "off")
				dir := copySimpleFixture(t)
				require.NoError(t, os.Mkdir(filepath.Join(dir, "asyncapi.yaml"), 0o755))
				return []string{"check", dir}
			},
			wantErr: "reading ",
		},
		{
			name: "should_report_an_out_of_date_artifact",
			setup: func(t *testing.T) []string {
				t.Setenv("GOWORK", "off")
				dir := copySimpleFixture(t)
				require.NoError(t, os.WriteFile(
					filepath.Join(dir, "asyncapi.yaml"),
					[]byte("asyncapi: 0.0.0\n"),
					0o644,
				))
				return []string{"check", dir}
			},
			wantErr: "out of date",
		},
		{
			name: "should_report_a_missing_artifact",
			setup: func(t *testing.T) []string {
				return fixtureArgs(t, "check")
			},
			wantErr: "not found",
		},
		{
			// The yaml default is not consulted when json is requested: the
			// missing artifact has to be named so the user sees which generate
			// run is needed.
			name: "should_name_the_missing_json_artifact",
			setup: func(t *testing.T) []string {
				return fixtureArgs(t, "check", "--format", "json")
			},
			wantErr: "asyncapi.json not found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			args := tc.setup(t)

			_, _, err := execute(t, BuildInfo{}, args...)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
			if tc.verify != nil {
				tc.verify(t, args)
			}
		})
	}
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
	assert.Contains(t, err.Error(), "securityScheme.oauth basic: name must match ^[\\w\\d.\\-_]+$")
	assert.Contains(
		t,
		err.Error(),
		"channel.orders/{orderId}: address expression {orderId} has no declared parameter",
	)
	assert.Contains(t, err.Error(), "operation.orders.send: duplicate action")
	assert.Contains(t, err.Error(), "channel.orders-shipped: duplicate address")
}

// TestCheckFormatAndOutput covers the flags check shares with generate: without
// them, check could only ever verify a yaml document written to the default
// path, and always failed against json or a custom -o.
func TestCheckFormatAndOutput(t *testing.T) {
	t.Setenv("GOWORK", "off")
	tests := []struct {
		name string
		args func(t *testing.T) []string
		want string // artifact the check compares against
	}{
		{
			name: "should_check_json_written_by_generate",
			args: func(*testing.T) []string { return []string{"--format", "json"} },
			want: "asyncapi.json",
		},
		{
			name: "should_check_a_custom_output_path",
			// A relative -o is resolved against the process cwd, which under
			// `go test` is this package, so the artifact has to be aimed at a
			// temp dir explicitly or the test drops custom.json in the source
			// tree.
			args: func(t *testing.T) []string {
				return []string{"--format", "json", "-o", filepath.Join(t.TempDir(), "custom.json")}
			},
			want: "custom.json",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := copySimpleFixture(t)
			args := tt.args(t)

			_, _, err := execute(t, BuildInfo{}, append([]string{"generate", dir}, args...)...)
			require.NoError(t, err)

			stdout, _, err := execute(t, BuildInfo{}, append([]string{"check", dir}, args...)...)
			require.NoError(t, err)
			assert.Contains(t, stdout, tt.want+" is up to date")
		})
	}
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
