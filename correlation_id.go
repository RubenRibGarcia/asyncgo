package asyncgo

import (
	"errors"
	"fmt"

	"github.com/RubenRibGarcia/asyncgo/spec"
)

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
	errs = appendKeyError(errs, "correlationId", c.name)
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
