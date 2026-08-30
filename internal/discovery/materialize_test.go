package discovery

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaterializeEmptyCatalogs(t *testing.T) {
	docs, err := Materialize(".", nil, nil)
	require.NoError(t, err)
	assert.Nil(t, docs)
}

func TestMaterializeMkdirTempError(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "does-not-exist")
	_, err := Materialize(dir, []Catalog{{PkgPath: "p", VarName: "V"}}, nil)
	require.Error(t, err)
}

func TestHarness(t *testing.T) {
	t.Run("should_emit_finalize_and_marshal_without_register", func(t *testing.T) {
		src := harness(nil, nil)
		assert.Contains(t, src, "package main")
		assert.Contains(t, src, "schema.Finalize")
		assert.Contains(t, src, "yaml.Marshal")
		assert.NotContains(t, src, "schema.Register")
	})

	t.Run("should_emit_imports_and_register_for_cats_and_refs", func(t *testing.T) {
		cats := []Catalog{
			{PkgPath: "example.com/a", VarName: "Cat"},
			{PkgPath: "example.com/b", VarName: "Cat2"},
		}
		refs := []combinatorRef{
			{ImportPath: "example.com/c", TypeName: "T1"},
			{ImportPath: "example.com/c", TypeName: "T2"},
			{ImportPath: "example.com/d", TypeName: "U1"},
		}
		src := harness(cats, refs)

		assert.Contains(t, src, "schema.Register")
		assert.Contains(t, src, `pkg0 "example.com/a"`)
		assert.Contains(t, src, `pkg1 "example.com/b"`)
		assert.Contains(t, src, `pkg2 "example.com/c"`)
		assert.Contains(t, src, `pkg3 "example.com/d"`)

		// Refs are grouped by import path and rendered in order.
		assert.Contains(t, src, "pkg2.T1{}, pkg2.T2{}")
		assert.Contains(t, src, "pkg3.U1{}")

		// Catalog values are referenced by their import alias.
		assert.Contains(t, src, "pkg0.Cat,")
		assert.Contains(t, src, "pkg1.Cat2,")
	})
}
