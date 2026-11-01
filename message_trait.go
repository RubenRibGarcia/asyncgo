package asyncgo

import (
	"errors"
	"fmt"

	"github.com/RubenRibGarcia/asyncgo/spec"
)

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
// protocol helpers (see the protocol sections in bindings.go) to set a single
// protocol.
func (t *messageTrait) Bindings(bindings spec.MessageBindings) *messageTrait {
	t.t.Bindings = bindings
	return t
}

func (t *messageTrait) apply(b *builder) error {
	var errs []error
	if t.name == "" {
		errs = append(errs, fmt.Errorf("messageTrait.name: is required"))
	}
	errs = appendKeyError(errs, "messageTrait", t.name)
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
