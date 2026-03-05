package ccip

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/smartcontractkit/chainlink-ccip/chains/evm/gobindings/generated/latest/token_pool"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
)

type ChainUpdateValue struct {
	RemoteChainSelector uint64
	Label               string
}

func (v ChainUpdateValue) String() string {
	return v.Label
}

func ParseChainUpdates(params analyzer.DecodedParameters) ([]token_pool.TokenPoolChainUpdate, error) {
	for _, param := range params {
		if param.Name() != "chainsToAdd" && param.Name() != "chains" {
			continue
		}

		raw := param.RawValue()
		if raw == nil {
			continue
		}

		result := abi.ConvertType(raw, new([]token_pool.TokenPoolChainUpdate))

		converted, ok := result.(*[]token_pool.TokenPoolChainUpdate)
		if !ok {
			return nil, fmt.Errorf("parameter %q: abi.ConvertType returned %T, expected *[]TokenPoolChainUpdate", param.Name(), result)
		}

		return *converted, nil
	}

	return nil, nil
}
