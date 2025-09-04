package provider

import (
	"context"
	"errors"
	"fmt"
	"github.com/xssnick/tonutils-go/liteclient"
	tonlib "github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"log"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/ton"
)

// RPCChainProviderConfig holds the configuration to initialize the RPCChainProvider.
type RPCChainProviderConfig struct {
	// Required: The HTTP RPC URL to connect to the Ton node.
	HTTPURL string
	// Optional: The WebSocket URL to connect to the Ton node.
	WSURL             string
	DeployerSignerGen PrivateKeyGenerator
	WalletVersion     string
}

// validate checks if the RPCChainProviderConfig is valid.
func (c RPCChainProviderConfig) validate() error {
	if c.HTTPURL == "" {
		return errors.New("rpc url is required")
	}
	if c.DeployerSignerGen == nil {
		return errors.New("deployer signer generator is required")
	}
	if _, err := getWalletVersionConfig(c.WalletVersion); err != nil {
		return err
	}

	return nil
}

var _ chain.Provider = (*RPCChainProvider)(nil)

// RPCChainProvider is a chain provider that provides a chain that connects to an TON node via
// RPC.
type RPCChainProvider struct {
	// Ton chain selector, used to identify the chain.
	selector uint64

	// RPCChainProviderConfig holds the configuration for the RPCChainProvider.
	config RPCChainProviderConfig

	// chain is the Ton chain instance that this provider manages. The Initialize method
	// sets up the chain.
	chain *ton.Chain
}

func NewRPCChainProvider(selector uint64, config RPCChainProviderConfig) *RPCChainProvider {
	return &RPCChainProvider{
		selector: selector,
		config:   config,
	}
}

// Initialize initializes the RPCChainProvider.
func (p *RPCChainProvider) Initialize(_ context.Context) (chain.BlockChain, error) {
	if p.chain != nil {
		return *p.chain, nil // Already initialized
	}

	if err := p.config.validate(); err != nil {
		return nil, fmt.Errorf("failed to validate provider config: %w", err)
	}

	// Initialize TON client
	connectionPool := liteclient.NewConnectionPool()
	// Connect to public LiteServer config. We use the RPC URL to get the config
	err := connectionPool.AddConnectionsFromConfigUrl(context.Background(), p.config.HTTPURL)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve ton network config: %w", err)
	}

	api := tonlib.NewAPIClient(connectionPool, tonlib.ProofCheckPolicySecure)

	// Wallet
	privateKey, err := p.config.DeployerSignerGen.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// (No need to validate that the version is supported, already done by p.config.validate)
	walletConfig, _ := getWalletVersionConfig(p.config.WalletVersion)
	tonWallet, err := wallet.FromPrivateKeyWithOptions(api, privateKey, walletConfig, wallet.WithWorkchain(0))

	if err != nil {
		return nil, fmt.Errorf("failed to init TON wallet: %w", err)
	}

	log.Printf("TON wallet loaded with address %s", tonWallet.WalletAddress().String())

	p.chain = &ton.Chain{
		ChainMetadata: ton.ChainMetadata{
			Selector: p.selector,
		},
		Client:        api,
		Wallet:        tonWallet,
		WalletAddress: tonWallet.WalletAddress(),
		URL:           p.config.HTTPURL,
	}

	return *p.chain, nil
}

// getWalletVersionConfig returns the wallet version. V5R1 is the default if version is empty.
func getWalletVersionConfig(version string) (wallet.VersionConfig, error) {
	switch version {
	case "V1R1":
		return wallet.V1R1, nil
	case "V1R2":
		return wallet.V1R2, nil
	case "V1R3":
		return wallet.V1R3, nil
	case "V2R1":
		return wallet.V2R1, nil
	case "V2R2":
		return wallet.V2R2, nil
	case "V3R1":
		return wallet.V3R1, nil
	case "V3R2":
		return wallet.V3R2, nil
	case "V4R1":
		return wallet.V4R1, nil
	case "V4R2":
		return wallet.V4R2, nil
	case "V5R1":
		return wallet.ConfigV5R1Beta{NetworkGlobalID: -239, Workchain: 0}, nil
	case "":
		return wallet.ConfigV5R1Beta{NetworkGlobalID: -239, Workchain: 0}, nil
	default:
		return nil, fmt.Errorf("unsupported wallet version: %s", version)
	}
}

// Name returns the name of the RPCChainProvider.
func (*RPCChainProvider) Name() string {
	return "TON RPC Chain Provider"
}

// ChainSelector returns the chain selector of the TON chain managed by this provider.
func (p *RPCChainProvider) ChainSelector() uint64 {
	return p.selector
}

// BlockChain returns the TON chain instance managed by this provider. You must call Initialize
// before using this method to ensure the chain is properly set up.
func (p *RPCChainProvider) BlockChain() chain.BlockChain {
	return *p.chain
}
