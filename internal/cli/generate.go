package cli

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/RubenRibGarcia/asyncgo/internal/discovery"
	"github.com/RubenRibGarcia/asyncgo/internal/logging"
	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/spf13/cobra"
)

// outputFormat ties an accepted --format value to the encoder it selects and to
// the filename used when the user did not name an output path.
type outputFormat struct {
	encode      func(*spec.AsyncAPI) ([]byte, error)
	defaultName string
}

// resolveFormat maps a --format value to its encoder. An unknown value is
// rejected here, before any directory is resolved or the discovery harness
// runs, so a typo writes nothing.
func resolveFormat(format string) (outputFormat, error) {
	switch format {
	case "yaml":
		return outputFormat{encode: (*spec.AsyncAPI).YAML, defaultName: "asyncapi.yaml"}, nil
	case "json":
		return outputFormat{encode: (*spec.AsyncAPI).JSONIndent, defaultName: "asyncapi.json"}, nil
	default:
		return outputFormat{}, fmt.Errorf(
			"unknown format %q: want %q or %q",
			format,
			"yaml",
			"json",
		)
	}
}

// writeFileAtomic writes data to path by writing a temporary file in the same
// directory and renaming it into place. The artifact is rewritten on every run,
// so a failure partway through os.WriteFile would truncate the previous document
// and leave a half-written one behind — and a half-written AsyncAPI document is
// still valid YAML for a prefix of itself in many cases, which makes it worse
// than an obvious corruption. The rename is atomic within a directory, so a
// reader sees either the old artifact or the new one.
//
// The temp name must not look like the discovery harness directories
// (_asyncgo-harness-*): those are skipped by load()'s pattern scan, and an
// artifact that the go tool ignores would be a confusing bug. It also has to sit
// in path's own directory, since os.Rename cannot cross filesystems.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	// Reserve the name in the target's own directory: os.Rename cannot cross
	// filesystems, and the random suffix keeps two concurrent generators from
	// writing through the same temp file.
	tmp, err := os.CreateTemp(filepath.Dir(path), ".asyncgo-write-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// The reservation only held the name; WriteFile fills it. Best effort after a
	// successful rename, and a failure here must not mask the real error.
	_ = tmp.Close()
	defer func() { _ = os.Remove(tmpName) }()

	if err := os.WriteFile(tmpName, data, perm); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// formatSize renders a byte count the way a human reads it (12.3 KiB). It is a
// log detail, so it rounds to one decimal and never says "bytes" for a section
// the user is about to dump into a diff.
func formatSize(n int) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	value := float64(n) / unit
	for _, suffix := range []string{"KiB", "MiB", "GiB"} {
		if value < unit {
			return fmt.Sprintf("%.1f %s", value, suffix)
		}
		value /= unit
	}
	return fmt.Sprintf("%.1f TiB", value)
}

func newGenerateCmd() *cobra.Command {
	var output, format, config string
	cmd := &cobra.Command{
		Use:   "generate [dir]",
		Short: "Write asyncapi.yaml or asyncapi.json for the module rooted at dir",
		Long: "Write the AsyncAPI document for the module rooted at dir.\n\n" +
			"A file destination writes one merged document for the whole module, which is " +
			"the default. A directory destination — an -o value with a trailing separator — " +
			"writes one document per catalog into it, so a module with several catalogs " +
			"commits one correct document each instead of a lossy merge.\n\n" +
			"In directory mode the filename is <name>.asyncapi.<format>, where <name> is the " +
			"one the catalog declares with asyncgo.Spec(...).Named, or <package>.<variable> " +
			"when it declares none.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			selected, err := resolveFormat(format)
			if err != nil {
				return err
			}

			dir, err := resolveDir(args)
			if err != nil {
				return fmt.Errorf("generating: %w", err)
			}

			path, mode, err := resolveOutput(dir, output, selected.defaultName)
			if err != nil {
				return fmt.Errorf("generating: %w", err)
			}

			opts := discovery.Options{
				Config: config,
				Log:    logging.FromContext(cmd.Context()),
			}
			if mode == modeDir {
				return generatePerCatalog(cmd, dir, path, format, selected, opts)
			}
			return generateMerged(cmd, dir, path, format, selected, opts)
		},
	}
	cmd.Flags().
		StringVar(&format, "format", "yaml", "output format: yaml or json")
	cmd.Flags().
		StringVarP(&output, "output", "o", "", "output file, or a directory (trailing separator) for one document per catalog (default: <dir>/asyncapi.<format>)")
	cmd.Flags().
		StringVar(&config, "config", "", "schema-mapping config file (default: <dir>/asyncgo.yaml)")
	return cmd
}

