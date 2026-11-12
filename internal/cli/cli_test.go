package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
	tests := []struct {
		name string
		info BuildInfo
		want string
	}{
		{
			name: "should_return_bare_version",
			info: BuildInfo{Version: "v1.2.3"},
			want: "v1.2.3",
		},
		{
			name: "should_append_commit_and_date",
			info: BuildInfo{
				Version: "v1.2.3",
				Commit:  "abc1234",
				Date:    "2026-08-18T00:00:00Z",
			},
			want: "v1.2.3 (commit abc1234, built 2026-08-18T00:00:00Z)",
		},
		{
			// A build that stamps the commit but not the date must not render
			// "built " with nothing after it.
			name: "should_omit_an_empty_date",
			info: BuildInfo{Version: "v1.2.3", Commit: "abc1234"},
			want: "v1.2.3 (commit abc1234)",
		},
		{
			name: "should_omit_an_empty_commit",
			info: BuildInfo{Version: "v1.2.3", Date: "2026-08-18T00:00:00Z"},
			want: "v1.2.3 (built 2026-08-18T00:00:00Z)",
		},
		{
			// "unknown" is the toolchain's placeholder for "not injected".
			name: "should_treat_an_unknown_commit_as_absent",
			info: BuildInfo{Version: "v1.2.3", Commit: "unknown", Date: "2026-08-18T00:00:00Z"},
			want: "v1.2.3 (built 2026-08-18T00:00:00Z)",
		},
		{
			name: "should_fall_back_to_devel_for_an_untagged_build",
			info: BuildInfo{},
			want: "devel",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, formatVersion(tc.info))
		})
	}
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
		wantMode    outputMode
	}{
		{
			name:        "should_default_to_dir_asyncapi_yaml",
			defaultName: "asyncapi.yaml",
			want:        filepath.Join(dir, "asyncapi.yaml"),
			wantMode:    modeFile,
		},
		{
			name:        "should_default_to_dir_asyncapi_json",
			defaultName: "asyncapi.json",
			want:        filepath.Join(dir, "asyncapi.json"),
			wantMode:    modeFile,
		},
		{
			// The trailing separator is the whole distinction between one merged
			// document and one document per catalog, so it resolves to the directory
			// itself rather than to a default file inside it.
			name:     "should_resolve_a_trailing_separator_to_the_directory",
			output:   "out/",
			want:     absPath(t, "out"),
			wantMode: modeDir,
		},
		{
			name:     "should_resolve_a_trailing_os_separator_to_the_directory",
			output:   "out" + string(os.PathSeparator),
			want:     absPath(t, "out"),
			wantMode: modeDir,
		},
		{
			name:        "should_absolutize_explicit_file_whatever_its_extension",
			output:      "custom.yaml",
			defaultName: "asyncapi.json",
			want:        absPath(t, "custom.yaml"),
			wantMode:    modeFile,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, mode, err := resolveOutput(dir, tc.output, tc.defaultName)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.wantMode, mode)
		})
	}
}

func TestArtifactName(t *testing.T) {
	t.Run("should_carry_the_asyncapi_infix", func(t *testing.T) {
		assert.Equal(t, "orders.asyncapi.yaml", artifactName("orders", "yaml"))
		assert.Equal(t, "orders.asyncapi.json", artifactName("orders", "json"))
	})

	t.Run("should_match_exactly_the_names_it_writes", func(t *testing.T) {
		// check adopts files by pattern and calls the ones nothing claims orphans,
		// so the pattern and the writer must not drift apart.
		for _, format := range []string{"yaml", "json"} {
			matched, err := filepath.Match(artifactPattern(format), artifactName("a.b", format))
			require.NoError(t, err)
			assert.True(
				t,
				matched,
				"%s must match %s",
				artifactName("a.b", format),
				artifactPattern(format),
			)

			other := "yaml"
			if format == "yaml" {
				other = "json"
			}
			matched, err = filepath.Match(artifactPattern(other), artifactName("a.b", format))
			require.NoError(t, err)
			assert.False(t, matched, "a %s artifact must not be adopted by a %s run", format, other)
		}
	})
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
			name: "should_write_one_document_per_catalog_for_a_directory",
			args: func(t *testing.T, dir string) ([]string, string) {
				outDir := t.TempDir()
				args := []string{"generate", "-o", outDir + string(os.PathSeparator), dir}
				return args, filepath.Join(outDir, "simple.Catalog.asyncapi.yaml")
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
				return args, filepath.Join(outDir, "simple.Catalog.asyncapi.json")
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

// addSecondCatalog writes a second catalog into its own package inside the
// fixture module. Discovery scope is the module (ADR 0010), so this is what makes
// the copy multi-catalog, without adding a committed fixture that the integration
// golden test would also have to cover. name is the document name the catalog
// declares, or "" for none.
func addSecondCatalog(t *testing.T, dir, name string) {
	t.Helper()

	pkgDir := filepath.Join(dir, "extra")
	require.NoError(t, os.MkdirAll(pkgDir, 0o755))

	named := ""
	if name != "" {
		named = fmt.Sprintf(".Named(%q)", name)
	}
	body := fmt.Sprintf(`package extra

import "github.com/RubenRibGarcia/asyncgo"

type Pinged struct {
	ID string %s
}

var Catalog = asyncgo.Spec(
	asyncgo.Info("Extra Service", "2.0.0"),
	asyncgo.Channels(
		asyncgo.Channel("pinged").
			Send(asyncgo.Operation().Message(asyncgo.MessageOf(Pinged{}).Name("Pinged"))),
	),
)%s
`, "`json:\"id\" asyncapi:\"required\"`", named)

	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "catalog.go"), []byte(body), 0o644))
}

