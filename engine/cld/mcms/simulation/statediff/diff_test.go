package statediff

import (
	"encoding/json"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	sel      = 5009297550715157269
	otherSel = uint64(111)
)

func view(addr, ctype string, fields map[string]any) map[string]any {
	v := map[string]any{"address": addr, "_type": ctype, "_requestedVersion": "1.6.0"}
	for k, f := range fields {
		v[k] = f
	}

	return v
}

func chainsDoc(views ...map[string]any) map[string]any {
	return map[string]any{"chains": map[string]any{strconv.FormatUint(sel, 10): views}}
}

// skippedDoc builds a document whose 0xabc view is also listed in _skipped.
func skippedDoc() map[string]any {
	d := chainsDoc(view("0xabc", "CallProxy", map[string]any{"owner": "0x1"}))
	d["_skipped"] = []any{map[string]any{
		"address": "0xabc", "chainSelector": strconv.FormatUint(sel, 10), "type": "CallProxy", "version": "1.1.0",
	}}

	return d
}

// rawDoc round-trips a value through JSON so tests construct inputs exactly
// as unmarshalled stategen output (numbers as float64, etc).
func rawDoc(t *testing.T, m map[string]any) map[string]json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(m)
	require.NoError(t, err)
	var out map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &out))

	return out
}

type changeProjection struct {
	path string
	kind ChangeKind
}

func projectChanges(t *testing.T, changes []Change) []changeProjection {
	t.Helper()
	if len(changes) == 0 {
		return nil
	}
	out := make([]changeProjection, 0, len(changes))
	for _, c := range changes {
		out = append(out, changeProjection{path: c.Path, kind: c.Kind})
	}

	return out
}

