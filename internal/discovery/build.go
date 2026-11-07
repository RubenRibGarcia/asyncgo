package discovery

import (
	"context"
	"fmt"

	"github.com/RubenRibGarcia/asyncgo/spec"
)

// Build loads the catalogs reachable from main in dir, materializes them into
// documents, applies comment-derived field descriptions, and merges the result.
// It returns the merged document and the number of catalogs found. The whole
// run — including the harness subprocess it spawns — is bounded by ctx.
func Build(ctx context.Context, dir string) (*spec.AsyncAPI, int, error) {
	pkgs, err := load(ctx, dir)
	if err != nil {
		return nil, 0, err
	}
	reachable := reachableFromMain(pkgs)
	cats := scanCatalogs(pkgs, reachable)
	if len(cats) == 0 {
		return nil, 0, fmt.Errorf(
			"no AsyncAPI catalogs (*asyncgo.SpecResult vars) reachable from main in %s",
			dir,
		)
	}
	docs, err := Materialize(ctx, dir, cats, collectCombinatorRefs(pkgs, reachable))
	if err != nil {
		return nil, 0, err
	}
	doc := Merge(docs...)
	applyDescriptions(doc, extractDescriptions(pkgs, reachable))
	return doc, len(cats), nil
}
