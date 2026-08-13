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
func Materialize(dir string, cats []Catalog) ([]*spec.AsyncAPI, error) {
	if len(cats) == 0 {
		return nil, nil
	}

	tmp, err := os.MkdirTemp(dir, "asyncgo-harness-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)

	if err := os.WriteFile(filepath.Join(tmp, "main.go"), []byte(harness(cats)), 0o644); err != nil {
		return nil, err
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

// harness generates a main package that imports the catalog packages and
// encodes each catalog value as a YAML sequence on stdout.
func harness(cats []Catalog) string {
	alias := map[string]string{} // pkg path -> import alias
	var imports []struct{ alias, path string }
	for _, c := range cats {
		if _, ok := alias[c.PkgPath]; ok {
			continue
		}
		a := fmt.Sprintf("pkg%d", len(alias))
		alias[c.PkgPath] = a
		imports = append(imports, struct{ alias, path string }{a, c.PkgPath})
	}

	var b strings.Builder
	b.WriteString("package main\n\nimport (\n\t\"os\"\n\n\t\"github.com/goccy/go-yaml\"\n")
	for _, imp := range imports {
		fmt.Fprintf(&b, "\t%s %q\n", imp.alias, imp.path)
	}
	b.WriteString(")\n\nfunc main() {\n\tdocs := []any{\n")
	for _, c := range cats {
		fmt.Fprintf(&b, "\t\t%s.%s,\n", alias[c.PkgPath], c.VarName)
	}
	b.WriteString("\t}\n\tout, err := yaml.Marshal(docs)\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\tif _, err := os.Stdout.Write(out); err != nil {\n\t\tpanic(err)\n\t}\n}\n")
	return b.String()
}
