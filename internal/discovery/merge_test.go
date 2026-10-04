package discovery

import (
	"testing"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeSkipsNilDocuments(t *testing.T) {
	out := Merge(nil)
	require.NotNil(t, out)
	assert.Equal(t, spec.Version, out.AsyncAPI)
	assert.Empty(t, out.Info.Title)
	assert.Nil(t, out.Components)

	doc := spec.New()
	doc.Info = spec.Info{Title: "T", Version: "1.0.0"}
	out = Merge(nil, doc, nil)
	assert.Equal(t, "T", out.Info.Title)
}

func TestMergeNilComponents(t *testing.T) {
	doc := spec.New()
	doc.Info = spec.Info{Title: "T", Version: "1.0.0"}

	out := Merge(doc)
	require.NotNil(t, out)
	assert.Equal(t, "T", out.Info.Title)
	assert.Nil(t, out.Components)
}

func TestMergeReplies(t *testing.T) {
	first := spec.New()
	first.Info = spec.Info{Title: "T", Version: "1.0.0"}
	first.Components = &spec.Components{
		Replies: map[string]*spec.OperationReply{
			"OrderReply": {
				Address: &spec.Reference{Ref: "#/components/replyAddresses/ReplyTo"},
			},
		},
		ReplyAddresses: map[string]*spec.OperationReplyAddress{
			"ReplyTo": {Location: "$message.header#/replyTo"},
		},
	}

	second := spec.New()
	second.Info = spec.Info{Title: "U", Version: "1.0.0"}
	second.Components = &spec.Components{
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
	}

	out := Merge(first, second)
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
}

func TestMergeSecuritySchemes(t *testing.T) {
	first := spec.New()
	first.Info = spec.Info{Title: "T", Version: "1.0.0"}
	first.Components = &spec.Components{
		SecuritySchemes: map[string]*spec.SecurityScheme{
			"oauth": {Type: "oauth2"},
		},
	}

	second := spec.New()
	second.Info = spec.Info{Title: "U", Version: "1.0.0"}
	second.Components = &spec.Components{
		SecuritySchemes: map[string]*spec.SecurityScheme{
			"basic": {Type: "http", Scheme: "basic"},
			"oauth": {Type: "http", Scheme: "basic"},
		},
	}

	out := Merge(first, second)
	require.NotNil(t, out.Components)
	assert.Len(t, out.Components.SecuritySchemes, 2)
	assert.Equal(t, "oauth2", out.Components.SecuritySchemes["oauth"].Type, "first occurrence wins")
	assert.Equal(t, "basic", out.Components.SecuritySchemes["basic"].Scheme)
}
