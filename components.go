package asyncgo

import (
	"fmt"

	"github.com/RubenRibGarcia/asyncgo/spec"
)

// Component registration.
//
// Eight builders register a component into the document and differ only in which
// map they write and how their errors are worded. registerComponent holds the part
// that has to stay identical: create the map on demand, keep the first entry when
// the name is a duplicate, and apply the register-anyway policy (see
// appendKeyError) — a value is registered even when its name or content failed
// validation, so a $ref written against it stays resolvable. A nil value is the one
// exception: null is not a valid component, so there is nothing to register.

// componentMaps access the document maps, so a caller names the map instead of
// repeating the nil check and the duplicate test.
var (
	serversMap         = func(b *builder) *map[string]*spec.Server { return &b.doc.Servers }
	schemasMap         = func(b *builder) *map[string]*spec.Schema { return &b.components().Schemas }
	securitySchemesMap = func(b *builder) *map[string]*spec.SecurityScheme { return &b.components().SecuritySchemes }
	repliesMap         = func(b *builder) *map[string]*spec.OperationReply { return &b.components().Replies }
	replyAddressesMap  = func(b *builder) *map[string]*spec.OperationReplyAddress { return &b.components().ReplyAddresses }
	correlationIDsMap  = func(b *builder) *map[string]*spec.CorrelationID { return &b.components().CorrelationIDs }
	operationTraitsMap = func(b *builder) *map[string]*spec.OperationTrait { return &b.components().OperationTraits }
	messageTraitsMap   = func(b *builder) *map[string]*spec.MessageTrait { return &b.components().MessageTraits }
)

// registerComponent records v under name in the map chosen by field, creating the
// map when the document has none. It returns the duplicate-name error, if any; the
// first entry always wins.
func registerComponent[V any](
	b *builder,
	field func(*builder) *map[string]*V,
	kind, name string,
	v *V,
) []error {
	m := field(b)
	if *m == nil {
		*m = map[string]*V{}
	}
	if _, dup := (*m)[name]; dup && name != "" {
		return []error{fmt.Errorf("%s.%s: duplicate name", kind, name)}
	}
	if name != "" && v != nil {
		(*m)[name] = v
	}
	return nil
}
