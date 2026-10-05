package asyncgo

import (
	"fmt"
	"reflect"

	"github.com/RubenRibGarcia/asyncgo/schema"
	"github.com/RubenRibGarcia/asyncgo/spec"
)

// message declares a message whose payload schema is derived from a Go type.
type message struct {
	typ          reflect.Type
	name         string
	title        string
	summary      string
	description  string
	contentType  string
	headers      *spec.Schema
	examples     []spec.MessageExample
	bindings     spec.MessageBindings
	tags         []spec.Tag
	externalDocs *spec.ExternalDocs
	traits       []*spec.Reference
}

// MessageOf declares a message whose payload schema is derived from the given
// Go value's type via reflection. The type is referenced, not duplicated: the
// catalog cannot drift from the data contract.
func MessageOf(v any) *message { return &message{typ: reflect.TypeOf(v)} }

func (m *message) Name(n string) *message          { m.name = n; return m }
func (m *message) Title(t string) *message         { m.title = t; return m }
func (m *message) Summary(s string) *message       { m.summary = s; return m }
func (m *message) Description(d string) *message   { m.description = d; return m }
func (m *message) ContentType(c string) *message   { m.contentType = c; return m }
func (m *message) Headers(h *spec.Schema) *message { m.headers = h; return m }

// Tags attaches tags for logical grouping and categorization of the message.
func (m *message) Tags(tags ...spec.Tag) *message {
	m.tags = append(m.tags, tags...)
	return m
}

// ExternalDocs attaches additional external documentation for the message.
func (m *message) ExternalDocs(d spec.ExternalDocs) *message {
	m.externalDocs = &d
	return m
}

// Example attaches a named payload example to the message.
func (m *message) Example(name string, payload any) *message {
	m.examples = append(m.examples, spec.MessageExample{Name: name, Payload: payload})
	return m
}

// Traits attaches one or more message traits to the message. Every trait must
// be declared via MessageTraits(...).
func (m *message) Traits(traits ...*messageTrait) *message {
	for _, tr := range traits {
		m.traits = append(m.traits, messageTraitRef(tr))
	}
	return m
}

func (m *message) build(b *builder) (*spec.Message, error) {
	if m.typ == nil {
		return nil, fmt.Errorf("message: nil payload type")
	}
	return &spec.Message{
		Name:         messageName(m),
		Title:        m.title,
		Summary:      m.summary,
		Description:  m.description,
		ContentType:  m.contentType,
		Headers:      m.headers,
		Tags:         m.tags,
		ExternalDocs: m.externalDocs,
		Examples:     m.examples,
		Bindings:     m.bindings,
		Traits:       m.traits,
		Payload:      schema.FromType(m.typ, b.defs),
	}, nil
}

func messageName(m *message) string {
	if m.name != "" {
		return m.name
	}
	if m.typ == nil {
		return "message"
	}
	t := m.typ
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Name() != "" {
		return t.Name()
	}
	return "message"
}
