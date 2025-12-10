package mcmsutils

import (
	"encoding/json"
	"time"

	"github.com/Masterminds/semver/v3"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	sollib "github.com/gagliardetto/solana-go"

	chainselectors "github.com/smartcontractkit/chain-selectors"
	mcmslib "github.com/smartcontractkit/mcms"
	mcmstypes "github.com/smartcontractkit/mcms/types"

	fchain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
	fchainaptos "github.com/smartcontractkit/chainlink-deployments-framework/chain/aptos"
	fchainevm "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	fchainsolana "github.com/smartcontractkit/chainlink-deployments-framework/chain/solana"
	fdatastore "github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

// stubAptosChain creates a stubbed Aptos chain
func stubAptosChain() fchainaptos.Chain {
	return fchainaptos.Chain{
		Selector: chainselectors.APTOS_LOCALNET.Selector,
	}
}

// stubEVMChain creates a stubbed EVM chain
func stubEVMChain() fchainevm.Chain {
	return fchainevm.Chain{
		Selector: chainselectors.ETHEREUM_TESTNET_SEPOLIA.Selector,
		Confirm: func(tx *gethtypes.Transaction) (uint64, error) { // This is a stubbed implementation of the Confirm function which always returns success
			return 0, nil
		},
	}
}

// stubSolanaChain creates a stubbed Solana chain
func stubSolanaChain() fchainsolana.Chain {
	// Create a dummy private key for testing (32 bytes repeated to make 64 bytes)
	privateKeyBytes := make([]byte, 64)
	for i := range 64 {
		privateKeyBytes[i] = byte(i%32 + 1)
	}
	dummyKey := sollib.PrivateKey(privateKeyBytes)

	return fchainsolana.Chain{
		Selector:    chainselectors.TEST_22222222222222222222222222222222222222222222.Selector,
		DeployerKey: &dummyKey,
	}
}

// stubEnvironment creates a stubbed environment with a single EVM chain
func stubEnvironment() fdeployment.Environment {
	chain := stubEVMChain()

	ds := fdatastore.NewMemoryDataStore()
	ds.Addresses().Add(fdatastore.AddressRef{ //nolint:errcheck // This will not fail in the test
		ChainSelector: chain.Selector,
		Address:       "0x1234567890123456789012345678901234567890",
		Type:          "CallProxy",
		Version:       semver.MustParse("1.0.0"),
	})

	return fdeployment.Environment{
		DataStore: ds.Seal(),
		BlockChains: fchain.NewBlockChainsFromSlice(
			[]fchain.BlockChain{chain},
		),
	}
}

// stubMCMSProposal stubs a minimal MCMS proposal for testing
func stubMCMSProposal() *mcmslib.Proposal {
	return &mcmslib.Proposal{
		BaseProposal: mcmslib.BaseProposal{
			Version:     "v1",
			Kind:        mcmstypes.KindProposal,
			Description: "Test MCMS Proposal",
			ValidUntil:  uint32(time.Now().Add(1 * time.Hour).Unix()), //nolint:gosec // This is for testing purposes only
			ChainMetadata: map[mcmstypes.ChainSelector]mcmstypes.ChainMetadata{
				mcmstypes.ChainSelector(stubEVMChain().Selector): {
					StartingOpCount: 0,
					MCMAddress:      "0x0000000000000000000000000000000000000000",
				},
			},
		},
		Operations: []mcmstypes.Operation{
			{
				ChainSelector: mcmstypes.ChainSelector(stubEVMChain().Selector),
				Transaction: mcmstypes.Transaction{
					To:               "0x123",
					AdditionalFields: json.RawMessage(`{"value": 0}`),
					Data:             []byte{1, 2, 3},
					OperationMetadata: mcmstypes.OperationMetadata{
						ContractType: "test",
						Tags:         []string{"test"},
					},
				},
			},
		},
	}
}

// stubTimelockProposal creates a minimal timelock proposal for testing supporting a single
// EVM chain
func stubTimelockProposal(
	action mcmstypes.TimelockAction,
) *mcmslib.TimelockProposal {
	selector := mcmstypes.ChainSelector(stubEVMChain().Selector)

	return &mcmslib.TimelockProposal{
		BaseProposal: mcmslib.BaseProposal{
			Version:    "v1",
			Kind:       mcmstypes.KindTimelockProposal,
			ValidUntil: uint32(time.Now().Add(1 * time.Hour).Unix()), //nolint:gosec // This is for testing purposes only
			ChainMetadata: map[mcmstypes.ChainSelector]mcmstypes.ChainMetadata{
				selector: {
					StartingOpCount: 0,
					MCMAddress:      "0x0000000000000000000000000000000000000000",
				},
			},
		},
		Action: action,
		TimelockAddresses: map[mcmstypes.ChainSelector]string{
			selector: "0x0000000000000000000000000000000000000000",
		},
		Operations: []mcmstypes.BatchOperation{
			{
				ChainSelector: selector,
				Transactions: []mcmstypes.Transaction{
					{
						To:               "0x123",
						AdditionalFields: json.RawMessage(`{"value": 0}`),
						Data:             []byte{1, 2, 3},
						OperationMetadata: mcmstypes.OperationMetadata{
							ContractType: "test",
							Tags:         []string{"test"},
						},
					},
				},
			},
		},
	}
}
