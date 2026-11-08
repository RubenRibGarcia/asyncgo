package discovery

import (
	"strconv"
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
			// Info is taken wholesale, not field by field: the first document with a
			// Title supplies the whole section, and every later one is ignored even
			// when it carries fields the chosen one leaves empty.
			name: "should_take_info_from_the_first_document_with_a_title",
			docs: []*spec.AsyncAPI{
				docWith("", &spec.Components{}),
				docWith("T", nil),
				docWith("U", nil),
			},
			verify: func(t *testing.T, out *spec.AsyncAPI) {
				t.Helper()
				assert.Equal(t, "T", out.Info.Title)
				assert.Equal(t, "1.0.0", out.Info.Version)
			},
		},
		{
			// A document with no Title contributes no Info at all — its Version does
			// not sneak in ahead of the document that does have one.
			name: "should_ignore_a_document_without_a_title",
			docs: []*spec.AsyncAPI{
				{AsyncAPI: spec.Version, Info: spec.Info{Version: "9.9.9"}},
				docWith("T", nil),
			},
			verify: func(t *testing.T, out *spec.AsyncAPI) {
				t.Helper()
				assert.Equal(t, "T", out.Info.Title)
				assert.Equal(t, "1.0.0", out.Info.Version)
			},
		},
		{
			name: "should_take_the_first_default_content_type",
			docs: []*spec.AsyncAPI{
				{AsyncAPI: spec.Version, DefaultContentType: "application/json"},
				{AsyncAPI: spec.Version, DefaultContentType: "application/xml"},
			},
			verify: func(t *testing.T, out *spec.AsyncAPI) {
				t.Helper()
				assert.Equal(t, "application/json", out.DefaultContentType)
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
			merged, _ := Merge(documentsOf(tc.docs)...)
			tc.verify(t, merged)
		})
	}
}

// documentsOf wraps bare documents as Document values. The fixtures are about
// content, so their catalogs are named by position; only the collision tests
// care about identity, and they build their own.
func documentsOf(docs []*spec.AsyncAPI) []Document {
	wrapped := make([]Document, 0, len(docs))
	for i, d := range docs {
		wrapped = append(wrapped, Document{Catalog: catalogNamed(i), Doc: d})
	}
	return wrapped
}

// catalogNamed builds a catalog identity for a test fixture. The name follows the
// position so a collision message can be asserted in full.
func catalogNamed(i int) Catalog {
	return Catalog{PkgPath: "example.com/pkg" + strconv.Itoa(i), VarName: "Catalog"}
}

