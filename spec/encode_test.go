package spec

import (
	"encoding/json"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeDefinitions(t *testing.T) {
	doc := New()
	doc.Components = &Components{
		Schemas: map[string]*Schema{
			"Event": {
				Definitions: map[string]*Schema{
					"inner": {Type: "string"},
				},
			},
		},
	}

	out, err := doc.YAML()
	require.NoError(t, err)

	// Draft 07 uses `definitions` (not 2019-09+ `$defs`).
	assert.Contains(t, string(out), "definitions:")
	assert.NotContains(t, string(out), "$defs")
}

func TestEncodeChannelServers(t *testing.T) {
	// Present: emits servers with $ref.
	withServers := New()
	withServers.Info = Info{Title: "Orders", Version: "1.0.0"}
	withServers.Channels = map[string]*Channel{
		"order-placed": {
			Address: "order-placed",
			Servers: []*Reference{{Ref: "#/servers/prod"}},
		},
	}
	out, err := withServers.YAML()
	require.NoError(t, err)
	assert.Contains(t, string(out), "servers:")
	assert.Contains(t, string(out), "#/servers/prod")

	// Omitted: a channel without servers has no `servers:` key.
	noServers := New()
	noServers.Info = Info{Title: "Orders", Version: "1.0.0"}
	noServers.Channels = map[string]*Channel{
		"order-placed": {Address: "order-placed"},
	}
	out, err = noServers.YAML()
	require.NoError(t, err)
	assert.NotContains(t, string(out), "servers:")
}

func TestEncodeJSON(t *testing.T) {
	doc := New()
	doc.Info = Info{Title: "Orders", Version: "1.0.0"}

	out, err := doc.JSON()
	require.NoError(t, err)
	assert.Contains(t, string(out), `"asyncapi":"3.1.0"`)
	assert.Contains(t, string(out), `"title":"Orders"`)
}

func TestEncodeJSONIndent(t *testing.T) {
	doc := New()
	doc.Info = Info{Title: "Orders", Version: "1.0.0"}
	doc.Servers = map[string]*Server{
		"prod": {Host: "broker:9092", Protocol: ProtocolKafka},
	}

	out, err := doc.JSONIndent()
	require.NoError(t, err)
	require.NotEmpty(t, out)

	// The artifact shape: parseable, 2-space indented, newline-terminated.
	var decoded AsyncAPI
	require.NoError(t, json.Unmarshal(out, &decoded))
	assert.Equal(t, Version, decoded.AsyncAPI)
	assert.Equal(t, "Orders", decoded.Info.Title)
	assert.Contains(t, string(out), "\n  \"asyncapi\": \"3.1.0\"")
	assert.Equal(t, byte('\n'), out[len(out)-1], "output must end with a newline")

	// The compact form stays on JSON(); JSONIndent must not emit it.
	assert.NotContains(t, string(out), `"asyncapi":"3.1.0"`)
}

func TestEncodeYAML(t *testing.T) {
	doc := New()
	doc.Info = Info{Title: "Orders", Version: "1.0.0"}
	doc.Servers = map[string]*Server{
		"prod": {Host: "broker:9092", Protocol: ProtocolKafka},
	}
	doc.Channels = map[string]*Channel{
		"order-placed": {
			Address: "order-placed",
			Messages: map[string]*Message{
				"orderPlaced": {
					Name:    "OrderPlaced",
					Payload: Ref("#/components/schemas/OrderPlaced"),
				},
			},
			Bindings: ChannelBindings{
				ProtocolKafka: &KafkaChannelBinding{Topic: "order-placed", Partitions: 3},
			},
		},
	}
	doc.Components = &Components{
		Schemas: map[string]*Schema{
			"OrderPlaced": {
				Type:       "object",
				Properties: map[string]*Schema{"id": {Type: "string"}},
				Required:   []string{"id"},
			},
		},
	}

	out, err := doc.YAML()
	require.NoError(t, err)
	s := string(out)

	for _, want := range []string{
		"asyncapi: 3.1.0",
		"title: Orders",
		"order-placed",
		"$ref:",
		"#/components/schemas/OrderPlaced",
		"type: object",
		"partitions: 3",
	} {
		assert.Contains(t, s, want)
	}
}

// TestEncodeYAMLEqualsJSON pins both codecs to the same document. They are
// independent implementations (goccy/go-yaml vs encoding/json) over a model
// whose Bindings, Example and Default fields are any-valued, so their outputs
// could diverge without any single-codec test noticing.
//
// Both sides are canonicalized through encoding/json before comparison: a
// direct require.Equal on the decoded documents is a false negative, because
// decoding YAML yields uint64(3) where decoding JSON yields float64(3) for the
// same any-valued binding number.
func TestEncodeYAMLEqualsJSON(t *testing.T) {
	doc := New()
	doc.Info = Info{Title: "Orders", Version: "1.0.0"}
	doc.Servers = map[string]*Server{
		"prod": {Host: "broker:9092", Protocol: ProtocolKafka},
	}
	doc.Channels = map[string]*Channel{
		"order-placed": {
			Address: "order-placed",
			Bindings: ChannelBindings{
				ProtocolKafka: &KafkaChannelBinding{Topic: "order-placed", Partitions: 3},
			},
		},
	}
	doc.Components = &Components{
		Schemas: map[string]*Schema{
			"OrderPlaced": {
				Type:       "object",
				Properties: map[string]*Schema{"id": {Type: "string", Example: "order-1"}},
				Required:   []string{"id"},
			},
		},
	}

	yamlOut, err := doc.YAML()
	require.NoError(t, err)
	jsonOut, err := doc.JSON()
	require.NoError(t, err)

	t.Run("should_decode_yaml_and_json_to_the_same_document", func(t *testing.T) {
		var fromYAML, fromJSON AsyncAPI
		require.NoError(t, yaml.Unmarshal(yamlOut, &fromYAML))
		require.NoError(t, json.Unmarshal(jsonOut, &fromJSON))

		assert.Equal(t, canonicalJSON(t, &fromYAML), canonicalJSON(t, &fromJSON))
	})

	t.Run("should_encode_the_indented_form_to_the_same_document", func(t *testing.T) {
		indented, err := doc.JSONIndent()
		require.NoError(t, err)

		// Decoded into any, both forms are key-sorted maps, so the comparison is
		// insensitive to the key order the encoder emitted.
		var fromCompact, fromIndented any
		require.NoError(t, json.Unmarshal(jsonOut, &fromCompact))
		require.NoError(t, json.Unmarshal(indented, &fromIndented))

		assert.Equal(t, canonicalJSON(t, fromCompact), canonicalJSON(t, fromIndented))
	})
}

// TestEncodeSecuritySchemes covers one case per security scheme type: the five
// types the golden fixture declares, plus a bare userPassword scheme.
func TestEncodeSecuritySchemes(t *testing.T) {
	tests := []struct {
		name     string
		scheme   SecurityScheme
		contains []string
	}{
		{
			name:     "should_encode_user_password",
			scheme:   SecurityScheme{Type: "userPassword"},
			contains: []string{"type: userPassword"},
		},
		{
			name:     "should_encode_api_key",
			scheme:   SecurityScheme{Type: "apiKey", In: "user"},
			contains: []string{"type: apiKey", "in: user"},
		},
		{
			name:     "should_encode_http",
			scheme:   SecurityScheme{Type: "http", Scheme: "bearer", BearerFormat: "JWT"},
			contains: []string{"type: http", "scheme: bearer", "bearerFormat: JWT"},
		},
		{
			name: "should_encode_oauth2",
			scheme: SecurityScheme{
				Type: "oauth2",
				Flows: &OAuthFlows{
					ClientCredentials: &OAuthFlow{
						TokenURL:        "https://example.com/oauth/token",
						AvailableScopes: map[string]string{"read:orders": "read orders"},
					},
				},
			},
			contains: []string{
				"type: oauth2",
				"flows:",
				"clientCredentials:",
				"tokenUrl: https://example.com/oauth/token",
				"availableScopes:",
				"read:orders: read orders",
			},
		},
		{
			name: "should_encode_open_id_connect",
			scheme: SecurityScheme{
				Type:             "openIdConnect",
				OpenIDConnectURL: "https://example.com/.well-known/openid-configuration",
				Scopes:           []string{"read:orders"},
			},
			contains: []string{
				"type: openIdConnect",
				"openIdConnectUrl: https://example.com/.well-known/openid-configuration",
				"scopes:",
				"- read:orders",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc := New()
			doc.Info = Info{Title: "Orders", Version: "1.0.0"}
			doc.Components = &Components{
				SecuritySchemes: map[string]*SecurityScheme{"auth": &tc.scheme},
			}

			out, err := doc.YAML()
			require.NoError(t, err)
			assert.Contains(t, string(out), "securitySchemes:")
			for _, want := range tc.contains {
				assert.Contains(t, string(out), want)
			}
		})
	}
}

func TestEncodeServerAndOperationSecurity(t *testing.T) {
	doc := New()
	doc.Info = Info{Title: "Orders", Version: "1.0.0"}
	doc.Servers = map[string]*Server{
		"prod": {
			Host:     "broker:9092",
			Protocol: ProtocolKafka,
			Security: []*Reference{{Ref: "#/components/securitySchemes/oauth"}},
		},
	}
	doc.Operations = map[string]*Operation{
		"order-placed.send": {
			Action:   ActionSend,
			Channel:  &Reference{Ref: "#/channels/order-placed"},
			Security: []*Reference{{Ref: "#/components/securitySchemes/basic"}},
		},
	}

	out, err := doc.YAML()
	require.NoError(t, err)
	assert.Contains(t, string(out), "security:")
	assert.Contains(t, string(out), "#/components/securitySchemes/oauth")
	assert.Contains(t, string(out), "#/components/securitySchemes/basic")

	jsonOut, err := doc.JSON()
	require.NoError(t, err)
	assert.Contains(
		t,
		string(jsonOut),
		`"security":[{"$ref":"#/components/securitySchemes/oauth"}]`,
	)
	assert.Contains(
		t,
		string(jsonOut),
		`"security":[{"$ref":"#/components/securitySchemes/basic"}]`,
	)
}

func TestEncodeSecuritySchemeOmitsZeroFields(t *testing.T) {
	doc := New()
	doc.Info = Info{Title: "Orders", Version: "1.0.0"}
	doc.Servers = map[string]*Server{"prod": {Host: "broker:9092", Protocol: ProtocolKafka}}

	out, err := doc.YAML()
	require.NoError(t, err)
	assert.NotContains(t, string(out), "securitySchemes:")
	assert.NotContains(t, string(out), "security:")

	doc.Components = &Components{
		SecuritySchemes: map[string]*SecurityScheme{"basic": {Type: "http", Scheme: "basic"}},
	}

	out, err = doc.YAML()
	require.NoError(t, err)
	assert.Contains(t, string(out), "securitySchemes:")
	assert.Contains(t, string(out), "type: http")
	assert.NotContains(t, string(out), "flows:")
	assert.NotContains(t, string(out), "openIdConnectUrl:")
}

// TestEncodeReply covers the reply refs the model can emit: an operation's
// reply, a reply's address, and a reply's channel and messages. The reply
// channel deliberately carries no address, which is the shape the specification
// requires when a reply names an address.
func TestEncodeReply(t *testing.T) {
	doc := New()
	doc.Info = Info{Title: "Orders", Version: "1.0.0"}
	doc.Channels = map[string]*Channel{
		"order-placed": {
			Address: "order-placed",
			Messages: map[string]*Message{
				"OrderPlaced": {Name: "OrderPlaced", Payload: &Schema{Type: "object"}},
			},
		},
		"order-replies": {
			Messages: map[string]*Message{
				"OrderAccepted": {Name: "OrderAccepted", Payload: &Schema{Type: "object"}},
			},
		},
	}
	doc.Operations = map[string]*Operation{
		"order-placed.send": {
			Action:  ActionSend,
			Channel: &Reference{Ref: "#/channels/order-placed"},
			Reply:   &Reference{Ref: "#/components/replies/OrderReply"},
		},
	}
	doc.Components = &Components{
		Replies: map[string]*OperationReply{
			"OrderReply": {Address: &Reference{Ref: "#/components/replyAddresses/ReplyTo"}},
			"OrderAcceptedReply": {
				Channel:  &Reference{Ref: "#/channels/order-replies"},
				Messages: []*Reference{{Ref: "#/channels/order-replies/messages/OrderAccepted"}},
			},
		},
		ReplyAddresses: map[string]*OperationReplyAddress{
			"ReplyTo": {Description: "Consumer inbox", Location: "$message.header#/replyTo"},
		},
	}

	yamlOut, err := doc.YAML()
	require.NoError(t, err)

	for _, want := range []string{
		"replies:",
		"replyAddresses:",
		"#/components/replies/OrderReply",
		"#/components/replyAddresses/ReplyTo",
		"#/channels/order-replies/messages/OrderAccepted",
		"description: Consumer inbox",
		`location: "$message.header#/replyTo"`,
	} {
		assert.Contains(t, string(yamlOut), want)
	}

	jsonOut, err := doc.JSON()
	require.NoError(t, err)
	for _, want := range []string{
		`"reply":{"$ref":"#/components/replies/OrderReply"}`,
		`"location":"$message.header#/replyTo"`,
		`"description":"Consumer inbox"`,
	} {
		assert.Contains(t, string(jsonOut), want)
	}

	t.Run("should_decode_yaml_and_json_to_the_same_document", func(t *testing.T) {
		var fromYAML, fromJSON AsyncAPI
		require.NoError(t, yaml.Unmarshal(yamlOut, &fromYAML))
		require.NoError(t, json.Unmarshal(jsonOut, &fromJSON))

		assert.Equal(t, canonicalJSON(t, &fromYAML), canonicalJSON(t, &fromJSON))
	})
}

// TestEncodeReplyAddressAlwaysEmitsLocation pins the one required field of an
// Operation Reply Address: it has to be emitted even when empty, because a
// dropped key would pass silently where an empty location is a validation error.
func TestEncodeReplyAddressAlwaysEmitsLocation(t *testing.T) {
	doc := New()
	doc.Info = Info{Title: "Orders", Version: "1.0.0"}
	doc.Components = &Components{
		ReplyAddresses: map[string]*OperationReplyAddress{"ReplyTo": {}},
	}

	out, err := doc.YAML()
	require.NoError(t, err)
	assert.Contains(t, string(out), "location:")

	jsonOut, err := doc.JSON()
	require.NoError(t, err)
	assert.Contains(t, string(jsonOut), `"location":""`)
}

func TestEncodeReplyOmitsZeroFields(t *testing.T) {
	doc := New()
	doc.Info = Info{Title: "Orders", Version: "1.0.0"}
	doc.Operations = map[string]*Operation{
		"order-placed.send": {
			Action:  ActionSend,
			Channel: &Reference{Ref: "#/channels/order-placed"},
		},
	}

	out, err := doc.YAML()
	require.NoError(t, err)
	assert.NotContains(t, string(out), "replies:")
	assert.NotContains(t, string(out), "replyAddresses:")
	assert.NotContains(t, string(out), "reply:")
}

// canonicalJSON renders v as JSON so two documents can be compared without
// depending on which decoder produced their any-valued fields. Map keys are
// sorted, so this comparison ignores key order inside an any-valued map: a
// document decoded from YAML or JSON holds map[string]any, while a hand-built
// document may hold a typed binding struct, and the two encode their keys in
// different orders.
func canonicalJSON(t *testing.T, v any) string {
	t.Helper()

	out, err := json.Marshal(v)
	require.NoError(t, err)

	return string(out)
}
