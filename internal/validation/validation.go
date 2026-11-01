// Package validation checks a generated AsyncAPI document against the
// official AsyncAPI 3.1.0 JSON Schema.
//
// The schema is vendored under this directory (see PROVENANCE.md), embedded
// into the binary, and never fetched at run time. It is compiled once per
// process and reused.
//
// One deliberate deviation from the published schema is applied in memory
// before compiling: the "Relax component key patterns" rule, see
// relaxComponentKeys and docs/adr/0007-vendored-schema-validation.md.
package validation

import (
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/RubenRibGarcia/asyncgo/internal/jsonpointer"
	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

//go:embed asyncapi-3.1.0.json
var schemaJSON []byte

const (
	// schemaRootURL is the $id of the embedded document. It must match, because
	// the compiler registers the document under it.
	schemaRootURL = "http://asyncapi.com/definitions/3.1.0/asyncapi.json"

	// componentsURL is the $id of the Components Object definition whose key
	// patterns are relaxed before compiling (see relaxComponentKeys).
	componentsURL = "http://asyncapi.com/definitions/3.1.0/components.json"

	// pinnedSHA256 is the SHA-256 digest of the vendored schema file, recorded
	// in PROVENANCE.md. A test fails when the embedded bytes stop matching it,
	// so an edit to the vendored file is a deliberate change rather than a
	// silent one.
	pinnedSHA256 = "51d3274899ad2875f25c18fd1aef4d5512f0a97be785d519740bde55a4162f61"
)

// omittedBindings lists the protocol binding keys the published 3.1.0 schema
// does not describe. Their bindings are specified in their own repositories and
// are absent from the released schema bundle, so a document that uses them is
// rejected on the key alone. Options.IgnoreMissingBindings suppresses exactly
// these keys and nothing else.
var omittedBindings = map[string]bool{
	"mqtt5":   true,
	"mercure": true,
}

// printer renders the validator's localized error messages.
var printer = message.NewPrinter(language.English)

// compiledSchema compiles the embedded schema once per process.
var compiledSchema = sync.OnceValues(compile)

// Options controls which violations are reported.
type Options struct {
	// IgnoreMissingBindings suppresses violations that only report a protocol
	// binding key the published 3.1.0 schema omits (mqtt5 and mercure). It does
	// not suppress anything inside those bindings, and no other key is affected.
	IgnoreMissingBindings bool
}

// Violation is one schema violation, cited by the node it applies to.
type Violation struct {
	// Pointer is the RFC 6901 JSON Pointer of the offending node. The empty
	// location (the whole document) renders as "/".
	Pointer string

	// Message describes the violation and the constraint it broke.
	Message string
}

// Violations reports every schema violation found in a document.
type Violations []Violation

// Error renders one line per violation.
func (v Violations) Error() string {
	var b strings.Builder
	b.WriteString("document is not valid AsyncAPI 3.1.0:")
	for _, violation := range v {
		fmt.Fprintf(&b, "\n  %s: %s", violation.Pointer, violation.Message)
	}
	return b.String()
}

// Validate checks doc against the official AsyncAPI 3.1.0 JSON Schema.
//
// The document is encoded to JSON and validated in its JSON form, because JSON
// Schema constrains the JSON data model rather than a YAML rendering of it.
// A document with violations returns a [Violations] error; a nil error means the
// document is valid under opts.
func Validate(doc *spec.AsyncAPI, opts Options) error {
	if doc == nil {
		return errors.New("validating: nil document")
	}

	schema, err := compiledSchema()
	if err != nil {
		return fmt.Errorf("validating: %w", err)
	}

	instance, err := instanceOf(doc)
	if err != nil {
		return fmt.Errorf("validating: %w", err)
	}

	if err := schema.Validate(instance); err != nil {
		var invalid *jsonschema.ValidationError
		if !errors.As(err, &invalid) {
			return fmt.Errorf("validating: %w", err)
		}
		if violations := collect(invalid, opts); len(violations) > 0 {
			return violations
		}
	}
	return nil
}

// compile builds the schema from the embedded bytes.
func compile() (*jsonschema.Schema, error) {
	if digest := fmt.Sprintf("%x", sha256.Sum256(schemaJSON)); digest != pinnedSHA256 {
		return nil, fmt.Errorf(
			"embedded schema digest %s does not match the pinned %s",
			digest,
			pinnedSHA256,
		)
	}

	root := make(map[string]any)
	if err := json.Unmarshal(schemaJSON, &root); err != nil {
		return nil, fmt.Errorf("decoding AsyncAPI 3.1.0 schema: %w", err)
	}
	relaxComponentKeys(root)

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(schemaRootURL, root); err != nil {
		return nil, fmt.Errorf("registering AsyncAPI 3.1.0 schema: %w", err)
	}
	schema, err := compiler.Compile(schemaRootURL)
	if err != nil {
		return nil, fmt.Errorf("compiling AsyncAPI 3.1.0 schema: %w", err)
	}
	return schema, nil
}

// relaxComponentKeys makes the schema validate component entries whose key does
// not match the specification's key pattern.
//
// Every components.<field> map in the published schema restricts its keys with
// patternProperties "^[\w\d\.\-_]+$" and declares no additionalProperties.
// asyncgo keys hoisted schemas "<package path>.<type name>", and a package path
// contains "/", so those entries match no pattern and are never validated — the
// published schema, and the AsyncAPI CLI, consider an invalid hoisted schema
// valid. Copying each pattern's value into additionalProperties on the same map
// keeps the official value subschema in force for every key.
//
// This only widens which keys are reachable; it does not weaken what they are
// checked against. See ADR 0007.
func relaxComponentKeys(root map[string]any) {
	definitions, ok := root["definitions"].(map[string]any)
	if !ok {
		return
	}
	components, ok := definitions[componentsURL].(map[string]any)
	if !ok {
		return
	}
	properties, ok := components["properties"].(map[string]any)
	if !ok {
		return
	}

	for _, raw := range properties {
		entry, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		patterns, ok := entry["patternProperties"].(map[string]any)
		if !ok || len(patterns) != 1 {
			continue
		}
		if _, ok := entry["additionalProperties"]; ok {
			continue
		}
		for _, value := range patterns {
			entry["additionalProperties"] = value
		}
	}
}

// instanceOf decodes doc's JSON encoding into the value shape the validator
// expects.
func instanceOf(doc *spec.AsyncAPI) (any, error) {
	raw, err := doc.JSON()
	if err != nil {
		return nil, fmt.Errorf("encoding document: %w", err)
	}
	var instance any
	if err := json.Unmarshal(raw, &instance); err != nil {
		return nil, fmt.Errorf("decoding document: %w", err)
	}
	return instance, nil
}

// leaf is one leaf cause of the validation error tree.
type leaf struct {
	errorKind jsonschema.ErrorKind
	violation Violation
	// missingRef marks a "missing property $ref" leaf: the Reference branch of a
	// "Reference-or-inline object" union failing.
	missingRef bool
}

// collect flattens the validator's error tree into a deterministic, de-duplicated
// report.
func collect(root *jsonschema.ValidationError, opts Options) Violations {
	var leaves []leaf

	var walk func(err *jsonschema.ValidationError)
	walk = func(err *jsonschema.ValidationError) {
		if len(err.Causes) == 0 {
			if collected, ok := report(err); ok {
				leaves = append(leaves, collected)
			}
			return
		}
		for _, cause := range err.Causes {
			walk(cause)
		}
	}
	walk(root)

	// A "missing $ref" leaf is an artifact of the union's Reference branch when
	// the same node also has a deeper violation — the other branch explains why.
	// On its own it is a Reference Object with no target, which is a real
	// violation.
	artifacts := make(map[string]bool, len(leaves))
	for _, l := range leaves {
		if !l.missingRef {
			// A sibling violation at the same node also explains the union.
			artifacts[l.violation.Pointer] = true
		}
		for i := strings.LastIndex(l.violation.Pointer, "/"); i > 0; {
			artifacts[l.violation.Pointer[:i]] = true
			i = strings.LastIndex(l.violation.Pointer[:i], "/")
		}
	}

	out := make(Violations, 0, len(leaves))
	reported := make(map[string]bool, len(leaves))
	for _, l := range leaves {
		if l.missingRef && artifacts[l.violation.Pointer] {
			continue
		}
		if opts.IgnoreMissingBindings && isOmittedBinding(l) {
			continue
		}
		if reported[l.violation.Pointer] {
			continue
		}
		reported[l.violation.Pointer] = true
		out = append(out, l.violation)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Pointer != out[j].Pointer {
			return out[i].Pointer < out[j].Pointer
		}
		return out[i].Message < out[j].Message
	})
	return out
}

