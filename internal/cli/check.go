package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/RubenRibGarcia/asyncgo/internal/discovery"
	"github.com/RubenRibGarcia/asyncgo/internal/docdiff"
	"github.com/RubenRibGarcia/asyncgo/internal/logging"
	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/goccy/go-yaml"
	"github.com/spf13/cobra"
)

// ErrOutOfDate marks the one check failure that is not a fault in the module: the
// artifact exists and is valid, it just no longer matches what generate would
// write. Callers — the ExitCode mapping, and a CI script testing errors.Is — can
// tell "regenerate the artifact" from "the build is broken", which a bare exit 1
// conflates.
var ErrOutOfDate = errors.New("document is out of date")

// maxFindingsPerSection caps how many stale or missing keys one section reports.
// A regenerated document can differ in hundreds of keys after a rename, and the
// first ten name the change; the rest would bury the report.
const maxFindingsPerSection = 10

// maxDiffChanges caps how many value changes --diff prints per artifact. Raising
// it above the section cap is deliberate: a value diff is a targeted request,
// and one schema rename can legitimately touch dozens of keys.
const maxDiffChanges = 50

func newCheckCmd() *cobra.Command {
	var output, format, config string
	var showDiff bool
	cmd := &cobra.Command{
		Use:   "check [dir]",
		Short: "Fail if the generated document is out of date",
		Long: "Fail if the generated document is out of date.\n\n" +
			"The document is compared byte for byte against the file `asyncgo generate` would " +
			"write for the same --format, --output and --config flags, so pass the same flags you " +
			"generate with. The default is asyncapi.yaml in dir.\n\n" +
			"A directory destination (an -o value with a trailing separator) checks the whole " +
			"artifact set: one document per catalog, plus any *.asyncapi.<format> file no " +
			"catalog claims. The set produces one verdict, so CI keeps branching on one code.\n\n" +
			"With -v, a stale artifact is reported as one finding per differing section key, so " +
			"CI says what changed instead of only that something did.\n\n" +
			"With --diff, it is reported as the value changes between the committed document and " +
			"the one generate would write: one entry per added, removed or changed key. The " +
			"comparison is over keys and values rather than text, so it reads the same for a " +
			"YAML and a JSON artifact — including a YAML file checked with --format json — and " +
			"it prints without -v.\n\n" +
			"Exit codes:\n" +
			"  0  the artifact(s) are up to date\n" +
			"  1  the check could not run (build error, missing artifact in file mode, I/O failure)\n" +
			"  2  usage error (unknown flag, too many arguments)\n" +
			"  3  the artifact set is out of date — regenerate it",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			selected, err := resolveFormat(format)
			if err != nil {
				return err
			}

			dir, err := resolveDir(args)
			if err != nil {
				return fmt.Errorf("checking: %w", err)
			}

			path, mode, err := resolveOutput(dir, output, selected.defaultName)
			if err != nil {
				return fmt.Errorf("checking: %w", err)
			}

			opts := discovery.Options{
				Config: config,
				Log:    logging.FromContext(cmd.Context()),
			}
			if mode == modeDir {
				return checkPerCatalog(cmd, dir, path, format, selected, opts, showDiff)
			}
			return checkMerged(cmd, dir, path, format, selected, opts, showDiff)
		},
	}
	cmd.Flags().
		StringVar(&format, "format", "yaml", "output format to check: yaml or json")
	cmd.Flags().
		StringVarP(&output, "output", "o", "", "output file, or a directory (trailing separator) for one document per catalog (default: <dir>/asyncapi.<format>)")
	cmd.Flags().
		StringVar(&config, "config", "", "schema-mapping config file (default: <dir>/asyncgo.yaml)")
	cmd.Flags().
		BoolVar(&showDiff, "diff", false, "print the value changes between the committed document and the one generate would write")
	return cmd
}

