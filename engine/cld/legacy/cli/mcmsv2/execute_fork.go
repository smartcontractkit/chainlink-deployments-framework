package mcmsv2

import (
	"context"
	"fmt"
	"maps"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/rpc"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	cldf_evm "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/legacy/cli/mcmsv2/layout"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func executeFork(
	ctx context.Context, lggr logger.Logger, cfg *cfgv2, testSigner bool,
) error {
	family, err := chainsel.GetSelectorFamily(cfg.chainSelector)
	if err != nil {
		return fmt.Errorf("failed to get selector family: %w", err)
	}
	if family != chainsel.FamilyEVM {
		lggr.Infof("Skipping fork execution: chain selector %d is not EVM. Family is %s", cfg.chainSelector, family)
		return nil // don’t fail, just exit cleanly
	}

	logTransactions(lggr, cfg)

	if len(cfg.forkedEnv.ChainConfigs[cfg.chainSelector].HTTPRPCs) == 0 {
		return fmt.Errorf("no rpcs loaded in forked environment for chain %d (fork tests require public RPCs)", cfg.chainSelector)
	}

	// get the chain URL, chain ID and MCM contract address
	url := cfg.forkedEnv.ChainConfigs[cfg.chainSelector].HTTPRPCs[0].External
	anvilClient := rpc.New(url, nil)
	chainID := cfg.forkedEnv.ChainConfigs[cfg.chainSelector].ChainID
	mcmsAddr := cfg.proposal.ChainMetadata[types.ChainSelector(cfg.chainSelector)].MCMAddress

	ctx, cancel := context.WithTimeout(ctx, 300*time.Second)
	defer cancel()
	if testSigner {
		if lerr := layout.SetMCMSigner(
			ctx,
			lggr,
			layout.MCMSLayout,
			blockchain.DefaultAnvilPrivateKey,
			blockchain.DefaultAnvilPublicKey,
			blockchain.DefaultAnvilPublicKey,
			url,
			chainID,
			mcmsAddr,
		); lerr != nil {
			return fmt.Errorf("failed to set signer: %w", lerr)
		}
	}
	// Override signatures for proposal
	privKey, err := crypto.HexToECDSA(blockchain.DefaultAnvilPrivateKey)
	if err != nil {
		return fmt.Errorf("failed to create private key: %w", err)
	}
	timelockAddress := common.HexToAddress(cfg.timelockProposal.TimelockAddresses[types.ChainSelector(cfg.chainSelector)])

	if err = overwriteProposalSignatureWithTestKey(cfg, privKey); err != nil {
		return fmt.Errorf("failed to overwrite proposal signature: %w", err)
	}

	// set root
	// TODO: improve error decoding on the mcms lib for set root.
	err = setRootCommand(ctx, lggr, cfg)
	if err != nil {
		return fmt.Errorf("failed to set root: %w", err)
	}
	lggr.Info("Root set successfully")
	// TODO: improve error decoding on the mcms lib for set root.
	err = executeChainCommand(ctx, lggr, cfg, true)
	if err != nil {
		return fmt.Errorf("failed to execute chain: %w", err)
	}
	lggr.Info("MCMs execute() success")
	lggr.Info("Waiting for the chain to be mined before executing timelock chain command")

	if err = anvilClient.EVMIncreaseTime(uint64(cfg.timelockProposal.Delay.Seconds())); err != nil {
		return fmt.Errorf("failed to increase time: %w", err)
	}
	if err = anvilClient.AnvilMine([]interface{}{1}); err != nil {
		return fmt.Errorf("failed to mine block: %w", err)
	}

	if cfg.timelockProposal.Action != types.TimelockActionSchedule {
		lggr.Infof("Proposal has type %s, skipping executing timelock chain command", cfg.timelockProposal.Action)
		return nil
	}

	lggr.Info("Executing timelock chain command")
	err = timelockExecuteChainCommand(ctx, lggr, cfg)
	if err != nil {
		lggr.Warnw("Timelock execute failed, starting calling individual ops for debugging", "err", err)
		// envdir := domain.EnvDir(cfg.envStr)
		// ab := cfg.env.ExistingAddresses
		// if errAb != nil {
		// 	return fmt.Errorf("failed to load address book: %w", err)
		// }
		if derr := diagnoseTimelockRevert(ctx, lggr, anvilClient.URL, cfg.chainSelector, cfg.timelockProposal.Operations,
			timelockAddress, cfg.env.ExistingAddresses, cfg.proposalCtx); derr != nil {
			lggr.Errorw("Diagnosis results", "err", derr)
			return fmt.Errorf("failed to timelock execute chain: %w", derr)
		}

		return fmt.Errorf("failed to timelock execute chain: %w", err)
	}
	lggr.Info("Timelock execute chain success")

	return nil
}

// --- helper types and functions ---

func logTransactions(lggr logger.Logger, cfg *cfgv2) {
	lggr.Infof("logging transactions sent to forked chain %v", cfg.chainSelector)

	chains := maps.Collect(cfg.blockchains.All())

	evmChain, ok := chains[cfg.chainSelector].(cldf_evm.Chain)
	if !ok {
		lggr.Warnf("failed to configure transaction logging for chain selector %v (not evm: %T)", cfg.chainSelector, chains[cfg.chainSelector])
		return
	}

	evmChain.Client = &loggingRpcClient{OnchainClient: evmChain.Client, txOpts: evmChain.DeployerKey, lggr: lggr}
	chains[cfg.chainSelector] = evmChain
	cfg.blockchains = chain.NewBlockChains(chains)
}

type loggingRpcClient struct {
	cldf_evm.OnchainClient
	txOpts *bind.TransactOpts
	lggr   logger.Logger
}

func (c *loggingRpcClient) SendTransaction(ctx context.Context, tx *gethtypes.Transaction) error {
	c.lggr.Infow("sending on-chain transaction", "from", c.txOpts.From, "to", tx.To(), "value", tx.Value(),
		"data", common.Bytes2Hex(tx.Data()))

	return c.OnchainClient.SendTransaction(ctx, tx)
}
