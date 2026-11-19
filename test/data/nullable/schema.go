// Package nullable is a discovery fixture for nilable derivation: a pointer,
// slice, map, interface, chan or func can hold JSON null, so a position of one of
// those types derives `oneOf: [<schema>, {type: "null"}]` instead of a schema
// that rejects the null encoding/json produces.
package nullable

import "time"

// Customer is referenced from a nullable pointer field, so its $ref sits inside
// the oneOf rather than beside a keyword JSON Schema would ignore.
type Customer struct {
	// ID identifies the customer.
	ID string `json:"id" asyncapi:"required"`
}

// Refund is one entry of a nullable array of pointers.
type Refund struct {
	// Amount is the refunded amount.
	Amount float64 `json:"amount"`
}

// OrderUpdated is the message payload. It mixes the shapes a nilable position
// takes: a plain pointer, a pointer to a hoisted struct, a pointer carrying tag
// keywords, a nilable slice and map, and an already-unconstrained field.
type OrderUpdated struct {
	// OrderID is always present and never null, so it carries no null branch.
	OrderID string `json:"order_id" asyncapi:"required"`
	// Customer is required but nullable: the key must be there, the value may be
	// null.
	Customer *Customer `json:"customer" asyncapi:"required"`
	// Note is an optional comment; omitempty changes what asyncgo writes, not
	// what a consumer may send.
	Note *string `json:"note,omitempty"`
	// Channel records how the order was placed. Its enum lands on the payload
	// branch, so the field still accepts null.
	Channel *string `json:"channel" asyncapi:"enum=web,mobile"`
	// Tags is a nilable slice.
	Tags []string `json:"tags"`
	// Attributes is a nilable map.
	Attributes map[string]string `json:"attributes"`
	// Refunds is a nilable array whose entries may themselves be nil.
	Refunds []*Refund `json:"refunds"`
	// ShipBy is a mapped time.Time behind a pointer, so the mapping's shape
	// becomes the payload branch.
	ShipBy *time.Time `json:"ship_by"`
	// Metadata is unconstrained, so it already admits null and is left alone.
	Metadata any `json:"metadata"`
}
