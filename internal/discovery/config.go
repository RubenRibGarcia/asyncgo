package discovery

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/RubenRibGarcia/asyncgo/internal/mappingconfig"
)

// configFileName is the project config file discovery reads from the directory a
// document is generated for — the module root, in practice, next to the artifact
// it shapes. One filename, no flag: both generate and check read it, so the two
// commands cannot drift apart, and CI does not have to repeat a flag.
const configFileName = "asyncgo.yaml"

// loadMappingConfig reads <dir>/asyncgo.yaml and returns its validated mappings.
//
// A missing file contributes nothing — the config is optional, and a project with
// no third-party types needs none. An unreadable or invalid file is an error
// rather than a warning: the file exists, so the project meant to configure
// something, and silently generating a document that ignores it would be worse
// than failing. Validation happens here, before any harness is generated.
func loadMappingConfig(dir string) ([]mappingconfig.Mapping, error) {
	data, err := os.ReadFile(filepath.Join(dir, configFileName))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading %s: %w", configFileName, err)
	}
	cfg, err := mappingconfig.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", configFileName, err)
	}
	return cfg.Mappings, nil
}
