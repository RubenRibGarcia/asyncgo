package asyncgo

import (
	"errors"
	"fmt"

	"github.com/RubenRibGarcia/asyncgo/spec"
)

type repliesItem []*reply

// Replies adds one or more reusable replies to the document's
// components.replies. Reference one from an operation via Operation.Reply.
func Replies(r ...*reply) Item { return repliesItem(r) }

func (r repliesItem) apply(b *builder) error {
	var errs []error
	for _, rp := range r {
		if err := rp.apply(b); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

type reply struct {
	name string
	r    spec.OperationReply
}

// Reply declares a reusable reply under the given name. Reference it from an
// operation via Operation.Reply.
func Reply(name string) *reply { return &reply{name: name} }

// Address sets the reply address the reply is sent to. The address must be
// declared via ReplyAddresses(...).
func (r *reply) Address(a *replyAddress) *reply {
	r.r.Address = replyAddressRef(a)
	return r
}

// Channel sets the channel the reply is performed in. The channel must be
// declared via Channels(...).
func (r *reply) Channel(ch *channel) *reply {
	r.r.Channel = channelRef(ch)
	return r
}

// Message attaches the messages the reply can carry on ch. The channel is a
// parameter because every reply message ref has to point into that channel's
// messages, which keeps Message independent of the order Channel is called in.
// The reply still has to name its channel via Channel(...) to be valid.
func (r *reply) Message(ch *channel, m ...*message) *reply {
	for _, msg := range m {
		r.r.Messages = append(r.r.Messages, channelMessageRef(ch, msg))
	}
	return r
}

func (r *reply) apply(b *builder) error {
	var errs []error
	if r.name == "" {
		errs = append(errs, fmt.Errorf("reply.name: is required"))
	}
	errs = appendKeyError(errs, "reply", r.name)
	c := b.components()
	if c.Replies == nil {
		c.Replies = map[string]*spec.OperationReply{}
	}
	if _, dup := c.Replies[r.name]; dup && r.name != "" {
		errs = append(errs, fmt.Errorf("reply.%s: duplicate name", r.name))
	} else {
		c.Replies[r.name] = &r.r
	}
	return errors.Join(errs...)
}
