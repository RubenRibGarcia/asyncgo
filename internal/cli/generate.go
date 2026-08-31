package cli

import (
	"fmt"
	"os"

	"github.com/RubenRibGarcia/asyncgo/internal/discovery"
	"github.com/spf13/cobra"
)

func newGenerateCmd() *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "generate [dir]",
		Short: "Write asyncapi.yaml for the module rooted at dir",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := resolveDir(args)
			if err != nil {
				return fmt.Errorf("generating: %w", err)
			}

			doc, n, err := discovery.Build(dir)
			if err != nil {
				return fmt.Errorf("generating: %w", err)
			}
			out, err := doc.YAML()
			if err != nil {
				return fmt.Errorf("encoding document: %w", err)
			}
			path, err := resolveOutput(dir, output)
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
		StringVarP(&output, "output", "o", "", "output file or directory (default: <dir>/asyncapi.yaml)")
	return cmd
}
