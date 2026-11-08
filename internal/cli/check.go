package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sort"

	"github.com/RubenRibGarcia/asyncgo/internal/discovery"
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

func newCheckCmd() *cobra.Command {
	var output, format, config string
	cmd := &cobra.Command{
		Use:   "check [dir]",
		Short: "Fail if the generated document is out of date",
		Long: "Fail if the generated document is out of date.\n\n" +
			"The document is compared byte for byte against the file `asyncgo generate` would " +
			"write for the same --format, --output and --config flags, so pass the same flags you " +
			"generate with. The default is asyncapi.yaml in dir.\n\n" +
			"With -v, a stale artifact is reported as one finding per differing section key, so " +
			"CI says what changed instead of only that something did.\n\n" +
			"Exit codes:\n" +
			"  0  the artifact is up to date\n" +
			"  1  the check could not run (build error, missing artifact, I/O failure)\n" +
			"  2  usage error (unknown flag, too many arguments)\n" +
			"  3  the artifact is out of date — regenerate it",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			selected, err := resolveFormat(format)
			if err != nil {
				return err
			}

			log := logging.FromContext(cmd.Context())

			dir, err := resolveDir(args)
			if err != nil {
				return fmt.Errorf("checking: %w", err)
			}

			doc, _, err := buildDoc(
				cmd,
				dir,
				discovery.Options{Config: config, Log: log},
				"checking",
			)
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

			path, err := resolveOutput(dir, output, selected.defaultName)
			if err != nil {
				return fmt.Errorf("checking: %w", err)
			}

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
				logging.ReportFindings(checkLog, findings)
				return fmt.Errorf("%s: %w; run `asyncgo generate`", path, ErrOutOfDate)
			}

			checkDone("byte for byte", "bytes", len(got))
			logging.Recap(log)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s is up to date\n", path)
			return nil
		},
	}
	cmd.Flags().
		StringVar(&format, "format", "yaml", "output format to check: yaml or json")
	cmd.Flags().
		StringVarP(&output, "output", "o", "", "output file or directory (default: <dir>/asyncapi.<format>)")
	cmd.Flags().
		StringVar(&config, "config", "", "schema-mapping config file (default: <dir>/asyncgo.yaml)")
	return cmd
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

// decodeArtifact parses an artifact with the decoder matching the format it was
// written in.
func decodeArtifact(data []byte, format string) (*spec.AsyncAPI, error) {
	doc := &spec.AsyncAPI{}
	if format == "json" {
		return doc, json.Unmarshal(data, doc)
	}
	return doc, yaml.Unmarshal(data, doc)
}

// byteOffsetFinding locates a difference the section comparison cannot describe.
func byteOffsetFinding(path string, want, got []byte) logging.Finding {
	offset := 0
	for offset < len(want) && offset < len(got) && want[offset] == got[offset] {
		offset++
	}
	return logging.Finding{
		Path:  path,
		Where: fmt.Sprintf("byte %d", offset),
		What:  "differs from what generate would write",
		Why: fmt.Sprintf(
			"artifact is %d bytes, generated is %d", len(want), len(got),
		),
	}
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
