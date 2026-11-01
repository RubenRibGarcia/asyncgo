package asyncgo

import (
	"testing"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestSpecAcceptsValidCatalogs(t *testing.T) {
	tests := []struct {
		name string
		spec func() *SpecResult
		want string
	}{
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
			name: "should_allow_message_correlation_id_from_inline",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(
						Channel("order-placed").
							Send(Operation().
								Message(MessageOf(OrderPlaced{}).
									Name("OrderPlaced").
									CorrelationIDFrom(spec.CorrelationID{
										Location: "$message.header#/correlationId",
									}))),
					),
				)
			},
			want: "",
		},
		{
			name: "should_reuse_an_identical_inline_correlation_id_across_messages",
			spec: func() *SpecResult {
				corr := spec.CorrelationID{Location: "$message.header#/correlationId"}
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(
						Channel("order-placed").
							Send(Operation().
								Message(MessageOf(OrderPlaced{}).Name("OrderPlaced").CorrelationIDFrom(corr)),
							),
						Channel("order-shipped").
							Send(Operation().
								Message(MessageOf(OrderPlaced{}).Name("OrderPlaced").CorrelationIDFrom(corr)),
							),
					),
				)
			},
			want: "",
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
			// The key a message derives from its payload type is the type's fully
			// qualified name, whose package separators the pattern would reject. It
			// is exempt, so an unauthored message never trips the key check.
			name: "should_allow_a_message_key_derived_from_the_payload_type_name",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(Channel("order-placed").Send(Operation().Message(
						MessageOf(OrderPlaced{}),
					))),
				)
			},
			want: "",
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
