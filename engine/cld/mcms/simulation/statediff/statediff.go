// Package statediff computes structural diffs between two raw (unformatted)
// stategen outputs: which named state paths changed, and which views could not
// be read reliably (Unreadable — coverage findings, never presented as changes).
//
// The input is the unformatted JSON produced by stategen.Run: a "chains" map
// keyed by chain selector whose values are arrays of view objects (each stamped
// with address, _qualifier, _type and _requestedVersion), plus optional
// "_errors" and "_skipped" arrays. Formatted output (Format: true) discards
// those sections and cannot be diffed.
package statediff

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// ChangeKind classifies a single observed state change.
type ChangeKind uint8

const (
	KindAdded ChangeKind = iota
	KindRemoved
	KindChanged
)

func (k ChangeKind) String() string {
	switch k {
	case KindAdded:
		return "added"
	case KindRemoved:
		return "removed"
	case KindChanged:
		return "changed"
	default:
		return "unknown"
	}
}

// MarshalJSON renders the kind as its name so artifacts stay human-readable.
func (k ChangeKind) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.String())
}

// UnmarshalJSON reads the kind back from its name, so artifacts round-trip.
func (k *ChangeKind) UnmarshalJSON(data []byte) error {
	var name string
	if err := json.Unmarshal(data, &name); err != nil {
		return fmt.Errorf("change kind %s is not a string: %w", data, err)
	}

	switch name {
	case "added":
		*k = KindAdded
	case "removed":
		*k = KindRemoved
	case "changed":
		*k = KindChanged
	default:
		return fmt.Errorf("unknown change kind %q", name)
	}

	return nil
}

// Change is one named before/after state difference.
type Change struct {
	ChainSelector uint64     `json:"chainSelector"`
	Address       string     `json:"address"`
	ContractType  string     `json:"contractType"`
	Path          string     `json:"path"`
	Kind          ChangeKind `json:"kind"`
	Before        any        `json:"before,omitempty"`
	After         any        `json:"after,omitempty"`
}

// Unreadable marks a view that could not be read reliably.
// Unreadable entries are tracked as coverage findings, never as changes.
type Unreadable struct {
	ChainSelector uint64 `json:"chainSelector"`
	Address       string `json:"address"`
	ContractType  string `json:"contractType"`
	Version       string `json:"version,omitempty"`
	Reason        string `json:"reason"`
}

// Reasons a view or object can be Unreadable.
const (
	// ReasonUnknownAddress: scope resolution found no datastore ref for the address.
	ReasonUnknownAddress = "unknown-address"
	// ReasonReadError: the view embeds an *_error field (a read failed mid-view).
	ReasonReadError = "read-error"
	// ReasonViewSkipped: address listed in top-level _skipped (no view implementation).
	ReasonViewSkipped = "view-skipped"
	// ReasonViewError: address listed in top-level _errors (the view call failed).
	ReasonViewError = "view-error"
	// ReasonMalformedJSON: one side of a view is not valid JSON.
	ReasonMalformedJSON = "malformed-json"
)

// ignoreRule suppresses changes at a view-relative path for views whose
// contract type contains TypeSubstring. An empty TypeSubstring matches all types.
type ignoreRule struct {
	TypeSubstring string
	Path          string
}

// defaultIgnoreRules suppresses paths that churn on every fork run without
// semantic content.
var defaultIgnoreRules = []ignoreRule{
	// Rate-limiter refresh stamps are wall-clock coupled: every config apply
	// rewrites them with the current fork timestamp (which the timelock leg
	// time-travels).
	{Path: "inboundRateLimiterConfig.lastUpdated"},
	{Path: "outboundRateLimiterConfig.lastUpdated"},
}

// DefaultIgnoredPaths returns the view-relative paths ignored by default.
// MCMS signer and quorum changes are business state. Any fork-harness
// compensation must happen before calling the generic differ.
func DefaultIgnoredPaths() []string {
	paths := make([]string, 0, len(defaultIgnoreRules))
	for _, r := range defaultIgnoreRules {
		paths = append(paths, r.Path)
	}

	return paths
}

// Option configures Diff.
type Option func(*options)

type options struct {
	ignored []ignoreRule
}

