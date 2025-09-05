package provider

import (
	"math/big"
	"strconv"
	"testing"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/smartcontractkit/freeport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/tron"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/tron/provider/testdata"
)

func Test_RPCChainProviderConfig_validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		giveConfigFunc func(*RPCChainProviderConfig)
		wantErr        string
	}{
		{
			name: "valid config",
		},
		{
			name:           "missing full node url",
			giveConfigFunc: func(c *RPCChainProviderConfig) { c.FullNodeURL = "" },
			wantErr:        "full node url is required",
		},
		{
			name:           "missing solidity node url",
			giveConfigFunc: func(c *RPCChainProviderConfig) { c.SolidityNodeURL = "" },
			wantErr:        "solidity node url is required",
		},
		{
			name:           "missing deployer signer generator",
			giveConfigFunc: func(c *RPCChainProviderConfig) { c.DeployerSignerGen = nil },
			wantErr:        "deployer signer generator is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// A valid configuration for the RPCChainProviderConfig
			signerGen, err := SignerRandom()
			require.NoError(t, err)

			config := RPCChainProviderConfig{
				FullNodeURL:       "http://localhost:8090",
				SolidityNodeURL:   "http://localhost:8091",
				DeployerSignerGen: signerGen,
			}

			if tt.giveConfigFunc != nil {
				tt.giveConfigFunc(&config)
			}

			validationErr := config.validate()
			if tt.wantErr != "" {
				require.ErrorContains(t, validationErr, tt.wantErr)
			} else {
				require.NoError(t, validationErr)
			}
		})
	}
}

func Test_RPCChainProvider_Initialize(t *testing.T) {
	t.Parallel()

	var (
		chainSelector = chainsel.TEST_22222222222222222222222222222222222222222222.Selector
		existingChain = &tron.Chain{}
	)

	tests := []struct {
		name              string
		giveSelector      uint64
		giveConfigFunc    func(t *testing.T) RPCChainProviderConfig
		giveExistingChain *tron.Chain // Use this to simulate an already initialized chain
		wantErr           string
	}{
		{
			name:         "valid initialization",
			giveSelector: chainSelector,
			giveConfigFunc: func(t *testing.T) RPCChainProviderConfig {
				t.Helper()

				signerGen, err := SignerRandom()
				require.NoError(t, err)

				return RPCChainProviderConfig{
					FullNodeURL:       "http://localhost:8090",
					SolidityNodeURL:   "http://localhost:8091",
					DeployerSignerGen: signerGen,
				}
			},
		},
		{
			name:              "returns an already initialized chain",
			giveSelector:      chainSelector,
			giveExistingChain: existingChain,
		},
		{
			name:         "fails config validation",
			giveSelector: chainSelector,
			giveConfigFunc: func(t *testing.T) RPCChainProviderConfig {
				t.Helper()

				return RPCChainProviderConfig{}
			},
			wantErr: "invalid Tron RPC config",
		},
		{
			name:         "initialization with nil signer generator",
			giveSelector: chainSelector,
			giveConfigFunc: func(t *testing.T) RPCChainProviderConfig {
				t.Helper()

				return RPCChainProviderConfig{
					FullNodeURL:       "http://localhost:8090",
					SolidityNodeURL:   "http://localhost:8091",
					DeployerSignerGen: nil,
				}
			},
			wantErr: "deployer signer generator is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var config RPCChainProviderConfig
			if tt.giveConfigFunc != nil {
				config = tt.giveConfigFunc(t)
			}

			p := NewRPCChainProvider(tt.giveSelector, config)

			if tt.giveExistingChain != nil {
				p.chain = tt.giveExistingChain
			}

			got, err := p.Initialize(t.Context())
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, p.chain)

				gotChain, ok := got.(tron.Chain)
				require.True(t, ok, "expected got to be of type tron.Chain")

				// For the already initialized chain case, we can skip the rest of the checks
				if tt.giveExistingChain != nil {
					return
				}

				// Otherwise, check the fields of the chain

				assert.Equal(t, tt.giveSelector, gotChain.Selector)
				assert.NotNil(t, gotChain.Client)
				assert.Equal(t, config.FullNodeURL, gotChain.URL)
				assert.NotNil(t, gotChain.SignHash)

				assert.NotNil(t, gotChain.Address)

				assert.NotNil(t, gotChain.SendAndConfirm)
				assert.NotNil(t, gotChain.DeployContractAndConfirm)
				assert.NotNil(t, gotChain.TriggerContractAndConfirm)
			}
		})
	}
}

