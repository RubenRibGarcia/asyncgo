package schema

import (
	"reflect"
	"strings"

	"github.com/RubenRibGarcia/asyncgo/internal/jsonpointer"
	"github.com/RubenRibGarcia/asyncgo/spec"
)

// schemasRefPrefix is the JSON Pointer prefix of a reference into
// components.schemas, the only shape a derivation emits.
const schemasRefPrefix = "#/components/schemas/"

// payloadDerivation records one message-payload derivation, so Finalize can redo
// it once every mapping tier has registered.
type payloadDerivation struct {
	typ   reflect.Type            // the payload type, as declared
	node  *spec.Schema            // the node the message holds
	owned map[string]*spec.Schema // components this derivation hoisted, by name
}

// deferredPayloads maps a document to the payload derivations its catalog made.
// Entries are consumed by Finalize, so a document that is never finalized leaves
// its record behind — the same lifetime the package's other registries assume.
var deferredPayloads = map[*spec.AsyncAPI][]payloadDerivation{}

// FromTypePayload derives the schema for a message payload type and records the
// derivation, so Finalize can redo it with every declared mapping in place.
//
// It exists because payload derivation happens too early for mappings to
// participate. A catalog is a package-level variable: Spec builds it, and this
// derivation runs, while the catalog package's own variables are being
// initialized. A declared mapping cannot be registered by then — a project's
// asyncgo.yaml mappings and the alias edges discovery finds are registered later,
// in the generated harness — so a payload derived with FromType would miss them
// while the same type reached from Finalize's combinator hoisting would not.
//
// Deriving here anyway is not wasted work: Spec's validation, its auto-hoist
// collision check, and PayloadExamples all read the derived schema, and a document
// with no payload schemas would be incomplete for every caller that does not
// finalize. Finalize therefore treats this result as provisional: it re-derives
// the type, rewrites node's contents in place, and replaces the components it
// hoisted — dropping the ones a mapping made obsolete.
//
// It is the payload entry point of FromType and is not otherwise special: pass
// the document the schema will belong to, the payload type, and the defs map the
// catalog hoists into.
func FromTypePayload(
	doc *spec.AsyncAPI,
	t reflect.Type,
	defs map[string]*spec.Schema,
) *spec.Schema {
	before := make(map[string]bool, len(defs))
	for name := range defs {
		before[name] = true
	}

	s := FromType(t, defs)
	if doc == nil || t == nil {
		return s
	}

	owned := make(map[string]*spec.Schema)
	for name, node := range defs {
		if !before[name] {
			owned[name] = node
		}
	}
	deferredPayloads[doc] = append(
		deferredPayloads[doc],
		payloadDerivation{typ: t, node: s, owned: owned},
	)
	return s
}

// rederivePayloads replaces doc's provisional payload schemas with derivations
// made now — after every mapping tier has registered — and drops the components
// that only the provisional derivation produced.
//
// Examples are the one annotation Spec writes onto a derived schema
// (MessageOf's PayloadExamples): replacing the node would drop them, so they are
// carried across from the provisional node. A mapping that declares its own
// `examples` is overridden by the message's, because the message's examples are
// the more specific declaration and Spec already proved they do not conflict.
func rederivePayloads(doc *spec.AsyncAPI) {
	sites, ok := deferredPayloads[doc]
	if !ok {
		return
	}
	delete(deferredPayloads, doc)

	for _, site := range sites {
		examples := provisionalExamples(doc, site)

		fresh := map[string]*spec.Schema{}
		derived := FromType(site.typ, fresh)

		var components map[string]*spec.Schema
		if doc.Components != nil {
			components = doc.Components.Schemas
		}
		if len(fresh) > 0 && components == nil {
			if doc.Components == nil {
				doc.Components = &spec.Components{}
			}
			if doc.Components.Schemas == nil {
				doc.Components.Schemas = map[string]*spec.Schema{}
			}
			components = doc.Components.Schemas
		}
		for name, node := range fresh {
			components[name] = node
		}
		for name, provisional := range site.owned {
			if _, kept := fresh[name]; kept {
				continue
			}
			// Identity, not the name: a component another derivation already
			// replaced must survive, since the re-derivation that replaced it is
			// as current as this one.
			if components[name] == provisional {
				delete(components, name)
			}
		}

		if len(examples) > 0 {
			target := derived
			if name, ok := componentName(derived.Ref); ok && fresh[name] != nil {
				target = fresh[name]
			}
			target.Examples = examples
		}
		*site.node = *derived
	}
}

// provisionalExamples returns the examples the provisional derivation carries: on
// the payload node itself for an inline schema, or on the component its $ref
// names. The owned entry is preferred because it is the node Spec wrote them
// onto, even if a shared type's earlier re-derivation replaced it in the document.
func provisionalExamples(doc *spec.AsyncAPI, site payloadDerivation) []any {
	target := site.node
	if name, ok := componentName(site.node.Ref); ok {
		if owned := site.owned[name]; owned != nil {
			target = owned
		} else if doc.Components != nil && doc.Components.Schemas[name] != nil {
			target = doc.Components.Schemas[name]
		}
	}
	return target.Examples
}

// componentName returns the components.schemas key a $ref names, or ok=false when
// ref points anywhere else.
func componentName(ref string) (string, bool) {
	raw, ok := strings.CutPrefix(ref, schemasRefPrefix)
	if !ok || raw == "" {
		return "", false
	}
	return jsonpointer.Unescape(raw), true
}
