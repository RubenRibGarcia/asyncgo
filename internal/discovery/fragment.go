package discovery

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/goccy/go-yaml"
)

// FragmentFile is the optional hand-authored fragment deep-merged over a
// generated document (see spec.Overlay). It lets parts of the spec be authored
// in YAML instead of a Go catalog.
const FragmentFile = "asyncapi.fragment.yaml"

// ApplyFragment reads FragmentFile from dir and overlays it on doc. It returns
// doc unchanged when no fragment file exists.
func ApplyFragment(dir string, doc *spec.AsyncAPI) (*spec.AsyncAPI, error) {
	path := filepath.Join(dir, FragmentFile)
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return doc, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", FragmentFile, err)
	}

	var frag spec.AsyncAPI
	if err := yaml.Unmarshal(raw, &frag); err != nil {
		return nil, fmt.Errorf("parsing %s: %v", FragmentFile, err)
	}
	merged, err := spec.Overlay(doc, &frag)
	if err != nil {
		return nil, fmt.Errorf("merging %s: %v", FragmentFile, err)
	}
	return merged, nil
}
