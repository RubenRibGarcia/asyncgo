// Package docdiff compares two decoded AsyncAPI documents and reports the value
// changes between them, one entry per changed key.
//
// It is the explanation behind `asyncgo check --diff`. Check's verdict is a byte
// comparison, and this package answers the question that follows — which values
// actually changed — independently of the encoding of either side. The caller
// decodes both documents, so YAML and JSON are compared the same way, including
// the mixed case where -o names a file whose encoding differs from --format.
//
// The comparison is a walk over the document, not a text diff: the AsyncAPI
// document is a mapping tree, so comparing keys and values reports a rename as
// two subtree changes instead of a hundred changed lines. A subtree one side does
// not have is the interesting case, so it is expanded into one entry per leaf — a
// new channel scrolls as its address, its messages and its bindings rather than
// as `{6 keys}` — while a subtree too deep or too wide to read keeps its single
// counted entry.
package docdiff

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Kind names what happened to a key.
type Kind string

const (
	// Added marks a key the committed document does not have.
	Added Kind = "added"
	// Removed marks a key the generated document no longer has.
	Removed Kind = "removed"
	// Changed marks a key both documents have, with a different value.
	Changed Kind = "changed"
)

// maxValueWidth caps how much of one value a Change renders. A payload schema
// can be a page of JSON, and the path already locates it: the value is there to
// confirm which key changed, not to reproduce the document.
const maxValueWidth = 60

// Expansion limits for a subtree one document does not have. Such a subtree is
// the change, so it is worth listing leaf by leaf; but an added payload schema can
// be hundreds of leaves, and printing those would bury the rest of the drift and
// push the report past the caller's row cap. The two limits bound that:
const (
	// maxExpandDepth is how far below the changed key the walk descends. A
	// container at the limit is one entry whose value is still the whole subtree,
	// so a deep branch costs one row instead of collapsing the useful shallow ones.
	maxExpandDepth = 3
	// maxExpandLeaves is how many leaf entries one changed subtree may
	// contribute. A subtree that needs more is reported as its single counted
	// entry; the budget is all-or-nothing so the report never shows an arbitrary
	// prefix of a subtree and implies it is the whole thing.
	maxExpandLeaves = 10
)

// Change is one difference between the committed document and the generated one.
type Change struct {
	// Path locates the change inside the document: dotted for identifier-ish
	// keys ("channels.order-placed.address") and bracketed for anything else,
	// because a schema key is a fully-qualified Go type and a channel can be an
	// address template — both contain dots or slashes that would make a dotted
	// path ambiguous. Empty only when the document root is not a mapping.
	Path string
	// Kind is what happened to Path.
	Kind Kind
	// Before is the value in the committed document. Nil for Added, where the
	// committed document has nothing at that path.
	Before any
	// After is the value generate would write. Nil for Removed.
	After any
}

// Changes reports every value difference between before and after, sorted by
// path so the same drift always reads the same way.
//
// Both arguments are generic decoded documents (map[string]any, []any, scalars).
// A mapping both sides carry is descended key by key, so a nested edit keeps its
// own verdict and a type change stays one entry. A mapping — or sequence — only
// one side has is expanded into one entry per leaf, bounded by maxExpandDepth and
// maxExpandLeaves; a container that does not fit keeps one counted entry, which
// is what makes a moved schema one row instead of a hundred.
func Changes(before, after any) []Change {
	w := &walker{}
	w.walk("", before, after)
	sort.Slice(
		w.changes, func(i, j int) bool { return w.changes[i].Path < w.changes[j].Path },
	)
	return w.changes
}

// walker accumulates changes during the traversal.
type walker struct {
	changes []Change
}

// walk compares the two values at path.
func (w *walker) walk(path string, before, after any) {
	b, ok := before.(map[string]any)
	if !ok {
		w.scalar(path, before, after)
		return
	}
	a, ok := after.(map[string]any)
	if !ok {
		// A mapping became a scalar (or a sequence): the type changed, so no
		// smaller location describes it.
		w.scalar(path, before, after)
		return
	}
	w.mapping(path, b, a)
}

// scalar records a change unless the two values denote the same thing.
func (w *walker) scalar(path string, before, after any) {
	if equal(before, after) {
		return
	}
	w.changes = append(
		w.changes,
		Change{Path: path, Kind: Changed, Before: before, After: after},
	)
}

// mapping compares two mappings key by key. Keys present on one side only are
// expanded into their leaves: their subtree is the change.
func (w *walker) mapping(path string, before, after map[string]any) {
	for key, b := range before {
		child := join(path, key)
		a, ok := after[key]
		if !ok {
			w.subtree(child, Removed, b, nil)
			continue
		}
		w.walk(child, b, a)
	}
	for key, a := range after {
		if _, ok := before[key]; ok {
			continue
		}
		w.subtree(join(path, key), Added, nil, a)
	}
}

// subtree records a value only one document has at path, as one entry per leaf
// when the subtree is shallow and small enough to read and as a single entry
// otherwise.
//
// before is nil for an addition and after is nil for a removal; the pair says the
// same thing the Kind does, so the summary entry and a leaf entry are built from
// the same two fields.
func (w *walker) subtree(path string, kind Kind, before, after any) {
	value := before
	if kind == Added {
		value = after
	}
	var leaves []Change
	if w.leaves(path, kind, value, 0, &leaves) {
		w.changes = append(w.changes, leaves...)
		return
	}
	w.changes = append(w.changes, Change{Path: path, Kind: kind, Before: before, After: after})
}

