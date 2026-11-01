// Package asyncgo is the developer-facing fluent DSL for declaring an AsyncAPI
// document. A catalog is a package-level variable of type *SpecResult built
// with Spec(...); the asyncgo CLI discovers such variables reachable from main
// and statically interprets them into an AsyncAPI document.
package asyncgo

import (
	"errors"
	"fmt"
	"maps"
	"regexp"

	"github.com/RubenRibGarcia/asyncgo/spec"
)

// Item is a fragment of an AsyncAPI document. The concrete builder types in
// this package implement it; the method is unexported so the set of Items is
// closed — users compose them only via the exported constructor functions.
type Item interface {
	apply(b *builder) error
}

// builder accumulates a document and the hoisted schemas it references, plus
// the PayloadExamples resolutions deferred to the end of the build.
type builder struct {
	doc  *spec.AsyncAPI
	defs map[string]*spec.Schema

	// pendingExamples holds PayloadExamples calls whose target is a $ref. They
	// are applied by resolvePayloadExamples once every item has run, since a
	// message may reference a component declared later in the same Spec() call.
	pendingExamples []payloadExamplesRef
}

// SpecResult is the outcome of building a catalog: the assembled document plus
// any validation errors encountered while applying its items. Doc is always
// non-nil; Err is nil when the catalog is valid.
type SpecResult struct {
	Doc  *spec.AsyncAPI
	Err  error
	errs []error
}

// ValidationErrors returns the individual validation errors that Err joins, or
// nil when the catalog is valid. The generator harness uses this to render the
// per-catalog report.
func (r *SpecResult) ValidationErrors() []error { return r.errs }

// Spec assembles an AsyncAPI document from the given fragments.
func Spec(items ...Item) *SpecResult {
	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}
	var errs []error
	for _, it := range items {
		if err := it.apply(b); err != nil {
			errs = append(errs, err)
		}
	}
	errs = append(errs, b.validateServerRefs()...)
	errs = append(errs, b.resolvePayloadExamples()...)
	errs = append(errs, b.validateSecurityRefs()...)
	errs = append(errs, b.validateReplyRefs()...)
	errs = append(errs, b.validateMessageRefs()...)
	errs = append(errs, b.validateOperationMessages()...)
	errs = append(errs, b.validateTraitRefs()...)
	errs = append(errs, b.validateSchemaNodes()...)
	if len(b.defs) > 0 {
		c := b.components()
		maps.Copy(c.Schemas, b.defs)
	}
	return &SpecResult{Doc: b.doc, Err: errors.Join(errs...), errs: errs}
}

func (b *builder) components() *spec.Components {
	if b.doc.Components == nil {
		b.doc.Components = &spec.Components{}
	}
	if b.doc.Components.Schemas == nil {
		b.doc.Components.Schemas = map[string]*spec.Schema{}
	}
	return b.doc.Components
}

// componentKeyPattern is the key pattern the AsyncAPI 3.1.0 schema applies to
// every components.* map, where it is expressed as patternProperties. A key that
// does not match is therefore not rejected by a downstream validator — the
// schema constrains keys this way rather than with additionalProperties: false
// — so this check is the only place the constraint is enforced, the same
// situation validateReplyRefs documents for the address/channel MUST.
const componentKeyPatternSrc = `^[\w\d.\-_]+$`

var componentKeyPattern = regexp.MustCompile(componentKeyPatternSrc)

// validateKey reports a component key that does not match the specification's
// pattern. kind names the component map ("schema", "securityScheme", ...) so
// the message names the offending object. Keys derived from a Go type's fully
// qualified name are never passed here: they carry the package path, whose
// separators the pattern rejects, and that kept deviation is recorded under B11
// in docs/asyncapi-3.1.0-coverage.md.
func validateKey(kind, name string) error {
	if componentKeyPattern.MatchString(name) {
		return nil
	}
	return fmt.Errorf("%s.%s: name must match %s", kind, name, componentKeyPatternSrc)
}

// appendKeyError appends the key-pattern violation of an authored component
// name, if any, so each component apply runs the check in one line. An empty
// name is left to the caller's required-field check, so a single mistake is not
// reported twice.
func appendKeyError(errs []error, kind, name string) []error {
	if name == "" {
		return errs
	}
	if err := validateKey(kind, name); err != nil {
		errs = append(errs, err)
	}
	return errs
}
