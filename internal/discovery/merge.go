package discovery

import "github.com/RubenRibGarcia/asyncgo/spec"

// Merge combines catalog documents into a single AsyncAPI document. Maps are
// unioned (first occurrence wins on key collision); Info and DefaultContentType
// are taken from the first document that defines them.
func Merge(docs ...*spec.AsyncAPI) *spec.AsyncAPI {
	out := spec.New()
	for _, d := range docs {
		if d == nil {
			continue
		}
		if out.Info.Title == "" && d.Info.Title != "" {
			out.Info = d.Info
		}
		if out.DefaultContentType == "" {
			out.DefaultContentType = d.DefaultContentType
		}
		out.Servers = mergeMap(out.Servers, d.Servers)
		out.Channels = mergeMap(out.Channels, d.Channels)
		out.Operations = mergeMap(out.Operations, d.Operations)
		mergeComponents(out, d)
	}
	return out
}

func mergeComponents(out, d *spec.AsyncAPI) {
	if d.Components == nil {
		return
	}
	if out.Components == nil {
		out.Components = &spec.Components{}
	}
	c := out.Components
	c.Schemas = mergeMap(c.Schemas, d.Components.Schemas)
	c.Servers = mergeMap(c.Servers, d.Components.Servers)
	c.Channels = mergeMap(c.Channels, d.Components.Channels)
	c.Operations = mergeMap(c.Operations, d.Components.Operations)
	c.Messages = mergeMap(c.Messages, d.Components.Messages)
	c.Parameters = mergeMap(c.Parameters, d.Components.Parameters)
	c.CorrelationIDs = mergeMap(c.CorrelationIDs, d.Components.CorrelationIDs)
}

func mergeMap[V any](dst, src map[string]V) map[string]V {
	if dst == nil {
		dst = map[string]V{}
	}
	for k, v := range src {
		if _, ok := dst[k]; !ok {
			dst[k] = v
		}
	}
	return dst
}