// leaves appends one entry per leaf under path and reports whether the subtree
// fitted the expansion budget. A subtree that does not fit is discarded as a
// whole, and the caller keeps its single counted entry.
//
// A node at maxExpandDepth is one entry regardless of its shape: that is the row
// that says "this branch exists and holds N keys" without reproducing it.
func (w *walker) leaves(path string, kind Kind, value any, depth int, out *[]Change) bool {
	if depth < maxExpandDepth {
		switch v := value.(type) {
		case map[string]any:
			if len(v) > 0 {
				for _, key := range sortedKeys(v) {
					if !w.leaves(join(path, key), kind, v[key], depth+1, out) {
						return false
					}
				}
				return true
			}
		case []any:
			if len(v) > 0 {
				for i, item := range v {
					if !w.leaves(indexed(path, i), kind, item, depth+1, out) {
						return false
					}
				}
				return true
			}
		}
	}

	// A scalar, or a container at the depth limit: the value is the whole entry.
	if len(*out) == maxExpandLeaves {
		return false
	}
	leaf := Change{Path: path, Kind: kind}
	if kind == Added {
		leaf.After = value
	} else {
		leaf.Before = value
	}
	*out = append(*out, leaf)
	return true
}

// sortedKeys returns the mapping's keys in order, so the leaf walk spends its
// budget the same way on every run.
func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// indexed appends a sequence position to a document path.
func indexed(parent string, i int) string {
	return parent + "[" + strconv.Itoa(i) + "]"
}

// equal compares two decoded values. Numbers compare by exact value, because the
// same number carries a different Go type depending on the decoder and the
// encoding — goccy/go-yaml returns uint64 for 3 and int64 for -7, while JSON
// returns float64 for both — and the artifact and the generated bytes may not
// even be in the same encoding, since -o may name a YAML file while --format is
// json.
func equal(a, b any) bool {
	if na, ok := number(a); ok {
		nb, ok := number(b)
		return ok && na.Cmp(nb) == 0
	}
	switch av := a.(type) {
	case nil:
		return b == nil
	case []any:
		bv, ok := b.([]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !equal(av[i], bv[i]) {
				return false
			}
		}
		return true
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for key, v := range av {
			other, ok := bv[key]
			if !ok || !equal(v, other) {
				return false
			}
		}
		return true
	default:
		// Strings, booleans, and the []byte a YAML !!binary scalar decodes to.
		// DeepEqual is type-aware, so a string "3" never equals a number that
		// renders the same way.
		return reflect.DeepEqual(a, b)
	}
}

// number converts a decoded numeric scalar to an exact rational. It recognises
// the types the decoders produce: goccy/go-yaml yields uint64, int64 and
// float64; encoding/json is read with UseNumber, so an integer keeps its digits
// rather than passing through float64.
func number(v any) (*big.Rat, bool) {
	switch n := v.(type) {
	case uint64:
		return new(big.Rat).SetInt(new(big.Int).SetUint64(n)), true
	case int64:
		return new(big.Rat).SetInt64(n), true
	case int:
		return new(big.Rat).SetInt64(int64(n)), true
	case float64:
		rat := new(big.Rat).SetFloat64(n)
		return rat, rat != nil
	case json.Number:
		rat, ok := new(big.Rat).SetString(n.String())
		return rat, ok
	}
	return nil, false
}

// identifier matches the keys rendered in dotted form. AsyncAPI keys are not all
// identifiers, so anything else is quoted instead of silently becoming an
// ambiguous path.
var identifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)

// join appends one key to a document path.
func join(parent, key string) string {
	if !identifier.MatchString(key) {
		return parent + "[" + strconv.Quote(key) + "]"
	}
	if parent == "" {
		return key
	}
	return parent + "." + key
}

// FormatValue renders a decoded value as compact JSON, so one value reads the
// same whether the document it came from was YAML or JSON. encoding/json already
// covers everything the decoders produce: strings quote and escape, numbers keep
// their exact text, null is null, and a YAML !!binary scalar becomes a base64
// string, which is the only faithful JSON rendering of it.
//
// A value too large to print is summarised rather than cut mid-token: a truncated
// JSON object is no longer JSON, and when the path already names the subtree, how
// many keys it holds is the useful part.
func FormatValue(v any) string {
	rendered, err := compactJSON(v)
	if err != nil {
		return clipText(fmt.Sprint(v))
	}
	if utf8.RuneCountInString(rendered) <= maxValueWidth {
		return rendered
	}
	switch t := v.(type) {
	case string:
		return clipString(t)
	case []byte:
		return clipString(base64.StdEncoding.EncodeToString(t))
	case map[string]any:
		return fmt.Sprintf("{%d keys}", len(t))
	case []any:
		return fmt.Sprintf("[%d items]", len(t))
	default:
		return clipText(rendered)
	}
}

// compactJSON encodes a value on one line with HTML escaping off, so a topic or
// address containing an ampersand is printed as written.
func compactJSON(v any) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return "", fmt.Errorf("encoding value as json: %w", err)
	}
	return strings.TrimSuffix(buf.String(), "\n"), nil
}

// clipString quotes a prefix of a long string and appends an ellipsis, so what is
// printed is a complete string literal followed by the marker rather than an
// unterminated quote.
func clipString(s string) string {
	const room = maxValueWidth - len(`""...`)
	runes := []rune(s)
	if len(runes) > room {
		s = string(runes[:room])
	}
	return strconv.Quote(s) + "..."
}

// clipText bounds a rendered scalar, on rune boundaries so the result stays valid
// UTF-8 in a terminal and in a CI annotation.
func clipText(s string) string {
	if utf8.RuneCountInString(s) <= maxValueWidth {
		return s
	}
	return string([]rune(s)[:maxValueWidth-len("...")]) + "..."
}
