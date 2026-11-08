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

// Location is the runtime expression the reply address resolves to, e.g.
// "$message.header#/replyTo". It is required: a reply address without one
// names no location.
func (a *replyAddress) Location(l string) *replyAddress { a.a.Location = l; return a }

func (a *replyAddress) apply(b *builder) error {
	var errs []error
	if a.name == "" {
		errs = append(errs, fmt.Errorf("replyAddress.name: is required"))
	}
	errs = appendKeyError(errs, "replyAddress", a.name)
	if a.a.Location == "" {
		errs = append(errs, fmt.Errorf("replyAddress.%s.location: is required", a.name))
	}
	errs = append(errs, registerComponent(b, replyAddressesMap, "replyAddress", a.name, &a.a)...)
	return errors.Join(errs...)
}
