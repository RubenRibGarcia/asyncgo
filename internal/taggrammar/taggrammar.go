// Package taggrammar parses the struct-tag syntax asyncgo reads in two places but
// in two processes: the reflection pass in schema/ derives Schema Objects from
// reflect.StructField, and the static pass in internal/discovery walks go/ast over
// the same source to collect doc comments and combinator types.
//
// It returns syntax only. Which directives exist, what a bare flag means and how a
// combinator's member names resolve are the callers' business: the two passes ask
// different questions of the same tag, and only the grammar has to agree.
//
// See docs/adr/0007-shared-tag-grammar.md.
package taggrammar

import "strings"

// The two levels of the asyncapi tag grammar. Directives are separated by
// DirectiveSeparator and the members of a list directive by ListSeparator; the
// characters differ so that a member can never be mistaken for the next
// directive. See docs/adr/0014-semicolon-separated-tag-directives.md.
const (
	DirectiveSeparator = ";"
	ListSeparator      = ","
)

// Directive is one DirectiveSeparator-separated part of a struct tag, split at
// the first "=" into a key and its value. HasValue separates a value directive —
// even one with an empty value, as in "format=" — from a bare flag, as in
// "readOnly". The distinction matters: "allOf" marks an embedded struct for
// composition, while "allOf=A" composes named types.
type Directive struct {
	Key      string
	Value    string
	HasValue bool
}

// Parse splits a struct tag into its directives. Directives are separated by
// ";", so a value may contain a comma and a "|": "examples=1,000" and
// "enum=a|b" are single values. The value of a list directive ("enum=draft,sent")
// is not split here; List and SplitList do that. A legacy whitespace-separated
// tag is one directive whose key contains a space, which the callers reject.
func Parse(tag string) []Directive {
	if tag == "" {
		return nil
	}
	parts := strings.Split(tag, DirectiveSeparator)
	ds := make([]Directive, len(parts))
	for i, part := range parts {
		key, value, hasValue := strings.Cut(part, "=")
		ds[i] = Directive{Key: key, Value: value, HasValue: hasValue}
	}
	return ds
}

// Flag reports whether the tag carries flag as a bare directive. A value directive
// of the same key is not a flag.
func Flag(ds []Directive, flag string) bool {
	for _, d := range ds {
		if d.Key == flag && !d.HasValue {
			return true
		}
	}
	return false
}

// Value returns the value of the first directive with the given key, or ok=false
// when the key is absent or carries no value: "format=" reports ok=true with an
// empty value, "format" alone reports ok=false.
func Value(ds []Directive, key string) (string, bool) {
	for _, d := range ds {
		if d.Key == key && d.HasValue {
			return d.Value, true
		}
	}
	return "", false
}

// SplitList splits a list directive's value into its members. It validates
// nothing: an empty member is a caller's CheckTag error, and an empty value
// yields one empty member.
func SplitList(value string) []string {
	return strings.Split(value, ListSeparator)
}

// List returns the members of the first directive with the given key, or
// ok=false when the key is absent or carries no value. It is the one split both
// passes use, so a list directive cannot be read two ways.
func List(ds []Directive, key string) ([]string, bool) {
	raw, ok := Value(ds, key)
	if !ok || raw == "" {
		return nil, false
	}
	return SplitList(raw), true
}

// JSONName reports the field name a json tag declares, with skip true for "-",
// which excludes the field. An empty name means the tag named none
// ("json:",omitempty") and the caller falls back to the Go field name, matching
// encoding/json.
//
// Only a tag that is exactly "-" skips: "-" followed by options names a field "-",
// which is what the reflection pass has always done.
func JSONName(tag string) (name string, skip bool) {
	if tag == "-" {
		return "", true
	}
	name, _, _ = strings.Cut(tag, ",")
	return name, false
}
