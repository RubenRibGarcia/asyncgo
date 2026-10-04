// Package asyncgo is the developer-facing fluent DSL for declaring an AsyncAPI
// document. A catalog is a package-level variable of type *SpecResult built
// with Spec(...); the asyncgo CLI discovers such variables reachable from main
// and statically interprets them into an AsyncAPI document.
package asyncgo

import (
	"errors"
	"fmt"
	"maps"
	"strings"

	"github.com/RubenRibGarcia/asyncgo/internal/jsonpointer"
	"github.com/RubenRibGarcia/asyncgo/spec"
)

// Item is a fragment of an AsyncAPI document. The concrete builder types in
// this package implement it; the method is unexported so the set of Items is
// closed — users compose them only via the exported constructor functions.
type Item interface {
	apply(b *builder) error
}

// builder accumulates a document and the hoisted schemas it references.
type builder struct {
	doc  *spec.AsyncAPI
	defs map[string]*spec.Schema
}

// SpecResult is the outcome of building a catalog: the assembled document plus
// any validation errors encountered while applying its items. Doc is always
// non-nil; Err is nil when the catalog is valid.
type SpecResult struct {
	Doc  *spec.AsyncAPI
	Err  error
	errs []error
}

// ValidationErrors returns the individual validation errors that Err joins, or
// nil when the catalog is valid. The generator harness uses this to render the
// per-catalog report.
func (r *SpecResult) ValidationErrors() []error { return r.errs }

// Spec assembles an AsyncAPI document from the given fragments.
func Spec(items ...Item) *SpecResult {
	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	var errs []error
	for _, it := range items {
		if err := it.apply(b); err != nil {
			errs = append(errs, err)
		}
	}
	errs = append(errs, b.validateServerRefs()...)
	errs = append(errs, b.validateSecurityRefs()...)
	if len(b.defs) > 0 {
		c := b.components()
		maps.Copy(c.Schemas, b.defs)
	}
	return &SpecResult{Doc: b.doc, Err: errors.Join(errs...), errs: errs}
}

func (b *builder) components() *spec.Components {
	if b.doc.Components == nil {
		b.doc.Components = &spec.Components{}
	}
	if b.doc.Components.Schemas == nil {
		b.doc.Components.Schemas = map[string]*spec.Schema{}
	}
	return b.doc.Components
}

// validateServerRefs checks that every channel server reference points at a
// server declared via Servers(...). It is a post-pass: it runs after all items
// are applied because declaration order is arbitrary (a channel may be applied
// before the server it references).
func (b *builder) validateServerRefs() []error {
	var errs []error
	for addr, ch := range b.doc.Channels {
		for _, ref := range ch.Servers {
			name := jsonpointer.Unescape(strings.TrimPrefix(ref.Ref, "#/servers/"))
			if _, ok := b.doc.Servers[name]; !ok {
				errs = append(
					errs,
					fmt.Errorf("channel.%s: references unknown server %q", addr, name),
				)
			}
		}
	}
	return errs
}

// validateSecurityRefs checks that every security reference on a server or an
// operation points at a scheme declared via SecuritySchemes(...). It is a
// post-pass for the same reason as validateServerRefs: declaration order is
// arbitrary.
func (b *builder) validateSecurityRefs() []error {
	const prefix = "#/components/securitySchemes/"

	var declared map[string]*spec.SecurityScheme
	if b.doc.Components != nil {
		declared = b.doc.Components.SecuritySchemes
	}

	var errs []error
	for name, srv := range b.doc.Servers {
		for _, ref := range srv.Security {
			scheme := jsonpointer.Unescape(strings.TrimPrefix(ref.Ref, prefix))
			if _, ok := declared[scheme]; !ok {
				errs = append(
					errs,
					fmt.Errorf("server.%s: references unknown security scheme %q", name, scheme),
				)
			}
		}
	}
	for key, op := range b.doc.Operations {
		for _, ref := range op.Security {
			scheme := jsonpointer.Unescape(strings.TrimPrefix(ref.Ref, prefix))
			if _, ok := declared[scheme]; !ok {
				errs = append(
					errs,
					fmt.Errorf("operation.%s: references unknown security scheme %q", key, scheme),
				)
			}
		}
	}
	return errs
}

// --- info -------------------------------------------------------------------

type infoBuilder struct {
	info spec.Info
}

// Info starts the info section with the required title and version.
func Info(title, version string) *infoBuilder {
	return &infoBuilder{info: spec.Info{Title: title, Version: version}}
}

