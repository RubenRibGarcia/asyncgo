// Package spec provides a typed object model for the AsyncAPI 3.1.0
// specification and codecs to serialize it to YAML or JSON.
//
// It is intentionally a plain data model with no opinions about how documents
// are produced: the asyncgo package builds documents through a fluent DSL, and
// internal/discovery builds them by statically interpreting that DSL.
package spec

// Version is the AsyncAPI specification version this package models.
const Version = "3.1.0"

// Operation actions.
const (
	ActionSend    = "send"
	ActionReceive = "receive"
)

// AsyncAPI is the root document of an AsyncAPI specification.
type AsyncAPI struct {
	AsyncAPI           string                `json:"asyncapi"                     yaml:"asyncapi"`
	ID                 string                `json:"id,omitempty"                 yaml:"id,omitempty"`
	Info               Info                  `json:"info"                         yaml:"info"`
	Servers            map[string]*Server    `json:"servers,omitempty"            yaml:"servers,omitempty"`
	DefaultContentType string                `json:"defaultContentType,omitempty" yaml:"defaultContentType,omitempty"`
	Channels           map[string]*Channel   `json:"channels,omitempty"           yaml:"channels,omitempty"`
	Operations         map[string]*Operation `json:"operations,omitempty"         yaml:"operations,omitempty"`
	Components         *Components           `json:"components,omitempty"         yaml:"components,omitempty"`
	Tags               []Tag                 `json:"tags,omitempty"               yaml:"tags,omitempty"`
	ExternalDocs       *ExternalDocs         `json:"externalDocs,omitempty"       yaml:"externalDocs,omitempty"`
}

// Info provides metadata about the API.
type Info struct {
	Title          string   `json:"title"                    yaml:"title"`
	Version        string   `json:"version"                  yaml:"version"`
	Description    string   `json:"description,omitempty"    yaml:"description,omitempty"`
	TermsOfService string   `json:"termsOfService,omitempty" yaml:"termsOfService,omitempty"`
	Contact        *Contact `json:"contact,omitempty"        yaml:"contact,omitempty"`
	License        *License `json:"license,omitempty"        yaml:"license,omitempty"`
	Tags           []Tag    `json:"tags,omitempty"           yaml:"tags,omitempty"`
}

// Contact is the contact information for the API.
type Contact struct {
	Name  string `json:"name,omitempty"  yaml:"name,omitempty"`
	URL   string `json:"url,omitempty"   yaml:"url,omitempty"`
	Email string `json:"email,omitempty" yaml:"email,omitempty"`
}

// License is the license information for the API.
type License struct {
	Name       string `json:"name"                 yaml:"name"`
	URL        string `json:"url,omitempty"        yaml:"url,omitempty"`
	Identifier string `json:"identifier,omitempty" yaml:"identifier,omitempty"`
}

// Tag is a metadata tag.
type Tag struct {
	Name         string        `json:"name"                   yaml:"name"`
	Description  string        `json:"description,omitempty"  yaml:"description,omitempty"`
	ExternalDocs *ExternalDocs `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
}

// ExternalDocs points to external documentation.
type ExternalDocs struct {
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	URL         string `json:"url"                   yaml:"url"`
}

// Server represents a message broker.
type Server struct {
	Host            string                     `json:"host"                      yaml:"host"`
	Protocol        string                     `json:"protocol"                  yaml:"protocol"`
	ProtocolVersion string                     `json:"protocolVersion,omitempty" yaml:"protocolVersion,omitempty"`
	Description     string                     `json:"description,omitempty"     yaml:"description,omitempty"`
	Variables       map[string]*ServerVariable `json:"variables,omitempty"       yaml:"variables,omitempty"`
	Security        []*Reference               `json:"security,omitempty"        yaml:"security,omitempty"`
	Tags            []Tag                      `json:"tags,omitempty"            yaml:"tags,omitempty"`
	Bindings        ServerBindings             `json:"bindings,omitempty"        yaml:"bindings,omitempty"`
}

// ServerVariable is a variable for server URL template substitution.
type ServerVariable struct {
	Enum        []string `json:"enum,omitempty"        yaml:"enum,omitempty"`
	Default     string   `json:"default,omitempty"     yaml:"default,omitempty"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Examples    []string `json:"examples,omitempty"    yaml:"examples,omitempty"`
}

