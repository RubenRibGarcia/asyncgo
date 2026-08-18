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

	var docs []*spec.AsyncAPI
	if err := yaml.Unmarshal(stdout.Bytes(), &docs); err != nil {
		return nil, fmt.Errorf("decoding harness output: %v", err)
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

	type catalogEntry struct{ alias, varName string }
	catEntries := make([]catalogEntry, 0, len(cats))
	for _, c := range cats {
		catEntries = append(catEntries, catalogEntry{addImport(c.PkgPath), c.VarName})
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
	b.WriteString(")\n\nfunc main() {\n")

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

	b.WriteString("\tdocs := []*spec.AsyncAPI{\n")
	for _, e := range catEntries {
		fmt.Fprintf(&b, "\t\t%s.%s,\n", e.alias, e.varName)
	}
	b.WriteString("\t}\n")
	b.WriteString("\tfor _, d := range docs {\n\t\tschema.Finalize(d)\n\t}\n")
	b.WriteString(
		"\tout, err := yaml.Marshal(docs)\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\tif _, err := os.Stdout.Write(out); err != nil {\n\t\tpanic(err)\n\t}\n}\n",
	)
	return b.String()
}
