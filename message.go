package asyncgo

import (
	"fmt"
	"reflect"

	"github.com/RubenRibGarcia/asyncgo/schema"
	"github.com/RubenRibGarcia/asyncgo/spec"
)

// message declares a message whose payload schema is either derived from a Go
// type (MessageOf) or supplied verbatim (MessageFrom).
type message struct {
	typ          reflect.Type
	payload      *spec.Schema // set by MessageFrom; mutually exclusive with typ
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

// MessageFrom declares a message whose payload is the given schema, emitted
// verbatim. Use it for a payload that cannot be derived from a Go type — pass
// spec.MultiFormat(...) for Avro or Protobuf, or spec.Ref(...) to point at a
// schema declared with Schema(...)/Schemas(...).
//
// The name is required: it is the key under the channel's messages map.
func MessageFrom(name string, payload *spec.Schema) *message {
	return &message{name: name, payload: payload}
}

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
	payload := m.payload
	switch {
	case payload != nil && m.typ != nil:
		return nil, fmt.Errorf(
			"message: payload schema and payload type are mutually exclusive",
		)
	case payload == nil && m.typ == nil:
		return nil, fmt.Errorf("message: nil payload type or schema")
	case payload == nil:
		payload = schema.FromType(m.typ, b.defs)
	}
	if m.name == "" && m.typ == nil {
		return nil, fmt.Errorf("message: name is required for a hand-authored payload")
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
		Payload:      payload,
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
