package asyncgo

import (
	"reflect"
	"testing"

	"github.com/RubenRibGarcia/asyncgo/schema"
	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInfoFields(t *testing.T) {
	i := Info("Orders", "1.0.0").
		Description("desc").
		TermsOfService("tos").
		Contact(spec.Contact{Name: "Orders Team", Email: "orders@example.com"}).
		License(spec.License{Name: "MIT"}).
		Tags(spec.Tag{Name: "orders"})

	assert.Equal(t, "desc", i.info.Description)
	assert.Equal(t, "tos", i.info.TermsOfService)
	require.NotNil(t, i.info.Contact)
	assert.Equal(t, "Orders Team", i.info.Contact.Name)
	require.NotNil(t, i.info.License)
	assert.Equal(t, "MIT", i.info.License.Name)
	require.Len(t, i.info.Tags, 1)
	assert.Equal(t, "orders", i.info.Tags[0].Name)
}

func TestServerFields(t *testing.T) {
	s := Server("prod", "kafka", "broker:9092").
		ProtocolVersion("1.0").
		Description("desc").
		Variable("host", spec.ServerVariable{Default: "broker"})

	assert.Equal(t, "1.0", s.s.ProtocolVersion)
	assert.Equal(t, "desc", s.s.Description)
	require.NotNil(t, s.s.Variables)
	assert.Equal(t, "broker", s.s.Variables["host"].Default)

	// A second variable does not reset the map.
	s.Variable("port", spec.ServerVariable{Default: "9092"})
	assert.Len(t, s.s.Variables, 2)
}

func TestChannelReceive(t *testing.T) {
	c := Channel("order-placed").
		Title("Order placed").
		Receive(Operation().Message(MessageOf(OrderPlaced{}).Name("OrderPlaced")))

	assert.Equal(t, "Order placed", c.s.Title)

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	require.NoError(t, c.apply(b))

	require.Contains(t, b.doc.Operations, "order-placed.receive")
	op := b.doc.Operations["order-placed.receive"]
	assert.Equal(t, spec.ActionReceive, op.Action)
	assert.Equal(t, "#/channels/order-placed", op.Channel.Ref)
	require.Contains(t, b.doc.Channels["order-placed"].Messages, "OrderPlaced")
}

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

func TestValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		spec func() *SpecResult
		want string
	}{
		{
			name: "should_return_error_when_server_host_is_empty",
			spec: func() *SpecResult {
				return Spec(Info("Orders", "1.0.0"), Servers(Server("prod", "kafka", "")))
			},
			want: "server.prod.host: is required",
		},
		{
			name: "should_return_error_when_server_protocol_is_empty",
			spec: func() *SpecResult {
				return Spec(Info("Orders", "1.0.0"), Servers(Server("prod", "", "broker:9092")))
			},
			want: "server.prod.protocol: is required",
		},
		{
			name: "should_return_error_when_server_name_is_empty",
			spec: func() *SpecResult {
				return Spec(Info("Orders", "1.0.0"), Servers(Server("", "kafka", "broker:9092")))
			},
			want: "server.name: is required",
		},
		{
			name: "should_return_error_when_server_name_is_duplicate",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Servers(
						Server("prod", "kafka", "broker:9092"),
						Server("prod", "kafka", "broker-staging:9092"),
					),
				)
			},
			want: "server.prod: duplicate name",
		},
		{
			name: "should_return_error_when_server_name_is_duplicate_across_servers_calls",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Servers(Server("prod", "kafka", "broker:9092")),
					Servers(Server("prod", "kafka", "broker-staging:9092")),
				)
			},
			want: "server.prod: duplicate name",
		},
		{
			name: "should_return_error_when_channel_address_is_duplicate",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(
						Channel("order-placed"),
						Channel("order-placed"),
					),
				)
			},
			want: "channel.order-placed: duplicate address",
		},
		{
			name: "should_return_error_when_channel_address_is_duplicate_across_channels_calls",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(Channel("order-placed")),
					Channels(Channel("order-placed")),
				)
			},
			want: "channel.order-placed: duplicate address",
		},
		{
			name: "should_join_duplicate_and_field_validation_errors",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Servers(
						Server("prod", "kafka", "broker:9092"),
						Server("prod", "kafka", ""),
					),
				)
			},
			want: "server.prod.host: is required\nserver.prod: duplicate name",
		},
		{
			name: "should_return_error_when_info_title_is_missing",
			spec: func() *SpecResult {
				return Spec(Info("", "1.0.0"))
			},
			want: "info.title: is required",
		},
		{
			name: "should_return_error_when_info_version_is_missing",
			spec: func() *SpecResult {
				return Spec(Info("Orders", ""))
			},
			want: "info.version: is required",
		},
		{
			name: "should_join_multiple_validation_errors",
			spec: func() *SpecResult {
				return Spec(
					Info("", ""),
					Servers(Server("prod", "", "")),
				)
			},
			want: "info.title: is required\ninfo.version: is required\nserver.prod.protocol: is required\nserver.prod.host: is required",
		},
		{
			name: "should_return_nil_error_for_valid_catalog",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Servers(Server("prod", "kafka", "broker:9092")),
				)
			},
			want: "",
		},
		{
			name: "should_return_error_when_message_payload_type_is_nil",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(
						Channel("order-placed").
							Send(Operation().Message(MessageOf(nil))),
					),
				)
			},
			want: "message: nil payload type or schema",
		},
		{
			name: "should_return_error_when_channel_references_unknown_server",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(
						Channel("order-placed").
							Servers(Server("prod", "kafka", "broker:9092")),
					),
				)
			},
			want: `channel.order-placed: references unknown server "prod"`,
		},
		{
			name: "should_join_multiple_unknown_server_references",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(
						Channel("order-placed").
							Servers(
								Server("prod", "kafka", "broker:9092"),
								Server("staging", "kafka", "broker-staging:9092"),
							),
					),
				)
			},
			want: "channel.order-placed: references unknown server \"prod\"\n" +
				"channel.order-placed: references unknown server \"staging\"",
		},
		{
			name: "should_join_unknown_server_references_across_channels_in_sorted_order",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(
						Channel("order-shipped").
							Servers(Server("staging", "kafka", "broker-staging:9092")),
						Channel("order-placed").
							Servers(Server("prod", "kafka", "broker:9092")),
					),
				)
			},
			want: "channel.order-placed: references unknown server \"prod\"\n" +
				"channel.order-shipped: references unknown server \"staging\"",
		},
		{
			name: "should_allow_channel_to_reference_server_declared_after_it",
			spec: func() *SpecResult {
				prod := Server("prod", "kafka", "broker:9092")
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(Channel("order-placed").Servers(prod)),
					Servers(prod),
				)
			},
			want: "",
		},
		{
			name: "should_return_error_when_security_scheme_name_is_empty",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					SecuritySchemes(SecurityScheme("", spec.SecurityScheme{Type: "userPassword"})),
				)
			},
			want: "securityScheme.name: is required",
		},
		{
			name: "should_return_error_when_security_scheme_type_is_empty",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					SecuritySchemes(SecurityScheme("oauth", spec.SecurityScheme{})),
				)
			},
			want: "securityScheme.oauth.type: is required",
		},
		{
			name: "should_return_error_when_security_scheme_name_is_duplicate",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					SecuritySchemes(
						SecurityScheme("oauth", spec.SecurityScheme{Type: "oauth2"}),
						SecurityScheme("oauth", spec.SecurityScheme{Type: "http"}),
					),
				)
			},
			want: "securityScheme.oauth: duplicate name",
		},
		{
			name: "should_return_error_when_server_references_unknown_security_scheme",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Servers(
						Server("prod", "kafka", "broker:9092").
							Security(SecurityScheme("oauth", spec.SecurityScheme{Type: "oauth2"})),
					),
				)
			},
			want: `server.prod: references unknown security scheme "oauth"`,
		},
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
			name: "should_allow_security_scheme_declared_after_it_is_referenced",
			spec: func() *SpecResult {
				oauth := SecurityScheme("oauth", spec.SecurityScheme{Type: "oauth2"})
				return Spec(
					Info("Orders", "1.0.0"),
					Servers(Server("prod", "kafka", "broker:9092").Security(oauth)),
					SecuritySchemes(oauth),
				)
			},
			want: "",
		},
		{
			name: "should_return_error_when_reply_name_is_empty",
			spec: func() *SpecResult {
				return Spec(Info("Orders", "1.0.0"), Replies(Reply("")))
			},
			want: "reply.name: is required",
		},
		{
			name: "should_return_error_when_reply_name_is_duplicate",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Replies(Reply("OrderReply"), Reply("OrderReply")),
				)
			},
			want: "reply.OrderReply: duplicate name",
		},
		{
			name: "should_return_error_when_reply_address_name_is_empty",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					ReplyAddresses(ReplyAddress("").Location("$message.header#/replyTo")),
				)
			},
			want: "replyAddress.name: is required",
		},
		{
			name: "should_return_error_when_reply_address_location_is_empty",
			spec: func() *SpecResult {
				return Spec(Info("Orders", "1.0.0"), ReplyAddresses(ReplyAddress("ReplyTo")))
			},
			want: "replyAddress.ReplyTo.location: is required",
		},
		{
			name: "should_return_error_when_reply_address_name_is_duplicate",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					ReplyAddresses(
						ReplyAddress("ReplyTo").Location("$message.header#/replyTo"),
						ReplyAddress("ReplyTo").Location("$message.header#/inbox"),
					),
				)
			},
			want: "replyAddress.ReplyTo: duplicate name",
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
			name: "should_return_error_when_reply_references_unknown_reply_address",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Replies(Reply("OrderReply").Address(ReplyAddress("ReplyTo"))),
				)
			},
			want: `reply.OrderReply: references unknown reply address "ReplyTo"`,
		},
		{
			name: "should_return_error_when_reply_references_unknown_channel",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Replies(Reply("OrderReply").Channel(Channel("order-replies"))),
				)
			},
			want: `reply.OrderReply: references unknown channel "order-replies"`,
		},
		{
			name: "should_return_error_when_reply_sets_address_and_channel",
			spec: func() *SpecResult {
				replyTo := ReplyAddress("ReplyTo").Location("$message.header#/replyTo")
				orderReplies := Channel("order-replies")
				return Spec(
					Info("Orders", "1.0.0"),
					ReplyAddresses(replyTo),
					Channels(orderReplies),
					Replies(Reply("OrderReply").Address(replyTo).Channel(orderReplies)),
				)
			},
			want: "reply.OrderReply: address and channel are mutually exclusive (the referenced channel must have no address)",
		},
		{
			name: "should_return_error_when_reply_messages_have_no_channel",
			spec: func() *SpecResult {
				orderReplies := Channel("order-replies")
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(orderReplies),
					Replies(Reply("OrderReply").Message(
						orderReplies,
						MessageOf(OrderPlaced{}).Name("OrderAccepted"),
					)),
				)
			},
			want: "reply.OrderReply: messages require a channel",
		},
		{
			name: "should_return_error_when_reply_message_is_not_in_referenced_channel",
			spec: func() *SpecResult {
				orderPlaced := Channel("order-placed")
				orderReplies := Channel("order-replies")
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(orderPlaced, orderReplies),
					Replies(Reply("OrderReply").
						Channel(orderReplies).
						Message(orderPlaced, MessageOf(OrderPlaced{}).Name("OrderAccepted"))),
				)
			},
			want: `reply.OrderReply: message "#/channels/order-placed/messages/OrderAccepted" is not in channel "order-replies"`,
		},
		{
			name: "should_allow_reply_declared_after_it_is_referenced",
			spec: func() *SpecResult {
				replyTo := ReplyAddress("ReplyTo").Location("$message.header#/replyTo")
				orderReply := Reply("OrderReply").Address(replyTo)
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(
						Channel("order-placed").
							Send(Operation().
								Reply(orderReply).
								Message(MessageOf(OrderPlaced{}).Name("OrderPlaced"))),
					),
					Replies(orderReply),
					ReplyAddresses(replyTo),
				)
			},
			want: "",
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
			name: "should_return_error_when_message_references_unknown_message_trait",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(
						Channel("order-placed").
							Send(Operation().
								Message(MessageOf(OrderPlaced{}).
									Name("OrderPlaced").
									Traits(MessageTrait("Missing")))),
					),
				)
			},
			want: `channel.order-placed.messages.OrderPlaced: references unknown message trait "Missing"`,
		},
		{
			name: "should_return_error_when_message_trait_references_unknown_correlation_id",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					MessageTraits(
						MessageTrait("Traced").CorrelationID(
							CorrelationID("Missing", spec.CorrelationID{
								Location: "$message.header#/correlationId",
							}),
						),
					),
				)
			},
			want: `messageTrait.Traced: references unknown correlation id "Missing"`,
		},
		{
			name: "should_return_error_when_operation_trait_references_unknown_security_scheme",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					OperationTraits(
						OperationTrait("Kafka").
							Security(SecurityScheme("basic", spec.SecurityScheme{Type: "http"})),
					),
				)
			},
			want: `operationTrait.Kafka: references unknown security scheme "basic"`,
		},
		{
			name: "should_return_error_when_operation_trait_name_is_empty",
			spec: func() *SpecResult {
				return Spec(Info("Orders", "1.0.0"), OperationTraits(OperationTrait("")))
			},
			want: "operationTrait.name: is required",
		},
		{
			name: "should_return_error_when_operation_trait_name_is_duplicate",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					OperationTraits(OperationTrait("Kafka"), OperationTrait("Kafka")),
				)
			},
			want: "operationTrait.Kafka: duplicate name",
		},
		{
			name: "should_return_error_when_message_trait_name_is_empty",
			spec: func() *SpecResult {
				return Spec(Info("Orders", "1.0.0"), MessageTraits(MessageTrait("")))
			},
			want: "messageTrait.name: is required",
		},
		{
			name: "should_return_error_when_message_trait_name_is_duplicate",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					MessageTraits(MessageTrait("Traced"), MessageTrait("Traced")),
				)
			},
			want: "messageTrait.Traced: duplicate name",
		},
		{
			name: "should_return_error_when_correlation_id_name_is_empty",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					CorrelationIDs(CorrelationID("", spec.CorrelationID{
						Location: "$message.header#/correlationId",
					})),
				)
			},
			want: "correlationId.name: is required",
		},
		{
			name: "should_return_error_when_correlation_id_location_is_empty",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					CorrelationIDs(CorrelationID("CorrelationID", spec.CorrelationID{})),
				)
			},
			want: "correlationId.CorrelationID.location: is required",
		},
		{
			name: "should_return_error_when_correlation_id_name_is_duplicate",
			spec: func() *SpecResult {
				corr := spec.CorrelationID{Location: "$message.header#/correlationId"}
				return Spec(
					Info("Orders", "1.0.0"),
					CorrelationIDs(
						CorrelationID("CorrelationID", corr),
						CorrelationID("CorrelationID", corr),
					),
				)
			},
			want: "correlationId.CorrelationID: duplicate name",
		},
		{
			name: "should_allow_operation_trait_declared_after_it_is_referenced",
			spec: func() *SpecResult {
				trait := OperationTrait("Kafka")
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(
						Channel("order-placed").
							Send(Operation().
								Traits(trait).
								Message(MessageOf(OrderPlaced{}).Name("OrderPlaced"))),
					),
					OperationTraits(trait),
				)
			},
			want: "",
		},
		{
			name: "should_allow_message_trait_declared_after_it_is_referenced",
			spec: func() *SpecResult {
				trait := MessageTrait("Traced")
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(
						Channel("order-placed").
							Send(Operation().
								Message(MessageOf(OrderPlaced{}).Name("OrderPlaced").Traits(trait)),
							),
					),
					MessageTraits(trait),
				)
			},
			want: "",
		},
		{
			name: "should_allow_correlation_id_declared_after_it_is_referenced",
			spec: func() *SpecResult {
				corr := CorrelationID("CorrelationID", spec.CorrelationID{
					Location: "$message.header#/correlationId",
				})
				trait := MessageTrait("Traced").CorrelationID(corr)
				return Spec(
					Info("Orders", "1.0.0"),
					MessageTraits(trait),
					CorrelationIDs(corr),
				)
			},
			want: "",
		},
		{
			name: "should_register_a_declared_schema_component",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Schemas(Schema("UserAvro", spec.MultiFormat(
						"application/vnd.apache.avro;version=1.9.0",
						map[string]any{"type": "record", "name": "User"},
					))),
				)
			},
			want: "",
		},
		{
			name: "should_resolve_ref_to_declared_schema_from_two_messages",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Schemas(Schema("UserAvro", spec.MultiFormat(
						"application/vnd.apache.avro;version=1.9.0",
						map[string]any{"type": "record", "name": "User"},
					))),
					Channels(
						Channel("order-placed").Send(Operation().Message(
							MessageFrom("OrderPlaced", spec.Ref("#/components/schemas/UserAvro")),
						)),
						Channel("order-shipped").Send(Operation().Message(
							MessageFrom("OrderShipped", spec.Ref("#/components/schemas/UserAvro")),
						)),
					),
				)
			},
			want: "",
		},
		{
			name: "should_return_error_when_schema_name_is_empty",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Schemas(Schema("", &spec.Schema{Type: "object"})),
				)
			},
			want: "schema.name: is required",
		},
		{
			name: "should_return_error_on_duplicate_schema_name",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Schemas(
						Schema("User", &spec.Schema{Type: "object"}),
						Schema("User", &spec.Schema{Type: "string"}),
					),
				)
			},
			want: "schema.User: duplicate name",
		},
		{
			name: "should_reject_schema_format_without_schema",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Schemas(Schema("Avro", &spec.Schema{
						SchemaFormat: "application/vnd.apache.avro;version=1.9.0",
					})),
				)
			},
			want: "schema.Avro.schema: is required alongside schemaFormat",
		},
		{
			name: "should_reject_schema_without_schema_format",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Schemas(Schema("Avro", &spec.Schema{
						Schema: map[string]any{"type": "record"},
					})),
				)
			},
			want: "schema.Avro.schemaFormat: is required alongside schema",
		},
		{
			name: "should_reject_schema_format_beside_json_schema_keywords",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Schemas(Schema("Avro", &spec.Schema{
						SchemaFormat: "application/vnd.apache.avro;version=1.9.0",
						Schema:       map[string]any{"type": "record"},
						Type:         "object",
					})),
				)
			},
			want: "schema.Avro: multi format schema must not carry JSON Schema keyword(s): type",
		},
		{
			name: "should_reject_declared_name_colliding_with_hoisted_schema",
			spec: func() *SpecResult {
				fqn := schema.Name(reflect.TypeOf(OrderPlaced{}))
				return Spec(
					Info("Orders", "1.0.0"),
					Schemas(Schema(fqn, &spec.Schema{Type: "object"})),
					Channels(
						Channel("order-placed").Send(Operation().Message(
							MessageOf(OrderPlaced{}).Name("OrderPlaced"),
						)),
					),
				)
			},
			want: "schema." + schema.Name(reflect.TypeOf(OrderPlaced{})) +
				": collides with an auto-hoisted schema of the same name",
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

func TestDuplicateKeepsFirstEntry(t *testing.T) {
	t.Run("should_keep_first_server_on_duplicate_name", func(t *testing.T) {
		res := Spec(
			Info("Orders", "1.0.0"),
			Servers(
				Server("prod", "kafka", "broker:9092"),
				Server("prod", "amqp", "broker-staging:5672"),
			),
		)
		require.Error(t, res.Err)
		require.Contains(t, res.Doc.Servers, "prod")
		assert.Equal(t, "kafka", res.Doc.Servers["prod"].Protocol)
		assert.Equal(t, "broker:9092", res.Doc.Servers["prod"].Host)
	})

	t.Run("should_keep_first_channel_on_duplicate_address", func(t *testing.T) {
		res := Spec(
			Info("Orders", "1.0.0"),
			Channels(
				Channel("order-placed").Title("First"),
				Channel("order-placed").Title("Second"),
			),
		)
		require.Error(t, res.Err)
		require.Contains(t, res.Doc.Channels, "order-placed")
		assert.Equal(t, "First", res.Doc.Channels["order-placed"].Title)
	})

	t.Run("should_keep_first_security_scheme_on_duplicate_name", func(t *testing.T) {
		res := Spec(
			Info("Orders", "1.0.0"),
			SecuritySchemes(
				SecurityScheme("oauth", spec.SecurityScheme{Type: "oauth2"}),
				SecurityScheme("oauth", spec.SecurityScheme{Type: "http"}),
			),
		)
		require.Error(t, res.Err)
		require.NotNil(t, res.Doc.Components)
		require.Contains(t, res.Doc.Components.SecuritySchemes, "oauth")
		assert.Equal(t, "oauth2", res.Doc.Components.SecuritySchemes["oauth"].Type)
	})
}

func TestSecuritySchemeFields(t *testing.T) {
	sc := SecurityScheme("oauth", spec.SecurityScheme{
		Type:             "oauth2",
		Description:      "OAuth 2.0",
		Name:             "Authorization",
		In:               "header",
		Scheme:           "bearer",
		BearerFormat:     "JWT",
		OpenIDConnectURL: "https://example.com/.well-known/openid-configuration",
		Scopes:           []string{"read:orders"},
		Flows: &spec.OAuthFlows{
			ClientCredentials: &spec.OAuthFlow{
				TokenURL:        "https://example.com/oauth/token",
				AvailableScopes: map[string]string{"read:orders": "read orders"},
			},
		},
	})

	assert.Equal(t, "oauth", sc.name)
	assert.Equal(t, "oauth2", sc.s.Type)
	assert.Equal(t, "header", sc.s.In)
	assert.Equal(t, "bearer", sc.s.Scheme)
	assert.Equal(t, "JWT", sc.s.BearerFormat)
	assert.Equal(t, []string{"read:orders"}, sc.s.Scopes)
	require.NotNil(t, sc.s.Flows)
	require.NotNil(t, sc.s.Flows.ClientCredentials)
	assert.Equal(t, "https://example.com/oauth/token", sc.s.Flows.ClientCredentials.TokenURL)
	assert.Equal(t, "read orders", sc.s.Flows.ClientCredentials.AvailableScopes["read:orders"])
}

func TestSecuritySchemesRegistersComponents(t *testing.T) {
	res := Spec(
		Info("Orders", "1.0.0"),
		SecuritySchemes(
			SecurityScheme("userPasswordAuth", spec.SecurityScheme{Type: "userPassword"}),
			SecurityScheme("apiKeyAuth", spec.SecurityScheme{Type: "apiKey", In: "user"}),
		),
	)

	require.NoError(t, res.Err)
	require.NotNil(t, res.Doc.Components)
	assert.Len(t, res.Doc.Components.SecuritySchemes, 2)
	assert.Equal(t, "userPassword", res.Doc.Components.SecuritySchemes["userPasswordAuth"].Type)
	assert.Equal(t, "user", res.Doc.Components.SecuritySchemes["apiKeyAuth"].In)
}

func TestServerSecurity(t *testing.T) {
	oauth := SecurityScheme("oauth", spec.SecurityScheme{Type: "oauth2"})

	s := Server("prod", "kafka", "broker:9092").Security(oauth)

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	require.NoError(t, Servers(s).apply(b))

	require.Len(t, b.doc.Servers["prod"].Security, 1)
	assert.Equal(t, "#/components/securitySchemes/oauth", b.doc.Servers["prod"].Security[0].Ref)
}

func TestServerSecurityAppendsAcrossCalls(t *testing.T) {
	oauth := SecurityScheme("oauth", spec.SecurityScheme{Type: "oauth2"})
	basic := SecurityScheme("basic", spec.SecurityScheme{Type: "http", Scheme: "basic"})

	s := Server("prod", "kafka", "broker:9092").Security(oauth).Security(basic)

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	require.NoError(t, Servers(s).apply(b))

	require.Len(t, b.doc.Servers["prod"].Security, 2)
	assert.Equal(t, "#/components/securitySchemes/oauth", b.doc.Servers["prod"].Security[0].Ref)
	assert.Equal(t, "#/components/securitySchemes/basic", b.doc.Servers["prod"].Security[1].Ref)
}

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

func TestReplyAddressFields(t *testing.T) {
	ra := ReplyAddress("ReplyTo").
		Description("Consumer inbox").
		Location("$message.header#/replyTo")

	assert.Equal(t, "ReplyTo", ra.name)
	assert.Equal(t, "Consumer inbox", ra.a.Description)
	assert.Equal(t, "$message.header#/replyTo", ra.a.Location)
}

func TestReplyFields(t *testing.T) {
	replyTo := ReplyAddress("ReplyTo").Location("$message.header#/replyTo")

	r := Reply("OrderReply").Address(replyTo)

	assert.Equal(t, "OrderReply", r.name)
	require.NotNil(t, r.r.Address)
	assert.Equal(t, "#/components/replyAddresses/ReplyTo", r.r.Address.Ref)
	assert.Nil(t, r.r.Channel)
	assert.Empty(t, r.r.Messages)
}

func TestReplyChannelAndMessages(t *testing.T) {
	orderReplies := Channel("order-replies")
	accepted := MessageOf(OrderPlaced{}).Name("OrderAccepted")
	rejected := MessageOf(OrderPlaced{}).Name("OrderRejected")

	r := Reply("OrderReply").
		Channel(orderReplies).
		Message(orderReplies, accepted).
		Message(orderReplies, rejected)

	require.NotNil(t, r.r.Channel)
	assert.Equal(t, "#/channels/order-replies", r.r.Channel.Ref)
	require.Len(t, r.r.Messages, 2)
	assert.Equal(t, "#/channels/order-replies/messages/OrderAccepted", r.r.Messages[0].Ref)
	assert.Equal(t, "#/channels/order-replies/messages/OrderRejected", r.r.Messages[1].Ref)
	assert.Nil(t, r.r.Address)
}

// Message names its channel, so the refs it builds do not depend on Channel
// having been called first.
func TestReplyMessageRefsAreOrderIndependent(t *testing.T) {
	orderReplies := Channel("order-replies")
	accepted := MessageOf(OrderPlaced{}).Name("OrderAccepted")

	messageFirst := Reply("A").Message(orderReplies, accepted).Channel(orderReplies)
	channelFirst := Reply("B").Channel(orderReplies).Message(orderReplies, accepted)

	assert.Equal(t, messageFirst.r.Messages, channelFirst.r.Messages)
	assert.Equal(t, messageFirst.r.Channel, channelFirst.r.Channel)
}

func TestReplyAddressesRegistersComponents(t *testing.T) {
	res := Spec(
		Info("Orders", "1.0.0"),
		ReplyAddresses(
			ReplyAddress("ReplyTo").Location("$message.header#/replyTo"),
			ReplyAddress("Inbox").Location("$message.header#/inbox"),
		),
	)

	require.NoError(t, res.Err)
	require.NotNil(t, res.Doc.Components)
	assert.Len(t, res.Doc.Components.ReplyAddresses, 2)
	assert.Equal(
		t,
		"$message.header#/replyTo",
		res.Doc.Components.ReplyAddresses["ReplyTo"].Location,
	)
}

func TestRepliesRegistersComponents(t *testing.T) {
	replyTo := ReplyAddress("ReplyTo").Location("$message.header#/replyTo")

	res := Spec(
		Info("Orders", "1.0.0"),
		ReplyAddresses(replyTo),
		Replies(Reply("OrderReply").Address(replyTo)),
	)

	require.NoError(t, res.Err)
	require.NotNil(t, res.Doc.Components)
	assert.Len(t, res.Doc.Components.Replies, 1)
	assert.Equal(
		t,
		"#/components/replyAddresses/ReplyTo",
		res.Doc.Components.Replies["OrderReply"].Address.Ref,
	)
}

func TestSchemasRegistersComponents(t *testing.T) {
	avro := spec.MultiFormat(
		"application/vnd.apache.avro;version=1.9.0",
		map[string]any{"type": "record", "name": "User"},
	)

	res := Spec(
		Info("Orders", "1.0.0"),
		Schemas(Schema("UserAvro", avro)),
		Channels(
			Channel("order-placed").Send(Operation().Message(
				MessageFrom("OrderPlaced", spec.Ref("#/components/schemas/UserAvro")),
			)),
			Channel("order-shipped").Send(Operation().Message(
				MessageFrom("OrderShipped", spec.Ref("#/components/schemas/UserAvro")),
			)),
		),
	)

	require.NoError(t, res.Err)
	require.NotNil(t, res.Doc.Components)
	require.Contains(t, res.Doc.Components.Schemas, "UserAvro")
	assert.Same(t, avro, res.Doc.Components.Schemas["UserAvro"])

	placed := res.Doc.Channels["order-placed"].Messages["OrderPlaced"]
	shipped := res.Doc.Channels["order-shipped"].Messages["OrderShipped"]
	require.NotNil(t, placed.Payload)
	require.NotNil(t, shipped.Payload)
	assert.Equal(t, "#/components/schemas/UserAvro", placed.Payload.Ref)
	assert.Equal(t, "#/components/schemas/UserAvro", shipped.Payload.Ref)
}

func TestReplyRefEscapesPointer(t *testing.T) {
	r := Reply("tenant/reply~prod")

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	require.NoError(t, Replies(r).apply(b))

	assert.Equal(t, "#/components/replies/tenant~1reply~0prod", replyRef(r).Ref)
	assert.Contains(t, b.doc.Components.Replies, "tenant/reply~prod")
}

func TestReplyAddressRefEscapesPointer(t *testing.T) {
	a := ReplyAddress("tenant/addr~prod").Location("$message.header#/replyTo")

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	require.NoError(t, ReplyAddresses(a).apply(b))

	assert.Equal(t, "#/components/replyAddresses/tenant~1addr~0prod", replyAddressRef(a).Ref)
	assert.Contains(t, b.doc.Components.ReplyAddresses, "tenant/addr~prod")
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

func TestOperationTraitFields(t *testing.T) {
	basic := SecurityScheme("basic", spec.SecurityScheme{Type: "http", Scheme: "basic"})

	tr := OperationTrait("KafkaOrders").
		Title("Kafka orders").
		Summary("Shared settings").
		Description("Applied to every orders operation").
		Security(basic).
		Tags(spec.Tag{Name: "orders"}).
		ExternalDocs(spec.ExternalDocs{URL: "https://example.com/orders"}).
		Kafka(spec.KafkaOperationBinding{
			GroupID:  &spec.Schema{Type: "string"},
			ClientID: &spec.Schema{Type: "string"},
		})

	assert.Equal(t, "KafkaOrders", tr.name)
	assert.Equal(t, "Kafka orders", tr.t.Title)
	assert.Equal(t, "Shared settings", tr.t.Summary)
	assert.Equal(t, "Applied to every orders operation", tr.t.Description)
	require.Len(t, tr.t.Security, 1)
	assert.Equal(t, "#/components/securitySchemes/basic", tr.t.Security[0].Ref)
	assert.Equal(t, []spec.Tag{{Name: "orders"}}, tr.t.Tags)
	require.NotNil(t, tr.t.ExternalDocs)
	assert.Equal(t, "https://example.com/orders", tr.t.ExternalDocs.URL)
	require.Contains(t, tr.t.Bindings, spec.ProtocolKafka)
	assert.Equal(
		t,
		&spec.Schema{Type: "string"},
		tr.t.Bindings[spec.ProtocolKafka].(*spec.KafkaOperationBinding).GroupID,
	)
}

func TestMessageTraitFields(t *testing.T) {
	corr := CorrelationID("CorrelationID", spec.CorrelationID{
		Description: "Correlation ID",
		Location:    "$message.header#/correlationId",
	})

	tr := MessageTrait("Traced").
		ContentType("application/json").
		Name("TracedMessage").
		Title("Traced").
		Summary("Shared headers").
		Description("Applied to every traced message").
		Headers(spec.Ref("#/components/schemas/Headers")).
		CorrelationID(corr).
		Tags(spec.Tag{Name: "traced"}).
		ExternalDocs(spec.ExternalDocs{URL: "https://example.com/traced"}).
		Example("simple", map[string]any{"order_id": "1"}).
		Kafka(spec.KafkaMessageBinding{Key: &spec.Schema{Type: "string"}})

	assert.Equal(t, "Traced", tr.name)
	assert.Equal(t, "application/json", tr.t.ContentType)
	assert.Equal(t, "TracedMessage", tr.t.Name)
	assert.Equal(t, "Traced", tr.t.Title)
	assert.Equal(t, "Shared headers", tr.t.Summary)
	assert.Equal(t, "Applied to every traced message", tr.t.Description)
	require.NotNil(t, tr.t.Headers)
	assert.Equal(t, "#/components/schemas/Headers", tr.t.Headers.Ref)
	require.NotNil(t, tr.t.CorrelationID)
	assert.Equal(t, "#/components/correlationIds/CorrelationID", tr.t.CorrelationID.Ref)
	assert.Equal(t, []spec.Tag{{Name: "traced"}}, tr.t.Tags)
	require.NotNil(t, tr.t.ExternalDocs)
	assert.Equal(t, "https://example.com/traced", tr.t.ExternalDocs.URL)
	require.Len(t, tr.t.Examples, 1)
	assert.Equal(t, "simple", tr.t.Examples[0].Name)
	require.Contains(t, tr.t.Bindings, spec.ProtocolKafka)
}

// TestOperationTraitBindingsReplaces pins the documented whole-map semantics:
// Bindings replaces the trait's protocol bindings, it does not merge them.
func TestOperationTraitBindingsReplaces(t *testing.T) {
	tr := OperationTrait("KafkaOrders").
		Bindings(spec.OperationBindings{
			spec.ProtocolAMQP: &spec.AMQPOperationBinding{Expiration: 30},
		})

	require.Len(t, tr.t.Bindings, 1)
	assert.Equal(
		t,
		30,
		tr.t.Bindings[spec.ProtocolAMQP].(*spec.AMQPOperationBinding).Expiration,
	)

	tr.Bindings(spec.OperationBindings{
		spec.ProtocolNATS: &spec.NATSOperationBinding{Queue: "q"},
	})

	require.Len(t, tr.t.Bindings, 1)
	assert.NotContains(t, tr.t.Bindings, spec.ProtocolAMQP)
	assert.Equal(t, "q", tr.t.Bindings[spec.ProtocolNATS].(*spec.NATSOperationBinding).Queue)
}

// TestMessageTraitBindingsReplaces is the message-trait counterpart of
// TestOperationTraitBindingsReplaces.
func TestMessageTraitBindingsReplaces(t *testing.T) {
	tr := MessageTrait("Traced").
		Bindings(spec.MessageBindings{
			spec.ProtocolAMQP: &spec.AMQPMessageBinding{ContentEncoding: "gzip"},
		})

	require.Len(t, tr.t.Bindings, 1)
	assert.Equal(
		t,
		"gzip",
		tr.t.Bindings[spec.ProtocolAMQP].(*spec.AMQPMessageBinding).ContentEncoding,
	)

	tr.Bindings(spec.MessageBindings{
		spec.ProtocolMQTT: &spec.MQTTMessageBinding{BindingVersion: "0.2.0"},
	})

	require.Len(t, tr.t.Bindings, 1)
	assert.NotContains(t, tr.t.Bindings, spec.ProtocolAMQP)
	assert.Equal(
		t,
		"0.2.0",
		tr.t.Bindings[spec.ProtocolMQTT].(*spec.MQTTMessageBinding).BindingVersion,
	)
}

func TestOperationTraitsRegistersComponents(t *testing.T) {
	res := Spec(
		Info("Orders", "1.0.0"),
		OperationTraits(
			OperationTrait("Kafka").Summary("kafka settings"),
			OperationTrait("AMQP").Summary("amqp settings"),
		),
	)

	require.NoError(t, res.Err)
	require.NotNil(t, res.Doc.Components)
	assert.Len(t, res.Doc.Components.OperationTraits, 2)
	assert.Equal(t, "kafka settings", res.Doc.Components.OperationTraits["Kafka"].Summary)
}

func TestMessageTraitsRegistersComponents(t *testing.T) {
	res := Spec(
		Info("Orders", "1.0.0"),
		MessageTraits(
			MessageTrait("Traced").ContentType("application/json"),
			MessageTrait("Untraced").ContentType("text/plain"),
		),
	)

	require.NoError(t, res.Err)
	require.NotNil(t, res.Doc.Components)
	assert.Len(t, res.Doc.Components.MessageTraits, 2)
	assert.Equal(t, "application/json", res.Doc.Components.MessageTraits["Traced"].ContentType)
}

func TestCorrelationIDsRegistersComponents(t *testing.T) {
	res := Spec(
		Info("Orders", "1.0.0"),
		CorrelationIDs(CorrelationID("CorrelationID", spec.CorrelationID{
			Description: "Correlation ID",
			Location:    "$message.header#/correlationId",
		})),
	)

	require.NoError(t, res.Err)
	require.NotNil(t, res.Doc.Components)
	assert.Len(t, res.Doc.Components.CorrelationIDs, 1)
	assert.Equal(
		t,
		"$message.header#/correlationId",
		res.Doc.Components.CorrelationIDs["CorrelationID"].Location,
	)
}

func TestOperationTraitRefEscapesPointer(t *testing.T) {
	tr := OperationTrait("tenant/trait~prod")

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	require.NoError(t, OperationTraits(tr).apply(b))

	assert.Equal(t, "#/components/operationTraits/tenant~1trait~0prod", operationTraitRef(tr).Ref)
	assert.Contains(t, b.doc.Components.OperationTraits, "tenant/trait~prod")
}

func TestMessageTraitRefEscapesPointer(t *testing.T) {
	tr := MessageTrait("tenant/trait~prod")

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	require.NoError(t, MessageTraits(tr).apply(b))

	assert.Equal(t, "#/components/messageTraits/tenant~1trait~0prod", messageTraitRef(tr).Ref)
	assert.Contains(t, b.doc.Components.MessageTraits, "tenant/trait~prod")
}

func TestCorrelationIDRefEscapesPointer(t *testing.T) {
	c := CorrelationID("tenant/id~prod", spec.CorrelationID{Location: "$message.header#/id"})

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	require.NoError(t, CorrelationIDs(c).apply(b))

	assert.Equal(t, "#/components/correlationIds/tenant~1id~0prod", correlationIDRef(c).Ref)
	assert.Contains(t, b.doc.Components.CorrelationIDs, "tenant/id~prod")
}

func TestOperationTraits(t *testing.T) {
	trait := OperationTrait("Kafka")

	c := Channel("order-placed").
		Send(Operation().
			Traits(trait).
			Message(MessageOf(OrderPlaced{}).Name("OrderPlaced")))

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	require.NoError(t, c.apply(b))

	require.Contains(t, b.doc.Operations, "order-placed.send")
	require.Len(t, b.doc.Operations["order-placed.send"].Traits, 1)
	assert.Equal(
		t,
		"#/components/operationTraits/Kafka",
		b.doc.Operations["order-placed.send"].Traits[0].Ref,
	)
}

// TestSharedOperationTraitOnTwoOperations is the unit-level counterpart of the
// fixture's headline case: one component, referenced by two operations.
func TestSharedOperationTraitOnTwoOperations(t *testing.T) {
	trait := OperationTrait("Kafka")

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	require.NoError(t, Channels(
		Channel("order-placed").Send(Operation().
			Traits(trait).
			Message(MessageOf(OrderPlaced{}).Name("OrderPlaced"))),
		Channel("order-shipped").Send(Operation().
			Traits(trait).
			Message(MessageOf(OrderPlaced{}).Name("OrderShipped"))),
	).apply(b))

	assert.Equal(
		t,
		"#/components/operationTraits/Kafka",
		b.doc.Operations["order-placed.send"].Traits[0].Ref,
	)
	assert.Equal(
		t,
		"#/components/operationTraits/Kafka",
		b.doc.Operations["order-shipped.send"].Traits[0].Ref,
	)
}

func TestMessageTraits(t *testing.T) {
	trait := MessageTrait("Traced")

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	msg, err := MessageOf(OrderPlaced{}).Name("OrderPlaced").Traits(trait).build(b)
	require.NoError(t, err)

	require.Len(t, msg.Traits, 1)
	assert.Equal(t, "#/components/messageTraits/Traced", msg.Traits[0].Ref)
}

func TestServerTagsAndExternalDocs(t *testing.T) {
	s := Server("prod", "kafka", "broker:9092").
		Tags(spec.Tag{Name: "prod"}).
		ExternalDocs(spec.ExternalDocs{URL: "https://example.com/prod"})

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	require.NoError(t, Servers(s).apply(b))

	assert.Equal(t, []spec.Tag{{Name: "prod"}}, b.doc.Servers["prod"].Tags)
	require.NotNil(t, b.doc.Servers["prod"].ExternalDocs)
	assert.Equal(t, "https://example.com/prod", b.doc.Servers["prod"].ExternalDocs.URL)
}

func TestChannelTagsAndExternalDocs(t *testing.T) {
	c := Channel("order-placed").
		Tags(spec.Tag{Name: "orders"}).
		ExternalDocs(spec.ExternalDocs{URL: "https://example.com/channel"})

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	require.NoError(t, c.apply(b))

	assert.Equal(t, []spec.Tag{{Name: "orders"}}, b.doc.Channels["order-placed"].Tags)
	require.NotNil(t, b.doc.Channels["order-placed"].ExternalDocs)
	assert.Equal(t, "https://example.com/channel", b.doc.Channels["order-placed"].ExternalDocs.URL)
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

func TestMessageTagsAndExternalDocs(t *testing.T) {
	m := MessageOf(OrderPlaced{}).Name("OrderPlaced").
		Tags(spec.Tag{Name: "orders"}).
		ExternalDocs(spec.ExternalDocs{URL: "https://example.com/message"})

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	msg, err := m.build(b)
	require.NoError(t, err)

	assert.Equal(t, []spec.Tag{{Name: "orders"}}, msg.Tags)
	require.NotNil(t, msg.ExternalDocs)
	assert.Equal(t, "https://example.com/message", msg.ExternalDocs.URL)
}
