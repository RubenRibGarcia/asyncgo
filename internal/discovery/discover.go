// Package discovery locates AsyncAPI catalogs (exported package-level
// variables of type *asyncgo.SpecResult) in a Go module, restricted to
// packages reachable from a main package, and materializes them into
// documents.
package discovery

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
	"sort"

	"github.com/RubenRibGarcia/asyncgo"
	"golang.org/x/tools/go/packages"
)

// Catalog is a discovered AsyncAPI catalog variable.
type Catalog struct {
	PkgPath string // import path of the package declaring the variable
	VarName string // exported variable name
}

// specResultType is the identity of *asyncgo.SpecResult, derived from the real
// type so it stays correct if the module path changes.
var specResultType = reflect.TypeFor[asyncgo.SpecResult]()

// load loads all packages in the module rooted at dir with the syntax and type
// information needed for catalog discovery and comment extraction.
func load(dir string) ([]*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedImports |
			packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo,
		Dir:   dir,
		Tests: false,
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("loading packages in %s: %w", dir, err)
	}
	if n := packages.PrintErrors(pkgs); n > 0 {
		return nil, fmt.Errorf("%d error(s) loading packages", n)
	}
	return pkgs, nil
}

// Find discovers exported package-level variables of type *asyncgo.SpecResult
// in the module rooted at dir, restricted to packages reachable from a main
// package. If the module has no main package, all module packages are
// considered.
func Find(dir string) ([]Catalog, error) {
	pkgs, err := load(dir)
	if err != nil {
		return nil, err
	}
	return scanCatalogs(pkgs, reachableFromMain(pkgs)), nil
}

// scanCatalogs returns catalog variables declared in the reachable packages,
// sorted by package path then variable name.
func scanCatalogs(pkgs []*packages.Package, reachable map[string]bool) []Catalog {
	var out []Catalog
	for _, pkg := range pkgs {
		if !reachable[pkg.ID] {
			continue
		}
		for _, file := range pkg.Syntax {
			out = append(out, scanFile(pkg, file)...)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].PkgPath != out[j].PkgPath {
			return out[i].PkgPath < out[j].PkgPath
		}
		return out[i].VarName < out[j].VarName
	})
	return out
}

// scanFile returns catalog variables declared at file scope in file.
func scanFile(pkg *packages.Package, file *ast.File) []Catalog {
	var out []Catalog
	for _, decl := range file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.VAR {
			continue
		}
		for _, s := range gd.Specs {
			vs, ok := s.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, name := range vs.Names {
				if !name.IsExported() {
					continue
				}
				obj := pkg.TypesInfo.Defs[name]
				if obj == nil {
					continue
				}
				if isSpecResult(obj.Type()) {
					out = append(out, Catalog{PkgPath: pkg.PkgPath, VarName: name.Name})
				}
			}
		}
	}
	return out
}

// isSpecResult reports whether t is *asyncgo.SpecResult.
func isSpecResult(t types.Type) bool {
	ptr, ok := t.(*types.Pointer)
	if !ok {
		return false
	}
	named, ok := ptr.Elem().(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj.Pkg() != nil && obj.Pkg().Path() == specResultType.PkgPath() && obj.Name() == specResultType.Name()
}

// reachableFromMain returns the set of package IDs reachable from any main
// package by following imports within the loaded set. If no main package
// exists, all loaded packages are considered reachable.
func reachableFromMain(pkgs []*packages.Package) map[string]bool {
	byID := make(map[string]*packages.Package, len(pkgs))
	var mains []*packages.Package
	for _, p := range pkgs {
		byID[p.ID] = p
		if p.Name == "main" {
			mains = append(mains, p)
		}
	}

	reach := make(map[string]bool)
	if len(mains) == 0 {
		for _, p := range pkgs {
			reach[p.ID] = true
		}
		return reach
	}

	var visit func(p *packages.Package)
	visit = func(p *packages.Package) {
		if p == nil || reach[p.ID] {
			return
		}
		reach[p.ID] = true
		for _, imp := range p.Imports {
			if _, ok := byID[imp.ID]; ok {
				visit(imp)
			}
		}
	}
	for _, m := range mains {
		visit(m)
	}
	return reach
}
