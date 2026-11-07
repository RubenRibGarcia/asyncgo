package asyncgo

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/RubenRibGarcia/asyncgo/internal/jsonpointer"
	"github.com/RubenRibGarcia/asyncgo/spec"
)

// validateServerRefs checks that every channel server reference points at a
// server declared via Servers(...). It is a post-pass: it runs after all items
// are applied because declaration order is arbitrary (a channel may be applied
// before the server it references).
func (b *builder) validateServerRefs() []error {
	var errs []error
	// Channels are walked in sorted order so the joined error message is stable:
	// SpecResult.Err is compared by exact string in tests.
	for _, addr := range slices.Sorted(maps.Keys(b.doc.Channels)) {
		ch := b.doc.Channels[addr]
		if ch == nil {
			// A hand-assembled document may hold a nil entry; skip it rather than
			// dereferencing it, like every other validator here.
			continue
		}
		for _, ref := range ch.Servers {
			name := refName(ref.Ref, serversRefPrefix)
			if _, ok := b.doc.Servers[name]; !ok {
				errs = append(
					errs,
					fmt.Errorf("channel.%s: references unknown server %q", addr, name),
				)
			}
		}
	}
	return errs
}

// validateSecurityRefs checks that every security reference on a server, an
// operation, or an operation trait points at a scheme declared via
// SecuritySchemes(...). It is a post-pass for the same reason as
// validateServerRefs: declaration order is arbitrary.
func (b *builder) validateSecurityRefs() []error {
	var declared map[string]*spec.SecurityScheme
	var declaredTraits map[string]*spec.OperationTrait
	if b.doc.Components != nil {
		declared = b.doc.Components.SecuritySchemes
		declaredTraits = b.doc.Components.OperationTraits
	}

	var errs []error
	// Servers, operations, and operation traits are walked in sorted order so the
	// joined error message is stable: SpecResult.Err is compared by exact string
	// in tests.
	for _, name := range slices.Sorted(maps.Keys(b.doc.Servers)) {
		srv := b.doc.Servers[name]
		if srv == nil {
			continue // see validateServerRefs: nil entries are skipped, not dereferenced
		}
		for _, ref := range srv.Security {
			scheme := refName(ref.Ref, securitySchemesRefPrefix)
			if _, ok := declared[scheme]; !ok {
				errs = append(
					errs,
					fmt.Errorf("server.%s: references unknown security scheme %q", name, scheme),
				)
			}
		}
	}
	for _, key := range slices.Sorted(maps.Keys(b.doc.Operations)) {
		op := b.doc.Operations[key]
		if op == nil {
			continue
		}
		for _, ref := range op.Security {
			scheme := refName(ref.Ref, securitySchemesRefPrefix)
			if _, ok := declared[scheme]; !ok {
				errs = append(
					errs,
					fmt.Errorf("operation.%s: references unknown security scheme %q", key, scheme),
				)
			}
		}
	}
	for _, name := range slices.Sorted(maps.Keys(declaredTraits)) {
		trait := declaredTraits[name]
		if trait == nil {
			continue
		}
		for _, ref := range trait.Security {
			scheme := refName(ref.Ref, securitySchemesRefPrefix)
			if _, ok := declared[scheme]; !ok {
				errs = append(errs, fmt.Errorf(
					"operationTrait.%s: references unknown security scheme %q",
					name,
					scheme,
				))
			}
		}
	}
	return errs
}

