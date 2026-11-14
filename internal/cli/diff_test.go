package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RubenRibGarcia/asyncgo/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCheckDiff pins the --diff contract: the value changes print at default
// verbosity (unlike the -v findings), the verdict and exit code are untouched,
// and one walker covers both encodings — including a YAML artifact checked with
// --format json, which -o explicitly allows.
func TestCheckDiff(t *testing.T) {
	t.Run("should_report_added_and_removed_keys_without_v", func(t *testing.T) {
		artifact := generateTo(t)
		drift(t, artifact)

		stdout, stderr, err := execute(
			t, BuildInfo{}, "check", simpleFixture(t), "-o", artifact, "--diff",
		)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrOutOfDate)
		assert.Equal(t, 3, ExitCode(err), "the diff is detail, not a new outcome")
		assert.Empty(t, stdout, "stdout stays reserved for the verdict")
		assert.Contains(t, stderr, artifact+": channels.order-placed: added (")
		assert.Contains(t, stderr, artifact+": channels.order-renamed: removed (")
	})

	t.Run("should_report_a_nested_value_change", func(t *testing.T) {
		artifact := generateTo(t)
		body := readArtifact(t, artifact)
		changed := strings.Replace(body, "address: order-placed", "address: orders", 1)
		require.NotEqual(t, body, changed)
		writeArtifact(t, artifact, changed)

		_, stderr, err := execute(
			t, BuildInfo{}, "check", simpleFixture(t), "-o", artifact, "--diff",
		)

		require.Error(t, err)
		assert.Contains(
			t,
			stderr,
			artifact+`: channels.order-placed.address: changed ("orders" -> "order-placed")`,
		)
	})

	t.Run("should_replace_the_section_findings_at_v", func(t *testing.T) {
		artifact := generateTo(t)
		drift(t, artifact)

		_, stderr, err := execute(
			t, BuildInfo{}, "check", simpleFixture(t), "-o", artifact, "--diff", "-v",
		)

		require.Error(t, err)
		assert.Contains(t, stderr, "channels.order-placed: added")
		assert.NotContains(t, stderr, ": stale", "the diff explains the drift once")
	})

	t.Run("should_print_nothing_when_the_artifact_is_up_to_date", func(t *testing.T) {
		artifact := generateTo(t)

		stdout, stderr, err := execute(
			t, BuildInfo{}, "check", simpleFixture(t), "-o", artifact, "--diff",
		)

		require.NoError(t, err)
		assert.Equal(t, artifact+" is up to date\n", stdout)
		assert.Empty(t, stderr)
	})

	t.Run("should_explain_a_formatting_only_difference", func(t *testing.T) {
		// Every value still matches, so a silent report would leave a failing
		// check unexplained.
		artifact := generateTo(t)
		body := readArtifact(t, artifact)
		writeArtifact(t, artifact, indentAll(body))

		_, stderr, err := execute(
			t, BuildInfo{}, "check", simpleFixture(t), "-o", artifact, "--diff",
		)

		require.Error(t, err)
		assert.Contains(t, stderr, "values match, encoding differs")
	})

	t.Run("should_fall_back_to_a_byte_offset_for_an_unparsable_artifact", func(t *testing.T) {
		artifact := generateTo(t)
		writeArtifact(t, artifact, "not: [valid\n")

		_, stderr, err := execute(
			t, BuildInfo{}, "check", simpleFixture(t), "-o", artifact, "--diff",
		)

		require.Error(t, err)
		assert.Regexp(t, `byte \d+: differs from what generate would write`, stderr)
	})

	t.Run("should_diff_a_json_artifact", func(t *testing.T) {
		dir := simpleFixture(t)
		artifact := filepath.Join(t.TempDir(), "asyncapi.json")
		_, _, err := execute(
			t, BuildInfo{}, "generate", dir, "--format", "json", "-o", artifact,
		)
		require.NoError(t, err)
		drift(t, artifact)

		_, stderr, err := execute(
			t, BuildInfo{}, "check", dir, "--format", "json", "-o", artifact, "--diff",
		)

		require.Error(t, err)
		assert.Contains(t, stderr, artifact+": channels.order-placed: added (")
		assert.Contains(t, stderr, artifact+": channels.order-renamed: removed (")
	})

	t.Run("should_diff_a_yaml_artifact_checked_as_json", func(t *testing.T) {
		// -o names the file and --format picks the encoding, so this pair is a
		// valid invocation: the artifact is YAML and the generated side is JSON.
		artifact := generateTo(t)
		drift(t, artifact)

		_, stderr, err := execute(
			t,
			BuildInfo{},
			"check",
			simpleFixture(t),
			"--format",
			"json",
			"-o",
			artifact,
			"--diff",
		)

		require.Error(t, err)
		assert.Contains(t, stderr, artifact+": channels.order-placed: added (")
		assert.Contains(t, stderr, artifact+": channels.order-renamed: removed (")
	})
}

