package cli

import (
	"errors"
	"fmt"

	"github.com/RubenRibGarcia/asyncgo/internal/discovery"
	"github.com/RubenRibGarcia/asyncgo/internal/validation"
	"github.com/spf13/cobra"
)

// newValidateCmd validates the document a module produces against the vendored
// AsyncAPI 3.1.0 JSON Schema. Unlike check it reads no committed artifact: the
// document is built and validated in memory, so a catalog mistake is reported
// as invalid rather than as out of date.
func newValidateCmd() *cobra.Command {
	var ignoreMissingBindings bool
	cmd := &cobra.Command{
		Use:   "validate [dir]",
		Short: "Validate the generated document against the AsyncAPI 3.1.0 schema",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := resolveDir(args)
			if err != nil {
				return fmt.Errorf("validating: %w", err)
			}

			doc, _, err := discovery.Build(dir)
			if err != nil {
				var catErrs discovery.CatalogErrors
				if errors.As(err, &catErrs) {
					return catErrs
				}
				return fmt.Errorf("validating: %w", err)
			}

			opts := validation.Options{IgnoreMissingBindings: ignoreMissingBindings}
			if err := validation.Validate(doc, opts); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(
				cmd.OutOrStdout(),
				"%s produces a valid AsyncAPI 3.1.0 document\n",
				dir,
			)
			return nil
		},
	}
	cmd.Flags().BoolVar(
		&ignoreMissingBindings,
		"ignore-missing-bindings",
		false,
		"ignore binding keys the published 3.1.0 schema omits (mqtt5, mercure)",
	)
	return cmd
}
