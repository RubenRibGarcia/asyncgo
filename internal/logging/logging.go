// Package logging renders the asyncgo CLI's diagnostics: one line per record on
// the command's error stream, phrased for somebody reading a terminal.
//
// Records are log/slog records, so a machine-readable rendering later is a
// handler swap rather than a change to every call site. The text handler prints
// the message and nothing else: a record's attributes exist for that future
// rendering, and interpolating them here would turn a sentence into the
// key=value debris this format exists to avoid. The one attribute the text
// rendering does read is the stage name, which it uses as a tag.
package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Level is the CLI verbosity, chosen by the -v count.
type Level int

const (
	// Quiet is the zero value and the default: warnings and errors only. That is
	// exactly what the CLI printed unconditionally before verbosity existed — an
	// unused schema mapping and a failed harness cleanup were never silent — so
	// dropping them would be a regression, not a quiet default.
	Quiet Level = iota
	// Verbose is -v: the stage story of a run.
	Verbose
	// Debug is -vv: internals, the harness's own stderr, and the timing recap.
	Debug
)

const (
	// prefix is what every record starts with, matching the "asyncgo: " prefix
	// main puts on a returned error.
	prefix = "asyncgo: "
	// stageKey is the attribute Step sets; it is the one attribute the text
	// rendering consumes.
	stageKey = "stage"
)

// ANSI escapes. They are only ever emitted when color is enabled for the writer
// (see colorEnabled), so a redirected or buffered run is plain text.
const (
	ansiReset  = "\x1b[0m"
	ansiBold   = "\x1b[1m"
	ansiDim    = "\x1b[2m"
	ansiRed    = "\x1b[31m"
	ansiYellow = "\x1b[33m"
)

// loggerContextKey is unexported so nothing outside this package can collide
// with it.
type loggerContextKey struct{}

// WithLogger returns ctx carrying log.
//
// It exists for the CLI layer only: a command body builds its
// discovery.Options from the context instead of threading a second parameter
// through every helper, while discovery itself always receives the logger as an
// explicit field and never reads it back out of a context.
func WithLogger(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey{}, log)
}

// FromContext returns the logger in ctx, or a discarding one when there is none,
// so a caller needs no nil branch.
func FromContext(ctx context.Context) *slog.Logger {
	if log, ok := ctx.Value(loggerContextKey{}).(*slog.Logger); ok && log != nil {
		return log
	}
	return slog.New(slog.DiscardHandler)
}

// OrQuiet returns log, or a discarding logger when log is nil, so a caller that
// treats the logger as optional needs no branch.
func OrQuiet(log *slog.Logger) *slog.Logger {
	if log == nil {
		return slog.New(slog.DiscardHandler)
	}
	return log
}

// Enabled reports whether log would emit a record at l. It exists so a call site
// can guard expensive per-item formatting without a context, and without
// allocating a message that would be dropped anyway.
func Enabled(log *slog.Logger, l slog.Level) bool {
	h, ok := OrQuiet(log).Handler().(*handler)
	if !ok {
		// An unknown handler is assumed to want the record: the guard only exists to
		// skip work, so guessing wrong costs formatting, not correctness.
		return true
	}
	return h.admits(l)
}

// Writer returns the stream log writes to, for the one case a call site needs to
// hand the same stream to something else: the harness subprocess's stderr is teed
// into it at Debug, so a compiler warning appears live instead of being swallowed.
// A discarded logger yields io.Discard.
func Writer(log *slog.Logger) io.Writer {
	if h, ok := OrQuiet(log).Handler().(*handler); ok {
		return h.w
	}
	return io.Discard
}

// Verbosity maps a -v count to a Level. Counts above Debug clamp rather than
// failing: a repeated flag in a shell alias should not break a build.
func Verbosity(count int) Level {
	switch {
	case count >= int(Debug):
		return Debug
	case count == int(Verbose):
		return Verbose
	default:
		return Quiet
	}
}

// New returns a logger writing CLI lines to w at level. w is typically the
// command's error stream; a nil logger is treated as Quiet by the helpers in
// this package, so a caller carrying an unset logger needs no branch.
func New(w io.Writer, l Level) *slog.Logger {
	return slog.New(&handler{
		mu:    &sync.Mutex{},
		w:     w,
		level: l,
		color: colorEnabled(w),
		start: time.Now(),
		rec:   &recorder{},
	})
}

// Tag returns log with the stage attribute set, without the timing Step adds. Use
// it for detail that belongs to a stage but is not worth a recap row of its own.
func Tag(log *slog.Logger, name string) *slog.Logger {
	return OrQuiet(log).With(slog.String(stageKey, name))
}

