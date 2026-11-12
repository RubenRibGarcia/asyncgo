package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/RubenRibGarcia/asyncgo/internal/logging"
	"github.com/RubenRibGarcia/asyncgo/internal/mappingconfig"
	"github.com/RubenRibGarcia/asyncgo/spec"
	"golang.org/x/tools/go/packages"
)

// configFileName is the mapping config file discovery auto-discovers in the
// directory a document is generated for — the module root, in practice, next to
// the artifact it shapes. Both generate and check fall back to it, so the two
// commands cannot drift apart, and --config overrides the path for both.
const configFileName = "asyncgo.yaml"

// Options is the run-time configuration of a build, plus the diagnostics sink
// every stage reports through.
//
// The logger is an explicit field rather than a context value so "which
// functions can report?" is answerable at the call site, and so tests can hand
// in a logger over a buffer. A nil Log means silence, which is what a caller
// that only wants the document passes.
//
// Build reads Config; Materialize reads Log. Sharing one type keeps the two from
// growing parallel option structs as either gains a knob.
type Options struct {
	// Config names the schema-mapping config file, resolved against the process's
	// working directory. Empty auto-discovers <dir>/asyncgo.yaml.
	Config string
	// Log receives the run's diagnostics. Nil is quiet.
	Log *slog.Logger
}

// Build loads the module's catalogs in dir, materializes them into documents,
// applies comment-derived field descriptions, and merges the result into one
// document. It returns the merged document and the number of catalogs found. The
// whole run — including the harness subprocess it spawns — is bounded by ctx.
//
// It is the single-document mode. A caller that wants one document per catalog
// calls BuildAll and skips the merge, which is lossy by design: Merge is
// first-wins, so a second catalog's Info never reaches the result.
//
// opts.Config names the schema-mapping config file (see Options); the discovery
// it triggers is reported through opts.Log.
func Build(ctx context.Context, dir string, opts Options) (*spec.AsyncAPI, int, error) {
	log := logging.OrQuiet(opts.Log)

	docs, err := BuildAll(ctx, dir, opts)
	if err != nil {
		return nil, 0, err
	}

	mergeLog, mergeDone := logging.Step(log, "merge")
	doc, collisions := Merge(docs...)
	mergeDone(
		mergedSummary(doc, len(collisions)),
		"channels", len(doc.Channels),
		"servers", len(doc.Servers),
		"schemas", schemaCount(doc),
		"collisions", len(collisions),
	)
	// Collisions are a correctness fact — the merged document is lossy for one of
	// the catalogs — so they are reported as findings, not as debug detail.
	logging.ReportFindings(mergeLog, collisionFindings(collisions))
	return doc, len(docs), nil
}

// BuildAll loads the module's catalogs in dir and materializes each into its own
// document, applying comment-derived field descriptions. It returns one Document
// per catalog, in discovery order, and never merges: a caller that needs a
// single document merges the result itself (see Build).
//
// Everything Build documents about ctx and opts applies here too; the difference
// is only that the merge stage, and the collisions it resolves, are the caller's
// business.
func BuildAll(ctx context.Context, dir string, opts Options) ([]Document, error) {
	log := logging.OrQuiet(opts.Log)

	// The config is read and validated first: a typo'd mapping name is a build error
	// naming the file, reported before the module load and before any harness runs —
	// not a panic inside a generated program, and not hidden behind a slower failure.
	mappings, err := loadMappingConfig(dir, opts.Config, log)
	if err != nil {
		return nil, err
	}

	loadLog, loadDone := logging.Step(log, "load")
	loadLog.Info("loading packages in " + dir + "...")
	pkgs, err := load(ctx, dir)
	if err != nil {
		return nil, err
	}
	imported := importedByMain(pkgs)
	loadDone(
		fmt.Sprintf("%s, %d imported by a main", count(len(pkgs), "package"), len(imported)),
		"packages", len(pkgs),
		"importedByMain", len(imported),
	)

	catLog, catDone := logging.Step(log, "catalogs")
	cats := scanCatalogs(pkgs)
	if len(cats) == 0 {
		return nil, fmt.Errorf(
			"no AsyncAPI catalogs (*asyncgo.SpecResult vars) found in %s",
			dir,
		)
	}
	catDone(count(len(cats), "catalog")+" found", "catalogs", len(cats))
	logCatalogs(catLog, dir, cats)
	warnCatalogsOutsideMain(catLog, cats, imported, countMains(pkgs) > 0)

	if err := checkDirectives(pkgs); err != nil {
		return nil, err
	}

	refs, err := collectCombinatorRefs(pkgs)
	if err != nil {
		return nil, err
	}
	aliases := collectAliasEdges(pkgs)
	logRefs(logging.Tag(log, "refs"), refs, aliases)

	docs, err := Materialize(ctx, dir, Plan{
		Catalogs: cats,
		Refs:     refs,
		Mappings: mappings,
		Aliases:  aliases,
	}, opts)
	if err != nil {
		return nil, err
	}
	// The declaring file is a source location, not a value, so the harness cannot
	// report it. Materialize returns documents in plan order, which is cats order.
	for i := range docs {
		if i < len(cats) {
			docs[i].Catalog.File = cats[i].File
		}
	}

	descriptions := extractDescriptions(pkgs)
	for _, d := range docs {
		applyDescriptions(d.Doc, descriptions)
	}
	logDescriptions(logging.Tag(log, "descriptions"), descriptions)
	return docs, nil
}

