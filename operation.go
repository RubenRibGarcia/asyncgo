package asyncgo

import "github.com/RubenRibGarcia/asyncgo/spec"

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
