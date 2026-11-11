package discovery

import (
	"errors"
	"fmt"
	"go/ast"
	"reflect"
	"strconv"

	"github.com/RubenRibGarcia/asyncgo/schema"
	"golang.org/x/tools/go/packages"
)

// checkDirectives rejects an asyncapi struct tag carrying a directive the
// generator does not honour.
//
// The reflection pass ignores a key it does not know and cannot do better: a
// reflect.Type carries no position, so a typo there surfaces as a schema quietly
// missing the constraint its author meant to add. The AST pass has the position
// and runs before the harness is built, so the same typo is a located catalog
// build error — exit 1 for generate and check — instead of a document nobody
// notices is wrong.
//
// Every violation is reported, not the first: a catalog that still spells the
// removed example= directive in a dozen places should take one run to fix, not
// twelve. schema.CheckTag owns the vocabulary; only the position comes from here.
func checkDirectives(pkgs []*packages.Package) error {
	var errs []error
	walkStructTypes(
		pkgs,
		func(pkg *packages.Package, _ *ast.TypeSpec, st *ast.StructType) {
			for _, field := range st.Fields.List {
				if field.Tag == nil {
					continue
				}
				raw, err := strconv.Unquote(field.Tag.Value)
				if err != nil {
					continue // not a quoted tag literal; nothing to check
				}
				tag := reflect.StructTag(raw).Get("asyncapi")
				if tag == "" {
					continue // json-only field, or no tag at all
				}
				pos := pkg.Fset.Position(field.Tag.Pos())
				for _, directiveErr := range schema.CheckTag(tag) {
					errs = append(errs, fmt.Errorf(
						"%s: asyncapi:%q: %w", pos, tag, directiveErr))
				}
			}
		},
	)
	return errors.Join(errs...)
}
