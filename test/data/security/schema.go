// Package security is a discovery fixture that declares every security scheme
// type the generator supports and references them from a server and an
// operation, so components.securitySchemes is exercised end to end.
package security

// OrderPlaced is the message emitted when an order is placed.
type OrderPlaced struct {
	OrderID string  `json:"order_id" asyncapi:"required"`
	Amount  float64 `json:"amount"   asyncapi:"required"`
}
