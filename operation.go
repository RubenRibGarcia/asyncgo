package asyncgo

import (
	"github.com/RubenRibGarcia/asyncgo/internal/jsonpointer"
	"github.com/RubenRibGarcia/asyncgo/spec"
)

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

// build assembles the Operation Object o declares and writes the messages it
// carries into ch, returning the errors found on the way. It mirrors
// message.build: registration is the caller's, so a duplicate action does not
// stop the operations declared after it from being built.
func (o *operation) build(
	b *builder,
	ch *spec.Channel,
	address string,
) (*spec.Operation, []error) {
	var errs []error
	specOp := &spec.Operation{
		Action:       o.action,
		Channel:      &spec.Reference{Ref: channelsRefPrefix + jsonpointer.Escape(address)},
		Title:        o.title,
		Summary:      o.summary,
		Description:  o.description,
		Tags:         o.tags,
		ExternalDocs: o.externalDocs,
		Security:     o.security,
		Bindings:     o.bindings,
		Traits:       o.traits,
		Reply:        o.replyReference(),
	}
	for _, m := range o.messages {
		sm, err := m.build(b)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		key := messageKey(m)
		// The key becomes the message's components.messages key, which the
		// specification patterns. A key derived from the payload type's fully
		// qualified name is exempt: appendKeyError skips an unauthored name.
		errs = appendKeyError(errs, "message", m.name)
		entry, err := hoistMessage(b, key, "channel."+address, sm)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if ch.Messages == nil {
			ch.Messages = map[string]*spec.Message{}
		}
		ch.Messages[key] = entry
		specOp.Messages = append(specOp.Messages, &spec.Reference{
			Ref: channelsRefPrefix + jsonpointer.Escape(
				address,
			) + channelMessagesSegment + jsonpointer.Escape(
				key,
			),
		})
	}
	errs = appendBindingErrors(errs, "operation."+address+"."+o.action, o.bindings.Validate())
	return specOp, errs
}