// checkMerged verifies the module's single merged document against path.
func checkMerged(
	cmd *cobra.Command,
	dir, path, format string,
	selected outputFormat,
	opts discovery.Options,
	showDiff bool,
) error {
	log := logging.FromContext(cmd.Context())

	doc, _, err := buildDoc(cmd, dir, opts, "checking")
	if err != nil {
		return err
	}

	_, encodeDone := logging.Step(log, "encode")
	got, err := selected.encode(doc)
	if err != nil {
		return fmt.Errorf("encoding document: %w", err)
	}
	encodeDone(
		format+", "+formatSize(len(got)),
		"format", format,
		"bytes", len(got),
	)

	checkLog, checkDone := logging.Step(log, "check")
	want, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s not found; run `asyncgo generate` first", path)
		}
		return fmt.Errorf("reading %s: %w", path, err)
	}

	// The error text and exit code stay exactly as they were; the findings are
	// the verbose detail behind it, so CI keeps parsing one familiar line.
	if findings, equal := compareArtifact(path, want, got, format, checkLog); !equal {
		checkLog.Info(path+" is out of date", "path", path)
		if showDiff {
			findings = diffFindings(checkLog, path, want, got, format)
		}
		reportFindings(checkLog, findings, showDiff)
		return fmt.Errorf("%s: %w; run `asyncgo generate`", path, ErrOutOfDate)
	}

	checkDone("byte for byte", "bytes", len(got))
	logging.Recap(log)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s is up to date\n", path)
	return nil
}

// maxNamedArtifacts caps how many filenames the directory-mode verdict lists. A
// regenerated directory can have dozens, and the first few name the change; the
// findings behind -v carry the rest.
const maxNamedArtifacts = 3

// checkPerCatalog verifies every artifact a directory generate would write, plus
// the absence of artifacts no catalog claims.
//
// The whole set produces one verdict and one exit code: a directory is
// regenerated as a unit, so CI needs to know that the set drifted, not to
// triage it file by file. The three failure classes are a stale file, an
// expected file that is absent (a catalog added since the last generate), and an
// orphan — an *.asyncapi.<format> file no catalog claims, which appears when a
// catalog is deleted or renamed.
func checkPerCatalog(
	cmd *cobra.Command,
	dir, outDir, format string,
	selected outputFormat,
	opts discovery.Options,
	showDiff bool,
) error {
	log := logging.FromContext(cmd.Context())

	docs, err := buildDocs(cmd, dir, opts, "checking")
	if err != nil {
		return err
	}
	artifacts, err := encodeArtifacts(docs, outDir, format, selected, log)
	if err != nil {
		return err
	}

	checkLog, checkDone := logging.Step(log, "check")
	expected := make(map[string]bool, len(artifacts))
	findings := make([]logging.Finding, 0, len(artifacts))
	stale := make([]string, 0, len(artifacts))
	for _, a := range artifacts {
		expected[filepath.Base(a.path)] = true
		want, err := os.ReadFile(a.path)
		if err != nil {
			if os.IsNotExist(err) {
				stale = append(stale, filepath.Base(a.path))
				findings = append(findings, logging.Finding{
					Path:  a.path,
					Where: "document",
					What:  "missing",
					Why:   "no committed artifact for a discovered catalog",
				})
				continue
			}
			return fmt.Errorf("reading %s: %w", a.path, err)
		}
		if f, equal := compareArtifact(a.path, want, a.data, format, checkLog); !equal {
			checkLog.Info(a.path+" is out of date", "path", a.path)
			if showDiff {
				findings = append(
					findings,
					diffFindings(checkLog, a.path, want, a.data, format)...,
				)
			} else {
				findings = append(findings, f...)
			}
			stale = append(stale, filepath.Base(a.path))
		}
	}

	orphans, err := artifactOrphans(outDir, format, expected)
	if err != nil {
		return fmt.Errorf("scanning %s: %w", outDir, err)
	}
	for _, path := range orphans {
		stale = append(stale, filepath.Base(path))
		findings = append(findings, logging.Finding{
			Path:  path,
			Where: "document",
			What:  "orphan",
			Why:   "no catalog claims it; delete it, or restore the catalog that produced it",
		})
	}

	if len(stale) > 0 {
		reportFindings(checkLog, findings, showDiff)
		return fmt.Errorf(
			"%s: %w (%s); run `asyncgo generate`",
			outDir, ErrOutOfDate, staleSummary(stale),
		)
	}

	checkDone(
		fmt.Sprintf("%d document(s) match", len(artifacts)),
		"documents", len(artifacts),
	)
	logging.Recap(log)
	_, _ = fmt.Fprintf(
		cmd.OutOrStdout(), "%s: %d document(s) up to date\n", outDir, len(artifacts),
	)
	return nil
}

