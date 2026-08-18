package schema

import (
	"strings"

	"github.com/RubenRibGarcia/asyncgo/spec"
)

// applyTag applies the asyncapi struct tag to the schema. Supported directives:
//
//	required       (handled by the caller; ignored here)
//	enum=a|b|c       enumerated string values
//	example=...      example value
//	format=...       JSON Schema format (e.g. "date-time", "uuid", "email")
//	oneOf=A|B        (handled by the caller via combinatorNames; ignored here)
//	anyOf=A|B        (handled by the caller via combinatorNames; ignored here)
//	allOf=A|B        (handled by the caller via combinatorNames; ignored here)
//
// Descriptions are not carried in the tag; the generator's discovery pass reads
// them from the field's doc comment instead.
func applyTag(s *spec.Schema, tag string) {
	for part := range strings.SplitSeq(tag, ",") {
		part = strings.TrimSpace(part)
		switch {
		case part == "" || part == "required":
			// nothing to set on the schema itself
		case strings.HasPrefix(part, "enum="):
			for v := range strings.SplitSeq(strings.TrimPrefix(part, "enum="), "|") {
				s.Enum = append(s.Enum, v)
			}
		case strings.HasPrefix(part, "example="):
			s.Example = strings.TrimPrefix(part, "example=")
		case strings.HasPrefix(part, "format="):
			s.Format = strings.TrimPrefix(part, "format=")
		}
	}
}

func hasFlag(tag, flag string) bool {
	for part := range strings.SplitSeq(tag, ",") {
		if strings.TrimSpace(part) == flag {
			return true
		}
	}
	return false
}

// combinatorNames returns the "|"-separated type names for a combinator
// directive (oneOf=/anyOf=/allOf=), or ok=false when the directive is absent.
func combinatorNames(tag, key string) ([]string, bool) {
	prefix := key + "="
	for part := range strings.SplitSeq(tag, ",") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, prefix) {
			raw := strings.TrimPrefix(part, prefix)
			if raw == "" {
				return nil, false
			}
			return strings.Split(raw, "|"), true
		}
	}
	return nil, false
}
