package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// ErrOutOfDate marks the one check failure that is not a fault in the module: the
// artifact exists and is valid, it just no longer matches what generate would
// write. Callers — the ExitCode mapping, and a CI script testing errors.Is — can
// tell "regenerate the artifact" from "the build is broken", which a bare exit 1
// conflates.
var ErrOutOfDate = errors.New("document is out of date")

func newCheckCmd() *cobra.Command {
	var output, format string
	cmd := &cobra.Command{
		Use:   "check [dir]",
		Short: "Fail if the generated document is out of date",
		Long: "Fail if the generated document is out of date.\n\n" +
			"The document is compared byte for byte against the file `asyncgo generate` would " +
			"write for the same --format and --output flags, so pass the same flags you generate " +
			"with. The default is asyncapi.yaml in dir.\n\n" +
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

			dir, err := resolveDir(args)
			if err != nil {
				return fmt.Errorf("checking: %w", err)
			}

			doc, _, err := buildDoc(cmd, dir, "checking")
			if err != nil {
				return err
			}
			got, err := selected.encode(doc)
			if err != nil {
				return fmt.Errorf("encoding document: %w", err)
			}

			path, err := resolveOutput(dir, output, selected.defaultName)
			if err != nil {
				return fmt.Errorf("checking: %w", err)
			}
			want, err := os.ReadFile(path)
			if err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("%s not found; run `asyncgo generate` first", path)
				}
				return fmt.Errorf("reading %s: %w", path, err)
			}
			if !bytes.Equal(got, want) {
				return fmt.Errorf("%s: %w; run `asyncgo generate`", path, ErrOutOfDate)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s is up to date\n", path)
			return nil
		},
	}
	cmd.Flags().
		StringVar(&format, "format", "yaml", "output format to check: yaml or json")
	cmd.Flags().
		StringVarP(&output, "output", "o", "", "output file or directory (default: <dir>/asyncapi.<format>)")
	return cmd
}
