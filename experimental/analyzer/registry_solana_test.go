package analyzer

import (
	"maps"
	"slices"
	"testing"

	"github.com/samber/lo"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

func Test_EnvironmentSolanaRegistry_GetSolanaInstructionDecoderByAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		registry      map[string]DecodeInstructionFn
		addresses     deployment.AddressesByChain
		chainSelector uint64
		address       string
		wantNil       bool
		wantErr       string
	}{
		{
			name: "success - decoder found",
			registry: map[string]DecodeInstructionFn{
				deployment.MustTypeAndVersionFromString("Timelock 1.0.0").String(): nil, // store nil decoder; type matches
			},
			addresses: deployment.AddressesByChain{
				1234: {
					"So1anaTimelock11111111111111111111111111111111": deployment.MustTypeAndVersionFromString("Timelock 1.0.0"),
				},
			},
			chainSelector: 1234,
			address:       "So1anaTimelock11111111111111111111111111111111",
			wantNil:       true, // returned decoder should be the stored nil
		},
		{
			name: "failure - unknown chain",
			registry: map[string]DecodeInstructionFn{
				deployment.MustTypeAndVersionFromString("Timelock 1.0.0").String(): nil,
			},
			addresses: deployment.AddressesByChain{
				1234: {
					"So1anaTimelock11111111111111111111111111111111": deployment.MustTypeAndVersionFromString("Timelock 1.0.0"),
				},
			},
			chainSelector: 5678,
			address:       "So1anaTimelock11111111111111111111111111111111",
			wantErr:       "no addresses found for chain selector 5678",
		},
		{
			name: "failure - unknown address",
			registry: map[string]DecodeInstructionFn{
				deployment.MustTypeAndVersionFromString("Timelock 1.0.0").String(): nil,
			},
			addresses: deployment.AddressesByChain{
				1234: {
					"So1anaTimelock11111111111111111111111111111111": deployment.MustTypeAndVersionFromString("Timelock 1.0.0"),
				},
			},
			chainSelector: 1234,
			address:       "Unknown11111111111111111111111111111111111111",
			wantErr:       "address Unknown11111111111111111111111111111111111111 not found for chain selector 1234",
		},
		{
			name:     "failure - type and version not in decoder registry",
			registry: map[string]DecodeInstructionFn{
				// intentionally empty; address maps to unknown type/version
			},
			addresses: deployment.AddressesByChain{
				1234: {
					"So1anaOther111111111111111111111111111111111": deployment.MustTypeAndVersionFromString("UnknownProgram 1.0.0"),
				},
			},
			chainSelector: 1234,
			address:       "So1anaOther111111111111111111111111111111111",
			// note: error text in implementation says "ABI not found ..." (copy/paste from EVM). Match exactly.
			wantErr: "ABI not found for type and version UnknownProgram 1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ds := datastore.NewMemoryDataStore()
			for chainSelector, addrMap := range tt.addresses {
				for addr, typeAndVer := range addrMap {
					err := ds.Addresses().Add(datastore.AddressRef{
						ChainSelector: chainSelector,
						Address:       addr,
						Type:          datastore.ContractType(typeAndVer.Type),
						Version:       &typeAndVer.Version,
					})
					require.NoError(t, err)
				}
			}
			reg, err := NewEnvironmentSolanaRegistry(
				deployment.Environment{
					ExistingAddresses: deployment.NewMemoryAddressBook(),
					DataStore:         ds.Seal(),
				},
				tt.registry,
			)
			require.NoError(t, err)

			decoder, err := reg.GetSolanaInstructionDecoderByAddress(tt.chainSelector, tt.address)

			if tt.wantErr == "" {
				require.NoError(t, err)
				if tt.wantNil {
					require.Nil(t, decoder)
				} else {
					require.NotNil(t, decoder)
				}
			} else {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func Test_EnvironmentSolanaRegistry_GetSolanaInstructionDecoderByType_And_Add(t *testing.T) {
	t.Parallel()

	reg, err := NewEnvironmentSolanaRegistry(deployment.Environment{
		ExistingAddresses: deployment.NewMemoryAddressBook(),
		DataStore:         datastore.NewMemoryDataStore().Seal(),
	}, map[string]DecodeInstructionFn{})
	require.NoError(t, err)

	// Use nil as a stand-in decoder; verifies registry wiring without needing the signature of DecodeInstructionFn.
	tv := deployment.MustTypeAndVersionFromString("Bypasser 2.0.0")

	// Not present initially
	_, err = reg.GetSolanaInstructionDecoderByType(tv)
	require.Error(t, err)
	require.ErrorContains(t, err, "ABI not found for type and version Bypasser 2.0.0")

	// Add then retrieve
	reg.AddSolanaInstructionDecoder(tv, nil)

	dec, err := reg.GetSolanaInstructionDecoderByType(tv)
	require.NoError(t, err)
	require.Nil(t, dec)
}

func Test_EnvironmentSolanaRegistry_Decoders_ReturnsDefensiveCopy(t *testing.T) {
	t.Parallel()

	tv := deployment.MustTypeAndVersionFromString("Bypasser 2.0.0")
	reg, err := NewEnvironmentSolanaRegistry(deployment.Environment{
		ExistingAddresses: deployment.NewMemoryAddressBook(),
		DataStore:         datastore.NewMemoryDataStore().Seal(),
	}, map[string]DecodeInstructionFn{tv.String(): nil})
	require.NoError(t, err)

	decoders := reg.Decoders()
	delete(decoders, tv.String())

	decoder, getErr := reg.GetSolanaInstructionDecoderByType(tv)
	require.NoError(t, getErr)
	require.Nil(t, decoder)
}

func Test_EnvironmentSolanaRegistry_NilDecoderMappings(t *testing.T) {
	t.Parallel()

	reg, err := NewEnvironmentSolanaRegistry(deployment.Environment{
		ExistingAddresses: deployment.NewMemoryAddressBookFromMap(map[uint64]map[string]deployment.TypeAndVersion{
			chainsel.SOLANA_DEVNET.Selector: {
				"TestContract1111111111111111111111111111111": deployment.MustTypeAndVersionFromString("TestContract 1.0.0"),
			},
		}),
		DataStore: datastore.NewMemoryDataStore().Seal(),
	}, nil)
	require.NoError(t, err)
	require.NotNil(t, reg)

	require.Equal(t, reg.addressesByChain, deployment.AddressesByChain{ //nolint:testifylint
		chainsel.SOLANA_DEVNET.Selector: {
			"TestContract1111111111111111111111111111111":  deployment.MustTypeAndVersionFromString("TestContract 1.0.0"),
			"BPFLoaderUpgradeab1e11111111111111111111111":  deployment.MustTypeAndVersionFromString("BPFLoaderUpgradeable 1.0.0"),
			"11111111111111111111111111111111":             deployment.MustTypeAndVersionFromString("System 1.0.0"),
			"ComputeBudget111111111111111111111111111111":  deployment.MustTypeAndVersionFromString("ComputeBudget 1.0.0"),
			"MemoSq4gqABAXKb96qnH8TysNcWxMyWCqXgDLGmfcHr":  deployment.MustTypeAndVersionFromString("Memo 1.0.0"),
			"Stake11111111111111111111111111111111111111":  deployment.MustTypeAndVersionFromString("Stake 1.0.0"),
			"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA":  deployment.MustTypeAndVersionFromString("Token 1.0.0"),
			"Vote111111111111111111111111111111111111111":  deployment.MustTypeAndVersionFromString("Vote 1.0.0"),
			"verifycLy8mB96wd9wqq3WDXQwM4oU6r42Th37Db9fC":  deployment.MustTypeAndVersionFromString("OtterVerify 1.0.0"),
			"CmPVzy88JSB4S223yCvFmBxTLobLya27KgEDeNPnqEub": deployment.MustTypeAndVersionFromString("TokenRegistry 1.0.0"),
		},
	})
	require.ElementsMatch(t, slices.Collect(maps.Keys(reg.registry)), []string{
		"Memo 1.0.0", "OtterVerify 1.0.0", "BPFLoaderUpgradeable 1.0.0", "ComputeBudget 1.0.0",
		"TokenRegistry 1.0.0", "Vote 1.0.0", "Stake 1.0.0", "System 1.0.0", "Token 1.0.0",
	})
	decodeFns := lo.FilterValues(reg.registry, func(_ string, fn DecodeInstructionFn) bool { return fn != nil })
	require.Len(t, decodeFns, len(nativePrograms))
}