func TestDiff(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		before, after  map[string]any
		opts           []Option
		wantChanges    []changeProjection
		wantUnreadable []Unreadable // only Address and Reason are asserted
	}{
		{
			name:   "identical documents produce no changes",
			before: chainsDoc(view("0xabc", "Router", map[string]any{"owner": "0x1"})),
			after:  chainsDoc(view("0xabc", "Router", map[string]any{"owner": "0x1"})),
		},
		{
			name: "_meta is excluded at any level",
			before: func() map[string]any {
				d := chainsDoc(view("0xabc", "Router", map[string]any{"owner": "0x1"}))
				d["_meta"] = map[string]any{"totalDuration": "1s"}

				return d
			}(),
			after: func() map[string]any {
				d := chainsDoc(view("0xabc", "Router", map[string]any{"owner": "0x1"}))
				d["_meta"] = map[string]any{"totalDuration": "2s"}

				return d
			}(),
		},
		{
			name: "nested scalar change",
			before: chainsDoc(view("0xabc", "OnRamp", map[string]any{
				"destChainConfigs": map[string]any{"7222032299962346917": map[string]any{"maxDataBytes": float64(30000)}},
			})),
			after: chainsDoc(view("0xabc", "OnRamp", map[string]any{
				"destChainConfigs": map[string]any{"7222032299962346917": map[string]any{"maxDataBytes": float64(50000)}},
			})),
			wantChanges: []changeProjection{
				{path: "chains." + strconv.FormatUint(sel, 10) + ".0xabc.destChainConfigs.7222032299962346917.maxDataBytes", kind: KindChanged},
			},
		},
		{
			name: "star_error key quarantines the whole view, never a change",
			before: chainsDoc(view("0xabc", "CapabilitiesRegistry", map[string]any{
				"owner": "0x1", "dons": []any{map[string]any{"id": float64(1), "configCount": float64(2)}},
			})),
			after: chainsDoc(view("0xabc", "CapabilitiesRegistry", map[string]any{
				"owner": "0x2", "dons": []any{map[string]any{"id": float64(1), "configCount": float64(99)}}, "dons_error": "timeout",
			})),
			wantUnreadable: []Unreadable{{Address: "0xabc", Reason: ReasonReadError}},
		},
		{
			name: "other views still diff when one view is quarantined",
			before: chainsDoc(
				view("0xabc", "CapabilitiesRegistry", map[string]any{"dons": "x", "dons_error": "timeout"}),
				view("0xdef", "Router", map[string]any{"owner": "0x1"}),
			),
			after: chainsDoc(
				view("0xabc", "CapabilitiesRegistry", map[string]any{"dons": "y", "dons_error": "timeout"}),
				view("0xdef", "Router", map[string]any{"owner": "0x2"}),
			),
			wantChanges: []changeProjection{
				{path: "chains." + strconv.FormatUint(sel, 10) + ".0xdef.owner", kind: KindChanged},
			},
			wantUnreadable: []Unreadable{{Address: "0xabc", Reason: ReasonReadError}},
		},
		{
			name: "H2 ignore list suppresses MCMS signer bookkeeping only",
			before: chainsDoc(
				view("0xabc", "ManyChainMultiSig", map[string]any{
					"config": map[string]any{"signers": []any{"a", "b"}, "groupQuorums": []any{float64(2)}, "owner": "0x1"},
				}),
			),
			after: chainsDoc(
				view("0xabc", "ManyChainMultiSig", map[string]any{
					"config": map[string]any{"signers": []any{"c"}, "groupQuorums": []any{float64(1)}, "owner": "0x1"},
				}),
			),
		},
		{
			name: "ignore rules do not leak to siblings or other types",
			before: chainsDoc(
				view("0xabc", "ManyChainMultiSig", map[string]any{
					"config": map[string]any{"groupQuorums": []any{float64(2)}, "owner": "0x1"},
				}),
				view("0xdef", "Router", map[string]any{"config": map[string]any{"groupQuorums": float64(2)}}),
			),
			after: chainsDoc(
				view("0xabc", "ManyChainMultiSig", map[string]any{
					"config": map[string]any{"groupQuorums": []any{float64(1)}, "owner": "0x2"},
				}),
				view("0xdef", "Router", map[string]any{"config": map[string]any{"groupQuorums": float64(3)}}),
			),
			wantChanges: []changeProjection{
				{path: "chains." + strconv.FormatUint(sel, 10) + ".0xabc.config.owner", kind: KindChanged},
				{path: "chains." + strconv.FormatUint(sel, 10) + ".0xdef.config.groupQuorums", kind: KindChanged},
			},
		},
		{
			name: "default lastUpdated suppression with sibling value still reported",
			before: chainsDoc(view("0xabc", "LockReleaseTokenPool", map[string]any{
				"remoteChainConfigs": map[string]any{"7222032299962346917": map[string]any{
					"inboundRateLimiterConfig":  map[string]any{"rate": float64(100), "lastUpdated": float64(1789139111)},
					"outboundRateLimiterConfig": map[string]any{"rate": float64(100), "lastUpdated": float64(1789139111)},
				}},
			})),
			after: chainsDoc(view("0xabc", "LockReleaseTokenPool", map[string]any{
				"remoteChainConfigs": map[string]any{"7222032299962346917": map[string]any{
					"inboundRateLimiterConfig":  map[string]any{"rate": float64(200), "lastUpdated": float64(1789150089)},
					"outboundRateLimiterConfig": map[string]any{"rate": float64(100), "lastUpdated": float64(1789150089)},
				}},
			})),
			wantChanges: []changeProjection{
				{path: "chains." + strconv.FormatUint(sel, 10) + ".0xabc.remoteChainConfigs.7222032299962346917.inboundRateLimiterConfig.rate", kind: KindChanged},
			},
		},
		{
			name: "keyed array reorder is not a change",
			before: chainsDoc(view("0xabc", "TokenAdminRegistry", map[string]any{
				"tokens": []any{
					map[string]any{"address": "0x1", "rate": float64(5)},
					map[string]any{"address": "0x2", "rate": float64(6)},
				},
			})),
			after: chainsDoc(view("0xabc", "TokenAdminRegistry", map[string]any{
				"tokens": []any{
					map[string]any{"address": "0x2", "rate": float64(6)},
					map[string]any{"address": "0x1", "rate": float64(5)},
				},
			})),
		},
		{
			name: "keyed array element added and removed",
			before: chainsDoc(view("0xabc", "TokenAdminRegistry", map[string]any{
				"tokens": []any{map[string]any{"address": "0x1", "rate": float64(5)}},
			})),
			after: chainsDoc(view("0xabc", "TokenAdminRegistry", map[string]any{
				"tokens": []any{
					map[string]any{"address": "0x2", "rate": float64(7)},
				},
			})),
			wantChanges: []changeProjection{
				{path: "chains." + strconv.FormatUint(sel, 10) + ".0xabc.tokens[0x1]", kind: KindRemoved},
				{path: "chains." + strconv.FormatUint(sel, 10) + ".0xabc.tokens[0x2]", kind: KindAdded},
			},
		},
		{
			name:   "positional array length mismatch yields element-level change",
			before: chainsDoc(view("0xabc", "OnRamp", map[string]any{"f": []any{float64(1), float64(2)}})),
			after:  chainsDoc(view("0xabc", "OnRamp", map[string]any{"f": []any{float64(1), float64(2), float64(3)}})),
			wantChanges: []changeProjection{
				{path: "chains." + strconv.FormatUint(sel, 10) + ".0xabc.f[2]", kind: KindAdded},
			},
		},
		{
			name:           "skipped views are Unreadable, not changes",
			before:         chainsDoc(view("0xabc", "CallProxy", map[string]any{"owner": "0x1"})),
			after:          skippedDoc(),
			wantUnreadable: []Unreadable{{Address: "0xabc", Reason: ReasonViewSkipped}},
		},
		{
			name:   "view added and removed",
			before: chainsDoc(view("0xabc", "Router", map[string]any{"owner": "0x1"})),
			after:  chainsDoc(view("0xdef", "Router", map[string]any{"owner": "0x1"})),
			wantChanges: []changeProjection{
				{path: "chains." + strconv.FormatUint(sel, 10) + ".0xabc", kind: KindRemoved},
				{path: "chains." + strconv.FormatUint(sel, 10) + ".0xdef", kind: KindAdded},
			},
		},
		{
			name: "multiple views per address are keyed by stamp tuple",
			before: chainsDoc(
				map[string]any{"address": "0xabc", "_qualifier": "monolith-AptosRouter", "_type": "AptosRouter", "owner": "0x1"},
				map[string]any{"address": "0xabc", "_qualifier": "monolith-AptosOnRamp", "_type": "AptosOnRamp", "config": float64(1)},
			),
			after: chainsDoc(
				map[string]any{"address": "0xabc", "_qualifier": "monolith-AptosRouter", "_type": "AptosRouter", "owner": "0x1"},
				map[string]any{"address": "0xabc", "_qualifier": "monolith-AptosOnRamp", "_type": "AptosOnRamp", "config": float64(2)},
			),
			wantChanges: []changeProjection{
				{path: "chains." + strconv.FormatUint(sel, 10) + ".0xabc_monolith-AptosOnRamp.config", kind: KindChanged},
			},
		},
		{
			name:   "empty before document yields chain added",
			before: map[string]any{},
			after:  chainsDoc(view("0xabc", "Router", map[string]any{"owner": "0x1"})),
			wantChanges: []changeProjection{
				{path: "chains." + strconv.FormatUint(sel, 10), kind: KindAdded},
			},
		},
		{
			name:   "custom ignored path option",
			before: chainsDoc(view("0xabc", "Router", map[string]any{"owner": "0x1"})),
			after:  chainsDoc(view("0xabc", "Router", map[string]any{"owner": "0x2"})),
			opts:   []Option{WithIgnoredPaths("owner")},
		},
		{
			name:   "custom ignored path does not suppress siblings",
			before: chainsDoc(view("0xabc", "Router", map[string]any{"cfg": map[string]any{"owner": "0x1", "other": float64(1)}})),
			after:  chainsDoc(view("0xabc", "Router", map[string]any{"cfg": map[string]any{"owner": "0x2", "other": float64(2)}})),
			opts:   []Option{WithIgnoredPaths("cfg.owner")},
			wantChanges: []changeProjection{
				{path: "chains." + strconv.FormatUint(sel, 10) + ".0xabc.cfg.other", kind: KindChanged},
			},
		},
		{
			name:   "top-level _meta appearing on one side only is not a change",
			before: chainsDoc(view("0xabc", "Router", map[string]any{"owner": "0x1"})),
			after: func() map[string]any {
				d := chainsDoc(view("0xabc", "Router", map[string]any{"owner": "0x1"}))
				d["_meta"] = map[string]any{"totalDuration": "2s"}

				return d
			}(),
		},
		{
			name: "nested _meta inside a view is excluded, siblings still diff",
			before: chainsDoc(view("0xabc", "Router", map[string]any{
				"_meta": map[string]any{"generatedAt": "1"}, "owner": "0x1",
			})),
			after: chainsDoc(view("0xabc", "Router", map[string]any{
				"_meta": map[string]any{"generatedAt": "2"}, "owner": "0x2",
			})),
			wantChanges: []changeProjection{
				{path: "chains." + strconv.FormatUint(sel, 10) + ".0xabc.owner", kind: KindChanged},
			},
		},
		{
			name: "chain removed",
			before: map[string]any{"chains": map[string]any{
				strconv.FormatUint(sel, 10):      []any{view("0xabc", "Router", map[string]any{"owner": "0x1"})},
				strconv.FormatUint(otherSel, 10): []any{view("0xdef", "Router", map[string]any{"owner": "0x2"})},
			}},
			after: map[string]any{"chains": map[string]any{
				strconv.FormatUint(sel, 10): []any{view("0xabc", "Router", map[string]any{"owner": "0x1"})},
			}},
			wantChanges: []changeProjection{
				{path: "chains." + strconv.FormatUint(otherSel, 10), kind: KindRemoved},
			},
		},
		{
			name:   "object key added",
			before: chainsDoc(view("0xabc", "Router", map[string]any{"a": float64(1)})),
			after:  chainsDoc(view("0xabc", "Router", map[string]any{"a": float64(1), "b": map[string]any{"c": float64(2)}})),
			wantChanges: []changeProjection{
				{path: "chains." + strconv.FormatUint(sel, 10) + ".0xabc.b", kind: KindAdded},
			},
		},
		{
			name:   "object key removed",
			before: chainsDoc(view("0xabc", "Router", map[string]any{"a": float64(1), "b": map[string]any{"c": float64(2)}})),
			after:  chainsDoc(view("0xabc", "Router", map[string]any{"a": float64(1)})),
			wantChanges: []changeProjection{
				{path: "chains." + strconv.FormatUint(sel, 10) + ".0xabc.b", kind: KindRemoved},
			},
		},
		{
			name: "*_error present on the before side only is Unreadable, not a change",
			before: chainsDoc(view("0xabc", "Router", map[string]any{
				"owner": "0x1", "owner_error": "timeout",
			})),
			after:          chainsDoc(view("0xabc", "Router", map[string]any{"owner": "0x2"})),
			wantUnreadable: []Unreadable{{Address: "0xabc", Reason: ReasonReadError}},
		},
		{
			name: "*_error on both sides with different messages is Unreadable, not equal state",
			before: chainsDoc(view("0xabc", "Router", map[string]any{
				"owner": "0x1", "owner_error": "timeout",
			})),
			after: chainsDoc(view("0xabc", "Router", map[string]any{
				"owner": "0x1", "owner_error": "reverted",
			})),
			wantUnreadable: []Unreadable{{Address: "0xabc", Reason: ReasonReadError}},
		},
		{
			name: "identical *_error on both sides is still Unreadable",
			before: chainsDoc(view("0xabc", "Router", map[string]any{
				"owner": "0x1", "owner_error": "timeout",
			})),
			after: chainsDoc(view("0xabc", "Router", map[string]any{
				"owner": "0x1", "owner_error": "timeout",
			})),
			wantUnreadable: []Unreadable{{Address: "0xabc", Reason: ReasonReadError}},
		},
		{
			name:           "_skipped on both sides yields a single deduped entry",
			before:         skippedDoc(),
			after:          skippedDoc(),
			wantUnreadable: []Unreadable{{Address: "0xabc", Reason: ReasonViewSkipped}},
		},
		{
			name: "keyed array element field changed",
			before: chainsDoc(view("0xabc", "TokenAdminRegistry", map[string]any{
				"tokens": []any{map[string]any{"address": "0x1", "rate": float64(5)}},
			})),
			after: chainsDoc(view("0xabc", "TokenAdminRegistry", map[string]any{
				"tokens": []any{map[string]any{"address": "0x1", "rate": float64(6)}},
			})),
			wantChanges: []changeProjection{
				{path: "chains." + strconv.FormatUint(sel, 10) + ".0xabc.tokens[0x1].rate", kind: KindChanged},
			},
		},
		{
			name:   "positional array scalar changed",
			before: chainsDoc(view("0xabc", "OnRamp", map[string]any{"f": []any{float64(1), float64(2)}})),
			after:  chainsDoc(view("0xabc", "OnRamp", map[string]any{"f": []any{float64(1), float64(3)}})),
			wantChanges: []changeProjection{
				{path: "chains." + strconv.FormatUint(sel, 10) + ".0xabc.f[1]", kind: KindChanged},
			},
		},
		{
			name:   "same address with a changed _type stamp yields added and removed",
			before: chainsDoc(view("0xabc", "Router", map[string]any{"owner": "0x1"})),
			after:  chainsDoc(view("0xabc", "FeeQuoter", map[string]any{"owner": "0x1"})),
			wantChanges: []changeProjection{
				{path: "chains." + strconv.FormatUint(sel, 10) + ".0xabc", kind: KindAdded},
				{path: "chains." + strconv.FormatUint(sel, 10) + ".0xabc", kind: KindRemoved},
			},
		},
		{
			name: "address-only view map is keyed and diffed like any other view",
			before: chainsDoc(
				map[string]any{"address": "0xabc"},
				view("0xdef", "Router", map[string]any{"owner": "0x1"}),
			),
			after: chainsDoc(
				map[string]any{"address": "0xabc"},
				view("0xdef", "Router", map[string]any{"owner": "0x2"}),
			),
			wantChanges: []changeProjection{
				{path: "chains." + strconv.FormatUint(sel, 10) + ".0xdef.owner", kind: KindChanged},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			changes, unreadable, err := Diff(rawDoc(t, tt.before), rawDoc(t, tt.after), tt.opts...)
			require.NoError(t, err)
			require.Equal(t, tt.wantChanges, projectChanges(t, changes))
			// Unreadables are emitted by map-ordered sections, so compare as
			// multisets of (address, reason), not by index.
			got := make([]string, 0, len(unreadable))
			for _, u := range unreadable {
				got = append(got, u.Address+"|"+u.Reason)
			}
			want := make([]string, 0, len(tt.wantUnreadable))
			for _, u := range tt.wantUnreadable {
				want = append(want, u.Address+"|"+u.Reason)
			}
			slices.Sort(got)
			slices.Sort(want)
			require.Equal(t, want, got)
		})
	}
}

