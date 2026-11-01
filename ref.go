package asyncgo

import (
	"fmt"
	"reflect"

	"github.com/RubenRibGarcia/asyncgo/internal/jsonpointer"
	"github.com/RubenRibGarcia/asyncgo/spec"
)

// securitySchemeRef is the JSON Reference to a scheme declared via
// SecurityScheme(...).
func securitySchemeRef(s *securityScheme) *spec.Reference {
	return &spec.Reference{
		Ref: "#/components/securitySchemes/" + jsonpointer.Escape(s.name),
	}
}

// operationTraitRef is the JSON Reference to an operation trait declared via
// OperationTraits(...).
func operationTraitRef(t *operationTrait) *spec.Reference {
	return &spec.Reference{
		Ref: "#/components/operationTraits/" + jsonpointer.Escape(t.name),
	}
}

// messageTraitRef is the JSON Reference to a message trait declared via
// MessageTraits(...).
func messageTraitRef(t *messageTrait) *spec.Reference {
	return &spec.Reference{Ref: "#/components/messageTraits/" + jsonpointer.Escape(t.name)}
}

// correlationIDRef is the JSON Reference to a correlation id declared via
// CorrelationIDs(...).
func correlationIDRef(c *correlationID) *spec.Reference {
	return &spec.Reference{Ref: "#/components/correlationIds/" + jsonpointer.Escape(c.name)}
}

// correlationIDFromRef hoists an inline correlation id authored with
// Message.CorrelationIDFrom / MessageTrait.CorrelationIDFrom. The component key
// is derived from the owning object's name — <owner>CorrelationID — so the id is
// emitted as a reusable $ref. An existing component with the same value is
// reused; a different value at the same key is a duplicate-name error.
func correlationIDFromRef(b *builder, owner string, c spec.CorrelationID) (*spec.Reference, error) {
	name := owner + "CorrelationID"
	if c.Location == "" {
		return nil, fmt.Errorf("correlationId.%s.location: is required", name)
	}
	comps := b.components()
	if comps.CorrelationIDs == nil {
		comps.CorrelationIDs = map[string]*spec.CorrelationID{}
	}
	if existing, ok := comps.CorrelationIDs[name]; ok {
		if *existing != c {
			return nil, fmt.Errorf("correlationId.%s: duplicate name", name)
		}
	} else {
		value := c
		comps.CorrelationIDs[name] = &value
	}
	return &spec.Reference{
		Ref: "#/components/correlationIds/" + jsonpointer.Escape(name),
	}, nil
}

// replyRef is the JSON Reference to a reply declared via Replies(...).
func replyRef(r *reply) *spec.Reference {
	return &spec.Reference{Ref: "#/components/replies/" + jsonpointer.Escape(r.name)}
}

// replyAddressRef is the JSON Reference to a reply address declared via
// ReplyAddresses(...).
func replyAddressRef(a *replyAddress) *spec.Reference {
	return &spec.Reference{Ref: "#/components/replyAddresses/" + jsonpointer.Escape(a.name)}
}

// channelRef is the JSON Reference to a channel declared via Channels(...).
func channelRef(ch *channel) *spec.Reference {
	return &spec.Reference{Ref: "#/channels/" + jsonpointer.Escape(ch.address)}
}

// channelMessageRef is the JSON Reference to a message carried by ch. It
// mirrors the refs channel.apply builds for an operation's own messages, so the
// two agree on the message's key in the channel.
func channelMessageRef(ch *channel, m *message) *spec.Reference {
	return &spec.Reference{
		Ref: "#/channels/" + jsonpointer.Escape(ch.address) + "/messages/" + jsonpointer.Escape(
			messageKey(m),
		),
	}
}

// hoistMessage registers msg in components.messages under key — the message's
// identity, as messageKey derives it — and returns the Reference Object a
// channel's messages map carries in its place. The same message may be carried
// by several channels, and an identical one may legitimately be declared twice,
// so an existing entry under the same key is reused when it is identical and
// rejected as a duplicate when it is not. owner names the declaring channel in
// that error.
//
// The returned entry is a Reference Object, whose Ref is the only field set: a
// channel messages entry may be a Message Object or a Reference Object, and the
// generator always hoists.
func hoistMessage(
	b *builder,
	key string,
	owner string,
	msg *spec.Message,
) (*spec.Message, error) {
	c := b.components()
	if c.Messages == nil {
		c.Messages = map[string]*spec.Message{}
	}
	if existing, ok := c.Messages[key]; ok {
		if !reflect.DeepEqual(existing, msg) {
			return nil, fmt.Errorf(
				"message.%s: duplicate name with different content (%s)",
				key,
				owner,
			)
		}
	} else {
		c.Messages[key] = msg
	}
	return &spec.Message{Ref: "#/components/messages/" + jsonpointer.Escape(key)}, nil
}