// validateReplyRefs checks that every operation reply reference points at a
// reply declared via Replies(...), that every declared reply's address and
// channel resolve, that a reply never combines an address with a channel, and
// that every message a reply names is one the reply's channel actually carries.
// It is a post-pass for the same reason as validateServerRefs: declaration order
// is arbitrary, so a reply may be referenced before it is declared.
//
// The address/channel rule is a specification MUST that no downstream validator
// can catch — `asyncapi validate` accepts a document that breaks it, because the
// constraint relates two objects and a JSON Schema cannot express that — so this
// is the only place it is enforced.
func (b *builder) validateReplyRefs() []error {
	var declaredReplies map[string]*spec.OperationReply
	var declaredAddresses map[string]*spec.OperationReplyAddress
	if b.doc.Components != nil {
		declaredReplies = b.doc.Components.Replies
		declaredAddresses = b.doc.Components.ReplyAddresses
	}

	var errs []error

	// Keys are walked in sorted order so the joined error message is stable:
	// SpecResult.Err is compared by exact string in tests.
	for _, key := range slices.Sorted(maps.Keys(b.doc.Operations)) {
		op := b.doc.Operations[key]
		if op == nil {
			continue // see validateServerRefs: nil entries are skipped, not dereferenced
		}
		ref := op.Reply
		if ref == nil {
			continue
		}
		name := refName(ref.Ref, repliesRefPrefix)
		if _, ok := declaredReplies[name]; !ok {
			errs = append(errs, fmt.Errorf("operation.%s: references unknown reply %q", key, name))
		}
	}

	for _, name := range slices.Sorted(maps.Keys(declaredReplies)) {
		r := declaredReplies[name]
		if r == nil {
			continue
		}
		if r.Address != nil {
			addr := refName(r.Address.Ref, replyAddressesRefPrefix)
			if _, ok := declaredAddresses[addr]; !ok {
				errs = append(
					errs,
					fmt.Errorf("reply.%s: references unknown reply address %q", name, addr),
				)
			}
		}
		if r.Address != nil && r.Channel != nil {
			errs = append(errs, fmt.Errorf(
				"reply.%s: address and channel are mutually exclusive (the referenced channel must have no address)",
				name,
			))
		}
		if len(r.Messages) > 0 && r.Channel == nil {
			errs = append(errs, fmt.Errorf("reply.%s: messages require a channel", name))
		}
		if r.Channel == nil {
			continue
		}
		channel := refName(r.Channel.Ref, channelsRefPrefix)
		ch, ok := b.doc.Channels[channel]
		if !ok || ch == nil {
			errs = append(
				errs,
				fmt.Errorf("reply.%s: references unknown channel %q", name, channel),
			)
			continue
		}
		errs = append(errs, channelMessageErrors("reply."+name, channel, ch, r.Messages)...)
	}

	return errs
}

// validateMessageRefs checks that every channel messages entry that is a
// Reference Object points at a message declared in components.messages. It is a
// post-pass for the same reason as validateServerRefs: declaration order is
// arbitrary. The generator hoists and references a message in one step, so it
// cannot produce a dangling reference; the check guards a document assembled
// through the spec package directly.
func (b *builder) validateMessageRefs() []error {
	var declared map[string]*spec.Message
	if b.doc.Components != nil {
		declared = b.doc.Components.Messages
	}

	var errs []error
	// Channels and their messages are walked in sorted order so the joined error
	// message is stable: SpecResult.Err is compared by exact string in tests.
	for _, address := range slices.Sorted(maps.Keys(b.doc.Channels)) {
		ch := b.doc.Channels[address]
		if ch == nil {
			continue
		}
		for _, key := range slices.Sorted(maps.Keys(ch.Messages)) {
			msg := ch.Messages[key]
			if msg == nil || msg.Ref == "" {
				continue
			}
			name := refName(msg.Ref, messagesRefPrefix)
			if _, ok := declared[name]; !ok {
				errs = append(errs, fmt.Errorf(
					"channel.%s.messages.%s: references unknown message %q",
					address,
					key,
					name,
				))
			}
		}
	}

	return errs
}

// validateOperationMessages checks that every message an operation references is
// one the operation's channel actually carries: the specification requires
// operation.messages to be a subset of channel.messages. It is a post-pass for
// the same reason as validateServerRefs, and it enforces a MUST no downstream
// validator can catch — the constraint relates two objects, which a JSON Schema
// cannot express.
func (b *builder) validateOperationMessages() []error {
	var errs []error
	// Keys are walked in sorted order so the joined error message is stable:
	// SpecResult.Err is compared by exact string in tests.
	for _, key := range slices.Sorted(maps.Keys(b.doc.Operations)) {
		op := b.doc.Operations[key]
		if op == nil || op.Channel == nil {
			continue
		}
		address := refName(op.Channel.Ref, channelsRefPrefix)
		ch := b.doc.Channels[address]
		if ch == nil {
			// Out of scope: channel.apply always points an operation at the
			// channel it registers, so a dangling channel reference cannot arise
			// from the DSL.
			continue
		}
		errs = append(errs, channelMessageErrors("operation."+key, address, ch, op.Messages)...)
	}

	return errs
}

// channelMessageErrors reports every message reference that is not one ch carries.
// owner names the referring object in the error, e.g. "reply.OrderReply".
//
// Both callers enforce the same specification MUST — an operation's or a reply's
// messages must be a subset of the channel's — so the membership test is written
// once. A reference counts as carried only when its own $ref names ch: a message
// of the same key reached through another channel is a miss.
func channelMessageErrors(
	owner string,
	channel string,
	ch *spec.Channel,
	refs []*spec.Reference,
) []error {
	prefix := channelsRefPrefix + jsonpointer.Escape(channel) + channelMessagesSegment
	var errs []error
	for _, ref := range refs {
		if ref == nil {
			continue // see validateServerRefs: nil entries are skipped, not dereferenced
		}
		if strings.HasPrefix(ref.Ref, prefix) {
			if _, carried := ch.Messages[refName(ref.Ref, prefix)]; carried {
				continue
			}
		}
		errs = append(errs, fmt.Errorf(
			"%s: message %q is not in channel %q",
			owner,
			ref.Ref,
			channel,
		))
	}
	return errs
}

