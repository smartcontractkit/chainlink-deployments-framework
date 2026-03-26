package tokenpool

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/smartcontractkit/chainlink-ccip/chains/evm/gobindings/generated/latest/token_pool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
)

func TestExtractChainUpdateParams(t *testing.T) {
	t.Parallel()

	chainUpdate := token_pool.TokenPoolChainUpdate{
		RemoteChainSelector: 9027416829622342829,
		OutboundRateLimiterConfig: token_pool.RateLimiterConfig{
			IsEnabled: true,
			Capacity:  big.NewInt(5_000_000_000_000_000_000),
			Rate:      big.NewInt(231_480_000_000_000),
		},
		InboundRateLimiterConfig: token_pool.RateLimiterConfig{
			IsEnabled: true,
			Capacity:  big.NewInt(5_000_000_000_000_000_000),
			Rate:      big.NewInt(231_480_000_000_000),
		},
	}

	t.Run("chainsToAdd parameter", func(t *testing.T) {
		t.Parallel()

		call := &stubConvertCall{
			inputs: analyzer.DecodedParameters{
				&stubConvertParam{name: "chainsToAdd", rawValue: []token_pool.TokenPoolChainUpdate{chainUpdate}},
				&stubConvertParam{name: "remoteChainSelectorsToRemove", rawValue: []uint64{}},
			},
		}

		adds, removes, err := extractChainUpdateParams(call)
		require.NoError(t, err)
		assert.Len(t, adds, 1)
		assert.Equal(t, uint64(9027416829622342829), adds[0].RemoteChainSelector)
		assert.Empty(t, removes)
	})

	t.Run("chains parameter - alt", func(t *testing.T) {
		t.Parallel()

		call := &stubConvertCall{
			inputs: analyzer.DecodedParameters{
				&stubConvertParam{name: "chains", rawValue: []token_pool.TokenPoolChainUpdate{chainUpdate}},
			},
		}

		adds, _, err := extractChainUpdateParams(call)
		require.NoError(t, err)
		assert.Len(t, adds, 1)
	})

	t.Run("removal selectors", func(t *testing.T) {
		t.Parallel()

		call := &stubConvertCall{
			inputs: analyzer.DecodedParameters{
				&stubConvertParam{name: "chainsToAdd", rawValue: []token_pool.TokenPoolChainUpdate{}},
				&stubConvertParam{name: "remoteChainSelectorsToRemove", rawValue: []uint64{100, 200}},
			},
		}

		adds, removes, err := extractChainUpdateParams(call)
		require.NoError(t, err)
		assert.Empty(t, adds)
		assert.Equal(t, []uint64{100, 200}, removes)
	})

	t.Run("nil rawValue skipped", func(t *testing.T) {
		t.Parallel()

		call := &stubConvertCall{
			inputs: analyzer.DecodedParameters{
				&stubConvertParam{name: "chainsToAdd", rawValue: nil},
			},
		}

		adds, removes, err := extractChainUpdateParams(call)
		require.NoError(t, err)
		assert.Empty(t, adds)
		assert.Empty(t, removes)
	})

	t.Run("no matching params returns empty", func(t *testing.T) {
		t.Parallel()

		call := &stubConvertCall{
			inputs: analyzer.DecodedParameters{
				&stubConvertParam{name: "unknownField", rawValue: "something"},
			},
		}

		adds, removes, err := extractChainUpdateParams(call)
		require.NoError(t, err)
		assert.Empty(t, adds)
		assert.Empty(t, removes)
	})
}

type stubConvertCall struct {
	inputs analyzer.DecodedParameters
}

func (s *stubConvertCall) To() string                         { return "" }
func (s *stubConvertCall) Name() string                       { return "applyChainUpdates" }
func (s *stubConvertCall) Inputs() analyzer.DecodedParameters { return s.inputs }
func (s *stubConvertCall) Outputs() analyzer.DecodedParameters {
	return nil
}
func (s *stubConvertCall) Data() []byte                      { return nil }
func (s *stubConvertCall) AdditionalFields() json.RawMessage { return nil }
func (s *stubConvertCall) ContractType() string              { return "BurnMintTokenPool" }
func (s *stubConvertCall) ContractVersion() string           { return "1.5.1" }

var _ analyzer.DecodedCall = (*stubConvertCall)(nil)

type stubConvertParam struct {
	name     string
	rawValue any
}

func (s *stubConvertParam) Name() string  { return s.name }
func (s *stubConvertParam) Type() string  { return "" }
func (s *stubConvertParam) Value() any    { return s.rawValue }
func (s *stubConvertParam) RawValue() any { return s.rawValue }

var _ analyzer.DecodedParameter = (*stubConvertParam)(nil)
