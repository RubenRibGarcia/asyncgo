package asyncgo

import (
	"testing"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOperationFields(t *testing.T) {
	o := Operation().
		Title("t").
		Summary("s").
		Description("d").
		Message(MessageOf(OrderPlaced{}))

	assert.Equal(t, "t", o.title)
	assert.Equal(t, "s", o.summary)
	assert.Equal(t, "d", o.description)
	assert.Len(t, o.messages, 1)
}

func TestOperationSecurity(t *testing.T) {
	basic := SecurityScheme("basic", spec.SecurityScheme{Type: "http", Scheme: "basic"})

	c := Channel("order-placed").
		Send(Operation().
			Security(basic).
			Message(MessageOf(OrderPlaced{}).Name("OrderPlaced")))

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	require.NoError(t, c.apply(b))

	require.Contains(t, b.doc.Operations, "order-placed.send")
	require.Len(t, b.doc.Operations["order-placed.send"].Security, 1)
	assert.Equal(
		t,
		"#/components/securitySchemes/basic",
		b.doc.Operations["order-placed.send"].Security[0].Ref,
	)
}

func TestOperationReply(t *testing.T) {
	replyTo := ReplyAddress("ReplyTo").Location("$message.header#/replyTo")
	orderReply := Reply("OrderReply").Address(replyTo)

	c := Channel("order-placed").
		Send(Operation().
			Reply(orderReply).
			Message(MessageOf(OrderPlaced{}).Name("OrderPlaced")))

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	require.NoError(t, c.apply(b))

	require.Contains(t, b.doc.Operations, "order-placed.send")
	require.NotNil(t, b.doc.Operations["order-placed.send"].Reply)
	assert.Equal(
		t,
		"#/components/replies/OrderReply",
		b.doc.Operations["order-placed.send"].Reply.Ref,
	)
}

func TestOperationWithoutReplyOmitsReply(t *testing.T) {
	c := Channel("order-placed").
		Send(Operation().Message(MessageOf(OrderPlaced{}).Name("OrderPlaced")))

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	require.NoError(t, c.apply(b))

	assert.Nil(t, b.doc.Operations["order-placed.send"].Reply)
}

func TestOperationTagsAndExternalDocs(t *testing.T) {
	c := Channel("order-placed").
		Send(Operation().
			Tags(spec.Tag{Name: "orders"}).
			ExternalDocs(spec.ExternalDocs{URL: "https://example.com/operation"}).
			Message(MessageOf(OrderPlaced{}).Name("OrderPlaced")))

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	require.NoError(t, c.apply(b))

	op := b.doc.Operations["order-placed.send"]
	assert.Equal(t, []spec.Tag{{Name: "orders"}}, op.Tags)
	require.NotNil(t, op.ExternalDocs)
	assert.Equal(t, "https://example.com/operation", op.ExternalDocs.URL)
}

func TestOperationValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		spec func() *SpecResult
		want string
	}{
		{
			name: "should_return_error_when_operation_references_unknown_security_scheme",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(
						Channel("order-placed").
							Send(Operation().
								Security(SecurityScheme("oauth", spec.SecurityScheme{Type: "oauth2"})),
							),
					),
				)
			},
			want: `operation.order-placed.send: references unknown security scheme "oauth"`,
		},
		{
			name: "should_join_unknown_security_refs_across_operations_in_sorted_order",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(
						Channel("order-shipped").
							Send(Operation().Security(
								SecurityScheme("shipping", spec.SecurityScheme{Type: "oauth2"}),
							)),
						Channel("order-placed").
							Send(Operation().Security(
								SecurityScheme("orders", spec.SecurityScheme{Type: "oauth2"}),
							)),
					),
				)
			},
			want: `operation.order-placed.send: references unknown security scheme "orders"` + "\n" +
				`operation.order-shipped.send: references unknown security scheme "shipping"`,
		},
		{
			name: "should_return_error_when_operation_references_unknown_reply",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(
						Channel("order-placed").
							Send(Operation().
								Reply(Reply("OrderReply")).
								Message(MessageOf(OrderPlaced{}).Name("OrderPlaced"))),
					),
				)
			},
			want: `operation.order-placed.send: references unknown reply "OrderReply"`,
		},
		{
			name: "should_return_error_when_operation_references_unknown_operation_trait",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(
						Channel("order-placed").
							Send(Operation().
								Traits(OperationTrait("Missing")).
								Message(MessageOf(OrderPlaced{}).Name("OrderPlaced"))),
					),
				)
			},
			want: `operation.order-placed.send: references unknown operation trait "Missing"`,
		},
		{
			name: "should_return_error_when_a_channel_repeats_an_action",
			spec: func() *SpecResult {
				op := func() *operation {
					return Operation().Message(MessageOf(OrderPlaced{}).Name("OrderPlaced"))
				}
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(Channel("order-placed").Send(op()).Send(op())),
				)
			},
			want: "operation.order-placed.send: duplicate action",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := tc.spec()
			if tc.want == "" {
				require.NoError(t, res.Err)
				assert.Nil(t, res.ValidationErrors())
				return
			}
			require.Error(t, res.Err)
			assert.Equal(t, tc.want, res.Err.Error())
		})
	}
}
