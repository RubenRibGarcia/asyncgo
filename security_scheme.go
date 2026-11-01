package asyncgo

import (
	"errors"
	"fmt"

	"github.com/RubenRibGarcia/asyncgo/spec"
)

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
	errs = appendKeyError(errs, "securityScheme", s.name)
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
