package asyncgo

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/RubenRibGarcia/asyncgo/internal/jsonpointer"
	"github.com/RubenRibGarcia/asyncgo/schema"
	"github.com/RubenRibGarcia/asyncgo/spec"
)

// message declares a message whose payload schema is either derived from a Go
// type (MessageOf) or supplied verbatim (MessageFrom).
type message struct {
	typ             reflect.Type
	payload         *spec.Schema // set by MessageFrom; mutually exclusive with typ
	name            string
	title           string
	summary         string
	description     string
	contentType     string
	headers         *spec.Schema
	examples        []spec.MessageExample
	payloadExamples []any
	bindings        spec.MessageBindings
	tags            []spec.Tag
	externalDocs    *spec.ExternalDocs
	traits          []*spec.Reference

	// correlationID is the $ref set by CorrelationID. correlationIDFrom is the
	// inline value set by CorrelationIDFrom, which build hoists into
	// components.correlationIds and turns into a $ref. The two are mutually
	// exclusive.
	correlationID     *spec.Reference
	correlationIDFrom *spec.CorrelationID
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

// CorrelationID points the message at a correlation id declared via
// CorrelationIDs(...).
func (m *message) CorrelationID(c *correlationID) *message {
	m.correlationID = correlationIDRef(c)
	return m
}

// CorrelationIDFrom attaches an inline correlation id. The builder registers it
// under components.correlationIds as <messageName>CorrelationID — the message's
// Name, or the payload type's name when unset — and references it from the
// message, so the id stays reusable and the message carries a $ref.
func (m *message) CorrelationIDFrom(c spec.CorrelationID) *message {
	m.correlationIDFrom = &c
	return m
}

// Example attaches a named payload example to the message.
func (m *message) Example(name string, payload any) *message {
	m.examples = append(m.examples, spec.MessageExample{Name: name, Payload: payload})
	return m
}

// PayloadExamples attaches JSON Schema Draft-07 examples to the message's
// payload schema. It is the schema-level counterpart of Example, which attaches
// a Message Example Object to the message itself.
//
// MessageOf always produces a payload that is a $ref into components.schemas,
// and JSON Schema does not apply keywords sitting beside a $ref, so the examples
// are written onto the referenced component — the only place they can take
// effect. An unresolvable $ref is an error, never a silent no-op.
//
// The component a payload type hoists to is shared by every channel carrying
// that type. Declaring the same examples twice is idempotent; two messages
// declaring different examples for one component is an error, because silently
// merging them would make the emitted document depend on item order.
func (m *message) PayloadExamples(examples ...any) *message {
	m.payloadExamples = append(m.payloadExamples, examples...)
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
	if err := m.queuePayloadExamples(b, payload); err != nil {
		return nil, err
	}

	correlationID := m.correlationID
	switch {
	case correlationID != nil && m.correlationIDFrom != nil:
		return nil, fmt.Errorf(
			"message: correlation id reference and inline correlation id are mutually exclusive",
		)
	case m.correlationIDFrom != nil:
		ref, err := correlationIDFromRef(b, messageName(m), *m.correlationIDFrom)
		if err != nil {
			return nil, err
		}
		correlationID = ref
	}

	return &spec.Message{
		Name:          messageName(m),
		Title:         m.title,
		Summary:       m.summary,
		Description:   m.description,
		ContentType:   m.contentType,
		Headers:       m.headers,
		CorrelationID: correlationID,
		Tags:          m.tags,
		ExternalDocs:  m.externalDocs,
		Examples:      m.examples,
		Bindings:      m.bindings,
		Traits:        m.traits,
		Payload:       payload,
	}, nil
}

// messagePayloadType returns the payload type a message derives its name and
// component key from, with pointers dereferenced, or nil when the payload was
// supplied by hand (MessageFrom).
func messagePayloadType(m *message) reflect.Type {
	if m.typ == nil {
		return nil
	}
	t := m.typ
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

// messageName is the message's Name field: the authored name, or the payload
// type's short name when MessageOf left it unset.
func messageName(m *message) string {
	if m.name != "" {
		return m.name
	}
	if t := messagePayloadType(m); t != nil && t.Name() != "" {
		return t.Name()
	}
	return "message"
}

// messageKey is the message's identity in the document: the key of its
// components.messages entry, and therefore the message id a channel's messages
// map and every $ref into the message use.
//
// A name the author set — MessageFrom's name argument, or MessageOf followed by
// Name — is that identity. Otherwise the name is derived from the payload type,
// and the key is the type's fully-qualified name (pkgPath.TypeName, as
// schema.Name defines it), so every channel carrying the same Go type shares one
// hoisted component while two packages' same-named types stay distinct.
func messageKey(m *message) string {
	if m.name != "" {
		return m.name
	}
	if t := messagePayloadType(m); t != nil && t.Name() != "" {
		return schema.Name(t)
	}
	return "message"
}

// payloadExamplesRef is a PayloadExamples call whose target is a $ref. It is
// resolved by resolvePayloadExamples after every item has been applied, because
// the component it names may be declared later in the same Spec(...) call.
type payloadExamplesRef struct {
	owner    string
	ref      string
	examples []any
}

// schemasRefPrefix is the $ref prefix of a components.schemas entry.
const schemasRefPrefix = "#/components/schemas/"

// queuePayloadExamples applies m's payload examples: immediately when the
// payload is inline, or as a deferred resolution when it is a $ref.
func (m *message) queuePayloadExamples(b *builder, payload *spec.Schema) error {
	if len(m.payloadExamples) == 0 {
		return nil
	}
	if payload.Ref == "" {
		return mergePayloadExamples(
			payload,
			messageName(m),
			"the payload schema",
			m.payloadExamples,
		)
	}
	b.pendingExamples = append(b.pendingExamples, payloadExamplesRef{
		owner:    messageName(m),
		ref:      payload.Ref,
		examples: m.payloadExamples,
	})
	return nil
}

// resolvePayloadExamples applies the deferred PayloadExamples calls. It runs
// with the other post-passes, once every Schema(...) item has registered its
// component and every payload type has been hoisted into b.defs.
func (b *builder) resolvePayloadExamples() []error {
	var errs []error
	for _, p := range b.pendingExamples {
		target, err := b.resolveSchemaRef(p.ref)
		if err != nil {
			errs = append(errs, fmt.Errorf("message.%s: examples: %w", p.owner, err))
			continue
		}
		if err := mergePayloadExamples(target, p.owner, p.ref, p.examples); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// resolveSchemaRef returns the components.schemas entry a $ref points at. Both
// maps are consulted: a payload type is hoisted into b.defs during derivation,
// while a component declared with Schema(...) lands in doc.Components.Schemas.
//
// A miss is reported rather than ignored: writing examples beside a $ref would
// emit a keyword that JSON Schema does not apply, so a document that says one
// thing and means another is worse than a build error.
func (b *builder) resolveSchemaRef(ref string) (*spec.Schema, error) {
	raw, ok := strings.CutPrefix(ref, schemasRefPrefix)
	if !ok {
		return nil, fmt.Errorf("payload %q is not a components.schemas reference", ref)
	}

	var declared map[string]*spec.Schema
	if b.doc.Components != nil {
		declared = b.doc.Components.Schemas
	}

	// The unescaped token first, then the raw one: a hoisted schema is keyed by
	// its unescaped fully-qualified name, while a hand-written $ref may name a
	// component literally.
	for _, name := range [...]string{jsonpointer.Unescape(raw), raw} {
		if s, ok := b.defs[name]; ok {
			return s, nil
		}
		if s, ok := declared[name]; ok {
			return s, nil
		}
	}
	return nil, fmt.Errorf(
		"payload %q: no components.schemas entry named %q",
		ref,
		jsonpointer.Unescape(raw),
	)
}

// mergePayloadExamples writes examples onto target, treating a repeated
// identical set as idempotent and a different one as a conflict.
//
// One hoisted component is shared by every channel carrying its payload type, so
// two messages may legitimately reach the same schema. Concatenating their
// examples would make the emitted document depend on item order, so a genuine
// disagreement is an error instead.
func mergePayloadExamples(target *spec.Schema, owner, where string, examples []any) error {
	switch {
	case len(examples) == 0:
		return nil
	case len(target.Examples) == 0:
		target.Examples = append([]any(nil), examples...)
		return nil
	case reflect.DeepEqual(target.Examples, examples):
		return nil
	default:
		return fmt.Errorf(
			"message.%s: examples: conflicts with the examples already declared on %s by another message",
			owner,
			where,
		)
	}
}
