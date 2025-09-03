package provider

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/provider/rpcclient"
)

func Test_ConfirmFuncGeth_ConfirmFunc(t *testing.T) {
	t.Parallel()

	// Generate an admin transactor which will be prefunded with some ETH
	adminKey, err := crypto.GenerateKey()
	require.NoError(t, err, "failed to generate admin key")

	adminTransactor, err := bind.NewKeyedTransactorWithChainID(adminKey, simChainID)
	require.NoError(t, err)

	// Generate another user transactor which acts as a recipient
	userKey, err := crypto.GenerateKey()
	require.NoError(t, err, "failed to generate user key")

	userTransactor, err := bind.NewKeyedTransactorWithChainID(userKey, simChainID)
	require.NoError(t, err)

	// Prefund the admin account
	genesis := types.GenesisAlloc{
		adminTransactor.From: {Balance: prefundAmountWei},
	}

	tests := []struct {
		name    string
		giveTx  func(*testing.T, *SimClient) *types.Transaction
		wantErr string
	}{
		{
			name: "successful confirmation",
			giveTx: func(t *testing.T, client *SimClient) *types.Transaction {
				t.Helper()

				// Get the nonce
				nonce, err := client.PendingNonceAt(t.Context(), adminTransactor.From)
				require.NoError(t, err)

				gasPrice, err := client.SuggestGasPrice(t.Context())
				require.NoError(t, err)

				// Create a transaction to send tokens. This will be used to test the confirmation function.
				tx := types.NewTransaction(
					nonce, userTransactor.From, big.NewInt(10000000000000000), 21000, gasPrice, nil,
				)

				signedTx, err := types.SignTx(tx, types.NewCancunSigner(simChainID), adminKey)
				require.NoError(t, err, "failed to sign transaction")

				// Send the transaction
				err = client.SendTransaction(t.Context(), signedTx)
				require.NoError(t, err)

				client.Commit() // Commit the transaction to the simulated backend

				return signedTx
			},
		},
		{
			name: "failed with nil tx",
			giveTx: func(t *testing.T, client *SimClient) *types.Transaction {
				t.Helper()

				return nil
			},
			wantErr: "tx was nil",
		},
		{
			name: "failed with context deadline exceeded",
			giveTx: func(t *testing.T, client *SimClient) *types.Transaction {
				t.Helper()

				// Get the nonce
				nonce, err := client.PendingNonceAt(t.Context(), adminTransactor.From)
				require.NoError(t, err)

				gasPrice, err := client.SuggestGasPrice(t.Context())
				require.NoError(t, err)

				// Create a transaction to send tokens. This will be used to test the confirmation function.
				tx := types.NewTransaction(
					nonce, userTransactor.From, big.NewInt(10000000000000000), 21000, gasPrice, nil,
				)

				return tx
			},
			wantErr: "context deadline exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Initialize the simulated backend with the genesis state
			backend := simulated.NewBackend(genesis, simulated.WithBlockGasLimit(50000000))
			backend.Commit() // Commit the genesis block

			// Wrap the backend in a SimClient for easier testing because it has access to more
			// methods.
			client := NewSimClient(t, backend)

			// Generate the transaction to confirm
			tx := tt.giveTx(t, client)

			// Generate the confirm function
			functor := ConfirmFuncGeth(1 * time.Second)
			confirmFunc, err := functor.Generate(
				t.Context(), chain_selectors.TEST_1000.Selector, client, adminTransactor.From,
			)
			require.NoError(t, err)

			// Run the confirm function with the transaction
			_, err = confirmFunc(tx)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

//nolint:paralleltest // This test cannot run in parallel due to a race condition in seth's log initialization
func Test_ConfirmFuncSeth_Generate(t *testing.T) {
	rpcSrv := newFakeRPCServer(t)

	var (
		chainSelector = chain_selectors.TEST_1000.Selector
		rpcURL        = rpcSrv.URL
		fromAddr      = common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")

		configPath        = writeSethConfigFile(t)
		invalidConfigPath = writeInvalidSethConfigFile(t)
	)

	tests := []struct {
		name           string
		giveRPCURL     string
		giveSelector   uint64
		giveClient     evm.OnchainClient
		giveConfigPath string
		wantErr        string
	}{
		{
			name:           "valid generation confirmation function",
			giveRPCURL:     rpcURL,
			giveSelector:   chainSelector,
			giveClient:     &rpcclient.MultiClient{},
			giveConfigPath: configPath,
		},
		{
			name:           "invalid client type",
			giveRPCURL:     rpcURL,
			giveSelector:   chainSelector,
			giveClient:     SimClient{},
			giveConfigPath: configPath,
			wantErr:        "expected client to be of type *rpcclient.MultiClient",
		},
		{
			name:           "invalid chain ID",
			giveRPCURL:     rpcURL,
			giveSelector:   1,
			giveClient:     &rpcclient.MultiClient{},
			giveConfigPath: configPath,
			wantErr:        "failed to get chain ID from selector",
		},
		{
			name:           "failed to setup seth client",
			giveRPCURL:     "http://invalid-url",
			giveSelector:   chainSelector,
			giveClient:     &rpcclient.MultiClient{},
			giveConfigPath: invalidConfigPath,
			wantErr:        "failed to setup seth client",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // This test cannot run in parallel due to a race condition in seth's log initialization
			functor := ConfirmFuncSeth(tt.giveRPCURL, 1*time.Second, []string{}, tt.giveConfigPath)

			got, err := functor.Generate(
				t.Context(), tt.giveSelector, tt.giveClient, fromAddr,
			)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got, "expected a non-nil confirmation function")
			}
		})
	}
}
