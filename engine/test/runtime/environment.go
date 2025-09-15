package runtime

import (
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	foperations "github.com/smartcontractkit/chainlink-deployments-framework/operations"
)

// newEnvFromState creates a fresh environment with updated state while preserving base
// configuration.
func newEnvFromState(fromEnv fdeployment.Environment, state *State) fdeployment.Environment {
	return fdeployment.Environment{
		// These fields never change from the initial environment
		Name:        fromEnv.Name,
		Logger:      fromEnv.Logger,
		GetContext:  fromEnv.GetContext,
		OCRSecrets:  fromEnv.OCRSecrets,
		Offchain:    fromEnv.Offchain,
		BlockChains: fromEnv.BlockChains,
		NodeIDs:     fromEnv.NodeIDs,
		Catalog:     fromEnv.Catalog,

		// These fields are updated by changesets and are pulled from state
		ExistingAddresses: state.AddressBook,
		DataStore:         state.DataStore,

		// These fields require new instances to avoid retaining state from previous changesets
		OperationsBundle: foperations.NewBundle(
			fromEnv.GetContext, fromEnv.Logger, foperations.NewMemoryReporter(),
		),
	}
}
