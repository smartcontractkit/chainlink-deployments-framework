package provider

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"

	aptoslib "github.com/aptos-labs/aptos-go-sdk"
	"github.com/aptos-labs/aptos-go-sdk/crypto"
	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-testing-framework/framework"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/smartcontractkit/freeport"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/aptos"
)

var _ chain.Provider = (*CTFChainProvider)(nil)

type CTFChainProvider struct {
	t        *testing.T
	selector uint64

	// Required
	adminAccount *aptoslib.Account

	chain *aptos.Chain
}

func NewCTFChainProvider(t *testing.T, selector uint64) (*CTFChainProvider, error) {
	t.Helper()

	p := &CTFChainProvider{
		t:        t,
		selector: selector,
	}

	return p, nil
}

func (p *CTFChainProvider) WithDefaultAdminAccount() *CTFChainProvider {
	addressStr := blockchain.DefaultAptosAccount
	var defaultAddress aptoslib.AccountAddress
	err := defaultAddress.ParseStringRelaxed(addressStr)
	require.NoError(p.t, err)

	privateKeyStr := blockchain.DefaultAptosPrivateKey
	privateKeyBytes, err := hex.DecodeString(strings.TrimPrefix(privateKeyStr, "0x"))
	require.NoError(p.t, err)
	privateKey := ed25519.NewKeyFromSeed(privateKeyBytes)

	p.t.Logf("Using default Aptos account: %s", addressStr)

	account, err := aptoslib.NewAccountFromSigner(&crypto.Ed25519PrivateKey{Inner: privateKey}, defaultAddress)
	require.NoError(p.t, err)

	p.adminAccount = account

	return p
}

func (p *CTFChainProvider) WithNewAdminAccount(account *aptoslib.Account) *CTFChainProvider {
	account, err := aptoslib.NewEd25519SingleSenderAccount()
	require.NoError(p.t, err)

	p.adminAccount = account

	return account
}

func (p *CTFChainProvider) Initialize() error {

	// initialize the docker network used by CTF
	// err := framework.DefaultNetwork(once)
	// require.NoError(t, err)

	// Get the Aptos Chain ID
	chainID, err := chain_selectors.GetChainIDFromSelector(p.selector)
	if err != nil {
		return fmt.Errorf("failed to get chain ID from selector %d: %w", p.selector, err)
	}

	// Start the CTF Container
	url, client, err := p.startContainer(chainID, p.adminAccount.Address().String())
	if err != nil {
		return fmt.Errorf("failed to start CTF container for chain %d: %w", chainID, err)
	}

	// Validate the chain

	// construct the chain
	p.chain = &aptos.Chain{
		Selector:       p.selector,
		Client:         client,
		DeployerSigner: p.adminAccount,
		URL:            url,
		Confirm: func(txHash string, opts ...any) error {
			userTx, err := client.WaitForTransaction(txHash, opts...)
			if err != nil {
				return err
			}
			if !userTx.Success {
				return fmt.Errorf("transaction failed: %s", userTx.VmStatus)
			}
			return nil
		},
	}

	return nil
}

func (*CTFChainProvider) Name() string {
	return "Aptos CTF Chain Provider"
}

func (p *CTFChainProvider) ChainSelector() uint64 {
	return p.selector
}

func (p *CTFChainProvider) BlockChain() chain.BlockChain {
	return p.chain
}

func (p *CTFChainProvider) Chain() *aptos.Chain {
	return p.chain
}

func (p *CTFChainProvider) startContainer(chainID string, adminAddr string) (string, *aptoslib.Client, error) {
	var (
		maxRetries    = 10
		url           string
		containerName string
	)

	for i := 0; i < maxRetries; i++ {
		// reserve all the ports we need explicitly to avoid port conflicts in other tests
		ports := freeport.GetN(p.t, 2)

		input := &blockchain.Input{
			Image:     "", // filled out by defaultAptos function
			Type:      "aptos",
			ChainID:   chainID,
			PublicKey: adminAddr,
			CustomPorts: []string{
				fmt.Sprintf("%d:8080", ports[0]),
				fmt.Sprintf("%d:8081", ports[1]),
			},
		}

		output, err := blockchain.NewBlockchainNetwork(input)
		if err != nil {
			p.t.Logf("Error creating Aptos network: %v", err)
			freeport.Return(ports)
			time.Sleep(time.Second)
			maxRetries -= 1

			continue
		}
		require.NoError(p.t, err)

		containerName = output.ContainerName
		testcontainers.CleanupContainer(p.t, output.Container)
		url = output.Nodes[0].ExternalHTTPUrl + "/v1"
		break
	}

	client, err := aptoslib.NewNodeClient(url, 0)
	require.NoError(p.t, err)

	var ready bool
	for i := 0; i < 30; i++ {
		time.Sleep(time.Second)
		_, err := client.GetChainId()
		if err != nil {
			t.Logf("API server not ready yet (attempt %d): %+v\n", i+1, err)
			continue
		}
		ready = true
		break
	}
	require.True(t, ready, "Aptos network not ready")
	time.Sleep(15 * time.Second) // we have slot errors that force retries if the chain is not given enough time to boot

	dc, err := framework.NewDockerClient()
	require.NoError(p.t, err)

	// incase we didn't use the default account above
	_, err = dc.ExecContainer(containerName, []string{
		"aptos", "account", "fund-with-faucet",
		"--account", p.adminAccount.Account.String(),
		"--amount", "100000000000",
	})
	require.NoError(p.t, err)

	return url, client, nil
}
