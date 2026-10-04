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
			name: "should_return_error_when_server_name_is_duplicate",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Servers(
						Server("prod", "kafka", "broker:9092"),
						Server("prod", "kafka", "broker-staging:9092"),
					),
				)
			},
			want: "server.prod: duplicate name",
		},
		{
			name: "should_return_error_when_server_name_is_duplicate_across_servers_calls",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Servers(Server("prod", "kafka", "broker:9092")),
					Servers(Server("prod", "kafka", "broker-staging:9092")),
				)
			},
			want: "server.prod: duplicate name",
		},
		{
			name: "should_return_error_when_channel_address_is_duplicate",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(
						Channel("order-placed"),
						Channel("order-placed"),
					),
				)
			},
			want: "channel.order-placed: duplicate address",
		},
		{
			name: "should_return_error_when_channel_address_is_duplicate_across_channels_calls",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(Channel("order-placed")),
					Channels(Channel("order-placed")),
				)
			},
			want: "channel.order-placed: duplicate address",
		},
		{
			name: "should_join_duplicate_and_field_validation_errors",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Servers(
						Server("prod", "kafka", "broker:9092"),
						Server("prod", "kafka", ""),
					),
				)
			},
			want: "server.prod.host: is required\nserver.prod: duplicate name",
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
				return Spec(
					Info("Orders", "1.0.0"),
					Servers(Server("prod", "kafka", "broker:9092")),
				)
			},
			want: "",
		},
		{
			name: "should_return_error_when_message_payload_type_is_nil",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(
						Channel("order-placed").
							Send(Operation().Message(MessageOf(nil))),
					),
				)
			},
			want: "message: nil payload type",
		},
		{
			name: "should_return_error_when_channel_references_unknown_server",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(
						Channel("order-placed").
							Servers(Server("prod", "kafka", "broker:9092")),
					),
				)
			},
			want: `channel.order-placed: references unknown server "prod"`,
		},
		{
			name: "should_join_multiple_unknown_server_references",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(
						Channel("order-placed").
							Servers(
								Server("prod", "kafka", "broker:9092"),
								Server("staging", "kafka", "broker-staging:9092"),
							),
					),
				)
			},
			want: "channel.order-placed: references unknown server \"prod\"\n" +
				"channel.order-placed: references unknown server \"staging\"",
		},
		{
			name: "should_allow_channel_to_reference_server_declared_after_it",
			spec: func() *SpecResult {
				prod := Server("prod", "kafka", "broker:9092")
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(Channel("order-placed").Servers(prod)),
					Servers(prod),
				)
			},
			want: "",
		},
		{
			name: "should_return_error_when_security_scheme_name_is_empty",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					SecuritySchemes(SecurityScheme("", spec.SecurityScheme{Type: "userPassword"})),
				)
			},
			want: "securityScheme.name: is required",
		},
		{
			name: "should_return_error_when_security_scheme_type_is_empty",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					SecuritySchemes(SecurityScheme("oauth", spec.SecurityScheme{})),
				)
			},
			want: "securityScheme.oauth.type: is required",
		},
		{
			name: "should_return_error_when_security_scheme_name_is_duplicate",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					SecuritySchemes(
						SecurityScheme("oauth", spec.SecurityScheme{Type: "oauth2"}),
						SecurityScheme("oauth", spec.SecurityScheme{Type: "http"}),
					),
				)
			},
			want: "securityScheme.oauth: duplicate name",
		},
		{
			name: "should_return_error_when_server_references_unknown_security_scheme",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Servers(
						Server("prod", "kafka", "broker:9092").
							Security(SecurityScheme("oauth", spec.SecurityScheme{Type: "oauth2"})),
					),
				)
			},
			want: `server.prod: references unknown security scheme "oauth"`,
		},
		{
			name: "should_return_error_when_operation_references_unknown_security_scheme",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(
						Channel("order-placed").
							Send(Operation().
								Security(SecurityScheme("oauth", spec.SecurityScheme{Type: "oauth2"})),
							),
					),
				)
			},
			want: `operation.order-placed.send: references unknown security scheme "oauth"`,
		},
		{
			name: "should_allow_security_scheme_declared_after_it_is_referenced",
			spec: func() *SpecResult {
				oauth := SecurityScheme("oauth", spec.SecurityScheme{Type: "oauth2"})
				return Spec(
					Info("Orders", "1.0.0"),
					Servers(Server("prod", "kafka", "broker:9092").Security(oauth)),
					SecuritySchemes(oauth),
				)
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

func TestDuplicateKeepsFirstEntry(t *testing.T) {
	t.Run("should_keep_first_server_on_duplicate_name", func(t *testing.T) {
		res := Spec(
			Info("Orders", "1.0.0"),
			Servers(
				Server("prod", "kafka", "broker:9092"),
				Server("prod", "amqp", "broker-staging:5672"),
			),
		)
		require.Error(t, res.Err)
		require.Contains(t, res.Doc.Servers, "prod")
		assert.Equal(t, "kafka", res.Doc.Servers["prod"].Protocol)
		assert.Equal(t, "broker:9092", res.Doc.Servers["prod"].Host)
	})

	t.Run("should_keep_first_channel_on_duplicate_address", func(t *testing.T) {
		res := Spec(
			Info("Orders", "1.0.0"),
			Channels(
				Channel("order-placed").Title("First"),
				Channel("order-placed").Title("Second"),
			),
		)
		require.Error(t, res.Err)
		require.Contains(t, res.Doc.Channels, "order-placed")
		assert.Equal(t, "First", res.Doc.Channels["order-placed"].Title)
	})

	t.Run("should_keep_first_security_scheme_on_duplicate_name", func(t *testing.T) {
		res := Spec(
			Info("Orders", "1.0.0"),
			SecuritySchemes(
				SecurityScheme("oauth", spec.SecurityScheme{Type: "oauth2"}),
				SecurityScheme("oauth", spec.SecurityScheme{Type: "http"}),
			),
		)
		require.Error(t, res.Err)
		require.NotNil(t, res.Doc.Components)
		require.Contains(t, res.Doc.Components.SecuritySchemes, "oauth")
		assert.Equal(t, "oauth2", res.Doc.Components.SecuritySchemes["oauth"].Type)
	})
}

func TestSecuritySchemeFields(t *testing.T) {
	sc := SecurityScheme("oauth", spec.SecurityScheme{
		Type:             "oauth2",
		Description:      "OAuth 2.0",
		Name:             "Authorization",
		In:               "header",
		Scheme:           "bearer",
		BearerFormat:     "JWT",
		OpenIDConnectURL: "https://example.com/.well-known/openid-configuration",
		Scopes:           []string{"read:orders"},
		Flows: &spec.OAuthFlows{
			ClientCredentials: &spec.OAuthFlow{
				TokenURL:        "https://example.com/oauth/token",
				AvailableScopes: map[string]string{"read:orders": "read orders"},
			},
		},
	})

	assert.Equal(t, "oauth", sc.name)
	assert.Equal(t, "oauth2", sc.s.Type)
	assert.Equal(t, "header", sc.s.In)
	assert.Equal(t, "bearer", sc.s.Scheme)
	assert.Equal(t, "JWT", sc.s.BearerFormat)
	assert.Equal(t, []string{"read:orders"}, sc.s.Scopes)
	require.NotNil(t, sc.s.Flows)
	require.NotNil(t, sc.s.Flows.ClientCredentials)
	assert.Equal(t, "https://example.com/oauth/token", sc.s.Flows.ClientCredentials.TokenURL)
	assert.Equal(t, "read orders", sc.s.Flows.ClientCredentials.AvailableScopes["read:orders"])
}

func TestSecuritySchemesRegistersComponents(t *testing.T) {
	res := Spec(
		Info("Orders", "1.0.0"),
		SecuritySchemes(
			SecurityScheme("userPasswordAuth", spec.SecurityScheme{Type: "userPassword"}),
			SecurityScheme("apiKeyAuth", spec.SecurityScheme{Type: "apiKey", In: "user"}),
		),
	)

	require.NoError(t, res.Err)
	require.NotNil(t, res.Doc.Components)
	assert.Len(t, res.Doc.Components.SecuritySchemes, 2)
	assert.Equal(t, "userPassword", res.Doc.Components.SecuritySchemes["userPasswordAuth"].Type)
	assert.Equal(t, "user", res.Doc.Components.SecuritySchemes["apiKeyAuth"].In)
}

func TestServerSecurity(t *testing.T) {
	oauth := SecurityScheme("oauth", spec.SecurityScheme{Type: "oauth2"})

	s := Server("prod", "kafka", "broker:9092").Security(oauth)

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	require.NoError(t, Servers(s).apply(b))

	require.Len(t, b.doc.Servers["prod"].Security, 1)
	assert.Equal(t, "#/components/securitySchemes/oauth", b.doc.Servers["prod"].Security[0].Ref)
}

func TestServerSecurityAppendsAcrossCalls(t *testing.T) {
	oauth := SecurityScheme("oauth", spec.SecurityScheme{Type: "oauth2"})
	basic := SecurityScheme("basic", spec.SecurityScheme{Type: "http", Scheme: "basic"})

	s := Server("prod", "kafka", "broker:9092").Security(oauth).Security(basic)

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	require.NoError(t, Servers(s).apply(b))

	require.Len(t, b.doc.Servers["prod"].Security, 2)
	assert.Equal(t, "#/components/securitySchemes/oauth", b.doc.Servers["prod"].Security[0].Ref)
	assert.Equal(t, "#/components/securitySchemes/basic", b.doc.Servers["prod"].Security[1].Ref)
}

func TestSecuritySchemeRefEscapesPointer(t *testing.T) {
	sc := SecurityScheme("tenant/oauth~prod", spec.SecurityScheme{Type: "oauth2"})

	s := Server("prod", "kafka", "broker:9092").Security(sc)

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	require.NoError(t, Servers(s).apply(b))

	assert.Equal(
		t,
		"#/components/securitySchemes/tenant~1oauth~0prod",
		b.doc.Servers["prod"].Security[0].Ref,
	)
}

func TestOperationSecurity(t *testing.T) {
	basic := SecurityScheme("basic", spec.SecurityScheme{Type: "http", Scheme: "basic"})

	c := Channel("order-placed").
		Send(Operation().
			Security(basic).
			Message(MessageOf(OrderPlaced{}).Name("OrderPlaced")))

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	require.NoError(t, c.apply(b))

	require.Contains(t, b.doc.Operations, "order-placed.send")
	require.Len(t, b.doc.Operations["order-placed.send"].Security, 1)
	assert.Equal(
		t,
		"#/components/securitySchemes/basic",
		b.doc.Operations["order-placed.send"].Security[0].Ref,
	)
}
