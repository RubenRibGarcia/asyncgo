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
	Doc *spec.AsyncAPI
	Err error

	// Name is the document name set by Named. It labels this catalog's own
	// document and, when generate writes into a directory, its filename. It is
	// empty when the catalog declares no name, in which case the generator
	// derives one from the declaring package and variable.
	Name string
}

// Named sets the catalog's document name and returns the same result, so it
// chains directly off Spec:
//
//	var Orders = asyncgo.Spec(...).Named("orders")
//
// The name is what `asyncgo generate -o docs/` writes this catalog's document
// as — docs/orders.asyncapi.yaml — so it is the one place a multi-catalog module
// chooses its filenames instead of accepting the derived default
// (<package>.<variable>). An empty name clears an earlier one, restoring the
// derived name.
//
// It is a method rather than an Item because it labels a catalog that is already
// assembled: a bare Name among Info, Servers and Channels would read as if it
// named a component.
func (r *SpecResult) Named(name string) *SpecResult {
	r.Name = name
	if err := validateDocumentName(name); err != nil {
		r.Err = errors.Join(r.Err, err)
	}
	return r
}

// validateDocumentName reports a name that cannot label a document. The grammar
// is the component-key pattern, which is also what makes the name safe as a
// filename stem: no separator, no traversal, no character that needs escaping
// in a path. An empty name is not a violation — it means "no explicit name",
// which generate replaces with the derived one.
func validateDocumentName(name string) error {
	if name == "" || componentKeyPattern.MatchString(name) {
		return nil
	}
	return fmt.Errorf("document name %q must match %s", name, componentKeyPatternSrc)
}

// ValidationErrors returns the individual validation errors that Err joins, or
// nil when the catalog is valid. The generator harness uses this to render the
// per-catalog report. Err is already the join of them (errors.Join exposes them
// through Unwrap() []error), so there is no second copy to keep in step.
func (r *SpecResult) ValidationErrors() []error {
	if r == nil || r.Err == nil {
		return nil
	}
	if joined, ok := r.Err.(interface{ Unwrap() []error }); ok {
		return joined.Unwrap()
	}
	return []error{r.Err}
}

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
	return &SpecResult{Doc: b.doc, Err: errors.Join(errs...)}
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
//
// It is also where the project's register-anyway policy is decided: when a
// component's name or content fails validation, the component is still registered
// (whenever it has content to register). The document is invalid either way, but
// keeping the entry means every $ref written against it still resolves, so the
// caller fixes the reported error instead of also chasing a dangling reference.
// ref_test.go pins it: a reply whose name breaks the key pattern is registered
// and its $ref stays a valid JSON Pointer. A nil value is the one exception —
// null is not a valid component, so nothing is registered.
func appendKeyError(errs []error, kind, name string) []error {
	if name == "" {
		return errs
	}
	if err := validateKey(kind, name); err != nil {
		errs = append(errs, err)
	}
	return errs
}