// writeArtifact writes body to path, creating no directories: every caller aims
// at a temp dir it just made.
func writeArtifact(t *testing.T, path, body string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
}

// readArtifact reads path as a string.
func readArtifact(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(content)
}

// TestGeneratePerCatalog covers directory mode: an -o with a trailing separator
// writes one document per catalog instead of merging them, which is what lets a
// multi-catalog module commit a correct document for each one.
func TestGeneratePerCatalog(t *testing.T) {
	t.Setenv("GOWORK", "off")

	t.Run("should_write_one_document_per_catalog", func(t *testing.T) {
		dir := copySimpleFixture(t)
		addSecondCatalog(t, dir, "")
		outDir := t.TempDir()

		stdout, _, err := execute(t, BuildInfo{}, "generate", dir, "-o", outDir+"/")
		require.NoError(t, err)

		simple := filepath.Join(outDir, "simple.Catalog.asyncapi.yaml")
		extra := filepath.Join(outDir, "extra.Catalog.asyncapi.yaml")
		assert.Contains(t, stdout, "wrote "+simple)
		assert.Contains(t, stdout, "wrote "+extra)
		require.FileExists(t, simple)
		require.FileExists(t, extra)

		// Each document keeps its own Info and only its own channels: the merge
		// drops the second catalog's title, which is the whole point of the mode.
		assert.Contains(t, readArtifact(t, simple), "title: Orders Service")
		assert.Contains(t, readArtifact(t, extra), "title: Extra Service")
		assert.Contains(t, readArtifact(t, extra), "pinged")
		assert.NotContains(t, readArtifact(t, extra), "order-placed")
		assert.NotContains(t, readArtifact(t, simple), "pinged")
	})

	t.Run("should_use_the_declared_name", func(t *testing.T) {
		dir := copySimpleFixture(t)
		addSecondCatalog(t, dir, "audit")
		outDir := t.TempDir()

		_, _, err := execute(t, BuildInfo{}, "generate", dir, "-o", outDir+"/")
		require.NoError(t, err)

		require.FileExists(t, filepath.Join(outDir, "audit.asyncapi.yaml"))
		require.NoFileExists(t, filepath.Join(outDir, "extra.Catalog.asyncapi.yaml"))
	})

	t.Run("should_reject_two_catalogs_that_share_a_name", func(t *testing.T) {
		dir := copySimpleFixture(t)
		// The derived stem is <package>.<variable>, so this collides with the
		// simple fixture's own Catalog.
		addSecondCatalog(t, dir, "simple.Catalog")
		outDir := t.TempDir()

		_, _, err := execute(t, BuildInfo{}, "generate", dir, "-o", outDir+"/")

		require.Error(t, err)
		assert.Contains(t, err.Error(), `duplicate document name "simple.Catalog"`)
		// Nothing is written: the whole set is validated and encoded before the
		// first file, so a failure cannot leave a half-written directory behind.
		written, err := filepath.Glob(filepath.Join(outDir, "*.asyncapi.yaml"))
		require.NoError(t, err)
		assert.Empty(t, written)
	})

	t.Run("should_encode_every_document_as_json", func(t *testing.T) {
		dir := copySimpleFixture(t)
		outDir := t.TempDir()

		_, _, err := execute(
			t, BuildInfo{}, "generate", dir, "-o", outDir+"/", "--format", "json",
		)
		require.NoError(t, err)

		content := readArtifact(t, filepath.Join(outDir, "simple.Catalog.asyncapi.json"))
		var decoded map[string]any
		require.NoError(t, json.Unmarshal([]byte(content), &decoded))
		assert.Equal(t, "3.1.0", decoded["asyncapi"])
	})

	t.Run("should_leave_the_merged_default_untouched", func(t *testing.T) {
		dir := copySimpleFixture(t)
		addSecondCatalog(t, dir, "")
		outDir := t.TempDir()

		_, _, err := execute(t, BuildInfo{}, "generate", dir, "-o", outDir+"/")
		require.NoError(t, err)

		require.NoFileExists(t, filepath.Join(dir, "asyncapi.yaml"))
	})
}

