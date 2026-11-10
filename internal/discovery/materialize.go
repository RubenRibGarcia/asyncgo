package discovery

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/RubenRibGarcia/asyncgo/internal/logging"
	"github.com/RubenRibGarcia/asyncgo/internal/mappingconfig"
	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/goccy/go-yaml"
)

// Plan is everything the generated harness needs to materialize a module's
// catalogs: the catalog variables to import, the types combinator directives
// reference by name, the schema mappings declared in asyncgo.yaml, and the alias
// edges derived from the module's own type declarations.
//
// It exists so the pieces discovery resolves statically travel to the harness as
// one value — each is emitted into the generated program in that order, and a
// caller cannot half-populate them by mistake.
type Plan struct {
	Catalogs []Catalog
	Refs     []combinatorRef
	Mappings []mappingconfig.Mapping
	Aliases  []aliasEdge
}

// CatalogError groups the validation errors of a single catalog.
type CatalogError struct {
	PkgPath string
	VarName string
	Errors  []string
}

// CatalogErrors reports validation errors grouped by catalog.
type CatalogErrors []CatalogError

// Error renders a per-catalog report.
func (e CatalogErrors) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "invalid AsyncAPI catalog(s): %d\n", len(e))
	for _, c := range e {
		fmt.Fprintf(&b, "\n%s.%s:\n", c.PkgPath, c.VarName)
		for _, msg := range c.Errors {
			// A message may carry a multi-error: an Item that reports several
			// violations joins them into one entry. Bullet every line, so each
			// violation reads as its own item.
			for _, line := range strings.Split(msg, "\n") {
				fmt.Fprintf(&b, "  - %s\n", line)
			}
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// harnessOutput is the structured payload the generated harness prints on
// stdout; the harness defines structurally identical types (same yaml tags).
type harnessOutput struct {
	Catalogs []catalogOutcome `yaml:"catalogs"`
	// Warnings are mapping diagnostics, not catalog failures: they are reported on
	// stderr and never fail the run.
	Warnings []string `yaml:"warnings,omitempty"`
}

type catalogOutcome struct {
	PkgPath string         `yaml:"pkgPath"`
	VarName string         `yaml:"varName"`
	Name    string         `yaml:"name,omitempty"`
	Doc     *spec.AsyncAPI `yaml:"doc,omitempty"`
	Errors  []string       `yaml:"errors,omitempty"`
}

// removeHarnessDir removes the transient harness directory and reports a failure
// through log. A directory left behind is a diagnostic rather than a build
// failure — the document was materialized either way — but it is not silence: it
// is a warning record, which the CLI prints by default because it printed today.
//
// keep suppresses the removal. A harness that failed is the one case where the
// generated program must survive: it is the only artifact that explains why a
// catalog would not compile, and the error already names it.
//
// remove is a parameter so the failure path can be tested without contriving an
// unremovable directory.
func removeHarnessDir(tmp string, log *slog.Logger, keep bool, remove func(string) error) {
	if keep {
		return
	}
	if err := remove(tmp); err != nil {
		// Best effort: a writer that fails cannot be reported to itself, and the
		// build already succeeded.
		log.Warn("removing harness directory " + tmp + ": " + err.Error())
	}
}

// Materialize produces the *spec.AsyncAPI document for each catalog by running
// a generated harness that imports the catalog packages and prints the catalog
// values as YAML.
//
// YAML (rather than JSON) is used so integer values inside the opaque
// *Bindings maps survive the round-trip: encoding/json decodes every number
// into float64, whereas the YAML decoder preserves int/uint64, so a binding
// field like Kafka "partitions: 3" stays an integer.
//
// Running the harness executes only package init functions of the catalog
// packages (never the main package, which is not importable); the catalog
// values themselves are already-built DSL results.
// Materialize materializes every catalog in plan into its own document. It
// returns one Document per valid catalog, in plan order, carrying the identity
// the harness reported — which is the only place the catalog's declared document
// name is known, since a name is a value produced by the DSL rather than a
// declaration the static scan can read.
func Materialize(
	ctx context.Context,
	dir string,
	plan Plan,
	opts Options,
) ([]Document, error) {
	if len(plan.Catalogs) == 0 {
		return nil, nil
	}
	log := logging.OrQuiet(opts.Log)
	stage, done := logging.Step(log, "harness")
	stage.Info("materializing " + count(len(plan.Catalogs), "catalog") + "...")

	// The underscore prefix is load-bearing: the go tool ignores directories whose
	// name starts with "_" when matching patterns, so load()'s `./...` scan cannot
	// see this transient directory. Without it, a concurrent load() in the same
	// directory can enumerate the harness and then fail when it is removed
	// mid-scan, reporting "error(s) loading packages".
	tmp, err := os.MkdirTemp(dir, "_asyncgo-harness-")
	if err != nil {
		return nil, fmt.Errorf("creating harness directory: %w", err)
	}
	// keep is set on the failure path below, so the generated program survives for
	// the user to read. The deferred call reads it through the closure.
	keep := false
	defer func() { removeHarnessDir(tmp, log, keep, os.RemoveAll) }()

	harnessPath := filepath.Join(tmp, "main.go")
	stage.Debug("writing harness program to " + harnessPath)
	if err := os.WriteFile(harnessPath, []byte(harness(plan)), 0o644); err != nil {
		return nil, fmt.Errorf("writing harness: %w", err)
	}

	// The start line names what the user is waiting for: go run compiles the module,
	// which is the slowest and least obvious step of a generate/check run.
	stage.Debug("running go run ./" + filepath.Base(tmp) + " (this compiles the module)")
	cmd := exec.CommandContext(ctx, "go", "run", "./"+filepath.Base(tmp))
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if logging.Enabled(log, slog.LevelDebug) {
		// Tee the subprocess's stderr into the log stream at Debug: a compiler
		// warning or an init() writing to stderr used to vanish on a successful run.
		cmd.Stderr = io.MultiWriter(&stderr, logging.Writer(log))
	}
	if err := cmd.Run(); err != nil {
		keep = true
		return nil, fmt.Errorf(
			"materializing catalogs: %w\n%sharness program kept for inspection: %s",
			err, stderr.String(), harnessPath,
		)
	}

	var out harnessOutput
	if err := yaml.Unmarshal(stdout.Bytes(), &out); err != nil {
		return nil, fmt.Errorf("decoding harness output: %w", err)
	}
	reportMappingWarnings(log, out.Warnings)
	var docs []Document
	var catErrs CatalogErrors
	for _, c := range out.Catalogs {
		if len(c.Errors) > 0 {
			catErrs = append(catErrs, CatalogError{
				PkgPath: c.PkgPath,
				VarName: c.VarName,
				Errors:  c.Errors,
			})
			continue
		}
		docs = append(docs, Document{
			Catalog: Catalog{PkgPath: c.PkgPath, VarName: c.VarName, Name: c.Name},
			Doc:     c.Doc,
		})
	}
	// The stage closes even when a catalog is invalid: the harness did run, and the
	// counts are what make the failure report below legible.
	done(
		count(len(docs), "catalog")+" valid, "+count(len(catErrs), "error"),
		"catalogs", len(docs),
		"errors", len(catErrs),
	)
	if len(catErrs) > 0 {
		return nil, catErrs
	}
	return docs, nil
}

// reportMappingWarnings reports one warning record per mapping that was
// registered but never consulted, so a typo'd fully-qualified name in
// asyncgo.yaml — or an API mapping declared in a package the harness never
// imports — is visible in generate/check output instead of silently ignored.
//
// It is a warning, so it is printed by default: it was before verbosity existed,
// and a stale mapping is a fact the user needs, not a diagnostic they opted into.
// log is a parameter so the formatting is testable over a buffer.
func reportMappingWarnings(log *slog.Logger, warnings []string) {
	for _, fqn := range warnings {
		// Best effort: a writer that fails cannot be reported to itself, and the
		// document was materialized either way.
		log.Warn(
			"schema mapping for " + fqn + " was never used: " +
				"no derived type matched that name",
		)
	}
}

// harness generates a main package that imports the catalog and referenced
// packages, registers the config's schema mappings and the discovered alias
// edges, registers the combinator-referenced types, finalizes each catalog
// (hoisting those types into components.schemas), and encodes each catalog value
// as a YAML sequence on stdout.
func harness(plan Plan) string {
	alias := map[string]string{} // pkg path -> import alias
	var imports []struct{ alias, path string }
	addImport := func(path string) string {
		if a, ok := alias[path]; ok {
			return a
		}
		a := fmt.Sprintf("pkg%d", len(alias))
		alias[path] = a
		imports = append(imports, struct{ alias, path string }{a, path})
		return a
	}

	type catalogEntry struct{ alias, varName, pkgPath string }
	catEntries := make([]catalogEntry, 0, len(plan.Catalogs))
	for _, c := range plan.Catalogs {
		catEntries = append(catEntries, catalogEntry{addImport(c.PkgPath), c.VarName, c.PkgPath})
	}

	// Group Register call types by import path, preserving the sorted refs
	// order so the harness output is deterministic. Each type is registered as
	// new(T) rather than T{}: a composite literal is only valid for struct,
	// array, slice and map types, so a combinator directive naming a non-struct
	// type (e.g. a named string with a SchemaProvider) would not compile.
	// schema.Register dereferences pointers, so new(T) registers T itself.
	type regGroup struct {
		path, alias string
		types       []string
	}
	var regGroups []regGroup
	regIndex := map[string]int{}
	for _, r := range plan.Refs {
		idx, ok := regIndex[r.ImportPath]
		if !ok {
			idx = len(regGroups)
			regIndex[r.ImportPath] = idx
			regGroups = append(
				regGroups,
				regGroup{path: r.ImportPath, alias: addImport(r.ImportPath)},
			)
		}
		regGroups[idx].types = append(regGroups[idx].types, r.TypeName)
	}

	var b strings.Builder
	b.WriteString("package main\n\nimport (\n")
	b.WriteString("\t\"os\"\n\n")
	b.WriteString("\t\"github.com/goccy/go-yaml\"\n")
	b.WriteString("\t\"github.com/RubenRibGarcia/asyncgo/schema\"\n")
	b.WriteString("\t\"github.com/RubenRibGarcia/asyncgo/spec\"\n")
	for _, imp := range imports {
		fmt.Fprintf(&b, "\t%s %q\n", imp.alias, imp.path)
	}
	b.WriteString(")\n\n")
	b.WriteString("type harnessOutput struct {\n")
	b.WriteString("\tCatalogs []catalogOutcome `yaml:\"catalogs\"`\n")
	b.WriteString("\tWarnings []string       `yaml:\"warnings,omitempty\"`\n")
	b.WriteString("}\n\n")
	b.WriteString("type catalogOutcome struct {\n")
	b.WriteString("\tPkgPath string         `yaml:\"pkgPath\"`\n")
	b.WriteString("\tVarName string         `yaml:\"varName\"`\n")
	b.WriteString("\tName    string         `yaml:\"name,omitempty\"`\n")
	b.WriteString("\tDoc     *spec.AsyncAPI `yaml:\"doc,omitempty\"`\n")
	b.WriteString("\tErrors  []string       `yaml:\"errors,omitempty\"`\n")
	b.WriteString("}\n\n")
	b.WriteString("func main() {\n")

	// Mappings register before the catalog values are touched: registration is
	// the harness's stand-in for the project's init()-time API mappings, and
	// config emissions run here in main() so they override both a built-in and an
	// API mapping for the same name (last registration wins).
	for _, m := range plan.Mappings {
		fmt.Fprintf(
			&b,
			"\tif err := schema.MapSchemaJSON(%q, %q); err != nil {\n\t\tpanic(err)\n\t}\n",
			m.FQN,
			m.Schema,
		)
	}
	for _, a := range plan.Aliases {
		fmt.Fprintf(&b, "\tschema.MapAlias(%q, %q)\n", a.Alias, a.Base)
	}

	if len(regGroups) > 0 {
		b.WriteString("\tschema.Register(\n")
		for _, g := range regGroups {
			b.WriteString("\t\t")
			for _, tn := range g.types {
				fmt.Fprintf(&b, "new(%s.%s), ", g.alias, tn)
			}
			b.WriteString("\n")
		}
		b.WriteString("\t)\n")
	}

	b.WriteString("\tout := harnessOutput{}\n")
	for _, e := range catEntries {
		fmt.Fprintf(&b, "\t{\n")
		fmt.Fprintf(
			&b,
			"\t\tc := catalogOutcome{PkgPath: %q, VarName: %q, Name: %s.%s.Name}\n",
			e.pkgPath,
			e.varName,
			e.alias,
			e.varName,
		)
		fmt.Fprintf(
			&b,
			"\t\tif errs := %s.%s.ValidationErrors(); len(errs) > 0 {\n",
			e.alias,
			e.varName,
		)
		b.WriteString("\t\t\tfor _, e := range errs {\n")
		b.WriteString("\t\t\t\tc.Errors = append(c.Errors, e.Error())\n")
		b.WriteString("\t\t\t}\n")
		b.WriteString("\t\t} else {\n")
		fmt.Fprintf(&b, "\t\t\tc.Doc = %s.%s.Doc\n", e.alias, e.varName)
		b.WriteString("\t\t}\n")
		b.WriteString("\t\tout.Catalogs = append(out.Catalogs, c)\n")
		b.WriteString("\t}\n")
	}
	b.WriteString(
		"\tfor _, c := range out.Catalogs {\n\t\tif c.Doc != nil {\n\t\t\tschema.Finalize(c.Doc)\n\t\t}\n\t}\n",
	)
	b.WriteString("\tout.Warnings = schema.MappingWarnings()\n")
	b.WriteString(
		"\toutBytes, err := yaml.Marshal(out)\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\tif _, err := os.Stdout.Write(outBytes); err != nil {\n\t\tpanic(err)\n\t}\n}\n",
	)
	return b.String()
}