// WithIgnoredPaths adds view-relative paths to the default ignore list.
// Matching is exact, prefix ("a.b" also suppresses "a.b.c") or suffix
// ("b.c" also suppresses "a.b.c"), so a path can be pinned at any depth.
func WithIgnoredPaths(paths ...string) Option {
	return func(o *options) {
		for _, p := range paths {
			o.ignored = append(o.ignored, ignoreRule{Path: p})
		}
	}
}

// Diff computes the structural difference between two raw stategen outputs.
func Diff(before, after map[string]json.RawMessage, opts ...Option) ([]Change, []Unreadable, error) {
	o := &options{ignored: defaultIgnoreRules}
	for _, opt := range opts {
		opt(o)
	}
	d := &differ{opts: o, seenUnreadable: map[string]struct{}{}}

	beforeChains, err := parseChains(before)
	if err != nil {
		return nil, nil, err
	}
	afterChains, err := parseChains(after)
	if err != nil {
		return nil, nil, err
	}

	bad, err := d.coveragePass(before)
	if err != nil {
		return nil, nil, err
	}
	afterBad, err := d.coveragePass(after)
	if err != nil {
		return nil, nil, err
	}
	for identity := range afterBad {
		bad[identity] = struct{}{}
	}
	d.quarantineViews(beforeChains, bad)
	d.quarantineViews(afterChains, bad)

	for _, sel := range sortedChainSelectors(beforeChains, afterChains) {
		d.diffChain(sel, beforeChains[sel], afterChains[sel], bad)
	}

	return d.changes, d.unreadables, nil
}

// DiffJSON is Diff over two raw stategen output documents ([]byte form).
func DiffJSON(before, after []byte, opts ...Option) ([]Change, []Unreadable, error) {
	var beforeMap, afterMap map[string]json.RawMessage
	if err := json.Unmarshal(before, &beforeMap); err != nil {
		return nil, nil, fmt.Errorf("parsing before document: %w", err)
	}
	if err := json.Unmarshal(after, &afterMap); err != nil {
		return nil, nil, fmt.Errorf("parsing after document: %w", err)
	}

	return Diff(beforeMap, afterMap, opts...)
}

// differ accumulates changes and unreadable findings for one Diff call.
type differ struct {
	opts           *options
	changes        []Change
	unreadables    []Unreadable
	seenUnreadable map[string]struct{}
}

func (d *differ) add(c Change) {
	d.changes = append(d.changes, c)
}

func (d *differ) unreadable(u Unreadable) {
	key := coverageKey(u.ChainSelector, u.Address) + "|" + u.ContractType + "|" + u.Version + "|" + u.Reason
	if _, seen := d.seenUnreadable[key]; seen {
		return
	}
	d.seenUnreadable[key] = struct{}{}
	d.unreadables = append(d.unreadables, u)
}

// coveragePass records _errors/_skipped entries as Unreadable and returns the
// set of addresses that must never be diffed.
func (d *differ) coveragePass(side map[string]json.RawMessage) (map[string]struct{}, error) {
	bad := map[string]struct{}{}
	for _, section := range []string{"_errors", "_skipped"} {
		reason := ReasonViewError
		if section == "_skipped" {
			reason = ReasonViewSkipped
		}
		raw, ok := side[section]
		if !ok || len(raw) == 0 {
			continue
		}
		var entries []coverageEntry
		if err := json.Unmarshal(raw, &entries); err != nil {
			return nil, fmt.Errorf("parsing %s coverage: %w", section, err)
		}
		for _, e := range entries {
			if e.ChainSelector.value() == 0 || strings.TrimSpace(e.Address) == "" {
				return nil, fmt.Errorf("%s coverage entry requires a chain selector and address", section)
			}
			bad[coverageKey(e.ChainSelector.value(), e.Address)] = struct{}{}
			d.unreadable(Unreadable{
				ChainSelector: e.ChainSelector.value(),
				Address:       e.Address,
				ContractType:  e.Type,
				Version:       e.Version,
				Reason:        reason,
			})
		}
	}

	return bad, nil
}

func coverageKey(selector uint64, address string) string {
	return strconv.FormatUint(selector, 10) + "|" + normalizeAddress(address)
}

