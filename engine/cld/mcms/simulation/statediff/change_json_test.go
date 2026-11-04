package statediff

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChangeUnmarshalRequiresExplicitKind(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{
		`{}`, `null`, `{"kind":null}`, `{"kind":""}`,
		`{"kind":"unknown"}`, `{"kind":0}`, `{"kind":false}`,
	} {
		t.Run(raw, func(t *testing.T) {
			t.Parallel()
			change := Change{Kind: KindRemoved, Before: "original"}
			require.ErrorContains(t, json.Unmarshal([]byte(raw), &change), "kind")
			require.Equal(t, Change{Kind: KindRemoved, Before: "original"}, change,
				"a failed decode must not overwrite the previous change")
		})
	}
}

func TestChangeUnmarshalPreservesExactValues(t *testing.T) {
	t.Parallel()
	var change Change
	require.NoError(t, json.Unmarshal([]byte(`{
		"kind":"changed","chainSelector":18446744073709551615,
		"before":{"value":1100000000000000000},
		"after":[1100000000000000001,115792089237316195423570985008687907853269984665640564039457584007913129639935]
	}`), &change))
	require.Equal(t, ^uint64(0), change.ChainSelector)
	require.Equal(t, KindChanged, change.Kind)
	require.Equal(t, json.Number("1100000000000000000"), change.Before.(map[string]any)["value"])
	require.Equal(t, []any{
		json.Number("1100000000000000001"),
		json.Number("115792089237316195423570985008687907853269984665640564039457584007913129639935"),
	}, change.After)
}

func TestChangeNullSidesRoundTrip(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{
		`{"kind":"added","after":null}`, `{"kind":"removed","before":null}`,
		`{"kind":"added"}`, `{"kind":"removed"}`,
		`{"kind":"changed","before":null,"after":null}`,
	} {
		t.Run(raw, func(t *testing.T) {
			t.Parallel()
			var change Change
			require.NoError(t, json.Unmarshal([]byte(raw), &change))
			require.Nil(t, change.Before)
			require.Nil(t, change.After)
			encoded, err := json.Marshal(change)
			require.NoError(t, err)
			var loaded Change
			require.NoError(t, json.Unmarshal(encoded, &loaded))
			require.Equal(t, change, loaded)
		})
	}
}
