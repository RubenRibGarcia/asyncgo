// Package discovery locates AsyncAPI catalogs (exported package-level
// variables of type *asyncgo.SpecResult) in a Go module and materializes them
// into documents. A document describes the module, not one of its binaries, so
// every package in the module is in scope — see scannable for the one
// exclusion.
package discovery

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path"
	"reflect"
	"sort"

	"github.com/RubenRibGarcia/asyncgo"
	"golang.org/x/tools/go/packages"
)

// Catalog is a discovered AsyncAPI catalog variable.
type Catalog struct {
	PkgPath string // import path of the package declaring the variable
	VarName string // exported variable name
	File    string // absolute path of the file declaring it, for diagnostics
	// Name is the document name the catalog declares with asyncgo.Spec(...).Named,
	// known only once the harness has evaluated the value. Empty means the catalog
	// declares no name and Stem derives one.
	Name string
}

// String renders the catalog as pkgPath.VarName, which is how every diagnostic
// names one.
func (c Catalog) String() string { return c.PkgPath + "." + c.VarName }

// Stem is the catalog's document name: the name declared with Named when there
// is one, otherwise "<declaring package basename>.<variable name>" — Catalog in
// package orders becomes "orders.Catalog".
//
// It is never empty and never contains a path separator, so it is safe as a
// filename stem. Deriving it from the catalog rather than from how many catalogs
// the module holds is deliberate: a count-derived name (asyncapi.yaml for one,
// <name>.asyncapi.yaml for several) would rename an existing artifact the moment
// a second catalog appeared, breaking both the committed file and check.
func (c Catalog) Stem() string {
	if c.Name != "" {
		return c.Name
	}
	base := path.Base(c.PkgPath)
	if base == "." || base == "/" {
		return c.VarName
	}
	return base + "." + c.VarName
}

// specResultType is the identity of *asyncgo.SpecResult, derived from the real
// type so it stays correct if the module path changes.
//
// This is the only place internal/discovery depends on the root package, and the
// dependency is deliberate: a catalog is recognised by its type identity, so the
// type has to be the one the DSL actually returns — a duplicate of SpecResult in
// spec would make an alias, and an alias is not an identity the discovery pass
// could trust to stay in step. The alternative (moving SpecResult into spec and
// aliasing it here) buys nothing at this size, so the edge is documented and kept.
var specResultType = reflect.TypeFor[asyncgo.SpecResult]()

// load loads all packages in the module rooted at dir with the syntax and type
// information needed for catalog discovery and comment extraction. It stops
// when ctx is cancelled.
func load(ctx context.Context, dir string) ([]*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedImports |
			packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo,
		Dir:     dir,
		Context: ctx,
		Tests:   false,
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

// scannable reports whether pkg's declarations can reach a document. Package
// main is the only exclusion: it is a program, not an importable package, so the
// materializing harness could not import it — a catalog or a
// combinator-referenced type declared there would fail the harness build instead
// of appearing in the document.
//
// Whether a main package imports pkg is deliberately not part of this test. A
// catalog is the module's declared topology, so it is in scope whether or not
// some binary happens to import it; narrowing on the import graph made adding a
// cmd/ directory silently delete catalogs from the document. Build reports the
// mismatch as a warning instead.
func scannable(pkg *packages.Package) bool { return pkg.Name != "main" }

// Find discovers exported package-level variables of type *asyncgo.SpecResult
// in the packages of the module rooted at dir.
func Find(ctx context.Context, dir string) ([]Catalog, error) {
	pkgs, err := load(ctx, dir)
	if err != nil {
		return nil, err
	}
	return scanCatalogs(pkgs), nil
}

// scanCatalogs returns catalog variables declared in the scannable packages,
// sorted by package path then variable name.
func scanCatalogs(pkgs []*packages.Package) []Catalog {
	var out []Catalog
	for _, pkg := range pkgs {
		if !scannable(pkg) {
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
					out = append(out, Catalog{
						PkgPath: pkg.PkgPath,
						VarName: name.Name,
						File:    pkg.Fset.Position(name.Pos()).Filename,
					})
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
	return obj.Pkg() != nil && obj.Pkg().Path() == specResultType.PkgPath() &&
		obj.Name() == specResultType.Name()
}

// importedByMain returns the import paths reachable from any main package by
// following imports within the loaded set. It is advisory only: it answers "which
// packages does a binary ship?" so Build can tell the user that a catalog is not
// wired into anything. It does not select what discovery scans, and a module with
// no main package yields an empty set — absence means "no binary here", not "no
// catalogs here", which is why the caller checks for mains separately.
func importedByMain(pkgs []*packages.Package) map[string]bool {
	byID := make(map[string]*packages.Package, len(pkgs))
	var mains []*packages.Package
	for _, p := range pkgs {
		byID[p.ID] = p
		if p.Name == "main" {
			mains = append(mains, p)
		}
	}

	imported := make(map[string]bool, len(pkgs))
	var visit func(p *packages.Package)
	visit = func(p *packages.Package) {
		if p == nil || imported[p.PkgPath] {
			return
		}
		imported[p.PkgPath] = true
		for _, imp := range p.Imports {
			if _, ok := byID[imp.ID]; ok {
				visit(imp)
			}
		}
	}
	for _, m := range mains {
		visit(m)
	}
	return imported
}
