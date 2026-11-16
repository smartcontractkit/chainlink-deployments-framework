package provider

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	sollib "github.com/gagliardetto/solana-go"
	solrpc "github.com/gagliardetto/solana-go/rpc"
	solCommonUtil "github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/common"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/solana"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/solana/provider/rpcclient"
)

// RPCChainProviderConfig holds the configuration to initialize the RPCChainProvider.
type RPCChainProviderConfig struct {
	// Required: The HTTP RPC URL to connect to the Solana node.
	HTTPURL string
	// Optional: The WebSocket URL to connect to the Solana node.
	WSURL string
	// Required unless DeployerSignerGen is set: A generator for the deployer key. Use
	// PrivateKeyFromRaw to create a deployer key from a private key. When DeployerSignerGen is
	// set, this key is only used by the Solana CLI to deploy programs.
	DeployerKeyGen PrivateKeyGenerator
	// Optional: A generator for a remote deployer signer, e.g. SignerGenKMS. When set, all
	// transactions are signed by this signer instead of the key from DeployerKeyGen.
	DeployerSignerGen SignerGenerator
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
	if c.DeployerKeyGen == nil && c.DeployerSignerGen == nil {
		return errors.New("deployer key generator or deployer signer generator is required")
	}
	if c.ProgramsPath == "" {
		return errors.New("programs path is required")
	}
	if c.KeypairDirPath == "" {
		return errors.New("keypair path is required")
	}

	if err := isValidFilepath(c.ProgramsPath); err != nil {
		return err
	}

	if err := isValidFilepath(c.KeypairDirPath); err != nil {
		return err
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

	// chain is the Solana chain instance that this provider manages. The Initialize method
	// sets up the chain.
	chain *solana.Chain
}

func NewRPCChainProvider(selector uint64, config RPCChainProviderConfig) *RPCChainProvider {
	return &RPCChainProvider{
		selector: selector,
		config:   config,
	}
}

// Initialize initializes the RPCChainProvider. It generates the deployer keypair and/or signer
// from the provided configuration, writes the keypair to the KeypairDirPath directory for the
// Solana CLI, and sets up the Solana client with the provided HTTP RPC URL. It returns the
// initialized Solana chain instance.
func (p *RPCChainProvider) Initialize(ctx context.Context) (chain.BlockChain, error) {
	if p.chain != nil {
		return *p.chain, nil // Already initialized
	}

	// Validate the provider configuration
	if err := p.config.validate(); err != nil {
		return nil, fmt.Errorf("failed to validate provider config: %w", err)
	}

	var (
		privKey     *sollib.PrivateKey
		keypairPath string
	)
	if p.config.DeployerKeyGen != nil {
		// Generate the deployer keypair
		key, err := p.config.DeployerKeyGen.Generate()
		if err != nil {
			return nil, fmt.Errorf("failed to generate deployer keypair: %w", err)
		}

		// Persist the deployer keypair to the KeypairDirPath for the Solana CLI to use
		keypairPath = filepath.Join(p.config.KeypairDirPath, "authority-keypair.json")
		if err = WritePrivateKeyToPath(keypairPath, key); err != nil {
			return nil, fmt.Errorf("failed to write deployer keypair to file: %w", err)
		}

		privKey = &key
	}

	// Initialize the Solana client with the provided HTTP RPC URL
	var client *rpcclient.Client
	if p.config.DeployerSignerGen != nil {
		signer, err := p.config.DeployerSignerGen.Generate(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to generate deployer signer: %w", err)
		}

		client = rpcclient.NewWithSigner(solrpc.New(p.config.HTTPURL), signer)
		// The local key, if any, is only used by the Solana CLI and must not sign as the deployer.
		privKey = nil
	} else {
		client = rpcclient.New(solrpc.New(p.config.HTTPURL), *privKey)
	}

	// Create the Solana chain instance with the provided configuration
	p.chain = &solana.Chain{
		Selector:       p.selector,
		Client:         client.Client,
		URL:            p.config.HTTPURL,
		WSURL:          p.config.WSURL,
		DeployerKey:    privKey,
		DeployerSigner: client.Signer,
		ProgramsPath:   p.config.ProgramsPath,
		KeypairPath:    keypairPath,
		SendAndConfirm: func(
			ctx context.Context, instructions []sollib.Instruction, txMods ...rpcclient.TxModifier,
		) error {
			_, err := client.SendAndConfirmTx(ctx, instructions,
				rpcclient.WithTxModifiers(txMods...),
				rpcclient.WithRetry(1, 50*time.Millisecond),
			)

			return err
		},
		Confirm: func(instructions []sollib.Instruction, opts ...solCommonUtil.TxModifier) error {
			if privKey == nil {
				// The legacy helper requires a private key, so a remote deployer sends through
				// the RPC client instead.
				txMods := make([]rpcclient.TxModifier, len(opts))
				for i, o := range opts {
					txMods[i] = rpcclient.TxModifier(o)
				}
				_, err := client.SendAndConfirmTx(context.Background(), instructions,
					rpcclient.WithTxModifiers(txMods...),
					rpcclient.WithRetry(1, 50*time.Millisecond),
					rpcclient.WithCommitment(solrpc.CommitmentFinalized),
				)

				return err
			}

			_, err := solCommonUtil.SendAndConfirm(
				context.Background(), client.Client, instructions, *privKey, solrpc.CommitmentFinalized, opts...,
			)

			return err
		},
	}

	return *p.chain, nil
}

// Name returns the name of the RPCChainProvider.
func (*RPCChainProvider) Name() string {
	return "Solana RPC Chain Provider"
}

// ChainSelector returns the chain selector of the Solana chain managed by this provider.
func (p *RPCChainProvider) ChainSelector() uint64 {
	return p.selector
}

// BlockChain returns the Solana chain instance managed by this provider. You must call Initialize
// before using this method to ensure the chain is properly set up.
func (p *RPCChainProvider) BlockChain() chain.BlockChain {
	return *p.chain
}
