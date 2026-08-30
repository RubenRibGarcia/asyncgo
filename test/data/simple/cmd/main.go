// Package main is the runnable entry point for the simple fixture. It exists
// so the discovery reachability walk has a main package to start from, which
// exercises the from-main reachability path without changing the generated
// document.
package main

import (
	"fmt"

	"github.com/RubenRibGarcia/asyncgo/test/data/simple"
)

func main() {
	fmt.Println(simple.Catalog.Info.Title)
}
