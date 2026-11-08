package asyncgo

import (
	"testing"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRefEscapesPointer pins one rule across every ref builder in the DSL: a
// component name is a JSON Pointer segment, so "/" becomes "~1" and "~" becomes
// "~0". A name carrying either character breaks the components.* key pattern,
// which is what makes these rows worth having — the key is reported as invalid
// and still registered, so the $ref written against it has to stay a valid
// pointer rather than dangle.
//
// The seven builders take different routes to their $ref — six name a component
// the item registers, one derives the key from its owner — so each row supplies
// its own run, and the table owns the assertions they share.
func TestRefEscapesPointer(t *testing.T) {
	tests := []struct {
		name string
		// run builds the object under test, registers what the DSL registers, and
		// returns the $ref to check.
		run func(t *testing.T, b *builder) *spec.Reference
		// wantRef is the escaped pointer the builder must emit.
		wantRef string
		// registered reports whether the broken key still reached the document,
		// which the register-anyway policy requires. nil where the ref is derived
		// rather than looked up in a component map.
		registered func(b *builder) bool
	}{
		{
			name: "should_escape_a_security_scheme_name",
			run: func(t *testing.T, b *builder) *spec.Reference {
				t.Helper()
				sc := SecurityScheme("tenant/oauth~prod", spec.SecurityScheme{Type: "oauth2"})
				s := Server("prod", "kafka", "broker:9092").Security(sc)

				// The scheme is referenced from a server, never applied here, so
				// no key-pattern error is reported on this path.
				require.NoError(t, Servers(s).apply(b))
				return b.doc.Servers["prod"].Security[0]
			},
			wantRef: "#/components/securitySchemes/tenant~1oauth~0prod",
		},
		{
			name: "should_escape_a_reply_name",
			run: func(t *testing.T, b *builder) *spec.Reference {
				t.Helper()
				r := Reply("tenant/reply~prod")
				require.Error(t, Replies(r).apply(b))
				return replyRef(r)
			},
			wantRef: "#/components/replies/tenant~1reply~0prod",
			registered: func(b *builder) bool {
				_, ok := b.doc.Components.Replies["tenant/reply~prod"]
				return ok
			},
		},
		{
			name: "should_escape_a_reply_address_name",
			run: func(t *testing.T, b *builder) *spec.Reference {
				t.Helper()
				a := ReplyAddress("tenant/addr~prod").Location("$message.header#/replyTo")
				require.Error(t, ReplyAddresses(a).apply(b))
				return replyAddressRef(a)
			},
			wantRef: "#/components/replyAddresses/tenant~1addr~0prod",
			registered: func(b *builder) bool {
				_, ok := b.doc.Components.ReplyAddresses["tenant/addr~prod"]
				return ok
			},
		},
		{
			name: "should_escape_an_operation_trait_name",
			run: func(t *testing.T, b *builder) *spec.Reference {
				t.Helper()
				tr := OperationTrait("tenant/trait~prod")
				require.Error(t, OperationTraits(tr).apply(b))
				return operationTraitRef(tr)
			},
			wantRef: "#/components/operationTraits/tenant~1trait~0prod",
			registered: func(b *builder) bool {
				_, ok := b.doc.Components.OperationTraits["tenant/trait~prod"]
				return ok
			},
		},
		{
			name: "should_escape_a_message_trait_name",
			run: func(t *testing.T, b *builder) *spec.Reference {
				t.Helper()
				tr := MessageTrait("tenant/trait~prod")
				require.Error(t, MessageTraits(tr).apply(b))
				return messageTraitRef(tr)
			},
			wantRef: "#/components/messageTraits/tenant~1trait~0prod",
			registered: func(b *builder) bool {
				_, ok := b.doc.Components.MessageTraits["tenant/trait~prod"]
				return ok
			},
		},
		{
			name: "should_escape_a_correlation_id_name",
			run: func(t *testing.T, b *builder) *spec.Reference {
				t.Helper()
				c := CorrelationID(
					"tenant/id~prod",
					spec.CorrelationID{Location: "$message.header#/id"},
				)
				require.Error(t, CorrelationIDs(c).apply(b))
				return correlationIDRef(c)
			},
			wantRef: "#/components/correlationIds/tenant~1id~0prod",
			registered: func(b *builder) bool {
				_, ok := b.doc.Components.CorrelationIDs["tenant/id~prod"]
				return ok
			},
		},
		{
			// The inline correlation id is the one ref derived from its owner's
			// name rather than looked up, so it is registered under
			// <owner>CorrelationID and escapes the same way.
			name: "should_escape_a_derived_inline_correlation_id",
			run: func(t *testing.T, b *builder) *spec.Reference {
				t.Helper()
				ref, err := correlationIDFromRef(b, "tenant/id~prod", spec.CorrelationID{
					Location: "$message.header#/id",
				})
				require.NoError(t, err)
				return ref
			},
			wantRef: "#/components/correlationIds/tenant~1id~0prodCorrelationID",
			registered: func(b *builder) bool {
				_, ok := b.doc.Components.CorrelationIDs["tenant/id~prodCorrelationID"]
				return ok
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}

			ref := tc.run(t, b)
			require.NotNil(t, ref)
			assert.Equal(t, tc.wantRef, ref.Ref)
			if tc.registered != nil {
				assert.True(t, tc.registered(b), "the component must still be registered")
			}
		})
	}
}