// artifactOrphans lists the files in outDir that claim to be this format's
// artifacts but that no catalog claims.
//
// It reads the directory rather than globbing it: outDir is a path the user
// typed, and filepath.Glob would read a bracket in a directory name as a pattern.
// The adoption rule is still artifactPattern, so the set check adopts and the set
// generate writes cannot drift apart.
func artifactOrphans(outDir, format string, expected map[string]bool) ([]string, error) {
	entries, err := os.ReadDir(outDir)
	if err != nil {
		return nil, err
	}
	pattern := artifactPattern(format)
	var orphans []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || expected[name] {
			continue
		}
		matched, err := filepath.Match(pattern, name)
		if err != nil {
			return nil, err
		}
		if matched {
			orphans = append(orphans, filepath.Join(outDir, name))
		}
	}
	return orphans, nil
}

// staleSummary names what needs attention without dumping every path. Sorted, so
// the same drift always reads the same way.
func staleSummary(stale []string) string {
	sorted := append([]string(nil), stale...)
	sort.Strings(sorted)
	if len(sorted) <= maxNamedArtifacts {
		return fmt.Sprintf(
			"%d document(s) need attention: %s",
			len(sorted), strings.Join(sorted, ", "),
		)
	}
	return fmt.Sprintf(
		"%d document(s) need attention: %s and %d more",
		len(sorted),
		strings.Join(sorted[:maxNamedArtifacts], ", "),
		len(sorted)-maxNamedArtifacts,
	)
}

// compareArtifact reports whether want is what generate would write, and when it
// is not, why: the findings locate the keys that are stale or missing.
//
// The byte comparison is authoritative — it is what the exit code reflects. The
// findings are a description of that difference, so they are best effort: an
// artifact that does not decode (a hand-edited or truncated file) still has to
// produce a location, which falls back to the first differing byte.
func compareArtifact(
	path string,
	want, got []byte,
	format string,
	log *slog.Logger,
) ([]logging.Finding, bool) {
	if bytes.Equal(want, got) {
		return nil, true
	}

	stale, err := decodeArtifact(want, format)
	if err != nil {
		// Someone edited the artifact by hand, or it was truncated. A byte offset is
		// the only honest location left.
		log.Debug("artifact does not decode: " + err.Error())
		return []logging.Finding{byteOffsetFinding(path, want, got)}, false
	}
	fresh, err := decodeArtifact(got, format)
	if err != nil {
		// The generated document always decodes; failing here is a bug in the
		// encoder, not a stale artifact, so say nothing rather than mislead.
		log.Debug("generated document does not decode: " + err.Error())
		return nil, false
	}
	return sectionFindings(path, stale, fresh), false
}

// decodeArtifact parses an artifact into the typed document. Like
// decodeDocument it tolerates an artifact whose encoding is not the one --format
// selected, because -o picks the file and --format picks the encoding: a YAML
// file checked with `--format json` should still be readable, not fall back to a
// byte offset because the JSON decoder rejected it.
func decodeArtifact(data []byte, format string) (*spec.AsyncAPI, error) {
	first, second := decodeArtifactYAML, decodeArtifactJSON
	if format == "json" {
		first, second = decodeArtifactJSON, decodeArtifactYAML
	}
	doc, err := first(data)
	if err == nil {
		return doc, nil
	}
	if other, otherErr := second(data); otherErr == nil {
		return other, nil
	}
	// Report the failure the requested decoder produced: it names the format the
	// user asked to check.
	return nil, err
}

