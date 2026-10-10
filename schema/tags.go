package schema

import (
	"strconv"
	"strings"

	"github.com/RubenRibGarcia/asyncgo/spec"
)

// applyTag applies the asyncapi struct tag to the schema. Supported directives:
//
//	required         (handled by the caller; ignored here)
//	enum=a|b|c       enumerated string values
//	examples=...     append one Draft-07 examples entry; repeatable
//	format=...       JSON Schema format (e.g. "date-time", "uuid", "email")
//	const=...        constant value
//	discriminator=.. polymorphism property name
//	minItems=N       array length bounds
//	maxItems=N
//	minProperties=N  object size bounds
//	maxProperties=N
//	readOnly         bare flags setting the matching boolean keyword
//	writeOnly
//	uniqueItems
//	deprecated
//	oneOf=A|B        (handled by the caller via combinatorNames; ignored here)
//	anyOf=A|B        (handled by the caller via combinatorNames; ignored here)
//	allOf=A|B        (handled by the caller via combinatorNames; ignored here)
//
// A malformed value — a non-numeric bound, an empty examples= — is ignored
// rather than reported: the FromType/fillFields/applyTag chain returns no error,
// matching how an unknown directive is already treated. A directive that no
// longer exists is ignored the same way, so the removed `example=` is silently
// dropped rather than mapped onto `examples=`. See the design doc
// (docs/designdoc/schema-keywords.md, D6).
//
// Descriptions are not carried in the tag; the generator's discovery pass reads
// them from the field's doc comment instead.
func applyTag(s *spec.Schema, tag string) {
	for part := range strings.SplitSeq(tag, ",") {
		part = strings.TrimSpace(part)
		switch {
		case part == "" || part == "required":
			// nothing to set on the schema itself
		case part == "readOnly":
			s.ReadOnly = true
		case part == "writeOnly":
			s.WriteOnly = true
		case part == "uniqueItems":
			s.UniqueItems = true
		case part == "deprecated":
			s.Deprecated = true
		case strings.HasPrefix(part, "enum="):
			for v := range strings.SplitSeq(strings.TrimPrefix(part, "enum="), "|") {
				s.Enum = append(s.Enum, v)
			}
		case strings.HasPrefix(part, "examples="):
			if v := strings.TrimPrefix(part, "examples="); v != "" {
				s.Examples = append(s.Examples, v)
			}
		case strings.HasPrefix(part, "format="):
			s.Format = strings.TrimPrefix(part, "format=")
		case strings.HasPrefix(part, "const="):
			if v := strings.TrimPrefix(part, "const="); v != "" {
				s.Const = v
			}
		case strings.HasPrefix(part, "discriminator="):
			if v := strings.TrimPrefix(part, "discriminator="); v != "" {
				s.Discriminator = v
			}
		case strings.HasPrefix(part, "minItems="):
			setUint(&s.MinItems, strings.TrimPrefix(part, "minItems="))
		case strings.HasPrefix(part, "maxItems="):
			setUint(&s.MaxItems, strings.TrimPrefix(part, "maxItems="))
		case strings.HasPrefix(part, "minProperties="):
			setUint(&s.MinProperties, strings.TrimPrefix(part, "minProperties="))
		case strings.HasPrefix(part, "maxProperties="):
			setUint(&s.MaxProperties, strings.TrimPrefix(part, "maxProperties="))
		}
	}
}

// setUint parses value as a decimal uint64 and points *dst at it. A value that
// does not parse leaves *dst alone — see the malformed-value note on applyTag.
func setUint(dst **uint64, value string) {
	n, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return
	}
	*dst = &n
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
