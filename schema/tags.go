package schema

import (
	"strings"

	"github.com/RubenRibGarcia/asyncgo/spec"
)

// applyTag applies the asyncgo struct tag to the schema. Supported directives:
//
//	required       (handled by the caller; ignored here)
//	description=...  human description
//	enum=a|b|c       enumerated string values
//	example=...      example value
//	format=...       JSON Schema format (e.g. "date-time", "uuid", "email")
func applyTag(s *spec.Schema, tag string) {
	for _, part := range strings.Split(tag, ",") {
		part = strings.TrimSpace(part)
		switch {
		case part == "" || part == "required":
			// nothing to set on the schema itself
		case strings.HasPrefix(part, "description="):
			s.Description = strings.TrimPrefix(part, "description=")
		case strings.HasPrefix(part, "enum="):
			for _, v := range strings.Split(strings.TrimPrefix(part, "enum="), "|") {
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
	for _, part := range strings.Split(tag, ",") {
		if strings.TrimSpace(part) == flag {
			return true
		}
	}
	return false
}
