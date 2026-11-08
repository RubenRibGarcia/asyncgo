package discovery

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/RubenRibGarcia/asyncgo/internal/mappingconfig"
	"github.com/RubenRibGarcia/asyncgo/spec"
)

// configFileName is the mapping config file discovery auto-discovers in the
// directory a document is generated for — the module root, in practice, next to
// the artifact it shapes. Both generate and check fall back to it, so the two
// commands cannot drift apart, and --config overrides the path for both.
const configFileName = "asyncgo.yaml"

// Build loads the catalogs reachable from main in dir, materializes them into
// documents, applies comment-derived field descriptions, and merges the result.
// It returns the merged document and the number of catalogs found. The whole
// run — including the harness subprocess it spawns — is bounded by ctx.
//
// configPath names the schema-mapping config file, resolved against the process's
// working directory. An empty path auto-discovers <dir>/asyncgo.yaml.
func Build(ctx context.Context, dir, configPath string) (*spec.AsyncAPI, int, error) {
	// The config is read and validated first: a typo'd mapping name is a build error
	// naming the file, reported before the module load and before any harness runs —
	// not a panic inside a generated program, and not hidden behind a slower failure.
	mappings, err := loadMappingConfig(dir, configPath)
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

// loadMappingConfig returns the validated mappings of the mapping config file.
//
// path is the file named by --config, resolved against the process's working
// directory; an empty path means "auto-discover <dir>/asyncgo.yaml". The two
// cases differ on one point deliberately: an auto-discovered file may be absent
// (the config is optional, and a project with no third-party types needs none),
// while a path the caller named explicitly must exist — otherwise a typo in a CI
// script would silently generate a document with no mappings at all.
//
// An explicit path also replaces the default file rather than merging with it, so
// exactly one mapping set shapes a run. An unreadable or invalid file is an error
// either way: the file exists, so the project meant to configure something, and
// silently generating a document that ignores it would be worse than failing.
func loadMappingConfig(dir, path string) ([]mappingconfig.Mapping, error) {
	explicit := path != ""
	name := path
	if !explicit {
		name = filepath.Join(dir, configFileName)
	}

	data, err := os.ReadFile(name)
	if err != nil {
		// Only the auto-discovered file is allowed to be missing.
		if !explicit && os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading %s: %w", name, err)
	}
	cfg, err := mappingconfig.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return cfg.Mappings, nil
}
