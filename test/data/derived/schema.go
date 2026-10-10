// Package derived is a discovery fixture for the message hoisting rules: a
// payload-derived component key (a MessageOf whose name is deliberately not
// pinned with Name), the JSON-pointer-escaped channel message id and operation
// $ref that key produces, and one hoisted message shared by two channels.
package derived

// OrderPlaced is the message payload both channels carry. Its fully-qualified
// type name is the components.messages key the generator derives for it.
type OrderPlaced struct {
	OrderID string `json:"order_id" asyncapi:"required"`
}