// Step marks the start of one stage of a run. The returned logger tags every
// record with the stage name; the returned func closes the stage by logging the
// summary sentence with the measured duration and recording it for the recap.
//
// done takes slog key/value pairs after the summary. The text rendering ignores
// them, but they are the machine rendering's vocabulary: a call site that counts
// something says the number twice on purpose, once in prose and once as an
// attribute (see the package doc). duration_ms is always attached.
//
// A stage that fails closes nothing: its caller returns the error instead of
// calling done, which is why the recap lists only the stages that completed.
// done is idempotent, so a double close cannot record the stage twice.
func Step(log *slog.Logger, name string) (*slog.Logger, func(summary string, attrs ...any)) {
	start := time.Now()
	staged := Tag(log, name)
	var once sync.Once
	done := func(summary string, attrs ...any) {
		once.Do(func() {
			elapsed := time.Since(start)
			if summary != "" {
				all := append(
					[]any{"duration_ms", elapsed.Milliseconds()}, attrs...,
				)
				staged.Info(summary+" in "+formatDuration(elapsed), all...)
			}
			if h, ok := log.Handler().(*handler); ok {
				h.rec.record(name, elapsed, summary)
			}
		})
	}
	return staged, done
}

// formatDuration renders a stage duration the way a human reads it: microseconds
// below a millisecond, and the standard Go units above it (1.204s, 40ms).
func formatDuration(d time.Duration) string {
	return d.Round(time.Microsecond).String()
}

// colorEnabled reports whether w should receive ANSI escapes. Only a real
// character device gets color, so a pipe, a file, or a test's bytes.Buffer never
// does; NO_COLOR and TERM=dumb are honored on top of that.
func colorEnabled(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// floor is the lowest slog level a CLI level admits.
func (l Level) floor() slog.Level {
	switch l {
	case Debug:
		return slog.LevelDebug
	case Verbose:
		return slog.LevelInfo
	default:
		return slog.LevelWarn
	}
}

// handler writes one record per line and drops records below the CLI level.
//
// It is not safe for concurrent use on its own: the mutex is a pointer shared
// between WithAttrs copies, so two goroutines logging through the same logger
// cannot interleave mid-line, and so the recap's rows stay consistent with the
// lines already printed.
type handler struct {
	mu    *sync.Mutex
	w     io.Writer
	level Level
	color bool
	start time.Time
	stage string
	rec   *recorder
}

// Enabled reports whether a record at l survives the CLI level. It is what makes
// a quiet run free: slog evaluates the guard before it formats anything, so a
// call site inside a traversal can skip the work entirely.
func (h *handler) Enabled(_ context.Context, l slog.Level) bool {
	return h.admits(l)
}

// admits is Enabled without a context, for the helpers in this package, which
// have no caller context to hand slog.
func (h *handler) admits(l slog.Level) bool {
	return l >= h.level.floor()
}

// Handle renders the record. The write is a single call under the mutex so a
// line reaches the writer atomically, which matters when the writer is shared
// with a subprocess's stderr or with another goroutine.
func (h *handler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := io.WriteString(h.w, h.line(r))
	return err
}

// line renders "asyncgo: +1.204s load: 42 packages" with the optional escapes.
// The stamp answers "when", the message answers "how long" — the stage's own
// duration is in the sentence, so the two are not the same measurement.
func (h *handler) line(r slog.Record) string {
	var b strings.Builder
	h.writeEscaped(&b, ansiBold)
	b.WriteString(prefix)
	h.writeEscaped(&b, ansiReset)
	if !h.start.IsZero() {
		b.WriteByte('+')
		b.WriteString(strconv.FormatFloat(time.Since(h.start).Seconds(), 'f', 3, 64))
		b.WriteString("s ")
	}
	if h.stage != "" {
		h.writeEscaped(&b, ansiDim)
		b.WriteString(h.stage)
		b.WriteString(": ")
		h.writeEscaped(&b, ansiReset)
	}
	switch {
	case r.Level >= slog.LevelError:
		h.writeEscaped(&b, ansiRed)
		b.WriteString(r.Message)
		h.writeEscaped(&b, ansiReset)
	case r.Level >= slog.LevelWarn:
		h.writeEscaped(&b, ansiYellow)
		b.WriteString(r.Message)
		h.writeEscaped(&b, ansiReset)
	default:
		b.WriteString(r.Message)
	}
	b.WriteByte('\n')
	return b.String()
}

// writeEscaped writes an escape sequence only when color is on, so the plain
// path never emits a stray reset.
func (h *handler) writeEscaped(b *strings.Builder, escape string) {
	if h.color {
		b.WriteString(escape)
	}
}

// WithAttrs returns a copy carrying the stage name. Other attributes are
// deliberately dropped: the text rendering is a sentence, and attributes are for
// the machine rendering only.
func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := *h
	for _, a := range attrs {
		if a.Key == stageKey {
			next.stage = a.Value.String()
		}
	}
	return &next
}

// WithGroup returns the handler unchanged for the same reason: the text
// rendering has no nesting to put a group in.
func (h *handler) WithGroup(_ string) slog.Handler { return h }
