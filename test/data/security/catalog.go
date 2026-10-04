package security

import (
	"github.com/RubenRibGarcia/asyncgo"
	"github.com/RubenRibGarcia/asyncgo/spec"
)

// The five security scheme types, one per AsyncAPI 3.1.0 shape: a bare
// userPassword, an apiKey whose location is required, an http scheme, an
// oauth2 scheme whose flows carry availableScopes, and an openIdConnect scheme
// whose discovery URL is required.
var (
	userPasswordAuth = asyncgo.SecurityScheme("userPasswordAuth", spec.SecurityScheme{
		Type:        "userPassword",
		Description: "Username and password over the broker connection",
	})

	apiKeyAuth = asyncgo.SecurityScheme("apiKeyAuth", spec.SecurityScheme{
		Type: "apiKey",
		In:   "user",
	})

	basicAuth = asyncgo.SecurityScheme("basicAuth", spec.SecurityScheme{
		Type:   "http",
		Scheme: "basic",
	})

	oauth = asyncgo.SecurityScheme("oauth", spec.SecurityScheme{
		Type: "oauth2",
		Flows: &spec.OAuthFlows{
			ClientCredentials: &spec.OAuthFlow{
				TokenURL: "https://auth.example.com/oauth/token",
				AvailableScopes: map[string]string{
					"read:orders": "Read orders",
				},
			},
		},
	})

	openIDConnect = asyncgo.SecurityScheme("openIDConnect", spec.SecurityScheme{
		Type:             "openIdConnect",
		OpenIDConnectURL: "https://auth.example.com/.well-known/openid-configuration",
		Scopes:           []string{"read:orders"},
	})
)

// Catalog is the AsyncAPI description of a secured orders service. The asyncgo
// CLI discovers it and generates asyncapi.yaml from it.
var Catalog = asyncgo.Spec(
	asyncgo.Info("Secured Orders Service", "1.0.0").
		Description("Order events published over an authenticated broker"),

	asyncgo.SecuritySchemes(
		userPasswordAuth,
		apiKeyAuth,
		basicAuth,
		oauth,
		openIDConnect,
	),

	asyncgo.Servers(prod),

	asyncgo.Channels(
		asyncgo.Channel("order-placed").
			Description("Emitted when an order is placed").
			Servers(prod).
			Send(asyncgo.Operation().
				Security(basicAuth).
				Message(asyncgo.MessageOf(OrderPlaced{}).Name("OrderPlaced"))),
	),
)

// prod is the authenticated Kafka broker the service publishes to.
var prod = asyncgo.Server("prod", "kafka", "broker.example.com:9092").
	Description("Production Kafka cluster").
	Security(oauth)
