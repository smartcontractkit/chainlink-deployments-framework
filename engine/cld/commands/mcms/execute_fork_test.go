package mcms

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	cldf_evm "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/solana"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func TestExecuteFork_FlagShortcuts(t *testing.T) {
	t.Parallel()

	cmd, err := newTestCommand(t)
	require.NoError(t, err)

	// Find the execute-fork subcommand
	executeForkCmd, _, err := cmd.Find([]string{"execute-fork"})
	require.NoError(t, err)

	// Verify shorthand flags exist
	tests := []struct {
		longFlag  string
		shortFlag string
	}{
		{"environment", "e"},
		{"proposal", "p"},
		{"proposalKind", "k"},
		{"selector", "s"},
	}

	for _, tt := range tests {
		flag := executeForkCmd.Flags().Lookup(tt.longFlag)
		require.NotNil(t, flag, "flag %s should exist", tt.longFlag)
		require.Equal(t, tt.shortFlag, flag.Shorthand, "flag %s should have shorthand %s", tt.longFlag, tt.shortFlag)
	}
}

func TestExecuteFork_RequiredFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		args         []string
		wantErrMatch string
	}{
		{
			name:         "missing all required flags",
			args:         []string{"execute-fork"},
			wantErrMatch: "required flag",
		},
		{
			name:         "missing proposal flag",
			args:         []string{"execute-fork", "-e", "staging", "-s", "1"},
			wantErrMatch: "required flag",
		},
		{
			name:         "missing environment flag",
			args:         []string{"execute-fork", "-p", "/path/to/proposal.json", "-s", "1"},
			wantErrMatch: "required flag",
		},
		{
			name:         "missing selector flag",
			args:         []string{"execute-fork", "-e", "staging", "-p", "/path/to/proposal.json"},
			wantErrMatch: "required flag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd, err := newTestCommand(t)
			require.NoError(t, err)

			cmd.SetArgs(tt.args)
			out := new(bytes.Buffer)
			cmd.SetOut(out)
			cmd.SetErr(out)

			execErr := cmd.Execute()
			require.ErrorContains(t, execErr, tt.wantErrMatch)
		})
	}
}

func TestExecuteFork_DefaultProposalKind(t *testing.T) {
	t.Parallel()

	cmd, err := newTestCommand(t)
	require.NoError(t, err)

	// Find the execute-fork subcommand
	executeForkCmd, _, err := cmd.Find([]string{"execute-fork"})
	require.NoError(t, err)

	// Verify default proposal kind - uses the types.KindTimelockProposal string value
	kindFlag := executeForkCmd.Flags().Lookup("proposalKind")
	require.NotNil(t, kindFlag)
	require.Equal(t, "TimelockProposal", kindFlag.DefValue, "default proposalKind should be 'TimelockProposal'")
}

func TestExecuteFork_TestSignerFlag(t *testing.T) {
	t.Parallel()

	cmd, err := newTestCommand(t)
	require.NoError(t, err)

	// Find the execute-fork subcommand
	executeForkCmd, _, err := cmd.Find([]string{"execute-fork"})
	require.NoError(t, err)

	// Verify test-signer flag exists and is boolean
	testSignerFlag := executeForkCmd.Flags().Lookup("test-signer")
	require.NotNil(t, testSignerFlag)
	require.Equal(t, "false", testSignerFlag.DefValue, "test-signer default should be false")
	require.Equal(t, "bool", testSignerFlag.Value.Type(), "test-signer should be boolean")
}

func TestExecuteFork_CommandDescriptions(t *testing.T) {
	t.Parallel()

	cmd, err := newTestCommand(t)
	require.NoError(t, err)

	// Find the execute-fork subcommand
	executeForkCmd, _, err := cmd.Find([]string{"execute-fork"})
	require.NoError(t, err)

	// Verify descriptions are set
	require.NotEmpty(t, executeForkCmd.Short, "Short description should be set")
	require.NotEmpty(t, executeForkCmd.Long, "Long description should be set")
	require.NotEmpty(t, executeForkCmd.Example, "Example should be set")
}

func TestForkConfig_Structure(t *testing.T) {
	t.Parallel()

	// Verify forkConfig has all necessary fields
	cfg := &forkConfig{}

	// These assertions verify the struct fields exist at compile time
	// and provide documentation of expected structure
	_ = cfg.kind
	_ = cfg.proposal
	_ = cfg.timelockProposal
	_ = cfg.chainSelector
	_ = cfg.blockchains
	_ = cfg.envStr
	_ = cfg.env
	_ = cfg.forkedEnv
	_ = cfg.fork
	_ = cfg.proposalCtx
}

func TestExecuteFork_Config_MissingProposalContextProvider(t *testing.T) {
	t.Parallel()

	// Create a command without ProposalContextProvider
	_, err := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain(t.TempDir(), "testdomain"),
		// ProposalContextProvider is missing
	})

	require.ErrorContains(t, err, "ProposalContextProvider")
}

func TestOverrideForkChainDeployerKeyToTestSigner_Success(t *testing.T) {
	t.Parallel()

	const selector = uint64(4286062357653186312)

	prevPrivKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	prevTxOpts, err := bind.NewKeyedTransactorWithChainID(prevPrivKey, big.NewInt(998))
	require.NoError(t, err)

	cfg := &forkConfig{
		chainSelector: selector,
		blockchains: chain.NewBlockChains(map[uint64]chain.BlockChain{
			selector: cldf_evm.Chain{
				Selector:    selector,
				DeployerKey: prevTxOpts,
			},
		}),
	}

	err = overrideForkChainDeployerKeyToTestSigner(cfg, "998")
	require.NoError(t, err)

	updatedChain := cfg.blockchains.EVMChains()[selector]
	require.NotNil(t, updatedChain.DeployerKey)
	require.Equal(t, blockchain.DefaultAnvilPublicKey, updatedChain.DeployerKey.From.Hex())
	require.NotEqual(t, prevTxOpts.From, updatedChain.DeployerKey.From)
}

func TestOverrideForkChainDeployerKeyToTestSigner_InvalidChainID(t *testing.T) {
	t.Parallel()

	const selector = uint64(4286062357653186312)
	cfg := &forkConfig{
		chainSelector: selector,
		blockchains: chain.NewBlockChains(map[uint64]chain.BlockChain{
			selector: cldf_evm.Chain{Selector: selector},
		}),
	}

	err := overrideForkChainDeployerKeyToTestSigner(cfg, "not-a-number")
	require.ErrorContains(t, err, "invalid chain id")
}

func TestOverrideForkChainDeployerKeyToTestSigner_NonEVMChain(t *testing.T) {
	t.Parallel()

	const selector = uint64(4286062357653186312)
	cfg := &forkConfig{
		chainSelector: selector,
		blockchains: chain.NewBlockChains(map[uint64]chain.BlockChain{
			selector: solana.Chain{Selector: selector},
		}),
	}

	err := overrideForkChainDeployerKeyToTestSigner(cfg, "998")
	require.ErrorContains(t, err, "is not an evm chain")
}
