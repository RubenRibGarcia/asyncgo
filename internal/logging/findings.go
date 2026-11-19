package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strings"
	"unicode/utf8"
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
	reportFindings(log, findings, slog.LevelInfo)
}

// ReportLocators writes findings at default verbosity, in the same one-line
// shape as ReportFindings.
//
// `--diff` is an explicit request to explain a failing check, and some members of
// an out-of-date set cannot be explained by a value diff — a missing artifact, an
// orphan file — because there is no second document to compare them against.
// Those keep the finding line rather than a headed block, and this is what makes
// them print without -v alongside the diffs.
func ReportLocators(log *slog.Logger, findings []Finding) {
	reportFindings(log, findings, slog.LevelWarn)
}

// reportFindings writes findings whose gate is the given level: Info for the -v
// findings, Warn (which Quiet admits) for the locators --diff cannot diff.
func reportFindings(log *slog.Logger, findings []Finding, gate slog.Level) {
	if len(findings) == 0 {
		return
	}
	SortFindings(findings)

	h, ok := log.Handler().(*handler)
	if !ok {
		// Unknown handler: degrade to one record per finding rather than dropping
		// them, so a caller that swapped the handler still sees the report. The
		// record level is the gate, so the two reporters keep their difference.
		for _, f := range findings {
			log.Log(context.Background(), gate, f.String())
		}
		return
	}
	if !h.admits(gate) {
		return
	}
	h.writeFindings(findings)
}

// ReportDiff writes one artifact's value diff: a header record naming the
// artifact and counting its changes, then one aligned row per change.
//
// It is the ungated counterpart of ReportFindings — `--diff` is the user asking
// for this detail, so it prints at default verbosity — and it replaces the
// finding shape's repeated "<path>: " prefix with the header: every row is about
// the artifact the header names, so repeating it per row would only crowd the
// keys out. rows arrive in the diff's own order (sorted by key); elided is how
// many changes the caller's cap dropped, and renders as a footer so a truncated
// block says what it is not showing.
func ReportDiff(log *slog.Logger, header string, rows []Finding, elided int) {
	if len(rows) == 0 && elided == 0 {
		return
	}

	h, ok := log.Handler().(*handler)
	if !ok {
		// Unknown handler: degrade to records at a level the default verbosity
		// admits, so a caller that swapped the handler still sees the detail it
		// explicitly asked for.
		log.Warn(header)
		for _, f := range rows {
			log.Warn(f.String())
		}
		return
	}
	h.writeDiff(header, rows, elided)
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

// maxWhereWidth is the width cap for the where column. A key deep enough to blow
// past it is rare, and letting it set the width would push every value in the
// block off to the right.
const maxWhereWidth = 56

// writeDiff renders the header and its aligned rows in one write, so a diff
// block cannot be interleaved with a concurrent record.
//
// Columns are aligned to the widest cell in this block, never to the terminal
// width, so the same diff always produces the same bytes. A block of one row
// (a byte offset, an encoding-only difference) gets no column legend: there is
// nothing to scan, and the row explains itself.
func (h *handler) writeDiff(header string, rows []Finding, elided int) {
	where := make([]string, len(rows))
	what := make([]string, len(rows))
	for i, f := range rows {
		where[i] = f.Where
		if where[i] == "" {
			where[i] = "document"
		}
		what[i] = f.What
	}
	whereWidth := min(max(maxLen(where), len("where")), maxWhereWidth)
	whatWidth := max(maxLen(what), len("change"))

	var b strings.Builder
	// The header goes through the record rendering on purpose: it is the one line
	// of the block that has to look like every other CLI record, because it is
	// what the rows belong to.
	b.WriteString(h.line(slog.Record{Level: slog.LevelWarn, Message: header}))
	if len(rows) > 1 {
		writeDiffRow(&b, "where", "change", "value", whereWidth, whatWidth)
	}
	for i, f := range rows {
		writeDiffRow(&b, where[i], what[i], f.Why, whereWidth, whatWidth)
	}
	if elided > 0 {
		fmt.Fprintf(&b, "  and %d more change(s)\n", elided)
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	_, _ = io.WriteString(h.w, b.String())
}

// writeDiffRow renders one aligned row. A cell wider than its column (a long
// verdict) is written whole rather than cut: the columns are a reading aid, and
// dropping part of a key would defeat the point of the block.
//
// A location wider than the column wraps instead: its path takes one line per
// segment run and the verdict moves to a continuation line, so a fully-qualified
// schema path costs two lines without pushing every other row's columns right.
func writeDiffRow(b *strings.Builder, where, what, value string, whereWidth, whatWidth int) {
	if utf8.RuneCountInString(where) <= whereWidth {
		b.WriteString("  ")
		b.WriteString(padRight(where, whereWidth))
		b.WriteString("  ")
		b.WriteString(padRight(what, whatWidth))
		writeDiffValue(b, value)
		return
	}
	for i, line := range wrapPath(where, whereWidth) {
		if i > 0 {
			b.WriteString("    ")
		} else {
			b.WriteString("  ")
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	// The verdict sits under the wrapped path, clear of both indents, so it reads
	// as this row's outcome rather than as a row of its own.
	b.WriteString("      ")
	b.WriteString(padRight(what, whatWidth))
	writeDiffValue(b, value)
}

// writeDiffValue closes a row with the value, when the change has one.
func writeDiffValue(b *strings.Builder, value string) {
	if value != "" {
		b.WriteString("  ")
		b.WriteString(value)
	}
	b.WriteByte('\n')
}

// wrapPath packs a document path into lines of at most width runes, breaking only
// at segment boundaries so a break never lands inside a key. A segment longer
// than width — a fully-qualified type name inside a bracket, typically — is left
// whole: eliding part of a location would make it uncopyable, and this is the one
// line every other row already paid for.
func wrapPath(path string, width int) []string {
	var (
		lines   []string
		line    strings.Builder
		lineLen int
	)
	for _, seg := range pathSegments(path) {
		segLen := utf8.RuneCountInString(seg)
		if lineLen > 0 && lineLen+segLen > width {
			lines = append(lines, line.String())
			line.Reset()
			lineLen = 0
		}
		line.WriteString(seg)
		lineLen += segLen
	}
	if line.Len() > 0 {
		lines = append(lines, line.String())
	}
	return lines
}

// pathSegments splits a document path at the boundaries its grammar defines: a
// dotted key starts a segment at every top-level '.', a quoted key at every '['.
// Dots inside brackets belong to the key — a Go type name is dotted — so the scan
// tracks bracket depth and the quoting inside it.
func pathSegments(path string) []string {
	var (
		segments []string
		start    int
		depth    int
		quoted   bool
		escaped  bool
	)
	for i, r := range path {
		switch {
		case escaped:
			escaped = false
		case quoted && r == '\\':
			escaped = true
		case quoted:
			if r == '"' {
				quoted = false
			}
		case depth > 0:
			switch r {
			case '"':
				quoted = true
			case ']':
				depth--
			}
		case r == '[':
			if i > start {
				segments = append(segments, path[start:i])
				start = i
			}
			depth++
		case r == '.':
			if i > start {
				segments = append(segments, path[start:i])
				start = i
			}
		}
	}
	return append(segments, path[start:])
}
