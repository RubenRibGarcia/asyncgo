package cli

import (
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
)

// BuildInfo carries the build metadata stamped by the release pipeline via
// -ldflags "-X main.version=... -X main.commit=... -X main.date=...". Untagged
// local builds use the defaults in cmd/asyncgo/main.go.
type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

// resolveVersion returns the build version, falling back to the module version
// when installed via `go install ...@vX.Y.Z` (where -ldflags are not applied).
func resolveVersion(injected string) string {
	if injected != "" && injected != "devel" {
		return injected
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" &&
		info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "devel"
}

// formatVersion renders the version string (without the "asyncgo " prefix),
// appending the commit and build date metadata that the build knows. Each part is
// omitted rather than rendered empty: a `go build` without ldflags supplies a
// commit but no date, and "(commit abc1234, built )" is worse than "(commit
// abc1234)". A commit of "unknown" is the toolchain's placeholder for "not
// injected" and is treated as absent.
func formatVersion(bi BuildInfo) string {
	v := resolveVersion(bi.Version)

	var parts []string
	if bi.Commit != "" && bi.Commit != "unknown" {
		parts = append(parts, "commit "+bi.Commit)
	}
	if bi.Date != "" {
		parts = append(parts, "built "+bi.Date)
	}
	if len(parts) > 0 {
		v += " (" + strings.Join(parts, ", ") + ")"
	}
	return v
}

func newVersionCmd(bi BuildInfo) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the asyncgo version",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "asyncgo %s\n", formatVersion(bi))
			return nil
		},
	}
}
