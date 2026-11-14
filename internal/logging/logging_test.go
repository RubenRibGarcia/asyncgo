package logging_test

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/RubenRibGarcia/asyncgo/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stampPattern matches the elapsed-time prefix every record carries.
var stampPattern = regexp.MustCompile(`(?m)^(asyncgo: )\+\d+\.\d{3}s `)

// withoutStamp strips the elapsed prefixes so a line can be compared without
// pinning a duration. The stamp's own shape is pinned by
// should_format_the_stamp_with_exactly_three_decimals.
func withoutStamp(s string) string {
	return stampPattern.ReplaceAllString(s, "$1")
}

func TestVerbosity(t *testing.T) {
	tests := []struct {
		name  string
		count int
		want  logging.Level
	}{
		{"should_be_quiet_without_the_flag", 0, logging.Quiet},
		{"should_be_verbose_with_one_v", 1, logging.Verbose},
		{"should_be_debug_with_two_v", 2, logging.Debug},
		{"should_clamp_above_debug", 7, logging.Debug},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, logging.Verbosity(tc.count))
		})
	}
}

// TestQuietPrintsWarningsOnly pins the default level: warnings and errors were
// printed before verbosity existed (an unused schema mapping, a failed harness
// cleanup), so Quiet drops the stage story without dropping them.
func TestQuietPrintsWarningsOnly(t *testing.T) {
	var out bytes.Buffer
	log := logging.New(&out, logging.Quiet)

	log.Info("the stage story")
	log.Debug("an internal detail")
	log.Warn("a stale mapping")
	log.Error("a broken harness")

	got := withoutStamp(out.String())
	assert.NotContains(t, got, "the stage story")
	assert.NotContains(t, got, "an internal detail")
	assert.Contains(t, got, "asyncgo: a stale mapping\n")
	assert.Contains(t, got, "asyncgo: a broken harness\n")
}

func TestStageTagAndStamp(t *testing.T) {
	t.Run("should_tag_records_with_the_stage_name", func(t *testing.T) {
		var out bytes.Buffer

		logging.Tag(logging.New(&out, logging.Verbose), "load").Info("42 packages")

		assert.Equal(t, "asyncgo: load: 42 packages\n", withoutStamp(out.String()))
	})

	t.Run("should_format_the_stamp_with_exactly_three_decimals", func(t *testing.T) {
		var out bytes.Buffer

		logging.New(&out, logging.Verbose).Info("anything")

		assert.Regexp(t, `^asyncgo: \+\d+\.\d{3}s anything\n$`, out.String())
	})

	t.Run("should_not_emit_escapes_for_a_non_terminal_writer", func(t *testing.T) {
		var out bytes.Buffer

		log := logging.New(&out, logging.Verbose)
		logging.Tag(log, "load").Warn("a warning")

		assert.NotContains(t, out.String(), "\x1b[",
			"a buffer is not a terminal, so it must not receive colour")
	})
}

func TestStep(t *testing.T) {
	t.Run("should_log_the_summary_with_its_duration", func(t *testing.T) {
		var out bytes.Buffer

		stage, done := logging.Step(logging.New(&out, logging.Verbose), "load")
		stage.Info("loading packages...")
		done("2 packages")

		got := withoutStamp(out.String())
		assert.Equal(t, "asyncgo: load: loading packages...\n", firstLine(got))
		assert.Regexp(t, `^asyncgo: load: 2 packages in .+\n$`, lastLine(got))
	})

	t.Run("should_not_record_a_stage_twice", func(t *testing.T) {
		var out bytes.Buffer

		_, done := logging.Step(logging.New(&out, logging.Verbose), "load")
		done("2 packages")
		done("2 packages")

		assert.Equal(t, 1, strings.Count(out.String(), "2 packages"))
	})

	t.Run("should_log_nothing_for_an_empty_summary", func(t *testing.T) {
		var out bytes.Buffer

		_, done := logging.Step(logging.New(&out, logging.Verbose), "config")
		done("")

		assert.Empty(t, out.String())
	})
}

func TestRecap(t *testing.T) {
	t.Run("should_print_an_aligned_table_at_debug", func(t *testing.T) {
		var out bytes.Buffer
		log := logging.New(&out, logging.Debug)

		_, loadDone := logging.Step(log, "load")
		loadDone("2 packages")
		_, mergeDone := logging.Step(log, "merge")
		mergeDone("1 channel")
		logging.Recap(log)

		got := out.String()
		assert.Contains(t, got, "asyncgo: recap\n")
		assert.Contains(t, got, "  load")
		assert.Contains(t, got, "2 packages")
		assert.Contains(t, got, "  total")
	})

	t.Run("should_print_nothing_below_debug", func(t *testing.T) {
		var out bytes.Buffer
		log := logging.New(&out, logging.Verbose)

		_, done := logging.Step(log, "load")
		done("2 packages")
		before := out.String()
		logging.Recap(log)

		assert.Equal(t, before, out.String(), "the recap restates the lines above it at -v")
	})

	t.Run("should_print_nothing_when_no_stage_completed", func(t *testing.T) {
		var out bytes.Buffer

		logging.Recap(logging.New(&out, logging.Debug))

		assert.Empty(t, out.String())
	})
}