// TestCheckPerCatalog covers the directory-mode verdict: one exit code for the
// whole artifact set, with stale, missing and orphan documents all reported.
func TestCheckPerCatalog(t *testing.T) {
	t.Setenv("GOWORK", "off")

	// prepare generates a two-catalog module's artifact set into a temp dir.
	prepare := func(t *testing.T) (dir, outDir string) {
		t.Helper()
		dir = copySimpleFixture(t)
		addSecondCatalog(t, dir, "")
		outDir = t.TempDir()

		_, _, err := execute(t, BuildInfo{}, "generate", dir, "-o", outDir+"/")
		require.NoError(t, err)
		return dir, outDir
	}

	t.Run("should_pass_when_every_document_matches", func(t *testing.T) {
		dir, outDir := prepare(t)

		stdout, _, err := execute(t, BuildInfo{}, "check", dir, "-o", outDir+"/")
		require.NoError(t, err)
		assert.Contains(t, stdout, "2 document(s) up to date")
	})

	t.Run("should_fail_when_one_document_is_stale", func(t *testing.T) {
		dir, outDir := prepare(t)
		writeArtifact(t, filepath.Join(outDir, "simple.Catalog.asyncapi.yaml"), "asyncapi: 0.0.0\n")

		_, _, err := execute(t, BuildInfo{}, "check", dir, "-o", outDir+"/")

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrOutOfDate, "a stale document keeps CI on one exit code")
		assert.Contains(t, err.Error(), "simple.Catalog.asyncapi.yaml")
	})

	t.Run("should_fail_when_a_document_is_missing", func(t *testing.T) {
		dir, outDir := prepare(t)
		require.NoError(t, os.Remove(filepath.Join(outDir, "extra.Catalog.asyncapi.yaml")))

		_, _, err := execute(t, BuildInfo{}, "check", dir, "-o", outDir+"/")

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrOutOfDate)
		assert.Contains(t, err.Error(), "extra.Catalog.asyncapi.yaml")
	})

	t.Run("should_fail_on_an_orphan_artifact", func(t *testing.T) {
		dir, outDir := prepare(t)
		writeArtifact(t, filepath.Join(outDir, "deleted.asyncapi.yaml"), "asyncapi: 3.1.0\n")

		_, _, err := execute(t, BuildInfo{}, "check", dir, "-o", outDir+"/")

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrOutOfDate)
		assert.Contains(t, err.Error(), "deleted.asyncapi.yaml")
	})

	t.Run("should_ignore_yaml_asyncgo_never_wrote", func(t *testing.T) {
		dir, outDir := prepare(t)
		// Attribution comes from the filename. Adopting unrelated YAML would fail CI
		// on a file nobody generated.
		writeArtifact(t, filepath.Join(outDir, "architecture.yaml"), "not: asyncapi\n")

		_, _, err := execute(t, BuildInfo{}, "check", dir, "-o", outDir+"/")
		require.NoError(t, err)
	})

	t.Run("should_report_the_orphan_as_a_finding_with_verbose", func(t *testing.T) {
		dir, outDir := prepare(t)
		writeArtifact(t, filepath.Join(outDir, "deleted.asyncapi.yaml"), "asyncapi: 3.1.0\n")

		_, stderr, err := execute(t, BuildInfo{}, "check", dir, "-o", outDir+"/", "-v")

		require.Error(t, err)
		assert.Contains(t, stderr, "orphan")
		assert.Contains(t, stderr, "deleted.asyncapi.yaml")
	})
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

