package provider

import (
	"context"
	"sync"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-evm/gethwrappers/shared/generated/initial/link_token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/tron"
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
			expectedErr: "deployer signer generator is required",
		},
		{
			name: "missing sync.Once",
			config: func() CTFChainProviderConfig {
				signerGen, err := SignerGenCTFDefault()
				require.NoError(t, err)

				return CTFChainProviderConfig{
					DeployerSignerGen: signerGen,
				}
			}(),
			expectedErr: "sync.Once instance is required",
		},
		{
			name: "valid config",
			config: func() CTFChainProviderConfig {
				signerGen, err := SignerGenCTFDefault()
				require.NoError(t, err)

				return CTFChainProviderConfig{
					DeployerSignerGen: signerGen,
					Once:              &sync.Once{},
				}
			}(),
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

	signerGen, err := SignerGenCTFDefault()
	require.NoError(t, err)

	config := CTFChainProviderConfig{
		DeployerSignerGen: signerGen,
		Once:              &sync.Once{},
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
			giveConfig: func() CTFChainProviderConfig {
				signerGen, err := SignerGenCTFDefault()
				require.NoError(t, err)

				return CTFChainProviderConfig{
					DeployerSignerGen: signerGen,
					Once:              &sync.Once{},
				}
			}(),
		},
		{
			name:         "fails config validation",
			giveSelector: chain_selectors.TRON_TESTNET_NILE.Selector,
			giveConfig: CTFChainProviderConfig{
				Once: &sync.Once{},
			},
			wantErr: "deployer signer generator is required",
		},
		{
			name:         "missing sync.Once",
			giveSelector: chain_selectors.TRON_TESTNET_NILE.Selector,
			giveConfig: func() CTFChainProviderConfig {
				signerGen, err := SignerGenCTFDefault()
				require.NoError(t, err)

				return CTFChainProviderConfig{
					DeployerSignerGen: signerGen,
				}
			}(),
			wantErr: "sync.Once instance is required",
		},
		{
			name:         "chain id not found for selector",
			giveSelector: 999999, // Invalid selector
			giveConfig: func() CTFChainProviderConfig {
				signerGen, err := SignerGenCTFDefault()
				require.NoError(t, err)

				return CTFChainProviderConfig{
					DeployerSignerGen: signerGen,
					Once:              &sync.Once{},
				}
			}(),
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
				require.NotEmpty(t, gotChain.SignHash)
				require.NotEmpty(t, gotChain.Address)
				require.NotEmpty(t, gotChain.URL)
				require.NotEmpty(t, gotChain.SendAndConfirm)
				require.NotEmpty(t, gotChain.DeployContractAndConfirm)
				require.NotEmpty(t, gotChain.TriggerContractAndConfirm)
			}
		})
	}
}

