package spec

import "testing"

func TestOverlay(t *testing.T) {
	base := New()
	base.Info = Info{Title: "Base", Version: "1.0.0", Description: "base desc"}
	base.Servers = map[string]*Server{
		"prod": {Host: "h", Protocol: ProtocolKafka},
	}
	base.Channels = map[string]*Channel{
		"a": {Address: "a", Description: "base channel"},
	}

	overlay := New()
	overlay.Info = Info{Title: "Overlay"} // only title set
	overlay.Servers = map[string]*Server{
		"prod": {Description: "from fragment"},
		"dev":  {Host: "localhost", Protocol: ProtocolNATS},
	}
	overlay.Channels = map[string]*Channel{
		"a": {Description: "overlay channel"},
		"b": {Address: "b"},
	}

	merged, err := Overlay(base, overlay)
	if err != nil {
		t.Fatal(err)
	}

	// Overlay wins on scalar; base preserved elsewhere.
	if merged.AsyncAPI != Version {
		t.Errorf("expected version %q preserved, got %q", Version, merged.AsyncAPI)
	}
	if merged.Info.Title != "Overlay" {
		t.Errorf("expected title Overlay, got %q", merged.Info.Title)
	}
	if merged.Info.Version != "1.0.0" {
		t.Errorf("expected version preserved, got %q", merged.Info.Version)
	}
	if merged.Info.Description != "base desc" {
		t.Errorf("expected description preserved, got %q", merged.Info.Description)
	}

	// Servers: prod deep-merged (host/protocol kept, description overlaid), dev added.
	prod := merged.Servers["prod"]
	if prod == nil {
		t.Fatal("expected prod server")
	}
	if prod.Host != "h" || prod.Protocol != ProtocolKafka {
		t.Errorf("expected prod host/protocol preserved, got %+v", prod)
	}
	if prod.Description != "from fragment" {
		t.Errorf("expected prod description overlaid, got %q", prod.Description)
	}
	if dev := merged.Servers["dev"]; dev == nil || dev.Protocol != ProtocolNATS {
		t.Errorf("expected dev server added, got %+v", dev)
	}

	// Channels: a deep-merged, b added.
	if merged.Channels["a"].Description != "overlay channel" {
		t.Errorf("expected channel a description overlaid, got %q", merged.Channels["a"].Description)
	}
	if merged.Channels["a"].Address != "a" {
		t.Errorf("expected channel a address preserved, got %q", merged.Channels["a"].Address)
	}
	if merged.Channels["b"] == nil || merged.Channels["b"].Address != "b" {
		t.Errorf("expected channel b added, got %+v", merged.Channels["b"])
	}
}

func TestOverlayDoesNotMutateBase(t *testing.T) {
	base := New()
	base.Info = Info{Title: "Base", Version: "1.0.0"}
	overlay := New()
	overlay.Info = Info{Title: "Overlay"}

	if _, err := Overlay(base, overlay); err != nil {
		t.Fatal(err)
	}
	if base.Info.Title != "Base" {
		t.Errorf("Overlay mutated base: title is %q", base.Info.Title)
	}
}