// Quarantine before walking either side, including newly added and removed
// views. Any nested getter failure makes the entire target unreadable, so it
// can never simultaneously claim readable changes from a sibling field.
func (d *differ) quarantineViews(chains map[uint64]chainViews, bad map[string]struct{}) {
	for _, sel := range sortedChainSelectors(chains, nil) {
		for _, entry := range chains[sel].views {
			reason := viewFailure(entry.obj)
			if reason == "" {
				continue
			}
			addr := viewString(entry.obj, "address")
			bad[coverageKey(sel, addr)] = struct{}{}
			d.unreadable(Unreadable{
				ChainSelector: sel,
				Address:       addr,
				ContractType:  viewString(entry.obj, "_type"),
				Version:       viewString(entry.obj, "_requestedVersion"),
				Reason:        reason,
			})
		}
	}
}

func viewFailure(value any) string {
	failure := ""
	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			if key == "_meta" {
				continue
			}
			if strings.HasSuffix(key, "_error") {
				return ReasonReadError
			}
			if key == "chainSelector" {
				if _, valid := selectorIdentity(child); !valid {
					failure = ReasonMalformedJSON
				}
			}
			if reason := viewFailure(child); reason != "" {
				if reason == ReasonReadError {
					return reason
				}
				failure = reason
			}
		}
	case []any:
		for _, child := range v {
			if reason := viewFailure(child); reason != "" {
				if reason == ReasonReadError {
					return reason
				}
				failure = reason
			}
		}
	}

	return failure
}

// coverageEntry is one _errors/_skipped record. ChainSelector arrives as
// either a JSON string or number depending on producer, so it uses a flexible
// unmarshalling type.
type coverageEntry struct {
	Address       string  `json:"address"`
	ChainSelector flexSel `json:"chainSelector"`
	Type          string  `json:"type"`
	Version       string  `json:"version"`
}

// flexSel unmarshals a chain selector given as either a JSON string or number.
type flexSel struct{ n uint64 }

func (f *flexSel) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "null" || s == "" {
		return nil
	}
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return fmt.Errorf("chain selector %q: %w", s, err)
	}
	f.n = n

	return nil
}

func (f flexSel) value() uint64 { return f.n }

// viewEntry is one view from a chain's array: the parsed object, or a
// malformed marker when the element could not be parsed as a JSON object.
type viewEntry struct {
	obj       map[string]any
	malformed bool
}

// chainViews is one chain's view entries, or a malformed marker when the
// chain's views section itself could not be parsed as an array.
type chainViews struct {
	views     []viewEntry
	malformed bool
}

// parseChains extracts the "chains" section as selector -> chain views. A
// missing section is an empty set (not an error): empty documents diff fine.
// Malformed data degrades to Unreadable coverage findings, never a whole-
// document error: a chain whose views section is not a parseable array marks
// the chain malformed, and a view element that is not a JSON object marks that
// element malformed — everything else still diffs.
func parseChains(side map[string]json.RawMessage) (map[uint64]chainViews, error) {
	out := map[uint64]chainViews{}
	raw, ok := side["chains"]
	if !ok || len(raw) == 0 {
		return out, nil
	}
	var selectorMap map[string]json.RawMessage
	if err := json.Unmarshal(raw, &selectorMap); err != nil {
		return nil, fmt.Errorf("parsing chains section: %w", err)
	}
	for selStr, viewsRaw := range selectorMap {
		sel, err := strconv.ParseUint(selStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing chain selector %q: %w", selStr, err)
		}
		var rawViews []json.RawMessage
		if err := json.Unmarshal(viewsRaw, &rawViews); err != nil {
			out[sel] = chainViews{malformed: true}

			continue
		}
		views := make([]viewEntry, 0, len(rawViews))
		for _, rv := range rawViews {
			var obj map[string]any
			if err := DecodeJSON(rv, &obj); err != nil || obj == nil {
				views = append(views, viewEntry{malformed: true})

				continue
			}
			views = append(views, viewEntry{obj: obj})
		}
		out[sel] = chainViews{views: views}
	}

	return out, nil
}

func sortedChainSelectors(a, b map[uint64]chainViews) []uint64 {
	set := map[uint64]struct{}{}
	for sel := range a {
		set[sel] = struct{}{}
	}
	for sel := range b {
		set[sel] = struct{}{}
	}
	sels := make([]uint64, 0, len(set))
	for sel := range set {
		sels = append(sels, sel)
	}
	sort.Slice(sels, func(i, j int) bool { return sels[i] < sels[j] })

	return sels
}

func chainPath(sel uint64) string {
	return "chains." + strconv.FormatUint(sel, 10)
}