func TestCTFChainProvider_ContainerStartup(t *testing.T) {
	t.Parallel()

	signerGen, err := SignerGenCTFDefault()
	require.NoError(t, err)

	config := CTFChainProviderConfig{
		DeployerSignerGen: signerGen,
		Once:              &sync.Once{},
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

	signerGen, err := SignerGenCTFDefault()
	require.NoError(t, err)

	config := CTFChainProviderConfig{
		DeployerSignerGen: signerGen,
		Once:              &sync.Once{},
	}

	chainSelector := chain_selectors.TRON_TESTNET_NILE.Selector

	// Create and initialize the CTF provider
	ctfProvider := NewCTFChainProvider(t, chainSelector, config)
	chainInstance, err := ctfProvider.Initialize(context.Background())
	require.NoError(t, err, "Failed to initialize CTF provider")

	// Extract the TRON chain from the interface
	tronChain, ok := chainInstance.(tron.Chain)
	require.True(t, ok, "Expected TRON chain instance")

	t.Logf("TRON CTF chain initialized: chainURL=%s, selector=%d", tronChain.URL, tronChain.Selector)

	//nolint:paralleltest // this subtest shares a local Tron node and must not run in parallel
	t.Run("SendTrxWithSendAndConfirm", func(t *testing.T) {
		// Generate a random receiver address
		receiverAddress, err := address.Base58ToAddress("TQtWBxe8wNAcio3evcfwMAqsdFzykpi6e7")
		require.NoError(t, err, "Failed to generate receiver address")

		t.Logf("Generated receiver address: receiver=%s", receiverAddress.String())

		// Query receiver balance before transfer
		beforeAccount, err := tronChain.Client.GetAccount(receiverAddress)
		require.NoError(t, err, "Failed to fetch receiver account before transfer")
		beforeBalance := beforeAccount.Balance
		t.Logf("Receiver balance before transfer: before balance=%d", beforeBalance)

		// Amount to transfer (1 TRX = 1_000_000 SUN)
		const amount int64 = 1_000_000 // 1 TRX

		// Create transfer transaction
		tx, err := tronChain.Client.Transfer(tronChain.Address, receiverAddress, amount)
		require.NoError(t, err, "Failed to create transfer transaction")

		// Send and confirm transaction with default options
		confirmRetryOptions := tron.DefaultConfirmRetryOptions()
		txInfo, err := tronChain.SendAndConfirm(t.Context(), tx, confirmRetryOptions)
		require.NoError(t, err, "Failed to send and confirm TRX transfer")

		t.Logf("Transfer transaction ID: txID=%s", txInfo.ID)
		t.Logf("Transfer transaction receipt: receipt=%+v", txInfo.Receipt)

		// Query receiver balance after transfer
		afterAccount, err := tronChain.Client.GetAccount(receiverAddress)
		require.NoError(t, err, "Failed to fetch receiver account after transfer")
		afterBalance := afterAccount.Balance
		t.Logf("Receiver balance after transfer: after balance=%d", afterBalance)

		// Assert balance increased by expected amount
		expectedBalance := beforeBalance + amount
		require.GreaterOrEqual(t, afterBalance, expectedBalance, "Receiver balance should have increased by the transferred amount")
	})

	//nolint:paralleltest // this subtest shares a local Tron node and must not run in parallel
	t.Run("DeployAndTriggerLinkContract", func(t *testing.T) {
		// Set deploy options, including custom fee limit for local deployment
		deployOptions := tron.DefaultDeployOptions()
		deployOptions.FeeLimit = 1_000_000_000

		// Deploy the LinkToken contract and wait for confirmation
		contractAddress, txInfo, err := tronChain.DeployContractAndConfirm(
			t.Context(), "LinkToken", link_token.LinkTokenABI, link_token.LinkTokenBin, nil, deployOptions)
		require.NoError(t, err, "Failed to deploy contract")

		// Log deployed contract address and deployment transaction details
		t.Logf("Deployed contract: contract address=%s", contractAddress.String())
		t.Logf("Deploy transaction ID: transaction id=%s", txInfo.ID)
		t.Logf("Deploy transaction result: receipt=%+v", txInfo.Receipt)

		// Log the address used to deploy contracts (chain address)
		t.Logf("Using chain address: chain address=%s", tronChain.Address.String())

		// Generate a random minter address
		minterAddress, err := address.Base58ToAddress("TQtWBxe8wNAcio3evcfwMAqsdFzykpi6e7")
		require.NoError(t, err, "Failed to generate minter address")

		// Check the minter role status before granting it
		beforeMinterResp, err := tronChain.Client.TriggerConstantContract(
			tronChain.Address, contractAddress, "isMinter(address)", []interface{}{"address", minterAddress})
		require.NoError(t, err, "Failed to check if minter is set before granting role")
		t.Logf("Before minter response: response=%+v", beforeMinterResp)

		// Assert minter role is initially false (not granted)
		require.Equal(t,
			"0000000000000000000000000000000000000000000000000000000000000000",
			beforeMinterResp.ConstantResult[0],
			"Minter should be set to false",
		)

		triggerOptions := tron.DefaultTriggerOptions()

		// Grant the minter role to the specified minter address and wait for confirmation
		grantMintResp, err := tronChain.TriggerContractAndConfirm(
			t.Context(), contractAddress, "grantMintRole(address)", []interface{}{"address", minterAddress}, triggerOptions)
		require.NoError(t, err, "Failed to grant mint role")

		// Log the transaction details for granting mint role
		t.Logf("Grant mint transaction ID: transaction id=%s", grantMintResp.ID)
		t.Logf("Grant mint transaction result: receipt=%+v", grantMintResp.Receipt)

		// Check the minter role status after granting it
		afterMinterResp, err := tronChain.Client.TriggerConstantContract(
			tronChain.Address, contractAddress, "isMinter(address)", []interface{}{"address", minterAddress})
		require.NoError(t, err, "Failed to check if minter is set after granting role")
		t.Logf("After minter response: response=%+v", afterMinterResp)

		// Assert minter role is now true (successfully granted)
		require.Equal(t,
			"0000000000000000000000000000000000000000000000000000000000000001",
			afterMinterResp.ConstantResult[0],
			"Minter should be set to true",
		)
	})
}

func Test_CTFChainProvider_Name(t *testing.T) {
	t.Parallel()

	p := CTFChainProvider{}
	assert.Equal(t, "TRON CTF Chain Provider", p.Name())
}

func Test_CTFChainProvider_ChainSelector(t *testing.T) {
	t.Parallel()

	p := CTFChainProvider{selector: chain_selectors.TRON_MAINNET.Selector}
	assert.Equal(t, chain_selectors.TRON_MAINNET.Selector, p.ChainSelector())
}

func Test_CTFChainProvider_BlockChain(t *testing.T) {
	t.Parallel()

	chain := &tron.Chain{}

	p := CTFChainProvider{
		chain: chain,
	}

	assert.Equal(t, *chain, p.BlockChain())
}