func decodeArtifactYAML(data []byte) (*spec.AsyncAPI, error) {
	doc := &spec.AsyncAPI{}
	if err := yaml.Unmarshal(data, doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func decodeArtifactJSON(data []byte) (*spec.AsyncAPI, error) {
	doc := &spec.AsyncAPI{}
	if err := json.Unmarshal(data, doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// reportFindings writes the collected findings with the reporter the run asked
// for. --diff is an explicit request, so its entries print at default verbosity;
// the -v findings keep their existing gate.
func reportFindings(log *slog.Logger, findings []logging.Finding, showDiff bool) {
	if showDiff {
		logging.ReportDiff(log, findings)
		return
	}
	logging.ReportFindings(log, findings)
}

// diffFindings explains a stale artifact as the value changes between the
// committed document and the one generate would write, one finding per added,
// removed or changed key.
//
// It decodes both sides as generic documents rather than typed ones: the walk
// needs to see values and keys the typed model does not carry, and the two sides
// may not even share an encoding. An artifact that does not decode at all gets
// the byte offset instead, since that is the only honest location left.
func diffFindings(
	log *slog.Logger,
	path string,
	want, got []byte,
	format string,
) []logging.Finding {
	before, err := decodeDocument(want, format)
	if err != nil {
		log.Debug("artifact does not decode as a document: " + err.Error())
		return []logging.Finding{byteOffsetFinding(path, want, got)}
	}
	after, err := decodeDocument(got, format)
	if err != nil {
		// The generated document always decodes; failing here is a bug in the
		// encoder, not a stale artifact, so say nothing rather than mislead.
		log.Debug("generated document does not decode: " + err.Error())
		return nil
	}

	changes := docdiff.Changes(before, after)
	if len(changes) == 0 {
		// Byte equality failed but every value matched: the encoding differs.
		return []logging.Finding{encodingOnlyFinding(path, want, got)}
	}

	findings := make([]logging.Finding, 0, len(changes)+1)
	for i, c := range changes {
		if i == maxDiffChanges {
			findings = append(findings, logging.Finding{
				Path:  path,
				Where: "document",
				What:  fmt.Sprintf("and %d more change(s)", len(changes)-i),
			})
			break
		}
		findings = append(findings, changeFinding(path, c))
	}
	return findings
}

// changeFinding renders one value change in the finding shape, so a diff entry
// and a drift entry read the same way.
func changeFinding(path string, c docdiff.Change) logging.Finding {
	where := c.Path
	if where == "" {
		where = "document"
	}
	f := logging.Finding{Path: path, Where: where, What: string(c.Kind)}
	switch c.Kind {
	case docdiff.Added:
		f.Why = docdiff.FormatValue(c.After)
	case docdiff.Removed:
		f.Why = docdiff.FormatValue(c.Before)
	case docdiff.Changed:
		f.Why = docdiff.FormatValue(c.Before) + " -> " + docdiff.FormatValue(c.After)
	}
	return f
}

// encodingOnlyFinding explains a byte difference the value comparison cannot
// see: the artifact decodes to the same document generate would write, so only
// the encoding differs — indentation, key order, line endings, or the anchors a
// YAML file may carry. Staying silent here would report a stale artifact with no
// explanation at all.
func encodingOnlyFinding(path string, want, got []byte) logging.Finding {
	return logging.Finding{
		Path:  path,
		Where: fmt.Sprintf("byte %d", firstDifference(want, got)),
		What:  "values match, encoding differs",
		Why:   fmt.Sprintf("artifact is %d bytes, generated is %d", len(want), len(got)),
	}
}

// decodeDocument decodes an artifact into the generic document the value diff
// walks. It tries the run's format first and falls back to the other one: --format
// picks the encoding and -o picks the file, so `-o spec.yaml --format json` is a
// valid invocation, and each parser accepts the other's input often enough that
// the pair recognises both encodings either way round.
func decodeDocument(data []byte, format string) (any, error) {
	first, second := decodeYAML, decodeJSON
	if format == "json" {
		first, second = decodeJSON, decodeYAML
	}
	doc, err := first(data)
	if err == nil {
		return doc, nil
	}
	if other, otherErr := second(data); otherErr == nil {
		return other, nil
	}
	return nil, err
}

// decodeYAML parses a YAML document, resolving anchors and aliases into values.
func decodeYAML(data []byte) (any, error) {
	var doc any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// decodeJSON parses a JSON document. UseNumber keeps an integer's digits, so a
// 64-bit binding value reaches the comparison without rounding through float64.
func decodeJSON(data []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var doc any
	if err := dec.Decode(&doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// byteOffsetFinding locates a difference the section comparison cannot describe.
func byteOffsetFinding(path string, want, got []byte) logging.Finding {
	return logging.Finding{
		Path:  path,
		Where: fmt.Sprintf("byte %d", firstDifference(want, got)),
		What:  "differs from what generate would write",
		Why: fmt.Sprintf(
			"artifact is %d bytes, generated is %d", len(want), len(got),
		),
	}
}

// firstDifference returns the offset of the first byte two documents disagree
// on, or the length of the shorter one when it is a prefix of the other.
func firstDifference(want, got []byte) int {
	offset := 0
	for offset < len(want) && offset < len(got) && want[offset] == got[offset] {
		offset++
	}
	return offset
}

// sectionKeys is one section of the document reduced to the set of keys it
// declares. The document is a known shape, so per-section key comparison answers
// "what changed" without a general structural diff.
type sectionKeys struct {
	name string
	keys map[string]bool
}

// keySet reduces a document map to its key set.
func keySet[V any](m map[string]V) map[string]bool {
	set := make(map[string]bool, len(m))
	for k := range m {
		set[k] = true
	}
	return set
}

// sectionsOf lists the sections check compares, in a fixed order so the report is
// stable.
func sectionsOf(doc *spec.AsyncAPI) []sectionKeys {
	sections := []sectionKeys{
		{name: "channels", keys: keySet(doc.Channels)},
		{name: "operations", keys: keySet(doc.Operations)},
		{name: "servers", keys: keySet(doc.Servers)},
	}
	c := doc.Components
	if c == nil {
		return sections
	}
	return append(
		sections,
		sectionKeys{name: "components.schemas", keys: keySet(c.Schemas)},
		sectionKeys{name: "components.messages", keys: keySet(c.Messages)},
		sectionKeys{name: "components.securitySchemes", keys: keySet(c.SecuritySchemes)},
		sectionKeys{name: "components.parameters", keys: keySet(c.Parameters)},
		sectionKeys{name: "components.channels", keys: keySet(c.Channels)},
		sectionKeys{name: "components.operations", keys: keySet(c.Operations)},
		sectionKeys{name: "components.servers", keys: keySet(c.Servers)},
		sectionKeys{name: "components.correlationIds", keys: keySet(c.CorrelationIDs)},
		sectionKeys{name: "components.operationTraits", keys: keySet(c.OperationTraits)},
		sectionKeys{name: "components.messageTraits", keys: keySet(c.MessageTraits)},
		sectionKeys{name: "components.replies", keys: keySet(c.Replies)},
		sectionKeys{name: "components.replyAddresses", keys: keySet(c.ReplyAddresses)},
	)
}

// sectionFindings compares the artifact's sections with the generated document's
// and reports each stale (in the artifact only) and missing (in the generated
// document only) key.
func sectionFindings(path string, stale, fresh *spec.AsyncAPI) []logging.Finding {
	freshKeys := map[string]map[string]bool{}
	for _, s := range sectionsOf(fresh) {
		freshKeys[s.name] = s.keys
	}

	var findings []logging.Finding
	for _, s := range sectionsOf(stale) {
		wanted := freshKeys[s.name]
		findings = append(
			findings,
			keyFindings(path, s.name, "stale", s.keys, wanted)...,
		)
		findings = append(
			findings,
			keyFindings(path, s.name, "missing", wanted, s.keys)...,
		)
	}
	return findings
}

// keyFindings reports the keys in have that are absent from other, capped at
// maxFindingsPerSection with a trailing count so the cap is visible rather than
// silent.
func keyFindings(
	path, section, what string,
	have, other map[string]bool,
) []logging.Finding {
	var keys []string
	for k := range have {
		if !other[k] {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	findings := make([]logging.Finding, 0, len(keys))
	for i, k := range keys {
		if i == maxFindingsPerSection {
			findings = append(findings, logging.Finding{
				Path:  path,
				Where: section,
				What:  fmt.Sprintf("and %d more", len(keys)-i),
			})
			break
		}
		findings = append(findings, logging.Finding{
			Path:  path,
			Where: section + "." + k,
			What:  what,
		})
	}
	return findings
}