func TestOrQuiet(t *testing.T) {
	t.Run("should_accept_a_nil_logger", func(t *testing.T) {
		log := logging.OrQuiet(nil)

		require.NotNil(t, log)
		assert.False(t, log.Enabled(t.Context(), slog.LevelWarn),
			"a discarded logger must not emit anything")
	})

	t.Run("should_return_the_logger_it_was_given", func(t *testing.T) {
		log := logging.New(&bytes.Buffer{}, logging.Debug)

		assert.Same(t, log, logging.OrQuiet(log))
	})
}

func TestEnabled(t *testing.T) {
	t.Run("should_report_the_cli_level_for_the_cli_handler", func(t *testing.T) {
		assert.False(
			t,
			logging.Enabled(logging.New(&bytes.Buffer{}, logging.Quiet), slog.LevelInfo),
		)
		assert.True(
			t,
			logging.Enabled(logging.New(&bytes.Buffer{}, logging.Verbose), slog.LevelInfo),
		)
	})
}

func TestFinding(t *testing.T) {
	t.Run("should_render_path_where_what_and_why", func(t *testing.T) {
		f := logging.Finding{
			Path:  "asyncapi.yaml",
			Where: "channels.orderCreated",
			What:  "missing",
			Why:   "declared by catalog.Admin",
		}

		assert.Equal(
			t,
			"asyncapi.yaml: channels.orderCreated: missing (declared by catalog.Admin)",
			f.String(),
		)
	})

	t.Run("should_omit_an_empty_path_and_why", func(t *testing.T) {
		f := logging.Finding{Where: "channels.c", What: "duplicate"}

		assert.Equal(t, "channels.c: duplicate", f.String())
	})
}

func TestReportFindings(t *testing.T) {
	t.Run("should_indent_findings_under_the_record_above", func(t *testing.T) {
		var out bytes.Buffer

		logging.ReportFindings(logging.New(&out, logging.Verbose), []logging.Finding{
			{Path: "asyncapi.yaml", Where: "channels.c", What: "missing"},
			{Path: "asyncapi.yaml", Where: "channels.b", What: "stale"},
		})

		assert.Equal(
			t,
			"  asyncapi.yaml: channels.b: stale\n  asyncapi.yaml: channels.c: missing\n",
			out.String(),
			"findings are sorted by path then location and carry no record prefix",
		)
	})

	t.Run("should_print_nothing_at_quiet", func(t *testing.T) {
		var out bytes.Buffer

		logging.ReportFindings(logging.New(&out, logging.Quiet), []logging.Finding{
			{Where: "channels.c", What: "missing"},
		})

		assert.Empty(t, out.String())
	})

	t.Run("should_print_nothing_for_an_empty_report", func(t *testing.T) {
		var out bytes.Buffer

		logging.ReportFindings(logging.New(&out, logging.Verbose), nil)

		assert.Empty(t, out.String())
	})

	t.Run("should_degrade_to_records_for_an_unknown_handler", func(t *testing.T) {
		var out bytes.Buffer
		plain := slog.New(slog.NewTextHandler(&out, nil))

		logging.ReportFindings(plain, []logging.Finding{{Where: "channels.c", What: "missing"}})
		logging.ReportDiff(plain, []logging.Finding{{Where: "channels.d", What: "added"}})

		assert.Contains(t, out.String(), "channels.c: missing")
		assert.Contains(t, out.String(), "channels.d: added")
	})
}

// TestReportDiff pins the difference from ReportFindings that makes --diff
// usable without -v: the entries print at default verbosity, and each document's
// own order survives the sort so a truncation notice ends its block.
func TestReportDiff(t *testing.T) {
	t.Run("should_print_at_quiet", func(t *testing.T) {
		var out bytes.Buffer

		logging.ReportDiff(logging.New(&out, logging.Quiet), []logging.Finding{
			{
				Path:  "asyncapi.yaml",
				Where: "channels.c",
				What:  "added",
				Why:   `{"address":"orders"}`,
			},
		})

		assert.Equal(
			t,
			`  asyncapi.yaml: channels.c: added ({"address":"orders"})`+"\n",
			out.String(),
			"--diff is an explicit request, so it must not need -v",
		)
	})

	t.Run("should_group_by_path_and_keep_each_documents_order", func(t *testing.T) {
		var out bytes.Buffer

		logging.ReportDiff(logging.New(&out, logging.Quiet), []logging.Finding{
			{Path: "b.yaml", Where: "x", What: "changed"},
			{Path: "a.yaml", Where: "z", What: "changed"},
			{Path: "a.yaml", Where: "a", What: "changed"},
			{Path: "a.yaml", Where: "document", What: "and 1 more change"},
		})

		assert.Equal(
			t,
			"  a.yaml: z: changed\n"+
				"  a.yaml: a: changed\n"+
				"  a.yaml: document: and 1 more change\n"+
				"  b.yaml: x: changed\n",
			out.String(),
		)
	})

	t.Run("should_print_nothing_for_an_empty_report", func(t *testing.T) {
		var out bytes.Buffer

		logging.ReportDiff(logging.New(&out, logging.Quiet), nil)

		assert.Empty(t, out.String())
	})
}

