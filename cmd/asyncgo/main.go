// Command asyncgo generates and checks AsyncAPI specification documents from
// Go code.
//
// Usage:
//
//	asyncgo generate [dir]   write asyncapi.yaml for the module rooted at dir
//	asyncgo check [dir]      fail if asyncapi.yaml is out of date
//
// If dir contains an asyncapi.fragment.yaml, it is deep-merged over the
// generated document (see spec.Overlay), so parts of the spec can be authored
// by hand instead of in a Go catalog.
package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/RubenRibGarcia/asyncgo/internal/discovery"
	"github.com/RubenRibGarcia/asyncgo/spec"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "asyncgo:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: asyncgo <generate|check> [dir]")
	}
	switch args[0] {
	case "generate":
		return generate(args[1:])
	case "check":
		return check(args[1:])
	default:
		return fmt.Errorf("unknown command %q (want generate or check)", args[0])
	}
}

func resolveDir(args []string) (string, error) {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}
	return filepath.Abs(dir)
}

// buildDocument derives the document from the catalogs reachable from main and
// applies the fragment file, if present. It returns the document and the number
// of catalogs found.
func buildDocument(dir string) (*spec.AsyncAPI, int, error) {
	cats, err := discovery.Find(dir)
	if err != nil {
		return nil, 0, err
	}
	if len(cats) == 0 {
		return nil, 0, fmt.Errorf("no AsyncAPI catalogs (*spec.AsyncAPI vars) reachable from main in %s", dir)
	}
	docs, err := discovery.Materialize(dir, cats)
	if err != nil {
		return nil, 0, err
	}
	doc := discovery.Merge(docs...)
	doc, err = discovery.ApplyFragment(dir, doc)
	if err != nil {
		return nil, 0, err
	}

	return doc, len(cats), nil
}

func generate(args []string) error {
	dir, err := resolveDir(args)
	if err != nil {
		return err
	}

	doc, n, err := buildDocument(dir)
	if err != nil {
		return err
	}
	out, err := doc.YAML()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "asyncapi.yaml")
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote %s (%d catalog(s))\n", path, n)
	return nil
}

func check(args []string) error {
	dir, err := resolveDir(args)
	if err != nil {
		return err
	}

	doc, _, err := buildDocument(dir)
	if err != nil {
		return err
	}
	got, err := doc.YAML()
	if err != nil {
		return err
	}

	path := filepath.Join(dir, "asyncapi.yaml")
	want, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s not found; run `asyncgo generate` first", path)
		}
		return err
	}
	if !bytes.Equal(got, want) {
		return fmt.Errorf("%s is out of date; run `asyncgo generate`", path)
	}
	fmt.Printf("%s is up to date\n", path)
	return nil
}
