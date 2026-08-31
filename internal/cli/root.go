// Package cli builds the asyncgo command-line interface on top of Cobra. It
// holds the command tree (generate, check, version) and the small path/version
// helpers shared between commands. The build metadata is injected by the thin
// main package in cmd/asyncgo.
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

func newRootCmd(bi BuildInfo) *cobra.Command {
	root := &cobra.Command{
		Use:   "asyncgo",
		Short: "Generate and check AsyncAPI specifications from Go code",
		// Errors are printed by main with an "asyncgo:" prefix.
		SilenceErrors: true,
		// Runtime errors (e.g. "no AsyncAPI catalogs") should not dump usage.
		SilenceUsage: true,
		Version:      formatVersion(bi),
	}
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
// empty output defaults to <dir>/asyncapi.yaml; a trailing path separator
// writes asyncapi.yaml inside the given directory; any other value is treated
// as the exact output file path (made absolute).
func resolveOutput(dir, output string) (string, error) {
	if output == "" {
		return filepath.Join(dir, "asyncapi.yaml"), nil
	}
	if strings.HasSuffix(output, string(os.PathSeparator)) || strings.HasSuffix(output, "/") {
		return filepath.Join(output, "asyncapi.yaml"), nil
	}
	abs, err := filepath.Abs(output)
	if err != nil {
		return "", fmt.Errorf("resolving output %q: %w", output, err)
	}
	return abs, nil
}
