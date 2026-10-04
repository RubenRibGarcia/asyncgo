package reply

import (
	"github.com/RubenRibGarcia/asyncgo"
)

var (
	orderPlaced   = asyncgo.MessageOf(OrderPlaced{}).Name("OrderPlaced")
	orderAccepted = asyncgo.MessageOf(OrderAccepted{}).Name("OrderAccepted")
)

// replyTo is the runtime expression locating where the reply is sent: the
// replyTo header of the request message.
var replyTo = asyncgo.ReplyAddress("ReplyTo").
	Description("Consumer inbox").
	Location("$message.header#/replyTo")

// orderReply answers the request at the replyTo address, and deliberately names
// no channel: the specification requires the channel of an addressed reply to
// have no address of its own, and a channel without an address is not yet
// expressible through the DSL.
var orderReply = asyncgo.Reply("OrderReply").Address(replyTo)

// acceptedReply answers on a dedicated reply channel instead of at an address,
// so the reply channel and reply message refs are exercised too.
var acceptedReply = asyncgo.Reply("OrderAcceptedReply").
	Channel(replyChannel).
	Message(replyChannel, orderAccepted)

// requestChannel carries the order request; its operation declares the reply.
var requestChannel = asyncgo.Channel("order-placed").
	Description("Order requests").
	Send(asyncgo.Operation().
		Message(orderPlaced).
		Reply(orderReply))

// replyChannel carries the acceptance. It is declared with its own operation
// because a channel only acquires messages through an operation, and
// reply.messages has to point at a message defined on the reply channel.
var replyChannel = asyncgo.Channel("order-replies").
	Description("Order replies").
	Receive(asyncgo.Operation().Message(orderAccepted))

// Catalog is the AsyncAPI description of an order service that answers requests
// with a reply. The asyncgo CLI discovers it and generates asyncapi.yaml from it.
var Catalog = asyncgo.Spec(
	asyncgo.Info("Request/Reply Orders Service", "1.0.0").
		Description("Order requests answered on a reply address"),

	asyncgo.ReplyAddresses(replyTo),

	asyncgo.Replies(orderReply, acceptedReply),

	asyncgo.Channels(requestChannel, replyChannel),
)
