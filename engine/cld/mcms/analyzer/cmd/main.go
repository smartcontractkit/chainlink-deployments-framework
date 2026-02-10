// Package main provides a standalone CLI for testing the proposal analyzer.
// Production usage should go through AnalyzerEngine.Run().
package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/smartcontractkit/mcms"

	"github.com/smartcontractkit/chainlink-ccip/chains/evm/gobindings/generated/latest/token_pool"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	cldfevm "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/analyzer/analyzers/ccip"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/analyzer/analyzers/evm"
	expanalyzer "github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <proposal.json>\n", os.Args[0])
		os.Exit(1)
	}

	if err := run(os.Args[1]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(proposalPath string) error {
	ctx := context.Background()

	proposal, err := loadProposal(proposalPath)
	if err != nil {
		return fmt.Errorf("load proposal: %w", err)
	}

	chains, err := connectChains(ctx, proposal)
	if err != nil {
		return fmt.Errorf("connect chains: %w", err)
	}

	env, proposalCtx, err := buildProposalContext(proposal, chains)
	if err != nil {
		return fmt.Errorf("build proposal context: %w", err)
	}

	engine := analyzer.NewAnalyzerEngine()
	if err := engine.RegisterAnalyzer(&evm.EVMFieldParameterAnalyzer{}); err != nil {
		return fmt.Errorf("register field analyzer: %w", err)
	}
	if err := engine.RegisterAnalyzer(&evm.ERC20TokenAmountAnalyzer{}); err != nil {
		return fmt.Errorf("register erc20 amount analyzer: %w", err)
	}
	if err := engine.RegisterAnalyzer(&evm.TimelockUpdateDelayAnalyzer{}); err != nil {
		return fmt.Errorf("register timelock analyzer: %w", err)
	}
	if err := engine.RegisterAnalyzer(&ccip.TokenPoolChainUpdatesAnalyzer{}); err != nil {
		return fmt.Errorf("register analyzer: %w", err)
	}

	result, err := engine.AnalyzeWithProposalContext(ctx, env, proposalCtx, proposal)
	if err != nil {
		return fmt.Errorf("analyze: %w", err)
	}

	fmt.Print(analyzer.RenderText(result, proposal.Description))

	return nil
}

func loadProposal(path string) (*mcms.TimelockProposal, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return mcms.NewTimelockProposal(f)
}

func connectChains(ctx context.Context, proposal *mcms.TimelockProposal) (chain.BlockChains, error) {
	evmChains := make(map[uint64]chain.BlockChain)

	for sel := range proposal.ChainMetadata {
		chainSel := uint64(sel)
		rpcURL := os.Getenv("RPC_" + strconv.FormatUint(chainSel, 10))

		if rpcURL == "" {
			return chain.BlockChains{}, fmt.Errorf("no RPC URL for chain %d (set RPC_%d=<url>)", chainSel, chainSel)
		}

		client, err := ethclient.DialContext(ctx, rpcURL)
		if err != nil {
			return chain.BlockChains{}, fmt.Errorf("dial chain %d: %w", chainSel, err)
		}

		evmChains[chainSel] = cldfevm.Chain{Selector: chainSel, Client: client}
	}

	return chain.NewBlockChains(evmChains), nil
}

func buildProposalContext(
	proposal *mcms.TimelockProposal,
	bc chain.BlockChains,
) (deployment.Environment, expanalyzer.ProposalContext, error) {
	addressesByChain := make(map[uint64]map[string]deployment.TypeAndVersion)
	for _, batch := range proposal.Operations {
		chainSel := uint64(batch.ChainSelector)
		if _, ok := addressesByChain[chainSel]; !ok {
			addressesByChain[chainSel] = make(map[string]deployment.TypeAndVersion)
		}

		for _, tx := range batch.Transactions {
			addressesByChain[chainSel][tx.To] = deployment.TypeAndVersion{
				Type: deployment.ContractType(tx.ContractType),
			}
		}
	}

	env := deployment.Environment{
		BlockChains:       bc,
		DataStore:         datastore.NewMemoryDataStore().Seal(),
		ExistingAddresses: deployment.NewMemoryAddressBookFromMap(addressesByChain),
	}

	proposalCtx, err := expanalyzer.NewDefaultProposalContext(env, expanalyzer.WithEVMABIMappings(evmABIMappings()))

	return env, proposalCtx, err
}

// in prod, real versions come from the CLD DataStore.
func evmABIMappings() map[string]string {
	return map[string]string{
		"TokenPool 0.0.0":                 token_pool.TokenPoolABI,
		"LockReleaseTokenPool 0.0.0":      token_pool.TokenPoolABI,
		"BurnMintTokenPool 0.0.0":         token_pool.TokenPoolABI,
		"BurnFromMintTokenPool 0.0.0":     token_pool.TokenPoolABI,
		"BurnWithFromMintTokenPool 0.0.0": token_pool.TokenPoolABI,
		"RBACTimelock 0.0.0":              expanalyzer.RBACTimelockMetaDataTesting.ABI,
		"ERC20 0.0.0":                     erc20ABI,
	}
}

const erc20ABI = `[
	{"inputs":[{"name":"spender","type":"address"},{"name":"amount","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"stateMutability":"nonpayable","type":"function"},
	{"inputs":[{"name":"recipient","type":"address"},{"name":"amount","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"stateMutability":"nonpayable","type":"function"},
	{"inputs":[{"name":"sender","type":"address"},{"name":"recipient","type":"address"},{"name":"amount","type":"uint256"}],"name":"transferFrom","outputs":[{"name":"","type":"bool"}],"stateMutability":"nonpayable","type":"function"},
	{"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"stateMutability":"view","type":"function"},
	{"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"stateMutability":"view","type":"function"}
]`