func TestDiffJSON(t *testing.T) {
	t.Parallel()

	before, err := json.Marshal(chainsDoc(view("0xabc", "Router", map[string]any{"owner": "0x1"})))
	require.NoError(t, err)
	after, err := json.Marshal(chainsDoc(view("0xabc", "Router", map[string]any{"owner": "0x2"})))
	require.NoError(t, err)

	changes, _, err := DiffJSON(before, after)
	require.NoError(t, err)
	require.Len(t, changes, 1)
	require.Equal(t, KindChanged, changes[0].Kind)
	require.Equal(t, "0x1", changes[0].Before)
	require.Equal(t, "0x2", changes[0].After)
}

func TestDiffJSON_MalformedInput(t *testing.T) {
	t.Parallel()

	_, _, err := DiffJSON([]byte("{not json"), []byte("{}"))
	require.ErrorContains(t, err, "parsing before document")
}

func TestDiff_MalformedViewElement(t *testing.T) {
	t.Parallel()

	// Hand-built raw documents: the table builders cannot express a view
	// element that is not a JSON object. The after side's second element is
	// the number 42 — unparseable as a view, so Unreadable — while the other
	// views still diff.
	selStr := strconv.FormatUint(sel, 10)
	before := map[string]json.RawMessage{"chains": json.RawMessage(
		`{"` + selStr + `":[{"address":"0xabc","_type":"Router","owner":"0x1"},{"address":"0xdef","_type":"Router","owner":"0x2"}]}`)}
	after := map[string]json.RawMessage{"chains": json.RawMessage(
		`{"` + selStr + `":[{"address":"0xabc","_type":"Router","owner":"0x3"},42]}`)}

	changes, unreadable, err := Diff(before, after)
	require.NoError(t, err)

	require.Equal(t, []changeProjection{
		{path: "chains." + selStr + ".0xabc.owner", kind: KindChanged},
		{path: "chains." + selStr + ".0xdef", kind: KindRemoved},
	}, projectChanges(t, changes))
	require.Equal(t, []Unreadable{{ChainSelector: sel, Reason: ReasonMalformedJSON}}, unreadable)
}