// TestGenerateRejectsAnUnknownDirective is the end-to-end form of the static tag
// pass: a misspelt directive used to derive a document that silently lacked the
// constraint its author meant to add. It has to reach the CLI as a located build
// error — not as a document, and not as exit 3, which would tell CI to regenerate
// a catalog that is broken rather than stale.
func TestGenerateRejectsAnUnknownDirective(t *testing.T) {
	t.Setenv("GOWORK", "off")
	dir := copySimpleFixture(t)

	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "typo.go"),
		[]byte("package simple\n\n"+
			"// TagTypo carries a misspelt directive.\n"+
			"type TagTypo struct {\n"+
			"\tAmount string `json:\"amount\" asyncapi:\"requird\"`\n"+
			"}\n"),
		0o644,
	))

	for _, cmd := range []string{"generate", "check"} {
		t.Run(cmd, func(t *testing.T) {
			_, _, err := execute(t, BuildInfo{}, cmd, dir)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "typo.go:5:", "the error names the position")
			assert.Contains(
				t,
				err.Error(),
				`asyncapi:"requird": unknown directive "requird"; did you mean "required"?`,
			)
			assert.Equal(t, 1, ExitCode(err), "a broken catalog is exit 1, not 3")
		})
	}
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

// simplePayloadFQN is the payload type of the simple fixture, which the --config
// tests map instead of deriving.
const simplePayloadFQN = "github.com/RubenRibGarcia/asyncgo/test/data/simple.OrderPlaced"

// writeMappingConfig writes a mapping config file and returns its path. It lives
// outside the module on purpose: --config names a file, and the tests must not be
// able to confuse it with the auto-discovered one.
func writeMappingConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "team.yaml")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
	return path
}

// TestConfigFlag covers --config on both commands: the named file is the only
// mapping set in force, generate and check read it alike, and a path that does not
// exist fails the command instead of silently mapping nothing.
func TestConfigFlag(t *testing.T) {
	t.Setenv("GOWORK", "off")

	uuidConfig := func() string {
		return writeMappingConfig(t, "mappings:\n  "+simplePayloadFQN+
			":\n    type: string\n    format: uuid\n")
	}

	t.Run("should_apply_the_mappings_from_the_named_config", func(t *testing.T) {
		dir := copySimpleFixture(t)

		out, _, err := execute(t, BuildInfo{}, "generate", dir, "--config", uuidConfig())
		require.NoError(t, err)
		assert.Contains(t, out, "(1 catalog(s))")

		artifact, err := os.ReadFile(filepath.Join(dir, "asyncapi.yaml"))
		require.NoError(t, err)
		// The payload is the mapping, not the derived object it replaced.
		assert.Contains(t, string(artifact), "format: uuid")
		assert.NotContains(t, string(artifact), "order_id")
	})

	t.Run("should_replace_the_default_config_file", func(t *testing.T) {
		dir := copySimpleFixture(t)
		// A default file that would be picked up without the flag, with a marker the
		// assertion can tell apart from the explicit file's.
		require.NoError(t, os.WriteFile(
			filepath.Join(dir, "asyncgo.yaml"),
			[]byte("mappings:\n  "+simplePayloadFQN+":\n    type: string\n    format: uri\n"),
			0o644,
		))

		_, _, err := execute(t, BuildInfo{}, "generate", dir, "--config", uuidConfig())
		require.NoError(t, err)

		artifact, err := os.ReadFile(filepath.Join(dir, "asyncapi.yaml"))
		require.NoError(t, err)
		assert.Contains(t, string(artifact), "format: uuid")
		assert.NotContains(t, string(artifact), "format: uri")
	})

	t.Run("should_check_against_the_same_named_config", func(t *testing.T) {
		dir := copySimpleFixture(t)
		config := uuidConfig()

		_, _, err := execute(t, BuildInfo{}, "generate", dir, "--config", config)
		require.NoError(t, err)

		out, _, err := execute(t, BuildInfo{}, "check", dir, "--config", config)
		require.NoError(t, err)
		assert.Contains(t, out, "is up to date")

		// Without the flag check derives the unmapped document, so it must see the
		// artifact as stale — proof the flag is what made the pass above succeed.
		_, _, err = execute(t, BuildInfo{}, "check", dir)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrOutOfDate)
	})

	t.Run("should_reject_a_missing_config_file", func(t *testing.T) {
		dir := copySimpleFixture(t)
		missing := filepath.Join(t.TempDir(), "typo.yaml")

		_, _, err := execute(t, BuildInfo{}, "generate", dir, "--config", missing)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "reading "+missing)
	})
}

