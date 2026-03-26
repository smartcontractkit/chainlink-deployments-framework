package tokenpool

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/smartcontractkit/chainlink-ccip/chains/evm/gobindings/generated/latest/token_pool"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/examples/ccip"
)

func extractChainUpdateParams(call analyzer.DecodedCall) ([]token_pool.TokenPoolChainUpdate, []uint64, error) {
	chainsToAdd, err := ccip.ParseChainUpdates(call.Inputs())
	if err != nil {
		return nil, nil, err
	}

	var selectorsToRemove []uint64

	for _, param := range call.Inputs() {
		if param.Name() != "remoteChainSelectorsToRemove" {
			continue
		}

		raw := param.RawValue()
		if raw == nil {
			continue
		}

		result := abi.ConvertType(raw, new([]uint64))

		converted, ok := result.(*[]uint64)
		if !ok {
			return nil, nil, fmt.Errorf("parameter %q: abi.ConvertType returned %T, expected *[]uint64", param.Name(), result)
		}

		selectorsToRemove = *converted
	}

	return chainsToAdd, selectorsToRemove, nil
}
