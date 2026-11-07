package spec

import (
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBindingRoundTripIsNotByteStable pins the one place the "YAML round-trip
// must be byte-stable" invariant does not hold.
//
// Schema and Message use one concrete struct per union, which keeps the
// harness's decode/re-encode lossless. A binding cannot: the *Bindings maps are
// map[string]any by design (the decode path needs any), so a typed binding struct
// marshals in declaration order, while the same binding decoded back into a map
// re-encodes with sorted keys. The divergence is deliberate and bounded: after
// the first crossing the document is stable, which is why the committed fixtures
// (generated through the harness's YAML hop) are sorted rather than
// declaration-ordered.
func TestBindingRoundTripIsNotByteStable(t *testing.T) {
	built := New()
	built.Info = Info{Title: "orders", Version: "1.0.0"}
	built.Servers = map[string]*Server{
		"prod": {
			Host:     "broker.ibmmq:1414",
			Protocol: ProtocolIBMMQ,
			Bindings: ServerBindings{ProtocolIBMMQ: IBMMQServerBinding{
				GroupID:              "orders",
				CCDTQueueManagerName: "QM1",
				BindingVersion:       "0.1.0",
			}},
		},
	}

	fromStruct, err := built.YAML()
	require.NoError(t, err)
	assert.Less(t,
		indexOf(t, fromStruct, "groupId:"),
		indexOf(t, fromStruct, "bindingVersion:"),
		"a typed binding struct marshals in declaration order",
	)

	var decoded AsyncAPI
	require.NoError(t, yaml.Unmarshal(fromStruct, &decoded))

	fromMap, err := decoded.YAML()
	require.NoError(t, err)
	assert.NotEqual(t, string(fromStruct), string(fromMap),
		"decoding a binding into a map and re-encoding sorts its keys",
	)
	assert.Less(t,
		indexOf(t, fromMap, "bindingVersion:"),
		indexOf(t, fromMap, "ccdtQueueManagerName:"),
		"the re-encoded binding is sorted by key",
	)

	// The divergence is bounded to the first crossing: a second decode/encode
	// changes nothing.
	var decodedAgain AsyncAPI
	require.NoError(t, yaml.Unmarshal(fromMap, &decodedAgain))
	fromMapAgain, err := decodedAgain.YAML()
	require.NoError(t, err)
	assert.Equal(t, string(fromMap), string(fromMapAgain))
}

// indexOf returns the offset of sub in b, failing the test when it is absent.
func indexOf(t *testing.T, b []byte, sub string) int {
	t.Helper()

	i := strings.Index(string(b), sub)
	require.GreaterOrEqual(t, i, 0, "%q not found in the encoded document", sub)
	return i
}
