package discovery

import (
	"fmt"

	"github.com/RubenRibGarcia/asyncgo/spec"
)

// Build loads the catalogs reachable from main in dir, materializes them into
// documents, applies comment-derived field descriptions, and merges the result.
// It returns the merged document and the number of catalogs found.
func Build(dir string) (*spec.AsyncAPI, int, error) {
	pkgs, err := load(dir)
	if err != nil {
		return nil, 0, err
	}
	reachable := reachableFromMain(pkgs)
	cats := scanCatalogs(pkgs, reachable)
	if len(cats) == 0 {
		return nil, 0, fmt.Errorf(
			"no AsyncAPI catalogs (*spec.AsyncAPI vars) reachable from main in %s",
			dir,
		)
	}
	docs, err := Materialize(dir, cats)
	if err != nil {
		return nil, 0, fmt.Errorf("materializing catalogs: %w", err)
	}
	doc := Merge(docs...)
	applyDescriptions(doc, extractDescriptions(pkgs, reachable))
	return doc, len(cats), nil
}
