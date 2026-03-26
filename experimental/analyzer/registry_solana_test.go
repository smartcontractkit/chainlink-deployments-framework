package analyzer

import (
	"testing"

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
					DataStore:         ds.Seal()},
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
