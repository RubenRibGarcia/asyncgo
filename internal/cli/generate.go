package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/spf13/cobra"
)

// outputFormat ties an accepted --format value to the encoder it selects and to
// the filename used when the user did not name an output path.
type outputFormat struct {
	encode      func(*spec.AsyncAPI) ([]byte, error)
	defaultName string
}

// resolveFormat maps a --format value to its encoder. An unknown value is
// rejected here, before any directory is resolved or the discovery harness
// runs, so a typo writes nothing.
func resolveFormat(format string) (outputFormat, error) {
	switch format {
	case "yaml":
		return outputFormat{encode: (*spec.AsyncAPI).YAML, defaultName: "asyncapi.yaml"}, nil
	case "json":
		return outputFormat{encode: (*spec.AsyncAPI).JSONIndent, defaultName: "asyncapi.json"}, nil
	default:
		return outputFormat{}, fmt.Errorf(
			"unknown format %q: want %q or %q",
			format,
			"yaml",
			"json",
		)
	}
}

// writeFileAtomic writes data to path by writing a temporary file in the same
// directory and renaming it into place. The artifact is rewritten on every run,
// so a failure partway through os.WriteFile would truncate the previous document
// and leave a half-written one behind — and a half-written AsyncAPI document is
// still valid YAML for a prefix of itself in many cases, which makes it worse
// than an obvious corruption. The rename is atomic within a directory, so a
// reader sees either the old artifact or the new one.
//
// The temp name must not look like the discovery harness directories
// (_asyncgo-harness-*): those are skipped by load()'s pattern scan, and an
// artifact that the go tool ignores would be a confusing bug. It also has to sit
// in path's own directory, since os.Rename cannot cross filesystems.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)

	tmp, err := os.CreateTemp(dir, ".asyncgo-write-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// Best effort: after a successful rename there is nothing to remove, and a
	// failure here must not mask the real error.
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	// Close before renaming: on some systems the rename is what makes the contents
	// durable, and a pending write must be flushed first.
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func newGenerateCmd() *cobra.Command {
	var output, format string
	cmd := &cobra.Command{
		Use:   "generate [dir]",
		Short: "Write asyncapi.yaml or asyncapi.json for the module rooted at dir",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			selected, err := resolveFormat(format)
			if err != nil {
				return err
			}

			dir, err := resolveDir(args)
			if err != nil {
				return fmt.Errorf("generating: %w", err)
			}

			doc, n, err := buildDoc(cmd, dir, "generating")
			if err != nil {
				return err
			}
			out, err := selected.encode(doc)
			if err != nil {
				return fmt.Errorf("encoding document: %w", err)
			}
			path, err := resolveOutput(dir, output, selected.defaultName)
			if err != nil {
				return fmt.Errorf("generating: %w", err)
			}
			if err := writeFileAtomic(path, out, 0o644); err != nil {
				return fmt.Errorf("writing %s: %w", path, err)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "wrote %s (%d catalog(s))\n", path, n)
			return nil
		},
	}
	cmd.Flags().
		StringVar(&format, "format", "yaml", "output format: yaml or json")
	cmd.Flags().
		StringVarP(&output, "output", "o", "", "output file or directory (default: <dir>/asyncapi.<format>)")
	return cmd
}
