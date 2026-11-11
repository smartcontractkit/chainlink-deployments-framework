package layout

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/smartcontractkit/ccip-owner-contracts/gethwrappers"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/simulation/statediff"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func TestChangeAddressSlotHonorsCancellation(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		<-r.Context().Done()
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Millisecond)
	defer cancel()
	started := time.Now()
	err := ChangeAddressSlotContext(ctx, logger.Nop(), MCMSLayout, server.URL, "_owner", "0x1", "0x2")
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Less(t, time.Since(started), time.Second)
}

// This test uses real MCMS bytecode on an isolated local Anvil process. In
// particular, the self-target case authors the exact temporary signer config:
// comparing a baseline taken after surgery would incorrectly show no change.
func TestTemporaryMCMSignerPreservesRootAndAuthoredEffects(t *testing.T) {
	t.Parallel()
	for _, scenario := range []string{"no authored config change", "self target equals temporary signer", "other MCMS target"} {
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()
			rpcURL, client := startAnvil(t)
			privateKey, err := crypto.HexToECDSA(blockchain.DefaultAnvilPrivateKey)
			require.NoError(t, err)
			auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(1337))
			require.NoError(t, err)
			auth.Context = t.Context()

			mcmAddress, mcmContract := deployConfiguredMCMS(t, client, auth)
			targetAddress, targetContract := mcmAddress, mcmContract
			if scenario == "other MCMS target" {
				targetAddress, targetContract = deployConfiguredMCMS(t, client, auth)
			}
			// Production ownership can be a contract, not the funded test key.
			require.NoError(t, ChangeAddressSlotContext(t.Context(), logger.Nop(), MCMSLayout, rpcURL, "_owner", mcmAddress.Hex(), mcmAddress.Hex()))
			require.NoError(t, ChangeAddressSlotContext(t.Context(), logger.Nop(), MCMSLayout, rpcURL, "_owner", targetAddress.Hex(), mcmAddress.Hex()))
			original, err := mcmContract.GetConfig(&bind.CallOpts{Context: t.Context()})
			require.NoError(t, err)
			before := configDocument(t, targetAddress, targetContract)

			restore, err := SetTemporaryMCMSigner(t.Context(), logger.Nop(), MCMSLayout,
				blockchain.DefaultAnvilPrivateKey, blockchain.DefaultAnvilPublicKey, blockchain.DefaultAnvilPublicKey,
				rpcURL, "1337", mcmAddress.Hex())
			require.NoError(t, err)
			installed, err := mcmContract.GetConfig(&bind.CallOpts{Context: t.Context()})
			require.NoError(t, err)
			require.Equal(t, testSignerConfig(blockchain.DefaultAnvilPublicKey), installed)
			owner, err := mcmContract.Owner(&bind.CallOpts{Context: t.Context()})
			require.NoError(t, err)
			require.Equal(t, mcmAddress, owner)

			// Benign non-empty calldata for the no-change scenario: a plain
			// transfer to an EOA ignores call data, so proposal validation
			// passes without authoring any state change.
			operation := types.Transaction{To: auth.From.Hex(), Data: []byte{0, 0, 0, 0}, AdditionalFields: json.RawMessage(`{"value":0}`)}
			if scenario != "no authored config change" {
				contractABI, abiErr := gethwrappers.ManyChainMultiSigMetaData.GetAbi()
				require.NoError(t, abiErr)
				operation.To = targetAddress.Hex()
				operation.Data, err = contractABI.Pack("setConfig", []common.Address{auth.From}, []uint8{0}, [32]uint8{1}, [32]uint8{}, false)
				require.NoError(t, err)
			}
			selector := types.ChainSelector(chainsel.GETH_TESTNET.Selector)
			proposal := &mcms.Proposal{
				BaseProposal: mcms.BaseProposal{
					Version: "v1", Kind: types.KindProposal, ValidUntil: 2082758399,
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{selector: {MCMAddress: mcmAddress.Hex()}},
				},
				Operations: []types.Operation{{ChainSelector: selector, Transaction: operation}},
			}
			inspector := evm.NewInspector(client)
			signable, err := mcms.NewSignable(proposal, map[types.ChainSelector]sdk.Inspector{selector: inspector})
			require.NoError(t, err)
			_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(privateKey))
			require.NoError(t, err)
			executable, err := mcms.NewExecutable(proposal, map[types.ChainSelector]sdk.Executor{
				selector: evm.NewExecutor(evm.NewEncoder(selector, 1, false, false), client, auth),
			})
			require.NoError(t, err)
			rootResult, err := executable.SetRoot(t.Context(), selector)
			require.NoError(t, err)
			confirmTx(t, client, rootResult.RawData.(*gethtypes.Transaction))
			rootBefore, err := mcmContract.GetRoot(&bind.CallOpts{Context: t.Context()})
			require.NoError(t, err)

			require.NoError(t, restore(t.Context()))
			restored, err := mcmContract.GetConfig(&bind.CallOpts{Context: t.Context()})
			require.NoError(t, err)
			require.Equal(t, original, restored)
			owner, err = mcmContract.Owner(&bind.CallOpts{Context: t.Context()})
			require.NoError(t, err)
			require.Equal(t, mcmAddress, owner)
			rootAfter, err := mcmContract.GetRoot(&bind.CallOpts{Context: t.Context()})
			require.NoError(t, err)
			require.Equal(t, rootBefore, rootAfter)
			// The temporary signing key is no longer a configured signer, but
			// the already-authenticated operation must still execute.
			result, err := executable.Execute(t.Context(), 0)
			require.NoError(t, err)
			confirmTx(t, client, result.RawData.(*gethtypes.Transaction))
			changes, unreadable, err := statediff.DiffJSON(before, configDocument(t, targetAddress, targetContract))
			require.NoError(t, err)
			require.Empty(t, unreadable)
			if scenario == "no authored config change" {
				require.Empty(t, changes)
			} else {
				require.NotEmpty(t, changes)
				for _, change := range changes {
					require.Equal(t, targetAddress.Hex(), common.HexToAddress(change.Address).Hex())
				}
				finalConfig, configErr := targetContract.GetConfig(&bind.CallOpts{Context: t.Context()})
				require.NoError(t, configErr)
				require.Equal(t, testSignerConfig(blockchain.DefaultAnvilPublicKey), finalConfig)
			}
		})
	}
}

