package discovery

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/goccy/go-yaml"
)

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
			fmt.Fprintf(&b, "  - %s\n", msg)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// harnessOutput is the structured payload the generated harness prints on
// stdout; the harness defines structurally identical types (same yaml tags).
type harnessOutput struct {
	Catalogs []catalogOutcome `yaml:"catalogs"`
}

type catalogOutcome struct {
	PkgPath string         `yaml:"pkgPath"`
	VarName string         `yaml:"varName"`
	Doc     *spec.AsyncAPI `yaml:"doc,omitempty"`
	Errors  []string       `yaml:"errors,omitempty"`
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
func Materialize(dir string, cats []Catalog, refs []combinatorRef) ([]*spec.AsyncAPI, error) {
	if len(cats) == 0 {
		return nil, nil
	}

	tmp, err := os.MkdirTemp(dir, "asyncgo-harness-")
	if err != nil {
		return nil, fmt.Errorf("creating harness directory: %w", err)
	}
	defer func() {
		err := os.RemoveAll(tmp)
		if err != nil {
			fmt.Printf("failed to remove harness directory: %v\n", err)
		}
	}()

	if err := os.WriteFile(
		filepath.Join(tmp, "main.go"),
		[]byte(harness(cats, refs)),
		0o644,
	); err != nil {
		return nil, fmt.Errorf("writing harness: %w", err)
	}

	cmd := exec.Command("go", "run", "./"+filepath.Base(tmp))
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("materializing catalogs: %v\n%s", err, stderr.String())
	}

	var out harnessOutput
	if err := yaml.Unmarshal(stdout.Bytes(), &out); err != nil {
		return nil, fmt.Errorf("decoding harness output: %v", err)
	}
	var docs []*spec.AsyncAPI
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
		docs = append(docs, c.Doc)
	}
	if len(catErrs) > 0 {
		return nil, catErrs
	}
	return docs, nil
}

// harness generates a main package that imports the catalog and referenced
// packages, registers the combinator-referenced types, finalizes each catalog
// (hoisting those types into components.schemas), and encodes each catalog
// value as a YAML sequence on stdout.
func harness(cats []Catalog, refs []combinatorRef) string {
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
	catEntries := make([]catalogEntry, 0, len(cats))
	for _, c := range cats {
		catEntries = append(catEntries, catalogEntry{addImport(c.PkgPath), c.VarName, c.PkgPath})
	}

	// Group Register call types by import path, preserving the sorted refs
	// order so the harness output is deterministic.
	type regGroup struct {
		path, alias string
		types       []string
	}
	var regGroups []regGroup
	regIndex := map[string]int{}
	for _, r := range refs {
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
	b.WriteString("}\n\n")
	b.WriteString("type catalogOutcome struct {\n")
	b.WriteString("\tPkgPath string         `yaml:\"pkgPath\"`\n")
	b.WriteString("\tVarName string         `yaml:\"varName\"`\n")
	b.WriteString("\tDoc     *spec.AsyncAPI `yaml:\"doc,omitempty\"`\n")
	b.WriteString("\tErrors  []string       `yaml:\"errors,omitempty\"`\n")
	b.WriteString("}\n\n")
	b.WriteString("func main() {\n")

	if len(regGroups) > 0 {
		b.WriteString("\tschema.Register(\n")
		for _, g := range regGroups {
			b.WriteString("\t\t")
			for _, tn := range g.types {
				fmt.Fprintf(&b, "%s.%s{}, ", g.alias, tn)
			}
			b.WriteString("\n")
		}
		b.WriteString("\t)\n")
	}

	b.WriteString("\tout := harnessOutput{}\n")
	for _, e := range catEntries {
		fmt.Fprintf(&b, "\t{\n")
		fmt.Fprintf(&b, "\t\tc := catalogOutcome{PkgPath: %q, VarName: %q}\n", e.pkgPath, e.varName)
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
	b.WriteString(
		"\toutBytes, err := yaml.Marshal(out)\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\tif _, err := os.Stdout.Write(outBytes); err != nil {\n\t\tpanic(err)\n\t}\n}\n",
	)
	return b.String()
}
