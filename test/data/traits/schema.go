package traits

// OrderPlaced is the message emitted when an order is placed.
type OrderPlaced struct {
	OrderID string `json:"order_id" asyncapi:"required"`
}

// OrderShipped is the message emitted when an order ships.
type OrderShipped struct {
	OrderID string `json:"order_id" asyncapi:"required"`
}
