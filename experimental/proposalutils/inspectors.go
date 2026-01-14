package proposalutils

import (
	"fmt"

	chainsel "github.com/smartcontractkit/chain-selectors"

	cldfChain "github.com/smartcontractkit/chainlink-deployments-framework/chain"

	"github.com/smartcontractkit/mcms/sdk/aptos"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/solana"

	"github.com/smartcontractkit/mcms/sdk"
	mcmsTypes "github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/proposalutils/chainsmetadata"
)

type InspectorFetcher interface {
	FetchInspectors(chainMetadata map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata, action mcmsTypes.TimelockAction) (map[mcmsTypes.ChainSelector]sdk.Inspector, error)
}

var _ InspectorFetcher = (*MCMInspectorFetcher)(nil)

type MCMInspectorFetcher struct {
	chains cldfChain.BlockChains
}

func NewMCMInspectorFetcher(chains cldfChain.BlockChains) *MCMInspectorFetcher {
	return &MCMInspectorFetcher{chains: chains}
}

// FetchInspectors gets a map of inspectors for the given chain metadata and chain clients
func (b *MCMInspectorFetcher) FetchInspectors(
	chainMetadata map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata,
	action mcmsTypes.TimelockAction) (map[mcmsTypes.ChainSelector]sdk.Inspector, error) {
	inspectors := map[mcmsTypes.ChainSelector]sdk.Inspector{}
	for chainSelector := range chainMetadata {
		inspector, err := GetInspectorFromChainSelector(b.chains, uint64(chainSelector), action)
		if err != nil {
			return nil, fmt.Errorf("error getting inspector for chain selector %d: %w", chainSelector, err)
		}
		inspectors[chainSelector] = inspector
	}

	return inspectors, nil
}

// GetInspectorFromChainSelector returns an inspector for the given chain selector and chain clients
func GetInspectorFromChainSelector(chains cldfChain.BlockChains, selector uint64, action mcmsTypes.TimelockAction) (sdk.Inspector, error) {
	fam, err := mcmsTypes.GetChainSelectorFamily(mcmsTypes.ChainSelector(selector))
	if err != nil {
		return nil, fmt.Errorf("error getting chainClient family: %w", err)
	}

	var inspector sdk.Inspector
	switch fam {
	case chainsel.FamilyEVM:
		inspector = evm.NewInspector(chains.EVMChains()[selector].Client)
	case chainsel.FamilySolana:
		inspector = solana.NewInspector(chains.SolanaChains()[selector].Client)
	case chainsel.FamilyAptos:
		role, err := chainsmetadata.AptosRoleFromAction(action)
		if err != nil {
			return nil, fmt.Errorf("error getting aptos role from proposal: %w", err)
		}
		chainClient := chains.AptosChains()[selector]
		inspector = aptos.NewInspector(chainClient.Client, role)
	default:
		return nil, fmt.Errorf("unsupported chain family %s", fam)
	}

	return inspector, nil
}
