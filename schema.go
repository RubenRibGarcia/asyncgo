package asyncgo

import (
	"errors"
	"fmt"

	"github.com/RubenRibGarcia/asyncgo/spec"
)

type schemasItem []*schemaDecl

// Schemas adds one or more reusable schemas to the document's
// components.schemas. Reference one from a message payload or headers via
// spec.Ref("#/components/schemas/<name>").
func Schemas(d ...*schemaDecl) Item { return schemasItem(d) }

func (s schemasItem) apply(b *builder) error {
	var errs []error
	for _, d := range s {
		if err := d.apply(b); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

type schemaDecl struct {
	name string
	s    *spec.Schema
}

// Schema declares a reusable schema under the given name: a JSON Schema Schema
// Object, or a Multi Format Schema Object built with spec.MultiFormat. Register
// it with Schemas(...) and reference it with spec.Ref(...).
func Schema(name string, s *spec.Schema) *schemaDecl {
	return &schemaDecl{name: name, s: s}
}

func (d *schemaDecl) apply(b *builder) error {
	var errs []error
	if d.name == "" {
		errs = append(errs, fmt.Errorf("schema.name: is required"))
	}
	errs = appendKeyError(errs, "schema", d.name)
	if d.s == nil {
		errs = append(errs, fmt.Errorf("schema.%s: is required", d.name))
	}
	c := b.components()
	if _, dup := c.Schemas[d.name]; dup && d.name != "" {
		errs = append(errs, fmt.Errorf("schema.%s: duplicate name", d.name))
	} else if d.name != "" && d.s != nil {
		// Register-anyway policy (see appendKeyError): a component is registered
		// even when its name or content was rejected, so a $ref written against it
		// stays resolvable and the error is the one thing the caller has to fix.
		// A nil schema is the single exception — "null" is not a Schema Object, so
		// there is nothing to register and the error above already names it.
		c.Schemas[d.name] = d.s
	}
	return errors.Join(errs...)
}
