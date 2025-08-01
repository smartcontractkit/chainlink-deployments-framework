package provider

import (
	"context"
	"sync"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-evm/gethwrappers/shared/generated/link_token"
	"github.com/smartcontractkit/chainlink-testing-framework/lib/logging"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/tron"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/tron/provider/rpcclient"
)

func TestCTFChainProviderConfig_validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      CTFChainProviderConfig
		expectedErr string
	}{
		{
			name:        "empty config",
			config:      CTFChainProviderConfig{},
			expectedErr: "deployer account generator is required",
		},
		{
			name: "missing sync.Once",
			config: CTFChainProviderConfig{
				DeployerAccountGen: AccountGenCTFDefault(),
			},
			expectedErr: "sync.Once instance is required",
		},
		{
			name: "valid config",
			config: CTFChainProviderConfig{
				DeployerAccountGen: AccountGenCTFDefault(),
				Once:               &sync.Once{},
			},
			expectedErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.config.validate()
			if tt.expectedErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErr)
			}
		})
	}
}

func TestNewCTFChainProvider(t *testing.T) {
	t.Parallel()

	config := CTFChainProviderConfig{
		DeployerAccountGen: AccountGenCTFDefault(),
		Once:               &sync.Once{},
	}

	provider := NewCTFChainProvider(t, 123456, config)
	require.NotNil(t, provider)
	require.Equal(t, uint64(123456), provider.selector)
	require.Equal(t, config, provider.config)
	require.Equal(t, t, provider.t)
	require.Nil(t, provider.chain)
}

func TestCTFChainProvider_Initialize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		giveSelector uint64
		giveConfig   CTFChainProviderConfig
		wantErr      string
	}{
		{
			name:         "valid initialization",
			giveSelector: chain_selectors.TRON_TESTNET_NILE.Selector,
			giveConfig: CTFChainProviderConfig{
				DeployerAccountGen: AccountGenCTFDefault(),
				Once:               &sync.Once{},
			},
		},
		{
			name:         "fails config validation",
			giveSelector: chain_selectors.TRON_TESTNET_NILE.Selector,
			giveConfig: CTFChainProviderConfig{
				Once: &sync.Once{},
			},
			wantErr: "deployer account generator is required",
		},
		{
			name:         "missing sync.Once",
			giveSelector: chain_selectors.TRON_TESTNET_NILE.Selector,
			giveConfig: CTFChainProviderConfig{
				DeployerAccountGen: AccountGenCTFDefault(),
			},
			wantErr: "sync.Once instance is required",
		},
		{
			name:         "chain id not found for selector",
			giveSelector: 999999, // Invalid selector
			giveConfig: CTFChainProviderConfig{
				DeployerAccountGen: AccountGenCTFDefault(),
				Once:               &sync.Once{},
			},
			wantErr: "failed to get chain ID from selector 999999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := NewCTFChainProvider(t, tt.giveSelector, tt.giveConfig)

			got, err := p.Initialize(context.Background())
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.NotNil(t, p.chain)

				// Check that the chain is of type tron.Chain and has the expected fields
				gotChain, ok := got.(tron.Chain)
				require.True(t, ok, "expected got to be of type tron.Chain")
				require.Equal(t, tt.giveSelector, gotChain.Selector)
				require.NotEmpty(t, gotChain.Client)
				require.NotEmpty(t, gotChain.Keystore)
				require.NotEmpty(t, gotChain.Address)
				require.NotEmpty(t, gotChain.URL)
			}
		})
	}
}

func TestCTFChainProvider_ContainerStartup(t *testing.T) {
	t.Parallel()
	config := CTFChainProviderConfig{
		DeployerAccountGen: AccountGenCTFDefault(),
		Once:               &sync.Once{},
	}

	provider := NewCTFChainProvider(t, chain_selectors.TRON_TESTNET_NILE.Selector, config)

	chainID, err := chain_selectors.GetChainIDFromSelector(chain_selectors.TRON_MAINNET.Selector)
	require.NoError(t, err)
	fullNodeURL, solidityNodeURL := provider.startContainer(chainID)
	require.NotEmpty(t, fullNodeURL)
	require.NotEmpty(t, solidityNodeURL)
	require.Contains(t, fullNodeURL, "/wallet")
	require.Contains(t, solidityNodeURL, "/walletsolidity")
}

func TestCTFProvider_SendAndConfirmTx_And_CheckContractDeployed(t *testing.T) {
	t.Parallel()

	logger := logging.GetTestLogger(t)

	config := CTFChainProviderConfig{
		DeployerAccountGen: AccountGenCTFDefault(),
		Once:               &sync.Once{},
	}

	chainSelector := chain_selectors.TRON_TESTNET_NILE.Selector

	// Create and initialize the CTF provider
	ctfProvider := NewCTFChainProvider(t, chainSelector, config)
	chainInstance, err := ctfProvider.Initialize(context.Background())
	require.NoError(t, err, "Failed to initialize CTF provider")

	// Extract the TRON chain from the interface
	tronChain, ok := chainInstance.(tron.Chain)
	require.True(t, ok, "Expected TRON chain instance")

	logger.Info().Str("chainURL", tronChain.URL).Uint64("selector", tronChain.Selector).Msg("TRON CTF chain initialized")

	// Create RPC client using the chain's components
	rpcClient := rpcclient.New(tronChain.Client, tronChain.Keystore, tronChain.Address)

	// Set deploy options, including custom fee limit for local deployment
	deployOptions := tron.DefaultDeployOptions()
	deployOptions.FeeLimit = 1_000_000_000

	deployResponse, err := rpcClient.Client.DeployContract(rpcClient.Account, "LinkToken", link_token.LinkTokenABI, link_token.LinkTokenBin, deployOptions.OeLimit, deployOptions.CurPercent, deployOptions.FeeLimit, nil)
	require.NoError(t, err, "Failed to create deploy contract transaction")

	txInfo, err := rpcClient.SendAndConfirmTx(context.Background(), &deployResponse.Transaction, deployOptions.ConfirmRetryOptions)
	require.NoError(t, err, "Failed to send and confirm transaction")

	logger.Info().Str("txID", txInfo.ID).Msg("Transaction ID")
	logger.Info().Any("receipt", txInfo.Receipt).Msg("Transaction receipt")
	logger.Info().Str("contract address", txInfo.ContractAddress).Msg("Deployed contract")

	contractAddress, err := address.StringToAddress(txInfo.ContractAddress)
	require.NoError(t, err, "Failed to parse contract address from transaction info")

	err = rpcClient.CheckContractDeployed(contractAddress)
	require.NoError(t, err, "Contract deployment check failed")
}
