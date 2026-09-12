package simulation

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms"
	mcmstypes "github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/simulation/statediff"
)

const (
	ethSel   = mcmstypes.ChainSelector(5009297550715157269)
	suiSel   = mcmstypes.ChainSelector(17529533435026248318)
	aptosSel = mcmstypes.ChainSelector(4741433654826277614)
)

func ref(sel uint64, addr, ctype, version string) datastore.AddressRef {
	v, _ := semver.NewVersion(version)

	return datastore.AddressRef{
		ChainSelector: sel,
		Address:       addr,
		Type:          datastore.ContractType(ctype),
		Version:       v,
	}
}

func seedDS(t *testing.T, refs ...datastore.AddressRef) *datastore.MemoryDataStore {
	t.Helper()
	ds := datastore.NewMemoryDataStore()
	for _, r := range refs {
		require.NoError(t, ds.Addresses().Add(r))
	}

	return ds
}

func TestResolveScope_EVMTargetsAndContext(t *testing.T) {
	t.Parallel()

	p := &mcms.TimelockProposal{
		BaseProposal: mcms.BaseProposal{
			ChainMetadata: map[mcmstypes.ChainSelector]mcmstypes.ChainMetadata{
				ethSel: {StartingOpCount: 1584, MCMAddress: "0xMCM"},
			},
		},
		TimelockAddresses: map[mcmstypes.ChainSelector]string{ethSel: "0xTL"},
		Operations: []mcmstypes.BatchOperation{
			{
				ChainSelector: ethSel,
				Transactions: []mcmstypes.Transaction{
					{To: "0xOnRamp", Data: []byte{1}},
					{To: "0xRouter", Data: []byte{2}},
				},
			},
		},
	}
	ds := seedDS(t,
		ref(uint64(ethSel), "0xMCM", "ProposerManyChainMultiSig", "1.0.0"),
		ref(uint64(ethSel), "0xTL", "RBACTimelock", "1.0.0"),
		ref(uint64(ethSel), "0xOnRamp", "EVM2EVMOnRamp", "1.6.0"),
		ref(uint64(ethSel), "0xRouter", "Router", "1.2.0"),
		ref(uint64(ethSel), "0xTKN", "ERC20Token", "1.0.0"),
	)

	scope, unreadable, err := ResolveScope(context.Background(), ds.Addresses(), p, ethSel)
	require.NoError(t, err)
	require.Empty(t, unreadable)

	// Targets come from the transactions' to-addresses, resolved via the
	// datastore (type/version authoritative even when the proposal carries
	// an empty contractType, as here).
	require.Len(t, scope.Targets, 2)
	require.Equal(t, "EVM2EVMOnRamp", string(scope.Targets[0].Type))
	require.Equal(t, "Router", string(scope.Targets[1].Type))

	// Context: MCM + timelock + every router/token ref on the chain. The
	// router overlaps with a target; both are kept (context guarantees view
	// resolution even for chains the proposal only reads).
	require.Len(t, scope.Context, 4)
	contextTypes := make([]string, 0, len(scope.Context))
	for _, r := range scope.Context {
		contextTypes = append(contextTypes, string(r.Type))
	}
	require.ElementsMatch(t, []string{"ProposerManyChainMultiSig", "RBACTimelock", "Router", "ERC20Token"}, contextTypes)
}

func TestResolveScope_SuiStateObjSecondaryTarget(t *testing.T) {
	t.Parallel()

	p := &mcms.TimelockProposal{
		BaseProposal: mcms.BaseProposal{
			ChainMetadata: map[mcmstypes.ChainSelector]mcmstypes.ChainMetadata{
				suiSel: {StartingOpCount: 5, MCMAddress: "0xSUI-MCM"},
			},
		},
		TimelockAddresses: map[mcmstypes.ChainSelector]string{suiSel: "0xSUI-TL"},
		Operations: []mcmstypes.BatchOperation{
			{
				ChainSelector: suiSel,
				Transactions: []mcmstypes.Transaction{{
					To:   "0xofframp",
					Data: []byte{1},
					AdditionalFields: json.RawMessage(
						`{"module_name":"offramp","function":"set_ocr3_config","state_obj":"0xstateobj"}`),
				}},
			},
		},
	}
	ds := seedDS(t,
		ref(uint64(suiSel), "0xSUI-MCM", "SuiManyChainMultisigObjectID", "1.0.0"),
		ref(uint64(suiSel), "0xSUI-TL", "SuiManyChainMultisigTimelockObjectID", "1.0.0"),
		ref(uint64(suiSel), "0xofframp", "SuiOffRamp", "1.0.0"),
		ref(uint64(suiSel), "0xstateobj", "SuiOffRampStateObjectID", "1.0.0"),
	)

	scope, unreadable, err := ResolveScope(context.Background(), ds.Addresses(), p, suiSel)
	require.NoError(t, err)
	require.Empty(t, unreadable)

	// The mutated object (state_obj) is a distinct diff target from to.
	require.Len(t, scope.Targets, 2)
	require.Equal(t, "SuiOffRamp", string(scope.Targets[0].Type))
	require.Equal(t, "SuiOffRampStateObjectID", string(scope.Targets[1].Type))
}

