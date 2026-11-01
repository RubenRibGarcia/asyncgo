package parameters

import (
	"github.com/RubenRibGarcia/asyncgo"
	"github.com/RubenRibGarcia/asyncgo/spec"
)

// userChannel is a dynamic channel: the address carries a Channel Address
// Expression, and the matching Parameter Object declares what the expression
// may hold. 3.1.0 has no `schema` on a parameter — enum, default, and examples
// are how a parameter describes its values — so all five fields are exercised
// here.
var userChannel = asyncgo.Channel("users.{userId}").
	Description("Per-user event stream").
	Parameter("userId", spec.Parameter{
		Enum:        []string{"u_1", "u_2"},
		Default:     "u_1",
		Description: "Id of the user.",
		Examples:    []string{"u_1", "u_2"},
		Location:    "$message.payload#/userId",
	}).
	Send(asyncgo.Operation().
		Message(asyncgo.MessageOf(UserUpdated{}).Name("UserUpdated")))

// Catalog is the AsyncAPI description of a service whose channel is addressed
// per user. The asyncgo CLI discovers it and generates asyncapi.yaml from it.
var Catalog = asyncgo.Spec(
	asyncgo.Info("Parameterized Users Service", "1.0.0").
		Description("Per-user events addressed through a channel parameter"),

	asyncgo.Channels(userChannel),
)