// TestMergeReportsCollisions pins the behaviour that motivated labelling
// documents: first-wins used to discard the losing definition with no trace, so a
// merged document could be quietly wrong for one of its catalogs.
func TestMergeReportsCollisions(t *testing.T) {
	t.Run("should_report_a_channel_two_catalogs_declared", func(t *testing.T) {
		first, second := catalogNamed(0), catalogNamed(1)
		a := spec.New()
		a.Channels = map[string]*spec.Channel{"order-placed": {Address: "orders"}}
		b := spec.New()
		b.Channels = map[string]*spec.Channel{"order-placed": {Address: "orders-v2"}}

		merged, collisions := Merge(
			Document{Catalog: first, Doc: a},
			Document{Catalog: second, Doc: b},
		)

		assert.Equal(t, "orders", merged.Channels["order-placed"].Address, "first wins")
		require.Len(t, collisions, 1)
		assert.Equal(t, "channels", collisions[0].Section)
		assert.Equal(t, "order-placed", collisions[0].Name)
		assert.Equal(t, first, collisions[0].Kept)
		assert.Equal(t, []Catalog{second}, collisions[0].Dropped)
	})

	t.Run("should_join_a_third_catalog_onto_the_same_collision", func(t *testing.T) {
		first, second, third := catalogNamed(0), catalogNamed(1), catalogNamed(2)
		doc := func(address string) *spec.AsyncAPI {
			d := spec.New()
			d.Channels = map[string]*spec.Channel{"c": {Address: address}}
			return d
		}

		_, collisions := Merge(
			Document{Catalog: first, Doc: doc("one")},
			Document{Catalog: second, Doc: doc("two")},
			Document{Catalog: third, Doc: doc("three")},
		)

		require.Len(t, collisions, 1, "one report per key, not one per loser")
		assert.Equal(t, first, collisions[0].Kept)
		assert.Equal(t, []Catalog{second, third}, collisions[0].Dropped)
	})

	t.Run("should_name_the_components_section_a_collision_happened_in", func(t *testing.T) {
		a := spec.New()
		a.Components = &spec.Components{Schemas: map[string]*spec.Schema{"X": {Type: "string"}}}
		b := spec.New()
		b.Components = &spec.Components{Schemas: map[string]*spec.Schema{"X": {Type: "integer"}}}

		_, collisions := Merge(
			Document{Catalog: catalogNamed(0), Doc: a},
			Document{Catalog: catalogNamed(1), Doc: b},
		)

		require.Len(t, collisions, 1)
		assert.Equal(t, "components.schemas", collisions[0].Section)
	})

	t.Run("should_report_a_losing_info_title", func(t *testing.T) {
		_, collisions := Merge(
			Document{Catalog: catalogNamed(0), Doc: docWith("Orders", nil)},
			Document{Catalog: catalogNamed(1), Doc: docWith("Admin", nil)},
		)

		require.Len(t, collisions, 1)
		assert.Equal(t, "info", collisions[0].Section)
		assert.Equal(t, "Admin", collisions[0].Name)
	})

	t.Run("should_report_nothing_when_keys_do_not_overlap", func(t *testing.T) {
		a := spec.New()
		a.Channels = map[string]*spec.Channel{"a": {Address: "a"}}
		b := spec.New()
		b.Channels = map[string]*spec.Channel{"b": {Address: "b"}}

		_, collisions := Merge(
			Document{Catalog: catalogNamed(0), Doc: a},
			Document{Catalog: catalogNamed(1), Doc: b},
		)

		assert.Empty(t, collisions)
	})

	t.Run("should_sort_collisions_by_section_then_name", func(t *testing.T) {
		a := spec.New()
		a.Channels = map[string]*spec.Channel{
			"zeta":  {Address: "z"},
			"alpha": {Address: "a"},
		}
		b := spec.New()
		b.Channels = map[string]*spec.Channel{
			"zeta":  {Address: "z"},
			"alpha": {Address: "a"},
		}

		_, collisions := Merge(
			Document{Catalog: catalogNamed(0), Doc: a},
			Document{Catalog: catalogNamed(1), Doc: b},
		)

		require.Len(t, collisions, 2)
		assert.Equal(t, "alpha", collisions[0].Name)
		assert.Equal(t, "zeta", collisions[1].Name)
	})
}

// TestCollisionFindings pins how a collision reaches the user: a located line naming
// both catalogs, so the drop is legible without reading the artifact.
func TestCollisionFindings(t *testing.T) {
	t.Run("should_name_the_kept_and_dropped_catalogs", func(t *testing.T) {
		findings := collisionFindings([]Collision{{
			Section: "channels",
			Name:    "order-placed",
			Kept:    Catalog{PkgPath: "example.com/admin", VarName: "AdminSpec"},
			Dropped: []Catalog{
				{PkgPath: "example.com/orders", VarName: "OrdersSpec"},
			},
		}})

		require.Len(t, findings, 1)
		assert.Equal(
			t,
			"channels.order-placed: duplicate, kept from example.com/admin.AdminSpec "+
				"(dropped from example.com/orders.OrdersSpec)",
			findings[0].String(),
		)
	})

	t.Run("should_list_every_dropped_catalog", func(t *testing.T) {
		findings := collisionFindings([]Collision{{
			Section: "channels",
			Name:    "c",
			Kept:    Catalog{PkgPath: "a", VarName: "A"},
			Dropped: []Catalog{
				{PkgPath: "b", VarName: "B"},
				{PkgPath: "c", VarName: "C"},
			},
		}})

		require.Len(t, findings, 1)
		assert.Contains(t, findings[0].String(), "dropped from b.B, c.C")
	})
}
