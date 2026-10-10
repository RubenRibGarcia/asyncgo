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
