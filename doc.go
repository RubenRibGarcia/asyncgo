// Package asyncgo is the developer-facing fluent DSL for declaring an AsyncAPI
// document. A catalog is a package-level variable of type *SpecResult built
// with Spec(...); the asyncgo CLI discovers such variables reachable from main
// and statically interprets them into an AsyncAPI document.
//
// Most setters take a pointer to another builder (a *reply, a *channel, a
// *securityScheme, ...) and turn it into a JSON Reference immediately, so a nil
// argument panics at the call site — it is not deferred into a validation error.
// Declare the referenced item first and pass its builder; Operation.Reply(nil) is
// the one exception, meaning "this operation has no reply".
//
// Plain setters — Title, Summary, Description, ContentType, ProtocolVersion and
// their kin — only assign the specification field of the same name, so their
// meaning is that field's meaning in the AsyncAPI specification and they carry
// no comment of their own. A setter that does anything else is documented where
// it is declared: one that turns a builder into a JSON Reference (Security,
// Traits, Servers, CorrelationID), one that registers a component
// (CorrelationIDFrom, PayloadExamples), or one whose argument is not the whole
// story (Message.Name, which is also the message's component key).
package asyncgo
