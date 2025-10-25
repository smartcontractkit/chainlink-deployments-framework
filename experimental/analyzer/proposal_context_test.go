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
