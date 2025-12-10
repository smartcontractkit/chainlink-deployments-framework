package provider

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/provider/rpcclient"
)

// RPCChainProviderConfig holds the configuration to initialize the RPCChainProvider.
type RPCChainProviderConfig struct {
	// Required: A generator for the deployer key. Use TransactorFromRaw to create a deployer
	// key from a private key, or TransactorFromKMS to create a deployer key from a KMS key.
	DeployerTransactorGen SignerGenerator
	// Required: At least one RPC must be provided to connect to the EVM node.
	RPCs []rpcclient.RPC
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
	ClientOpts []func(client *rpcclient.MultiClient)
	// Optional: A generator for the additional user transactors. If not provided, no user
	// transactors will be generated.
	UsersTransactorGen []SignerGenerator
	// Optional: Logger is the logger to use for the RPCChainProvider. If not provided, a default
	// logger will be used.
	Logger logger.Logger
}

// validate checks if the RPCChainProviderConfig is valid.
func (c RPCChainProviderConfig) validate() error {
	if c.DeployerTransactorGen == nil {
		return errors.New("deployer transactor generator is required")
	}
	if c.ConfirmFunctor == nil {
		return errors.New("confirm functor is required")
	}
	if len(c.RPCs) == 0 {
		return errors.New("at least one RPC is required")
	}

	return nil
}

// RPCChainProvider is a chain provider that provides a chain that connects to an EVM node via RPC.
type RPCChainProvider struct {
	selector uint64
	config   RPCChainProviderConfig

	chain *evm.Chain
}

// NewRPCChainProvider creates a new RPCChainProvider with the given selector and configuration.
func NewRPCChainProvider(
	selector uint64, config RPCChainProviderConfig,
) *RPCChainProvider {
	return &RPCChainProvider{
		selector: selector,
		config:   config,
	}
}

// Initialize initializes the RPCChainProvider, setting up the EVM chain with the provided
// configuration. It returns the initialized chain.BlockChain or an error if initialization fails.
func (p *RPCChainProvider) Initialize(ctx context.Context) (chain.BlockChain, error) {
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
	chainIDStr, err := chainsel.GetChainIDFromSelector(p.selector)
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

	// Generate the other user transactors
	users := make([]*bind.TransactOpts, 0, len(p.config.UsersTransactorGen))
	for _, g := range p.config.UsersTransactorGen {
		u, gerr := g.Generate(chainID)
		if gerr != nil {
			return nil, fmt.Errorf("failed to generate user transactor: %w", err)
		}

		users = append(users, u)
	}

	// Setup the client.
	client, err := rpcclient.NewMultiClient(p.config.Logger, rpcclient.RPCConfig{
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

	p.chain = &evm.Chain{
		Selector:    p.selector,
		Client:      client,
		DeployerKey: deployerKey,
		Users:       users,
		Confirm:     confirmFunc,
		SignHash:    p.config.DeployerTransactorGen.SignHash,
	}

	return *p.chain, nil
}

// Name returns the name of the RPCChainProvider.
func (*RPCChainProvider) Name() string {
	return "EVM RPC Chain Provider"
}

// ChainSelector returns the chain selector of the simulated chain managed by this provider.
func (p *RPCChainProvider) ChainSelector() uint64 {
	return p.selector
}

// BlockChain returns the simulated chain instance managed by this provider. You must call Initialize
// before using this method to ensure the chain is properly set up.
func (p *RPCChainProvider) BlockChain() chain.BlockChain {
	return *p.chain
}
