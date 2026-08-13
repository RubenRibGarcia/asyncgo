package spec

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOverlay(t *testing.T) {
	base := New()
	base.Info = Info{Title: "Base", Version: "1.0.0", Description: "base desc"}
	base.Servers = map[string]*Server{
		"prod": {Host: "h", Protocol: ProtocolKafka},
	}
	base.Channels = map[string]*Channel{
		"a": {Address: "a", Description: "base channel"},
	}

	overlay := New()
	overlay.Info = Info{Title: "Overlay"} // only title set
	overlay.Servers = map[string]*Server{
		"prod": {Description: "from fragment"},
		"dev":  {Host: "localhost", Protocol: ProtocolNATS},
	}
	overlay.Channels = map[string]*Channel{
		"a": {Description: "overlay channel"},
		"b": {Address: "b"},
	}

	merged, err := Overlay(base, overlay)
	require.NoError(t, err)

	// Overlay wins on scalar; base preserved elsewhere.
	assert.Equal(t, Version, merged.AsyncAPI)
	assert.Equal(t, "Overlay", merged.Info.Title)
	assert.Equal(t, "1.0.0", merged.Info.Version)
	assert.Equal(t, "base desc", merged.Info.Description)

	// Servers: prod deep-merged (host/protocol kept, description overlaid), dev added.
	require.Contains(t, merged.Servers, "prod")
	prod := merged.Servers["prod"]
	assert.Equal(t, "h", prod.Host)
	assert.Equal(t, ProtocolKafka, prod.Protocol)
	assert.Equal(t, "from fragment", prod.Description)

	require.Contains(t, merged.Servers, "dev")
	assert.Equal(t, ProtocolNATS, merged.Servers["dev"].Protocol)

	// Channels: a deep-merged, b added.
	require.Contains(t, merged.Channels, "a")
	assert.Equal(t, "overlay channel", merged.Channels["a"].Description)
	assert.Equal(t, "a", merged.Channels["a"].Address)

	require.Contains(t, merged.Channels, "b")
	assert.Equal(t, "b", merged.Channels["b"].Address)
}

func TestOverlayDoesNotMutateBase(t *testing.T) {
	base := New()
	base.Info = Info{Title: "Base", Version: "1.0.0"}
	overlay := New()
	overlay.Info = Info{Title: "Overlay"}

	_, err := Overlay(base, overlay)
	require.NoError(t, err)
	assert.Equal(t, "Base", base.Info.Title)
}
