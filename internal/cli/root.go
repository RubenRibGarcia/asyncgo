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

// resolveOutput computes the destination path for the generated document. An
// empty output defaults to <dir>/<defaultName>; a trailing path separator writes
// defaultName inside the given directory; any other value is treated as the
// exact output file path (made absolute). The default filename carries the
// selected output format (asyncapi.yaml or asyncapi.json); an explicit path is
// honored verbatim, whatever its extension.
func resolveOutput(dir, output, defaultName string) (string, error) {
	if output == "" {
		return filepath.Join(dir, defaultName), nil
	}
	if strings.HasSuffix(output, string(os.PathSeparator)) || strings.HasSuffix(output, "/") {
		return filepath.Join(output, defaultName), nil
	}
	abs, err := filepath.Abs(output)
	if err != nil {
		return "", fmt.Errorf("resolving output %q: %w", output, err)
	}
	return abs, nil
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
		var catErrs discovery.CatalogErrors
		if errors.As(err, &catErrs) {
			return nil, 0, catErrs
		}
		return nil, 0, fmt.Errorf("%s: %w", verb, err)
	}
	return doc, n, nil
}
