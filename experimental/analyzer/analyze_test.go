package analyzer

import (
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/proposalutils"

	cldfds "github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

func TestNewDefaultProposalContext(t *testing.T) {
	t.Parallel()

	// --- arrange ---

	datastore := cldfds.NewMemoryDataStore()
	err := datastore.Addresses().Add(cldfds.AddressRef{
		ChainSelector: 9012,
		Address:       "0xcallproxyaddress",
		Type:          "CallProxy",
		Version:       semver.MustParse("9.0.1"),
		Qualifier:     "",
		Labels:        cldfds.NewLabelSet("call-proxy-label"),
	})
	require.NoError(t, err)

	env := cldf.Environment{
		ExistingAddresses: cldf.NewMemoryAddressBook(),
		DataStore:         datastore.Seal(),
	}

	// --- act ---
	got, err := NewDefaultProposalContext(env)

	// --- assert ---
	require.NoError(t, err)
	require.IsType(t, &DefaultProposalContext{}, got)
	require.Equal(t, got.(*DefaultProposalContext).AddressesByChain, cldf.AddressesByChain{ //nolint:testifylint
		9012: {
			"0xcallproxyaddress": cldf.TypeAndVersion{
				Type:    cldf.ContractType("CallProxy"),
				Version: *semver.MustParse("9.0.1"),
				Labels:  cldf.NewLabelSet("call-proxy-label"),
			},
		},
	})
}

func Test_DefaultProposalContext_ArgumentContext(t *testing.T) {
	t.Parallel()

	addresses := cldf.AddressesByChain{
		1234: {
			"0xrbacTimelockAddress": cldf.MustTypeAndVersionFromString("RBACTimelock 1.0.0"),
		},
		5678: {
			"0xmanyChainMultisigAddress": cldf.MustTypeAndVersionFromString("ManyChainMultisig 1.0.0"),
		},
	}
	proposalContext := &DefaultProposalContext{AddressesByChain: addresses}

	got := proposalContext.ArgumentContext(5678)

	require.Equal(t, got, &proposalutils.ArgumentContext{ //nolint:testifylint
		Ctx: map[string]any{
			"AddressesByChain": cldf.AddressesByChain{
				5678: {
					"0xmanyChainMultisigAddress": cldf.MustTypeAndVersionFromString("ManyChainMultisig 1.0.0"),
				},
			},
		},
	})
}