// TestCheckPerCatalogDiff pins directory mode: a stale artifact is explained by
// its value changes, while the missing and orphan cases stay one line each —
// there is no second document to compare them against.
func TestCheckPerCatalogDiff(t *testing.T) {
	t.Setenv("GOWORK", "off")

	dir := copySimpleFixture(t)
	addSecondCatalog(t, dir, "")
	outDir := t.TempDir()
	_, _, err := execute(t, BuildInfo{}, "generate", dir, "-o", outDir+"/")
	require.NoError(t, err)

	stale := filepath.Join(outDir, "simple.Catalog.asyncapi.yaml")
	drift(t, stale)
	require.NoError(t, os.Remove(filepath.Join(outDir, "extra.Catalog.asyncapi.yaml")))
	writeArtifact(t, filepath.Join(outDir, "deleted.asyncapi.yaml"), "asyncapi: 3.1.0\n")

	_, stderr, err := execute(t, BuildInfo{}, "check", dir, "-o", outDir+"/", "--diff")

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrOutOfDate)
	assert.Equal(t, 3, ExitCode(err))
	assert.Contains(t, stderr, stale+": channels.order-placed: added (")
	assert.Contains(t, stderr, "deleted.asyncapi.yaml: document: orphan")
	assert.Contains(t, stderr, "extra.Catalog.asyncapi.yaml: document: missing")
}

// TestDiffFindings covers what a CLI run cannot cheaply produce: a diff larger
// than the cap, and a document whose root changes shape.
func TestDiffFindings(t *testing.T) {
	log := logging.New(&bytes.Buffer{}, logging.Quiet)

	t.Run("should_cap_the_report_and_count_the_rest", func(t *testing.T) {
		var before strings.Builder
		before.WriteString("channels:\n")
		for i := range 60 {
			fmt.Fprintf(&before, "  k%02d: {}\n", i)
		}

		findings := diffFindings(
			log, "asyncapi.yaml", []byte(before.String()), []byte("channels: {}\n"), "yaml",
		)

		require.Len(t, findings, maxDiffChanges+1)
		assert.Equal(t, "channels.k00", findings[0].Where)
		assert.Equal(t, "removed", findings[0].What)
		assert.Equal(t, "and 10 more change(s)", findings[maxDiffChanges].What)
		assert.Equal(t, maxDiffChanges, len(findings)-1)
	})

	t.Run("should_name_the_root_when_the_document_shape_changes", func(t *testing.T) {
		findings := diffFindings(log, "asyncapi.yaml", []byte("[]\n"), []byte("{}\n"), "yaml")

		require.Len(t, findings, 1)
		assert.Equal(t, "document", findings[0].Where)
		assert.Equal(t, "changed", findings[0].What)
	})
}

// indentAll prefixes every line with two spaces, which leaves the YAML document
// semantically identical but changes its bytes.
func indentAll(body string) string {
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = "  " + line
		}
	}
	return strings.Join(lines, "\n")
}
