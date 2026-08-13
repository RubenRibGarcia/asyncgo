package discovery

import (
	"path/filepath"
	"testing"
)

func TestFindMaterializeMerge(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("..", "..", "examples", "orders"))
	if err != nil {
		t.Fatal(err)
	}

	cats, err := Find(dir)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(cats) != 1 {
		t.Fatalf("expected 1 catalog, got %d: %+v", len(cats), cats)
	}
	if cats[0].PkgPath != "example.com/orders/orders" || cats[0].VarName != "Catalog" {
		t.Fatalf("unexpected catalog: %+v", cats[0])
	}

	docs, err := Materialize(dir, cats)
	if err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 document, got %d", len(docs))
	}

	merged := Merge(docs...)
	if merged.Info.Title != "Orders Service" {
		t.Errorf("unexpected info title %q", merged.Info.Title)
	}
	ch := merged.Channels["order-placed"]
	if ch == nil {
		t.Fatalf("expected channel order-placed")
	}
	if _, ok := ch.Messages["OrderPlaced"]; !ok {
		t.Errorf("expected message OrderPlaced, got %v", ch.Messages)
	}

	// Schema is hoisted under the fully-qualified type name.
	key := "example.com/orders/orders.OrderPlaced"
	sch, ok := merged.Components.Schemas[key]
	if !ok {
		t.Fatalf("expected schema %q, got %v", key, merged.Components.Schemas)
	}
	if len(sch.Required) != 2 {
		t.Errorf("expected 2 required fields, got %v", sch.Required)
	}
}
