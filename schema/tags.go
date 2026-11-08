package schema

import (
	"strconv"
	"strings"

	"github.com/RubenRibGarcia/asyncgo/internal/taggrammar"
	"github.com/RubenRibGarcia/asyncgo/spec"
)

// applyTag applies the asyncapi struct tag to the schema. Supported directives:
//
//	required             (handled by the caller; ignored here)
//	enum=a|b|c           enumerated string values
//	examples=...         append one Draft-07 examples entry; repeatable
//	format=...           JSON Schema format (e.g. "date-time", "uuid", "email")
//	contentEncoding=...  Draft-07 content encoding (e.g. "base64"); overriding it
//	                     on a []byte field leaves the derived format=byte alone
//	contentMediaType=... Draft-07 media type (e.g. "image/png"); never derived
//	const=...            constant value
//	discriminator=..     polymorphism property name
//	minItems=N           array length bounds
//	maxItems=N
//	minProperties=N      object size bounds
//	maxProperties=N
//	readOnly             bare flags setting the matching boolean keyword
//	writeOnly
//	uniqueItems
//	deprecated
//	oneOf=A|B            (handled by the caller via combinatorNames; ignored here)
//	anyOf=A|B            (handled by the caller via combinatorNames; ignored here)
//	allOf=A|B            (handled by the caller via combinatorNames; ignored here)
//
// A malformed value — a non-numeric bound, an empty examples=, an empty enum= —
// is ignored
// rather than reported: the FromType/fillFields/applyTag chain returns no error,
// matching how an unknown directive is already treated. A directive that no
// longer exists is ignored the same way, so the removed `example=` is silently
// dropped rather than mapped onto `examples=`. See the design doc
// (docs/designdoc/schema-keywords.md, D6).
//
// Directives are separated by whitespace — `asyncapi:"required examples=1,000"`
// yields the example "1,000" — so a value may contain a comma but not a space. A
// legacy comma-separated tag is a single unknown directive and is therefore
// ignored like any other.
//
// Descriptions are not carried in the tag; the generator's discovery pass reads
// them from the field's doc comment instead.
func applyTag(s *spec.Schema, ds []taggrammar.Directive) {
	for _, d := range ds {
		if !d.HasValue {
			switch d.Key {
			case "readOnly":
				s.ReadOnly = true
			case "writeOnly":
				s.WriteOnly = true
			case "uniqueItems":
				s.UniqueItems = true
			case "deprecated":
				s.Deprecated = true
			}
			continue
		}
		switch d.Key {
		case "enum":
			if d.Value != "" {
				for e := range strings.SplitSeq(d.Value, "|") {
					s.Enum = append(s.Enum, e)
				}
			}
		case "examples":
			if d.Value != "" {
				s.Examples = append(s.Examples, d.Value)
			}
		case "format":
			s.Format = d.Value
		case "contentEncoding":
			if d.Value != "" {
				s.ContentEncoding = d.Value
			}
		case "contentMediaType":
			if d.Value != "" {
				s.ContentMediaType = d.Value
			}
		case "const":
			if d.Value != "" {
				s.Const = d.Value
			}
		case "discriminator":
			if d.Value != "" {
				s.Discriminator = d.Value
			}
		case "minItems":
			setUint(&s.MinItems, d.Value)
		case "maxItems":
			setUint(&s.MaxItems, d.Value)
		case "minProperties":
			setUint(&s.MinProperties, d.Value)
		case "maxProperties":
			setUint(&s.MaxProperties, d.Value)
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

// combinatorNames returns the "|"-separated type names for a combinator
// directive (oneOf=/anyOf=/allOf=), or ok=false when the directive is absent or
// has an empty value.
func combinatorNames(ds []taggrammar.Directive, key string) ([]string, bool) {
	raw, ok := taggrammar.Value(ds, key)
	if !ok || raw == "" {
		return nil, false
	}
	return strings.Split(raw, "|"), true
}
