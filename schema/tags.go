package schema

import (
	"fmt"
	"sort"
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
// is ignored rather than reported: the FromType/fillFields/applyTag chain returns
// no error. So is an unknown directive, and that leniency is deliberate rather
// than vestigial: the reflection pass also runs in process, where the static pass
// that validates tags (internal/discovery) never ran. CheckTag is the arbiter for
// the CLI path, which rejects a directive this function would ignore before the
// generated harness is built. See the design doc
// (docs/designdoc/schema-keywords.md, D6 and O2).
//
// Directives are separated by whitespace — `asyncapi:"required examples=1,000"`
// yields the example "1,000" — so a value may contain a comma but not a space. A
// legacy comma-separated tag is a single unknown directive: CheckTag names it,
// and applyTag ignores it.
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

// directiveKind is the syntax a directive key accepts. It is the whole of what
// CheckTag enforces beyond the key itself.
type directiveKind int

const (
	// flagDirective is a bare key, as in "readOnly". A value is a mistake:
	// "readOnly=true" would otherwise parse as a value directive no case matches.
	flagDirective directiveKind = iota

	// flagOrValueDirective is legal either way: bare "allOf" composes an embedded
	// struct, while "allOf=A|B" names the types to compose.
	flagOrValueDirective

	// valueDirective takes one non-empty value, as in "format=date-time".
	valueDirective

	// listDirective takes a non-empty "|"-separated list, as in "enum=draft|sent".
	// No member may be empty: "oneOf=A|" is a typo, not a member named "".
	listDirective
)

// directiveKinds is the closed vocabulary of the asyncapi tag: every directive
// applyTag applies, plus the four fillFields applies (required, and the three
// combinators). A key that is not here is rejected by CheckTag.
var directiveKinds = map[string]directiveKind{
	"allOf":            flagOrValueDirective,
	"anyOf":            listDirective,
	"const":            valueDirective,
	"contentEncoding":  valueDirective,
	"contentMediaType": valueDirective,
	"deprecated":       flagDirective,
	"discriminator":    valueDirective,
	"enum":             listDirective,
	"examples":         valueDirective,
	"format":           valueDirective,
	"maxItems":         valueDirective,
	"maxProperties":    valueDirective,
	"minItems":         valueDirective,
	"minProperties":    valueDirective,
	"oneOf":            listDirective,
	"readOnly":         flagDirective,
	"required":         flagDirective,
	"uniqueItems":      flagDirective,
	"writeOnly":        flagDirective,
}

// DirectiveKeys returns the directives asyncgo honours, sorted. It is the
// vocabulary CheckTag accepts, exported so help text can name it and a test can
// pin it.
func DirectiveKeys() []string {
	keys := make([]string, 0, len(directiveKinds))
	for key := range directiveKinds {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// CheckTag validates one asyncapi struct tag, returning one error per violation
// and nil when the tag is clean.
//
// This is what makes the vocabulary closed. applyTag ignores a key it does not
// know, so without CheckTag a tag carrying a misspelt directive derives a schema
// missing the constraint its author meant to add and says nothing about it. The
// static pass in internal/discovery calls this for every tagged field, so a typo
// fails generate and check instead of quietly changing the document.
func CheckTag(tag string) []error {
	var errs []error
	for _, d := range taggrammar.Parse(tag) {
		errs = append(errs, checkDirective(d)...)
	}
	return errs
}

// checkDirective applies the syntax rule for one directive.
func checkDirective(d taggrammar.Directive) []error {
	if d.Key == "" {
		return []error{fmt.Errorf("directive has no name")}
	}
	if strings.Contains(d.Key, ",") {
		// The grammar splits on whitespace, so the old comma-separated form
		// arrives as one key: "required,examples=x" is a single directive
		// named "required,examples".
		return []error{fmt.Errorf(
			"%q: directives are separated by spaces, not commas", d.Key)}
	}
	kind, known := directiveKinds[d.Key]
	if !known {
		return unknownDirective(d.Key)
	}
	switch kind {
	case flagDirective:
		if d.HasValue {
			return []error{fmt.Errorf("%q is a flag and takes no value", d.Key)}
		}
	case valueDirective:
		return checkValue(d)
	case listDirective:
		if errs := checkValue(d); len(errs) > 0 {
			return errs // "oneOf" alone has no value to split into members
		}
		return checkList(d)
	case flagOrValueDirective:
		if d.HasValue {
			if errs := checkValue(d); len(errs) > 0 {
				return errs
			}
			return checkList(d)
		}
	}
	return nil
}

// checkValue reports a value directive that is missing its value or spells it as
// empty. Every one of these is a no-op at derivation time, so nothing that works
// today starts failing.
func checkValue(d taggrammar.Directive) []error {
	switch {
	case !d.HasValue:
		return []error{fmt.Errorf("%q requires a value", d.Key)}
	case d.Value == "":
		return []error{fmt.Errorf("%q has an empty value", d.Key)}
	}
	return nil
}

// checkList reports an empty member in a "|"-separated list. One error is enough:
// the message is the same for every empty member, and the fix is the same too.
func checkList(d taggrammar.Directive) []error {
	for member := range strings.SplitSeq(d.Value, "|") {
		if member == "" {
			return []error{fmt.Errorf("%q has an empty entry", d.Key)}
		}
	}
	return nil
}

// unknownDirective names the closest known directive when the key looks like a
// typo, and lists the vocabulary when it does not. It returns a slice like its
// siblings, so every rule in this file has one shape.
func unknownDirective(key string) []error {
	if close := closestDirective(key); close != "" {
		return []error{fmt.Errorf("unknown directive %q; did you mean %q?", key, close)}
	}
	return []error{fmt.Errorf(
		"unknown directive %q; want one of %s",
		key,
		strings.Join(DirectiveKeys(), ", "),
	)}
}

// closestDirective returns the known directive within two edits of key, or ""
// when none is close enough to be a plausible typo. Two edits cover a doubled or
// transposed letter without inventing a suggestion for a genuinely unrelated key.
func closestDirective(key string) string {
	closest, best := "", 3
	for _, known := range DirectiveKeys() {
		if d := editDistance(key, known); d < best {
			closest, best = known, d
		}
	}
	return closest
}

// editDistance is the Levenshtein distance between a and b.
func editDistance(a, b string) int {
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}
