package discovery

import (
	"testing"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// docWith builds a catalog document titled so that the first-document-wins rule
// on Info is observable, carrying the given components (nil for none).
func docWith(title string, c *spec.Components) *spec.AsyncAPI {
	d := spec.New()
	d.Info = spec.Info{Title: title, Version: "1.0.0"}
	d.Components = c
	return d
}

// TestMerge covers Merge's map unioning (first occurrence wins on key collision),
// its Info rule and its nil handling. Every case shares the same shape — two
// documents colliding on one key and differing on another — so they are rows of
// one table rather than five tests that repeated the fixture setup.
func TestMerge(t *testing.T) {
	tests := []struct {
		name   string
		docs   []*spec.AsyncAPI
		verify func(t *testing.T, out *spec.AsyncAPI)
	}{
		{
			name: "should_return_an_empty_document_for_a_nil_document",
			docs: []*spec.AsyncAPI{nil},
			verify: func(t *testing.T, out *spec.AsyncAPI) {
				t.Helper()
				require.NotNil(t, out)
				assert.Equal(t, spec.Version, out.AsyncAPI)
				assert.Empty(t, out.Info.Title)
				assert.Nil(t, out.Components)
			},
		},
		{
			name: "should_skip_nil_documents_between_real_ones",
			docs: []*spec.AsyncAPI{nil, docWith("T", nil), nil},
			verify: func(t *testing.T, out *spec.AsyncAPI) {
				t.Helper()
				assert.Equal(t, "T", out.Info.Title)
			},
		},
		{
			name: "should_leave_components_nil_when_no_document_has_any",
			docs: []*spec.AsyncAPI{docWith("T", nil)},
			verify: func(t *testing.T, out *spec.AsyncAPI) {
				t.Helper()
				assert.Equal(t, "T", out.Info.Title)
				assert.Nil(t, out.Components)
			},
		},
		{
			name: "should_merge_replies_and_reply_addresses",
			docs: []*spec.AsyncAPI{
				docWith("T", &spec.Components{
					Replies: map[string]*spec.OperationReply{
						"OrderReply": {
							Address: &spec.Reference{Ref: "#/components/replyAddresses/ReplyTo"},
						},
					},
					ReplyAddresses: map[string]*spec.OperationReplyAddress{
						"ReplyTo": {Location: "$message.header#/replyTo"},
					},
				}),
				docWith("U", &spec.Components{
					Replies: map[string]*spec.OperationReply{
						"OrderReply": {
							Address: &spec.Reference{Ref: "#/components/replyAddresses/Other"},
						},
						"OtherReply": {Channel: &spec.Reference{Ref: "#/channels/order-replies"}},
					},
					ReplyAddresses: map[string]*spec.OperationReplyAddress{
						"ReplyTo": {Location: "$message.header#/other"},
						"Other":   {Location: "$message.header#/other"},
					},
				}),
			},
			verify: func(t *testing.T, out *spec.AsyncAPI) {
				t.Helper()
				require.NotNil(t, out.Components)
				assert.Len(t, out.Components.Replies, 2)
				assert.Len(t, out.Components.ReplyAddresses, 2)
				assert.Equal(
					t,
					"#/components/replyAddresses/ReplyTo",
					out.Components.Replies["OrderReply"].Address.Ref,
					"first occurrence wins",
				)
				assert.Equal(
					t,
					"$message.header#/replyTo",
					out.Components.ReplyAddresses["ReplyTo"].Location,
					"first occurrence wins",
				)
				assert.Equal(
					t,
					"#/channels/order-replies",
					out.Components.Replies["OtherReply"].Channel.Ref,
				)
			},
		},
		{
			name: "should_merge_security_schemes",
			docs: []*spec.AsyncAPI{
				docWith("T", &spec.Components{
					SecuritySchemes: map[string]*spec.SecurityScheme{
						"oauth": {Type: "oauth2"},
					},
				}),
				docWith("U", &spec.Components{
					SecuritySchemes: map[string]*spec.SecurityScheme{
						"basic": {Type: "http", Scheme: "basic"},
						"oauth": {Type: "http", Scheme: "basic"},
					},
				}),
			},
			verify: func(t *testing.T, out *spec.AsyncAPI) {
				t.Helper()
				require.NotNil(t, out.Components)
				assert.Len(t, out.Components.SecuritySchemes, 2)
				assert.Equal(
					t,
					"oauth2",
					out.Components.SecuritySchemes["oauth"].Type,
					"first occurrence wins",
				)
				assert.Equal(t, "basic", out.Components.SecuritySchemes["basic"].Scheme)
			},
		},
		{
			name: "should_merge_operation_and_message_traits",
			docs: []*spec.AsyncAPI{
				docWith("T", &spec.Components{
					OperationTraits: map[string]*spec.OperationTrait{
						"Kafka": {Summary: "first"},
					},
					MessageTraits: map[string]*spec.MessageTrait{
						"Traced": {ContentType: "application/json"},
					},
				}),
				docWith("U", &spec.Components{
					OperationTraits: map[string]*spec.OperationTrait{
						"Kafka": {Summary: "second"},
						"AMQP":  {Summary: "amqp"},
					},
					MessageTraits: map[string]*spec.MessageTrait{
						"Traced": {ContentType: "text/plain"},
						"Other":  {},
					},
				}),
			},
			verify: func(t *testing.T, out *spec.AsyncAPI) {
				t.Helper()
				require.NotNil(t, out.Components)
				assert.Len(t, out.Components.OperationTraits, 2)
				assert.Len(t, out.Components.MessageTraits, 2)
				assert.Equal(
					t,
					"first",
					out.Components.OperationTraits["Kafka"].Summary,
					"first occurrence wins",
				)
				assert.Equal(
					t,
					"application/json",
					out.Components.MessageTraits["Traced"].ContentType,
					"first occurrence wins",
				)
				assert.Equal(t, "amqp", out.Components.OperationTraits["AMQP"].Summary)
			},
		},
		{
			name: "should_merge_correlation_ids",
			docs: []*spec.AsyncAPI{
				docWith("T", &spec.Components{
					CorrelationIDs: map[string]*spec.CorrelationID{
						"CorrelationID": {Location: "$message.header#/first"},
					},
				}),
				docWith("U", &spec.Components{
					CorrelationIDs: map[string]*spec.CorrelationID{
						"CorrelationID": {Location: "$message.header#/second"},
						"Other":         {Location: "$message.header#/other"},
					},
				}),
			},
			verify: func(t *testing.T, out *spec.AsyncAPI) {
				t.Helper()
				require.NotNil(t, out.Components)
				assert.Len(t, out.Components.CorrelationIDs, 2)
				assert.Equal(
					t,
					"$message.header#/first",
					out.Components.CorrelationIDs["CorrelationID"].Location,
					"first occurrence wins",
				)
				assert.Equal(
					t,
					"$message.header#/other",
					out.Components.CorrelationIDs["Other"].Location,
				)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.verify(t, Merge(tc.docs...))
		})
	}
}
