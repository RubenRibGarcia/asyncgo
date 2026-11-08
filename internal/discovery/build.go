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
	// The config is validated first, before the module is even loaded, so a typo in
	// asyncgo.yaml is a build error naming the file rather than a panic inside a
	// generated program — and it does not hide behind a slower failure.
	mappings, err := loadMappingConfig(dir)
	if err != nil {
		return nil, 0, err
	}

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
	refs, err := collectCombinatorRefs(pkgs, reachable)
	if err != nil {
		return nil, 0, err
	}
	docs, err := Materialize(ctx, dir, Plan{
		Catalogs: cats,
		Refs:     refs,
		Mappings: mappings,
		Aliases:  collectAliasEdges(pkgs, reachable),
	})
	if err != nil {
		return nil, 0, err
	}
	doc := Merge(docs...)
	applyDescriptions(doc, extractDescriptions(pkgs, reachable))
	return doc, len(cats), nil
}