func (i *infoBuilder) Description(s string) *infoBuilder    { i.info.Description = s; return i }
func (i *infoBuilder) TermsOfService(s string) *infoBuilder { i.info.TermsOfService = s; return i }
func (i *infoBuilder) Contact(c spec.Contact) *infoBuilder  { i.info.Contact = &c; return i }
func (i *infoBuilder) License(l spec.License) *infoBuilder  { i.info.License = &l; return i }
func (i *infoBuilder) Tags(tags ...spec.Tag) *infoBuilder   { i.info.Tags = tags; return i }

func (i *infoBuilder) apply(b *builder) error {
	var errs []error
	if i.info.Title == "" {
		errs = append(errs, fmt.Errorf("info.title: is required"))
	}
	if i.info.Version == "" {
		errs = append(errs, fmt.Errorf("info.version: is required"))
	}
	b.doc.Info = i.info
	return errors.Join(errs...)
}

// --- default content type ----------------------------------------------------

type contentTypeItem string

// DefaultContentType sets the document's default content type.
func DefaultContentType(s string) Item { return contentTypeItem(s) }

func (c contentTypeItem) apply(
	b *builder,
) error {
	b.doc.DefaultContentType = string(c)
	return nil
}

// --- servers -----------------------------------------------------------------

type serversItem []*server

// Servers adds one or more servers to the document.
func Servers(s ...*server) Item { return serversItem(s) }

