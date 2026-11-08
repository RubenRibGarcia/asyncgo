package logging

import (
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// stageTiming is one completed stage: what it was called, how long it took, and
// the summary sentence its caller closed it with.
type stageTiming struct {
	name    string
	elapsed time.Duration
	summary string
}

// recorder collects the stages a run completed so the recap can align them. It
// is shared by every handler derived from one New, which is why the handler
// holds a pointer to it.
type recorder struct {
	mu     sync.Mutex
	stages []stageTiming
}

// record appends a completed stage. Order is completion order, which is also
// execution order for the sequential pipeline this measures.
func (r *recorder) record(name string, elapsed time.Duration, summary string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stages = append(r.stages, stageTiming{name: name, elapsed: elapsed, summary: summary})
}

// snapshot copies the recorded stages for rendering.
func (r *recorder) snapshot() []stageTiming {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]stageTiming(nil), r.stages...)
}

// Recap writes the per-stage timing table at Debug. It is a no-op when no stage
// was recorded (the run failed before completing one) or when the level does not
// want it, so a command can call it unconditionally.
//
// The table is aligned to the widest name and the widest duration in this run,
// never to the terminal width, so the same run always produces the same bytes.
// Total is wall-clock from the logger's construction, not the sum of the rows:
// the gap between the two is the pipeline's own overhead, and hiding it would be
// the one thing a timing table must not do.
func Recap(log *slog.Logger) {
	h, ok := log.Handler().(*handler)
	if !ok || !h.admits(slog.LevelDebug) {
		return
	}
	stages := h.rec.snapshot()
	if len(stages) == 0 {
		return
	}

	names := make([]string, 0, len(stages)+1)
	durations := make([]string, 0, len(stages)+1)
	for _, s := range stages {
		names = append(names, s.name)
		durations = append(durations, formatDuration(s.elapsed))
	}
	names = append(names, "total")
	durations = append(durations, formatDuration(time.Since(h.start)))

	nameWidth := maxLen(names)
	durWidth := maxLen(durations)

	var b strings.Builder
	b.WriteString(prefix)
	b.WriteString("recap\n")
	for i, s := range stages {
		b.WriteString("  ")
		b.WriteString(padRight(names[i], nameWidth))
		b.WriteString("  ")
		b.WriteString(padLeft(durations[i], durWidth))
		if s.summary != "" {
			b.WriteString("  ")
			b.WriteString(s.summary)
		}
		b.WriteByte('\n')
	}
	b.WriteString("  ")
	b.WriteString(padRight("total", nameWidth))
	b.WriteString("  ")
	b.WriteString(padLeft(durations[len(durations)-1], durWidth))
	b.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, _ = io.WriteString(h.w, b.String())
}

// maxLen is the longest string in names, used for column widths.
func maxLen(names []string) int {
	longest := 0
	for _, n := range names {
		longest = max(longest, len(n))
	}
	return longest
}

// padRight pads s with spaces to width.
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// padLeft pads s with spaces on the left to width, so durations line up on their
// units.
func padLeft(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return strings.Repeat(" ", width-len(s)) + s
}
