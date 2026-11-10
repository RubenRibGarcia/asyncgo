// Package main is the runnable entry point for the simple fixture. It exists so
// the discovery tests can prove a main package does not narrow the document:
// TestBuildIgnoresMainImportGraph compares this module against a copy whose main
// imports nothing, and both must generate the same asyncapi.yaml.
package main

import (
	"fmt"

	"github.com/RubenRibGarcia/asyncgo/test/data/simple"
)

func main() {
	fmt.Println(simple.Catalog.Doc.Info.Title)
}
