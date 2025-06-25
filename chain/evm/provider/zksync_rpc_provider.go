package provider

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	zkAccounts "github.com/zksync-sdk/zksync2-go/accounts"
	zkClients "github.com/zksync-sdk/zksync2-go/clients"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

// ZkSyncRPCChainProviderConfig holds the configuration to initialize the RPCChainProvider. While
// zkSync uses the same underlying EVM chain, it requires a different setup for the deployer key
// and signer.
type ZkSyncRPCChainProviderConfig struct {
	// Required: A generator for the deployer key. Use TransactorFromRaw to create a deployer
	// key from a private key, or TransactorFromKMS to create a deployer key from a KMS key.
	DeployerTransactorGen TransactorGenerator
	// Required: A generator for the ZkSync signer. The generator you choose should match the
	// type of deployer transactor generator you are using. For example, if you are using
	// TransactorFromRaw, you should use ZkSyncSignerFromRaw, or if you are using
	// TransactorFromKMS, you should use ZkSyncSignerFromKMS.
	SignerGenerator ZkSyncSignerGenerator
	// Required: At least one RPC must be provided to connect to the EVM node.
	RPCs []deployment.RPC
	// Required: ConfirmFunctor is a type that generates a confirmation function for transactions.
	// Use ConfirmFuncGeth to use the Geth client for transaction confirmation, or
	// ConfirmFuncSeth to use the Seth client for transaction confirmation with richer debugging.
	//
	// If in doubt, use ConfirmFuncGeth.
	ConfirmFunctor ConfirmFunctor
	// Optional: ClientOpts are additional options to configure the MultiClient used by the
	// RPCChainProvider. These options are applied to the MultiClient instance created by the
	// RPCChainProvider. You can use this to set up custom HTTP clients, timeouts, or other
	// configurations for the RPC connections.
	ClientOpts []func(client *deployment.MultiClient)
	// Optional: Logger is the logger to use for the RPCChainProvider. If not provided, a default
	// logger will be used.
	Logger logger.Logger
}

// validate checks if the config is valid.
func (c ZkSyncRPCChainProviderConfig) validate() error {
	if c.DeployerTransactorGen == nil {
		return errors.New("deployer transactor generator is required")
	}
	if c.SignerGenerator == nil {
		return errors.New("signer generator is required")
	}
	if c.ConfirmFunctor == nil {
		return errors.New("confirm functor is required")
	}
	if len(c.RPCs) == 0 {
		return errors.New("at least one RPC is required")
	}

	return nil
}

var _ chain.Provider = (*ZkSyncRPCChainProvider)(nil)

// ZkSyncRPCChainProvider is a chain provider that provides a chain that connects to an EVM node
// via RPC.
type ZkSyncRPCChainProvider struct {
	selector uint64
	config   ZkSyncRPCChainProviderConfig

	chain *evm.Chain
}

// NewZkSyncRPCChainProvider creates a new instance with the given selector and configuration.
func NewZkSyncRPCChainProvider(
	selector uint64, config ZkSyncRPCChainProviderConfig,
) *ZkSyncRPCChainProvider {
	return &ZkSyncRPCChainProvider{
		selector: selector,
		config:   config,
	}
}

// Initialize initializes the ZkSyncRPCChainProvider, setting up the EVM chain with the provided
// configuration. It returns the initialized chain.BlockChain or an error if initialization fails.
func (p *ZkSyncRPCChainProvider) Initialize(ctx context.Context) (chain.BlockChain, error) {
	if p.chain != nil {
		return *p.chain, nil // Already initialized
	}

	// Set up the logger if not provided
	if p.config.Logger == nil {
		lggr, err := logger.New()
		if err != nil {
			return nil, fmt.Errorf("failed to create default logger: %w", err)
		}
		p.config.Logger = lggr
	}

	// Validate the provider configuration
	if err := p.config.validate(); err != nil {
		return nil, fmt.Errorf("failed to validate provider config: %w", err)
	}

	// Get the Chain ID
	chainIDStr, err := chain_selectors.GetChainIDFromSelector(p.selector)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID from selector %d: %w", p.selector, err)
	}

	chainID, ok := new(big.Int).SetString(chainIDStr, 10)
	if !ok {
		return nil, fmt.Errorf("failed to convert chain ID %s to big.Int", chainIDStr)
	}

	// Generate the deployer key using the provided transactor generator
	deployerKey, err := p.config.DeployerTransactorGen.Generate(chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate deployer key: %w", err)
	}

	// Setup the client.
	client, err := deployment.NewMultiClient(p.config.Logger, deployment.RPCConfig{
		ChainSelector: p.selector,
		RPCs:          p.config.RPCs,
	}, p.config.ClientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create multi-client: %w", err)
	}

	// Setup the confirm function
	confirmFunc, err := p.config.ConfirmFunctor.Generate(
		ctx, p.selector, client, deployerKey.From,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate confirm function: %w", err)
	}

	// Initialize the zksync client and wallet
	clientZk := zkClients.NewClient(client.Client.Client())
	signer, err := p.config.SignerGenerator.Generate(chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate zkSync signer: %w", err)
	}

	deployerKeyZkSyncVM, err := zkAccounts.NewWalletFromSigner(signer, clientZk, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create zkSync deployer key: %w", err)
	}

	p.chain = &evm.Chain{
		Selector:            p.selector,
		Client:              client,
		DeployerKey:         deployerKey,
		Confirm:             confirmFunc,
		IsZkSyncVM:          true,
		ClientZkSyncVM:      clientZk,
		DeployerKeyZkSyncVM: deployerKeyZkSyncVM,
	}

	return *p.chain, nil
}

// Name returns the name of the ZkSyncRPCChainProvider.
func (*ZkSyncRPCChainProvider) Name() string {
	return "ZkSync EVM RPC Chain Provider"
}

// ChainSelector returns the chain selector of the simulated chain managed by this provider.
func (p *ZkSyncRPCChainProvider) ChainSelector() uint64 {
	return p.selector
}

// BlockChain returns the simulated chain instance managed by this provider. You must call Initialize
// before using this method to ensure the chain is properly set up.
func (p *ZkSyncRPCChainProvider) BlockChain() chain.BlockChain {
	return *p.chain
}
