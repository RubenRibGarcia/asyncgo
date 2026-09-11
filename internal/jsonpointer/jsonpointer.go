// Package jsonpointer implements the reference-token escaping rules of
// RFC 6901 (JSON Pointer).
//
// It is shared by the asyncgo DSL and the schema package, which both build
// $ref pointers into an AsyncAPI document.
package jsonpointer

import "strings"

// Escape applies RFC 6901 escaping to a JSON Pointer reference token:
// "~" -> "~0" and "/" -> "~1" (in that order).
func Escape(s string) string {
	s = strings.ReplaceAll(s, "~", "~0")
	s = strings.ReplaceAll(s, "/", "~1")
	return s
}

// Unescape reverses [Escape], turning "~1" back into "/" and "~0" back into
// "~" (in that order).
func Unescape(s string) string {
	s = strings.ReplaceAll(s, "~1", "/")
	s = strings.ReplaceAll(s, "~0", "~")
	return s
}
