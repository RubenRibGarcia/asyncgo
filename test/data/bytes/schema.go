// Package bytes is a discovery fixture for the []byte derivation: a byte slice
// derives to a base64-encoded string (format: byte plus contentEncoding: base64,
// the Draft-07 keyword matching what encoding/json does), and the content
// keywords are overridable per field from the asyncapi tag.
package bytes

// Attachment is the message payload. Its []byte fields exercise the derivation,
// and Thumbnail exercises the contentMediaType directive.
type Attachment struct {
	// Name is the attachment's file name.
	Name string `json:"name" asyncapi:"required"`
	// Body is the raw file content, derived as base64.
	Body []byte `json:"body" asyncapi:"required"`
	// Thumbnail is a preview image, whose media type the Go type cannot carry.
	Thumbnail []byte `json:"thumbnail" asyncapi:"contentMediaType=image/png"`
}
