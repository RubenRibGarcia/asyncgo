// Package provider is a test fixture demonstrating a custom schema provider: a
// type with custom (de)serialization declares its own wire schema via
// spec.SchemaProvider, overriding reflection-derived derivation.
package provider

import "github.com/RubenRibGarcia/asyncgo/spec"

// Money is serialized on the wire as a single "12.34 USD" string rather than
// an object, so it declares a custom schema.
type Money struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

// AsyncAPISchema declares the wire schema in place of the struct fields.
func (Money) AsyncAPISchema() *spec.Schema {
	return &spec.Schema{
		Type:    "string",
		Pattern: `^\d+\.\d{2} [A-Z]{3}$`,
		Example: "12.34 USD",
	}
}

// OrderPlaced is the message emitted when an order is placed; its price field
// is the custom Money type.
type OrderPlaced struct {
	OrderID string `json:"order_id" asyncapi:"required"`
	Price   Money  `json:"price"`
}
