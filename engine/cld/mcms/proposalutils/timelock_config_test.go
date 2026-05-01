package proposalutils

import (
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/gagliardetto/solana-go"
	ownerhelpers "github.com/smartcontractkit/ccip-owner-contracts/pkg/gethwrappers"
	chainsel "github.com/smartcontractkit/chain-selectors"
	tonstate "github.com/smartcontractkit/chainlink-ton/deployment/state"
	mcmssolanasdk "github.com/smartcontractkit/mcms/sdk/solana"
	mcmstypes "github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/require"
	"github.com/xssnick/tonutils-go/address"

	cldf_aptos "github.com/smartcontractkit/chainlink-deployments-framework/chain/aptos"
	cldf_evm "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	mcmscontracts "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/contracts/mcms"
)

type stubEVMState struct {
	contracts MCMSWithTimelockContracts
}

func (s stubEVMState) TimelockContracts() MCMSWithTimelockContracts {
	return s.contracts
}

type stubSolanaState struct {
	programs MCMSWithTimelockPrograms
}

func (s stubSolanaState) TimelockPrograms() MCMSWithTimelockPrograms {
	return s.programs
}

func TestTimelockConfigMCMBasedOnActionDefaultsToSchedule(t *testing.T) {
	t.Parallel()

	cfg := TimelockConfig{}
	proposer := &ownerhelpers.ManyChainMultiSig{}

	got, err := cfg.MCMBasedOnAction(stubEVMState{
		contracts: MCMSWithTimelockContracts{ProposerMcm: proposer},
	})

	require.NoError(t, err)
	require.Same(t, proposer, got)
	require.Equal(t, mcmstypes.TimelockActionSchedule, cfg.MCMSAction)
}

func TestTimelockConfigMCMBasedOnActionSelectsRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		action    mcmstypes.TimelockAction
		contracts MCMSWithTimelockContracts
		want      func(MCMSWithTimelockContracts) *ownerhelpers.ManyChainMultiSig
	}{
		{
			name:      "schedule",
			action:    mcmstypes.TimelockActionSchedule,
			contracts: MCMSWithTimelockContracts{ProposerMcm: &ownerhelpers.ManyChainMultiSig{}},
			want:      func(c MCMSWithTimelockContracts) *ownerhelpers.ManyChainMultiSig { return c.ProposerMcm },
		},
		{
			name:      "cancel",
			action:    mcmstypes.TimelockActionCancel,
			contracts: MCMSWithTimelockContracts{CancellerMcm: &ownerhelpers.ManyChainMultiSig{}},
			want:      func(c MCMSWithTimelockContracts) *ownerhelpers.ManyChainMultiSig { return c.CancellerMcm },
		},
		{
			name:      "bypass",
			action:    mcmstypes.TimelockActionBypass,
			contracts: MCMSWithTimelockContracts{BypasserMcm: &ownerhelpers.ManyChainMultiSig{}},
			want:      func(c MCMSWithTimelockContracts) *ownerhelpers.ManyChainMultiSig { return c.BypasserMcm },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := TimelockConfig{MCMSAction: tt.action}

			got, err := cfg.MCMBasedOnAction(stubEVMState{contracts: tt.contracts})

			require.NoError(t, err)
			require.Same(t, tt.want(tt.contracts), got)
		})
	}
}

func TestTimelockConfigMCMBasedOnActionErrorsOnMissingRole(t *testing.T) {
	t.Parallel()

	cfg := TimelockConfig{MCMSAction: mcmstypes.TimelockActionCancel}

	_, err := cfg.MCMBasedOnAction(stubEVMState{})

	require.EqualError(t, err, "missing cancellerMcm")
}

