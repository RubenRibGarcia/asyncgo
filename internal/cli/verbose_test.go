package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// simpleFixture is the module the verbose tests generate from: one catalog, one
// channel, so the stage story is short enough to assert in full.
func simpleFixture(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("..", "..", "test", "data", "simple"))
	require.NoError(t, err)
	return dir
}

// generateTo writes the fixture's document to a temp path and returns it.
func generateTo(t *testing.T) string {
	t.Helper()
	dir := simpleFixture(t)
	out := filepath.Join(t.TempDir(), "asyncapi.yaml")

	_, _, err := execute(t, BuildInfo{}, "generate", dir, "-o", out)
	require.NoError(t, err)
	return out
}

// drift rewrites a channel name in the artifact, which is the smallest edit that
// leaves a valid document that no longer matches the module.
func drift(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	drifted := strings.ReplaceAll(string(data), "order-placed", "order-renamed")
	require.NotEqual(t, string(data), drifted, "the fixture must contain that channel")
	require.NoError(t, os.WriteFile(path, []byte(drifted), 0o644))
}

// TestGenerateVerbose pins the shape of the stage story: quiet by default, tagged
// and narrated at -v, recapped at -vv, and never on stdout — which stays reserved
// for the artifact line.
func TestGenerateVerbose(t *testing.T) {
	dir := simpleFixture(t)

	t.Run("should_print_no_stage_lines_without_the_flag", func(t *testing.T) {
		out := filepath.Join(t.TempDir(), "asyncapi.yaml")

		stdout, stderr, err := execute(t, BuildInfo{}, "generate", dir, "-o", out)

		require.NoError(t, err)
		assert.Empty(t, stderr)
		assert.Equal(t, "wrote "+out+" (1 catalog(s))\n", stdout)
	})

	t.Run("should_narrate_each_stage_with_v", func(t *testing.T) {
		out := filepath.Join(t.TempDir(), "asyncapi.yaml")

		stdout, stderr, err := execute(
			t, BuildInfo{}, "generate", dir, "-o", out, "-v",
		)

		require.NoError(t, err)
		assert.Equal(t, "wrote "+out+" (1 catalog(s))\n", stdout,
			"verbosity must not leak into stdout")
		for _, stage := range []string{
			"config:", "load:", "catalogs:", "harness:", "merge:", "encode:", "write:",
		} {
			assert.Contains(t, stderr, " "+stage+" ", "stage %s must be narrated", stage)
		}
		assert.NotContains(t, stderr, "recap", "the recap is a -vv detail")
	})

	t.Run("should_add_internals_and_a_recap_with_vv", func(t *testing.T) {
		out := filepath.Join(t.TempDir(), "asyncapi.yaml")

		_, stderr, err := execute(
			t, BuildInfo{}, "generate", dir, "-o", out, "-vv",
		)

		require.NoError(t, err)
		assert.Contains(t, stderr, "refs:", "combinator refs are a debug detail")
		assert.Contains(t, stderr, "descriptions:")
		assert.Contains(t, stderr, "asyncgo: recap\n")
		assert.Contains(t, stderr, "  total")
	})

	t.Run("should_not_change_the_artifact", func(t *testing.T) {
		quiet := filepath.Join(t.TempDir(), "quiet.yaml")
		verbose := filepath.Join(t.TempDir(), "verbose.yaml")

		_, _, err := execute(t, BuildInfo{}, "generate", dir, "-o", quiet)
		require.NoError(t, err)
		_, _, err = execute(t, BuildInfo{}, "generate", dir, "-o", verbose, "-vv")
		require.NoError(t, err)

		wantQuiet, err := os.ReadFile(quiet)
		require.NoError(t, err)
		wantVerbose, err := os.ReadFile(verbose)
		require.NoError(t, err)
		assert.Equal(t, wantQuiet, wantVerbose,
			"how loudly the command talks must not affect what it writes")
	})
}

// TestCheckVerbose pins the drift report: the message and exit code stay exactly
// as they were, and -v adds one finding per stale and missing key.
func TestCheckVerbose(t *testing.T) {
	t.Run("should_report_what_differs_with_v", func(t *testing.T) {
		artifact := generateTo(t)
		drift(t, artifact)

		stdout, stderr, err := execute(
			t, BuildInfo{}, "check", simpleFixture(t), "-o", artifact, "-v",
		)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrOutOfDate)
		assert.Equal(t, 3, ExitCode(err), "a stale artifact keeps its own exit code")
		assert.Empty(t, stdout)
		assert.Contains(t, stderr, artifact+": channels.order-renamed: stale")
		assert.Contains(t, stderr, artifact+": channels.order-placed: missing")
		assert.Contains(t, stderr, artifact+" is out of date")
	})

	t.Run("should_report_nothing_without_the_flag", func(t *testing.T) {
		artifact := generateTo(t)
		drift(t, artifact)

		_, stderr, err := execute(
			t, BuildInfo{}, "check", simpleFixture(t), "-o", artifact,
		)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrOutOfDate)
		assert.Empty(t, stderr, "the default output must not change for CI parsers")
	})

	t.Run("should_fall_back_to_a_byte_offset_for_an_unparsable_artifact", func(t *testing.T) {
		artifact := generateTo(t)
		require.NoError(t, os.WriteFile(artifact, []byte("not: [valid\n"), 0o644))

		_, stderr, err := execute(
			t, BuildInfo{}, "check", simpleFixture(t), "-o", artifact, "-v",
		)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrOutOfDate)
		assert.Regexp(t, `byte \d+: differs from what generate would write`, stderr)
	})

	t.Run("should_report_matching_artifact_as_up_to_date", func(t *testing.T) {
		artifact := generateTo(t)

		stdout, stderr, err := execute(
			t, BuildInfo{}, "check", simpleFixture(t), "-o", artifact, "-v",
		)

		require.NoError(t, err)
		assert.Equal(t, artifact+" is up to date\n", stdout)
		assert.NotContains(t, stderr, "stale")
	})
}