// diffChain diffs one chain's view arrays, keyed by view identity
// (address, _qualifier, _type, _requestedVersion). Views without an address
// stamp fall back to positional identity. A malformed chain or view element is
// one Unreadable finding per chain and is never diffed.
func (d *differ) diffChain(sel uint64, beforeCV, afterCV chainViews, bad map[string]struct{}) {
	if beforeCV.malformed || afterCV.malformed {
		d.unreadable(Unreadable{ChainSelector: sel, Reason: ReasonMalformedJSON})

		return
	}
	before := indexViews(beforeCV.views)
	after := indexViews(afterCV.views)

	keys := make([]string, 0, len(before)+len(after))
	for k := range before {
		keys = append(keys, k)
	}
	for k := range after {
		if _, ok := before[k]; !ok {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	for _, key := range keys {
		bv, inBefore := before[key]
		av, inAfter := after[key]
		if (inBefore && bv.malformed) || (inAfter && av.malformed) {
			d.unreadable(Unreadable{ChainSelector: sel, Reason: ReasonMalformedJSON})

			continue
		}
		addr := viewAddress(key)
		if _, unreadable := bad[coverageKey(sel, addr)]; unreadable {
			continue
		}
		path := chainPath(sel) + "." + viewPathKey(key)
		ctype := viewString(av.obj, "_type")
		if ctype == "" {
			ctype = viewString(bv.obj, "_type")
		}
		switch {
		case !inBefore:
			d.add(Change{ChainSelector: sel, Address: addr, ContractType: ctype, Path: path, Kind: KindAdded, After: av.obj})
		case !inAfter:
			d.add(Change{ChainSelector: sel, Address: addr, ContractType: ctype, Path: path, Kind: KindRemoved, Before: bv.obj})
		default:
			d.walk(sel, addr, ctype, path, "", bv.obj, av.obj)
		}
	}
}

// indexViews keys each view by its identity stamp tuple (or position).
func indexViews(views []viewEntry) map[string]viewEntry {
	out := make(map[string]viewEntry, len(views))
	for i, v := range views {
		addr := normalizeAddress(viewString(v.obj, "address"))
		key := addr + "|" + viewString(v.obj, "_qualifier") + "|" + viewString(v.obj, "_type") + "|" + viewString(v.obj, "_requestedVersion")
		if addr == "" {
			key = "#" + strconv.Itoa(i)
		}
		out[key] = v
	}

	return out
}

// viewAddress extracts the address part of an indexViews key; positional keys
// have none.
func viewAddress(key string) string {
	if strings.HasPrefix(key, "#") {
		return ""
	}

	return strings.SplitN(key, "|", 2)[0]
}

// viewPathKey renders an indexViews key as the path segment for changes: the
// address plus its qualifier when set (type and version travel on the Change).
func viewPathKey(key string) string {
	if strings.HasPrefix(key, "#") {
		return key
	}
	parts := strings.Split(key, "|")
	if len(parts) > 1 && parts[1] != "" {
		return parts[0] + "_" + parts[1]
	}

	return parts[0]
}

func viewString(v map[string]any, field string) string {
	if v == nil {
		return ""
	}
	s, _ := v[field].(string)

	return s
}

// walk recursively diffs one value pair. rel is the path relative to the view
// root, used for ignore-rule matching; path is the full change path.
func (d *differ) walk(sel uint64, addr, ctype, path, rel string, a, b any) {
	if am, ok := a.(map[string]any); ok {
		if bm, ok := b.(map[string]any); ok {
			keys := make([]string, 0, len(am)+len(bm))
			for k := range am {
				keys = append(keys, k)
			}
			for k := range bm {
				if _, ok := am[k]; !ok {
					keys = append(keys, k)
				}
			}
			sort.Strings(keys)
			for _, k := range keys {
				if k == "_meta" {
					continue
				}
				childRel := k
				if rel != "" {
					childRel = rel + "." + k
				}
				if d.opts.isIgnored(ctype, childRel) {
					continue
				}
				childPath := path + "." + k
				av, inA := am[k]
				bv, inB := bm[k]
				switch {
				case !inA:
					d.add(Change{ChainSelector: sel, Address: addr, ContractType: ctype, Path: childPath, Kind: KindAdded, After: bv})
				case !inB:
					d.add(Change{ChainSelector: sel, Address: addr, ContractType: ctype, Path: childPath, Kind: KindRemoved, Before: av})
				default:
					d.walk(sel, addr, ctype, childPath, childRel, av, bv)
				}
			}

			return
		}
	}
	if as, ok := a.([]any); ok {
		if bs, ok := b.([]any); ok {
			d.walkArray(sel, addr, ctype, path, rel, as, bs)

			return
		}
	}
	if !EqualJSONValues(a, b) {
		d.add(Change{ChainSelector: sel, Address: addr, ContractType: ctype, Path: path, Kind: KindChanged, Before: a, After: b})
	}
}

// walkArray diffs two arrays. When every element on both sides carries an
// address or chainSelector, elements are keyed by that value (reorder-tolerant);
// otherwise they are compared positionally. Duplicate keys also fall back to
// positions: a keyed map must never silently discard an ambiguous element.
func (d *differ) walkArray(sel uint64, addr, ctype, path, rel string, a, b []any) {
	ka, kb := elementKeys(a), elementKeys(b)
	if allKeyed(ka) && allKeyed(kb) {
		am := keyElements(a, ka)
		bm := keyElements(b, kb)
		keys := make([]string, 0, len(am)+len(bm))
		for k := range am {
			keys = append(keys, k)
		}
		for k := range bm {
			if _, ok := am[k]; !ok {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		for _, k := range keys {
			childRel := rel + "." + k
			if d.opts.isIgnored(ctype, childRel) {
				continue
			}
			childPath := path + "[" + k + "]"
			av, inA := am[k]
			bv, inB := bm[k]
			switch {
			case !inA:
				d.add(Change{ChainSelector: sel, Address: addr, ContractType: ctype, Path: childPath, Kind: KindAdded, After: bv})
			case !inB:
				d.add(Change{ChainSelector: sel, Address: addr, ContractType: ctype, Path: childPath, Kind: KindRemoved, Before: av})
			default:
				d.walk(sel, addr, ctype, childPath, childRel, av, bv)
			}
		}

		return
	}
	for i := 0; i < len(a) || i < len(b); i++ {
		childPath := path + "[" + strconv.Itoa(i) + "]"
		switch {
		case i >= len(a):
			d.add(Change{ChainSelector: sel, Address: addr, ContractType: ctype, Path: childPath, Kind: KindAdded, After: b[i]})
		case i >= len(b):
			d.add(Change{ChainSelector: sel, Address: addr, ContractType: ctype, Path: childPath, Kind: KindRemoved, Before: a[i]})
		default:
			childRel := rel + "." + strconv.Itoa(i)
			if d.opts.isIgnored(ctype, childRel) {
				continue
			}
			d.walk(sel, addr, ctype, childPath, childRel, a[i], b[i])
		}
	}
}

// elementKeys returns the key of each element (address or chainSelector
// field), or "" for elements without one (including non-objects).
func elementKeys(arr []any) []string {
	keys := make([]string, len(arr))
	for i, e := range arr {
		keys[i] = elementKey(e)
	}

	return keys
}

func elementKey(e any) string {
	m, ok := e.(map[string]any)
	if !ok {
		return ""
	}
	if address, ok := m["address"].(string); ok && address != "" {
		return normalizeAddress(address)
	}
	if selector, valid := selectorIdentity(m["chainSelector"]); valid {
		return selector
	}

	return ""
}

func selectorIdentity(value any) (string, bool) {
	switch v := value.(type) {
	case json.Number:
		return selectorNumber(v)
	case string:
		selector, err := strconv.ParseUint(v, 10, 64)
		return strconv.FormatUint(selector, 10), err == nil
	default:
		return "", false
	}
}

func allKeyed(keys []string) bool {
	if len(keys) == 0 {
		return true
	}
	seen := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		if k == "" {
			return false
		}
		if _, duplicate := seen[k]; duplicate {
			return false
		}
		seen[k] = struct{}{}
	}

	return true
}

func keyElements(arr []any, keys []string) map[string]any {
	out := make(map[string]any, len(arr))
	for i, e := range arr {
		out[keys[i]] = e
	}

	return out
}

// isIgnored reports whether changes at the view-relative path rel are
// suppressed for a view of contract type ctype.
func (o *options) isIgnored(ctype, rel string) bool {
	for _, r := range o.ignored {
		if r.TypeSubstring != "" && !strings.Contains(ctype, r.TypeSubstring) {
			continue
		}
		if rel == r.Path || strings.HasPrefix(rel, r.Path+".") || strings.HasSuffix(rel, "."+r.Path) {
			return true
		}
	}

	return false
}