func TestResolveScope_UnknownAddressIsUnreadable(t *testing.T) {
	t.Parallel()

	p := &mcms.TimelockProposal{
		BaseProposal: mcms.BaseProposal{
			ChainMetadata: map[mcmstypes.ChainSelector]mcmstypes.ChainMetadata{
				ethSel: {StartingOpCount: 1, MCMAddress: "0xMCM"},
			},
		},
		TimelockAddresses: map[mcmstypes.ChainSelector]string{ethSel: "0xTL"},
		Operations: []mcmstypes.BatchOperation{
			{
				ChainSelector: ethSel,
				Transactions:  []mcmstypes.Transaction{{To: "0xghost", Data: []byte{1}}},
			},
		},
	}
	ds := seedDS(t, ref(uint64(ethSel), "0xMCM", "ProposerManyChainMultiSig", "1.0.0"))

	scope, unreadable, err := ResolveScope(context.Background(), ds.Addresses(), p, ethSel)
	require.NoError(t, err)
	require.Empty(t, scope.Targets)
	require.Equal(t, []statediff.Unreadable{{
		ChainSelector: uint64(ethSel),
		Address:       "0xghost",
		Reason:        statediff.ReasonUnknownAddress,
	}}, unreadable)
}

func TestResolveScope_MissingChainMetadata(t *testing.T) {
	t.Parallel()

	p := &mcms.TimelockProposal{
		BaseProposal: mcms.BaseProposal{
			ChainMetadata: map[mcmstypes.ChainSelector]mcmstypes.ChainMetadata{
				ethSel: {StartingOpCount: 1, MCMAddress: "0xMCM"},
			},
		},
	}
	other := mcmstypes.ChainSelector(1111)

	_, _, err := ResolveScope(context.Background(), datastore.NewMemoryDataStore().Addresses(), p, other)
	require.ErrorContains(t, err, "no chain metadata for selector 1111")
}

// loadFixtureRefs seeds an in-memory datastore from the committed mainnet ref
// table (testdata/refs — see its README for provenance and refresh).
func loadFixtureRefs(t *testing.T) *datastore.MemoryDataStore {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "refs", "mainnet-fixtures.json"))
	require.NoError(t, err)
	var refs []datastore.AddressRef
	require.NoError(t, json.Unmarshal(raw, &refs))
	ds := datastore.NewMemoryDataStore()
	for _, r := range refs {
		require.NoError(t, ds.Addresses().Add(r))
	}

	return ds
}

// loadFixtureProposal loads a committed mainnet proposal fixture. The
// fixtures are expired (historical), so LoadProposal returns the proposal
// alongside a validity error — accepted here exactly as execute-fork accepts
// it via acceptExpiredProposal.
func loadFixtureProposal(t *testing.T, name string) *mcms.TimelockProposal {
	t.Helper()
	loaded, _ := mcms.LoadProposal(mcmstypes.KindTimelockProposal, filepath.Join("testdata", "proposals", name))
	require.NotNil(t, loaded, "loading proposal %s", name)
	proposal, ok := loaded.(*mcms.TimelockProposal)
	require.True(t, ok, "expected TimelockProposal, got %T", loaded)

	return proposal
}

func contextTypes(scope Scope) []string {
	out := make([]string, 0, len(scope.Context))
	for _, r := range scope.Context {
		out = append(out, string(r.Type))
	}

	return out
}