func (s serversItem) apply(b *builder) error {
	if b.doc.Servers == nil {
		b.doc.Servers = map[string]*spec.Server{}
	}
	var errs []error
	for _, sv := range s {
		if err := sv.apply(b); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

type server struct {
	name string
	s    spec.Server
}

// Server declares a server (broker) with a name, protocol, and host.
func Server(name, protocol, host string) *server {
	return &server{name: name, s: spec.Server{Protocol: protocol, Host: host}}
}

func (s *server) ProtocolVersion(v string) *server { s.s.ProtocolVersion = v; return s }
func (s *server) Description(d string) *server     { s.s.Description = d; return s }

// Variable declares a server URL variable.
func (s *server) Variable(name string, v spec.ServerVariable) *server {
	if s.s.Variables == nil {
		s.s.Variables = map[string]*spec.ServerVariable{}
	}
	s.s.Variables[name] = &v
	return s
}

// Security declares the security schemes a client can use with this server.
// Every scheme must be declared via SecuritySchemes(...).
func (s *server) Security(schemes ...*securityScheme) *server {
	for _, sc := range schemes {
		s.s.Security = append(s.s.Security, securitySchemeRef(sc))
	}
	return s
}

func (s *server) apply(b *builder) error {
	var errs []error
	if s.name == "" {
		errs = append(errs, fmt.Errorf("server.name: is required"))
	}
	if s.s.Protocol == "" {
		errs = append(errs, fmt.Errorf("server.%s.protocol: is required", s.name))
	}
	if s.s.Host == "" {
		errs = append(errs, fmt.Errorf("server.%s.host: is required", s.name))
	}
	if _, dup := b.doc.Servers[s.name]; dup && s.name != "" {
		errs = append(errs, fmt.Errorf("server.%s: duplicate name", s.name))
	} else {
		b.doc.Servers[s.name] = &s.s
	}
	return errors.Join(errs...)
}

// --- security schemes --------------------------------------------------------

type securitySchemesItem []*securityScheme

// SecuritySchemes adds one or more security schemes to the document's
// components.securitySchemes.
func SecuritySchemes(s ...*securityScheme) Item { return securitySchemesItem(s) }

func (s securitySchemesItem) apply(b *builder) error {
	var errs []error
	for _, sc := range s {
		if err := sc.apply(b); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

type securityScheme struct {
	name string
	s    spec.SecurityScheme
}

// SecurityScheme declares a reusable security scheme under the given name.
// Reference it from a server or an operation via their Security methods.
func SecurityScheme(name string, s spec.SecurityScheme) *securityScheme {
	return &securityScheme{name: name, s: s}
}

func (s *securityScheme) apply(b *builder) error {
	var errs []error
	if s.name == "" {
		errs = append(errs, fmt.Errorf("securityScheme.name: is required"))
	}
	if s.s.Type == "" {
		errs = append(errs, fmt.Errorf("securityScheme.%s.type: is required", s.name))
	}
	c := b.components()
	if c.SecuritySchemes == nil {
		c.SecuritySchemes = map[string]*spec.SecurityScheme{}
	}
	if _, dup := c.SecuritySchemes[s.name]; dup && s.name != "" {
		errs = append(errs, fmt.Errorf("securityScheme.%s: duplicate name", s.name))
	} else {
		c.SecuritySchemes[s.name] = &s.s
	}
	return errors.Join(errs...)
}

// securitySchemeRef is the JSON Reference to a scheme declared via
// SecurityScheme(...).
func securitySchemeRef(s *securityScheme) *spec.Reference {
	return &spec.Reference{
		Ref: "#/components/securitySchemes/" + jsonpointer.Escape(s.name),
	}
}

// --- channels ----------------------------------------------------------------

type channelsItem []*channel

// Channels adds one or more channels to the document.
func Channels(c ...*channel) Item { return channelsItem(c) }

func (c channelsItem) apply(b *builder) error {
	if b.doc.Channels == nil {
		b.doc.Channels = map[string]*spec.Channel{}
	}
	var errs []error
	for _, ch := range c {
		if err := ch.apply(b); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

type channel struct {
	address string
	s       spec.Channel
	ops     []*operation
}

// Channel declares a channel addressed by a topic/queue/subject.
func Channel(address string) *channel {
	return &channel{address: address, s: spec.Channel{Address: address}}
}

func (c *channel) Title(t string) *channel       { c.s.Title = t; return c }
func (c *channel) Description(d string) *channel { c.s.Description = d; return c }

// Servers references the servers (declared via Servers(...)) on which this
// channel is available. If empty, the channel is available on all servers.
func (c *channel) Servers(s ...*server) *channel {
	for _, sv := range s {
		c.s.Servers = append(
			c.s.Servers,
			&spec.Reference{Ref: "#/servers/" + jsonpointer.Escape(sv.name)},
		)
	}
	return c
}

// Send attaches a send operation to the channel.
func (c *channel) Send(op *operation) *channel {
	op.action = spec.ActionSend
	c.ops = append(c.ops, op)
	return c
}

// Receive attaches a receive operation to the channel.
func (c *channel) Receive(op *operation) *channel {
	op.action = spec.ActionReceive
	c.ops = append(c.ops, op)
	return c
}

func (c *channel) apply(b *builder) error {
	var errs []error
	if c.address == "" {
		errs = append(errs, fmt.Errorf("channel.address: is required"))
	}
	if b.doc.Channels == nil {
		b.doc.Channels = map[string]*spec.Channel{}
	}
	if b.doc.Operations == nil {
		b.doc.Operations = map[string]*spec.Operation{}
	}

	ch := &c.s
	if _, dup := b.doc.Channels[c.address]; dup && c.address != "" {
		errs = append(errs, fmt.Errorf("channel.%s: duplicate address", c.address))
	} else {
		b.doc.Channels[c.address] = ch
	}

	for _, op := range c.ops {
		specOp := &spec.Operation{
			Action:      op.action,
			Channel:     &spec.Reference{Ref: "#/channels/" + jsonpointer.Escape(c.address)},
			Title:       op.title,
			Summary:     op.summary,
			Description: op.description,
			Security:    op.security,
			Bindings:    op.bindings,
		}
		for _, m := range op.messages {
			sm, err := m.build(b)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			if ch.Messages == nil {
				ch.Messages = map[string]*spec.Message{}
			}
			ch.Messages[sm.Name] = sm
			specOp.Messages = append(specOp.Messages, &spec.Reference{
				Ref: "#/channels/" + jsonpointer.Escape(
					c.address,
				) + "/messages/" + jsonpointer.Escape(
					sm.Name,
				),
			})
		}
		b.doc.Operations[c.address+"."+op.action] = specOp
	}
	return errors.Join(errs...)
}

// --- operation ---------------------------------------------------------------

type operation struct {
	action      string
	title       string
	summary     string
	description string
	messages    []*message
	security    []*spec.Reference
	bindings    spec.OperationBindings
}

// Operation declares an operation on a channel.
func Operation() *operation { return &operation{} }

func (o *operation) Title(t string) *operation       { o.title = t; return o }
func (o *operation) Summary(s string) *operation     { o.summary = s; return o }
func (o *operation) Description(d string) *operation { o.description = d; return o }

// Message attaches one or more messages to the operation.
func (o *operation) Message(m ...*message) *operation {
	o.messages = append(o.messages, m...)
	return o
}

// Security declares the security schemes a client must satisfy to use this
// operation. Every scheme must be declared via SecuritySchemes(...).
func (o *operation) Security(schemes ...*securityScheme) *operation {
	for _, sc := range schemes {
		o.security = append(o.security, securitySchemeRef(sc))
	}
	return o
}
