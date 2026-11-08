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

// Directive is one whitespace-separated part of a struct tag, split at the first
// "=" into a key and its value. HasValue separates a value directive — even one
// with an empty value, as in "format=" — from a bare flag, as in "readOnly". The
// distinction matters: "allOf" marks an embedded struct for composition, while
// "allOf=A" composes named types.
type Directive struct {
	Key      string
	Value    string
	HasValue bool
}

// Parse splits a struct tag into its directives. Directives are separated by
// whitespace, so a value may contain a comma — "examples=1,000" yields the single
// example "1,000" — but not a space. A legacy comma-separated tag is therefore one
// unrecognised directive, and the callers ignore it like any other unknown key.
func Parse(tag string) []Directive {
	parts := strings.Fields(tag)
	if len(parts) == 0 {
		return nil
	}
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
