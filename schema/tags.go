package schema

import (
	"strconv"
	"strings"

	"github.com/RubenRibGarcia/asyncgo/spec"
)

// directive is one whitespace-separated part of an asyncapi struct tag, split at
// the first "=" into a key and its value. hasValue separates a value directive
// (even one with an empty value, as in "format=") from a bare flag (readOnly,
// required, allOf).
type directive struct {
	key      string
	value    string
	hasValue bool
}

// parseTag splits an asyncapi struct tag into its directives once. Every reader —
// applyTag for the value directives, hasFlag for the bare flags, combinatorNames
// for the member lists — works on the result, instead of each re-splitting the
// same tag.
func parseTag(tag string) []directive {
	parts := strings.Fields(tag)
	if len(parts) == 0 {
		return nil
	}
	ds := make([]directive, len(parts))
	for i, part := range parts {
		key, value, hasValue := strings.Cut(part, "=")
		ds[i] = directive{key: key, value: value, hasValue: hasValue}
	}
	return ds
}

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
func applyTag(s *spec.Schema, ds []directive) {
	for _, d := range ds {
		if !d.hasValue {
			switch d.key {
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
		switch d.key {
		case "enum":
			if d.value != "" {
				for e := range strings.SplitSeq(d.value, "|") {
					s.Enum = append(s.Enum, e)
				}
			}
		case "examples":
			if d.value != "" {
				s.Examples = append(s.Examples, d.value)
			}
		case "format":
			s.Format = d.value
		case "contentEncoding":
			if d.value != "" {
				s.ContentEncoding = d.value
			}
		case "contentMediaType":
			if d.value != "" {
				s.ContentMediaType = d.value
			}
		case "const":
			if d.value != "" {
				s.Const = d.value
			}
		case "discriminator":
			if d.value != "" {
				s.Discriminator = d.value
			}
		case "minItems":
			setUint(&s.MinItems, d.value)
		case "maxItems":
			setUint(&s.MaxItems, d.value)
		case "minProperties":
			setUint(&s.MinProperties, d.value)
		case "maxProperties":
			setUint(&s.MaxProperties, d.value)
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

// hasFlag reports whether ds carries flag as a bare directive. A value directive
// of the same key is not a flag: "allOf=A" composes types, "allOf" marks an
// embedded struct for composition.
func hasFlag(ds []directive, flag string) bool {
	for _, d := range ds {
		if d.key == flag && !d.hasValue {
			return true
		}
	}
	return false
}

// combinatorNames returns the "|"-separated type names for a combinator
// directive (oneOf=/anyOf=/allOf=), or ok=false when the directive is absent or
// has an empty value.
func combinatorNames(ds []directive, key string) ([]string, bool) {
	for _, d := range ds {
		if d.key == key && d.hasValue {
			if d.value == "" {
				return nil, false
			}
			return strings.Split(d.value, "|"), true
		}
	}
	return nil, false
}
