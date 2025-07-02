package provider

import (
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chain_selectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/testutils"
)

func Test_ZkSyncCTFChainProviderConfig_validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  ZkSyncCTFChainProviderConfig
		wantErr string
	}{
		{
			name: "valid config",
			config: ZkSyncCTFChainProviderConfig{
				Once: testutils.DefaultNetworkOnce,
			},
			wantErr: "",
		},
		{
			name: "missing sync.Once instance",
			config: ZkSyncCTFChainProviderConfig{
				Once: nil,
			},
			wantErr: "sync.Once instance is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.validate()
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_CTFChainProvider_Initialize(t *testing.T) {
	t.Skip("Skipping test for CTF chain provider initialization, too flaky when starting container," +
		" enable once the issue is resolved")
	t.Parallel()

	var chainSelector = chain_selectors.TEST_1000.Selector

	tests := []struct {
		name         string
		giveSelector uint64
		giveConfig   ZkSyncCTFChainProviderConfig
		wantErr      string
	}{
		{
			name:         "valid initialization",
			giveSelector: chainSelector,
			giveConfig: ZkSyncCTFChainProviderConfig{
				Once: testutils.DefaultNetworkOnce,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := NewZkSyncCTFChainProvider(t, tt.giveSelector, tt.giveConfig)

			got, err := p.Initialize(t.Context())
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, p.chain)

				// Check that the chain is of type evm.Chain and has the expected fields
				gotChain, ok := got.(evm.Chain)
				require.True(t, ok, "expected got to be of type evm.Chain")
				assert.Equal(t, tt.giveSelector, gotChain.Selector)
				assert.NotEmpty(t, gotChain.Client)
				assert.NotEmpty(t, gotChain.DeployerKey)
				assert.NotEmpty(t, gotChain.Users)
				assert.NotNil(t, gotChain.Confirm)
				assert.True(t, gotChain.IsZkSyncVM)
				assert.NotNil(t, gotChain.ClientZkSyncVM)
				assert.NotNil(t, gotChain.DeployerKeyZkSyncVM)
				assert.NotNil(t, gotChain.SignHash)
			}
		})
	}
}

func Test_ZkSyncCTFChainProvider_Name(t *testing.T) {
	t.Parallel()

	p := &ZkSyncCTFChainProvider{}
	assert.Equal(t, "ZkSync EVM CTF Chain Provider", p.Name())
}

func Test_ZkSyncCTFChainProvider_ChainSelector(t *testing.T) {
	t.Parallel()

	p := &ZkSyncCTFChainProvider{selector: chain_selectors.TEST_1000.Selector}
	assert.Equal(t, chain_selectors.TEST_1000.Selector, p.ChainSelector())
}

func Test_ZkSyncCTFChainProvider_BlockChain(t *testing.T) {
	t.Parallel()

	chain := &evm.Chain{}

	p := &ZkSyncCTFChainProvider{
		chain: chain,
	}

	assert.Equal(t, *chain, p.BlockChain())
}

func Test_ZkSyncCTFChainProvider_SignHash(t *testing.T) {
	t.Skip("Skipping test, too flaky when starting container," +
		" enable once the issue is resolved")
	t.Parallel()

	var chainSelector = chain_selectors.TEST_1000.Selector

	p := NewZkSyncCTFChainProvider(t, chainSelector, ZkSyncCTFChainProviderConfig{
		Once: testutils.DefaultNetworkOnce,
	})

	chain, err := p.Initialize(t.Context())
	require.NoError(t, err)

	evmChain, ok := chain.(evm.Chain)
	require.True(t, ok, "expected chain to be of type evm.Chain")
	require.NotNil(t, evmChain.SignHash, "SignHash function should not be nil")

	// Test signing a hash
	testMessage := []byte("test message for signing")
	testHash := crypto.Keccak256(testMessage)

	signature, err := evmChain.SignHash(testHash)
	require.NoError(t, err, "SignHash should not return an error")
	require.NotEmpty(t, signature, "signature should not be empty")
	require.Len(t, signature, 65, "Ethereum signature should be 65 bytes")

	// Test that SignHash is deterministic for the same input
	signature2, err := evmChain.SignHash(testHash)
	require.NoError(t, err)
	require.Equal(t, signature, signature2, "SignHash should be deterministic for the same input")

	// Test with different hash produces different signature
	differentMessage := []byte("different test message")
	differentHash := crypto.Keccak256(differentMessage)

	differentSignature, err := evmChain.SignHash(differentHash)
	require.NoError(t, err)
	require.NotEqual(t, signature, differentSignature, "different hashes should produce different signatures")

	// Test with empty hash (but still 32 bytes as required)
	emptyHash := make([]byte, 32) // Create a 32-byte zero hash
	emptySignature, err := evmChain.SignHash(emptyHash)
	require.NoError(t, err)
	require.NotEmpty(t, emptySignature)
	require.Len(t, emptySignature, 65)

	// Verify that the signature can be used to recover the correct address
	// (This tests that the signature is properly formatted)
	recoveredPubKey, err := crypto.SigToPub(testHash, signature)
	require.NoError(t, err, "should be able to recover public key from signature")

	recoveredAddr := crypto.PubkeyToAddress(*recoveredPubKey)
	require.Equal(t, evmChain.DeployerKey.From, recoveredAddr, "recovered address should match the deployer address")
}
