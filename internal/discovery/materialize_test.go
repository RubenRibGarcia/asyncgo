package discovery

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/RubenRibGarcia/asyncgo/internal/logging"
	"github.com/RubenRibGarcia/asyncgo/internal/mappingconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cleanupHarnessDirs removes the harness directories a failing materialize run
// leaves behind. Keeping a failed harness is the designed behaviour — the
// generated program is the only explanation of why it would not compile — so a
// test that fails a harness deliberately has to clean up after itself.
func cleanupHarnessDirs(t *testing.T, dir string) {
	t.Helper()
	t.Cleanup(func() {
		matches, err := filepath.Glob(filepath.Join(dir, "_asyncgo-harness-*"))
		if err != nil {
			return
		}
		for _, match := range matches {
			_ = os.RemoveAll(match)
		}
	})
}

// stampPattern matches the elapsed-time prefix the logging handler puts on every
// record. Assertions strip it so a golden line does not pin a duration; the
// stamp's format is pinned once in the logging package's own tests.
var stampPattern = regexp.MustCompile(`(?m)^(asyncgo: )\+\d+\.\d{3}s `)

// withoutStamp returns s with the elapsed prefixes removed.
func withoutStamp(s string) string {
	return stampPattern.ReplaceAllString(s, "$1")
}

func TestMaterializeEmptyCatalogs(t *testing.T) {
	docs, err := Materialize(t.Context(), ".", Plan{}, Options{})
	require.NoError(t, err)
	assert.Nil(t, docs)
}

// TestMaterializeStopsOnCancelledContext pins the context plumbing: the harness
// subprocess compiles and runs user packages, so an unbounded run can hang
// generate/check forever on a stuck build or a blocking init.
func TestMaterializeStopsOnCancelledContext(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("..", "..", "test", "data", "simple"))
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	cleanupHarnessDirs(t, dir)

	cats := []Catalog{{
		PkgPath: "github.com/RubenRibGarcia/asyncgo/test/data/simple",
		VarName: "Catalog",
	}}

	docs, err := Materialize(ctx, dir, Plan{Catalogs: cats}, Options{})
	require.Error(t, err)
	assert.Nil(t, docs)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestMaterializeMkdirTempError(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "does-not-exist")
	_, err := Materialize(
		t.Context(),
		dir,
		Plan{Catalogs: []Catalog{{PkgPath: "p", VarName: "V"}}},
		Options{},
	)
	require.Error(t, err)
}

// TestMaterializeWrapsHarnessError pins the error chain: a harness subprocess
// failure has to stay reachable via errors.As/Is, since the exit status alone
// says nothing about why the build or a user init panicked.
func TestMaterializeWrapsHarnessError(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("..", "..", "test", "data", "simple"))
	require.NoError(t, err)
	cleanupHarnessDirs(t, dir)

	// The harness references the catalog var by name, so a var that does not
	// exist fails the generated program's build.
	cats := []Catalog{{
		PkgPath: "github.com/RubenRibGarcia/asyncgo/test/data/simple",
		VarName: "NoSuchCatalog",
	}}

	docs, err := Materialize(t.Context(), dir, Plan{Catalogs: cats}, Options{})
	require.Error(t, err)
	assert.Nil(t, docs)

	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr, "the subprocess failure must stay in the chain")
	assert.Contains(t, err.Error(), "materializing catalogs")
}

func TestHarness(t *testing.T) {
	t.Run("should_emit_finalize_and_marshal_without_register", func(t *testing.T) {
		src := harness(Plan{})
		assert.Contains(t, src, "package main")
		assert.Contains(t, src, "schema.Finalize")
		assert.Contains(t, src, "yaml.Marshal")
		assert.NotContains(t, src, "schema.Register")
		assert.NotContains(t, src, "schema.MapSchemaJSON")
		assert.NotContains(t, src, "schema.MapAlias")
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
		src := harness(Plan{Catalogs: cats, Refs: refs})

		assert.Contains(t, src, "schema.Register")
		assert.Contains(t, src, `pkg0 "example.com/a"`)
		assert.Contains(t, src, `pkg1 "example.com/b"`)
		assert.Contains(t, src, `pkg2 "example.com/c"`)
		assert.Contains(t, src, `pkg3 "example.com/d"`)

		// Refs are grouped by import path and rendered in order. new(T) is used
		// rather than T{} so a non-struct type (a named string, say) compiles.
		assert.Contains(t, src, "new(pkg2.T1), new(pkg2.T2)")
		assert.Contains(t, src, "new(pkg3.U1)")

		// Catalog values are referenced by their import alias.
		assert.Contains(t, src, "pkg0.Cat.ValidationErrors()")
		assert.Contains(t, src, "pkg0.Cat.Doc")
		assert.Contains(t, src, "pkg1.Cat2.ValidationErrors()")
		assert.Contains(t, src, "pkg1.Cat2.Doc")
	})

	t.Run("should_emit_mappings_and_aliases_before_register", func(t *testing.T) {
		src := harness(Plan{
			Catalogs: []Catalog{{PkgPath: "example.com/a", VarName: "Cat"}},
			Refs:     []combinatorRef{{ImportPath: "example.com/c", TypeName: "T1"}},
			Mappings: []mappingconfig.Mapping{
				{FQN: "example.com/x.UUID", Schema: `{"type":"string","format":"uuid"}`},
				{FQN: "example.com/x.Qty", Schema: `{"minimum":1,"type":"integer"}`},
			},
			Aliases: []aliasEdge{
				{Alias: "example.com/a.OrderID", Base: "example.com/x.UUID"},
			},
		})

		assert.Contains(
			t,
			src,
			`if err := schema.MapSchemaJSON("example.com/x.UUID", "{\"type\":\"string\",\"format\":\"uuid\"}"); err != nil {`,
		)
		assert.Contains(t, src,
			`schema.MapSchemaJSON("example.com/x.Qty", "{\"minimum\":1,\"type\":\"integer\"}")`)
		assert.Contains(t, src, `schema.MapAlias("example.com/a.OrderID", "example.com/x.UUID")`)

		// Config mappings and alias edges have to run before Register/Finalize, or
		// the registered types would derive before the mapping exists.
		assert.Less(t, strings.Index(src, "schema.MapSchemaJSON"),
			strings.Index(src, "schema.MapAlias"))
		assert.Less(t, strings.Index(src, "schema.MapAlias"),
			strings.Index(src, "schema.Register"))
	})

	t.Run("should_report_mapping_warnings", func(t *testing.T) {
		src := harness(Plan{})
		assert.Contains(t, src, "out.Warnings = schema.MappingWarnings()")
	})
}