func Test_RPCChainProvider_Name(t *testing.T) {
	t.Parallel()

	p := RPCChainProvider{}
	assert.Equal(t, "Tron RPC Chain Provider", p.Name())
}

func Test_RPCChainProvider_ChainSelector(t *testing.T) {
	t.Parallel()

	p := RPCChainProvider{selector: chainsel.TRON_MAINNET.Selector}
	assert.Equal(t, chainsel.TRON_MAINNET.Selector, p.ChainSelector())
}

func Test_RPCChainProvider_BlockChain(t *testing.T) {
	t.Parallel()

	chain := &tron.Chain{}

	p := RPCChainProvider{
		chain: chain,
	}

	assert.Equal(t, *chain, p.BlockChain())
}

func Test_Tron_SendTransfer_And_DeployContract(t *testing.T) {
	t.Parallel()

	tronChain := setupLocalStack(t)

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
			t.Context(), "LinkToken", testdata.LinkTokenMetaDataABI, testdata.LinkTokenMetaDataBIN, nil, deployOptions)
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

func setupLocalStack(t *testing.T) *tron.Chain {
	t.Helper()

	var (
		attempts = uint(10)
		bc       *blockchain.Output
	)

	// Retry logic to handle port conflicts using retry.DoWithData
	bc, err := retry.DoWithData(func() (*blockchain.Output, error) {
		port := freeport.GetOne(t)

		output, rerr := blockchain.NewBlockchainNetwork(&blockchain.Input{
			Type:  blockchain.TypeTron,
			Port:  strconv.Itoa(port),
			Image: "tronbox/tre:dev", // dev supports arm (mac) and amd (ci)
		})
		if rerr != nil {
			// Return the ports to freeport to avoid leaking them during retries
			freeport.Return([]int{port})
			return nil, rerr
		}

		return output, nil
	},
		retry.Context(t.Context()),
		retry.Attempts(attempts),
		retry.Delay(1*time.Second),
		retry.DelayType(retry.FixedDelay),
		retry.OnRetry(func(attempt uint, err error) {
			t.Logf("Attempt %d/%d: Failed to start CTF TRON container: %v", attempt+1, attempts, err)
		}),
	)
	require.NoError(t, err, "Failed to start CTF TRON container after %d attempts", attempts)

	fullNodeUrl := bc.Nodes[0].ExternalHTTPUrl + "/wallet"
	solidityNodeUrl := bc.Nodes[0].ExternalHTTPUrl + "/walletsolidity"

	t.Logf("TRON node config: fullNodeUrl=%s, solidityNodeUrl=%s", fullNodeUrl, solidityNodeUrl)

	chainSelector := chainsel.TEST_22222222222222222222222222222222222222222222.Selector
	signerGenerator, err := SignerGenPrivateKey(blockchain.TRONAccounts.PrivateKeys[0])
	require.NoError(t, err)

	rpcClient := NewRPCChainProvider(chainSelector, RPCChainProviderConfig{
		FullNodeURL:       fullNodeUrl,
		SolidityNodeURL:   solidityNodeUrl,
		DeployerSignerGen: signerGenerator,
	})

	chain, err := rpcClient.Initialize(t.Context())
	require.NoError(t, err, "Failed to initialize Tron chain provider")

	tronChain, ok := chain.(tron.Chain)
	require.True(t, ok, "Expected chain to be of type tron.Chain")

	blockInfo, err := tronChain.Client.GetNowBlock()
	require.NoError(t, err, "Failed to get current block")

	blockId := blockInfo.BlockID
	chainIdHex := blockId[len(blockId)-8:]
	chainIdInt := new(big.Int)
	chainIdInt.SetString(chainIdHex, 16)
	chainId := chainIdInt.String()
	t.Logf("Read first block: chain id=%s", chainId)

	return &tronChain
}
