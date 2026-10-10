// Package asyncgo is the developer-facing fluent DSL for declaring an AsyncAPI
// document. A catalog is a package-level variable of type *SpecResult built
// with Spec(...); the asyncgo CLI discovers such variables reachable from main
// and statically interprets them into an AsyncAPI document.
package asyncgo

import (
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
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

// builder accumulates a document and the hoisted schemas it references, plus
// the PayloadExamples resolutions deferred to the end of the build.
type builder struct {
	doc  *spec.AsyncAPI
	defs map[string]*spec.Schema

	// pendingExamples holds PayloadExamples calls whose target is a $ref. They
	// are applied by resolvePayloadExamples once every item has run, since a
	// message may reference a component declared later in the same Spec() call.
	pendingExamples []payloadExamplesRef
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
	errs = append(errs, b.resolvePayloadExamples()...)
	errs = append(errs, b.validateSecurityRefs()...)
	errs = append(errs, b.validateReplyRefs()...)
	errs = append(errs, b.validateMessageRefs()...)
	errs = append(errs, b.validateTraitRefs()...)
	errs = append(errs, b.validateSchemaNodes()...)
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
	// Channels are walked in sorted order so the joined error message is stable:
	// SpecResult.Err is compared by exact string in tests.
	for _, addr := range slices.Sorted(maps.Keys(b.doc.Channels)) {
		for _, ref := range b.doc.Channels[addr].Servers {
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

// validateSecurityRefs checks that every security reference on a server, an
// operation, or an operation trait points at a scheme declared via
// SecuritySchemes(...). It is a post-pass for the same reason as
// validateServerRefs: declaration order is arbitrary.
func (b *builder) validateSecurityRefs() []error {
	const prefix = "#/components/securitySchemes/"

	var declared map[string]*spec.SecurityScheme
	var declaredTraits map[string]*spec.OperationTrait
	if b.doc.Components != nil {
		declared = b.doc.Components.SecuritySchemes
		declaredTraits = b.doc.Components.OperationTraits
	}

	var errs []error
	// Servers, operations, and operation traits are walked in sorted order so the
	// joined error message is stable: SpecResult.Err is compared by exact string
	// in tests.
	for _, name := range slices.Sorted(maps.Keys(b.doc.Servers)) {
		for _, ref := range b.doc.Servers[name].Security {
			scheme := jsonpointer.Unescape(strings.TrimPrefix(ref.Ref, prefix))
			if _, ok := declared[scheme]; !ok {
				errs = append(
					errs,
					fmt.Errorf("server.%s: references unknown security scheme %q", name, scheme),
				)
			}
		}
	}
	for _, key := range slices.Sorted(maps.Keys(b.doc.Operations)) {
		for _, ref := range b.doc.Operations[key].Security {
			scheme := jsonpointer.Unescape(strings.TrimPrefix(ref.Ref, prefix))
			if _, ok := declared[scheme]; !ok {
				errs = append(
					errs,
					fmt.Errorf("operation.%s: references unknown security scheme %q", key, scheme),
				)
			}
		}
	}
	for _, name := range slices.Sorted(maps.Keys(declaredTraits)) {
		for _, ref := range declaredTraits[name].Security {
			scheme := jsonpointer.Unescape(strings.TrimPrefix(ref.Ref, prefix))
			if _, ok := declared[scheme]; !ok {
				errs = append(errs, fmt.Errorf(
					"operationTrait.%s: references unknown security scheme %q",
					name,
					scheme,
				))
			}
		}
	}
	return errs
}

// validateReplyRefs checks that every operation reply reference points at a
// reply declared via Replies(...), that every declared reply's address and
// channel resolve, that a reply never combines an address with a channel, and
// that every message a reply names is one the reply's channel actually carries.
// It is a post-pass for the same reason as validateServerRefs: declaration order
// is arbitrary, so a reply may be referenced before it is declared.
//
// The address/channel rule is a specification MUST that no downstream validator
// can catch — `asyncapi validate` accepts a document that breaks it, because the
// constraint relates two objects and a JSON Schema cannot express that — so this
// is the only place it is enforced.
func (b *builder) validateReplyRefs() []error {
	const (
		repliesPrefix        = "#/components/replies/"
		replyAddressesPrefix = "#/components/replyAddresses/"
		channelsPrefix       = "#/channels/"
	)

	var declaredReplies map[string]*spec.OperationReply
	var declaredAddresses map[string]*spec.OperationReplyAddress
	if b.doc.Components != nil {
		declaredReplies = b.doc.Components.Replies
		declaredAddresses = b.doc.Components.ReplyAddresses
	}

	var errs []error

	// Keys are walked in sorted order so the joined error message is stable:
	// SpecResult.Err is compared by exact string in tests.
	for _, key := range slices.Sorted(maps.Keys(b.doc.Operations)) {
		ref := b.doc.Operations[key].Reply
		if ref == nil {
			continue
		}
		name := jsonpointer.Unescape(strings.TrimPrefix(ref.Ref, repliesPrefix))
		if _, ok := declaredReplies[name]; !ok {
			errs = append(errs, fmt.Errorf("operation.%s: references unknown reply %q", key, name))
		}
	}

	for _, name := range slices.Sorted(maps.Keys(declaredReplies)) {
		r := declaredReplies[name]
		if r.Address != nil {
			addr := jsonpointer.Unescape(strings.TrimPrefix(r.Address.Ref, replyAddressesPrefix))
			if _, ok := declaredAddresses[addr]; !ok {
				errs = append(
					errs,
					fmt.Errorf("reply.%s: references unknown reply address %q", name, addr),
				)
			}
		}
		if r.Address != nil && r.Channel != nil {
			errs = append(errs, fmt.Errorf(
				"reply.%s: address and channel are mutually exclusive (the referenced channel must have no address)",
				name,
			))
		}
		if len(r.Messages) > 0 && r.Channel == nil {
			errs = append(errs, fmt.Errorf("reply.%s: messages require a channel", name))
		}
		if r.Channel == nil {
			continue
		}
		channel := jsonpointer.Unescape(strings.TrimPrefix(r.Channel.Ref, channelsPrefix))
		if _, ok := b.doc.Channels[channel]; !ok {
			errs = append(
				errs,
				fmt.Errorf("reply.%s: references unknown channel %q", name, channel),
			)
			continue
		}
		prefix := channelsPrefix + jsonpointer.Escape(channel) + "/messages/"
		for _, msg := range r.Messages {
			if !strings.HasPrefix(msg.Ref, prefix) {
				errs = append(errs, fmt.Errorf(
					"reply.%s: message %q is not in channel %q",
					name,
					msg.Ref,
					channel,
				))
				continue
			}
			key := jsonpointer.Unescape(strings.TrimPrefix(msg.Ref, prefix))
			if _, ok := b.doc.Channels[channel].Messages[key]; !ok {
				errs = append(errs, fmt.Errorf(
					"reply.%s: message %q is not in channel %q",
					name,
					msg.Ref,
					channel,
				))
			}
		}
	}

	return errs
}

// validateMessageRefs checks that every channel messages entry that is a
// Reference Object points at a message declared in components.messages. It is a
// post-pass for the same reason as validateServerRefs: declaration order is
// arbitrary. The generator hoists and references a message in one step, so it
// cannot produce a dangling reference; the check guards a document assembled
// through the spec package directly.
func (b *builder) validateMessageRefs() []error {
	const prefix = "#/components/messages/"

	var declared map[string]*spec.Message
	if b.doc.Components != nil {
		declared = b.doc.Components.Messages
	}

	var errs []error
	// Channels and their messages are walked in sorted order so the joined error
	// message is stable: SpecResult.Err is compared by exact string in tests.
	for _, address := range slices.Sorted(maps.Keys(b.doc.Channels)) {
		ch := b.doc.Channels[address]
		if ch == nil {
			continue
		}
		for _, key := range slices.Sorted(maps.Keys(ch.Messages)) {
			msg := ch.Messages[key]
			if msg == nil || msg.Ref == "" {
				continue
			}
			name := jsonpointer.Unescape(strings.TrimPrefix(msg.Ref, prefix))
			if _, ok := declared[name]; !ok {
				errs = append(errs, fmt.Errorf(
					"channel.%s.messages.%s: references unknown message %q",
					address,
					key,
					name,
				))
			}
		}
	}

	return errs
}

// validateTraitRefs checks that every operation and message trait reference
// points at a trait declared via OperationTraits(...) / MessageTraits(...), and
// that every message trait's correlation id points at a component declared via
// CorrelationIDs(...). It is a post-pass for the same reason as
// validateServerRefs: declaration order is arbitrary, so a trait may be
// referenced before it is declared.
func (b *builder) validateTraitRefs() []error {
	const (
		operationTraitsPrefix = "#/components/operationTraits/"
		messageTraitsPrefix   = "#/components/messageTraits/"
		correlationIDsPrefix  = "#/components/correlationIds/"
	)

	var declaredOperationTraits map[string]*spec.OperationTrait
	var declaredMessageTraits map[string]*spec.MessageTrait
	var declaredCorrelationIDs map[string]*spec.CorrelationID
	var declaredMessages map[string]*spec.Message
	if b.doc.Components != nil {
		declaredOperationTraits = b.doc.Components.OperationTraits
		declaredMessageTraits = b.doc.Components.MessageTraits
		declaredCorrelationIDs = b.doc.Components.CorrelationIDs
		declaredMessages = b.doc.Components.Messages
	}

	var errs []error

	// Keys are walked in sorted order so the joined error message is stable:
	// SpecResult.Err is compared by exact string in tests.
	for _, key := range slices.Sorted(maps.Keys(b.doc.Operations)) {
		for _, ref := range b.doc.Operations[key].Traits {
			name := jsonpointer.Unescape(strings.TrimPrefix(ref.Ref, operationTraitsPrefix))
			if _, ok := declaredOperationTraits[name]; !ok {
				errs = append(errs, fmt.Errorf(
					"operation.%s: references unknown operation trait %q",
					key,
					name,
				))
			}
		}
	}

	for _, key := range slices.Sorted(maps.Keys(declaredMessages)) {
		msg := declaredMessages[key]
		if msg == nil {
			continue
		}
		for _, ref := range msg.Traits {
			trait := jsonpointer.Unescape(strings.TrimPrefix(ref.Ref, messageTraitsPrefix))
			if _, ok := declaredMessageTraits[trait]; !ok {
				errs = append(errs, fmt.Errorf(
					"message.%s: references unknown message trait %q",
					key,
					trait,
				))
			}
		}
		if ref := msg.CorrelationID; ref != nil {
			id := jsonpointer.Unescape(strings.TrimPrefix(ref.Ref, correlationIDsPrefix))
			if _, ok := declaredCorrelationIDs[id]; !ok {
				errs = append(errs, fmt.Errorf(
					"message.%s: references unknown correlation id %q",
					key,
					id,
				))
			}
		}
	}

	for _, name := range slices.Sorted(maps.Keys(declaredMessageTraits)) {
		ref := declaredMessageTraits[name].CorrelationID
		if ref == nil {
			continue
		}
		id := jsonpointer.Unescape(strings.TrimPrefix(ref.Ref, correlationIDsPrefix))
		if _, ok := declaredCorrelationIDs[id]; !ok {
			errs = append(errs, fmt.Errorf(
				"messageTrait.%s: references unknown correlation id %q",
				name,
				id,
			))
		}
	}

	return errs
}

// validateSchemaNodes checks every schema that reaches the document for the Multi
// Format Schema Object invariants: schemaFormat and schema are set together, a
// multi-format node carries no JSON Schema keyword, and a declared
// components.schemas name does not collide with an auto-hoisted schema. It runs
// before Spec merges the hoisted definitions, because that merge is what a
// collision would silently corrupt.
func (b *builder) validateSchemaNodes() []error {
	var errs []error

	if comps := b.doc.Components; comps != nil {
		for _, name := range slices.Sorted(maps.Keys(comps.Schemas)) {
			if _, hoisted := b.defs[name]; hoisted {
				errs = append(errs, fmt.Errorf(
					"schema.%s: collides with an auto-hoisted schema of the same name",
					name,
				))
			}
			errs = append(errs, schemaNodeErrors(comps.Schemas[name], "schema."+name)...)
		}
		for _, name := range slices.Sorted(maps.Keys(comps.MessageTraits)) {
			if tr := comps.MessageTraits[name]; tr != nil {
				errs = append(
					errs,
					schemaNodeErrors(tr.Headers, "messageTrait."+name+".headers")...,
				)
			}
		}
		// Messages are hoisted into components.messages and a channel's messages
		// map holds only references to them (validateMessageRefs checks that the
		// references resolve), so every payload and header that can reach the
		// document is walked here.
		for _, key := range slices.Sorted(maps.Keys(comps.Messages)) {
			msg := comps.Messages[key]
			if msg == nil {
				continue
			}
			base := "message." + key
			errs = append(errs, schemaNodeErrors(msg.Payload, base+".payload")...)
			errs = append(errs, schemaNodeErrors(msg.Headers, base+".headers")...)
		}
	}

	return errs
}

// schemaNodeErrors reports the Multi Format Schema Object invariants violated by s
// or by any schema nested inside it. path names the node in the document and is
// extended with the child keyword on the way down.
func schemaNodeErrors(s *spec.Schema, path string) []error {
	if s == nil {
		return nil
	}

	var errs []error
	if s.SchemaFormat != "" || s.Schema != nil {
		if s.SchemaFormat == "" {
			errs = append(
				errs,
				fmt.Errorf("%s.schemaFormat: is required alongside schema", path),
			)
		}
		if s.Schema == nil {
			errs = append(
				errs,
				fmt.Errorf("%s.schema: is required alongside schemaFormat", path),
			)
		}
		if kw := jsonSchemaKeywords(s); len(kw) > 0 {
			errs = append(errs, fmt.Errorf(
				"%s: multi format schema must not carry JSON Schema keyword(s): %s",
				path,
				strings.Join(kw, ", "),
			))
		}
	}

	for _, child := range schemaChildren(s) {
		errs = append(errs, schemaNodeErrors(child.schema, path+"."+child.name)...)
	}
	return errs
}

// schemaChild is a nested schema together with the path segment naming it.
type schemaChild struct {
	name   string
	schema *spec.Schema
}

// schemaChildren returns the nested schemas of s in a deterministic order.
//
// Every schema-valued keyword is walked, not just the combinators: an invalid
// Multi Format node nested under one of them must still be reported by
// schemaNodeErrors. jsonSchemaKeywords needs no matching edit — it reflects over
// the struct — and that asymmetry is exactly what this list has to cover, so a
// new *Schema field belongs here too.
func schemaChildren(s *spec.Schema) []schemaChild {
	var children []schemaChild
	for _, name := range slices.Sorted(maps.Keys(s.Properties)) {
		children = append(children, schemaChild{"properties." + name, s.Properties[name]})
	}
	for _, name := range slices.Sorted(maps.Keys(s.Definitions)) {
		children = append(children, schemaChild{"definitions." + name, s.Definitions[name]})
	}
	for i, m := range s.AllOf {
		children = append(children, schemaChild{fmt.Sprintf("allOf.%d", i), m})
	}
	for i, m := range s.OneOf {
		children = append(children, schemaChild{fmt.Sprintf("oneOf.%d", i), m})
	}
	for i, m := range s.AnyOf {
		children = append(children, schemaChild{fmt.Sprintf("anyOf.%d", i), m})
	}
	if s.Items != nil {
		children = append(children, schemaChild{"items", s.Items})
	}
	if s.AdditionalProperties != nil {
		children = append(children, schemaChild{"additionalProperties", s.AdditionalProperties})
	}
	if s.Not != nil {
		children = append(children, schemaChild{"not", s.Not})
	}
	for _, name := range slices.Sorted(maps.Keys(s.PatternProperties)) {
		children = append(
			children,
			schemaChild{"patternProperties." + name, s.PatternProperties[name]},
		)
	}
	for _, name := range slices.Sorted(maps.Keys(s.Dependencies)) {
		children = append(children, schemaChild{"dependencies." + name, s.Dependencies[name]})
	}
	for _, kw := range []struct {
		name string
		node *spec.Schema
	}{
		{"additionalItems", s.AdditionalItems},
		{"contains", s.Contains},
		{"propertyNames", s.PropertyNames},
		{"if", s.If},
		{"then", s.Then},
		{"else", s.Else},
	} {
		if kw.node != nil {
			children = append(children, schemaChild{kw.name, kw.node})
		}
	}
	return children
}

// multiFormatFields are the Schema fields that make a node a Multi Format Schema
// Object; every other field is a JSON Schema keyword.
var multiFormatFields = map[string]bool{"SchemaFormat": true, "Schema": true}

// jsonSchemaKeywords returns the JSON names of the JSON Schema keyword fields set
// on s. It reflects over the struct rather than listing the fields, so a keyword
// added later is covered without touching this check.
func jsonSchemaKeywords(s *spec.Schema) []string {
	v := reflect.ValueOf(*s)
	t := v.Type()
	var set []string
	for i := range t.NumField() {
		f := t.Field(i)
		if multiFormatFields[f.Name] || v.Field(i).IsZero() {
			continue
		}
		name := f.Name
		if before, _, found := strings.Cut(f.Tag.Get("json"), ","); found {
			name = before
		}
		set = append(set, name)
	}
	return set
}

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

// Tags attaches tags for logical grouping and categorization of the server.
func (s *server) Tags(tags ...spec.Tag) *server {
	s.s.Tags = append(s.s.Tags, tags...)
	return s
}

// ExternalDocs attaches additional external documentation for the server.
func (s *server) ExternalDocs(d spec.ExternalDocs) *server {
	s.s.ExternalDocs = &d
	return s
}

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

// operationTraitRef is the JSON Reference to an operation trait declared via
// OperationTraits(...).
func operationTraitRef(t *operationTrait) *spec.Reference {
	return &spec.Reference{
		Ref: "#/components/operationTraits/" + jsonpointer.Escape(t.name),
	}
}

// messageTraitRef is the JSON Reference to a message trait declared via
// MessageTraits(...).
func messageTraitRef(t *messageTrait) *spec.Reference {
	return &spec.Reference{Ref: "#/components/messageTraits/" + jsonpointer.Escape(t.name)}
}

// correlationIDRef is the JSON Reference to a correlation id declared via
// CorrelationIDs(...).
func correlationIDRef(c *correlationID) *spec.Reference {
	return &spec.Reference{Ref: "#/components/correlationIds/" + jsonpointer.Escape(c.name)}
}

// correlationIDFromRef hoists an inline correlation id authored with
// Message.CorrelationIDFrom / MessageTrait.CorrelationIDFrom. The component key
// is derived from the owning object's name — <owner>CorrelationID — so the id is
// emitted as a reusable $ref. An existing component with the same value is
// reused; a different value at the same key is a duplicate-name error.
func correlationIDFromRef(b *builder, owner string, c spec.CorrelationID) (*spec.Reference, error) {
	name := owner + "CorrelationID"
	if c.Location == "" {
		return nil, fmt.Errorf("correlationId.%s.location: is required", name)
	}
	comps := b.components()
	if comps.CorrelationIDs == nil {
		comps.CorrelationIDs = map[string]*spec.CorrelationID{}
	}
	if existing, ok := comps.CorrelationIDs[name]; ok {
		if *existing != c {
			return nil, fmt.Errorf("correlationId.%s: duplicate name", name)
		}
	} else {
		value := c
		comps.CorrelationIDs[name] = &value
	}
	return &spec.Reference{
		Ref: "#/components/correlationIds/" + jsonpointer.Escape(name),
	}, nil
}

// replyRef is the JSON Reference to a reply declared via Replies(...).
func replyRef(r *reply) *spec.Reference {
	return &spec.Reference{Ref: "#/components/replies/" + jsonpointer.Escape(r.name)}
}

// replyAddressRef is the JSON Reference to a reply address declared via
// ReplyAddresses(...).
func replyAddressRef(a *replyAddress) *spec.Reference {
	return &spec.Reference{Ref: "#/components/replyAddresses/" + jsonpointer.Escape(a.name)}
}

// channelRef is the JSON Reference to a channel declared via Channels(...).
func channelRef(ch *channel) *spec.Reference {
	return &spec.Reference{Ref: "#/channels/" + jsonpointer.Escape(ch.address)}
}

// channelMessageRef is the JSON Reference to a message carried by ch. It
// mirrors the refs channel.apply builds for an operation's own messages, so the
// two agree on the message's key in the channel.
func channelMessageRef(ch *channel, m *message) *spec.Reference {
	return &spec.Reference{
		Ref: "#/channels/" + jsonpointer.Escape(ch.address) + "/messages/" + jsonpointer.Escape(
			messageKey(m),
		),
	}
}

// hoistMessage registers msg in components.messages under key — the message's
// identity, as messageKey derives it — and returns the Reference Object a
// channel's messages map carries in its place. The same message may be carried
// by several channels, and an identical one may legitimately be declared twice,
// so an existing entry under the same key is reused when it is identical and
// rejected as a duplicate when it is not. owner names the declaring channel in
// that error.
//
// The returned entry is a Reference Object, whose Ref is the only field set: a
// channel messages entry may be a Message Object or a Reference Object, and the
// generator always hoists.
func hoistMessage(
	b *builder,
	key string,
	owner string,
	msg *spec.Message,
) (*spec.Message, error) {
	c := b.components()
	if c.Messages == nil {
		c.Messages = map[string]*spec.Message{}
	}
	if existing, ok := c.Messages[key]; ok {
		if !reflect.DeepEqual(existing, msg) {
			return nil, fmt.Errorf(
				"message.%s: duplicate name with different content (%s)",
				key,
				owner,
			)
		}
	} else {
		c.Messages[key] = msg
	}
	return &spec.Message{Ref: "#/components/messages/" + jsonpointer.Escape(key)}, nil
}

// --- traits ------------------------------------------------------------------

type operationTraitsItem []*operationTrait

// OperationTraits adds one or more reusable operation traits to the document's
// components.operationTraits. Reference one from an operation via
// Operation.Traits.
func OperationTraits(t ...*operationTrait) Item { return operationTraitsItem(t) }

func (t operationTraitsItem) apply(b *builder) error {
	var errs []error
	for _, tr := range t {
		if err := tr.apply(b); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// operationTrait is an Operation Trait Object under construction. Its field set
// is the shareable subset of an operation: action, channel, messages, and traits
// are not settable, because the specification excludes them.
type operationTrait struct {
	name string
	t    spec.OperationTrait
}

// OperationTrait declares a reusable operation trait under the given name.
// Reference it from an operation via Operation.Traits.
func OperationTrait(name string) *operationTrait {
	return &operationTrait{name: name}
}

func (t *operationTrait) Title(v string) *operationTrait       { t.t.Title = v; return t }
func (t *operationTrait) Summary(v string) *operationTrait     { t.t.Summary = v; return t }
func (t *operationTrait) Description(v string) *operationTrait { t.t.Description = v; return t }

func (t *operationTrait) Tags(tags ...spec.Tag) *operationTrait {
	t.t.Tags = append(t.t.Tags, tags...)
	return t
}

func (t *operationTrait) ExternalDocs(d spec.ExternalDocs) *operationTrait {
	t.t.ExternalDocs = &d
	return t
}

// Security declares the security schemes a client must satisfy to use an
// operation carrying this trait. Every scheme must be declared via
// SecuritySchemes(...).
func (t *operationTrait) Security(schemes ...*securityScheme) *operationTrait {
	for _, sc := range schemes {
		t.t.Security = append(t.t.Security, securitySchemeRef(sc))
	}
	return t
}

// Bindings replaces the trait's protocol bindings wholesale. Prefer the typed
// Kafka/AMQP/NATS/MQTT helpers (see bindings.go) to set a single protocol.
func (t *operationTrait) Bindings(bindings spec.OperationBindings) *operationTrait {
	t.t.Bindings = bindings
	return t
}

func (t *operationTrait) apply(b *builder) error {
	var errs []error
	if t.name == "" {
		errs = append(errs, fmt.Errorf("operationTrait.name: is required"))
	}
	c := b.components()
	if c.OperationTraits == nil {
		c.OperationTraits = map[string]*spec.OperationTrait{}
	}
	if _, dup := c.OperationTraits[t.name]; dup && t.name != "" {
		errs = append(errs, fmt.Errorf("operationTrait.%s: duplicate name", t.name))
	} else {
		c.OperationTraits[t.name] = &t.t
	}
	return errors.Join(errs...)
}

type messageTraitsItem []*messageTrait

// MessageTraits adds one or more reusable message traits to the document's
// components.messageTraits. Reference one from a message via its Traits method.
func MessageTraits(t ...*messageTrait) Item { return messageTraitsItem(t) }

func (t messageTraitsItem) apply(b *builder) error {
	var errs []error
	for _, tr := range t {
		if err := tr.apply(b); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// messageTrait is a Message Trait Object under construction. Its field set is
// the shareable subset of a message: payload and traits are not settable,
// because the specification excludes them.
type messageTrait struct {
	name string
	t    spec.MessageTrait

	// correlationIDFrom is the inline value set by CorrelationIDFrom. It is
	// mutually exclusive with t.CorrelationID and is hoisted into
	// components.correlationIds by apply.
	correlationIDFrom *spec.CorrelationID
}

// MessageTrait declares a reusable message trait under the given name.
// Reference it from a message via its Traits method.
func MessageTrait(name string) *messageTrait {
	return &messageTrait{name: name}
}

func (t *messageTrait) Headers(h *spec.Schema) *messageTrait { t.t.Headers = h; return t }
func (t *messageTrait) ContentType(v string) *messageTrait   { t.t.ContentType = v; return t }
func (t *messageTrait) Name(v string) *messageTrait          { t.t.Name = v; return t }
func (t *messageTrait) Title(v string) *messageTrait         { t.t.Title = v; return t }
func (t *messageTrait) Summary(v string) *messageTrait       { t.t.Summary = v; return t }
func (t *messageTrait) Description(v string) *messageTrait   { t.t.Description = v; return t }

// CorrelationID points the trait at a correlation id declared via
// CorrelationIDs(...).
func (t *messageTrait) CorrelationID(c *correlationID) *messageTrait {
	t.t.CorrelationID = correlationIDRef(c)
	return t
}

// CorrelationIDFrom attaches an inline correlation id. The builder registers it
// under components.correlationIds as <traitName>CorrelationID and references it
// from the trait, so the id stays reusable and the trait carries a $ref.
func (t *messageTrait) CorrelationIDFrom(c spec.CorrelationID) *messageTrait {
	t.correlationIDFrom = &c
	return t
}

func (t *messageTrait) Tags(tags ...spec.Tag) *messageTrait {
	t.t.Tags = append(t.t.Tags, tags...)
	return t
}

func (t *messageTrait) ExternalDocs(d spec.ExternalDocs) *messageTrait {
	t.t.ExternalDocs = &d
	return t
}

// Example attaches a named payload example to the trait.
func (t *messageTrait) Example(name string, payload any) *messageTrait {
	t.t.Examples = append(t.t.Examples, spec.MessageExample{Name: name, Payload: payload})
	return t
}

// Bindings replaces the trait's protocol bindings wholesale. Prefer the typed
// Kafka/AMQP/NATS/MQTT helpers (see bindings.go) to set a single protocol.
func (t *messageTrait) Bindings(bindings spec.MessageBindings) *messageTrait {
	t.t.Bindings = bindings
	return t
}

func (t *messageTrait) apply(b *builder) error {
	var errs []error
	if t.name == "" {
		errs = append(errs, fmt.Errorf("messageTrait.name: is required"))
	}
	switch {
	case t.t.CorrelationID != nil && t.correlationIDFrom != nil:
		errs = append(errs, fmt.Errorf(
			"messageTrait.%s: correlation id reference and inline correlation id are mutually exclusive",
			t.name,
		))
	case t.correlationIDFrom != nil && t.name != "":
		ref, err := correlationIDFromRef(b, t.name, *t.correlationIDFrom)
		if err != nil {
			errs = append(errs, err)
		} else {
			t.t.CorrelationID = ref
		}
	}
	c := b.components()
	if c.MessageTraits == nil {
		c.MessageTraits = map[string]*spec.MessageTrait{}
	}
	if _, dup := c.MessageTraits[t.name]; dup && t.name != "" {
		errs = append(errs, fmt.Errorf("messageTrait.%s: duplicate name", t.name))
	} else {
		c.MessageTraits[t.name] = &t.t
	}
	return errors.Join(errs...)
}

// --- schemas -----------------------------------------------------------------

type schemasItem []*schemaDecl

// Schemas adds one or more reusable schemas to the document's
// components.schemas. Reference one from a message payload or headers via
// spec.Ref("#/components/schemas/<name>").
func Schemas(d ...*schemaDecl) Item { return schemasItem(d) }

func (s schemasItem) apply(b *builder) error {
	var errs []error
	for _, d := range s {
		if err := d.apply(b); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

type schemaDecl struct {
	name string
	s    *spec.Schema
}

// Schema declares a reusable schema under the given name: a JSON Schema Schema
// Object, or a Multi Format Schema Object built with spec.MultiFormat. Register
// it with Schemas(...) and reference it with spec.Ref(...).
func Schema(name string, s *spec.Schema) *schemaDecl {
	return &schemaDecl{name: name, s: s}
}

func (d *schemaDecl) apply(b *builder) error {
	var errs []error
	if d.name == "" {
		errs = append(errs, fmt.Errorf("schema.name: is required"))
	}
	if d.s == nil {
		errs = append(errs, fmt.Errorf("schema.%s: is required", d.name))
	}
	c := b.components()
	if _, dup := c.Schemas[d.name]; dup && d.name != "" {
		errs = append(errs, fmt.Errorf("schema.%s: duplicate name", d.name))
	} else if d.name != "" && d.s != nil {
		c.Schemas[d.name] = d.s
	}
	return errors.Join(errs...)
}

// --- correlation ids ---------------------------------------------------------

type correlationIDsItem []*correlationID

// CorrelationIDs adds one or more reusable correlation ids to the document's
// components.correlationIds. Reference one from a message trait via
// MessageTrait.CorrelationID.
func CorrelationIDs(c ...*correlationID) Item { return correlationIDsItem(c) }

func (c correlationIDsItem) apply(b *builder) error {
	var errs []error
	for _, ci := range c {
		if err := ci.apply(b); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

type correlationID struct {
	name string
	c    spec.CorrelationID
}

// CorrelationID declares a reusable correlation id under the given name.
func CorrelationID(name string, c spec.CorrelationID) *correlationID {
	return &correlationID{name: name, c: c}
}

func (c *correlationID) apply(b *builder) error {
	var errs []error
	if c.name == "" {
		errs = append(errs, fmt.Errorf("correlationId.name: is required"))
	}
	if c.c.Location == "" {
		errs = append(errs, fmt.Errorf("correlationId.%s.location: is required", c.name))
	}
	comps := b.components()
	if comps.CorrelationIDs == nil {
		comps.CorrelationIDs = map[string]*spec.CorrelationID{}
	}
	if _, dup := comps.CorrelationIDs[c.name]; dup && c.name != "" {
		errs = append(errs, fmt.Errorf("correlationId.%s: duplicate name", c.name))
	} else {
		comps.CorrelationIDs[c.name] = &c.c
	}
	return errors.Join(errs...)
}

// --- replies -----------------------------------------------------------------

type replyAddressesItem []*replyAddress

// ReplyAddresses adds one or more reusable reply addresses to the document's
// components.replyAddresses. Reference one from a reply via Reply.Address.
func ReplyAddresses(a ...*replyAddress) Item { return replyAddressesItem(a) }

func (a replyAddressesItem) apply(b *builder) error {
	var errs []error
	for _, ra := range a {
		if err := ra.apply(b); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

type replyAddress struct {
	name string
	a    spec.OperationReplyAddress
}

// ReplyAddress declares a reusable reply address under the given name: the
// runtime expression that locates where an operation's reply is sent.
func ReplyAddress(name string) *replyAddress { return &replyAddress{name: name} }

func (a *replyAddress) Description(d string) *replyAddress { a.a.Description = d; return a }
func (a *replyAddress) Location(l string) *replyAddress    { a.a.Location = l; return a }

func (a *replyAddress) apply(b *builder) error {
	var errs []error
	if a.name == "" {
		errs = append(errs, fmt.Errorf("replyAddress.name: is required"))
	}
	if a.a.Location == "" {
		errs = append(errs, fmt.Errorf("replyAddress.%s.location: is required", a.name))
	}
	c := b.components()
	if c.ReplyAddresses == nil {
		c.ReplyAddresses = map[string]*spec.OperationReplyAddress{}
	}
	if _, dup := c.ReplyAddresses[a.name]; dup && a.name != "" {
		errs = append(errs, fmt.Errorf("replyAddress.%s: duplicate name", a.name))
	} else {
		c.ReplyAddresses[a.name] = &a.a
	}
	return errors.Join(errs...)
}

type repliesItem []*reply

// Replies adds one or more reusable replies to the document's
// components.replies. Reference one from an operation via Operation.Reply.
func Replies(r ...*reply) Item { return repliesItem(r) }

func (r repliesItem) apply(b *builder) error {
	var errs []error
	for _, rp := range r {
		if err := rp.apply(b); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

type reply struct {
	name string
	r    spec.OperationReply
}

// Reply declares a reusable reply under the given name. Reference it from an
// operation via Operation.Reply.
func Reply(name string) *reply { return &reply{name: name} }

// Address sets the reply address the reply is sent to. The address must be
// declared via ReplyAddresses(...).
func (r *reply) Address(a *replyAddress) *reply {
	r.r.Address = replyAddressRef(a)
	return r
}

// Channel sets the channel the reply is performed in. The channel must be
// declared via Channels(...).
func (r *reply) Channel(ch *channel) *reply {
	r.r.Channel = channelRef(ch)
	return r
}

// Message attaches the messages the reply can carry on ch. The channel is a
// parameter because every reply message ref has to point into that channel's
// messages, which keeps Message independent of the order Channel is called in.
// The reply still has to name its channel via Channel(...) to be valid.
func (r *reply) Message(ch *channel, m ...*message) *reply {
	for _, msg := range m {
		r.r.Messages = append(r.r.Messages, channelMessageRef(ch, msg))
	}
	return r
}

func (r *reply) apply(b *builder) error {
	var errs []error
	if r.name == "" {
		errs = append(errs, fmt.Errorf("reply.name: is required"))
	}
	c := b.components()
	if c.Replies == nil {
		c.Replies = map[string]*spec.OperationReply{}
	}
	if _, dup := c.Replies[r.name]; dup && r.name != "" {
		errs = append(errs, fmt.Errorf("reply.%s: duplicate name", r.name))
	} else {
		c.Replies[r.name] = &r.r
	}
	return errors.Join(errs...)
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

// Tags attaches tags for logical grouping and categorization of the channel.
func (c *channel) Tags(tags ...spec.Tag) *channel {
	c.s.Tags = append(c.s.Tags, tags...)
	return c
}

// ExternalDocs attaches additional external documentation for the channel.
func (c *channel) ExternalDocs(d spec.ExternalDocs) *channel {
	c.s.ExternalDocs = &d
	return c
}

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
			Action:       op.action,
			Channel:      &spec.Reference{Ref: "#/channels/" + jsonpointer.Escape(c.address)},
			Title:        op.title,
			Summary:      op.summary,
			Description:  op.description,
			Tags:         op.tags,
			ExternalDocs: op.externalDocs,
			Security:     op.security,
			Bindings:     op.bindings,
			Traits:       op.traits,
			Reply:        op.replyReference(),
		}
		for _, m := range op.messages {
			sm, err := m.build(b)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			key := messageKey(m)
			entry, err := hoistMessage(b, key, "channel."+c.address, sm)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			if ch.Messages == nil {
				ch.Messages = map[string]*spec.Message{}
			}
			ch.Messages[key] = entry
			specOp.Messages = append(specOp.Messages, &spec.Reference{
				Ref: "#/channels/" + jsonpointer.Escape(
					c.address,
				) + "/messages/" + jsonpointer.Escape(
					key,
				),
			})
		}
		b.doc.Operations[c.address+"."+op.action] = specOp
	}
	return errors.Join(errs...)
}

// --- operation ---------------------------------------------------------------

type operation struct {
	action       string
	title        string
	summary      string
	description  string
	tags         []spec.Tag
	externalDocs *spec.ExternalDocs
	messages     []*message
	security     []*spec.Reference
	bindings     spec.OperationBindings
	traits       []*spec.Reference
	reply        *reply
}

// Operation declares an operation on a channel.
func Operation() *operation { return &operation{} }

func (o *operation) Title(t string) *operation       { o.title = t; return o }
func (o *operation) Summary(s string) *operation     { o.summary = s; return o }
func (o *operation) Description(d string) *operation { o.description = d; return o }

// Tags attaches tags for logical grouping and categorization of the operation.
func (o *operation) Tags(tags ...spec.Tag) *operation {
	o.tags = append(o.tags, tags...)
	return o
}

// ExternalDocs attaches additional external documentation for the operation.
func (o *operation) ExternalDocs(d spec.ExternalDocs) *operation {
	o.externalDocs = &d
	return o
}

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

// Reply declares the reply this operation produces, which is what makes it a
// request/reply operation. The reply must be declared via Replies(...).
func (o *operation) Reply(r *reply) *operation {
	o.reply = r
	return o
}

// Traits attaches one or more operation traits to the operation. Every trait
// must be declared via OperationTraits(...).
func (o *operation) Traits(traits ...*operationTrait) *operation {
	for _, tr := range traits {
		o.traits = append(o.traits, operationTraitRef(tr))
	}
	return o
}

// replyReference is the JSON Reference to the declared reply, or nil when the
// operation has none — which keeps the `reply` key out of the emitted document.
func (o *operation) replyReference() *spec.Reference {
	if o.reply == nil {
		return nil
	}
	return replyRef(o.reply)
}
