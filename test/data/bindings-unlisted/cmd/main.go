// Package main is the runnable entry point for the bindings-unlisted fixture. It
// exists so the discovery reachability walk has a main package to start from.
package main

import (
	"fmt"

	unlisted "github.com/RubenRibGarcia/asyncgo/test/data/bindings-unlisted"
)

func main() {
	fmt.Println(unlisted.Catalog.Doc.Info.Title)
}
