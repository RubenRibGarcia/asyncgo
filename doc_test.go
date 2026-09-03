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
	s := Server("prod", "kafka", "broker:9092").
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
	require.NoError(t, c.apply(b))

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

func TestValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		spec func() *SpecResult
		want string
	}{
		{
			name: "should_return_error_when_server_host_is_empty",
			spec: func() *SpecResult {
				return Spec(Info("Orders", "1.0.0"), Servers(Server("prod", "kafka", "")))
			},
			want: "server.prod.host: is required",
		},
		{
			name: "should_return_error_when_server_protocol_is_empty",
			spec: func() *SpecResult {
				return Spec(Info("Orders", "1.0.0"), Servers(Server("prod", "", "broker:9092")))
			},
			want: "server.prod.protocol: is required",
		},
		{
			name: "should_return_error_when_server_name_is_empty",
			spec: func() *SpecResult {
				return Spec(Info("Orders", "1.0.0"), Servers(Server("", "kafka", "broker:9092")))
			},
			want: "server.name: is required",
		},
		{
			name: "should_return_error_when_info_title_is_missing",
			spec: func() *SpecResult {
				return Spec(Info("", "1.0.0"))
			},
			want: "info.title: is required",
		},
		{
			name: "should_return_error_when_info_version_is_missing",
			spec: func() *SpecResult {
				return Spec(Info("Orders", ""))
			},
			want: "info.version: is required",
		},
		{
			name: "should_join_multiple_validation_errors",
			spec: func() *SpecResult {
				return Spec(
					Info("", ""),
					Servers(Server("prod", "", "")),
				)
			},
			want: "info.title: is required\ninfo.version: is required\nserver.prod.protocol: is required\nserver.prod.host: is required",
		},
		{
			name: "should_return_nil_error_for_valid_catalog",
			spec: func() *SpecResult {
				return Spec(Info("Orders", "1.0.0"), Servers(Server("prod", "kafka", "broker:9092")))
			},
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := tc.spec()
			if tc.want == "" {
				require.NoError(t, res.Err)
				assert.Nil(t, res.ValidationErrors())
				return
			}
			require.Error(t, res.Err)
			assert.Equal(t, tc.want, res.Err.Error())
		})
	}
}