// TestExitCode pins the process exit status each outcome maps to. CI tells "the
// artifact is stale" from "the build is broken" by this number, so the mapping is
// part of the CLI contract: 0 success, 1 failure, 3 out of date, with 2 already
// claimed by Cobra for usage errors.
func TestExitCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "should_return_zero_for_success", want: 0},
		{
			name: "should_return_three_for_an_out_of_date_artifact",
			err:  fmt.Errorf("%s: %w; run generate", "asyncapi.yaml", ErrOutOfDate),
			want: 3,
		},
		{
			// Wrapped twice: errors.Is has to survive the layers a command adds.
			name: "should_return_three_for_a_wrapped_out_of_date_error",
			err:  fmt.Errorf("checking: %w", fmt.Errorf("%s: %w", "asyncapi.yaml", ErrOutOfDate)),
			want: 3,
		},
		{name: "should_return_one_for_any_other_error", err: errors.New("boom"), want: 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ExitCode(tc.err))
		})
	}
}

// TestCheckOutOfDateSentinel pins that a stale artifact is reported as
// ErrOutOfDate rather than as a plain error, which is what makes the exit code
// above reachable from the command.
func TestCheckOutOfDateSentinel(t *testing.T) {
	t.Setenv("GOWORK", "off")
	dir := copySimpleFixture(t)
	require.NoError(t, execute1(t, "generate", dir))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "asyncapi.yaml"),
		[]byte("asyncapi: 0.0.0\n"),
		0o644,
	))

	_, _, err := execute(t, BuildInfo{}, "check", dir)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrOutOfDate)
	assert.Equal(t, 3, ExitCode(err))
}

// execute1 runs a command that must succeed, for tests that need a generated
// artifact before the assertion they care about.
func execute1(t *testing.T, args ...string) error {
	t.Helper()
	_, _, err := execute(t, BuildInfo{}, args...)
	return err
}

// TestWriteFileAtomic pins what the artifact write guarantees: the target is
// replaced, and a failed write leaves neither a partial artifact nor a temp file
// behind. `generate` rewrites the same path on every run, so a truncated artifact
// from a failed write would be worse than no write at all.
func TestWriteFileAtomic(t *testing.T) {
	t.Run("should_replace_an_existing_artifact", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "asyncapi.yaml")
		require.NoError(t, os.WriteFile(path, []byte("old\n"), 0o644))

		require.NoError(t, writeFileAtomic(path, []byte("new\n"), 0o644))

		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, "new\n", string(got))
	})

	t.Run("should_leave_no_temp_file_behind", func(t *testing.T) {
		dir := t.TempDir()

		require.NoError(t, writeFileAtomic(filepath.Join(dir, "asyncapi.yaml"), []byte("x"), 0o644))

		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		require.Len(t, entries, 1)
		assert.Equal(t, "asyncapi.yaml", entries[0].Name())
	})

	t.Run("should_fail_when_the_target_directory_does_not_exist", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "missing")

		require.Error(
			t,
			writeFileAtomic(filepath.Join(missing, "asyncapi.yaml"), []byte("x"), 0o644),
		)

		require.NoDirExists(t, missing)
	})

	t.Run("should_leave_the_target_untouched_when_the_rename_fails", func(t *testing.T) {
		dir := t.TempDir()
		// A non-empty directory in place of the artifact makes os.Rename fail with
		// ENOTEMPTY regardless of the user running the test, so the failure path is
		// exercised without relying on permissions.
		path := filepath.Join(dir, "asyncapi.yaml")
		require.NoError(t, os.Mkdir(path, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(path, "keep.yaml"), []byte("keep"), 0o644))

		require.Error(t, writeFileAtomic(path, []byte("new"), 0o644))

		// The previous artifact survives, and the temp file was removed rather than
		// left beside it for the next run to trip over.
		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		require.Len(t, entries, 1)
		assert.Equal(t, "asyncapi.yaml", entries[0].Name())
		keep, err := os.ReadFile(filepath.Join(path, "keep.yaml"))
		require.NoError(t, err)
		assert.Equal(t, "keep", string(keep))
	})
}