// report turns one leaf of the error tree into a leaf, or reports false when the
// leaf carries no information of its own.
func report(err *jsonschema.ValidationError) (leaf, bool) {
	if _, ok := err.ErrorKind.(*kind.Reference); ok {
		// "This $ref failed": the branch's own causes carry the detail.
		return leaf{}, false
	}

	missingRef := false
	if keyword, ok := err.ErrorKind.(*kind.Required); ok {
		missingRef = len(keyword.Missing) == 1 && keyword.Missing[0] == "$ref"
	}

	return leaf{
		errorKind: err.ErrorKind,
		violation: Violation{
			Pointer: pointer(err.InstanceLocation),
			Message: err.ErrorKind.LocalizedString(printer),
		},
		missingRef: missingRef,
	}, true
}

// isOmittedBinding reports whether the violation only says that a bindings
// object carries a protocol key the published schema does not describe.
func isOmittedBinding(l leaf) bool {
	additional, ok := l.errorKind.(*kind.AdditionalProperties)
	if !ok || len(additional.Properties) == 0 {
		return false
	}
	if !strings.HasSuffix(l.violation.Pointer, "/bindings") {
		return false
	}
	for _, name := range additional.Properties {
		if !omittedBindings[name] {
			return false
		}
	}
	return true
}

// pointer renders an instance location as an RFC 6901 JSON Pointer.
func pointer(location []string) string {
	if len(location) == 0 {
		return "/"
	}
	escaped := make([]string, len(location))
	for i, token := range location {
		escaped[i] = jsonpointer.Escape(token)
	}
	return "/" + strings.Join(escaped, "/")
}
