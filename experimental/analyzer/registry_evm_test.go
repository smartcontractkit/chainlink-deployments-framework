package analyzer

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

func Test_EnvironmentEVMRegistry_GetABIByAddress(t *testing.T) {
	t.Parallel()

	// Minimal valid ABIs for parsing + comparison
	rbacTimelockABI := `[{"type":"function","name":"foo","stateMutability":"view","inputs":[],"outputs":[{"type":"uint256","name":""}]}]`

	tests := []struct {
		name          string
		abiRegistry   map[string]string
		addresses     deployment.AddressesByChain
		chainSelector uint64
		address       string
		wantABI       string
		wantErr       string
	}{
		{
			name: "success - RBACTimelock",
			abiRegistry: map[string]string{
				deployment.MustTypeAndVersionFromString("RBACTimelock 1.0.0").String(): rbacTimelockABI,
			},
			addresses: deployment.AddressesByChain{
				1234: {
					"0xrbacTimelockAddress": deployment.MustTypeAndVersionFromString("RBACTimelock 1.0.0"),
				},
			},
			chainSelector: 1234,
			address:       "0xrbacTimelockAddress", // mixed case is ok
			wantABI:       rbacTimelockABI,
		},
		{
			name: "failure - unknown chain",
			abiRegistry: map[string]string{
				deployment.MustTypeAndVersionFromString("RBACTimelock 1.0.0").String(): rbacTimelockABI,
			},
			addresses: deployment.AddressesByChain{
				1234: {
					"0xrbacTimelockAddress": deployment.MustTypeAndVersionFromString("RBACTimelock 1.0.0"),
				},
			},
			chainSelector: 5678,
			address:       "0xrbacTimelockAddress",
			wantErr:       "no addresses found for chain selector 5678",
		},
		{
			name: "failure - unknown address",
			abiRegistry: map[string]string{
				deployment.MustTypeAndVersionFromString("RBACTimelock 1.0.0").String(): rbacTimelockABI,
			},
			addresses: deployment.AddressesByChain{
				1234: {
					"0xrbacTimelockAddress": deployment.MustTypeAndVersionFromString("RBACTimelock 1.0.0"),
				},
			},
			chainSelector: 1234,
			address:       "0xunknownAddress",
			wantErr:       "address 0xunknownAddress not found for chain selector 1234",
		},
		{
			name:        "failure - type and version not in abi registry",
			abiRegistry: map[string]string{
				// intentionally empty for UnknownContractType
			},
			addresses: deployment.AddressesByChain{
				1234: {
					"0xunknownAddress": deployment.MustTypeAndVersionFromString("UnknownContractType 1.0.0"),
				},
			},
			chainSelector: 1234,
			address:       "0xunknownAddress",
			wantErr:       "ABI not found for type and version UnknownContractType 1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ds := datastore.NewMemoryDataStore()
			// populate the in-memory datastore with the test addresses
			for chainSel, addrMap := range tt.addresses {
				for addr, tv := range addrMap {
					ref := datastore.AddressRef{
						ChainSelector: chainSel,
						Address:       addr, // already lowercased in tt.addresses where needed
						Type:          datastore.ContractType(tv.Type),
						Version:       &tv.Version,
					}
					if !tv.Labels.IsEmpty() {
						ref.Labels = datastore.NewLabelSet(tv.Labels.List()...)
					}
					require.NoError(t, ds.Addresses().Add(ref))
				}
			}

			// now seal and build the env/registry
			env := deployment.Environment{DataStore: ds.Seal(), ExistingAddresses: deployment.NewMemoryAddressBook()}
			reg, err := NewEnvironmentEVMRegistry(env, tt.abiRegistry)
			require.NoError(t, err)

			_, abiStr, err := reg.GetABIByAddress(tt.chainSelector, tt.address)

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Equal(t, tt.wantABI, abiStr)
			} else {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}
