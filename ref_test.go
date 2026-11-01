package asyncgo

import (
	"testing"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecuritySchemeRefEscapesPointer(t *testing.T) {
	sc := SecurityScheme("tenant/oauth~prod", spec.SecurityScheme{Type: "oauth2"})

	s := Server("prod", "kafka", "broker:9092").Security(sc)

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	require.NoError(t, Servers(s).apply(b))

	assert.Equal(
		t,
		"#/components/securitySchemes/tenant~1oauth~0prod",
		b.doc.Servers["prod"].Security[0].Ref,
	)
}

func TestReplyRefEscapesPointer(t *testing.T) {
	r := Reply("tenant/reply~prod")

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	// The name breaks the components.* key pattern, so apply reports it. The
	// reply is still registered, and its $ref must remain a valid JSON Pointer.
	require.Error(t, Replies(r).apply(b))

	assert.Equal(t, "#/components/replies/tenant~1reply~0prod", replyRef(r).Ref)
	assert.Contains(t, b.doc.Components.Replies, "tenant/reply~prod")
}

func TestReplyAddressRefEscapesPointer(t *testing.T) {
	a := ReplyAddress("tenant/addr~prod").Location("$message.header#/replyTo")

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	// The name breaks the components.* key pattern, so apply reports it. The
	// reply address is still registered, and its $ref must remain a valid JSON
	// Pointer.
	require.Error(t, ReplyAddresses(a).apply(b))

	assert.Equal(t, "#/components/replyAddresses/tenant~1addr~0prod", replyAddressRef(a).Ref)
	assert.Contains(t, b.doc.Components.ReplyAddresses, "tenant/addr~prod")
}

func TestOperationTraitRefEscapesPointer(t *testing.T) {
	tr := OperationTrait("tenant/trait~prod")

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	// The name breaks the components.* key pattern, so apply reports it. The
	// trait is still registered, and its $ref must remain a valid JSON Pointer.
	require.Error(t, OperationTraits(tr).apply(b))

	assert.Equal(t, "#/components/operationTraits/tenant~1trait~0prod", operationTraitRef(tr).Ref)
	assert.Contains(t, b.doc.Components.OperationTraits, "tenant/trait~prod")
}

func TestMessageTraitRefEscapesPointer(t *testing.T) {
	tr := MessageTrait("tenant/trait~prod")

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	// The name breaks the components.* key pattern, so apply reports it. The
	// trait is still registered, and its $ref must remain a valid JSON Pointer.
	require.Error(t, MessageTraits(tr).apply(b))

	assert.Equal(t, "#/components/messageTraits/tenant~1trait~0prod", messageTraitRef(tr).Ref)
	assert.Contains(t, b.doc.Components.MessageTraits, "tenant/trait~prod")
}

func TestCorrelationIDRefEscapesPointer(t *testing.T) {
	c := CorrelationID("tenant/id~prod", spec.CorrelationID{Location: "$message.header#/id"})

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	// The name breaks the components.* key pattern, so apply reports it. The
	// correlation id is still registered, and its $ref must remain a valid JSON
	// Pointer.
	require.Error(t, CorrelationIDs(c).apply(b))

	assert.Equal(t, "#/components/correlationIds/tenant~1id~0prod", correlationIDRef(c).Ref)
	assert.Contains(t, b.doc.Components.CorrelationIDs, "tenant/id~prod")
}

func TestCorrelationIDFromRefEscapesPointer(t *testing.T) {
	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}

	ref, err := correlationIDFromRef(b, "tenant/id~prod", spec.CorrelationID{
		Location: "$message.header#/id",
	})
	require.NoError(t, err)

	assert.Equal(t, "#/components/correlationIds/tenant~1id~0prodCorrelationID", ref.Ref)
	assert.Contains(t, b.doc.Components.CorrelationIDs, "tenant/id~prodCorrelationID")
}
