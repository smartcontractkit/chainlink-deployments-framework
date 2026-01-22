package mcmsv2

import (
	"context"
	"crypto/ecdsa"
	"errors"
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
	"github.com/smartcontractkit/mcms"
	mcmssdk "github.com/smartcontractkit/mcms/sdk"
	mcmstypes "github.com/smartcontractkit/mcms/types"

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
	mcmAddress := cfg.proposal.ChainMetadata[mcmstypes.ChainSelector(cfg.chainSelector)].MCMAddress
	timelockAddress := common.HexToAddress(cfg.timelockProposal.TimelockAddresses[mcmstypes.ChainSelector(cfg.chainSelector)])

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
			mcmAddress,
		); lerr != nil {
			return fmt.Errorf("failed to set signer: %w", lerr)
		}

		// Override signatures for proposal
		privKey, lerr := crypto.HexToECDSA(blockchain.DefaultAnvilPrivateKey)
		if lerr != nil {
			return fmt.Errorf("failed to parse anvil's default private key: %w", lerr)
		}

		lerr = overwriteProposalSignatureWithTestKey(ctx, cfg, privKey)
		if lerr != nil {
			return fmt.Errorf("failed to overwrite proposal signature: %w", lerr)
		}
	}

	// set root
	// TODO: improve error decoding on the mcms lib for "set root".
	err = setRootCommand(ctx, lggr, cfg)
	if err != nil {
		return fmt.Errorf("MCM.setRoot() - failure: %w", err)
	}
	lggr.Info("MCM.setRoot() - success")

	// TODO: improve error decoding on the mcms lib for "execute chain".
	err = executeChainCommand(ctx, lggr, cfg, true)
	if err != nil {
		return fmt.Errorf("MCM.execute() - failure: %w", err)
	}
	lggr.Info("MCM.execute() - success")

	lggr.Info("Wait for the chain to be mined before executing timelock chain command")
	if err = anvilClient.EVMIncreaseTime(uint64(cfg.timelockProposal.Delay.Seconds())); err != nil {
		return fmt.Errorf("failed to increase time: %w", err)
	}
	if err = anvilClient.AnvilMine([]interface{}{1}); err != nil {
		return fmt.Errorf("failed to mine block: %w", err)
	}

	if cfg.timelockProposal.Action != mcmstypes.TimelockActionSchedule {
		lggr.Infof("Proposal has type %s, skipping executing timelock chain command", cfg.timelockProposal.Action)
		return nil
	}

	lggr.Info("Executing timelock chain command")
	err = timelockExecuteChainCommand(ctx, lggr, cfg)
	if err != nil {
		lggr.Warnw("Timelock.execute() - failure; starting calling individual ops for debugging", "err", err)
		if derr := diagnoseTimelockRevert(ctx, lggr, anvilClient.URL, cfg.chainSelector, cfg.timelockProposal.Operations,
			timelockAddress, cfg.env.ExistingAddresses, cfg.proposalCtx); derr != nil { //nolint:staticcheck
			lggr.Errorw("Diagnosis results", "err", derr)
			return fmt.Errorf("failed to timelock execute chain: %w", derr)
		}

		return fmt.Errorf("failed to timelock execute chain: %w", err)
	}
	lggr.Info("Timelock.execute() - success")

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

// overwriteProposalSignatureWithTestKey overwrites the proposal's signature with a test key signature.
func overwriteProposalSignatureWithTestKey(ctx context.Context, cfg *cfgv2, testKey *ecdsa.PrivateKey) error {
	p := &cfg.proposal

	// Override the proposal fields that are used in the signing hash to ensure no errors occur related to those.
	if time.Unix(int64(p.ValidUntil), 0).Before(time.Now().Add(10 * time.Minute)) {
		p.ValidUntil = uint32(time.Now().Add(5 * time.Hour).Unix()) //nolint:gosec // G404: time-based validity is acceptable for test signatures
	}
	p.Signatures = nil
	p.OverridePreviousRoot = true

	inspector, err := getInspectorFromChainSelector(*cfg)
	if err != nil {
		return fmt.Errorf("error getting inspector from chain selector: %w", err)
	}
	signable, errSignable := mcms.NewSignable(p, map[mcmstypes.ChainSelector]mcmssdk.Inspector{
		mcmstypes.ChainSelector(cfg.chainSelector): inspector,
	})
	if errSignable != nil {
		return fmt.Errorf("error creating signable: %w", errSignable)
	}

	signature, err := signable.SignAndAppend(mcms.NewPrivateKeySigner(testKey))
	p.Signatures = []mcmstypes.Signature{signature}
	if err != nil {
		return fmt.Errorf("error creating signable: %w", err)
	}

	quorumMet, err := signable.CheckQuorum(ctx, mcmstypes.ChainSelector(cfg.chainSelector))
	if err != nil {
		return fmt.Errorf("failed to check quorum: %w", err)
	}
	if !quorumMet {
		return errors.New("quorum not met")
	}

	return nil
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
