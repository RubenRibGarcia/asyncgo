package asyncgo

import (
	"testing"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestServerTagsAndExternalDocs(t *testing.T) {
	s := Server("prod", "kafka", "broker:9092").
		Tags(spec.Tag{Name: "prod"}).
		ExternalDocs(spec.ExternalDocs{URL: "https://example.com/prod"})

	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	require.NoError(t, Servers(s).apply(b))

	assert.Equal(t, []spec.Tag{{Name: "prod"}}, b.doc.Servers["prod"].Tags)
	require.NotNil(t, b.doc.Servers["prod"].ExternalDocs)
	assert.Equal(t, "https://example.com/prod", b.doc.Servers["prod"].ExternalDocs.URL)
}

func TestServerValidationErrors(t *testing.T) {
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
