//go:generate go run github.com/RubenRibGarcia/asyncgo/cmd/asyncgo generate .

// Command orders is a minimal example of a service whose AsyncAPI document is
// generated with asyncgo.
package main

import (
	"fmt"

	"example.com/orders/orders"
)

func main() {
	fmt.Println(orders.Catalog.Info.Title)
}
