package schema

import (
	"maps"
	"testing"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// restoreRegistry snapshots the package-level registry and puts it back when the
// test ends. Register writes to that global and Finalize reads it, so a test
// without this leaks its types into every test that runs after it — including
// the ones that assert on the length of the registry or on Finalize's output.
// Tests that register must run it first.
func restoreRegistry(t *testing.T) {
	t.Helper()
	saved := maps.Clone(registry)
	t.Cleanup(func() { registry = saved })
}

func TestRegister(t *testing.T) {
	restoreRegistry(t)

	t.Run("should_dereference_pointers", func(t *testing.T) {
		Register(&CustomString{})
		assert.Contains(
			t,
			registry,
			"github.com/RubenRibGarcia/asyncgo/schema.CustomString",
		)
	})

	t.Run("should_ignore_nil_types", func(t *testing.T) {
		before := len(registry)
		assert.NotPanics(t, func() { Register(nil) })
		assert.Len(t, registry, before)
	})

	t.Run("should_ignore_anonymous_types", func(t *testing.T) {
		before := len(registry)
		Register(struct{ A string }{})
		assert.Len(t, registry, before)
	})

	t.Run("should_be_idempotent", func(t *testing.T) {
		const key = "github.com/RubenRibGarcia/asyncgo/schema.CustomString"

		Register(CustomString{})
		Register(CustomString{})

		// Registering twice must produce the document one registration produces,
		// not merely leave the key present: comparing the whole schema catches a
		// second registration that overwrote the first with something else.
		doc := spec.New()
		Finalize(doc)

		require.NotNil(t, doc.Components)
		require.Contains(t, doc.Components.Schemas, key)
		assert.Equal(t, &spec.Schema{Type: "string", Format: "uuid"}, doc.Components.Schemas[key])
	})
}

func TestFinalize(t *testing.T) {
	restoreRegistry(t)

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
