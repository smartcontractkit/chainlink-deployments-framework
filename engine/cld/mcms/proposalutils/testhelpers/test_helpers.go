package testhelpers

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smartcontractkit/ccip-owner-contracts/pkg/config"
	bindings "github.com/smartcontractkit/ccip-owner-contracts/pkg/gethwrappers"
	chainsel "github.com/smartcontractkit/chain-selectors"
	mcmslib "github.com/smartcontractkit/mcms"
	mcmschainwrappers "github.com/smartcontractkit/mcms/chainwrappers"
	mcmstypes "github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/require"

	cldfmcmsadapters "github.com/smartcontractkit/chainlink-deployments-framework/chain/mcms/adapters"
	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	proposalutils "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalutils"
)

// TestXXXMCMSSigner is a throwaway private key used for signing MCMS proposals in tests.
var TestXXXMCMSSigner *ecdsa.PrivateKey

func init() {
	key, err := crypto.GenerateKey()
	if err != nil {
		panic(err)
	}
	TestXXXMCMSSigner = key
}

func SingleGroupMCMSLegacy(t *testing.T) config.Config {
	t.Helper()

	publicKey := TestXXXMCMSSigner.Public().(*ecdsa.PublicKey)
	// Convert the public key to an Ethereum address
	address := crypto.PubkeyToAddress(*publicKey)
	c, err := config.NewConfig(1, []common.Address{address}, []config.Config{})
	require.NoError(t, err)

	return *c
}

func SingleGroupMCMS(t *testing.T) mcmstypes.Config {
	t.Helper()

	publicKey := TestXXXMCMSSigner.Public().(*ecdsa.PublicKey)
	// Convert the public key to an Ethereum address
	address := crypto.PubkeyToAddress(*publicKey)
	c, err := mcmstypes.NewConfig(1, []common.Address{address}, []mcmstypes.Config{})
	require.NoError(t, err)

	return c
}

// SignMCMSTimelockProposal - Signs an MCMS timelock proposal.
func SignMCMSTimelockProposal(t *testing.T, env cldf.Environment, proposal *mcmslib.TimelockProposal, realBackend bool) *mcmslib.Proposal {
	t.Helper()

	converters, err := mcmschainwrappers.BuildConverters(proposal.ChainMetadata)
	require.NoError(t, err)

	mcmsChains := cldfmcmsadapters.Wrap(env.BlockChains)
	inspectors, err := mcmschainwrappers.BuildInspectors(&mcmsChains, proposal.ChainMetadata, proposal.Action)
	require.NoError(t, err)

	p, _, err := proposal.Convert(env.GetContext(), converters)
	require.NoError(t, err)

	p.UseSimulatedBackend(!realBackend)

	signable, err := mcmslib.NewSignable(&p, inspectors)
	require.NoError(t, err)

	err = signable.ValidateConfigs(env.GetContext())
	require.NoError(t, err)

	signer := mcmslib.NewPrivateKeySigner(TestXXXMCMSSigner)
	_, err = signable.SignAndAppend(signer)
	require.NoError(t, err)

	quorumMet, err := signable.ValidateSignatures(env.GetContext())
	require.NoError(t, err)
	require.True(t, quorumMet)

	return &p
}

// SignMCMSProposal - Signs an MCMS proposal. For timelock proposal, use SignMCMSTimelockProposal instead.
func SignMCMSProposal(t *testing.T, env cldf.Environment, proposal *mcmslib.Proposal) *mcmslib.Proposal {
	t.Helper()

	mcmsChains := cldfmcmsadapters.Wrap(env.BlockChains)
	inspectors, err := mcmschainwrappers.BuildInspectors(&mcmsChains, proposal.ChainMetadata, mcmstypes.TimelockActionSchedule)
	require.NoError(t, err)

	proposal.UseSimulatedBackend(true)

	signable, err := mcmslib.NewSignable(proposal, inspectors)
	require.NoError(t, err)

	err = signable.ValidateConfigs(env.GetContext())
	require.NoError(t, err)

	signer := mcmslib.NewPrivateKeySigner(TestXXXMCMSSigner)
	_, err = signable.SignAndAppend(signer)
	require.NoError(t, err)

	quorumMet, err := signable.ValidateSignatures(env.GetContext())
	require.NoError(t, err)
	require.True(t, quorumMet)

	return proposal
}

