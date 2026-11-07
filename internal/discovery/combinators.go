package discovery

import (
	"fmt"
	"go/ast"
	"go/token"
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
//
// A reference the generated harness cannot emit as `new(alias.TypeName)` is an
// error naming the offending tag, rather than a Go syntax error surfacing as an
// opaque `exit status 1` from the harness build.
func collectCombinatorRefs(
	pkgs []*packages.Package,
	reachable map[string]bool,
) ([]combinatorRef, error) {
	seen := map[combinatorRef]bool{}
	var refErr error
	walkStructTypes(
		pkgs,
		reachable,
		func(pkg *packages.Package, _ *ast.TypeSpec, st *ast.StructType) {
			for _, field := range st.Fields.List {
				if field.Tag == nil || refErr != nil {
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
						ref, err := resolveCombinatorName(name, pkg.PkgPath)
						if err != nil {
							refErr = fmt.Errorf("asyncapi:%q: %w", key+"="+name, err)
							return
						}
						seen[ref] = true
					}
				}
			}
		},
	)
	if refErr != nil {
		return nil, refErr
	}

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
	return out, nil
}

// resolveCombinatorName resolves a combinator directive type name to its
// (importPath, typeName) pair. A name containing "/" is already fully
// qualified and is split at the final "."; otherwise the name is resolved
// against the declaring package.
//
// The generated harness emits `new(alias.TypeName)`, so a reference that cannot
// take that shape is an error here rather than a Go syntax error later: a
// qualified name needs both a path and a dot, and a package-local name has to be
// a bare Go identifier.
func resolveCombinatorName(name, declaringPkg string) (combinatorRef, error) {
	if strings.Contains(name, "/") {
		i := strings.LastIndex(name, ".")
		if i <= 0 || i == len(name)-1 {
			return combinatorRef{}, fmt.Errorf(
				"%q is not a fully-qualified type name; use pkgPath.TypeName",
				name,
			)
		}
		ref := combinatorRef{ImportPath: name[:i], TypeName: name[i+1:]}
		if !token.IsIdentifier(ref.TypeName) {
			return combinatorRef{}, fmt.Errorf("%q is not a Go type name", ref.TypeName)
		}
		return ref, nil
	}
	if !token.IsIdentifier(name) {
		return combinatorRef{}, fmt.Errorf(
			"%q is not a bare Go type name; a type from another package needs its full import path (pkgPath.TypeName)",
			name,
		)
	}
	return combinatorRef{ImportPath: declaringPkg, TypeName: name}, nil
}

// combinatorNamesInTag mirrors schema.combinatorNames: it returns the
// "|"-separated type names for a combinator directive, or ok=false when the
// directive is absent.
func combinatorNamesInTag(tag, key string) ([]string, bool) {
	prefix := key + "="
	for part := range strings.FieldsSeq(tag) {
		if after, ok := strings.CutPrefix(part, prefix); ok {
			raw := after
			if raw == "" {
				return nil, false
			}
			return strings.Split(raw, "|"), true
		}
	}
	return nil, false
}
