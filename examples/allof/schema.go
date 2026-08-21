// Package allof demonstrates the allOf combinator: a named embedded struct is
// composed via allOf when its field is tagged asyncapi:"allOf".
package allof

// BaseSchema carries fields shared by all order events.
type BaseSchema struct {
	// Unique identifier for the order
	ID string `json:"id" asyncapi:"required"`
}

// OrderPlaced is the message emitted when an order is placed.
type OrderPlaced struct {
	BaseSchema `        asyncapi:"allOf"`
	Amount     float64 `asyncapi:"required" json:"amount"`
	// Optional note from the customer
	Note string `json:"note"`
}
