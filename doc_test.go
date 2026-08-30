package asyncgo

import (
	"testing"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInfoFields(t *testing.T) {
	i := Info("Orders", "1.0.0").
		Description("desc").
		TermsOfService("tos").
		Contact(spec.Contact{Name: "Orders Team", Email: "orders@example.com"}).
		License(spec.License{Name: "MIT"}).
		Tags(spec.Tag{Name: "orders"})

	assert.Equal(t, "desc", i.info.Description)
	assert.Equal(t, "tos", i.info.TermsOfService)
	require.NotNil(t, i.info.Contact)
	assert.Equal(t, "Orders Team", i.info.Contact.Name)
	require.NotNil(t, i.info.License)
	assert.Equal(t, "MIT", i.info.License.Name)
	require.Len(t, i.info.Tags, 1)
	assert.Equal(t, "orders", i.info.Tags[0].Name)
}

func TestServerFields(t *testing.T) {
	s := Server("prod", "kafka").
		ProtocolVersion("1.0").
		Description("desc").
		Variable("host", spec.ServerVariable{Default: "broker"})

	assert.Equal(t, "1.0", s.s.ProtocolVersion)
	assert.Equal(t, "desc", s.s.Description)
	require.NotNil(t, s.s.Variables)
	assert.Equal(t, "broker", s.s.Variables["host"].Default)

	// A second variable does not reset the map.
	s.Variable("port", spec.ServerVariable{Default: "9092"})
	assert.Len(t, s.s.Variables, 2)
}

func TestChannelReceive(t *testing.T) {
	c := Channel("order-placed").
		Title("Order placed").
		Receive(Operation().Message(MessageOf(OrderPlaced{}).Name("OrderPlaced")))

	assert.Equal(t, "Order placed", c.s.Title)

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	c.apply(b)

	require.Contains(t, b.doc.Operations, "order-placed.receive")
	op := b.doc.Operations["order-placed.receive"]
	assert.Equal(t, spec.ActionReceive, op.Action)
	assert.Equal(t, "#/channels/order-placed", op.Channel.Ref)
	require.Contains(t, b.doc.Channels["order-placed"].Messages, "OrderPlaced")
}

func TestOperationFields(t *testing.T) {
	o := Operation().
		Title("t").
		Summary("s").
		Description("d").
		Message(MessageOf(OrderPlaced{}))

	assert.Equal(t, "t", o.title)
	assert.Equal(t, "s", o.summary)
	assert.Equal(t, "d", o.description)
	assert.Len(t, o.messages, 1)
}