// validateTraitRefs checks that every operation and message trait reference
// points at a trait declared via OperationTraits(...) / MessageTraits(...), and
// that every message trait's correlation id points at a component declared via
// CorrelationIDs(...). It is a post-pass for the same reason as
// validateServerRefs: declaration order is arbitrary, so a trait may be
// referenced before it is declared.
func (b *builder) validateTraitRefs() []error {
	var declaredOperationTraits map[string]*spec.OperationTrait
	var declaredMessageTraits map[string]*spec.MessageTrait
	var declaredCorrelationIDs map[string]*spec.CorrelationID
	var declaredMessages map[string]*spec.Message
	if b.doc.Components != nil {
		declaredOperationTraits = b.doc.Components.OperationTraits
		declaredMessageTraits = b.doc.Components.MessageTraits
		declaredCorrelationIDs = b.doc.Components.CorrelationIDs
		declaredMessages = b.doc.Components.Messages
	}

	var errs []error

	// Keys are walked in sorted order so the joined error message is stable:
	// SpecResult.Err is compared by exact string in tests.
	for _, key := range slices.Sorted(maps.Keys(b.doc.Operations)) {
		op := b.doc.Operations[key]
		if op == nil {
			continue // see validateServerRefs: nil entries are skipped, not dereferenced
		}
		for _, ref := range op.Traits {
			name := refName(ref.Ref, operationTraitsRefPrefix)
			if _, ok := declaredOperationTraits[name]; !ok {
				errs = append(errs, fmt.Errorf(
					"operation.%s: references unknown operation trait %q",
					key,
					name,
				))
			}
		}
	}

	for _, key := range slices.Sorted(maps.Keys(declaredMessages)) {
		msg := declaredMessages[key]
		if msg == nil {
			continue
		}
		for _, ref := range msg.Traits {
			trait := refName(ref.Ref, messageTraitsRefPrefix)
			if _, ok := declaredMessageTraits[trait]; !ok {
				errs = append(errs, fmt.Errorf(
					"message.%s: references unknown message trait %q",
					key,
					trait,
				))
			}
		}
		if ref := msg.CorrelationID; ref != nil {
			id := refName(ref.Ref, correlationIDsRefPrefix)
			if _, ok := declaredCorrelationIDs[id]; !ok {
				errs = append(errs, fmt.Errorf(
					"message.%s: references unknown correlation id %q",
					key,
					id,
				))
			}
		}
	}

	for _, name := range slices.Sorted(maps.Keys(declaredMessageTraits)) {
		trait := declaredMessageTraits[name]
		if trait == nil {
			continue
		}
		ref := trait.CorrelationID
		if ref == nil {
			continue
		}
		id := refName(ref.Ref, correlationIDsRefPrefix)
		if _, ok := declaredCorrelationIDs[id]; !ok {
			errs = append(errs, fmt.Errorf(
				"messageTrait.%s: references unknown correlation id %q",
				name,
				id,
			))
		}
	}

	return errs
}

// validateSchemaNodes checks every schema that reaches the document for the Multi
// Format Schema Object invariants: schemaFormat and schema are set together, a
// multi-format node carries no JSON Schema keyword, and a declared
// components.schemas name does not collide with an auto-hoisted schema. It runs
// before Spec merges the hoisted definitions, because that merge is what a
// collision would silently corrupt.
func (b *builder) validateSchemaNodes() []error {
	var errs []error

	if comps := b.doc.Components; comps != nil {
		for _, name := range slices.Sorted(maps.Keys(comps.Schemas)) {
			if _, hoisted := b.defs[name]; hoisted {
				errs = append(errs, fmt.Errorf(
					"schema.%s: collides with an auto-hoisted schema of the same name",
					name,
				))
			}
			errs = append(errs, schemaNodeErrors(comps.Schemas[name], "schema."+name)...)
		}
		for _, name := range slices.Sorted(maps.Keys(comps.MessageTraits)) {
			if tr := comps.MessageTraits[name]; tr != nil {
				errs = append(
					errs,
					schemaNodeErrors(tr.Headers, "messageTrait."+name+".headers")...,
				)
			}
		}
		// Messages are hoisted into components.messages and a channel's messages
		// map holds only references to them (validateMessageRefs checks that the
		// references resolve), so every payload and header that can reach the
		// document is walked here.
		for _, key := range slices.Sorted(maps.Keys(comps.Messages)) {
			msg := comps.Messages[key]
			if msg == nil {
				continue
			}
			base := "message." + key
			errs = append(errs, schemaNodeErrors(msg.Payload, base+".payload")...)
			errs = append(errs, schemaNodeErrors(msg.Headers, base+".headers")...)
		}
	}

	return errs
}

