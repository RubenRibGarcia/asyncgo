package schema

import (
	"testing"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
)

func TestRegister(t *testing.T) {
	t.Run("should_dereference_pointers", func(t *testing.T) {
		Register(&CustomString{})
		assert.Contains(
			t,
			registry,
			"github.com/RubenRibGarcia/asyncgo/schema.CustomString",
		)
	})

	t.Run("should_ignore_anonymous_types", func(t *testing.T) {
		before := len(registry)
		Register(struct{ A string }{})
		assert.Len(t, registry, before)
	})

	t.Run("should_be_idempotent", func(t *testing.T) {
		Register(CustomString{})
		Register(CustomString{})
		assert.Contains(
			t,
			registry,
			"github.com/RubenRibGarcia/asyncgo/schema.CustomString",
		)
	})
}

func TestFinalize(t *testing.T) {
	t.Run("should_be_noop_on_nil_document", func(t *testing.T) {
		assert.NotPanics(t, func() { Finalize(nil) })
	})

	t.Run("should_not_overwrite_existing_schemas", func(t *testing.T) {
		Register(CustomString{})
		const key = "github.com/RubenRibGarcia/asyncgo/schema.CustomString"

		sentinel := &spec.Schema{Type: "sentinel"}
		doc := spec.New()
		doc.Components = &spec.Components{
			Schemas: map[string]*spec.Schema{key: sentinel},
		}

		Finalize(doc)

		assert.Same(t, sentinel, doc.Components.Schemas[key])
		assert.Equal(t, "sentinel", doc.Components.Schemas[key].Type)
	})
}
