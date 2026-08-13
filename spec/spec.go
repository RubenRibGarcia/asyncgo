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
	AsyncAPI           string                `json:"asyncapi" yaml:"asyncapi"`
	ID                 string                `json:"id,omitempty" yaml:"id,omitempty"`
	Info               Info                  `json:"info" yaml:"info"`
	Servers            map[string]*Server    `json:"servers,omitempty" yaml:"servers,omitempty"`
	DefaultContentType string                `json:"defaultContentType,omitempty" yaml:"defaultContentType,omitempty"`
	Channels           map[string]*Channel   `json:"channels,omitempty" yaml:"channels,omitempty"`
	Operations         map[string]*Operation `json:"operations,omitempty" yaml:"operations,omitempty"`
	Components         *Components           `json:"components,omitempty" yaml:"components,omitempty"`
	Tags               []Tag                 `json:"tags,omitempty" yaml:"tags,omitempty"`
	ExternalDocs       *ExternalDocs         `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
}

// Info provides metadata about the API.
type Info struct {
	Title          string   `json:"title" yaml:"title"`
	Version        string   `json:"version" yaml:"version"`
	Description    string   `json:"description,omitempty" yaml:"description,omitempty"`
	TermsOfService string   `json:"termsOfService,omitempty" yaml:"termsOfService,omitempty"`
	Contact        *Contact `json:"contact,omitempty" yaml:"contact,omitempty"`
	License        *License `json:"license,omitempty" yaml:"license,omitempty"`
	Tags           []Tag    `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// Contact is the contact information for the API.
type Contact struct {
	Name  string `json:"name,omitempty" yaml:"name,omitempty"`
	URL   string `json:"url,omitempty" yaml:"url,omitempty"`
	Email string `json:"email,omitempty" yaml:"email,omitempty"`
}

// License is the license information for the API.
type License struct {
	Name       string `json:"name" yaml:"name"`
	URL        string `json:"url,omitempty" yaml:"url,omitempty"`
	Identifier string `json:"identifier,omitempty" yaml:"identifier,omitempty"`
}

// Tag is a metadata tag.
type Tag struct {
	Name         string        `json:"name" yaml:"name"`
	Description  string        `json:"description,omitempty" yaml:"description,omitempty"`
	ExternalDocs *ExternalDocs `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
}

// ExternalDocs points to external documentation.
type ExternalDocs struct {
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	URL         string `json:"url" yaml:"url"`
}

// Server represents a message broker.
type Server struct {
	Host            string                     `json:"host" yaml:"host"`
	Protocol        string                     `json:"protocol" yaml:"protocol"`
	ProtocolVersion string                     `json:"protocolVersion,omitempty" yaml:"protocolVersion,omitempty"`
	Description     string                     `json:"description,omitempty" yaml:"description,omitempty"`
	Variables       map[string]*ServerVariable `json:"variables,omitempty" yaml:"variables,omitempty"`
	Security        []SecurityRequirement      `json:"security,omitempty" yaml:"security,omitempty"`
	Tags            []Tag                      `json:"tags,omitempty" yaml:"tags,omitempty"`
	Bindings        ServerBindings             `json:"bindings,omitempty" yaml:"bindings,omitempty"`
}

// ServerVariable is a variable for server URL template substitution.
type ServerVariable struct {
	Enum        []string `json:"enum,omitempty" yaml:"enum,omitempty"`
	Default     string   `json:"default,omitempty" yaml:"default,omitempty"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Examples    []string `json:"examples,omitempty" yaml:"examples,omitempty"`
}

// SecurityRequirement maps a security scheme to the scopes it requires.
type SecurityRequirement map[string][]string

