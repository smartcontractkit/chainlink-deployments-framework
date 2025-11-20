package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/xssnick/tonutils-go/liteclient"
	tonlib "github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/ton"
)

type WalletVersion string

// Allowed TON wallet versions
const (
	WalletVersionV3R2    WalletVersion = "V3R2"
	WalletVersionV4R2    WalletVersion = "V4R2"
	WalletVersionV5R1    WalletVersion = "V5R1"
	WalletVersionDefault WalletVersion = ""
)

// RPCChainProviderConfig holds the configuration to initialize the RPCChainProvider.
type RPCChainProviderConfig struct {
	// Required: The liteserver URL to connect to the Ton node (format: liteserver://publickey@host:port).
	HTTPURL string
	// Optional: The WebSocket URL to connect to the Ton node.
	WSURL string
	// Required: A generator for the deployer key. Use PrivateKeyFromRaw to create a deployer
	// key from a private key.
	DeployerSignerGen PrivateKeyGenerator
	// Optional: The TON wallet version to use. Supported versions are: V1R1, V1R2, V1R3, V2R1,
	// V2R2, V3R1, V3R2, V4R1, V4R2 and V5R1. If no value provided, V5R1 is used as default.
	WalletVersion WalletVersion
}

// validateLiteserverURL validates the format of a liteserver URL
func validateLiteserverURL(liteserverURL string) error {
	if liteserverURL == "" {
		return errors.New("liteserver url is required")
	}

	if !strings.HasPrefix(liteserverURL, "liteserver://") {
		return errors.New("invalid liteserver URL format: expected liteserver:// prefix")
	}

	// Remove the liteserver:// prefix
	urlPart := strings.TrimPrefix(liteserverURL, "liteserver://")

	// Split by @ to separate publickey and host:port
	parts := strings.Split(urlPart, "@")
	if len(parts) != 2 {
		return errors.New("invalid liteserver URL format: expected publickey@host:port")
	}

	publicKey := parts[0]
	hostPort := parts[1]

	if publicKey == "" {
		return errors.New("invalid liteserver URL format: public key cannot be empty")
	}

	if hostPort == "" {
		return errors.New("invalid liteserver URL format: host:port cannot be empty")
	}

	return nil
}

// validate checks if the RPCChainProviderConfig is valid.
func (c RPCChainProviderConfig) validate() error {
	if err := validateLiteserverURL(c.HTTPURL); err != nil {
		return err
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

// setupConnection creates and tests a connection to the TON liteserver
func setupConnection(ctx context.Context, liteserverURL string) (*tonlib.APIClient, error) {
	connectionPool, err := createLiteclientConnectionPool(ctx, liteserverURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to liteserver: %w", err)
	}

	api := tonlib.NewAPIClient(connectionPool, tonlib.ProofCheckPolicyFast)

	// Test connection and get current block
	mb, err := api.GetMasterchainInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get masterchain info: %w", err)
	}

	// Set starting point to verify master block proofs chain
	api.SetTrustedBlock(mb)

	return api, nil
}

// createWallet creates a TON wallet from the given private key and API client
func createWallet(api tonlib.APIClientWrapped, privateKey []byte, version WalletVersion) (*wallet.Wallet, error) {
	walletConfig, err := getWalletVersionConfig(version)
	if err != nil {
		return nil, fmt.Errorf("unsupported wallet version: %w", err)
	}

	tonWallet, err := wallet.FromPrivateKeyWithOptions(api, privateKey, walletConfig, wallet.WithWorkchain(0))
	if err != nil {
		return nil, fmt.Errorf("failed to init TON wallet: %w", err)
	}

	return tonWallet, nil
}

// Initialize initializes the RPCChainProvider.
func (p *RPCChainProvider) Initialize(ctx context.Context) (chain.BlockChain, error) {
	if p.chain != nil {
		return *p.chain, nil // Already initialized
	}

	if err := p.config.validate(); err != nil {
		return nil, fmt.Errorf("failed to validate provider config: %w", err)
	}

	// Setup connection to TON network
	api, err := setupConnection(ctx, p.config.HTTPURL)
	if err != nil {
		return nil, err
	}

	// Generate private key for wallet
	privateKey, err := p.config.DeployerSignerGen.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create wallet
	tonWallet, err := createWallet(api, privateKey, p.config.WalletVersion)
	if err != nil {
		return nil, err
	}

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

// createLiteclientConnectionPool creates connection pool returning concrete type for production use
func createLiteclientConnectionPool(ctx context.Context, liteserverURL string) (*liteclient.ConnectionPool, error) {
	// Validate URL format first
	if err := validateLiteserverURL(liteserverURL); err != nil {
		return nil, err
	}

	// Parse the URL
	urlPart := strings.TrimPrefix(liteserverURL, "liteserver://")
	parts := strings.Split(urlPart, "@")
	publicKey := parts[0]
	hostPort := parts[1]

	pool := liteclient.NewConnectionPool()
	err := pool.AddConnection(ctx, hostPort, publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to add liteserver connection: %w", err)
	}

	return pool, nil
}

// GetWalletVersionConfig returns the wallet version. V5R1 is the default if version is empty.
func getWalletVersionConfig(version WalletVersion) (wallet.VersionConfig, error) {
	switch version {
	case WalletVersionV3R2:
		return wallet.V3R2, nil
	case WalletVersionV4R2:
		return wallet.V4R2, nil
	case WalletVersionV5R1:
		return wallet.ConfigV5R1Final{NetworkGlobalID: wallet.MainnetGlobalID, Workchain: 0}, nil
	case WalletVersionDefault:
		return wallet.ConfigV5R1Final{NetworkGlobalID: wallet.MainnetGlobalID, Workchain: 0}, nil
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
