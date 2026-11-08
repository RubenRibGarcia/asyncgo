package discovery

import (
	"go/ast"
	"go/token"
	"go/types"
	"sort"

	"golang.org/x/tools/go/packages"
)

// aliasEdge is one `type X Y` declaration found in a reachable package: Alias is
// declared as Base. Both are fully-qualified names.
type aliasEdge struct {
	Alias string // pkgPath.TypeName of the declared type
	Base  string // pkgPath.TypeName of the type it was declared from
}

// collectAliasEdges walks the named types declared in the reachable packages and
// returns one edge per derived type, deduplicated and sorted by alias.
//
// Derivation needs these edges because reflection cannot walk the declared-from
// chain: reflect.Type for `type OrderID uuid.UUID` exposes only OrderID and its
// kind (an array), with no link to uuid.UUID. The discovery pass already loads
// syntax and type information, so the link is resolvable statically and the
// harness records it as schema.MapAlias, letting derivation find the base type's
// mapping.
//
// One edge per declaration, rather than resolving each alias to its ultimate
// base here: that keeps the harness generator dumb and lets a mapping registered
// on an intermediate alias apply to the types derived from it.
//
// Declarations inside third-party packages are not scanned — only module packages
// are loaded — so a chain that crosses into a dependency stops there. That is the
// documented limit of alias inheritance.
func collectAliasEdges(pkgs []*packages.Package, reachable map[string]bool) []aliasEdge {
	seen := map[aliasEdge]bool{}
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
					if edge, ok := aliasEdgeFor(pkg, ts); ok {
						seen[edge] = true
					}
				}
			}
		}
	}

	out := make([]aliasEdge, 0, len(seen))
	for edge := range seen {
		out = append(out, edge)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Alias != out[j].Alias {
			return out[i].Alias < out[j].Alias
		}
		return out[i].Base < out[j].Base
	})
	return out
}

// aliasEdgeFor resolves a single type declaration.
//
// It returns ok=false for everything that needs no edge: a Go type alias
// (`type A = B`) introduces no new type, so reflection already sees B; a base that
// is predeclared (`type ID string`) or not itself a named type (`type Tags
// []string`) has no mapping to inherit; and a generic instantiation (`type IDs
// Box[string]`) cannot be named by the alias's FQN anyway, since name-keyed lookup
// does not carry type arguments.
func aliasEdgeFor(pkg *packages.Package, ts *ast.TypeSpec) (aliasEdge, bool) {
	obj, ok := pkg.TypesInfo.Defs[ts.Name].(*types.TypeName)
	if !ok || obj.Pkg() == nil || obj.IsAlias() {
		return aliasEdge{}, false
	}

	t := pkg.TypesInfo.TypeOf(ts.Type)
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	named, ok := t.(*types.Named)
	if !ok {
		return aliasEdge{}, false
	}
	if args := named.TypeArgs(); args != nil && args.Len() > 0 {
		return aliasEdge{}, false
	}
	base := named.Obj()
	if base == nil || base.Pkg() == nil || base.IsAlias() {
		return aliasEdge{}, false
	}

	return aliasEdge{
		Alias: obj.Pkg().Path() + "." + obj.Name(),
		Base:  base.Pkg().Path() + "." + base.Name(),
	}, true
}
