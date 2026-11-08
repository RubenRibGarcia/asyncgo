package discovery

import (
	"sort"

	"github.com/RubenRibGarcia/asyncgo/spec"
)

// Document pairs a catalog's identity with the document it materialized.
//
// Merge needs the identity and not just the document: a collision has to say
// what was dropped in favour of what, and a document on its own cannot name the
// catalog it came from.
type Document struct {
	Catalog Catalog
	Doc     *spec.AsyncAPI
}

// Collision is a map key that two or more catalogs defined. The first catalog in
// merge order won; Dropped lists the catalogs whose definition did not reach the
// merged document.
type Collision struct {
	Section string // "channels", "components.schemas", "info", ...
	Name    string // the key both catalogs wanted
	Kept    Catalog
	Dropped []Catalog
}

// collisionKey identifies a collision so a third catalog defining the same key
// joins the existing report instead of producing a duplicate one.
type collisionKey struct {
	section string
	name    string
}

// merger accumulates the per-section key owners of the merged document, so a
// first-wins drop can name both sides. It is a value local to one Merge call —
// there is no shared state to race on.
type merger struct {
	owners     map[string]map[string]Catalog // section -> key -> catalog that won
	collisions []Collision
	index      map[collisionKey]int
}

func newMerger() *merger {
	return &merger{
		owners: map[string]map[string]Catalog{},
		index:  map[collisionKey]int{},
	}
}

// mapInto unions src into dst, keeping dst's value on a collision and recording
// who lost. It mirrors the first-wins rule Merge has always had; the difference
// is that the loss is now reportable.
//
// It is a free function rather than a method because Go does not allow type
// parameters on methods, and Merge needs this for a different value type per
// section.
func mapInto[V any](
	m *merger,
	section string,
	dst, src map[string]V,
	owner Catalog,
) map[string]V {
	if dst == nil {
		dst = map[string]V{}
	}
	sectionOwners, ok := m.owners[section]
	if !ok {
		sectionOwners = map[string]Catalog{}
		m.owners[section] = sectionOwners
	}
	for k, v := range src {
		if kept, taken := sectionOwners[k]; taken {
			m.record(section, k, kept, owner)
			continue
		}
		dst[k] = v
		sectionOwners[k] = owner
	}
	return dst
}

// record appends a collision, joining one that already exists for the same key.
func (m *merger) record(section, name string, kept, dropped Catalog) {
	key := collisionKey{section: section, name: name}
	if i, ok := m.index[key]; ok {
		m.collisions[i].Dropped = append(m.collisions[i].Dropped, dropped)
		return
	}
	m.index[key] = len(m.collisions)
	m.collisions = append(m.collisions, Collision{
		Section: section,
		Name:    name,
		Kept:    kept,
		Dropped: []Catalog{dropped},
	})
}

// sorted returns the collisions ordered by section then name. Map iteration is
// random, and a report that reorders between runs cannot be diffed or asserted.
func (m *merger) sorted() []Collision {
	sort.Slice(m.collisions, func(i, j int) bool {
		if m.collisions[i].Section != m.collisions[j].Section {
			return m.collisions[i].Section < m.collisions[j].Section
		}
		return m.collisions[i].Name < m.collisions[j].Name
	})
	return m.collisions
}

// Merge combines catalog documents into a single AsyncAPI document. Maps are
// unioned (first occurrence wins on key collision); Info and DefaultContentType
// are taken from the first document that defines them.
//
// "Defines" is narrower than it sounds for Info: the whole Info struct is taken
// from the first document whose Title is non-empty, and a document without a
// Title contributes nothing — even a Version, a Description or a License. There
// is no field-wise merge, so a catalog that only sets the version is silently
// ignored, and every other catalog's Info is dropped once one document has a
// Title. That is deliberate (a single Info section is the only legal shape for a
// merged document) but it is not obvious from the call site; TestMerge's
// should_take_info_from_the_first_document_with_a_title pins it. Merging
// field-wise — filling gaps in the chosen document from the others — is a
// possible future change, not this one. A losing Info is now reported as a
// collision, so the drop is at least visible.
//
// The second return value is the collisions it resolved, sorted by section then
// name. Recovering them used to be impossible: first-wins discarded the losing
// definition without a trace, which made a merged document quietly wrong for one
// of its catalogs.
func Merge(docs ...Document) (*spec.AsyncAPI, []Collision) {
	out := spec.New()
	m := newMerger()
	var infoOwner Catalog
	for _, d := range docs {
		if d.Doc == nil {
			continue
		}
		switch {
		case d.Doc.Info.Title == "":
			// Contributes nothing to Info; see the note above.
		case out.Info.Title == "":
			out.Info = d.Doc.Info
			infoOwner = d.Catalog
		default:
			m.record("info", d.Doc.Info.Title, infoOwner, d.Catalog)
		}
		if out.DefaultContentType == "" {
			out.DefaultContentType = d.Doc.DefaultContentType
		}
		out.Servers = mapInto(m, "servers", out.Servers, d.Doc.Servers, d.Catalog)
		out.Channels = mapInto(m, "channels", out.Channels, d.Doc.Channels, d.Catalog)
		out.Operations = mapInto(
			m, "operations", out.Operations, d.Doc.Operations, d.Catalog,
		)
		m.components(out, d)
	}
	return out, m.sorted()
}

// components merges every components.* map, each under its own section name so a
// collision report says which one it happened in.
func (m *merger) components(out *spec.AsyncAPI, d Document) {
	if d.Doc.Components == nil {
		return
	}
	if out.Components == nil {
		out.Components = &spec.Components{}
	}
	c := out.Components
	p := d.Doc.Components
	c.Schemas = mapInto(m, "components.schemas", c.Schemas, p.Schemas, d.Catalog)
	c.Servers = mapInto(m, "components.servers", c.Servers, p.Servers, d.Catalog)
	c.Channels = mapInto(m, "components.channels", c.Channels, p.Channels, d.Catalog)
	c.Operations = mapInto(
		m, "components.operations", c.Operations, p.Operations, d.Catalog,
	)
	c.Messages = mapInto(m, "components.messages", c.Messages, p.Messages, d.Catalog)
	c.SecuritySchemes = mapInto(
		m, "components.securitySchemes", c.SecuritySchemes, p.SecuritySchemes, d.Catalog,
	)
	c.Parameters = mapInto(
		m, "components.parameters", c.Parameters, p.Parameters, d.Catalog,
	)
	c.CorrelationIDs = mapInto(
		m, "components.correlationIds", c.CorrelationIDs, p.CorrelationIDs, d.Catalog,
	)
	c.OperationTraits = mapInto(
		m, "components.operationTraits", c.OperationTraits, p.OperationTraits, d.Catalog,
	)
	c.MessageTraits = mapInto(
		m, "components.messageTraits", c.MessageTraits, p.MessageTraits, d.Catalog,
	)
	c.Replies = mapInto(m, "components.replies", c.Replies, p.Replies, d.Catalog)
	c.ReplyAddresses = mapInto(
		m, "components.replyAddresses", c.ReplyAddresses, p.ReplyAddresses, d.Catalog,
	)
}
