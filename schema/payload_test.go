package schema

import (
	"maps"
	"reflect"
	"testing"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mappingPayload is a message payload whose mapping-sensitive field is derived by
// kind until Finalize redoes the derivation.
type mappingPayload struct {
	ID  mappingUUID   `json:"id"`
	Rec mappingRecord `json:"rec"`
}

var mappingPayloadName = Name(reflect.TypeFor[mappingPayload]())

// buildProvisional derives a payload the way a catalog does — before any mapping
// is registered — and copies the hoisted schemas into the document the way Spec
// does, returning the document the deferred derivation is attached to.
func buildProvisional(t *testing.T, typ reflect.Type) (*spec.AsyncAPI, map[string]*spec.Schema) {
	t.Helper()

	doc := spec.New()
	defs := map[string]*spec.Schema{}
	FromTypePayload(doc, typ, defs)
	doc.Components = &spec.Components{Schemas: maps.Clone(defs)} // shares the nodes
	return doc, defs
}

func TestRederivePayloads(t *testing.T) {
	isolated(t, "should_rederive_a_payload_with_the_mapping_registered_later", func(t *testing.T) {
		doc, _ := buildProvisional(t, reflect.TypeFor[mappingPayload]())

		// Before Finalize the document carries the provisional, kind-derived schema.
		require.Contains(t, doc.Components.Schemas, mappingPayloadName)
		id := doc.Components.Schemas[mappingPayloadName].Properties["id"]
		require.NotNil(t, id)
		assert.Equal(t, "array", id.Type)

		// The harness registers asyncgo.yaml's mappings after the catalogs are built.
		MapSchemaByName(mappingUUIDName, uuidSchema())
		Finalize(doc)

		id = doc.Components.Schemas[mappingPayloadName].Properties["id"]
		require.NotNil(t, id)
		assert.Equal(t, "string", id.Type)
		assert.Equal(t, "uuid", id.Format)
	})

	isolated(t, "should_inline_a_payload_whose_whole_type_is_mapped", func(t *testing.T) {
		doc, _ := buildProvisional(t, reflect.TypeFor[mappingRecord]())
		require.Contains(t, doc.Components.Schemas, mappingRecordName)

		MapSchemaByName(mappingRecordName, &spec.Schema{Type: "string", Format: "csv"})
		Finalize(doc)

		// A mapped type is inlined at every use site, so the provisional component
		// is obsolete — and a component nothing references is noise in the document.
		assert.NotContains(t, doc.Components.Schemas, mappingRecordName)
	})

	isolated(t, "should_drop_a_component_a_mapping_made_obsolete", func(t *testing.T) {
		doc, _ := buildProvisional(t, reflect.TypeFor[mappingPayload]())
		require.Contains(t, doc.Components.Schemas, mappingRecordName)

		MapSchemaByName(mappingRecordName, &spec.Schema{Type: "string", Format: "csv"})
		Finalize(doc)

		assert.NotContains(t, doc.Components.Schemas, mappingRecordName)
		rec := doc.Components.Schemas[mappingPayloadName].Properties["rec"]
		require.NotNil(t, rec)
		assert.Equal(t, "csv", rec.Format)
	})

	isolated(t, "should_preserve_payload_examples_on_a_hoisted_payload", func(t *testing.T) {
		doc, defs := buildProvisional(t, reflect.TypeFor[mappingPayload]())
		// PayloadExamples writes onto the hoisted component, as resolvePayloadExamples
		// does for a $ref payload.
		examples := []any{map[string]any{"id": "x"}}
		defs[mappingPayloadName].Examples = examples

		MapSchemaByName(mappingUUIDName, uuidSchema())
		Finalize(doc)

		assert.Equal(t, examples, doc.Components.Schemas[mappingPayloadName].Examples)
	})

	isolated(t, "should_preserve_payload_examples_on_an_inline_payload", func(t *testing.T) {
		doc := spec.New()
		defs := map[string]*spec.Schema{}
		node := FromTypePayload(doc, reflect.TypeFor[mappingUUID](), defs)
		// Pin the node into a message, the way message.build does.
		msg := &spec.Message{Payload: node}
		examples := []any{"018f"}
		node.Examples = examples

		MapSchemaByName(mappingUUIDName, uuidSchema())
		Finalize(doc)

		assert.Equal(t, "uuid", msg.Payload.Format)
		assert.Equal(t, examples, msg.Payload.Examples)
	})

	isolated(t, "should_rewrite_the_payload_node_in_place", func(t *testing.T) {
		doc := spec.New()
		node := FromTypePayload(doc, reflect.TypeFor[mappingPayload](), map[string]*spec.Schema{})
		msg := &spec.Message{Payload: node}
		require.NotEmpty(t, msg.Payload.Ref)

		MapSchemaByName(mappingUUIDName, uuidSchema())
		Finalize(doc)

		// The message holds the same node it always did; Finalize rewrote its
		// contents rather than replacing the pointer.
		assert.Same(t, node, msg.Payload)
		assert.Equal(t, msg.Payload.Ref, node.Ref)
	})

	isolated(t, "should_consume_the_deferred_derivation", func(t *testing.T) {
		doc, _ := buildProvisional(t, reflect.TypeFor[mappingPayload]())
		require.Contains(t, deferredPayloads, doc)

		Finalize(doc)

		assert.NotContains(t, deferredPayloads, doc)
	})

	isolated(t, "should_leave_a_document_without_deferred_derivations_alone", func(t *testing.T) {
		doc := spec.New()
		assert.NotPanics(t, func() { Finalize(doc) })
		assert.Nil(t, doc.Components)
	})

	isolated(t, "should_derive_a_payload_without_a_document", func(t *testing.T) {
		// A nil document cannot key a deferred derivation; the schema still derives.
		s := FromTypePayload(nil, reflect.TypeFor[mappingUUID](), map[string]*spec.Schema{})

		assert.Equal(t, "array", s.Type)
		assert.Empty(t, deferredPayloads)
	})
}
