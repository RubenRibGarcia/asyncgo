package asyncgo

import (
	"errors"
	"fmt"

	"github.com/RubenRibGarcia/asyncgo/spec"
)

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
// protocol helpers (see the protocol sections in bindings.go) to set a single
// protocol.
func (t *operationTrait) Bindings(bindings spec.OperationBindings) *operationTrait {
	t.t.Bindings = bindings
	return t
}

func (t *operationTrait) apply(b *builder) error {
	var errs []error
	if t.name == "" {
		errs = append(errs, fmt.Errorf("operationTrait.name: is required"))
	}
	errs = appendKeyError(errs, "operationTrait", t.name)
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
