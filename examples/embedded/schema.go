// Package orders contains the domain types and the AsyncAPI catalog for the
// example orders service.
package simple

type BaseSchema struct {
	// Unique identifier for the order
	ID string `json:"id" asyncapi:"required"`
}

// OrderPlaced is the message emitted when an order is placed.
type OrderPlaced struct {
	BaseSchema
	Amount float64 `json:"amount" asyncapi:"required"`
	// Optional note from the customer
	Note string `json:"note"`
}
