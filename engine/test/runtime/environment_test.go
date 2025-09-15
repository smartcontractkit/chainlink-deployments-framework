package runtime

import (
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fdatastore "github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	foperations "github.com/smartcontractkit/chainlink-deployments-framework/operations"
)

func TestNewEnvFromState(t *testing.T) {
	t.Parallel()

	// Setup test data
	lggr := logger.Test(t)
	getCtx := t.Context

	// Create initial environment with sample data
	initialAddrBook := fdeployment.NewMemoryAddressBook()
	initialDatastore := fdatastore.NewMemoryDataStore().Seal()

	fromEnv := fdeployment.Environment{
		Name:              "test-env",
		Logger:            lggr,
		GetContext:        getCtx,
		ExistingAddresses: initialAddrBook,
		DataStore:         initialDatastore,
		NodeIDs:           []string{"node1", "node2"},
		OperationsBundle:  foperations.NewBundle(getCtx, lggr, foperations.NewMemoryReporter()),
	}

	// Create state with updated data
	addrBookState := fdeployment.NewMemoryAddressBookFromMap(map[uint64]map[string]fdeployment.TypeAndVersion{
		1: {
			"addr1": fdeployment.NewTypeAndVersion("type1", *semver.MustParse("1.0.0")),
		},
	})

	datastoreState := fdatastore.NewMemoryDataStore()
	err := datastoreState.AddressRefStore.Add(fdatastore.AddressRef{
		ChainSelector: 1,
		Address:       "addr1",
		Type:          fdatastore.ContractType("type1"),
		Version:       semver.MustParse("1.0.0"),
		Qualifier:     "qual1",
	})
	require.NoError(t, err)

	state := &State{
		AddressBook: addrBookState,
		DataStore:   datastoreState.Seal(),
	}

	got := newEnvFromState(fromEnv, state)

	// Verify preserved fields from original environment
	assert.Equal(t, fromEnv.Name, got.Name)
	assert.Equal(t, fromEnv.Logger, got.Logger)
	assert.Equal(t, fromEnv.GetContext(), got.GetContext())
	assert.Equal(t, fromEnv.NodeIDs, got.NodeIDs)
	assert.Equal(t, fromEnv.OCRSecrets, got.OCRSecrets)
	assert.Equal(t, fromEnv.BlockChains, got.BlockChains)
	assert.Equal(t, fromEnv.Offchain, got.Offchain)
	assert.Equal(t, fromEnv.Catalog, got.Catalog)

	// Verify updated fields from state
	assert.Equal(t, state.AddressBook, got.ExistingAddresses) //nolint:staticcheck // SA1019 (Deprecated): We still need to support AddressBook for now
	assert.Equal(t, state.DataStore, got.DataStore)

	// Verify new instances are created (should be different objects)
	assert.NotNil(t, got.OperationsBundle)
	assert.NotSame(t, &fromEnv.OperationsBundle, &got.OperationsBundle)
}
