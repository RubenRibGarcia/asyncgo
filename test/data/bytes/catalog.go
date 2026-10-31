package bytes

import "github.com/RubenRibGarcia/asyncgo"

// attachmentUploaded carries the binary payload the fixture is named for.
var attachmentUploaded = asyncgo.MessageOf(Attachment{})

// Catalog is the AsyncAPI description of a service publishing binary
// attachments. The asyncgo CLI discovers it and generates asyncapi.yaml from it.
var Catalog = asyncgo.Spec(
	asyncgo.Info("Binary Payloads", "1.0.0").
		Description("A payload whose byte slices derive to base64-encoded strings"),

	asyncgo.Channels(
		asyncgo.Channel("attachment-uploaded").
			Description("Emitted when an attachment is uploaded").
			Send(asyncgo.Operation().Message(attachmentUploaded)),
	),
)