func TestCatalogErrorsError(t *testing.T) {
	t.Run("should_report_errors_grouped_by_catalog", func(t *testing.T) {
		errs := CatalogErrors{
			{
				PkgPath: "example.com/app",
				VarName: "Catalog",
				Errors: []string{
					"server.prod.host: is required",
					"server.staging.host: is required",
				},
			},
		}
		assert.Equal(
			t,
			"invalid AsyncAPI catalog(s): 1\n\nexample.com/app.Catalog:\n  - server.prod.host: is required\n  - server.staging.host: is required",
			errs.Error(),
		)
	})

	t.Run("should_bullet_every_line_of_a_joined_message", func(t *testing.T) {
		errs := CatalogErrors{
			{
				PkgPath: "example.com/app",
				VarName: "Catalog",
				Errors: []string{
					"channel.orders: duplicate address\noperation.orders.send: duplicate action",
				},
			},
		}
		assert.Equal(
			t,
			"invalid AsyncAPI catalog(s): 1\n\nexample.com/app.Catalog:\n"+
				"  - channel.orders: duplicate address\n  - operation.orders.send: duplicate action",
			errs.Error(),
		)
	})
}

// TestHarnessTagsMatchTheDecoder pins the coupling between the harness source and
// harnessOutput/catalogOutcome. The generated program declares the same structs as
// string literals, and only a comment ties the two together: a tag edited in one
// place and not the other would decode silently into an empty field instead of
// failing. The test asserts the emitted source carries the exact tag lines the
// decoder expects.
func TestHarnessTagsMatchTheDecoder(t *testing.T) {
	src := harness(Plan{})

	tests := []struct {
		name string
		want string
	}{
		{"catalogs", "Catalogs []catalogOutcome `yaml:\"catalogs\"`"},
		{"pkgPath", "PkgPath string         `yaml:\"pkgPath\"`"},
		{"varName", "VarName string         `yaml:\"varName\"`"},
		{"doc", "Doc     *spec.AsyncAPI `yaml:\"doc,omitempty\"`"},
		{"errors", "Errors  []string       `yaml:\"errors,omitempty\"`"},
		{"warnings", "Warnings []string       `yaml:\"warnings,omitempty\"`"},
	}
	for _, tc := range tests {
		t.Run("should_emit_the_"+tc.name+"_tag", func(t *testing.T) {
			assert.Contains(t, src, tc.want,
				"the generated struct field must match the decoder's tag")
		})
	}
}

// TestRemoveHarnessDir pins where a cleanup failure goes. The harness directory
// is transient and its removal happens after the document has been materialized,
// so the failure is a diagnostic — but it must reach the caller's stderr rather
// than stdout (which carries the artifact messages) and it must not be silent.
func TestRemoveHarnessDir(t *testing.T) {
	t.Run("should_stay_silent_when_the_directory_is_removed", func(t *testing.T) {
		var out bytes.Buffer

		removeHarnessDir(t.TempDir(), logging.New(&out, logging.Quiet), false, os.RemoveAll)

		assert.Empty(t, out.String())
	})

	t.Run("should_report_a_cleanup_failure_on_the_writer", func(t *testing.T) {
		var out bytes.Buffer
		boom := errors.New("device busy")

		removeHarnessDir(
			"/tmp/harness",
			logging.New(&out, logging.Quiet),
			false,
			func(string) error { return boom },
		)

		assert.Equal(
			t,
			"asyncgo: removing harness directory /tmp/harness: device busy\n",
			withoutStamp(out.String()),
		)
	})
}

// TestReportMappingWarnings pins where an unused mapping surfaces and how: one
// line per mapping on the diagnostics writer, so a typo'd
// fully-qualified name in asyncgo.yaml is visible in generate/check output without
// failing the run.
func TestReportMappingWarnings(t *testing.T) {
	t.Run("should_stay_silent_when_nothing_was_left_unused", func(t *testing.T) {
		var out bytes.Buffer

		reportMappingWarnings(logging.New(&out, logging.Quiet), nil)

		assert.Empty(t, out.String())
	})

	t.Run("should_report_one_line_per_unused_mapping", func(t *testing.T) {
		var out bytes.Buffer

		reportMappingWarnings(
			logging.New(&out, logging.Quiet),
			[]string{"example.com/a.One", "example.com/b.Two"},
		)

		assert.Equal(
			t,
			"asyncgo: schema mapping for example.com/a.One was never used: "+
				"no derived type matched that name\n"+
				"asyncgo: schema mapping for example.com/b.Two was never used: "+
				"no derived type matched that name\n",
			withoutStamp(out.String()),
		)
	})
}
