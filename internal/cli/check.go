package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/RubenRibGarcia/asyncgo/internal/discovery"
	"github.com/spf13/cobra"
)

func newCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check [dir]",
		Short: "Fail if asyncapi.yaml is out of date",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := resolveDir(args)
			if err != nil {
				return fmt.Errorf("checking: %w", err)
			}

			doc, _, err := discovery.Build(dir)
			if err != nil {
				var catErrs discovery.CatalogErrors
				if errors.As(err, &catErrs) {
					return catErrs
				}
				return fmt.Errorf("checking: %w", err)
			}
			got, err := doc.YAML()
			if err != nil {
				return fmt.Errorf("encoding document: %w", err)
			}

			path := filepath.Join(dir, "asyncapi.yaml")
			want, err := os.ReadFile(path)
			if err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("%s not found; run `asyncgo generate` first", path)
				}
				return fmt.Errorf("reading %s: %w", path, err)
			}
			if !bytes.Equal(got, want) {
				return fmt.Errorf("%s is out of date; run `asyncgo generate`", path)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s is up to date\n", path)
			return nil
		},
	}
}