// SecurityScheme is an AsyncAPI 3.1.0 Security Scheme Object: it declares how
// a client authenticates against a server or an operation.
type SecurityScheme struct {
	Type             string      `json:"type"                       yaml:"type"`
	Description      string      `json:"description,omitempty"      yaml:"description,omitempty"`
	Name             string      `json:"name,omitempty"             yaml:"name,omitempty"`
	In               string      `json:"in,omitempty"               yaml:"in,omitempty"`
	Scheme           string      `json:"scheme,omitempty"           yaml:"scheme,omitempty"`
	BearerFormat     string      `json:"bearerFormat,omitempty"     yaml:"bearerFormat,omitempty"`
	Flows            *OAuthFlows `json:"flows,omitempty"            yaml:"flows,omitempty"`
	OpenIDConnectURL string      `json:"openIdConnectUrl,omitempty" yaml:"openIdConnectUrl,omitempty"`
	Scopes           []string    `json:"scopes,omitempty"           yaml:"scopes,omitempty"`
}

// OAuthFlows configures the OAuth flows a SecurityScheme supports.
type OAuthFlows struct {
	Implicit          *OAuthFlow `json:"implicit,omitempty"          yaml:"implicit,omitempty"`
	Password          *OAuthFlow `json:"password,omitempty"          yaml:"password,omitempty"`
	ClientCredentials *OAuthFlow `json:"clientCredentials,omitempty" yaml:"clientCredentials,omitempty"`
	AuthorizationCode *OAuthFlow `json:"authorizationCode,omitempty" yaml:"authorizationCode,omitempty"`
}

// OAuthFlow is the configuration for a single OAuth flow. The specification
// marks AvailableScopes required for oauth2, but it stays omitempty here: a nil
// map without omitempty marshals to null, which is worse than omitting the key.
type OAuthFlow struct {
	AuthorizationURL string            `json:"authorizationUrl,omitempty" yaml:"authorizationUrl,omitempty"`
	TokenURL         string            `json:"tokenUrl,omitempty"         yaml:"tokenUrl,omitempty"`
	RefreshURL       string            `json:"refreshUrl,omitempty"       yaml:"refreshUrl,omitempty"`
	AvailableScopes  map[string]string `json:"availableScopes,omitempty"  yaml:"availableScopes,omitempty"`
}

// Channel describes a channel/topic/queue on which messages flow.
type Channel struct {
	Address     string                `json:"address,omitempty"     yaml:"address,omitempty"`
	Messages    map[string]*Message   `json:"messages,omitempty"    yaml:"messages,omitempty"`
	Title       string                `json:"title,omitempty"       yaml:"title,omitempty"`
	Description string                `json:"description,omitempty" yaml:"description,omitempty"`
	Servers     []*Reference          `json:"servers,omitempty"     yaml:"servers,omitempty"`
	Parameters  map[string]*Parameter `json:"parameters,omitempty"  yaml:"parameters,omitempty"`
	Tags        []Tag                 `json:"tags,omitempty"        yaml:"tags,omitempty"`
	Bindings    ChannelBindings       `json:"bindings,omitempty"    yaml:"bindings,omitempty"`
}

