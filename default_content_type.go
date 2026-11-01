package asyncgo

type contentTypeItem string

// DefaultContentType sets the document's default content type.
func DefaultContentType(s string) Item { return contentTypeItem(s) }

func (c contentTypeItem) apply(
	b *builder,
) error {
	b.doc.DefaultContentType = string(c)
	return nil
}