func TestTimelockConfigMCMBasedOnActionSolanaSelectsRole(t *testing.T) {
	t.Parallel()

	program := solana.NewWallet().PublicKey()
	programs := MCMSWithTimelockPrograms{
		McmProgram:       program,
		ProposerMcmSeed:  mcmssolanasdk.PDASeed([32]byte{1}),
		CancellerMcmSeed: mcmssolanasdk.PDASeed([32]byte{2}),
		BypasserMcmSeed:  mcmssolanasdk.PDASeed([32]byte{3}),
	}

	tests := []struct {
		name   string
		action mcmstypes.TimelockAction
		want   string
	}{
		{
			name:   "schedule",
			action: mcmstypes.TimelockActionSchedule,
			want:   mcmssolanasdk.ContractAddress(program, programs.ProposerMcmSeed),
		},
		{
			name:   "cancel",
			action: mcmstypes.TimelockActionCancel,
			want:   mcmssolanasdk.ContractAddress(program, programs.CancellerMcmSeed),
		},
		{
			name:   "bypass",
			action: mcmstypes.TimelockActionBypass,
			want:   mcmssolanasdk.ContractAddress(program, programs.BypasserMcmSeed),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := TimelockConfig{MCMSAction: tt.action}

			got, err := cfg.MCMBasedOnActionSolana(stubSolanaState{programs: programs})

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestTimelockConfigMCMBasedOnActionTONDefaultsToSchedule(t *testing.T) {
	t.Parallel()

	cfg := TimelockConfig{}
	proposer := address.MustParseAddr("EQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAM9c")

	got, err := cfg.MCMBasedOnActionTON(&tonstate.MCMSSuiteState{
		Proposer: proposer,
	})

	require.NoError(t, err)
	require.Equal(t, proposer.String(), got)
	require.Equal(t, mcmstypes.TimelockActionSchedule, cfg.MCMSAction)
}

func TestTimelockConfigMCMBasedOnActionTONSelectsRole(t *testing.T) {
	t.Parallel()

	proposer := address.MustParseAddr("EQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAM9c")
	canceller := address.MustParseAddr("EQAREREREREREREREREREREREREREREREREREREREREREeYT")
	bypasser := address.MustParseAddr("EQAiIiIiIiIiIiIiIiIiIiIiIiIiIiIiIiIiIiIiIiIiIp3c")

	tests := []struct {
		name   string
		action mcmstypes.TimelockAction
		state  *tonstate.MCMSSuiteState
		want   string
	}{
		{
			name:   "schedule",
			action: mcmstypes.TimelockActionSchedule,
			state: &tonstate.MCMSSuiteState{
				Proposer: proposer,
			},
			want: proposer.String(),
		},
		{
			name:   "cancel",
			action: mcmstypes.TimelockActionCancel,
			state: &tonstate.MCMSSuiteState{
				Canceller: canceller,
			},
			want: canceller.String(),
		},
		{
			name:   "bypass",
			action: mcmstypes.TimelockActionBypass,
			state: &tonstate.MCMSSuiteState{
				Bypasser: bypasser,
			},
			want: bypasser.String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := TimelockConfig{MCMSAction: tt.action}

			got, err := cfg.MCMBasedOnActionTON(tt.state)

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestTimelockConfigMCMBasedOnActionTONErrorsOnMissingRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		action mcmstypes.TimelockAction
		want   string
	}{
		{
			name:   "schedule",
			action: mcmstypes.TimelockActionSchedule,
			want:   "missing TON proposer",
		},
		{
			name:   "cancel",
			action: mcmstypes.TimelockActionCancel,
			want:   "missing TON canceller",
		},
		{
			name:   "bypass",
			action: mcmstypes.TimelockActionBypass,
			want:   "missing TON bypasser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := TimelockConfig{MCMSAction: tt.action}

			_, err := cfg.MCMBasedOnActionTON(&tonstate.MCMSSuiteState{})

			require.EqualError(t, err, tt.want)
		})
	}
}

func TestTimelockConfigValidate(t *testing.T) {
	t.Parallel()

	chain := cldf_evm.Chain{Selector: chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector}
	cfg := TimelockConfig{}

	err := cfg.Validate(chain, stubEVMState{
		contracts: MCMSWithTimelockContracts{
			ProposerMcm: &ownerhelpers.ManyChainMultiSig{},
			Timelock:    &ownerhelpers.RBACTimelock{},
			CallProxy:   &ownerhelpers.CallProxy{},
		},
	})

	require.NoError(t, err)
	require.Equal(t, mcmstypes.TimelockActionSchedule, cfg.MCMSAction)
}

func TestTimelockConfigValidateErrorsOnMissingCallProxy(t *testing.T) {
	t.Parallel()

	chain := cldf_evm.Chain{Selector: chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector}
	cfg := TimelockConfig{}

	err := cfg.Validate(chain, stubEVMState{
		contracts: MCMSWithTimelockContracts{
			ProposerMcm: &ownerhelpers.ManyChainMultiSig{},
			Timelock:    &ownerhelpers.RBACTimelock{},
		},
	})

	require.ErrorContains(t, err, "missing callProxy")
}

func TestTimelockConfigValidateSolana(t *testing.T) {
	t.Parallel()

	ab := cldf.NewMemoryAddressBook()
	version := *semver.MustParse("1.0.0")
	chainSelector := chainsel.SOLANA_DEVNET.Selector
	program := solana.NewWallet().PublicKey()

	require.NoError(t, ab.Save(chainSelector, mcmssolanasdk.ContractAddress(program, mcmssolanasdk.PDASeed([32]byte{1})), cldf.NewTypeAndVersion(mcmscontracts.RBACTimelock, version)))
	require.NoError(t, ab.Save(chainSelector, mcmssolanasdk.ContractAddress(program, mcmssolanasdk.PDASeed([32]byte{2})), cldf.NewTypeAndVersion(mcmscontracts.ProposerManyChainMultisig, version)))

	cfg := TimelockConfig{}
	err := cfg.ValidateSolana(cldf.Environment{ExistingAddresses: ab}, chainSelector)

	require.NoError(t, err)
	require.Equal(t, mcmstypes.TimelockActionSchedule, cfg.MCMSAction)
}

func TestTimelockConfigValidateAptos(t *testing.T) {
	t.Parallel()

	cfg := TimelockConfig{}
	var addr aptos.AccountAddress

	require.NoError(t, addr.ParseStringRelaxed("0x1"))
	require.NoError(t, cfg.ValidateAptos(cldf_aptos.Chain{Selector: chainsel.APTOS_MAINNET.Selector}, addr))

	err := cfg.ValidateAptos(cldf_aptos.Chain{Selector: chainsel.APTOS_MAINNET.Selector}, aptos.AccountAddress{})
	require.ErrorContains(t, err, "aptos MCMS contract not present")
}
