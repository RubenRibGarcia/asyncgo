// Command asyncgo generates and checks AsyncAPI specification documents from
// Go code.
//
// Usage:
//
//	asyncgo generate [dir]   write asyncapi.yaml for the module rooted at dir
//	asyncgo check [dir]      fail if asyncapi.yaml is out of date
//	asyncgo version          print the asyncgo version
package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/RubenRibGarcia/asyncgo/internal/discovery"
	"github.com/RubenRibGarcia/asyncgo/spec"
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
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "asyncgo:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: asyncgo <generate|check|version> [dir]")
	}
	switch args[0] {
	case "generate":
		return generate(args[1:])
	case "check":
		return check(args[1:])
	case "version":
		printVersion()
		return nil
	default:
		return fmt.Errorf("unknown command %q (want generate, check, or version)", args[0])
	}
}

// resolveVersion returns the build version, falling back to the module version
// when installed via `go install ...@vX.Y.Z` (where -ldflags are not applied).
func resolveVersion() string {
	if version != "devel" {
		return version
	}
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" &&
		bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	return "devel"
}

func printVersion() {
	v := resolveVersion()
	if commit != "unknown" {
		v += fmt.Sprintf(" (commit %s, built %s)", commit, date)
	}
	fmt.Printf("asyncgo %s\n", v)
}

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

// buildDocument derives the document from the catalogs reachable from main. It
// returns the document and the number of catalogs found.
func buildDocument(dir string) (*spec.AsyncAPI, int, error) {
	return discovery.Build(dir)
}

func generate(args []string) error {
	dir, err := resolveDir(args)
	if err != nil {
		return fmt.Errorf("generating: %w", err)
	}

	doc, n, err := buildDocument(dir)
	if err != nil {
		return fmt.Errorf("generating: %w", err)
	}
	out, err := doc.YAML()
	if err != nil {
		return fmt.Errorf("encoding document: %w", err)
	}
	path := filepath.Join(dir, "asyncapi.yaml")
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	fmt.Printf("wrote %s (%d catalog(s))\n", path, n)
	return nil
}

func check(args []string) error {
	dir, err := resolveDir(args)
	if err != nil {
		return fmt.Errorf("checking: %w", err)
	}

	doc, _, err := buildDocument(dir)
	if err != nil {
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
	fmt.Printf("%s is up to date\n", path)
	return nil
}
