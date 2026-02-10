package ccip

import (
	"fmt"

	"github.com/smartcontractkit/chainlink-ccip/chains/evm/gobindings/generated/latest/token_pool"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/analyzer"
	evmutil "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/analyzer/analyzers/evm"
)

func extractChainUpdateParams(call analyzer.DecodedCall) ([]token_pool.TokenPoolChainUpdate, []uint64, error) {
	var chainsToAdd []token_pool.TokenPoolChainUpdate
	var selectorsToRemove []uint64

	for _, param := range call.Inputs() {
		switch param.Name {
		case "chainsToAdd", "chains":
			updates, err := evmutil.ConvertParamSlice[token_pool.TokenPoolChainUpdate](param)
			if err != nil {
				return nil, nil, fmt.Errorf("parse %s: %w", param.Name, err)
			}

			chainsToAdd = updates
		case "remoteChainSelectorsToRemove":
			selectors, err := evmutil.ConvertParamSlice[uint64](param)
			if err != nil {
				return nil, nil, fmt.Errorf("parse remoteChainSelectorsToRemove: %w", err)
			}

			selectorsToRemove = selectors
		}
	}

	return chainsToAdd, selectorsToRemove, nil
}
