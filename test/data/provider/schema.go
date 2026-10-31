// Package provider is a test fixture demonstrating a custom schema provider: a
// type with custom (de)serialization declares its own wire schema via
// spec.SchemaProvider, overriding reflection-derived derivation. It covers both
// kinds — a named struct (Money) and a named non-struct (OrderID).
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
		Type:     "string",
		Pattern:  `^\d+\.\d{2} [A-Z]{3}$`,
		Examples: []any{"12.34 USD"},
	}
}

// OrderID is a named string whose constraint is declared once instead of being
// inlined at every use site: the underlying kind would derive a bare
// {type: string}.
type OrderID string

// AsyncAPISchema declares the wire schema for the named string; being a
// non-struct provider, it is hoisted under its fully-qualified name and
// referenced via $ref, exactly like Money.
func (OrderID) AsyncAPISchema() *spec.Schema {
	minLen := uint64(10)
	return &spec.Schema{
		Type:      "string",
		Pattern:   `^ord_[0-9]{6}$`,
		MinLength: &minLen,
		Examples:  []any{"ord_000123"},
	}
}

// OrderPlaced is the message emitted when an order is placed; its price field
// is the custom Money type and its order_id field is the named non-struct
// provider OrderID, so both arrive as $refs.
type OrderPlaced struct {
	OrderID OrderID `json:"order_id" asyncapi:"required"`
	Price   Money   `json:"price"`
}
