// Command asyncgo generates and checks AsyncAPI specification documents from
// Go code. The command tree lives in internal/cli and is built with Cobra;
// this package only carries the build metadata stamped by the release pipeline.
package main

import (
	"fmt"
	"os"

	"github.com/RubenRibGarcia/asyncgo/internal/cli"
)

// Build metadata. The release workflow stamps these via
// -ldflags "-X main.version=vX.Y.Z -X main.commit=<sha> -X main.date=<rfc3339>".
// Untagged local builds report "devel".
var (
	version = "devel"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	bi := cli.BuildInfo{Version: version, Commit: commit, Date: date}
	if err := cli.Execute(os.Args[1:], bi); err != nil {
		fmt.Fprintln(os.Stderr, "asyncgo:", err)
		os.Exit(1)
	}
}
