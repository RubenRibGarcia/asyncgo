// Package mappings is a discovery fixture for external type mappings: a project
// declares in asyncgo.yaml how a type it does not own derives, a derived alias
// inherits that mapping without a second declaration, and a built-in mapping
// covers a stdlib type that would otherwise derive as an empty object.
package mappings

import (
	"math/big"
	"time"
)

// UUID stands in for a third-party named byte array — gofrs/uuid.UUID is the type
// the feature exists for. Kind derivation would render it as an array of integers;
// asyncgo.yaml maps it to a UUID string instead, which is exactly what a mapping
// can do and a SchemaProvider cannot for a type the project does not own.
type UUID [16]byte

// UUIDAlias is a Go type alias: it introduces no new type, so no alias edge is
// emitted and it derives exactly like UUID.
type UUIDAlias = UUID

// OrderID is a derived alias: it inherits UUID's mapping through the alias edge
// discovery emits, with no declaration of its own.
type OrderID UUID

// LineID is an alias of an alias, so inheritance crosses two declarations.
type LineID OrderID

// OrderRef is a named pointer: the alias edge names UUID, and the mapping is
// reached after the pointer is dereferenced.
type OrderRef *UUID

// Quantity is declared from a predeclared type, so it has no mapping to inherit
// and derives as an integer by kind.
type Quantity int

// OrderPlaced is the message payload. Each field exercises one shape a mapping has
// to survive: inline, slice, slice-of-slice would add nothing, derived alias,
// alias of alias, Go alias, named pointer.
type OrderPlaced struct {
	// ID is the order's identifier, mapped to a UUID string by asyncgo.yaml.
	ID UUID `json:"id" asyncapi:"required"`
	// Refs are the identifiers of related orders, mapped through the element type.
	Refs []UUID `json:"refs"`
	// Customer is a derived alias of UUID, inheriting its mapping.
	Customer OrderID `json:"customer"`
	// Line is an alias of an alias, inheriting the mapping two hops away.
	Line LineID `json:"line"`
	// Saved is a Go type alias for UUID, so reflection already sees UUID.
	Saved UUIDAlias `json:"saved"`
	// Link is a named pointer to UUID.
	Link OrderRef `json:"link"`
	// Quantity has no mapping and derives as an integer by kind.
	Quantity Quantity `json:"quantity"`
	// Total is a *math/big.Int, covered by a built-in mapping: encoding/json marshals
	// it as a bare JSON number, where kind derivation would give an empty object.
	Total *big.Int `json:"total"`
	// CreatedAt is covered by the built-in time.Time mapping.
	CreatedAt time.Time `json:"created_at"`
}