// firstLine and lastLine read the ends of a multi-line capture.
func firstLine(s string) string {
	lines := strings.SplitN(s, "\n", 2)
	return lines[0] + "\n"
}

// lastLine returns the last non-empty line, with its trailing newline.
func lastLine(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	return lines[len(lines)-1] + "\n"
}

func TestWithLoggerAndFromContext(t *testing.T) {
	t.Run("should_return_a_discarding_logger_without_one", func(t *testing.T) {
		log := logging.FromContext(t.Context())

		require.NotNil(t, log)
		assert.False(t, log.Enabled(t.Context(), slog.LevelError))
	})

	t.Run("should_return_the_logger_the_context_carries", func(t *testing.T) {
		want := logging.New(&bytes.Buffer{}, logging.Verbose)

		got := logging.FromContext(logging.WithLogger(t.Context(), want))

		assert.Same(t, want, got)
	})
}

func TestWriter(t *testing.T) {
	t.Run("should_return_the_handlers_stream", func(t *testing.T) {
		var out bytes.Buffer

		w := logging.Writer(logging.New(&out, logging.Verbose))
		_, err := w.Write([]byte("direct"))

		require.NoError(t, err)
		assert.Equal(t, "direct", out.String())
	})

	t.Run("should_discard_for_a_foreign_handler", func(t *testing.T) {
		log := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

		assert.Equal(t, io.Discard, logging.Writer(log))
	})

	t.Run("should_discard_for_a_nil_logger", func(t *testing.T) {
		assert.Equal(t, io.Discard, logging.Writer(nil))
	})
}

func TestEnabledWithForeignHandler(t *testing.T) {
	// A handler this package does not own is assumed to want the record: the guard
	// exists to skip formatting work, not to filter by somebody else's rules.
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	assert.True(t, logging.Enabled(log, slog.LevelDebug))
}

func TestAttributesAreNotPrinted(t *testing.T) {
	t.Run("should_drop_a_non_stage_attribute", func(t *testing.T) {
		var out bytes.Buffer

		logging.New(&out, logging.Verbose).With("packages", 42).Info("loaded")

		assert.Equal(t, "asyncgo: loaded\n", withoutStamp(out.String()))
	})

	t.Run("should_drop_a_group", func(t *testing.T) {
		var out bytes.Buffer

		logging.New(&out, logging.Verbose).WithGroup("load").Info("loaded")

		assert.Equal(t, "asyncgo: loaded\n", withoutStamp(out.String()))
	})
}

// TestColorIsSuppressed pins the three ways a run can be plain text even with a
// file as the writer: NO_COLOR, TERM=dumb, and a writer that is not a terminal.
func TestColorIsSuppressed(t *testing.T) {
	fileWriter := func(t *testing.T) *os.File {
		t.Helper()
		f, err := os.CreateTemp(t.TempDir(), "out")
		require.NoError(t, err)
		t.Cleanup(func() { _ = f.Close() })
		return f
	}

	t.Run("should_honour_no_color", func(t *testing.T) {
		t.Setenv("NO_COLOR", "1")
		f := fileWriter(t)

		logging.Tag(logging.New(f, logging.Verbose), "load").Warn("a warning")

		content, err := os.ReadFile(f.Name())
		require.NoError(t, err)
		assert.NotContains(t, string(content), "\x1b[")
	})

	t.Run("should_honour_term_dumb", func(t *testing.T) {
		t.Setenv("TERM", "dumb")
		f := fileWriter(t)

		logging.Tag(logging.New(f, logging.Verbose), "load").Error("a failure")

		content, err := os.ReadFile(f.Name())
		require.NoError(t, err)
		assert.NotContains(t, string(content), "\x1b[")
	})

	t.Run("should_not_colour_a_regular_file", func(t *testing.T) {
		f := fileWriter(t)

		logging.Tag(logging.New(f, logging.Verbose), "load").Info("anything")

		content, err := os.ReadFile(f.Name())
		require.NoError(t, err)
		assert.Contains(t, string(content), "asyncgo: ", "the line is still written")
		assert.NotContains(t, string(content), "\x1b[")
	})
}
