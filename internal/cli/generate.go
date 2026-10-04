package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/RubenRibGarcia/asyncgo/internal/discovery"
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

			doc, n, err := discovery.Build(dir)
			if err != nil {
				var catErrs discovery.CatalogErrors
				if errors.As(err, &catErrs) {
					return catErrs
				}
				return fmt.Errorf("generating: %w", err)
			}
			out, err := selected.encode(doc)
			if err != nil {
				return fmt.Errorf("encoding document: %w", err)
			}
			path, err := resolveOutput(dir, output, selected.defaultName)
			if err != nil {
				return fmt.Errorf("generating: %w", err)
			}
			if err := os.WriteFile(path, out, 0o644); err != nil {
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