// ExecuteMCMSProposalV2 executes an MCMS proposal on all chains.
// For timelock proposals, use ExecuteMCMSTimelockProposalV2 instead.
func ExecuteMCMSProposalV2(t *testing.T, env cldf.Environment, proposal *mcmslib.Proposal) error {
	t.Helper()

	t.Log("Executing proposal")

	encoders, err := proposal.GetEncoders()
	if err != nil {
		return fmt.Errorf("[ExecuteMCMSProposalV2] failed to get encoders: %w", err)
	}

	mcmsChains := cldfmcmsadapters.Wrap(env.BlockChains)
	executors, err := mcmschainwrappers.BuildExecutors(&mcmsChains, proposal.ChainMetadata, encoders, mcmstypes.TimelockActionSchedule)
	if err != nil {
		return fmt.Errorf("[ExecuteMCMSProposalV2] failed to build executors: %w", err)
	}

	executable, err := mcmslib.NewExecutable(proposal, executors)
	if err != nil {
		return fmt.Errorf("[ExecuteMCMSProposalV2] failed to build executable: %w", err)
	}

	for chainSelector := range executors {
		t.Logf("[ExecuteMCMSProposalV2] Setting root on chain %d...", chainSelector)
		root, err := executable.SetRoot(env.GetContext(), chainSelector)
		if err != nil {
			return fmt.Errorf("[ExecuteMCMSProposalV2] SetRoot failed: %w", err)
		}

		family, err := chainsel.GetSelectorFamily(uint64(chainSelector))
		if err != nil {
			return fmt.Errorf("[ExecuteMCMSProposalV2] failed to get chain family for selector %d: %w", chainSelector, err)
		}

		if family == chainsel.FamilyEVM {
			evmChain, ok := env.BlockChains.EVMChains()[uint64(chainSelector)]
			if !ok {
				return fmt.Errorf("[ExecuteMCMSProposalV2] EVM chain not found for selector %d", chainSelector)
			}
			evmTransaction, ok := root.RawData.(*gethtypes.Transaction)
			if !ok {
				return fmt.Errorf("[ExecuteMCMSProposalV2] unexpected RawData type %T", root.RawData)
			}
			t.Logf("[ExecuteMCMSProposalV2] SetRoot EVM tx hash: %s", evmTransaction.Hash().String())
			if _, err = evmChain.Confirm(evmTransaction); err != nil {
				return fmt.Errorf("[ExecuteMCMSProposalV2] Confirm failed: %w", err)
			}
		}
		if family == chainsel.FamilyAptos {
			aptosChain, ok := env.BlockChains.AptosChains()[uint64(chainSelector)]
			if !ok {
				return fmt.Errorf("[ExecuteMCMSProposalV2] Aptos chain not found for selector %d", chainSelector)
			}
			t.Logf("[ExecuteMCMSProposalV2] SetRoot Aptos tx hash: %s", root.Hash)
			if err = aptosChain.Confirm(root.Hash); err != nil {
				return fmt.Errorf("[ExecuteMCMSProposalV2] Confirm failed: %w", err)
			}
		}
	}

	for i, op := range proposal.Operations {
		t.Logf("[ExecuteMCMSProposalV2] Executing operation index=%d on chain %d...", i, uint64(op.ChainSelector))
		result, err := executable.Execute(env.GetContext(), i)
		if err != nil {
			return fmt.Errorf("[ExecuteMCMSProposalV2] Execute failed: %w", err)
		}

		family, err := chainsel.GetSelectorFamily(uint64(op.ChainSelector))
		if err != nil {
			return fmt.Errorf("[ExecuteMCMSProposalV2] failed to get chain family for selector %d: %w", op.ChainSelector, err)
		}

		if family == chainsel.FamilyEVM {
			evmChain, ok := env.BlockChains.EVMChains()[uint64(op.ChainSelector)]
			if !ok {
				return fmt.Errorf("[ExecuteMCMSProposalV2] EVM chain not found for selector %d", op.ChainSelector)
			}
			evmTransaction, ok := result.RawData.(*gethtypes.Transaction)
			if !ok {
				return fmt.Errorf("[ExecuteMCMSProposalV2] unexpected RawData type %T", result.RawData)
			}
			t.Logf("[ExecuteMCMSProposalV2] Operation %d EVM tx hash: %s", i, evmTransaction.Hash().String())
			if _, err = evmChain.Confirm(evmTransaction); err != nil {
				return fmt.Errorf("[ExecuteMCMSProposalV2] Confirm failed: %w", err)
			}
		}
		if family == chainsel.FamilyAptos {
			aptosChain, ok := env.BlockChains.AptosChains()[uint64(op.ChainSelector)]
			if !ok {
				return fmt.Errorf("[ExecuteMCMSProposalV2] Aptos chain not found for selector %d", op.ChainSelector)
			}
			t.Logf("[ExecuteMCMSProposalV2] Operation %d Aptos tx hash: %s", i, result.Hash)
			if err = aptosChain.Confirm(result.Hash); err != nil {
				return fmt.Errorf("[ExecuteMCMSProposalV2] Confirm failed: %w", err)
			}
		}
	}

	return nil
}