// generateMerged writes the module's single merged document to path. It is the
// default mode and is unchanged: the document covers every catalog, and Merge's
// first-wins drops are reported as collisions.
func generateMerged(
	cmd *cobra.Command,
	dir, path, format string,
	selected outputFormat,
	opts discovery.Options,
) error {
	log := logging.FromContext(cmd.Context())

	doc, n, err := buildDoc(cmd, dir, opts, "generating")
	if err != nil {
		return err
	}
	_, encodeDone := logging.Step(log, "encode")
	out, err := selected.encode(doc)
	if err != nil {
		return fmt.Errorf("encoding document: %w", err)
	}
	encodeDone(
		format+", "+formatSize(len(out)),
		"format", format,
		"bytes", len(out),
	)
	_, writeDone := logging.Step(log, "write")
	if err := writeFileAtomic(path, out, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	writeDone(path, "path", path)
	// The recap is printed before the artifact line so the two streams read in
	// the order the run happened: diagnostics, then the result.
	logging.Recap(log)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "wrote %s (%d catalog(s))\n", path, n)
	return nil
}

// generatePerCatalog writes one document per catalog into outDir.
//
// Every document is validated, name-checked and encoded before the first file is
// written. writeFileAtomic keeps one file from being half-written, but a set is
// the unit here: failing on the third of five documents would otherwise leave a
// directory that is neither the old artifact set nor the new one.
func generatePerCatalog(
	cmd *cobra.Command,
	dir, outDir, format string,
	selected outputFormat,
	opts discovery.Options,
) error {
	log := logging.FromContext(cmd.Context())

	docs, err := buildDocs(cmd, dir, opts, "generating")
	if err != nil {
		return err
	}
	artifacts, err := encodeArtifacts(docs, outDir, format, selected, log)
	if err != nil {
		return err
	}

	_, writeDone := logging.Step(log, "write")
	for _, a := range artifacts {
		if err := writeFileAtomic(a.path, a.data, 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", a.path, err)
		}
	}
	writeDone(
		fmt.Sprintf("%d document(s) into %s", len(artifacts), outDir),
		"path", outDir,
		"documents", len(artifacts),
	)
	logging.Recap(log)
	// One line per file, so the shape of a directory run is visible at default
	// verbosity: a trailing separator is easy to mistype, and this is what shows
	// which model the run actually took.
	for _, a := range artifacts {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", a.path)
	}
	return nil
}

// artifact is one encoded document and the path it belongs at.
type artifact struct {
	path string
	data []byte
}

// encodeArtifacts encodes one document per catalog and resolves each filename,
// rejecting the one conflict directory mode can create: two catalogs resolving
// to the same name. Encoding happens here, before any file is written, so a
// failure leaves no partial artifact set behind.
func encodeArtifacts(
	docs []discovery.Document,
	outDir, format string,
	selected outputFormat,
	log *slog.Logger,
) ([]artifact, error) {
	_, encodeDone := logging.Step(log, "encode")
	artifacts := make([]artifact, 0, len(docs))
	owners := make(map[string]string, len(docs))
	total := 0
	for _, d := range docs {
		stem := d.Catalog.Stem()
		name := artifactName(stem, format)
		if kept, taken := owners[name]; taken {
			return nil, fmt.Errorf(
				"duplicate document name %q: %s and %s would both be written to %s; "+
					"call .Named() on one of them",
				stem, kept, d.Catalog, name,
			)
		}
		owners[name] = d.Catalog.String()
		data, err := selected.encode(d.Doc)
		if err != nil {
			return nil, fmt.Errorf("encoding document for %s: %w", d.Catalog, err)
		}
		total += len(data)
		artifacts = append(artifacts, artifact{
			path: filepath.Join(outDir, name),
			data: data,
		})
	}
	encodeDone(
		fmt.Sprintf("%d document(s), %s", len(artifacts), formatSize(total)),
		"format", format,
		"documents", len(artifacts),
		"bytes", total,
	)
	return artifacts, nil
}
