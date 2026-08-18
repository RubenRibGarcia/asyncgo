package discovery

import (
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
	"strconv"
	"strings"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"golang.org/x/tools/go/packages"
)

// descriptions maps a hoisted schema name (pkgPath.TypeName) to a map of JSON
// field name -> doc-comment text.
type descriptions map[string]map[string]string

// extractDescriptions walks the reachable packages' syntax and collects the doc
// comment of every exported struct field, keyed by the same schema name and
// JSON field name the reflection-based derivation emits. It mirrors
// schema.fillObject's field rules: unexported fields and json:"-" are skipped,
// json tags drive the field name, and anonymous embedded structs are flattened
// into their parent.
func extractDescriptions(pkgs []*packages.Package, reachable map[string]bool) descriptions {
	named := map[*types.TypeName]*ast.StructType{}
	walkStructTypes(
		pkgs,
		reachable,
		func(pkg *packages.Package, ts *ast.TypeSpec, st *ast.StructType) {
			if obj := pkg.TypesInfo.Defs[ts.Name]; obj != nil {
				if tn, ok := obj.(*types.TypeName); ok {
					named[tn] = st
				}
			}
		},
	)

	out := descriptions{}
	walkStructTypes(
		pkgs,
		reachable,
		func(pkg *packages.Package, ts *ast.TypeSpec, st *ast.StructType) {
			obj := pkg.TypesInfo.Defs[ts.Name]
			tn, ok := obj.(*types.TypeName)
			if !ok || tn.Pkg() == nil {
				return
			}
			out[tn.Pkg().Path()+"."+tn.Name()] = collectFieldDescriptions(pkg, st, named)
		},
	)
	return out
}

// walkStructTypes calls fn for every named struct type declared in a reachable
// package.
func walkStructTypes(
	pkgs []*packages.Package,
	reachable map[string]bool,
	fn func(pkg *packages.Package, ts *ast.TypeSpec, st *ast.StructType),
) {
	for _, pkg := range pkgs {
		if !reachable[pkg.ID] {
			continue
		}
		for _, file := range pkg.Syntax {
			for _, decl := range file.Decls {
				gd, ok := decl.(*ast.GenDecl)
				if !ok || gd.Tok != token.TYPE {
					continue
				}
				for _, spec := range gd.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					st, ok := ts.Type.(*ast.StructType)
					if !ok {
						continue
					}
					fn(pkg, ts, st)
				}
			}
		}
	}
}

func collectFieldDescriptions(
	pkg *packages.Package,
	st *ast.StructType,
	named map[*types.TypeName]*ast.StructType,
) map[string]string {
	out := map[string]string{}
	for _, field := range st.Fields.List {
		if len(field.Names) == 0 { // anonymous embedded field
			for k, v := range embeddedFieldDescriptions(pkg, field.Type, named) {
				out[k] = v
			}
			continue
		}
		for _, name := range field.Names {
			if !name.IsExported() {
				continue
			}
			jsonName := jsonFieldName(field, name.Name)
			if jsonName == "" { // json:"-"
				continue
			}
			if d := fieldDoc(field); d != "" {
				out[jsonName] = d
			}
		}
	}
	return out
}

// embeddedFieldDescriptions returns the flattened field descriptions of an
// embedded struct field. Anonymous inline structs are walked directly; named
// struct types are resolved through the type checker back to their AST so their
// fields' comments can be collected. Embedded types from packages outside the
// loaded set have no AST and are skipped.
func embeddedFieldDescriptions(
	pkg *packages.Package,
	expr ast.Expr,
	named map[*types.TypeName]*ast.StructType,
) map[string]string {
	switch e := expr.(type) {
	case *ast.StructType:
		return collectFieldDescriptions(pkg, e, named)
	case *ast.StarExpr:
		if inner, ok := e.X.(*ast.StructType); ok {
			return collectFieldDescriptions(pkg, inner, named)
		}
	}
	if st := embeddedStructAST(pkg, expr, named); st != nil {
		return collectFieldDescriptions(pkg, st, named)
	}
	return nil
}

// embeddedStructAST resolves an embedded field's type expression to the AST of
// the named struct type it denotes, or nil if it is not an embedded struct
// whose source is in the loaded packages.
func embeddedStructAST(
	pkg *packages.Package,
	expr ast.Expr,
	named map[*types.TypeName]*ast.StructType,
) *ast.StructType {
	if pkg == nil {
		return nil
	}
	for {
		star, ok := expr.(*ast.StarExpr)
		if !ok {
			break
		}
		expr = star.X
	}
	var ident *ast.Ident
	switch e := expr.(type) {
	case *ast.Ident:
		ident = e
	case *ast.SelectorExpr:
		ident = e.Sel
	default:
		return nil
	}
	obj := pkg.TypesInfo.Uses[ident]
	tn, ok := obj.(*types.TypeName)
	if !ok {
		return nil
	}
	return named[tn]
}

// jsonFieldName mirrors schema.jsonName: the json struct tag drives the field
// name, "-" skips the field, and the Go field name is the fallback.
func jsonFieldName(field *ast.Field, fallback string) string {
	if field.Tag == nil {
		return fallback
	}
	tag, err := strconv.Unquote(field.Tag.Value)
	if err != nil {
		return fallback
	}
	t := reflect.StructTag(tag).Get("json")
	if t == "" {
		return fallback
	}
	if t == "-" {
		return ""
	}
	if before, _, ok := strings.Cut(t, ","); ok {
		return before
	}
	return t
}

func fieldDoc(field *ast.Field) string {
	if field.Doc != nil {
		return strings.TrimSpace(field.Doc.Text())
	}
	if field.Comment != nil {
		return strings.TrimSpace(field.Comment.Text())
	}
	return ""
}

// applyDescriptions fills in the Description of schema properties from the
// extracted field comments.
func applyDescriptions(doc *spec.AsyncAPI, desc descriptions) {
	if doc == nil || doc.Components == nil {
		return
	}
	for name, fields := range desc {
		schema, ok := doc.Components.Schemas[name]
		if !ok || schema == nil {
			continue
		}
		applySchemaDescriptions(schema, fields)
	}
}

// applySchemaDescriptions applies the field descriptions to a schema and, for
// allOf-composed structs, to the member that holds the struct's own fields.
func applySchemaDescriptions(schema *spec.Schema, fields map[string]string) {
	if schema == nil {
		return
	}
	if schema.Properties != nil {
		for jsonName, d := range fields {
			if prop, ok := schema.Properties[jsonName]; ok && prop != nil {
				prop.Description = d
			}
		}
	}
	for _, member := range schema.AllOf {
		applySchemaDescriptions(member, fields)
	}
}
