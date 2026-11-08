package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/RubenRibGarcia/asyncgo/internal/discovery"
	"github.com/RubenRibGarcia/asyncgo/internal/logging"
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
	// Reserve the name in the target's own directory: os.Rename cannot cross
	// filesystems, and the random suffix keeps two concurrent generators from
	// writing through the same temp file.
	tmp, err := os.CreateTemp(filepath.Dir(path), ".asyncgo-write-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// The reservation only held the name; WriteFile fills it. Best effort after a
	// successful rename, and a failure here must not mask the real error.
	_ = tmp.Close()
	defer func() { _ = os.Remove(tmpName) }()

	if err := os.WriteFile(tmpName, data, perm); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// formatSize renders a byte count the way a human reads it (12.3 KiB). It is a
// log detail, so it rounds to one decimal and never says "bytes" for a section
// the user is about to dump into a diff.
func formatSize(n int) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	value := float64(n) / unit
	for _, suffix := range []string{"KiB", "MiB", "GiB"} {
		if value < unit {
			return fmt.Sprintf("%.1f %s", value, suffix)
		}
		value /= unit
	}
	return fmt.Sprintf("%.1f TiB", value)
}

func newGenerateCmd() *cobra.Command {
	var output, format, config string
	cmd := &cobra.Command{
		Use:   "generate [dir]",
		Short: "Write asyncapi.yaml or asyncapi.json for the module rooted at dir",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			selected, err := resolveFormat(format)
			if err != nil {
				return err
			}

			log := logging.FromContext(cmd.Context())

			dir, err := resolveDir(args)
			if err != nil {
				return fmt.Errorf("generating: %w", err)
			}

			doc, n, err := buildDoc(
				cmd,
				dir,
				discovery.Options{Config: config, Log: log},
				"generating",
			)
			if err != nil {
				return err
			}
			_, encodeDone := logging.Step(log, "encode")
			out, err := selected.encode(doc)
			if err != nil {
				return fmt.Errorf("encoding document: %w", err)
			}
			encodeDone(
				format+", "+formatSize(len(out)),
				"format", format,
				"bytes", len(out),
			)
			path, err := resolveOutput(dir, output, selected.defaultName)
			if err != nil {
				return fmt.Errorf("generating: %w", err)
			}
			_, writeDone := logging.Step(log, "write")
			if err := writeFileAtomic(path, out, 0o644); err != nil {
				return fmt.Errorf("writing %s: %w", path, err)
			}
			writeDone(path, "path", path)
			// The recap is printed before the artifact line so the two streams read in
			// the order the run happened: diagnostics, then the result.
			logging.Recap(log)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "wrote %s (%d catalog(s))\n", path, n)
			return nil
		},
	}
	cmd.Flags().
		StringVar(&format, "format", "yaml", "output format: yaml or json")
	cmd.Flags().
		StringVarP(&output, "output", "o", "", "output file or directory (default: <dir>/asyncapi.<format>)")
	cmd.Flags().
		StringVar(&config, "config", "", "schema-mapping config file (default: <dir>/asyncgo.yaml)")
	return cmd
}
