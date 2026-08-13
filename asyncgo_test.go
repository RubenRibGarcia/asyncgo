package asyncgo

import (
	"testing"

	"github.com/RubenRibGarcia/asyncgo/spec"
)

type OrderPlaced struct {
	OrderID string  `json:"order_id" asyncgo:"required"`
	Amount  float64 `json:"amount" asyncgo:"required"`
	Note    string  `json:"note"`
}

func TestSpecBuildsDocument(t *testing.T) {
	doc := Spec(
		Info("Orders Service", "1.0.0").Description("Order lifecycle events"),
		DefaultContentType("application/json"),
		Servers(Server("prod", "kafka").Host("broker:9092")),
		Channels(
			Channel("order-placed").
				Description("Emitted when an order is placed").
				Send(Operation().
					Message(MessageOf(OrderPlaced{}).Name("OrderPlaced"))).
				Kafka(spec.KafkaChannelBinding{Topic: "order-placed"}),
		),
	)

	if doc.AsyncAPI != "3.1.0" {
		t.Errorf("expected version 3.1.0, got %q", doc.AsyncAPI)
	}
	if doc.Info.Title != "Orders Service" {
		t.Errorf("unexpected title %q", doc.Info.Title)
	}
	if doc.DefaultContentType != "application/json" {
		t.Errorf("unexpected content type %q", doc.DefaultContentType)
	}
	if s := doc.Servers["prod"]; s == nil || s.Host != "broker:9092" {
		t.Errorf("unexpected server: %+v", doc.Servers["prod"])
	}

	ch := doc.Channels["order-placed"]
	if ch == nil {
		t.Fatal("expected channel order-placed")
	}
	if ch.Address != "order-placed" {
		t.Errorf("unexpected address %q", ch.Address)
	}
	if _, ok := ch.Messages["OrderPlaced"]; !ok {
		t.Errorf("expected message OrderPlaced, got %v", ch.Messages)
	}

	op := doc.Operations["order-placed.send"]
	if op == nil {
		t.Fatal("expected operation order-placed.send")
	}
	if op.Action != spec.ActionSend {
		t.Errorf("expected send, got %q", op.Action)
	}
	if op.Channel == nil || op.Channel.Ref != "#/channels/order-placed" {
		t.Errorf("unexpected channel ref: %+v", op.Channel)
	}

	// Payload is a $ref; schema is hoisted into components under the
	// fully-qualified type name.
	msg := ch.Messages["OrderPlaced"]
	if msg.Payload == nil || msg.Payload.Ref == "" {
		t.Fatalf("expected payload $ref, got %+v", msg.Payload)
	}
	key := "github.com/RubenRibGarcia/asyncgo.OrderPlaced"
	sch, ok := doc.Components.Schemas[key]
	if !ok {
		t.Fatalf("expected schema %q, got %v", key, doc.Components.Schemas)
	}
	if len(sch.Required) != 2 {
		t.Errorf("expected 2 required fields, got %v", sch.Required)
	}

	// Channel binding survived.
	if _, ok := ch.Bindings[spec.ProtocolKafka]; !ok {
		t.Errorf("expected kafka channel binding")
	}
}

func TestMessageOfInfersName(t *testing.T) {
	m := MessageOf(OrderPlaced{})
	got := messageName(m)
	if got != "OrderPlaced" {
		t.Errorf("expected inferred name OrderPlaced, got %q", got)
	}
}
