package rpcclient

import (
	"fmt"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/avast/retry-go/v4"
	sollib "github.com/gagliardetto/solana-go"
	solsystem "github.com/gagliardetto/solana-go/programs/system"
	solrpc "github.com/gagliardetto/solana-go/rpc"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-testing-framework/framework"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/smartcontractkit/freeport"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/testutils"
)

func Test_Client_SendAndConfirmTx(t *testing.T) {
	t.Parallel()

	var (
		chainID = chainsel.TEST_22222222222222222222222222222222222222222222.ChainID
	)
	// The admin key is a private key assigned as the token mint authority
	minterKey, err := sollib.NewRandomPrivateKey()
	require.NoError(t, err)

	httpURL := startSolanaContainer(t, chainID, minterKey.PublicKey())

	// Create the Solana client wrapper for the minter account
	client := New(solrpc.New(httpURL), minterKey)

	// Create the receiver account
	receiverKey, err := sollib.NewRandomPrivateKey()
	require.NoError(t, err)

	// Test sending a transaction with a retry mechanism
	_, err = client.SendAndConfirmTx(
		t.Context(),
		[]sollib.Instruction{
			solsystem.NewTransferInstruction(
				sollib.LAMPORTS_PER_SOL*2,
				minterKey.PublicKey(),
				receiverKey.PublicKey(),
			).Build(),
		},
		WithRetry(3, 1*time.Second),
	)
	require.NoError(t, err)

	// Check the balance of the receiver account
	balanceRes, err := client.GetBalance(t.Context(), receiverKey.PublicKey(), solrpc.CommitmentConfirmed)
	require.NoError(t, err)

	require.Equal(t, 2*sollib.LAMPORTS_PER_SOL, balanceRes.Value)
}

// startSolanaContainer starts a Solana container using the Chainlink Testing Framework,
// initializing the Docker network, reserving ports, and setting up the Solana node.
//
// The container is automatically cleaned up and ports are released after the calling test
// completes.
//
// Returns the external HTTP URL of the Solana node.
func startSolanaContainer(
	t *testing.T, chainID string, publicKey sollib.PublicKey,
) string {
	t.Helper()

	// initialize the docker network used by CTF
	err := framework.DefaultNetwork(testutils.DefaultNetworkOnce)
	require.NoError(t, err)

	// solana requires 2 ports, one for http and one for ws, but only allows one to be specified
	// the other is +1 of the first one
	// must reserve 2 to avoid port conflicts in the freeport library with other tests
	// https://github.com/smartcontractkit/chainlink-testing-framework/blob/e109695d311e6ed42ca3194907571ce6454fae8d/framework/components/blockchain/blockchain.go#L39
	ports := freeport.GetN(t, 2)

	image := ""
	if runtime.GOOS == "linux" {
		image = "solanalabs/solana:v1.18.26" // TODO: workaround on linux
	}

	bcInput := &blockchain.Input{
		Image:     image,
		Type:      blockchain.TypeSolana,
		ChainID:   chainID,
		PublicKey: publicKey.String(),
		Port:      strconv.Itoa(ports[0]),
	}

	output, err := blockchain.NewBlockchainNetwork(bcInput)
	require.NoError(t, err)

	// Close the container after the test is done
	testcontainers.CleanupContainer(t, output.Container)

	// Wait for the Solana node to be healthy
	checkSolanaNodeHealth(t, output.Nodes[0].ExternalHTTPUrl)

	return output.Nodes[0].ExternalHTTPUrl
}

// checkSolanaNodeHealth checks the health of the Solana node by querying its health endpoint.
// We expect that node will be available within 30 seconds, with a 1 second delay between attempts,
// however this is an assumption.
func checkSolanaNodeHealth(t *testing.T, httpURL string) {
	t.Helper()

	solclient := solrpc.New(httpURL)
	err := retry.Do(func() error {
		out, rerr := solclient.GetHealth(t.Context())
		if rerr != nil {
			return rerr
		}
		if out != solrpc.HealthOk {
			return fmt.Errorf("API server not healthy yet: %s", out)
		}

		return nil
	},
		retry.Context(t.Context()),
		retry.Attempts(30),
		retry.Delay(1*time.Second),
		retry.DelayType(retry.FixedDelay),
	)
	require.NoError(t, err, "API server is not healthy")
}