// Channel describes a channel/topic/queue on which messages flow.
type Channel struct {
	Address     string                `json:"address,omitempty" yaml:"address,omitempty"`
	Messages    map[string]*Message   `json:"messages,omitempty" yaml:"messages,omitempty"`
	Title       string                `json:"title,omitempty" yaml:"title,omitempty"`
	Description string                `json:"description,omitempty" yaml:"description,omitempty"`
	Parameters  map[string]*Parameter `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Tags        []Tag                 `json:"tags,omitempty" yaml:"tags,omitempty"`
	Bindings    ChannelBindings       `json:"bindings,omitempty" yaml:"bindings,omitempty"`
}

// Operation describes an application-defined operation on a channel.
type Operation struct {
	Action       string                `json:"action" yaml:"action"` // "send" | "receive"
	Channel      *Reference            `json:"channel" yaml:"channel"`
	Title        string                `json:"title,omitempty" yaml:"title,omitempty"`
	Summary      string                `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description  string                `json:"description,omitempty" yaml:"description,omitempty"`
	Security     []SecurityRequirement `json:"security,omitempty" yaml:"security,omitempty"`
	Tags         []Tag                 `json:"tags,omitempty" yaml:"tags,omitempty"`
	ExternalDocs *ExternalDocs         `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
	Bindings     OperationBindings     `json:"bindings,omitempty" yaml:"bindings,omitempty"`
	Traits       []*Reference          `json:"traits,omitempty" yaml:"traits,omitempty"`
	Messages     []*Reference          `json:"messages,omitempty" yaml:"messages,omitempty"`
}

// Reference is a JSON Reference to a reusable component.
type Reference struct {
	Ref string `json:"$ref" yaml:"$ref"`
}

// Message describes a message exchanged on a channel.
type Message struct {
	Headers       *Schema          `json:"headers,omitempty" yaml:"headers,omitempty"`
	Payload       *Schema          `json:"payload,omitempty" yaml:"payload,omitempty"`
	CorrelationID *Reference       `json:"correlationId,omitempty" yaml:"correlationId,omitempty"`
	ContentType   string           `json:"contentType,omitempty" yaml:"contentType,omitempty"`
	Name          string           `json:"name,omitempty" yaml:"name,omitempty"`
	Title         string           `json:"title,omitempty" yaml:"title,omitempty"`
	Summary       string           `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description   string           `json:"description,omitempty" yaml:"description,omitempty"`
	Tags          []Tag            `json:"tags,omitempty" yaml:"tags,omitempty"`
	ExternalDocs  *ExternalDocs    `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
	Bindings      MessageBindings  `json:"bindings,omitempty" yaml:"bindings,omitempty"`
	Examples      []MessageExample `json:"examples,omitempty" yaml:"examples,omitempty"`
	Traits        []*Reference     `json:"traits,omitempty" yaml:"traits,omitempty"`
}

// MessageExample is a named example of a message payload.
type MessageExample struct {
	Name    string         `json:"name,omitempty" yaml:"name,omitempty"`
	Summary string         `json:"summary,omitempty" yaml:"summary,omitempty"`
	Headers map[string]any `json:"headers,omitempty" yaml:"headers,omitempty"`
	Payload any            `json:"payload,omitempty" yaml:"payload,omitempty"`
}

// Parameter describes a channel parameter.
type Parameter struct {
	Description string  `json:"description,omitempty" yaml:"description,omitempty"`
	Schema      *Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
	Location    string  `json:"location,omitempty" yaml:"location,omitempty"`
}

// CorrelationID identifies a message by a correlation value.
type CorrelationID struct {
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Location    string `json:"location" yaml:"location"`
}

// Components holds reusable objects for the API.
type Components struct {
	Schemas        map[string]*Schema        `json:"schemas,omitempty" yaml:"schemas,omitempty"`
	Servers        map[string]*Server        `json:"servers,omitempty" yaml:"servers,omitempty"`
	Channels       map[string]*Channel       `json:"channels,omitempty" yaml:"channels,omitempty"`
	Operations     map[string]*Operation     `json:"operations,omitempty" yaml:"operations,omitempty"`
	Messages       map[string]*Message       `json:"messages,omitempty" yaml:"messages,omitempty"`
	Parameters     map[string]*Parameter     `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	CorrelationIDs map[string]*CorrelationID `json:"correlationIds,omitempty" yaml:"correlationIds,omitempty"`
}
