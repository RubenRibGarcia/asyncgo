// Package orders contains the domain types and the AsyncAPI catalog for the
// example orders service.
package orders

// OrderPlaced is the message emitted when an order is placed.
type OrderPlaced struct {
	OrderID string  `json:"order_id" asyncgo:"required"`
	Amount  float64 `json:"amount"   asyncgo:"required"`
	Note    string  `json:"note"     asyncgo:"description=Optional note from the customer"`
}