// TestResolveScope_FixtureSuiStateObj exercises the Sui secondary-target rule
// on a real committed proposal: every transaction mutates a state object that
// is distinct from its to-address, and both resolve through the datastore.
func TestResolveScope_FixtureSuiStateObj(t *testing.T) {
	t.Parallel()

	proposal := loadFixtureProposal(t,
		"1786654670927127434-ccip-mainnet-set_ocr3_config_mcms_timelock_proposal_0.json")

	scope, unreadable, err := ResolveScope(context.Background(), loadFixtureRefs(t).Addresses(), proposal, suiSel)
	require.NoError(t, err)
	require.Empty(t, unreadable)

	targetTypes := make([]string, 0, len(scope.Targets))
	for _, r := range scope.Targets {
		targetTypes = append(targetTypes, string(r.Type))
	}
	require.ElementsMatch(t, []string{"SuiOffRamp", "SuiOffRampStateObjectID"}, targetTypes)
	t.Logf("sui fixture: targets=%d context=%d", len(scope.Targets), len(scope.Context))

	// Context: the chain's MCM + timelock + its router and token refs.
	require.Contains(t, contextTypes(scope), "SuiManyChainMultisigObjectID")
	require.Contains(t, contextTypes(scope), "SuiManyChainMultisigTimelockObjectID")
	require.Contains(t, contextTypes(scope), "SuiRouter")
}

// TestResolveScope_FixtureEVMMultiTarget exercises a real EVM multi-target
// proposal: both to-addresses resolve with datastore-authoritative types (the
// proposal's own contractType strings are generic labels), and the context
// carries the chain's MCM, timelock and every router/token ref.
func TestResolveScope_FixtureEVMMultiTarget(t *testing.T) {
	t.Parallel()

	proposal := loadFixtureProposal(t,
		"1787168710406750-ccip-mainnet-transfer_ownership_raw-deploy_cctp_chains_merged_proposal.json")

	scope, unreadable, err := ResolveScope(context.Background(), loadFixtureRefs(t).Addresses(), proposal, ethSel)
	require.NoError(t, err)
	require.Empty(t, unreadable)

	targetTypes := make([]string, 0, len(scope.Targets))
	for _, r := range scope.Targets {
		targetTypes = append(targetTypes, string(r.Type))
	}
	require.ElementsMatch(t, []string{"SiloedUSDCTokenPool", "USDCTokenPoolProxy"}, targetTypes)

	t.Logf("evm fixture: targets=%d context=%d", len(scope.Targets), len(scope.Context))
	require.Len(t, scope.Context, 135)
	require.Contains(t, contextTypes(scope), "ProposerManyChainMultiSig")
	require.Contains(t, contextTypes(scope), "RBACTimelock")
	require.Contains(t, contextTypes(scope), "Router")
}

// TestResolveScope_FixtureEmptyContractType exercises a real proposal whose
// transactions carry an empty contractType (the Aptos leg): the datastore
// supplies type and version. On that chain the MCM and timelock addresses are
// identical, so context dedupes them to a single entry.
func TestResolveScope_FixtureEmptyContractType(t *testing.T) {
	t.Parallel()

	proposal := loadFixtureProposal(t,
		"1786555924138823-ccip-mainnet-promote_candidates_and_set_ocr3-update_chain_configs_merged_proposal.json")

	scope, unreadable, err := ResolveScope(context.Background(), loadFixtureRefs(t).Addresses(), proposal, aptosSel)
	require.NoError(t, err)
	require.Empty(t, unreadable)
	require.Len(t, scope.Targets, 1)
	require.Equal(t, "AptosCCIP", string(scope.Targets[0].Type))
	require.Equal(t, "1.6.0", scope.Targets[0].Version.String())
	t.Logf("aptos fixture: targets=%d context=%d", len(scope.Targets), len(scope.Context))
	require.Len(t, scope.Context, 11)
	require.Contains(t, contextTypes(scope), "AptosManyChainMultisig")

	// The same proposal's Ethereum leg resolves its named targets normally.
	evmScope, evmUnreadable, err := ResolveScope(context.Background(), loadFixtureRefs(t).Addresses(), proposal, ethSel)
	require.NoError(t, err)
	require.Empty(t, evmUnreadable)
	require.Len(t, evmScope.Targets, 2)
	evmTargetTypes := make([]string, 0, len(evmScope.Targets))
	for _, r := range evmScope.Targets {
		evmTargetTypes = append(evmTargetTypes, string(r.Type))
	}
	require.ElementsMatch(t, []string{"CapabilitiesRegistry", "CCIPHome"}, evmTargetTypes)
	require.Len(t, evmScope.Context, 135)
}
