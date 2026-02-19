package helper

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStringToBigIntIfOverflowInt64(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		wantOK bool
		want   string
	}{
		{"fits int64", "12345", false, ""},
		{"fits uint64 but not int64", "18446744073709551615", false, ""},
		{"overflows uint64", "18446744073709551616", true, "18446744073709551616"},
		{"very large number", "115792089237316195423570985008687907853269984665640564039457584007913129639935", true, "115792089237316195423570985008687907853269984665640564039457584007913129639935"},
		{"non-numeric string", "hello", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := stringToBigIntIfOverflowInt64(tt.input)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				require.NotNil(t, got)
				assert.Equal(t, tt.want, got.String())
			}
		})
	}
}

func TestCoerceBigIntStrings(t *testing.T) {
	bigVal := "18446744073709551616" // overflows uint64
	expectedBig, _ := new(big.Int).SetString(bigVal, 10)

	t.Run("nil and scalars pass through", func(t *testing.T) {
		assert.Nil(t, CoerceBigIntStrings(nil))
		assert.Equal(t, 42, CoerceBigIntStrings(42))
		assert.Equal(t, "12345", CoerceBigIntStrings("12345"))
	})

	t.Run("overflow string converted to big.Int", func(t *testing.T) {
		result := CoerceBigIntStrings(bigVal)
		bi, ok := result.(*big.Int)
		require.True(t, ok, "expected *big.Int, got %T", result)
		assert.Equal(t, expectedBig, bi)
	})

	t.Run("map with mixed values", func(t *testing.T) {
		input := map[string]any{
			"name":  "test",
			"value": bigVal,
			"count": 5,
		}
		result := CoerceBigIntStrings(input).(map[string]any)
		assert.Equal(t, "test", result["name"])
		assert.Equal(t, expectedBig, result["value"])
		assert.Equal(t, 5, result["count"])
	})

	t.Run("nested maps and slices", func(t *testing.T) {
		input := map[string]any{
			"level1": map[string]any{
				"level2": []any{
					map[string]any{"level3": bigVal},
				},
			},
		}
		result := CoerceBigIntStrings(input).(map[string]any)
		l1 := result["level1"].(map[string]any)
		l2 := l1["level2"].([]any)
		l3 := l2[0].(map[string]any)
		assert.Equal(t, expectedBig, l3["level3"])
	})

	t.Run("slice of typed maps", func(t *testing.T) {
		input := []map[string]any{
			{"key": bigVal},
			{"key": "small"},
		}
		result := CoerceBigIntStrings(input).([]map[string]any)
		assert.Equal(t, expectedBig, result[0]["key"])
		assert.Equal(t, "small", result[1]["key"])
	})
}

func TestCoerceBigIntStringsForKeys(t *testing.T) {
	bigVal := "18446744073709551616"
	expectedBig, _ := new(big.Int).SetString(bigVal, 10)

	t.Run("matchFunc controls conversion", func(t *testing.T) {
		input := map[string]any{"target": bigVal, "other": bigVal}

		// matchAll converts everything
		result := CoerceBigIntStringsForKeys(input, func(string) bool { return true }).(map[string]any)
		assert.Equal(t, expectedBig, result["target"])

		// matchNone keeps strings
		input = map[string]any{"target": bigVal}
		result = CoerceBigIntStringsForKeys(input, func(string) bool { return false }).(map[string]any)
		assert.Equal(t, bigVal, result["target"])
	})

	t.Run("selective key matching", func(t *testing.T) {
		matchTarget := func(key string) bool { return key == ".target" }
		input := map[string]any{"target": bigVal, "other": bigVal}
		result := CoerceBigIntStringsForKeys(input, matchTarget).(map[string]any)
		assert.Equal(t, expectedBig, result["target"])
		assert.Equal(t, bigVal, result["other"])
	})

	t.Run("nested and list key paths", func(t *testing.T) {
		matchNested := func(key string) bool { return key == ".outer.inner" }
		input := map[string]any{
			"outer": map[string]any{"inner": bigVal, "skip": bigVal},
		}
		result := CoerceBigIntStringsForKeys(input, matchNested).(map[string]any)
		inner := result["outer"].(map[string]any)
		assert.Equal(t, expectedBig, inner["inner"])
		assert.Equal(t, bigVal, inner["skip"])

		matchList := func(key string) bool { return key == ".items.[].val" }
		input2 := map[string]any{
			"items": []map[string]any{{"val": bigVal}},
		}
		result2 := CoerceBigIntStringsForKeys(input2, matchList).(map[string]any)
		items := result2["items"].([]map[string]any)
		assert.Equal(t, expectedBig, items[0]["val"])
	})
}

func TestDefaultMatchKeysToFix(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		match bool
	}{
		{
			"contractparamsperchain maxfeejuelspermsg",
			".[].0006_deploy_evm_chain_contracts.payload.contractparamsperchain.7759470850252068959.feequoterparams.maxfeejuelspermsg",
			true,
		},
		{
			"chains maxfeejuelspermsg",
			".[].0009_deploy_ton_ccip_contracts.payload.chains.13879075125137744094.maxfeejuelspermsg",
			true,
		},
		{
			"tokenprices",
			".[].0016_configure_lanes.payload.lanes.[].chaina.tokenprices.0xb1D4538B4571d411F07960EF2838Ce337FE1E80E",
			true,
		},
		{
			"gasmultiplierweipereth",
			".[].0016_configure_lanes.payload.lanes.[].chainb.feequoterdestchainconfig.gasmultiplierweipereth",
			true,
		},
		{
			"unrelated key",
			".some.random.key",
			false,
		},
		{
			"chains without digit selector",
			".[].0009_deploy_ton_ccip_contracts.payload.chains.notadigit.maxfeejuelspermsg",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.match, DefaultMatchKeysToFix(tt.key))
		})
	}
}