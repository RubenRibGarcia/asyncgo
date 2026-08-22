package spec

import (
	"testing"

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
