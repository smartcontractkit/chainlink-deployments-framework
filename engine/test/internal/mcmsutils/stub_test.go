package mcmsutils

import (
	sollib "github.com/gagliardetto/solana-go"

	fchainaptos "github.com/smartcontractkit/chainlink-deployments-framework/chain/aptos"
	fchainevm "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	fchainsolana "github.com/smartcontractkit/chainlink-deployments-framework/chain/solana"
)

// stubAptosChain creates a stubbed Aptos chain
func stubAptosChain() fchainaptos.Chain {
	return fchainaptos.Chain{
		Selector: 10,
	}
}

// stubEVMChain creates a stubbed EVM chain
func stubEVMChain() fchainevm.Chain {
	return fchainevm.Chain{
		Selector: 20,
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
		Selector:    30,
		DeployerKey: &dummyKey,
	}
}