func TestDiff_MalformedChainViews(t *testing.T) {
	t.Parallel()

	// The second chain's views section is a JSON object, not an array: the
	// chain is one Unreadable finding; the healthy chain still diffs.
	selStr := strconv.FormatUint(sel, 10)
	otherStr := strconv.FormatUint(otherSel, 10)
	before := map[string]json.RawMessage{"chains": json.RawMessage(
		`{"` + selStr + `":[{"address":"0xabc","_type":"Router","owner":"0x1"}],"` + otherStr + `":{"not":"an array"}}`)}
	after := map[string]json.RawMessage{"chains": json.RawMessage(
		`{"` + selStr + `":[{"address":"0xabc","_type":"Router","owner":"0x2"}],"` + otherStr + `":{"not":"an array"}}`)}

	changes, unreadable, err := Diff(before, after)
	require.NoError(t, err)

	require.Equal(t, []changeProjection{
		{path: "chains." + selStr + ".0xabc.owner", kind: KindChanged},
	}, projectChanges(t, changes))
	require.Equal(t, []Unreadable{{ChainSelector: otherSel, Reason: ReasonMalformedJSON}}, unreadable)
}

// TestDiffReplay locks the Step 0 pinned replay ground truth (proposal
// 1787082290466662, fork pinned to block 25784394): exactly the five genuine
// mova-lane creation effects, the CapabilitiesRegistry view quarantined by
// its post-side dons_error, rate-limiter stamps suppressed, and the H2 MCMS
// config churn suppressed. If this test fails after a differ change, the
// change regressed one of those rules — reconcile against the fixture, do
// not relax the assertion.
func TestDiffReplay(t *testing.T) {
	t.Parallel()

	pre, err := os.ReadFile("testdata/replay-pinned-pre.json")
	require.NoError(t, err)
	post, err := os.ReadFile("testdata/replay-pinned-post.json")
	require.NoError(t, err)

	changes, unreadable, err := DiffJSON(pre, post)
	require.NoError(t, err)

	const (
		offRampAddr     = "0x26d3681dfc9e4c8c79cfbf461adec8a21d5d73c5"
		feeQuoterAddr   = "0x300f2ca3e3867133baea866c89096f097d57bf57"
		routerAddr      = "0x80226fc0ee2b096224eeac085bb9a8cba1146f7d"
		onRampAddr      = "0x913814782144864e523c3fdb78e3ca25d2c2aeca"
		capRegistryAddr = "0x006bc1f599a10b73c88cc3cd19a92829c4ac1e83"
		callProxyAddr   = "0xa85a03f8c1808b1ce52b7dd72f738306d3f23ea6"
	)

	// The five genuine effects, as (lowercased address, path fragment) pairs.
	wantEffects := map[string]bool{
		offRampAddr + "|sourceChainConfigs":       true,
		feeQuoterAddr + "|destinationChainConfig": true,
		routerAddr + "|onRamps":                   true,
		routerAddr + "|offRamps":                  true,
		onRampAddr + "|destChainSpecificData":     true,
	}
	require.Len(t, changes, len(wantEffects))

	unreadableAddrs := map[string]struct{}{}
	for _, u := range unreadable {
		unreadableAddrs[strings.ToLower(u.Address)] = struct{}{}
	}

	for _, c := range changes {
		require.Equal(t, KindAdded, c.Kind, "unexpected kind at %s", c.Path)
		require.NotContains(t, unreadableAddrs, strings.ToLower(c.Address),
			"a diffed view must never also be Unreadable: %s", c.Path)
		// Suppression anchors: no rate-limiter stamp or MCMS config churn may
		// survive as a change.
		for _, banned := range []string{"lastUpdated", "signers", "groupQuorums", "groupParents"} {
			require.NotContains(t, c.Path, banned, "suppressed path survived: %s", c.Path)
		}
		addr := strings.ToLower(c.Address)
		matched := false
		for _, frag := range []string{"sourceChainConfigs", "destinationChainConfig", "onRamps", "offRamps", "destChainSpecificData"} {
			key := addr + "|" + frag
			if wantEffects[key] && strings.Contains(c.Path, frag) {
				delete(wantEffects, key)
				matched = true

				break
			}
		}
		require.True(t, matched, "unexpected change: %s %s", c.ContractType, c.Path)
	}
	require.Empty(t, wantEffects, "expected genuine effects missing from the diff")

	// 64 Unreadable, fixture-verified: 62 distinct views carrying *_error keys
	// at any depth (61 at the view root — the CapabilitiesRegistry dons_error
	// among them — plus one TokenAdminRegistry with a nested-only error key),
	// and the 2 _skipped coverage entries (the CallProxy among them).
	require.Len(t, unreadable, 64)
	byReason := map[string]int{}
	for _, u := range unreadable {
		byReason[u.Reason]++
	}
	for r, n := range byReason {
		t.Logf("unreadable reason %q: %d", r, n)
	}
	require.Equal(t, 62, byReason[ReasonReadError])
	require.Equal(t, 2, byReason[ReasonViewSkipped])

	var capRegistryQuarantined, callProxySkipped bool
	for _, u := range unreadable {
		switch {
		case u.Reason == ReasonReadError && strings.EqualFold(u.Address, capRegistryAddr):
			capRegistryQuarantined = true
		case u.Reason == ReasonViewSkipped && strings.EqualFold(u.Address, callProxyAddr):
			callProxySkipped = true
		}
	}
	require.True(t, capRegistryQuarantined, "CapabilitiesRegistry dons_error quarantine missing")
	require.True(t, callProxySkipped, "CallProxy _skipped entry missing")
}
