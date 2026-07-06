package proposalutils

import (
	"encoding/json"
	"fmt"

	chainsel "github.com/smartcontractkit/chain-selectors"
	mcmschainwrappers "github.com/smartcontractkit/mcms/chainwrappers"
	mcmssdk "github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/aptos"
	cantonsdk "github.com/smartcontractkit/mcms/sdk/canton"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/sdk/ton"
	mcmstypes "github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	cldfmcmsadapters "github.com/smartcontractkit/chainlink-deployments-framework/chain/mcms/adapters"
)

// McmsTimelockInspectorForChain builds a read-only sdk.TimelockInspector for a chain loaded
// in blockChains. Proposal chain metadata is required for Sui and Aptos timelocks.
//
// MCMS exposes chain-family switching via chainwrappers.BuildInspector for MCMS contracts,
// but not for timelock inspectors. This helper centralizes that dispatch for CLDF callers
// until mcms adds an equivalent chainwrappers.BuildTimelockInspector.
func McmsTimelockInspectorForChain(
	blockChains chain.BlockChains,
	chainSelector uint64,
	metadata mcmstypes.ChainMetadata,
) (mcmssdk.TimelockInspector, error) {
	chainAccessor := cldfmcmsadapters.Wrap(blockChains)

	return buildTimelockInspector(&chainAccessor, mcmstypes.ChainSelector(chainSelector), metadata)
}

func buildTimelockInspector(
	chains mcmschainwrappers.ChainAccessor,
	chainSelector mcmstypes.ChainSelector,
	metadata mcmstypes.ChainMetadata,
) (mcmssdk.TimelockInspector, error) {
	family, err := mcmstypes.GetChainSelectorFamily(chainSelector)
	if err != nil {
		return nil, fmt.Errorf("chain family: %w", err)
	}

	rawSelector := uint64(chainSelector)
	switch family {
	case chainsel.FamilyEVM:
		client, ok := chains.EVMClient(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing EVM chain client for selector %d", rawSelector)
		}

		return evm.NewTimelockInspector(client), nil
	case chainsel.FamilySolana:
		client, ok := chains.SolanaClient(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing Solana chain client for selector %d", rawSelector)
		}

		return solana.NewTimelockInspector(client), nil
	case chainsel.FamilyAptos:
		client, ok := chains.AptosClient(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing Aptos chain client for selector %d", rawSelector)
		}

		mcmsType := aptos.MCMSTypeRegular
		if len(metadata.AdditionalFields) > 0 {
			var afm aptos.AdditionalFieldsMetadata
			if unmarshalErr := json.Unmarshal(metadata.AdditionalFields, &afm); unmarshalErr != nil {
				return nil, fmt.Errorf("parse aptos metadata for selector %d: %w", rawSelector, unmarshalErr)
			}
			mcmsType = afm.MCMSType
		}

		return aptos.NewTimelockInspectorWithMCMSType(client, mcmsType), nil
	case chainsel.FamilySui:
		client, ok := chains.SuiClient(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing Sui chain client for selector %d", rawSelector)
		}
		signer, ok := chains.SuiSigner(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing Sui signer for selector %d", rawSelector)
		}

		suiMetadata, err := sui.SuiMetadata(metadata)
		if err != nil {
			return nil, fmt.Errorf("parse sui metadata for selector %d: %w", rawSelector, err)
		}

		return sui.NewTimelockInspector(client, signer, suiMetadata.McmsPackageID)
	case chainsel.FamilyTon:
		client, ok := chains.TonClient(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing TON chain client for selector %d", rawSelector)
		}

		return ton.NewTimelockInspector(client), nil
	case chainsel.FamilyCanton:
		ch, ok := chains.CantonChain(rawSelector)
		if !ok || len(ch.Participants) == 0 {
			return nil, fmt.Errorf("missing Canton chain participant for selector %d", rawSelector)
		}
		participant := ch.Participants[0]
		mcmsParties := cantonsdk.MCMSPartiesForChain(ch)

		return cantonsdk.NewTimelockInspector(
			participant.LedgerServices.Command,
			participant.LedgerServices.State,
			participant.PartyID,
			mcmsParties,
		), nil
	default:
		return nil, fmt.Errorf("unsupported chain family %q", family)
	}
}
