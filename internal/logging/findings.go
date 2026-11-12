package logging

import (
	"io"
	"log/slog"
	"sort"
	"strings"
)

// Finding is one located problem: where it is, what is wrong, and why.
//
// It exists so the three kinds of problem the CLI reports — an out-of-date
// artifact, a mapping that never matched, a document key two catalogs defined —
// share one shape. Rendering it compiler-style ("path: where: what (why)") makes
// it clickable in a terminal, and keeping it a value rather than a formatted
// string is what would let a future annotation renderer consume the same slice.
type Finding struct {
	// Path is the file the user has to edit: the artifact for a drift finding,
	// asyncgo.yaml for an unused mapping. Empty for a conflict in the merged
	// document, which has no single file.
	Path string
	// Where is the location inside the document or config, e.g.
	// "channels.orderCreated" or "components.schemas.OrderV2".
	Where string
	// What is the short verdict: "missing", "stale", "differs", "duplicate",
	// "mapping never used".
	What string
	// Why is an optional parenthetical: who declared it, who it was dropped in
	// favour of.
	Why string
}

// String renders the finding compiler-style, without the leading indent.
func (f Finding) String() string {
	var b strings.Builder
	if f.Path != "" {
		b.WriteString(f.Path)
		b.WriteString(": ")
	}
	b.WriteString(f.Where)
	b.WriteString(": ")
	b.WriteString(f.What)
	if f.Why != "" {
		b.WriteString(" (")
		b.WriteString(f.Why)
		b.WriteString(")")
	}
	return b.String()
}

// SortFindings orders findings by Path then Where, so a report is deterministic
// no matter which map iteration produced it.
func SortFindings(findings []Finding) {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		return findings[i].Where < findings[j].Where
	})
}

// ReportFindings writes findings at Verbose, indented under the record that
// produced them. They are the deliberate exception to "every line is a prefixed
// record": a finding is subordinate detail of the line above it, and the block
// is always contiguous.
func ReportFindings(log *slog.Logger, findings []Finding) {
	if len(findings) == 0 {
		return
	}
	SortFindings(findings)

	h, ok := log.Handler().(*handler)
	if !ok {
		// Unknown handler: degrade to one record per finding rather than dropping
		// them, so a caller that swapped the handler still sees the report.
		for _, f := range findings {
			log.Info(f.String())
		}
		return
	}
	if !h.admits(slog.LevelInfo) {
		return
	}
	h.writeFindings(findings)
}

// ReportDiff writes the value changes behind an out-of-date artifact, indented
// the same way as findings and in the same shape.
//
// It is the ungated counterpart of ReportFindings: `--diff` is the user asking
// for this detail, so the entries print at default verbosity too, whereas drift
// findings are `-v` narration. Entries are grouped by path with a stable sort, so
// each document keeps the order its diff produced (sorted by key) and a
// truncation notice stays at the end of its own block instead of sorting into
// the middle of it.
func ReportDiff(log *slog.Logger, findings []Finding) {
	if len(findings) == 0 {
		return
	}
	sort.SliceStable(findings, func(i, j int) bool {
		return findings[i].Path < findings[j].Path
	})

	h, ok := log.Handler().(*handler)
	if !ok {
		// Unknown handler: degrade to one record per entry, at a level the default
		// verbosity admits, so a caller that swapped the handler still sees the
		// detail it explicitly asked for.
		for _, f := range findings {
			log.Warn(f.String())
		}
		return
	}
	h.writeFindings(findings)
}

// writeFindings renders the indented block in one write, so the block cannot be
// interleaved with a concurrent record.
func (h *handler) writeFindings(findings []Finding) {
	var b strings.Builder
	for _, f := range findings {
		b.WriteString("  ")
		b.WriteString(f.String())
		b.WriteByte('\n')
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	_, _ = io.WriteString(h.w, b.String())
}
