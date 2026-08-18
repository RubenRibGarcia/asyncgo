package discovery

import (
	"go/ast"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

// combinatorRef is a fully-resolved reference to a named type that a combinator
// directive (oneOf=/anyOf=/allOf=) names. The harness registers it so Finalize
// can hoist its schema.
type combinatorRef struct {
	ImportPath string // import path of the declaring package
	TypeName   string // type name
}

// combinatorKeys are the tag directives that name combinator member types.
var combinatorKeys = []string{"oneOf", "anyOf", "allOf"}

// collectCombinatorRefs walks the reachable structs and collects the named
// types referenced by combinator directives, applying the same name-resolution
// rule as schema.refs: a name containing "/" is a fully-qualified name
// (pkgPath.TypeName); otherwise it is resolved against the declaring package.
// Results are de-duplicated and sorted by import path then type name.
func collectCombinatorRefs(pkgs []*packages.Package, reachable map[string]bool) []combinatorRef {
	seen := map[combinatorRef]bool{}
	walkStructTypes(
		pkgs,
		reachable,
		func(pkg *packages.Package, _ *ast.TypeSpec, st *ast.StructType) {
			for _, field := range st.Fields.List {
				if field.Tag == nil {
					continue
				}
				raw, err := strconv.Unquote(field.Tag.Value)
				if err != nil {
					continue
				}
				tag := reflect.StructTag(raw).Get("asyncapi")
				if tag == "" {
					continue
				}
				for _, key := range combinatorKeys {
					names, ok := combinatorNamesInTag(tag, key)
					if !ok {
						continue
					}
					for _, name := range names {
						seen[resolveCombinatorName(name, pkg.PkgPath)] = true
					}
				}
			}
		},
	)

	out := make([]combinatorRef, 0, len(seen))
	for ref := range seen {
		out = append(out, ref)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ImportPath != out[j].ImportPath {
			return out[i].ImportPath < out[j].ImportPath
		}
		return out[i].TypeName < out[j].TypeName
	})
	return out
}

// resolveCombinatorName resolves a combinator directive type name to its
// (importPath, typeName) pair. A name containing "/" is already fully
// qualified and is split at the final "."; otherwise the name is resolved
// against the declaring package.
func resolveCombinatorName(name, declaringPkg string) combinatorRef {
	if strings.Contains(name, "/") {
		if i := strings.LastIndex(name, "."); i >= 0 {
			return combinatorRef{ImportPath: name[:i], TypeName: name[i+1:]}
		}
		return combinatorRef{ImportPath: name}
	}
	return combinatorRef{ImportPath: declaringPkg, TypeName: name}
}

// combinatorNamesInTag mirrors schema.combinatorNames: it returns the
// "|"-separated type names for a combinator directive, or ok=false when the
// directive is absent.
func combinatorNamesInTag(tag, key string) ([]string, bool) {
	prefix := key + "="
	for _, part := range strings.Split(tag, ",") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, prefix) {
			raw := strings.TrimPrefix(part, prefix)
			if raw == "" {
				return nil, false
			}
			return strings.Split(raw, "|"), true
		}
	}
	return nil, false
}