// ExecuteMCMSTimelockProposalV2 executes an MCMS timelock proposal.
// It optionally sets a callProxy to execute calls through a proxy.
// If the callProxy is not set, the calls are executed directly to the timelock.
func ExecuteMCMSTimelockProposalV2(t *testing.T, env cldf.Environment, timelockProposal *mcmslib.TimelockProposal, opts ...mcmslib.Option) error {
	t.Helper()

	t.Log("Executing timelock proposal")

	mcmsChains := cldfmcmsadapters.Wrap(env.BlockChains)
	executors, err := mcmschainwrappers.BuildTimelockExecutors(&mcmsChains, timelockProposal.ChainMetadata,
		timelockProposal.Action)
	if err != nil {
		return fmt.Errorf("[ExecuteMCMSTimelockProposalV2] failed to build timelock executors: %w", err)
	}

	timelockExecutable, err := mcmslib.NewTimelockExecutable(env.GetContext(), timelockProposal, executors)
	if err != nil {
		return fmt.Errorf("[ExecuteMCMSTimelockProposalV2] failed to build timelock executable: %w", err)
	}

	deadline := time.Now().Add(100 * time.Second)
	for {
		if err := timelockExecutable.IsReady(env.GetContext()); err == nil {
			break
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("[ExecuteMCMSTimelockProposalV2] proposal not ready after 100s: %w",
				timelockExecutable.IsReady(env.GetContext()))
		}
		time.Sleep(50 * time.Millisecond)
	}

	for i, op := range timelockProposal.Operations {
		family, err := chainsel.GetSelectorFamily(uint64(op.ChainSelector))
		if err != nil {
			return fmt.Errorf("[ExecuteMCMSTimelockProposalV2] failed to get chain family for selector %d: %w", op.ChainSelector, err)
		}

		opOpts := slices.Clone(opts)
		if family == chainsel.FamilyEVM {
			callProxy, errCall := findCallProxyAddress(env, uint64(op.ChainSelector), timelockProposal.TimelockAddresses[op.ChainSelector])
			if errCall != nil {
				return fmt.Errorf("[ExecuteMCMSTimelockProposalV2] failed to find call proxy address: %w", errCall)
			}
			opOpts = append(opOpts, mcmslib.WithCallProxy(callProxy))
			t.Logf("[ExecuteMCMSTimelockProposalV2] Using EVM chain with chainID=%d, timelock address %s call proxy %s",
				uint64(op.ChainSelector), timelockProposal.TimelockAddresses[op.ChainSelector], callProxy)
		}

		tx, err := timelockExecutable.Execute(env.GetContext(), i, opOpts...)
		if err != nil {
			return fmt.Errorf("[ExecuteMCMSTimelockProposalV2] Execute failed: %w", err)
		}
		t.Logf("[ExecuteMCMSTimelockProposalV2] Executed timelock operation index=%d on chain %d (tx %v)", i, uint64(op.ChainSelector), tx.Hash)

		if family == chainsel.FamilyEVM {
			evmChain, ok := env.BlockChains.EVMChains()[uint64(op.ChainSelector)]
			if !ok {
				return fmt.Errorf("[ExecuteMCMSTimelockProposalV2] EVM chain not found for selector %d", op.ChainSelector)
			}
			evmTransaction, ok := tx.RawData.(*gethtypes.Transaction)
			if !ok {
				return fmt.Errorf("[ExecuteMCMSTimelockProposalV2] unexpected RawData type %T", tx.RawData)
			}
			if _, err = evmChain.Confirm(evmTransaction); err != nil {
				return fmt.Errorf("[ExecuteMCMSTimelockProposalV2] Confirm on EVM failed: %w", err)
			}
		}
		if family == chainsel.FamilyAptos {
			aptosChain, ok := env.BlockChains.AptosChains()[uint64(op.ChainSelector)]
			if !ok {
				return fmt.Errorf("[ExecuteMCMSTimelockProposalV2] Aptos chain not found for selector %d", op.ChainSelector)
			}
			if err = aptosChain.Confirm(tx.Hash); err != nil {
				return fmt.Errorf("[ExecuteMCMSTimelockProposalV2] Confirm on Aptos failed: %w", err)
			}
		}
	}

	return nil
}

func SingleGroupTimelockConfig(t *testing.T) proposalutils.MCMSWithTimelockConfig {
	t.Helper()

	return proposalutils.MCMSWithTimelockConfig{
		Canceller:        SingleGroupMCMS(t),
		Bypasser:         SingleGroupMCMS(t),
		Proposer:         SingleGroupMCMS(t),
		TimelockMinDelay: big.NewInt(0),
	}
}

func findCallProxyAddress(env cldf.Environment, chainSelector uint64, timelockAddr string) (string, error) {
	evmChain, ok := env.BlockChains.EVMChains()[chainSelector]
	if !ok {
		return "", fmt.Errorf("EVM chain not found for selector %d", chainSelector)
	}

	timelock, err := bindings.NewRBACTimelock(common.HexToAddress(timelockAddr), evmChain.Client)
	if err != nil {
		return "", fmt.Errorf("failed to create timelock binding: %w", err)
	}

	role, err := timelock.EXECUTORROLE(&bind.CallOpts{
		Context: env.GetContext(),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get executor role: %w", err)
	}

	addr, err := timelock.GetRoleMember(&bind.CallOpts{
		Context: env.GetContext(),
	}, role, big.NewInt(0))
	if err != nil {
		return "", fmt.Errorf("failed to get role member: %w", err)
	}

	if addr == (common.Address{}) {
		return "", errors.New("executor role has no members; is the timelock initialized?")
	}

	return addr.Hex(), nil
}
