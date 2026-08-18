// Package orders contains the domain types and the AsyncAPI catalog for the
// example orders service.
package simple

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

// OrderCancelled is emitted when an order is cancelled.
type OrderCancelled struct {
	OrderID string `json:"order_id" asyncapi:"required"`
	// Reason for the cancellation
	Reason string `json:"reason"`
}

// OrderEvent is an envelope whose data field is a union of order event types.
type OrderEvent struct {
	Data any `json:"data" asyncapi:"required,oneOf=OrderPlaced|OrderCancelled"`
}
