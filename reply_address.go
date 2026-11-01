package asyncgo

import (
	"errors"
	"fmt"

	"github.com/RubenRibGarcia/asyncgo/spec"
)

type replyAddressesItem []*replyAddress

// ReplyAddresses adds one or more reusable reply addresses to the document's
// components.replyAddresses. Reference one from a reply via Reply.Address.
func ReplyAddresses(a ...*replyAddress) Item { return replyAddressesItem(a) }

func (a replyAddressesItem) apply(b *builder) error {
	var errs []error
	for _, ra := range a {
		if err := ra.apply(b); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

type replyAddress struct {
	name string
	a    spec.OperationReplyAddress
}

// ReplyAddress declares a reusable reply address under the given name: the
// runtime expression that locates where an operation's reply is sent.
func ReplyAddress(name string) *replyAddress { return &replyAddress{name: name} }

func (a *replyAddress) Description(d string) *replyAddress { a.a.Description = d; return a }
func (a *replyAddress) Location(l string) *replyAddress    { a.a.Location = l; return a }

func (a *replyAddress) apply(b *builder) error {
	var errs []error
	if a.name == "" {
		errs = append(errs, fmt.Errorf("replyAddress.name: is required"))
	}
	errs = appendKeyError(errs, "replyAddress", a.name)
	if a.a.Location == "" {
		errs = append(errs, fmt.Errorf("replyAddress.%s.location: is required", a.name))
	}
	c := b.components()
	if c.ReplyAddresses == nil {
		c.ReplyAddresses = map[string]*spec.OperationReplyAddress{}
	}
	if _, dup := c.ReplyAddresses[a.name]; dup && a.name != "" {
		errs = append(errs, fmt.Errorf("replyAddress.%s: duplicate name", a.name))
	} else {
		c.ReplyAddresses[a.name] = &a.a
	}
	return errors.Join(errs...)
}
