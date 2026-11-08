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
package asyncgo
