// Package reply is a discovery fixture that declares a request/reply catalog: a
// request channel whose send operation replies to a runtime-expression address,
// and a reply channel carrying the acceptance message. It exercises
// components.replies and components.replyAddresses end to end.
package reply

// OrderPlaced is the request message published when an order is placed.
type OrderPlaced struct {
	OrderID string `json:"order_id" asyncapi:"required"`
}

// OrderAccepted is the reply message acknowledging the order.
type OrderAccepted struct {
	OrderID string `json:"order_id" asyncapi:"required"`
}
