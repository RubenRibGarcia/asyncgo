package asyncgo

import (
	"errors"
	"fmt"

	"github.com/RubenRibGarcia/asyncgo/spec"
)

type infoBuilder struct {
	info spec.Info
}

// Info starts the info section with the required title and version.
func Info(title, version string) *infoBuilder {
	return &infoBuilder{info: spec.Info{Title: title, Version: version}}
}

func (i *infoBuilder) Description(s string) *infoBuilder    { i.info.Description = s; return i }
func (i *infoBuilder) TermsOfService(s string) *infoBuilder { i.info.TermsOfService = s; return i }
func (i *infoBuilder) Contact(c spec.Contact) *infoBuilder  { i.info.Contact = &c; return i }
func (i *infoBuilder) License(l spec.License) *infoBuilder  { i.info.License = &l; return i }
func (i *infoBuilder) Tags(tags ...spec.Tag) *infoBuilder   { i.info.Tags = tags; return i }

func (i *infoBuilder) apply(b *builder) error {
	var errs []error
	if i.info.Title == "" {
		errs = append(errs, fmt.Errorf("info.title: is required"))
	}
	if i.info.Version == "" {
		errs = append(errs, fmt.Errorf("info.version: is required"))
	}
	b.doc.Info = i.info
	return errors.Join(errs...)
}
