package asyncgo

import (
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"

	"github.com/RubenRibGarcia/asyncgo/internal/jsonpointer"
	"github.com/RubenRibGarcia/asyncgo/spec"
)

type channelsItem []*channel

// Channels adds one or more channels to the document.
func Channels(c ...*channel) Item { return channelsItem(c) }

func (c channelsItem) apply(b *builder) error {
	if b.doc.Channels == nil {
		b.doc.Channels = map[string]*spec.Channel{}
	}
	var errs []error
	for _, ch := range c {
		if err := ch.apply(b); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

type channel struct {
	address string
	s       spec.Channel
	ops     []*operation
}

// Channel declares a channel addressed by a topic/queue/subject.
func Channel(address string) *channel {
	return &channel{address: address, s: spec.Channel{Address: address}}
}

func (c *channel) Title(t string) *channel       { c.s.Title = t; return c }
func (c *channel) Description(d string) *channel { c.s.Description = d; return c }

// Tags attaches tags for logical grouping and categorization of the channel.
func (c *channel) Tags(tags ...spec.Tag) *channel {
	c.s.Tags = append(c.s.Tags, tags...)
	return c
}

// ExternalDocs attaches additional external documentation for the channel.
func (c *channel) ExternalDocs(d spec.ExternalDocs) *channel {
	c.s.ExternalDocs = &d
	return c
}

// Servers references the servers (declared via Servers(...)) on which this
// channel is available. If empty, the channel is available on all servers.
func (c *channel) Servers(s ...*server) *channel {
	for _, sv := range s {
		c.s.Servers = append(
			c.s.Servers,
			&spec.Reference{Ref: "#/servers/" + jsonpointer.Escape(sv.name)},
		)
	}
	return c
}

// Parameter declares a channel parameter, which the address references as
// {name}. A parameter is a string substitution into the address, so the values
// it admits are declared through spec.Parameter's Enum, Default, and Examples —
// not through a schema, which 3.1.0 removed from the Parameter Object. Every
// Channel Address Expression in the address must have a corresponding
// parameter, and every declared parameter must appear in the address.
func (c *channel) Parameter(name string, p spec.Parameter) *channel {
	if c.s.Parameters == nil {
		c.s.Parameters = map[string]*spec.Parameter{}
	}
	c.s.Parameters[name] = &p
	return c
}

// channelAddressExpression matches a Channel Address Expression, {name}, inside
// a channel address.
var channelAddressExpression = regexp.MustCompile(`\{([^{}]+)\}`)

// validateParameters checks a channel's address expressions against the
// parameters it declares. A channel with neither is silent, so an ordinary
// address costs nothing.
func (c *channel) validateParameters() []error {
	if c.address == "" {
		return nil
	}

	var errs []error
	used := map[string]bool{}
	for _, expr := range channelAddressExpression.FindAllStringSubmatch(c.address, -1) {
		name := expr[1]
		if _, ok := c.s.Parameters[name]; !ok {
			errs = append(errs, fmt.Errorf(
				"channel.%s: address expression {%s} has no declared parameter",
				c.address,
				name,
			))
			continue
		}
		used[name] = true
	}
	// Parameters are walked in sorted order so the joined error message is
	// stable: SpecResult.Err is compared by exact string in tests.
	for _, name := range slices.Sorted(maps.Keys(c.s.Parameters)) {
		if !used[name] {
			errs = append(errs, fmt.Errorf(
				"channel.%s: parameter %q is not used in the address",
				c.address,
				name,
			))
		}
	}
	return errs
}

// Send attaches a send operation to the channel.
func (c *channel) Send(op *operation) *channel {
	op.action = spec.ActionSend
	c.ops = append(c.ops, op)
	return c
}

// Receive attaches a receive operation to the channel.
func (c *channel) Receive(op *operation) *channel {
	op.action = spec.ActionReceive
	c.ops = append(c.ops, op)
	return c
}

func (c *channel) apply(b *builder) error {
	var errs []error
	if c.address == "" {
		errs = append(errs, fmt.Errorf("channel.address: is required"))
	}
	errs = append(errs, c.validateParameters()...)
	if b.doc.Channels == nil {
		b.doc.Channels = map[string]*spec.Channel{}
	}
	if b.doc.Operations == nil {
		b.doc.Operations = map[string]*spec.Operation{}
	}

	ch := &c.s
	if _, dup := b.doc.Channels[c.address]; dup && c.address != "" {
		errs = append(errs, fmt.Errorf("channel.%s: duplicate address", c.address))
	} else {
		b.doc.Channels[c.address] = ch
	}

	for _, op := range c.ops {
		specOp := &spec.Operation{
			Action:       op.action,
			Channel:      &spec.Reference{Ref: "#/channels/" + jsonpointer.Escape(c.address)},
			Title:        op.title,
			Summary:      op.summary,
			Description:  op.description,
			Tags:         op.tags,
			ExternalDocs: op.externalDocs,
			Security:     op.security,
			Bindings:     op.bindings,
			Traits:       op.traits,
			Reply:        op.replyReference(),
		}
		for _, m := range op.messages {
			sm, err := m.build(b)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			key := messageKey(m)
			// The key becomes the message's components.messages key, which the
			// specification patterns. A key derived from the payload type's fully
			// qualified name is exempt: appendKeyError skips an unauthored name.
			errs = appendKeyError(errs, "message", m.name)
			entry, err := hoistMessage(b, key, "channel."+c.address, sm)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			if ch.Messages == nil {
				ch.Messages = map[string]*spec.Message{}
			}
			ch.Messages[key] = entry
			specOp.Messages = append(specOp.Messages, &spec.Reference{
				Ref: "#/channels/" + jsonpointer.Escape(
					c.address,
				) + "/messages/" + jsonpointer.Escape(
					key,
				),
			})
		}
		// An operation's map key is <channel>.<action>, so a second Send/Receive on
		// the same channel would overwrite the first silently. The guard mirrors
		// the sibling duplicate checks: report it and keep the first entry.
		opKey := c.address + "." + op.action
		if _, dup := b.doc.Operations[opKey]; dup && c.address != "" {
			errs = append(errs, fmt.Errorf("operation.%s: duplicate action", opKey))
		} else {
			b.doc.Operations[opKey] = specOp
		}
	}
	return errors.Join(errs...)
}
