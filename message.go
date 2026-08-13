package asyncgo

import (
	"reflect"

	"github.com/RubenRibGarcia/asyncgo/schema"
	"github.com/RubenRibGarcia/asyncgo/spec"
)

// message declares a message whose payload schema is derived from a Go type.
type message struct {
	typ         reflect.Type
	name        string
	title       string
	summary     string
	description string
	contentType string
	headers     *spec.Schema
	examples    []spec.MessageExample
	bindings    spec.MessageBindings
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

// Example attaches a named payload example to the message.
func (m *message) Example(name string, payload any) *message {
	m.examples = append(m.examples, spec.MessageExample{Name: name, Payload: payload})
	return m
}

func (m *message) build(b *builder) *spec.Message {
	return &spec.Message{
		Name:        messageName(m),
		Title:       m.title,
		Summary:     m.summary,
		Description: m.description,
		ContentType: m.contentType,
		Headers:     m.headers,
		Examples:    m.examples,
		Bindings:    m.bindings,
		Payload:     schema.FromType(m.typ, b.defs),
	}
}

func messageName(m *message) string {
	if m.name != "" {
		return m.name
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
