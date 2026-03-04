package tokenpool

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/smartcontractkit/chainlink-ccip/chains/evm/gobindings/generated/latest/token_pool"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/decoder"
)

func extractChainUpdateParams(call decoder.DecodedCall) ([]token_pool.TokenPoolChainUpdate, []uint64, error) {
	var chainsToAdd []token_pool.TokenPoolChainUpdate
	var selectorsToRemove []uint64

	for _, param := range call.Inputs() {
		raw := param.RawValue()
		if raw == nil {
			continue
		}

		switch param.Name() {
		case "chainsToAdd", "chains":
			result := abi.ConvertType(raw, new([]token_pool.TokenPoolChainUpdate))

			converted, ok := result.(*[]token_pool.TokenPoolChainUpdate)
			if !ok {
				return nil, nil, fmt.Errorf("parameter %q: abi.ConvertType returned %T, expected *[]TokenPoolChainUpdate", param.Name(), result)
			}

			chainsToAdd = *converted
		case "remoteChainSelectorsToRemove":
			result := abi.ConvertType(raw, new([]uint64))

			converted, ok := result.(*[]uint64)
			if !ok {
				return nil, nil, fmt.Errorf("parameter %q: abi.ConvertType returned %T, expected *[]uint64", param.Name(), result)
			}

			selectorsToRemove = *converted
		}
	}

	return chainsToAdd, selectorsToRemove, nil
}
