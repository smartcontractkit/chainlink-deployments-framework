package provider

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	solrpc "github.com/gagliardetto/solana-go/rpc"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/solana"
)

// RPCChainProviderConfig holds the configuration to initialize the RPCChainProvider.
type RPCChainProviderConfig struct {
	// Required: The HTTP RPC URL to connect to the Solana node
	HTTPURL string
	// Required: A generator for the deployer key. Use PrivateKeyFromRaw to create a deployer
	// key from a private key.
	DeployerKeyGen PrivateKeyGenerator
	// Required: The absolute path to the directory containing the Solana CLI binaries.
	ProgramsPath string
	// Required: The absolute path to the directory that will contain the keypair file used for
	// deploying programs for the Solana CLI.
	KeypairDirPath string
}

// validate checks if the RPCChainProviderConfig is valid.
func (c RPCChainProviderConfig) validate() error {
	if c.HTTPURL == "" {
		return errors.New("http url is required")
	}
	if c.DeployerKeyGen == nil {
		return errors.New("deployer key generator is required")
	}
	if c.ProgramsPath == "" {
		return errors.New("programs path is required")
	}
	if c.KeypairDirPath == "" {
		return errors.New("keypair path is required")
	}

	if err := c.isValidFilepath(c.ProgramsPath); err != nil {
		return err
	}

	if err := c.isValidFilepath(c.KeypairDirPath); err != nil {
		return err
	}

	return nil
}

// isValidFilepath checks if the provided file path exists and is absolute.
func (c RPCChainProviderConfig) isValidFilepath(fp string) error {
	_, err := os.Stat(fp)
	if os.IsNotExist(err) {
		return fmt.Errorf("required file does not exist: %s", fp)
	}

	if !filepath.IsAbs(fp) {
		return fmt.Errorf("required file is not absolute: %s", fp)
	}

	return nil
}

var _ chain.Provider = (*RPCChainProvider)(nil)

// RPCChainProvider is a chain provider that provides a chain that connects to an Solana node via
// RPC.
type RPCChainProvider struct {
	// Solana chain selector, used to identify the chain.
	selector uint64

	// RPCChainProviderConfig holds the configuration for the RPCChainProvider.
	config RPCChainProviderConfig

	// chain is the Aptos chain instance that this provider manages. The Initialize method
	// sets up the chain.
	chain *solana.Chain
}

func NewRPCChainProvider(selector uint64, config RPCChainProviderConfig) *RPCChainProvider {
	return &RPCChainProvider{
		selector: selector,
		config:   config,
	}
}

// Initialize initializes the RPCChainProvider. It generates the deployer keypair from the provided
// configuration, writes it to the specified KeypairPath directory, and sets up the Solana client
// with the provided HTTP RPC URL. It returns the initialized Solana chain instance.
func (p *RPCChainProvider) Initialize() (chain.BlockChain, error) {
	if p.chain != nil {
		return p.chain, nil // Already initialized
	}

	// Validate the provider configuration
	if err := p.config.validate(); err != nil {
		return nil, fmt.Errorf("failed to validate provider config: %w", err)
	}

	// Generate the deployer keypair
	privKey, err := p.config.DeployerKeyGen.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate deployer keypair: %w", err)
	}

	// Persist the deployer keypair to the KeypairDirPath for the Solana CLI to use
	keypairPath := filepath.Join(p.config.KeypairDirPath, "authority-keypair.json")
	if err := writePrivateKeyToPath(keypairPath, privKey); err != nil {
		return nil, fmt.Errorf("failed to write deployer keypair to file: %w", err)
	}

	// Initialize the Solana client with the provided HTTP RPC URL
	client := solrpc.New(p.config.HTTPURL)

	// Create the Solana chain instance with the provided configuration
	p.chain = &solana.Chain{
		Selector:     p.selector,
		Client:       client,
		URL:          p.config.HTTPURL,
		DeployerKey:  &privKey,
		ProgramsPath: p.config.ProgramsPath,
		KeypairPath:  keypairPath,
	}

	return *p.chain, nil
}

// Name returns the name of the RPCChainProvider.
func (*RPCChainProvider) Name() string {
	return "Solana RPC Chain Provider"
}

// ChainSelector returns the chain selector of the Aptos chain managed by this provider.
func (p *RPCChainProvider) ChainSelector() uint64 {
	return p.selector
}

// BlockChain returns the Aptos chain instance managed by this provider. You must call Initialize
// before using this method to ensure the chain is properly set up.
func (p *RPCChainProvider) BlockChain() chain.BlockChain {
	return p.chain
}