func startAnvil(t *testing.T) (string, *ethclient.Client) {
	t.Helper()
	path, err := exec.LookPath("anvil")
	if err != nil {
		t.Skip("local Anvil binary is required for the MCMS bytecode integration test")
	}
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	require.NoError(t, listener.Close())
	command := exec.CommandContext(t.Context(), path, "--host", "127.0.0.1", "--port", strconv.Itoa(port), "--chain-id", "1337", "--silent")
	require.NoError(t, command.Start())
	t.Cleanup(func() {
		_ = command.Process.Kill()
		_ = command.Wait()
	})
	rpcURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	client, err := ethclient.DialContext(t.Context(), rpcURL)
	require.NoError(t, err)
	t.Cleanup(client.Close)
	require.Eventually(t, func() bool {
		_, probeErr := client.BlockNumber(t.Context())
		return probeErr == nil
	}, 5*time.Second, 20*time.Millisecond)

	return rpcURL, client
}

func deployConfiguredMCMS(t *testing.T, client *ethclient.Client, auth *bind.TransactOpts) (common.Address, *gethwrappers.ManyChainMultiSig) {
	t.Helper()
	address, tx, contract, err := gethwrappers.DeployManyChainMultiSig(auth, client)
	require.NoError(t, err)
	confirmTx(t, client, tx)
	signers := make([]common.Address, 2)
	for i := range signers {
		privateKey, keyErr := crypto.GenerateKey()
		require.NoError(t, keyErr)
		signers[i] = crypto.PubkeyToAddress(privateKey.PublicKey)
	}
	slices.SortFunc(signers, func(a, b common.Address) int { return a.Cmp(b) })
	tx, err = contract.SetConfig(auth, signers, []uint8{1, 1}, [32]uint8{1, 2}, [32]uint8{}, false)
	require.NoError(t, err)
	confirmTx(t, client, tx)

	return address, contract
}

func confirmTx(t *testing.T, client *ethclient.Client, tx *gethtypes.Transaction) {
	t.Helper()
	receipt, err := bind.WaitMined(t.Context(), client, tx)
	require.NoError(t, err)
	require.Equal(t, gethtypes.ReceiptStatusSuccessful, receipt.Status)
}

func configDocument(t *testing.T, address common.Address, contract *gethwrappers.ManyChainMultiSig) []byte {
	t.Helper()
	config, err := contract.GetConfig(&bind.CallOpts{Context: t.Context()})
	require.NoError(t, err)
	raw, err := json.Marshal(map[string]any{"chains": map[string]any{strconv.FormatUint(chainsel.GETH_TESTNET.Selector, 10): []any{
		map[string]any{"address": address.Hex(), "_type": "ManyChainMultiSig", "_requestedVersion": "1.0.0", "config": map[string]any{
			"signers": config.Signers, "groupQuorums": config.GroupQuorums, "groupParents": config.GroupParents,
		}},
	}}})
	require.NoError(t, err)

	return raw
}
