// Package cli builds the asyncgo command-line interface on top of Cobra. It
// holds the command tree (generate, check, version) and the small path/version
// helpers shared between commands. The build metadata is injected by the thin
// main package in cmd/asyncgo.
package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/RubenRibGarcia/asyncgo/internal/discovery"
	"github.com/RubenRibGarcia/asyncgo/internal/logging"
	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/spf13/cobra"
)

// Execute builds the command tree and runs it with the given arguments, writing
// to the process stdout/stderr. It returns the command's error so the caller
// can decide how to report it.
func Execute(args []string, bi BuildInfo) error {
	root := newRootCmd(bi)
	root.SetArgs(args)
	return root.Execute()
}

// ExitCode maps a command error to the process exit status. Zero is success, 1 is
// any failure the command could not classify, and ErrOutOfDate is 3 — a stale
// artifact is a different outcome from a broken one, and CI needs to tell them
// apart. Cobra has already claimed 2 for usage errors (an unknown flag, too many
// arguments), so the codes stay disjoint.
func ExitCode(err error) int {
	switch {
	case err == nil:
		return 0
	case errors.Is(err, ErrOutOfDate):
		return 3
	default:
		return 1
	}
}

func newRootCmd(bi BuildInfo) *cobra.Command {
	var verbosity int
	root := &cobra.Command{
		Use:   "asyncgo",
		Short: "Generate and check AsyncAPI specifications from Go code",
		// Errors are printed by main with an "asyncgo:" prefix.
		SilenceErrors: true,
		// Runtime errors (e.g. "no AsyncAPI catalogs") should not dump usage.
		SilenceUsage: true,
		Version:      formatVersion(bi),
		// The logger is built here, once per command tree, so every subcommand
		// reports through the streams Cobra was given — which is what makes the
		// diagnostics capturable in tests and redirectable by a caller.
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			log := logging.New(cmd.ErrOrStderr(), logging.Verbosity(verbosity))
			cmd.SetContext(logging.WithLogger(cmd.Context(), log))
		},
	}
	root.PersistentFlags().CountVarP(&verbosity, "verbose", "v",
		"log what the command is doing to stderr (repeat for more detail)")
	root.SetVersionTemplate("asyncgo {{.Version}}\n")
	root.AddCommand(newGenerateCmd(), newCheckCmd(), newVersionCmd(bi))
	return root
}

// resolveDir resolves the module directory from the positional argument,
// defaulting to the current directory.
func resolveDir(args []string) (string, error) {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolving directory %q: %w", dir, err)
	}
	return abs, nil
}

// outputMode is the document model a destination selects. It is derived from
// the shape of the -o value and nothing else, so generate and check cannot
// disagree about how many documents a run covers.
type outputMode int

const (
	// modeFile writes one merged document to the resolved path. This is the
	// default and the behaviour every existing invocation gets.
	modeFile outputMode = iota
	// modeDir writes one document per catalog into the resolved directory, each
	// named from its catalog.
	modeDir
)

// resolveOutput computes the destination of a run and the document model it
// selects. An empty output defaults to <dir>/<defaultName> and one merged
// document; a trailing path separator means that directory and one document per
// catalog; any other value is the exact output file path (made absolute), again
// one merged document.
//
// The trailing separator is the whole distinction, so it is resolved in one
// place and reported back to both commands. -o names a destination and --format
// picks the encoding: an explicit file path is honored verbatim, whatever its
// extension.
func resolveOutput(dir, output, defaultName string) (string, outputMode, error) {
	if output == "" {
		return filepath.Join(dir, defaultName), modeFile, nil
	}
	abs, err := filepath.Abs(output)
	if err != nil {
		return "", modeFile, fmt.Errorf("resolving output %q: %w", output, err)
	}
	if strings.HasSuffix(output, string(os.PathSeparator)) || strings.HasSuffix(output, "/") {
		return abs, modeDir, nil
	}
	return abs, modeFile, nil
}

// artifactName is the filename one catalog's document is written as in
// directory mode. The "asyncapi" infix is load-bearing: it is what lets check
// tell its own artifacts from any other YAML in the directory, so an unrelated
// docs/architecture.yaml is never mistaken for a stale document and an artifact
// whose catalog was deleted is never silently unverified.
func artifactName(stem, format string) string {
	return stem + ".asyncapi." + format
}

// artifactPattern matches the artifacts of one format: exactly the files check
// adopts, and therefore the only files it can report as orphans.
func artifactPattern(format string) string {
	return "*.asyncapi." + format
}

// buildDoc materializes the catalog rooted at dir into a merged document. opts
// carries the schema-mapping config path and the diagnostics sink, and is
// forwarded verbatim, so the error naming a config file the user chose is the
// path they typed. A catalog validation failure is returned as is, so the command
// prints the per-catalog report; any other failure is wrapped with verb, which
// names the command in the message ("generating: ...").
func buildDoc(
	cmd *cobra.Command,
	dir string,
	opts discovery.Options,
	verb string,
) (*spec.AsyncAPI, int, error) {
	doc, n, err := discovery.Build(cmd.Context(), dir, opts)
	if err != nil {
		return nil, 0, buildDocError(err, verb)
	}
	return doc, n, nil
}

// buildDocs materializes every catalog rooted at dir into its own document,
// without merging. It is buildDoc's counterpart for directory mode and shares
// its error mapping, so a catalog validation report reads the same either way.
func buildDocs(
	cmd *cobra.Command,
	dir string,
	opts discovery.Options,
	verb string,
) ([]discovery.Document, error) {
	docs, err := discovery.BuildAll(cmd.Context(), dir, opts)
	if err != nil {
		return nil, buildDocError(err, verb)
	}
	return docs, nil
}

// buildDocError maps a discovery failure to what the command should print: a
// catalog validation report is returned as is, and every other failure is
// wrapped with verb, which names the command in the message ("generating: ...").
func buildDocError(err error, verb string) error {
	var catErrs discovery.CatalogErrors
	if errors.As(err, &catErrs) {
		return catErrs
	}
	return fmt.Errorf("%s: %w", verb, err)
}
