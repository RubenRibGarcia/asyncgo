package asyncgo

import (
	"testing"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestSecuritySchemeValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		spec func() *SpecResult
		want string
	}{
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
			name: "should_reject_a_security_scheme_name_outside_the_key_pattern",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					SecuritySchemes(
						SecurityScheme("oauth basic", spec.SecurityScheme{Type: "http"}),
					),
				)
			},
			want: "securityScheme.oauth basic: name must match ^[\\w\\d.\\-_]+$",
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
