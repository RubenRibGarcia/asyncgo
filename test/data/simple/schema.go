// Package simple is a test fixture: a minimal catalog whose message payload is
// derived from a single struct with required and optional fields.
package simple

// OrderPlaced is the message emitted when an order is placed.
type OrderPlaced struct {
	OrderID string  `json:"order_id" asyncapi:"required"`
	Amount  float64 `json:"amount"   asyncapi:"required"`
	// Optional note from the customer
	Note string `json:"note"`
}
