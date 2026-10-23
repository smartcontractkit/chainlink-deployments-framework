package statediff

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func numericDocument(value string) []byte {
	return []byte(`{"chains":{"1":[{"address":"0xabc","_type":"FeeQuoter","_requestedVersion":"1.6.0","value":` + value + `}]}}`)
}

func TestDiffExactNumbers(t *testing.T) {
	t.Parallel()
	for _, values := range [][2]string{
		{"9007199254740992", "9007199254740993"},
		{"1100000000000000000", "1100000000000000001"},
		{"115792089237316195423570985008687907853269984665640564039457584007913129639934", "115792089237316195423570985008687907853269984665640564039457584007913129639935"},
	} {
		t.Run(values[0], func(t *testing.T) {
			t.Parallel()
			changes, unreadable, err := DiffJSON(numericDocument(values[0]), numericDocument(values[1]))
			require.NoError(t, err)
			require.Empty(t, unreadable)
			require.Len(t, changes, 1)
			require.Equal(t, json.Number(values[0]), changes[0].Before)
			require.Equal(t, json.Number(values[1]), changes[0].After)
		})
	}
	for _, values := range [][2]string{{"1", "1.0"}, {"-0", "0.0e12"}, {"100", "1e2"}, {"1.23e-7", "0.000000123"}, {"1e1000000000", "10e999999999"}} {
		changes, unreadable, err := DiffJSON(numericDocument(values[0]), numericDocument(values[1]))
		require.NoError(t, err)
		require.Empty(t, changes, "%s and %s are equivalent", values[0], values[1])
		require.Empty(t, unreadable)
	}
}

func TestDecodeJSONRejectsTrailingValues(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{`{} {}`, `{} garbage`, `{"value":01}`} {
		var value any
		require.Error(t, DecodeJSON([]byte(raw), &value))
	}
	var value map[string]any
	require.NoError(t, DecodeJSON([]byte("{\"value\":9007199254740993}\n "), &value))
	require.Equal(t, json.Number("9007199254740993"), value["value"])
}

func TestEqualJSONValuesTypedArtifacts(t *testing.T) {
	t.Parallel()
	type artifact struct {
		Values []any `json:"values"`
	}
	a := artifact{Values: []any{json.Number("9007199254740993"), uint64(100)}}
	b := artifact{Values: []any{json.Number("9007199254740993.0"), json.Number("1e2")}}
	require.True(t, EqualJSONValues(a, &b))
	b.Values[0] = json.Number("9007199254740992")
	require.False(t, EqualJSONValues(a, b))
	require.False(t, EqualJSONValues(map[string]any(nil), map[string]any{}))
	require.False(t, EqualJSONValues([]any(nil), []any{}))
}

func TestDiffQuarantinesEntireReadErrorView(t *testing.T) {
	t.Parallel()
	good := view("0xabc", "Router", map[string]any{"owner": "0xold"})
	for _, fields := range []map[string]any{
		{"owner": "0xnew", "cfg": map[string]any{"owner_error": "timeout"}},
		{"owner": "0xnew", "tokens": []any{map[string]any{"address": "0xtoken", "name_error": "timeout"}}},
		{"owner_error": "timeout"},
	} {
		failed := view("0xabc", "Router", fields)
		for _, pair := range [][2]map[string]any{
			{chainsDoc(good), chainsDoc(failed)},
			{chainsDoc(failed), chainsDoc(good)},
			{chainsDoc(), chainsDoc(failed)},
			{chainsDoc(failed), chainsDoc()},
			{{}, chainsDoc(failed)},
			{chainsDoc(failed), {}},
		} {
			changes, unreadable, err := Diff(rawDoc(t, pair[0]), rawDoc(t, pair[1]))
			require.NoError(t, err)
			require.Empty(t, changes)
			require.Equal(t, []Unreadable{{ChainSelector: sel, Address: "0xabc", ContractType: "Router", Version: "1.6.0", Reason: ReasonReadError}}, unreadable)
		}
	}
}

func TestDiffCoverageIsScopedToChain(t *testing.T) {
	t.Parallel()
	before := []byte(`{"chains":{"1":[{"address":"0xabc","value":1}],"2":[{"address":"0xabc","value":1}]},"_skipped":[{"chainSelector":"1","address":"0xABC","type":"Router","version":"1.2.0"}]}`)
	after := []byte(`{"chains":{"1":[{"address":"0xabc","value":2}],"2":[{"address":"0xabc","value":2}]}}`)
	changes, unreadable, err := DiffJSON(before, after)
	require.NoError(t, err)
	require.Len(t, unreadable, 1)
	require.Len(t, changes, 1)
	require.Equal(t, uint64(2), changes[0].ChainSelector)
}

func TestDiffArrayIdentity(t *testing.T) {
	t.Parallel()
	t.Run("full width selector remains exact", func(t *testing.T) {
		t.Parallel()
		before := numericDocument(`[{"chainSelector":5009297550715157269,"n":1},{"chainSelector":5009297550715157270,"n":2}]`)
		after := numericDocument(`[{"chainSelector":5009297550715157270,"n":2},{"chainSelector":5009297550715157269,"n":3}]`)
		changes, unreadable, err := DiffJSON(before, after)
		require.NoError(t, err)
		require.Empty(t, unreadable)
		require.Len(t, changes, 1)
		require.Equal(t, "chains.1.0xabc.value[5009297550715157269].n", changes[0].Path)
	})
	t.Run("duplicate identities cannot discard an element", func(t *testing.T) {
		t.Parallel()
		before := numericDocument(`[{"address":"0x1","n":1},{"address":"0x1","n":2}]`)
		after := numericDocument(`[{"address":"0x1","n":3},{"address":"0x1","n":2}]`)
		changes, unreadable, err := DiffJSON(before, after)
		require.NoError(t, err)
		require.Empty(t, unreadable)
		require.Len(t, changes, 1)
		require.Equal(t, "chains.1.0xabc.value[0].n", changes[0].Path)
	})
	t.Run("invalid numeric selectors quarantine the view", func(t *testing.T) {
		t.Parallel()
		for _, selector := range []string{"-1", "1.5", "18446744073709551616", "1e1000000000"} {
			changes, unreadable, err := DiffJSON(numericDocument(`[]`), numericDocument(`[{"chainSelector":`+selector+`}]`))
			require.NoError(t, err)
			require.Empty(t, changes)
			require.Len(t, unreadable, 1)
			require.Equal(t, Unreadable{ChainSelector: 1, Address: "0xabc", ContractType: "FeeQuoter", Version: "1.6.0", Reason: ReasonMalformedJSON}, unreadable[0])
		}
	})
}

func TestDiffCoveragePreservesCaseSensitiveAddresses(t *testing.T) {
	t.Parallel()
	before := []byte(`{"chains":{"1":[{"address":"Base58","n":1},{"address":"base58","n":1}]},"_errors":[{"chainSelector":1,"address":"Base58"}]}`)
	after := []byte(`{"chains":{"1":[{"address":"Base58","n":2},{"address":"base58","n":2}]}}`)
	changes, unreadable, err := DiffJSON(before, after)
	require.NoError(t, err)
	require.Len(t, unreadable, 1)
	require.Len(t, changes, 1)
	require.Equal(t, "base58", changes[0].Address)
}
