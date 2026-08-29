// Package oneof is a test fixture demonstrating the oneOf combinator: a union
// field tagged asyncapi:"oneOf=A|B" emits a oneOf of $refs, and the generator
// hoists the referenced types into components.schemas automatically.
package oneof

// OrderPlaced is the message emitted when an order is placed.
type OrderPlaced struct {
	OrderID string  `json:"order_id" asyncapi:"required"`
	Amount  float64 `json:"amount"   asyncapi:"required"`
	// Optional note from the customer
	Note string `json:"note"`
}

// OrderCancelled is emitted when an order is cancelled.
type OrderCancelled struct {
	OrderID string `json:"order_id" asyncapi:"required"`
	// Reason for the cancellation
	Reason string `json:"reason"`
}

// OrderEvent is an envelope whose data field is exactly one of the order
// event types.
type OrderEvent struct {
	Data any `json:"data" asyncapi:"required,oneOf=OrderPlaced|OrderCancelled"`
}
