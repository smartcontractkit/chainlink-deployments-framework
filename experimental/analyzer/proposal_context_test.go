package analyzer

import (
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

func TestWithRenderer(t *testing.T) {
	t.Parallel()

	// Create a proper environment
	ds := datastore.NewMemoryDataStore()
	env := deployment.Environment{
		ExistingAddresses: deployment.NewMemoryAddressBook(),
		DataStore:         ds.Seal(),
	}

	customRenderer := NewMarkdownRenderer()

	ctx, err := NewDefaultProposalContext(env, WithRenderer(customRenderer))
	require.NoError(t, err)
	require.NotNil(t, ctx)

	// Verify that the custom renderer is set
	retrievedRenderer := ctx.GetRenderer()
	require.Equal(t, customRenderer, retrievedRenderer)
}

func TestNewDefaultProposalContext(t *testing.T) {
	t.Parallel()

	ds := datastore.NewMemoryDataStore()
	err := ds.Addresses().Add(datastore.AddressRef{
		ChainSelector: 9012,
		Address:       "0xcallproxyaddress",
		Type:          "CallProxy",
		Version:       semver.MustParse("9.0.1"),
		Qualifier:     "",
		Labels:        datastore.NewLabelSet("call-proxy-label"),
	})
	require.NoError(t, err)

	env := deployment.Environment{
		ExistingAddresses: deployment.NewMemoryAddressBook(),
		DataStore:         ds.Seal(),
	}

	got, err := NewDefaultProposalContext(env)

	require.NoError(t, err)
	require.IsType(t, &DefaultProposalContext{}, got)
	require.Equal(t, got.(*DefaultProposalContext).AddressesByChain, deployment.AddressesByChain{ //nolint:testifylint
		9012: {
			"0xcallproxyaddress": deployment.TypeAndVersion{
				Type:    deployment.ContractType("CallProxy"),
				Version: *semver.MustParse("9.0.1"),
				Labels:  deployment.NewLabelSet("call-proxy-label"),
			},
		},
	})
}

// TestNewDefaultProposalContext_UpgradedInPlace checks that a contract with refs of several versions at one address
// resolves to the highest version whatever order the refs come in, so the analyzer picks the decoder for the code
// that is deployed. Refs of another type at the same address still override it.
func TestNewDefaultProposalContext_UpgradedInPlace(t *testing.T) {
	t.Parallel()

	const (
		chainSelector = 124615329519749607
		router        = "Ccip842gzYHhvdDkSyi2YVCoAWPbYJoApMFzSxQroE9C"
	)
	ref := func(version string) datastore.AddressRef {
		return datastore.AddressRef{
			ChainSelector: chainSelector,
			Address:       router,
			Type:          "Router",
			Version:       semver.MustParse(version),
		}
	}

	tests := []struct {
		name string
		refs []datastore.AddressRef
		want deployment.TypeAndVersion
	}{
		{
			name: "older ref last",
			refs: []datastore.AddressRef{ref("2.1.0"), ref("1.6.0")},
			want: deployment.MustTypeAndVersionFromString("Router 2.1.0"),
		},
		{
			name: "newer ref last",
			refs: []datastore.AddressRef{ref("1.6.0"), ref("2.1.0")},
			want: deployment.MustTypeAndVersionFromString("Router 2.1.0"),
		},
		{
			name: "other type at the same address",
			refs: []datastore.AddressRef{ref("2.1.0"), {
				ChainSelector: chainSelector,
				Address:       router,
				Type:          "TestRouter",
				Version:       semver.MustParse("1.0.0"),
			}},
			want: deployment.MustTypeAndVersionFromString("TestRouter 1.0.0"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ds := datastore.NewMemoryDataStore()
			for _, r := range tt.refs {
				require.NoError(t, ds.Addresses().Add(r))
			}
			env := deployment.Environment{
				ExistingAddresses: deployment.NewMemoryAddressBook(),
				DataStore:         ds.Seal(),
			}

			got, err := NewDefaultProposalContext(env)
			require.NoError(t, err)
			require.Equal(t, tt.want, got.(*DefaultProposalContext).AddressesByChain[chainSelector][router]) //nolint:testifylint

			solReg, err := NewEnvironmentSolanaRegistry(env, nil)
			require.NoError(t, err)
			require.Equal(t, tt.want, solReg.addressesByChain[chainSelector][router])
		})
	}
}

func Test_DefaultProposalContext_FieldContext(t *testing.T) {
	t.Parallel()

	addresses := deployment.AddressesByChain{
		1234: {
			"0xrbacTimelockAddress": deployment.MustTypeAndVersionFromString("RBACTimelock 1.0.0"),
		},
		5678: {
			"0xmanyChainMultisigAddress": deployment.MustTypeAndVersionFromString("ManyChainMultisig 1.0.0"),
		},
	}
	proposalContext := &DefaultProposalContext{
		AddressesByChain: addresses,
		renderer:         NewMarkdownRenderer(),
	}

	got := proposalContext.FieldsContext(5678)

	require.Equal(t, got, &FieldContext{ //nolint:testifylint
		Ctx: map[string]any{
			"AddressesByChain": deployment.AddressesByChain{
				5678: {
					"0xmanyChainMultisigAddress": deployment.MustTypeAndVersionFromString("ManyChainMultisig 1.0.0"),
				},
			},
		},
	})
}