// count renders "1 catalog" or "2 catalogs". Every closing sentence in the log
// counts something, and a log that says "1 catalogs" reads like a bug report
// about the log.
func count(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

// countMains counts the main packages in pkgs. Whether a module has a binary at
// all decides which warning is honest: a library with no main cannot have a
// catalog "imported by main", so warnCatalogsOutsideMain stays quiet.
func countMains(pkgs []*packages.Package) int {
	mains := 0
	for _, p := range pkgs {
		if p.Name == "main" {
			mains++
		}
	}
	return mains
}

// warnCatalogsOutsideMain reports a catalog that no main package imports.
//
// It is a warning rather than a gate: the catalog is in the document either way,
// because the document describes the module and a library module has no binary to
// import anything. The warning exists because the opposite reading — "my catalog
// silently vanished" — is exactly what the old reachable-from-main filter
// produced, and it is worth one line to rule out. hasMain suppresses it for a
// module with no main package at all, where every catalog would trip it.
func warnCatalogsOutsideMain(
	log *slog.Logger,
	cats []Catalog,
	imported map[string]bool,
	hasMain bool,
) {
	if !hasMain {
		return
	}
	for _, c := range cats {
		if imported[c.PkgPath] {
			continue
		}
		log.Warn(
			c.String() + " is not imported by any main package; it is still generated",
		)
	}
}

// logCatalogs reports each catalog's declaring file at Debug, so a wrong or
// unexpected catalog can be traced to the source that declares it.
func logCatalogs(log *slog.Logger, dir string, cats []Catalog) {
	if !log.Enabled(context.Background(), slog.LevelDebug) {
		return
	}
	for _, c := range cats {
		file := c.File
		if rel, err := filepath.Rel(dir, c.File); err == nil {
			file = rel
		}
		log.Debug(fmt.Sprintf("%s declared in %s", c, file))
	}
}

// logRefs reports the combinator registration work at Debug: the refs the
// harness will register and the alias edges it will emit.
func logRefs(log *slog.Logger, refs []combinatorRef, aliases []aliasEdge) {
	if !log.Enabled(context.Background(), slog.LevelDebug) {
		return
	}
	pkgs := map[string]bool{}
	for _, r := range refs {
		pkgs[r.ImportPath] = true
	}
	log.Debug(fmt.Sprintf(
		"%d combinator refs from %d packages, %d alias edges",
		len(refs), len(pkgs), len(aliases),
	))
}

// logDescriptions reports how many doc-comment descriptions were applied, at
// Debug: it is the only evidence that a field's comment reached the document.
func logDescriptions(log *slog.Logger, descriptions descriptions) {
	if !log.Enabled(context.Background(), slog.LevelDebug) {
		return
	}
	fields := 0
	for _, schemaFields := range descriptions {
		fields += len(schemaFields)
	}
	log.Debug(fmt.Sprintf(
		"applied %d field descriptions across %d schemas",
		fields, len(descriptions),
	))
}

// mergedSummary is the merge stage's closing sentence: what the document ended up
// containing, so a missing section is visible without reading the artifact.
func mergedSummary(doc *spec.AsyncAPI, collisions int) string {
	return count(len(doc.Channels), "channel") + ", " +
		count(len(doc.Servers), "server") + ", " +
		count(schemaCount(doc), "schema") + ", " +
		count(collisions, "collision")
}

// schemaCount is the number of hoisted component schemas, tolerating a document
// with no components section at all.
func schemaCount(doc *spec.AsyncAPI) int {
	if doc.Components == nil {
		return 0
	}
	return len(doc.Components.Schemas)
}

// collisionFindings renders merge collisions as findings. A collision has no
// file to point at — it is a conflict inside the merged document — so Path is
// left empty and the location is the document path itself.
func collisionFindings(collisions []Collision) []logging.Finding {
	findings := make([]logging.Finding, 0, len(collisions))
	for _, c := range collisions {
		dropped := make([]string, 0, len(c.Dropped))
		for _, d := range c.Dropped {
			dropped = append(dropped, d.String())
		}
		findings = append(findings, logging.Finding{
			Where: c.Section + "." + c.Name,
			What:  "duplicate, kept from " + c.Kept.String(),
			Why:   "dropped from " + strings.Join(dropped, ", "),
		})
	}
	return findings
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
func loadMappingConfig(dir, path string, log *slog.Logger) ([]mappingconfig.Mapping, error) {
	explicit := path != ""
	name := path
	if !explicit {
		name = filepath.Join(dir, configFileName)
	}

	stage, done := logging.Step(log, "config")
	data, err := os.ReadFile(name)
	if err != nil {
		// Only the auto-discovered file is allowed to be missing.
		if !explicit && os.IsNotExist(err) {
			stage.Info("no " + configFileName + " found, using built-in mappings only")
			done("")
			return nil, nil
		}
		return nil, fmt.Errorf("reading %s: %w", name, err)
	}
	source := "auto-discovered"
	if explicit {
		source = "explicit"
	}
	stage.Info(fmt.Sprintf("using %s (%s)", name, source))
	cfg, err := mappingconfig.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	done(count(len(cfg.Mappings), "mapping"), "mappings", len(cfg.Mappings))
	return cfg.Mappings, nil
}
