package spec

import (
	"encoding/json"
	"reflect"
	"strings"
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
// whose Bindings, Const, Examples and Default fields are any-valued, so their
// outputs could diverge without any single-codec test noticing.
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
				Properties: map[string]*Schema{"id": {Type: "string", Const: "order-1"}},
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

// TestEncodeReferenceMessage covers a channel messages entry that is a
// Reference Object pointing at components.messages: the shape the generator
// emits for a hoisted message, and the referencing half of the Message Object |
// Reference Object union.
func TestEncodeReferenceMessage(t *testing.T) {
	doc := New()
	doc.Info = Info{Title: "Orders", Version: "1.0.0"}
	doc.Channels = map[string]*Channel{
		"order-placed": {
			Address: "order-placed",
			Messages: map[string]*Message{
				"OrderPlaced": {Ref: "#/components/messages/OrderPlaced"},
			},
		},
	}
	doc.Components = &Components{
		Messages: map[string]*Message{
			"OrderPlaced": {
				Name:    "OrderPlaced",
				Payload: Ref("#/components/schemas/OrderPlaced"),
			},
		},
	}

	yamlOut, err := doc.YAML()
	require.NoError(t, err)
	for _, want := range []string{
		"messages:",
		"#/components/messages/OrderPlaced",
		"name: OrderPlaced",
	} {
		assert.Contains(t, string(yamlOut), want)
	}

	jsonOut, err := doc.JSON()
	require.NoError(t, err)
	assert.Contains(
		t,
		string(jsonOut),
		`"channels":{"order-placed":{"address":"order-placed","messages":`+
			`{"OrderPlaced":{"$ref":"#/components/messages/OrderPlaced"}}}}`,
	)

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

func TestEncodeOperationTraits(t *testing.T) {
	doc := New()
	doc.Info = Info{Title: "Orders", Version: "1.0.0"}
	doc.Operations = map[string]*Operation{
		"order-placed.send": {
			Action:  ActionSend,
			Channel: &Reference{Ref: "#/channels/order-placed"},
			Traits:  []*Reference{{Ref: "#/components/operationTraits/Kafka"}},
		},
	}
	doc.Components = &Components{
		OperationTraits: map[string]*OperationTrait{
			"Kafka": {
				Summary:      "Shared Kafka settings",
				Security:     []*Reference{{Ref: "#/components/securitySchemes/basic"}},
				Tags:         []Tag{{Name: "orders"}},
				ExternalDocs: &ExternalDocs{URL: "https://example.com/traits"},
				Bindings: OperationBindings{
					ProtocolKafka: &KafkaOperationBinding{
						GroupID:  &Schema{Type: "string"},
						ClientID: &Schema{Type: "string"},
					},
				},
			},
		},
	}

	yamlOut, err := doc.YAML()
	require.NoError(t, err)
	for _, want := range []string{
		"operationTraits:",
		"#/components/operationTraits/Kafka",
		"summary: Shared Kafka settings",
		"externalDocs:",
		"groupId:",
		"type: string",
	} {
		assert.Contains(t, string(yamlOut), want)
	}

	jsonOut, err := doc.JSON()
	require.NoError(t, err)
	for _, want := range []string{
		`"operationTraits":`,
		`"traits":[{"$ref":"#/components/operationTraits/Kafka"}]`,
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

func TestEncodeMessageTraits(t *testing.T) {
	doc := New()
	doc.Info = Info{Title: "Orders", Version: "1.0.0"}
	doc.Channels = map[string]*Channel{
		"order-placed": {
			Address: "order-placed",
			Messages: map[string]*Message{
				"OrderPlaced": {
					Name:    "OrderPlaced",
					Payload: &Schema{Type: "object"},
					Traits:  []*Reference{{Ref: "#/components/messageTraits/Traced"}},
				},
			},
		},
	}
	doc.Components = &Components{
		MessageTraits: map[string]*MessageTrait{
			"Traced": {
				ContentType:   "application/json",
				CorrelationID: &Reference{Ref: "#/components/correlationIds/CorrelationID"},
				Tags:          []Tag{{Name: "traced"}},
			},
		},
		CorrelationIDs: map[string]*CorrelationID{
			"CorrelationID": {
				Description: "Correlation ID",
				Location:    "$message.header#/correlationId",
			},
		},
	}

	yamlOut, err := doc.YAML()
	require.NoError(t, err)
	for _, want := range []string{
		"messageTraits:",
		"correlationIds:",
		"#/components/messageTraits/Traced",
		"contentType: application/json",
		`location: "$message.header#/correlationId"`,
	} {
		assert.Contains(t, string(yamlOut), want)
	}

	jsonOut, err := doc.JSON()
	require.NoError(t, err)
	for _, want := range []string{
		`"messageTraits":`,
		`"traits":[{"$ref":"#/components/messageTraits/Traced"}]`,
		`"correlationId":{"$ref":"#/components/correlationIds/CorrelationID"}`,
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

// TestEncodeCorrelationIDAlwaysEmitsLocation pins the one required field of a
// Correlation ID Object: like an Operation Reply Address, an empty location has
// to be emitted and fail validation rather than silently drop the key.
func TestEncodeCorrelationIDAlwaysEmitsLocation(t *testing.T) {
	doc := New()
	doc.Info = Info{Title: "Orders", Version: "1.0.0"}
	doc.Components = &Components{
		CorrelationIDs: map[string]*CorrelationID{"CorrelationID": {}},
	}

	out, err := doc.YAML()
	require.NoError(t, err)
	assert.Contains(t, string(out), "location:")

	jsonOut, err := doc.JSON()
	require.NoError(t, err)
	assert.Contains(t, string(jsonOut), `"location":""`)
}

func TestEncodeTagsAndExternalDocs(t *testing.T) {
	doc := New()
	doc.Info = Info{Title: "Orders", Version: "1.0.0"}
	doc.Servers = map[string]*Server{
		"prod": {
			Host:     "broker:9092",
			Protocol: ProtocolKafka,
			Tags:     []Tag{{Name: "prod"}},
			ExternalDocs: &ExternalDocs{
				Description: "Server docs",
				URL:         "https://example.com/server",
			},
		},
	}
	doc.Channels = map[string]*Channel{
		"order-placed": {
			Address: "order-placed",
			Tags:    []Tag{{Name: "orders"}},
			ExternalDocs: &ExternalDocs{
				URL: "https://example.com/channel",
			},
		},
	}

	out, err := doc.YAML()
	require.NoError(t, err)
	for _, want := range []string{
		"tags:",
		"externalDocs:",
		"name: prod",
		"url: https://example.com/server",
		"url: https://example.com/channel",
	} {
		assert.Contains(t, string(out), want)
	}
}

// TestEncodeOmitsNonSpecFields pins the exported surface to AsyncAPI 3.1.0:
// the root object has exactly eight fields and License exactly two. The three
// fields removed by issue #20 must not reappear at the root or under `license`.
// Nested `tags:`/`externalDocs:` on servers, channels, and messages stay legal,
// so the assertions are scoped to the root and license maps — a nested key
// cannot mask a root regression.
func TestEncodeOmitsNonSpecFields(t *testing.T) {
	doc := New()
	doc.Info = Info{
		Title:   "Orders",
		Version: "1.0.0",
		License: &License{Name: "MIT", URL: "https://opensource.org/license/mit"},
	}
	doc.Servers = map[string]*Server{
		"prod": {
			Host:     "broker:9092",
			Protocol: ProtocolKafka,
			Tags:     []Tag{{Name: "prod"}},
			ExternalDocs: &ExternalDocs{
				URL: "https://example.com/server",
			},
		},
	}
	doc.Channels = map[string]*Channel{
		"order-placed": {Address: "order-placed"},
	}

	out, err := doc.YAML()
	require.NoError(t, err)

	var root map[string]any
	require.NoError(t, yaml.Unmarshal(out, &root))

	// Every root key has to be one of the eight 3.1.0 root-object fields.
	specRootFields := map[string]bool{
		"asyncapi": true, "id": true, "info": true, "servers": true,
		"defaultContentType": true, "channels": true, "operations": true,
		"components": true,
	}
	for key := range root {
		assert.Truef(t, specRootFields[key], "unexpected root field %q", key)
	}
	_, hasTags := root["tags"]
	assert.False(t, hasTags, "root must not emit tags")
	_, hasExternalDocs := root["externalDocs"]
	assert.False(t, hasExternalDocs, "root must not emit externalDocs")

	info, ok := root["info"].(map[string]any)
	require.True(t, ok, "info must decode to a map")
	license, ok := info["license"].(map[string]any)
	require.True(t, ok, "info.license must decode to a map")
	assert.Contains(t, license, "name")
	assert.Contains(t, license, "url")
	assert.Len(t, license, 2)
	_, hasIdentifier := license["identifier"]
	assert.False(t, hasIdentifier, "license must not emit identifier")

	// The nested, legal tags/externalDocs must still be emitted — proof the
	// fixture exercises them rather than passing vacuously.
	assert.Contains(t, string(out), "tags:")
	assert.Contains(t, string(out), "externalDocs:")
}

// specFields returns a struct type's JSON field names — the wire names — in
// declaration order. A field without a `json` tag falls back to its Go name.
func specFields(t *testing.T, v any) []string {
	t.Helper()
	typ := reflect.TypeOf(v)
	out := make([]string, 0, typ.NumField())
	for i := range typ.NumField() {
		f := typ.Field(i)
		name := f.Tag.Get("json")
		if i := strings.IndexByte(name, ','); i >= 0 {
			name = name[:i]
		}
		if name == "" {
			name = f.Name
		}
		out = append(out, name)
	}
	return out
}

// TestStructFieldsMatchSpec pins AsyncAPI and License to their 3.1.0 field
// lists in declaration order, mirroring the anchor-based derivation in
// docs/asyncapi-3.1.0-coverage.md §9. It is the structural counterpart to
// TestEncodeOmitsNonSpecFields: this catches a modeled-but-extra field, the
// encode test catches one that leaks into the wire format.
func TestStructFieldsMatchSpec(t *testing.T) {
	assert.Equal(t, []string{
		"asyncapi", "id", "info", "servers",
		"defaultContentType", "channels", "operations", "components",
	}, specFields(t, AsyncAPI{}))
	assert.Equal(t, []string{"name", "url"}, specFields(t, License{}))
}

// multiFormatDoc returns a document whose only component is a Multi Format
// Schema Object carrying an opaque Avro body.
func multiFormatDoc() *AsyncAPI {
	doc := New()
	doc.Info = Info{Title: "Orders", Version: "1.0.0"}
	doc.Components = &Components{
		Schemas: map[string]*Schema{
			"UserAvro": MultiFormat(
				"application/vnd.apache.avro;version=1.9.0",
				map[string]any{
					"type": "record",
					"name": "User",
					"fields": []any{
						map[string]any{"name": "displayName", "type": "string"},
					},
				},
			),
		},
	}
	return doc
}

// TestEncodeMultiFormatSchema covers the Multi Format Schema Object codec: the
// two fields are emitted verbatim in declaration order, the opaque body survives
// the harness round-trip (marshal -> unmarshal -> marshal), and a plain Schema
// Object is unaffected.
func TestEncodeMultiFormatSchema(t *testing.T) {
	t.Run("should_emit_schema_format_verbatim", func(t *testing.T) {
		out, err := multiFormatDoc().YAML()
		require.NoError(t, err)

		assert.Contains(t, string(out),
			"schemaFormat: application/vnd.apache.avro;version=1.9.0")
		assert.Contains(t, string(out), "name: User")
		assert.Contains(t, string(out), "type: record")
	})

	t.Run("should_emit_schema_format_before_schema", func(t *testing.T) {
		out, err := multiFormatDoc().YAML()
		require.NoError(t, err)

		formatAt := strings.Index(string(out), "schemaFormat:")
		bodyAt := strings.Index(string(out), "schema:")
		require.NotEqual(t, -1, formatAt)
		require.NotEqual(t, -1, bodyAt)
		assert.Less(t, formatAt, bodyAt)
	})

	t.Run("should_round_trip_opaque_body_through_yaml", func(t *testing.T) {
		first, err := multiFormatDoc().YAML()
		require.NoError(t, err)

		var decoded AsyncAPI
		require.NoError(t, yaml.Unmarshal(first, &decoded))

		second, err := decoded.YAML()
		require.NoError(t, err)
		assert.Equal(t, string(first), string(second),
			"the harness round-trip must be byte-stable")

		body, ok := decoded.Components.Schemas["UserAvro"].Schema.(map[string]any)
		require.True(t, ok, "the opaque body must decode to a map")
		assert.Equal(t, "record", body["type"])
		assert.Equal(t, "User", body["name"])
	})

	t.Run("should_round_trip_multi_format_through_json", func(t *testing.T) {
		out, err := multiFormatDoc().JSON()
		require.NoError(t, err)

		var decoded AsyncAPI
		require.NoError(t, json.Unmarshal(out, &decoded))

		schema := decoded.Components.Schemas["UserAvro"]
		require.NotNil(t, schema)
		assert.Equal(t, "application/vnd.apache.avro;version=1.9.0", schema.SchemaFormat)
		assert.Equal(t, canonicalJSON(t, multiFormatDoc()), canonicalJSON(t, &decoded))
	})

	t.Run("should_leave_plain_schema_encoding_unchanged", func(t *testing.T) {
		doc := New()
		doc.Info = Info{Title: "Orders", Version: "1.0.0"}
		doc.Components = &Components{
			Schemas: map[string]*Schema{
				"Order": {
					Type:       "object",
					Properties: map[string]*Schema{"id": {Type: "string"}},
					Required:   []string{"id"},
				},
			},
		}

		out, err := doc.YAML()
		require.NoError(t, err)

		assert.NotContains(t, string(out), "schemaFormat")
		assert.Equal(t, `asyncapi: 3.1.0
info:
  title: Orders
  version: 1.0.0
components:
  schemas:
    Order:
      type: object
      properties:
        id:
          type: string
      required:
      - id
`, string(out))
	})

	t.Run("should_emit_a_string_body_verbatim", func(t *testing.T) {
		// A Protobuf payload is a .proto string rather than a map. Protobuf is
		// emitted verbatim like any other format; the pinned AsyncAPI CLI cannot
		// validate it (no parser is registered for it), so this unit test is the
		// coverage for the string-shaped opaque body.
		doc := New()
		doc.Info = Info{Title: "Orders", Version: "1.0.0"}
		doc.Components = &Components{
			Schemas: map[string]*Schema{
				"OrderProto": MultiFormat(
					"application/vnd.google.protobuf;version=3",
					"message Order { string order_id = 1; }",
				),
			},
		}

		out, err := doc.YAML()
		require.NoError(t, err)

		var decoded AsyncAPI
		require.NoError(t, yaml.Unmarshal(out, &decoded))
		assert.Equal(t,
			"application/vnd.google.protobuf;version=3",
			decoded.Components.Schemas["OrderProto"].SchemaFormat)
		assert.Equal(t,
			"message Order { string order_id = 1; }",
			decoded.Components.Schemas["OrderProto"].Schema)
	})
}

// schemaKeywordDoc returns a document whose only schema is an object with set
// applied — the fixture every keyword round-trip case starts from.
func schemaKeywordDoc(set func(*Schema)) *AsyncAPI {
	s := &Schema{Type: "object"}
	set(s)

	doc := New()
	doc.Info = Info{Title: "Orders", Version: "1.0.0"}
	doc.Components = &Components{Schemas: map[string]*Schema{"Schema": s}}
	return doc
}

// decodedSchema returns the fixture schema after a round-trip through a codec.
func decodedSchema(t *testing.T, doc *AsyncAPI) *Schema {
	t.Helper()
	require.NotNil(t, doc.Components)
	s := doc.Components.Schemas["Schema"]
	require.NotNil(t, s, "the round-tripped document must still carry the schema")
	return s
}

// TestEncodeSchemaKeywords covers every Schema keyword this library models: for
// each one the fixture document must emit the exact wire keyword, decode it
// back, and survive the YAML round-trip byte-identically — the invariant the
// one-struct spec.Schema design exists to protect, since the discovery harness
// materializes every document through YAML.
func TestEncodeSchemaKeywords(t *testing.T) {
	u64 := func(v uint64) *uint64 { return &v }

	tt := []struct {
		name   string
		wire   string
		set    func(*Schema)
		verify func(*testing.T, *Schema)
	}{
		{
			name: "should_round_trip_id",
			wire: "$id",
			set:  func(s *Schema) { s.ID = "https://example.com/order.schema.json" },
			verify: func(t *testing.T, s *Schema) {
				assert.Equal(t, "https://example.com/order.schema.json", s.ID)
			},
		},
		{
			name: "should_round_trip_schema_uri",
			wire: "$schema",
			set:  func(s *Schema) { s.SchemaURI = "http://json-schema.org/draft-07/schema#" },
			verify: func(t *testing.T, s *Schema) {
				assert.Equal(t, "http://json-schema.org/draft-07/schema#", s.SchemaURI)
			},
		},
		{
			name: "should_round_trip_comment",
			wire: "$comment",
			set:  func(s *Schema) { s.Comment = "hand-authored" },
			verify: func(t *testing.T, s *Schema) {
				assert.Equal(t, "hand-authored", s.Comment)
			},
		},
		{
			name: "should_round_trip_external_docs",
			wire: "externalDocs",
			set: func(s *Schema) {
				s.ExternalDocs = &ExternalDocs{URL: "https://example.com/schema"}
			},
			verify: func(t *testing.T, s *Schema) {
				require.NotNil(t, s.ExternalDocs)
				assert.Equal(t, "https://example.com/schema", s.ExternalDocs.URL)
			},
		},
		{
			name: "should_round_trip_deprecated",
			wire: "deprecated",
			set:  func(s *Schema) { s.Deprecated = true },
			verify: func(t *testing.T, s *Schema) {
				assert.True(t, s.Deprecated)
			},
		},
		{
			name: "should_round_trip_additional_items",
			wire: "additionalItems",
			set:  func(s *Schema) { s.AdditionalItems = &Schema{Type: "string"} },
			verify: func(t *testing.T, s *Schema) {
				require.NotNil(t, s.AdditionalItems)
				assert.Equal(t, "string", s.AdditionalItems.Type)
			},
		},
		{
			name: "should_round_trip_pattern_properties",
			wire: "patternProperties",
			set: func(s *Schema) {
				s.PatternProperties = map[string]*Schema{"^x-": {Type: "string"}}
			},
			verify: func(t *testing.T, s *Schema) {
				require.Contains(t, s.PatternProperties, "^x-")
				assert.Equal(t, "string", s.PatternProperties["^x-"].Type)
			},
		},
		{
			name: "should_round_trip_property_names",
			wire: "propertyNames",
			set:  func(s *Schema) { s.PropertyNames = &Schema{Pattern: "^[a-z]+$"} },
			verify: func(t *testing.T, s *Schema) {
				require.NotNil(t, s.PropertyNames)
				assert.Equal(t, "^[a-z]+$", s.PropertyNames.Pattern)
			},
		},
		{
			name: "should_round_trip_dependencies",
			wire: "dependencies",
			set: func(s *Schema) {
				s.Dependencies = map[string]*Schema{
					"creditCard": {Required: []string{"billing"}},
				}
			},
			verify: func(t *testing.T, s *Schema) {
				require.Contains(t, s.Dependencies, "creditCard")
				assert.Equal(t, []string{"billing"}, s.Dependencies["creditCard"].Required)
			},
		},
		{
			name: "should_round_trip_contains",
			wire: "contains",
			set:  func(s *Schema) { s.Contains = &Schema{Type: "string"} },
			verify: func(t *testing.T, s *Schema) {
				require.NotNil(t, s.Contains)
				assert.Equal(t, "string", s.Contains.Type)
			},
		},
		{
			name: "should_round_trip_const",
			wire: "const",
			set:  func(s *Schema) { s.Const = "OrderPlaced" },
			verify: func(t *testing.T, s *Schema) {
				assert.Equal(t, "OrderPlaced", s.Const)
			},
		},
		{
			name: "should_round_trip_examples",
			wire: "examples",
			set:  func(s *Schema) { s.Examples = []any{"sku-1", "sku-2"} },
			verify: func(t *testing.T, s *Schema) {
				assert.Equal(t, []any{"sku-1", "sku-2"}, s.Examples)
			},
		},
		{
			name: "should_round_trip_if",
			wire: "if",
			set: func(s *Schema) {
				s.If = &Schema{Properties: map[string]*Schema{"kind": {Const: "a"}}}
			},
			verify: func(t *testing.T, s *Schema) {
				require.NotNil(t, s.If)
				require.Contains(t, s.If.Properties, "kind")
				assert.Equal(t, "a", s.If.Properties["kind"].Const)
			},
		},
		{
			name: "should_round_trip_then",
			wire: "then",
			set:  func(s *Schema) { s.Then = &Schema{Required: []string{"a"}} },
			verify: func(t *testing.T, s *Schema) {
				require.NotNil(t, s.Then)
				assert.Equal(t, []string{"a"}, s.Then.Required)
			},
		},
		{
			name: "should_round_trip_else",
			wire: "else",
			set:  func(s *Schema) { s.Else = &Schema{Required: []string{"b"}} },
			verify: func(t *testing.T, s *Schema) {
				require.NotNil(t, s.Else)
				assert.Equal(t, []string{"b"}, s.Else.Required)
			},
		},
		{
			name: "should_round_trip_min_items",
			wire: "minItems",
			set:  func(s *Schema) { s.MinItems = u64(1) },
			verify: func(t *testing.T, s *Schema) {
				require.NotNil(t, s.MinItems)
				assert.Equal(t, uint64(1), *s.MinItems)
			},
		},
		{
			name: "should_round_trip_max_items",
			wire: "maxItems",
			set:  func(s *Schema) { s.MaxItems = u64(10) },
			verify: func(t *testing.T, s *Schema) {
				require.NotNil(t, s.MaxItems)
				assert.Equal(t, uint64(10), *s.MaxItems)
			},
		},
		{
			name: "should_round_trip_unique_items",
			wire: "uniqueItems",
			set:  func(s *Schema) { s.UniqueItems = true },
			verify: func(t *testing.T, s *Schema) {
				assert.True(t, s.UniqueItems)
			},
		},
		{
			name: "should_round_trip_min_properties",
			wire: "minProperties",
			set:  func(s *Schema) { s.MinProperties = u64(1) },
			verify: func(t *testing.T, s *Schema) {
				require.NotNil(t, s.MinProperties)
				assert.Equal(t, uint64(1), *s.MinProperties)
			},
		},
		{
			name: "should_round_trip_max_properties",
			wire: "maxProperties",
			set:  func(s *Schema) { s.MaxProperties = u64(8) },
			verify: func(t *testing.T, s *Schema) {
				require.NotNil(t, s.MaxProperties)
				assert.Equal(t, uint64(8), *s.MaxProperties)
			},
		},
		{
			name: "should_round_trip_read_only",
			wire: "readOnly",
			set:  func(s *Schema) { s.ReadOnly = true },
			verify: func(t *testing.T, s *Schema) {
				assert.True(t, s.ReadOnly)
			},
		},
		{
			name: "should_round_trip_write_only",
			wire: "writeOnly",
			set:  func(s *Schema) { s.WriteOnly = true },
			verify: func(t *testing.T, s *Schema) {
				assert.True(t, s.WriteOnly)
			},
		},
		{
			name: "should_round_trip_discriminator",
			wire: "discriminator",
			set:  func(s *Schema) { s.Discriminator = "kind" },
			verify: func(t *testing.T, s *Schema) {
				assert.Equal(t, "kind", s.Discriminator)
			},
		},
	}

	require.Len(t, tt, 23, "every modeled Schema keyword needs a case")

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			doc := schemaKeywordDoc(tc.set)

			yamlOut, err := doc.YAML()
			require.NoError(t, err)
			assert.Contains(t, string(yamlOut), tc.wire+":", "the wire keyword must be emitted")

			var fromYAML AsyncAPI
			require.NoError(t, yaml.Unmarshal(yamlOut, &fromYAML))
			tc.verify(t, decodedSchema(t, &fromYAML))

			// The harness materializes through YAML: a non-byte-stable round-trip
			// would rewrite committed golden files.
			yamlAgain, err := fromYAML.YAML()
			require.NoError(t, err)
			assert.Equal(t, string(yamlOut), string(yamlAgain),
				"the YAML round-trip must be byte-stable")

			jsonOut, err := doc.JSON()
			require.NoError(t, err)
			assert.Contains(t, string(jsonOut), `"`+tc.wire+`":`,
				"the wire keyword must be emitted in JSON too")

			var fromJSON AsyncAPI
			require.NoError(t, json.Unmarshal(jsonOut, &fromJSON))
			tc.verify(t, decodedSchema(t, &fromJSON))
		})
	}
}

// TestEncodeSchemaDiscriminatorIsAScalar pins the AsyncAPI 3.1.0 shape: the
// discriminator is the *name* of the property that differentiates the schemas,
// not the OpenAPI Discriminator Object that wraps it.
func TestEncodeSchemaDiscriminatorIsAScalar(t *testing.T) {
	doc := schemaKeywordDoc(func(s *Schema) {
		s.Properties = map[string]*Schema{"kind": {Type: "string"}}
		s.Required = []string{"kind"}
		s.Discriminator = "kind"
	})

	out, err := doc.YAML()
	require.NoError(t, err)

	assert.Contains(t, string(out), "discriminator: kind")
	assert.NotContains(t, string(out), "propertyName",
		"discriminator is a plain string in AsyncAPI 3.1.0, not the OpenAPI object")
}

// TestEncodeSchemaExamples covers the one example keyword 3.1.0 defines. The 2.x
// singular `example` was removed as a spec deviation, so nothing in an emitted
// document may reintroduce that key.
func TestEncodeSchemaExamples(t *testing.T) {
	doc := schemaKeywordDoc(func(s *Schema) {
		s.Examples = []any{"draft-07", 7}
	})

	out, err := doc.YAML()
	require.NoError(t, err)
	assert.Contains(t, string(out), "examples:")
	assert.NotContains(t, string(out), "example:",
		"the removed singular `example` keyword must not be emitted")

	var decoded AsyncAPI
	require.NoError(t, yaml.Unmarshal(out, &decoded))

	// Examples is []any, so a decoded number is uint64 where the source was int;
	// canonicalJSON is how the rest of this file compares any-valued documents.
	assert.Equal(t, canonicalJSON(t, doc), canonicalJSON(t, &decoded))

	s := decodedSchema(t, &decoded)
	require.Len(t, s.Examples, 2)
	assert.Equal(t, "draft-07", s.Examples[0])
}

// TestSchemaFieldsMatchSpec pins spec.Schema's wire names in declaration order,
// mirroring TestStructFieldsMatchSpec for AsyncAPI and License. Declaration
// order is emitted key order, so a reshuffle here silently rewrites documents;
// this is the structural counterpart to TestEncodeSchemaKeywords.
func TestSchemaFieldsMatchSpec(t *testing.T) {
	assert.Equal(t, []string{
		"schemaFormat", "schema",
		"$ref", "$id", "$schema",
		"type", "title", "description", "format", "$comment", "externalDocs", "deprecated",
		"properties", "required", "items", "additionalProperties", "additionalItems",
		"patternProperties", "propertyNames", "dependencies", "contains",
		"enum", "const", "examples", "default",
		"definitions", "oneOf", "allOf", "anyOf", "not", "if", "then", "else",
		"minLength", "maxLength", "pattern",
		"minimum", "maximum", "exclusiveMinimum", "exclusiveMaximum", "multipleOf",
		"minItems", "maxItems", "uniqueItems",
		"minProperties", "maxProperties",
		"readOnly", "writeOnly",
		"discriminator",
	}, specFields(t, Schema{}))
}