// Operation describes an application-defined operation on a channel.
type Operation struct {
	Action       string            `json:"action"                 yaml:"action"` // "send" | "receive"
	Channel      *Reference        `json:"channel"                yaml:"channel"`
	Title        string            `json:"title,omitempty"        yaml:"title,omitempty"`
	Summary      string            `json:"summary,omitempty"      yaml:"summary,omitempty"`
	Description  string            `json:"description,omitempty"  yaml:"description,omitempty"`
	Security     []*Reference      `json:"security,omitempty"     yaml:"security,omitempty"`
	Tags         []Tag             `json:"tags,omitempty"         yaml:"tags,omitempty"`
	ExternalDocs *ExternalDocs     `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
	Bindings     OperationBindings `json:"bindings,omitempty"     yaml:"bindings,omitempty"`
	Traits       []*Reference      `json:"traits,omitempty"       yaml:"traits,omitempty"`
	Messages     []*Reference      `json:"messages,omitempty"     yaml:"messages,omitempty"`
}

// Reference is a JSON Reference to a reusable component.
type Reference struct {
	Ref string `json:"$ref" yaml:"$ref"`
}

// Message describes a message exchanged on a channel.
type Message struct {
	Headers       *Schema          `json:"headers,omitempty"       yaml:"headers,omitempty"`
	Payload       *Schema          `json:"payload,omitempty"       yaml:"payload,omitempty"`
	CorrelationID *Reference       `json:"correlationId,omitempty" yaml:"correlationId,omitempty"`
	ContentType   string           `json:"contentType,omitempty"   yaml:"contentType,omitempty"`
	Name          string           `json:"name,omitempty"          yaml:"name,omitempty"`
	Title         string           `json:"title,omitempty"         yaml:"title,omitempty"`
	Summary       string           `json:"summary,omitempty"       yaml:"summary,omitempty"`
	Description   string           `json:"description,omitempty"   yaml:"description,omitempty"`
	Tags          []Tag            `json:"tags,omitempty"          yaml:"tags,omitempty"`
	ExternalDocs  *ExternalDocs    `json:"externalDocs,omitempty"  yaml:"externalDocs,omitempty"`
	Bindings      MessageBindings  `json:"bindings,omitempty"      yaml:"bindings,omitempty"`
	Examples      []MessageExample `json:"examples,omitempty"      yaml:"examples,omitempty"`
	Traits        []*Reference     `json:"traits,omitempty"        yaml:"traits,omitempty"`
}

// MessageExample is a named example of a message payload.
type MessageExample struct {
	Name    string         `json:"name,omitempty"    yaml:"name,omitempty"`
	Summary string         `json:"summary,omitempty" yaml:"summary,omitempty"`
	Headers map[string]any `json:"headers,omitempty" yaml:"headers,omitempty"`
	Payload any            `json:"payload,omitempty" yaml:"payload,omitempty"`
}

// Parameter describes a channel parameter.
type Parameter struct {
	Description string  `json:"description,omitempty" yaml:"description,omitempty"`
	Schema      *Schema `json:"schema,omitempty"      yaml:"schema,omitempty"`
	Location    string  `json:"location,omitempty"    yaml:"location,omitempty"`
}

// CorrelationID identifies a message by a correlation value.
type CorrelationID struct {
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Location    string `json:"location"              yaml:"location"`
}

// Components holds reusable objects for the API.
type Components struct {
	Schemas         map[string]*Schema         `json:"schemas,omitempty"         yaml:"schemas,omitempty"`
	Servers         map[string]*Server         `json:"servers,omitempty"         yaml:"servers,omitempty"`
	Channels        map[string]*Channel        `json:"channels,omitempty"        yaml:"channels,omitempty"`
	Operations      map[string]*Operation      `json:"operations,omitempty"      yaml:"operations,omitempty"`
	Messages        map[string]*Message        `json:"messages,omitempty"        yaml:"messages,omitempty"`
	SecuritySchemes map[string]*SecurityScheme `json:"securitySchemes,omitempty" yaml:"securitySchemes,omitempty"`
	Parameters      map[string]*Parameter      `json:"parameters,omitempty"      yaml:"parameters,omitempty"`
	CorrelationIDs  map[string]*CorrelationID  `json:"correlationIds,omitempty"  yaml:"correlationIds,omitempty"`
}
