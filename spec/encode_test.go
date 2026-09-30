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