// schemaNodeErrors reports the Multi Format Schema Object invariants violated by s
// or by any schema nested inside it. path names the node in the document and is
// extended with the child keyword on the way down.
func schemaNodeErrors(s *spec.Schema, path string) []error {
	if s == nil {
		return nil
	}

	var errs []error
	if s.SchemaFormat != "" || s.Schema != nil {
		if s.SchemaFormat == "" {
			errs = append(
				errs,
				fmt.Errorf("%s.schemaFormat: is required alongside schema", path),
			)
		}
		if s.Schema == nil {
			errs = append(
				errs,
				fmt.Errorf("%s.schema: is required alongside schemaFormat", path),
			)
		}
		if kw := jsonSchemaKeywords(s); len(kw) > 0 {
			errs = append(errs, fmt.Errorf(
				"%s: multi format schema must not carry JSON Schema keyword(s): %s",
				path,
				strings.Join(kw, ", "),
			))
		}
	}

	for _, child := range schemaChildren(s) {
		errs = append(errs, schemaNodeErrors(child.schema, path+"."+child.name)...)
	}
	return errs
}

// schemaChild is a nested schema together with the path segment naming it.
type schemaChild struct {
	name   string
	schema *spec.Schema
}

// schemaChildren returns the nested schemas of s in a deterministic order.
//
// Every schema-valued keyword is walked, not just the combinators: an invalid
// Multi Format node nested under one of them must still be reported by
// schemaNodeErrors. jsonSchemaKeywords needs no matching edit — it reflects over
// the struct — and that asymmetry is exactly what this list has to cover, so a
// new *Schema field belongs here too.
func schemaChildren(s *spec.Schema) []schemaChild {
	var children []schemaChild
	for _, name := range slices.Sorted(maps.Keys(s.Properties)) {
		children = append(children, schemaChild{"properties." + name, s.Properties[name]})
	}
	for _, name := range slices.Sorted(maps.Keys(s.Definitions)) {
		children = append(children, schemaChild{"definitions." + name, s.Definitions[name]})
	}
	for i, m := range s.AllOf {
		children = append(children, schemaChild{fmt.Sprintf("allOf.%d", i), m})
	}
	for i, m := range s.OneOf {
		children = append(children, schemaChild{fmt.Sprintf("oneOf.%d", i), m})
	}
	for i, m := range s.AnyOf {
		children = append(children, schemaChild{fmt.Sprintf("anyOf.%d", i), m})
	}
	if s.Items != nil {
		children = append(children, schemaChild{"items", s.Items})
	}
	if s.AdditionalProperties != nil {
		children = append(children, schemaChild{"additionalProperties", s.AdditionalProperties})
	}
	if s.Not != nil {
		children = append(children, schemaChild{"not", s.Not})
	}
	for _, name := range slices.Sorted(maps.Keys(s.PatternProperties)) {
		children = append(
			children,
			schemaChild{"patternProperties." + name, s.PatternProperties[name]},
		)
	}
	for _, name := range slices.Sorted(maps.Keys(s.Dependencies)) {
		children = append(children, schemaChild{"dependencies." + name, s.Dependencies[name]})
	}
	for _, kw := range []struct {
		name string
		node *spec.Schema
	}{
		{"additionalItems", s.AdditionalItems},
		{"contains", s.Contains},
		{"propertyNames", s.PropertyNames},
		{"if", s.If},
		{"then", s.Then},
		{"else", s.Else},
	} {
		if kw.node != nil {
			children = append(children, schemaChild{kw.name, kw.node})
		}
	}
	return children
}

// multiFormatFields are the Schema fields that make a node a Multi Format Schema
// Object; every other field is a JSON Schema keyword.
var multiFormatFields = map[string]bool{"SchemaFormat": true, "Schema": true}

// jsonSchemaKeywords returns the JSON names of the JSON Schema keyword fields set
// on s. It reflects over the struct rather than listing the fields, so a keyword
// added later is covered without touching this check.
func jsonSchemaKeywords(s *spec.Schema) []string {
	v := reflect.ValueOf(*s)
	t := v.Type()
	var set []string
	for i := range t.NumField() {
		f := t.Field(i)
		if multiFormatFields[f.Name] || v.Field(i).IsZero() {
			continue
		}
		name := f.Name
		if before, _, found := strings.Cut(f.Tag.Get("json"), ","); found {
			name = before
		}
		set = append(set, name)
	}
	return set
}
