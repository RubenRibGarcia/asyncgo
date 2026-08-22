// Package asyncgo is the developer-facing fluent DSL for declaring an AsyncAPI
// document. A catalog is a package-level variable of type *spec.AsyncAPI built
// with Spec(...); the asyncgo CLI discovers such variables reachable from main
// and statically interprets them into an AsyncAPI document.
package asyncgo

import (
	"maps"
	"strings"

	"github.com/RubenRibGarcia/asyncgo/spec"
)

// Item is a fragment of an AsyncAPI document. The concrete builder types in
// this package implement it; the method is unexported so the set of Items is
// closed — users compose them only via the exported constructor functions.
type Item interface {
	apply(b *builder)
}

// builder accumulates a document and the hoisted schemas it references.
type builder struct {
	doc  *spec.AsyncAPI
	defs map[string]*spec.Schema
}

// Spec assembles an AsyncAPI document from the given fragments.
func Spec(items ...Item) *spec.AsyncAPI {
	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	for _, it := range items {
		it.apply(b)
	}
	if len(b.defs) > 0 {
		c := b.components()
		maps.Copy(c.Schemas, b.defs)
	}
	return b.doc
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

// ptrEscape applies JSON Pointer escaping (RFC 6901) to a reference token.
func ptrEscape(s string) string {
	s = strings.ReplaceAll(s, "~", "~0")
	s = strings.ReplaceAll(s, "/", "~1")
	return s
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

func (i *infoBuilder) apply(b *builder) { b.doc.Info = i.info }

// --- default content type ----------------------------------------------------

type contentTypeItem string

// DefaultContentType sets the document's default content type.
func DefaultContentType(s string) Item { return contentTypeItem(s) }

func (c contentTypeItem) apply(b *builder) { b.doc.DefaultContentType = string(c) }

// --- servers -----------------------------------------------------------------

type serversItem []*server

// Servers adds one or more servers to the document.
func Servers(s ...*server) Item { return serversItem(s) }

func (s serversItem) apply(b *builder) {
	if b.doc.Servers == nil {
		b.doc.Servers = map[string]*spec.Server{}
	}
	for _, sv := range s {
		sv.apply(b)
	}
}

type server struct {
	name string
	s    spec.Server
}

// Server declares a server (broker) with a name and protocol.
func Server(name, protocol string) *server {
	return &server{name: name, s: spec.Server{Protocol: protocol}}
}

func (s *server) Host(h string) *server            { s.s.Host = h; return s }
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

func (s *server) apply(b *builder) { b.doc.Servers[s.name] = &s.s }

// --- channels ----------------------------------------------------------------

type channelsItem []*channel

// Channels adds one or more channels to the document.
func Channels(c ...*channel) Item { return channelsItem(c) }

func (c channelsItem) apply(b *builder) {
	if b.doc.Channels == nil {
		b.doc.Channels = map[string]*spec.Channel{}
	}
	for _, ch := range c {
		ch.apply(b)
	}
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
		c.s.Servers = append(c.s.Servers, &spec.Reference{Ref: "#/servers/" + ptrEscape(sv.name)})
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

func (c *channel) apply(b *builder) {
	if b.doc.Channels == nil {
		b.doc.Channels = map[string]*spec.Channel{}
	}
	if b.doc.Operations == nil {
		b.doc.Operations = map[string]*spec.Operation{}
	}

	ch := &c.s
	b.doc.Channels[c.address] = ch

	for _, op := range c.ops {
		specOp := &spec.Operation{
			Action:      op.action,
			Channel:     &spec.Reference{Ref: "#/channels/" + ptrEscape(c.address)},
			Title:       op.title,
			Summary:     op.summary,
			Description: op.description,
			Bindings:    op.bindings,
		}
		for _, m := range op.messages {
			sm := m.build(b)
			if ch.Messages == nil {
				ch.Messages = map[string]*spec.Message{}
			}
			ch.Messages[sm.Name] = sm
			specOp.Messages = append(specOp.Messages, &spec.Reference{
				Ref: "#/channels/" + ptrEscape(c.address) + "/messages/" + ptrEscape(sm.Name),
			})
		}
		b.doc.Operations[c.address+"."+op.action] = specOp
	}
}

// --- operation ---------------------------------------------------------------

type operation struct {
	action      string
	title       string
	summary     string
	description string
	messages    []*message
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
