package rpcclient

import (
	"context"
	"encoding/hex"
	"math/big"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-evm/gethwrappers/shared/generated/link_token"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/smartcontractkit/chainlink-testing-framework/lib/logging"
	"github.com/smartcontractkit/chainlink-tron/relayer/sdk"
	"github.com/smartcontractkit/freeport"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/tron"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/tron/keystore"
)

func TestConfirmRetryOpts_DefaultsAndOverrides(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Test default options
	opts := confirmRetryOpts(ctx, tron.DefaultConfirmRetryOptions())
	require.Len(t, opts, 4)

	// Confirm context is set correctly
	var hasCtx bool
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		hasCtx = true
	}
	require.True(t, hasCtx)

	// Test with custom options
	customOpts := confirmRetryOpts(ctx, tron.ConfirmRetryOptions{
		RetryAttempts: 3,
		RetryDelay:    50 * time.Millisecond,
	})
	require.Len(t, customOpts, 4)
}

func TestNewClient(t *testing.T) {
	t.Parallel()

	dummyAddr := address.Address{}
	cli := New(nil, nil, dummyAddr)
	require.NotNil(t, cli)
	require.Equal(t, dummyAddr, cli.Account)
	require.Nil(t, cli.Client)
	require.Nil(t, cli.Keystore)
}

func Test_Tron_SendAndConfirmTx_And_CheckContractDeployed(t *testing.T) {
	t.Parallel()

	logger := logging.GetTestLogger(t)
	rpcClient := setupLocalStack(t, logger)

	// Set deploy options, including custom fee limit for local deployment
	deployOptions := tron.DefaultDeployOptions()
	deployOptions.FeeLimit = 1_000_000_000

	deployResponse, err := rpcClient.Client.DeployContract(rpcClient.Account, "LinkToken", link_token.LinkTokenABI, link_token.LinkTokenBin, deployOptions.OeLimit, deployOptions.CurPercent, deployOptions.FeeLimit, nil)
	require.NoError(t, err, "Failed to create deploy contract transaction")

	txInfo, err := rpcClient.SendAndConfirmTx(t.Context(), &deployResponse.Transaction, deployOptions.ConfirmRetryOptions)
	require.NoError(t, err, "Failed to send and confirm transaction")

	logger.Info().Str("txID", txInfo.ID).Msg("Transaction ID")
	logger.Info().Any("receipt", txInfo.Receipt).Msg("Transaction receipt")
	logger.Info().Str("contract address", txInfo.ContractAddress).Msg("Deployed contract")

	contractAddress, err := address.StringToAddress(txInfo.ContractAddress)
	require.NoError(t, err, "Failed to parse contract address from transaction info")

	err = rpcClient.CheckContractDeployed(contractAddress)
	require.NoError(t, err, "Contract deployment check failed")
}

func setupLocalStack(t *testing.T, logger zerolog.Logger) *Client {
	t.Helper()

	port := freeport.GetOne(t)
	bc, err := blockchain.NewBlockchainNetwork(&blockchain.Input{
		Type:  blockchain.TypeTron,
		Port:  strconv.Itoa(port),
		Image: "tronbox/tre:dev", // dev supports arm (mac) and amd (ci)
	})
	require.NoError(t, err, "Failed to create blockchain network")

	fullNodeUrl := bc.Nodes[0].ExternalHTTPUrl + "/wallet"
	solidityNodeUrl := bc.Nodes[0].ExternalHTTPUrl + "/walletsolidity"

	logger.Info().Str("fullNodeUrl", fullNodeUrl).Str("solidityNodeUrl", solidityNodeUrl).Msg("TRON node config")

	fullNodeUrlObj, err := url.Parse(fullNodeUrl)
	require.NoError(t, err, "Failed to parse full node URL")

	solidityNodeUrlObj, err := url.Parse(solidityNodeUrl)
	require.NoError(t, err, "Failed to parse solidity node URL")

	combinedClient, err := sdk.CreateCombinedClient(fullNodeUrlObj, solidityNodeUrlObj)
	require.NoError(t, err, "Failed to create combined client")

	// Decode the hex-encoded private key string
	privBytes, err := hex.DecodeString(blockchain.TRONAccounts.PrivateKeys[0])
	require.NoError(t, err, "Failed to decode private key bytes")

	// Parse the bytes into an *ecdsa.PrivateKey
	privKey, err := crypto.ToECDSA(privBytes)
	require.NoError(t, err, "Failed to parse private key")

	keystore, addr := keystore.NewKeystore(privKey)

	rpcClient := New(combinedClient, keystore, addr)

	blockInfo, err := rpcClient.Client.GetNowBlock()
	require.NoError(t, err, "Failed to get current block")

	blockId := blockInfo.BlockID
	chainIdHex := blockId[len(blockId)-8:]
	chainIdInt := new(big.Int)
	chainIdInt.SetString(chainIdHex, 16)
	chainId := chainIdInt.String()
	logger.Info().Str("chain id", chainId).Msg("Read first block")

	return rpcClient
}
