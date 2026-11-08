// Package mappingconfig parses and validates the schema-mapping configuration.
//
// Two consumers share it, deliberately: the schema package loads its embedded
// built-in table through Parse at init, and internal/discovery parses the
// project's asyncgo.yaml through the same call. One parser is what makes a
// built-in exactly a config entry — extending the built-in set is a data-only
// change — and it guarantees a project's file cannot express a mapping the
// built-ins could not, or vice versa.
//
// The validated config is exposed as compact JSON rather than as a
// *spec.Schema: the generated harness embeds each mapping as a
// schema.MapSchemaJSON argument, so JSON is the one shape both consumers can
// use without re-rendering Schema literals (which would need regex escaping and
// []any round-trips for enum).
package mappingconfig

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/token"
	"sort"
	"strings"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/goccy/go-yaml"
)

// Section is the only top-level key a config file may carry. The file is shaped
// for growth — a future section is an additive change — but today anything else
// is rejected rather than silently ignored, so a typo cannot quietly disable a
// feature.
const Section = "mappings"

// allowedKeys is the whitelist of Schema Object keywords a mapping value may
// set: annotations, string/numeric/array constraints, and the access flags.
// Everything structural is out of scope for v1 — properties, items, $ref and the
// combinators describe a shape rather than a scalar wire format, which is what a
// mapping is for. Kept sorted so the rejection message is deterministic.
var allowedKeys = []string{
	"const",
	"contentEncoding",
	"contentMediaType",
	"default",
	"deprecated",
	"description",
	"enum",
	"examples",
	"exclusiveMaximum",
	"exclusiveMinimum",
	"format",
	"maxItems",
	"maxLength",
	"maximum",
	"minItems",
	"minLength",
	"minimum",
	"pattern",
	"readOnly",
	"type",
	"writeOnly",
}

// allowedKey is allowedKeys as a set, so a lookup is not a linear scan.
var allowedKey = func() map[string]bool {
	out := make(map[string]bool, len(allowedKeys))
	for _, k := range allowedKeys {
		out[k] = true
	}
	return out
}()

// Mapping is one validated mapping: a type's fully-qualified name and the Schema
// Object that overrides its derivation, encoded as compact JSON.
type Mapping struct {
	FQN    string // pkgPath.TypeName
	Schema string // compact JSON of the Schema Object
}

// Config is the parsed contents of a config file. Mappings are sorted by FQN, so
// an emitted harness is byte-identical between runs.
type Config struct {
	Mappings []Mapping
}

// Parse validates a config file and returns its mappings.
//
// An empty document declares nothing and is not an error — a project may keep an
// asyncgo.yaml for the sections a later release adds. A missing Section is
// treated the same way. Everything else is checked here, before any harness is
// generated: unknown sections, unknown keywords, non-fully-qualified type names,
// nested objects, and values that do not decode into a spec.Schema.
func Parse(data []byte) (*Config, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return &Config{}, nil
	}

	doc := map[string]any{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	keys := make([]string, 0, len(doc))
	for key := range doc {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if key != Section {
			return nil, fmt.Errorf("unknown section %q; allowed: %s", key, Section)
		}
	}

	raw, ok := doc[Section]
	if !ok {
		return &Config{}, nil // no mappings section: nothing to register
	}
	entries, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf(
			"%s: want a mapping of pkgPath.TypeName to a Schema Object",
			Section,
		)
	}

	fqns := make([]string, 0, len(entries))
	for fqn := range entries {
		fqns = append(fqns, fqn)
	}
	sort.Strings(fqns)

	cfg := &Config{Mappings: make([]Mapping, 0, len(fqns))}
	for _, fqn := range fqns {
		if err := validateFQN(fqn); err != nil {
			return nil, fmt.Errorf("%s: %w", Section, err)
		}
		value, ok := entries[fqn].(map[string]any)
		if !ok || value == nil {
			return nil, fmt.Errorf(
				"%s.%s: want a mapping of schema keywords, e.g. {type: string}",
				Section,
				fqn,
			)
		}
		if err := validateValue(fqn, value); err != nil {
			return nil, err
		}
		payload, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("%s.%s: %w", Section, fqn, err)
		}
		// Decoding the payload here, rather than leaving it to the harness's
		// MapSchemaJSON call, keeps a mistyped value (minLength: lots) a config
		// error with a file-level message instead of a panic inside a generated
		// program.
		var s spec.Schema
		if err := json.Unmarshal(payload, &s); err != nil {
			return nil, fmt.Errorf("%s.%s: %w", Section, fqn, err)
		}
		cfg.Mappings = append(cfg.Mappings, Mapping{FQN: fqn, Schema: string(payload)})
	}
	return cfg, nil
}

// validateFQN applies the same shape rule combinator directives use: a mapping is
// keyed by pkgPath.TypeName. A semi-qualified pkg.Type cannot match the name
// reflection produces, so it is rejected rather than silently never consulted.
func validateFQN(fqn string) error {
	i := strings.LastIndex(fqn, ".")
	if i <= 0 || i == len(fqn)-1 || !strings.Contains(fqn, "/") {
		return fmt.Errorf("%q is not a fully-qualified type name; use pkgPath.TypeName", fqn)
	}
	if name := fqn[i+1:]; !token.IsIdentifier(name) {
		return fmt.Errorf("%q is not a Go type name", name)
	}
	return nil
}

// validateValue checks one mapping's keywords against the whitelist and rejects a
// nested object anywhere in a value.
func validateValue(fqn string, value map[string]any) error {
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if !allowedKey[key] {
			return fmt.Errorf(
				"%s.%s: unknown keyword %q; allowed: %s",
				Section,
				fqn,
				key,
				strings.Join(allowedKeys, ", "),
			)
		}
		if err := rejectNested(fqn, key, value[key]); err != nil {
			return err
		}
	}
	return nil
}

// rejectNested rejects an object anywhere in a value, including inside a list. A
// nested object is a combinator member, a $ref-bearing schema, or a mistake —
// all outside the scalar-shaped surface a mapping whitelists.
func rejectNested(fqn, key string, value any) error {
	switch v := value.(type) {
	case map[string]any:
		return fmt.Errorf(
			"%s.%s: %q must be a scalar or a list, not a nested object",
			Section,
			fqn,
			key,
		)
	case []any:
		for _, item := range v {
			if err := rejectNested(fqn, key, item); err != nil {
				return err
			}
		}
	}
	return nil
}
